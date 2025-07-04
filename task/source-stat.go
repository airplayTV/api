package task

import (
	"errors"
	"fmt"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/tidwall/gjson"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"log"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
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
			for _, source := range appSourceList {
				log.Println(fmt.Sprintf("[执行任务] chunk %d %s", chunkIdx, source.Handler.Name()))
				var tmpR = x.parseVideoResolution(source)
				tmpList = append(tmpList, tmpR)
				tmpIdx++
			}
			ch <- tmpList
		}(chunkIdx, chunk)
	}

	var resolutionList = make([]model.VideoResolution, 0)
	for range chunks {
		resolutionList = append(resolutionList, <-ch...)
	}

	slices.SortFunc(resolutionList, func(a, b model.VideoResolution) int {
		// 格式化数据，防止出现1926(1920)这种傻逼数据影响排序
		a.Width = a.Width / 10 * 10
		b.Width = b.Width / 10 * 10
		if b.Width != a.Width {
			return b.Width - a.Width // 降序
		}
		if b.Latency != a.Latency {
			return int(a.Latency - b.Latency) // 升序
		}
		return strings.Compare(a.Time, b.Time)
	})

	var p = filepath.Join(util.AppPath(), fmt.Sprintf("cache/stat/source-stat-%s.json", time.Now().Format("2006010215")))
	if err := util.WriteFile(p, util.ToBytes(resolutionList)); err != nil {
		log.Println("[SourceStat写文件失败]", err.Error())
	}

	log.Println(fmt.Sprintf("[完成任务] ok %s", p))
}

func (x SourceStat) parseVideoResolution(h model.SourceHandler) (tmpR model.VideoResolution) {
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
	probe, err := ffmpeg.Probe(tmpUrl, ffmpeg.KwArgs{
		"allowed_extensions": "ALL",
		"extension_picky":    0,
		"timeout":            1000000 * 15, // 15秒
		"show_entries":       "stream=width,height",
	})
	if err != nil {
		return
	}

	var result = gjson.Parse(probe)
	if !result.Get("programs").IsArray() || len(result.Get("programs").Array()) == 0 {
		log.Println("[probe]", probe)
		return width, height, errors.New("programs没有数据")
	}
	var resolution = result.Get("programs").Array()[0].Get("streams").Array()[0]
	width = int(resolution.Get("width").Int())
	height = int(resolution.Get("height").Int())

	return
}
