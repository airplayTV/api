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
	"slices"
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
			log.Println("[taskHandler.recover]", err)
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
			log.Println("[taskHandler.recover]", err)
		}
	}()

	var resolutionList = make([]model.VideoResolution, 0)

	var appSourceMap = model.AppSourceMap()

	var idx = 0
	for _, source := range appSourceMap {
		log.Println("[resolveSource]", idx, source.Handler.Name())
		var tmpR = x.parseVideoResolution(source)
		resolutionList = append(resolutionList, tmpR)
		idx++
	}
	slices.SortFunc(resolutionList, func(a, b model.VideoResolution) int {
		return b.Width - a.Width
	})

	var p = filepath.Join(util.AppPath(), fmt.Sprintf("cache/stat/source-stat-%s.json", time.Now().Format("2006010215")))
	if err := util.WriteFile(p, util.ToBytes(resolutionList)); err != nil {
		log.Println("[SourceStat写文件失败]", err.Error())
	}

	log.Println(fmt.Sprintf("[resolveSource] ok %s", p))
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
