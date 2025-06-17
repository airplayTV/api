package controller

import (
	"errors"
	"fmt"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/lixiang4u/goWebsocket"
	"github.com/tidwall/gjson"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
	"log"
	"path/filepath"
	"slices"
	"time"
)

type VideoResolution struct {
	Source string // 视频源
	Name   string // 视频名
	Url    string // 视频地址
	TsUrl  string // 视频播放的TS地址
	Width  int    //视频 resolution 宽度
	Height int    //视频 resolution 高度
	Err    string // 错误原因
}

func RunParseResolution() {
	var resolutionList = make([]VideoResolution, 0)

	for _, source := range sourceMap {
		var tmpR = parseVideoResolution(source)
		resolutionList = append(resolutionList, tmpR)

		log.Println("[resolutionList]", goWebsocket.ToJson(resolutionList))
	}

	slices.SortFunc(resolutionList, func(a, b VideoResolution) int {
		return a.Width - b.Width
	})

	var p = filepath.Join(util.AppPath(), fmt.Sprintf("cache/stat/source-stat-%s.json", time.Now().Format("20060102")))
	_ = util.WriteFile(p, util.ToBytes(resolutionList))
}

func parseVideoResolution(h SourceHandler) (tmpR VideoResolution) {
	tmpR.Source = h.Handler.Name()

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

	var err error
	tmpR.Width, tmpR.Height, err = getMpegResolution(tmpSource.Url)
	if err != nil {
		tmpR.Err = err.Error()
		return
	}

	return tmpR
}

func getMpegResolution(tmpUrl string) (width, height int, err error) {
	probe, err := ffmpeg_go.Probe(tmpUrl, ffmpeg_go.KwArgs{"show_entries": "stream=width,height"})
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

func getMpegTSResolution(tmpUrl string) (width, height int, err error) {
	// 根据给定ts文件，下载并解析resolution
	var httpClient = util.HttpClient{}
	_, buff, err := httpClient.GetResponse(tmpUrl, 1024*1024)
	if err != nil {
		return
	}
	var saveFile = filepath.Join(util.AppPath(), fmt.Sprintf("cache/tmp/%s", util.StringMd5(tmpUrl)))
	if err = util.WriteFile(saveFile, buff); err != nil {
		return
	}

	probe, err := ffmpeg_go.Probe(saveFile, ffmpeg_go.KwArgs{"show_entries": "stream=width,height"})
	if err != nil {
		return
	}

	var result = gjson.Parse(probe)
	if !result.Get("programs").IsArray() || len(result.Get("programs").Array()) == 0 {
		log.Println("[probe]", probe)
		return width, height, errors.New("programs空")
	}
	var resolution = result.Get("programs").Array()[0].Get("streams").Array()[0]
	width = int(resolution.Get("width").Int())
	height = int(resolution.Get("height").Int())

	return
}
