package task

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/lixiang4u/goWebsocket"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"io"
	"log"
	"path/filepath"
	"runtime/debug"
	"time"
)

type SourceStat struct {
}

func NewSourceStat() *SourceStat {
	return &SourceStat{}
}

func (x SourceStat) Run() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("[taskHandler.recover0]", err)
			log.Println("[taskHandler.recover0]", string(debug.Stack()))
		}
	}()

	var ticker = time.NewTicker(time.Hour * 1 / 2)
	x.taskHandler()
	for {
		select {
		case <-ticker.C:
			log.Println("[SourceStat.Run]")
			x.taskHandler()
		}
	}

}

func (x SourceStat) taskHandler() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("[taskHandler.recover1]", err)
			log.Println("[taskHandler.recover1]", string(debug.Stack()))
		}
	}()

	log.Println("[开始执行任务]")

	var chCount = 5 // 分组
	var chunks = make([][]model.SourceHandler, chCount)
	var idx = 0
	for _, tmpHandler := range model.AppSourceMap() {
		if idx >= chCount {
			idx = 0
		}
		if chunks[idx] == nil {
			chunks[idx] = make([]model.SourceHandler, 0)
		}
		chunks[idx] = append(chunks[idx], tmpHandler)
		idx = idx + 1
	}

	var ch = make(chan []model.VideoResolution, chCount)
	for chunkIdx, chunk := range chunks {
		go func(chunkIdx int, appSourceList []model.SourceHandler) {
			var tmpIdx = 0
			var tmpList = make([]model.VideoResolution, 0)

			defer func() {
				if err := recover(); err != nil {
					log.Println("[taskHandler.recover4]", err)
					log.Println("[taskHandler.recover4]", string(debug.Stack()))
				}
				ch <- tmpList
			}()

			for _, source := range appSourceList {
				var tmpR = x.parseVideoResolution(source)
				tmpList = append(tmpList, tmpR)
				tmpIdx++
			}
		}(chunkIdx, chunk)
	}

	var resolutionList = make([]model.VideoResolution, 0)
	for range chunks {
		resolutionList = append(resolutionList, <-ch...)
	}

	var date = cast.ToInt64(time.Now().Format("20060102150405"))
	for i, item := range resolutionList {
		item.Date = date
		resolutionList[i] = item
	}
	_ = model.VideoResolution{}.SaveAll(resolutionList)

	log.Println(fmt.Sprintf("[完成任务] ok %d", date))
}

func (x SourceStat) parseVideoResolution(h model.SourceHandler) (tmpR model.VideoResolution) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("[taskHandler.recover3]", err)
			log.Println("[taskHandler.recover3]", string(debug.Stack()))
		}
	}()

	tmpR.Source = h.Handler.Name()
	tmpR.Time = time.Now().Format(time.DateTime)

	var resp = h.Handler.VideoList("", "1")

	var videoList []model.Video
	switch resp.(type) {
	case model.Success:
		videoList = resp.(model.Success).Data.(model.Pager).List
	case model.Error:
		tmpR.Err = resp.(model.Error).Msg
		return
	}
	if len(videoList) <= 0 {
		tmpR.Err = "没有可用视频列表"
		return
	}

	var tmpVideo model.Video
	resp = h.Handler.Detail(videoList[0].Id)
	switch resp.(type) {
	case model.Success:
		tmpVideo = resp.(model.Success).Data.(model.Video)
	case model.Error:
		tmpR.Err = resp.(model.Error).Msg
		return
	}
	if len(tmpVideo.Links) <= 0 {
		tmpR.Err = "没有可用播放列表"
		return
	}

	tmpR.Name = videoList[0].Name
	tmpR.Vid = videoList[0].Id

	var tmpSource model.Source
	resp = h.Handler.Source(tmpVideo.Links[0].Id, videoList[0].Id)
	switch resp.(type) {
	case model.Success:
		tmpSource = resp.(model.Success).Data.(model.Source)
	case model.Error:
		tmpR.Err = resp.(model.Error).Msg
		return
	}
	if len(tmpSource.Url) <= 0 {
		tmpR.Err = "没有可用播放地址"
		return
	}

	tmpR.Url = tmpSource.Url
	tmpR.Pid = tmpSource.Id

	var err error
	var ts1 = time.Now().UnixMilli() // 毫秒
	tmpR.Width, tmpR.Height, err = x.getMpegResolution(tmpSource.Url)
	tmpR.Latency = time.Now().UnixMilli() - ts1
	if err != nil {
		tmpR.Err = err.Error()
		return
	}

	// 格式化数据，防止出现1926(1920)这种傻逼数据影响排序
	tmpR.Width = tmpR.Width / 10 * 10

	return tmpR
}

func (x SourceStat) downloadMp4(source, tmpUrl string) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("[downloadMp4.recover]", err)
			log.Println("[downloadMp4.recover]", string(debug.Stack()))
		}
	}()

	var outputFile = filepath.Join(util.AppPath(), fmt.Sprintf("d-%s-%s.mp4", source, time.Now().Format("20060102150405")))
	var inputKwArgs = ffmpeg.KwArgs{"allowed_extensions": "ALL", "extension_picky": 0}
	var outputKwArgs = ffmpeg.KwArgs{"c": "copy"}
	if err := ffmpeg.Input(tmpUrl, inputKwArgs).Output(outputFile, outputKwArgs).OverWriteOutput().ErrorToStdOut().Run(); err != nil {
		log.Println("[下载失败]", err.Error(), outputFile)
		return err
	} else {
		log.Println("[下载成功]", outputFile)
		return nil
	}
}

func (x SourceStat) getMpegResolution(tmpUrl string) (width, height int, err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("[taskHandler.recover4]", err)
			log.Println("[taskHandler.recover4]", string(debug.Stack()))
		}
	}()

	var kwArgs = ffmpeg.KwArgs{
		"allowed_extensions": "ALL",
		"extension_picky":    0,
		"timeout":            1000000 * 15, // 15秒
		"show_entries":       "stream=width,height",
	}

	probe, err := ffmpeg.ProbeWithTimeout(tmpUrl, time.Second*30, kwArgs)
	width, height, err = x.parseProbe(probe)
	if err != nil {
		log.Println("[ProbeReader.tmpUrl]", tmpUrl)
		log.Println("[ProbeReader.Error]", err.Error())
		return
	}
	if width*height > 100 {
		return
	}
	buff, err := x.parsePngMaskTsSegment(tmpUrl)
	if err != nil {
		log.Println("[ProbeReader.tmpUrl2]", tmpUrl)
		log.Println("[ProbeReader.Error2]", err.Error())
		return
	}
	probe, err = ffmpeg.ProbeReaderWithTimeout(buff, time.Second*30, kwArgs)
	width, height, err = x.parseProbe(probe)
	if err != nil {
		log.Println("[ProbeReader.buff]", buff.String())
		log.Println("[ProbeReader.Error]", err.Error())
		return
	}

	return
}

func (x SourceStat) parseProbe(probe string) (width, height int, err error) {
	var result = gjson.Parse(probe)
	if !result.Get("programs").IsArray() || len(result.Get("programs").Array()) == 0 {
		err = errors.New(fmt.Sprintf("probe解析异常：%s", probe))
		return
	}
	var resolution = result.Get("programs").Array()[0].Get("streams").Array()[0]
	width = int(resolution.Get("width").Int())
	height = int(resolution.Get("height").Int())

	return
}

func (x SourceStat) parsePngMaskTsSegment(tmpUrl string) (buffer *bytes.Buffer, err error) {
	tmpList, err := util.ParsePlayUrlList(tmpUrl)
	if err != nil {
		return
	}
	log.Println("[tmpList]", goWebsocket.ToJson(tmpList))
	var httpClient = util.HttpClient{}
	buff, err := httpClient.Get(tmpList[0])
	if err != nil {
		return
	}
	var newBuffer = &bytes.Buffer{}

	buffer = bytes.NewBuffer(buff)
	_, err = io.CopyN(io.Discard, buffer, 8)
	if err != nil {
		return
	}
	_, err = io.Copy(newBuffer, buffer)
	if err != nil {
		return
	}

	return newBuffer, err
}
