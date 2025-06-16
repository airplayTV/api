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
)

type VideoResolution struct {
	Source string
	Url    string
	TsUrl  string
	Width  int
	Height int
	Err    string
}

func RunParseResolution() {
	var resolutionList = make([]VideoResolution, 0)

	for _, source := range sourceMap {
		var tmpR = parseVideoResolution(source)
		resolutionList = append(resolutionList, tmpR)

		log.Println("[resolutionList]", goWebsocket.ToJson(resolutionList))
	}

	log.Println("[end]")
}

func parseVideoResolution(h SourceHandler) (tmpR VideoResolution) {
	tmpR.Source = h.Handler.Name()

	var resp = h.Handler.VideoList("", "1")

	var videoList []model.Video
	switch resp.(type) {
	case model.Success:
		videoList = resp.(model.Success).Data.(model.Pager).List
	case model.Error:
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
	}
	if len(tmpVideo.Links) <= 0 {
		tmpR.Err = "没有可用播放列表"
		return
	}

	var tmpSource model.Source
	resp = h.Handler.Source(tmpVideo.Links[0].Id, videoList[0].Id)
	switch resp.(type) {
	case model.Success:
		tmpSource = resp.(model.Success).Data.(model.Source)
	case model.Error:
	}
	if len(tmpSource.Url) <= 0 {
		tmpR.Err = "没有可用播放地址"
		return
	}

	tmpR.Url = tmpSource.Url

	playUrlList, err := util.ParsePlayUrlList(tmpSource.Url)
	if err != nil || len(playUrlList) == 0 {
		tmpR.Err = "m3u8文件解析失败"
		return
	}

	log.Println("[playUrlList]", goWebsocket.ToJson(playUrlList))

	for idx, tmpUrl := range playUrlList {
		if idx > 3 {
			break
		}
		tmpWidth, tmpHeight, err := getMpegTSResolution(tmpUrl)
		if err == nil && tmpWidth > 0 {
			tmpR.Width = tmpWidth
			tmpR.Height = tmpHeight
			tmpR.TsUrl = tmpUrl
			break
		}
		log.Println("[即将尝试解析]", idx, tmpUrl)
	}
	if tmpR.Width <= 0 {
		tmpR.Err = "m3u8文件解析失败"
		return
	}

	return tmpR
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
