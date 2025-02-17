package handler

import (
	"errors"
	"fmt"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"log"
	"strings"
)

type KczyHandler struct {
	Handler
}

func (x KczyHandler) Init() IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.UserAgent, useragent)
	x.httpClient.AddHeader(headers.Origin, kczyHost)
	x.httpClient.AddHeader(headers.Referer, kczyHost)
	return x
}

func (x KczyHandler) Name() string {
	return "快车资源"
}

func (x KczyHandler) TagList() interface{} {
	// tag列表：https://caiji.kczyapi.com/api.php/provide/vod/?ac=list
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "1"})
	tags = append(tags, gin.H{"name": "连续剧", "value": "2"})
	tags = append(tags, gin.H{"name": "动漫", "value": "4"})
	tags = append(tags, gin.H{"name": "综艺", "value": "3"})
	tags = append(tags, gin.H{"name": "邵氏电影", "value": "70"})
	tags = append(tags, gin.H{"name": "其它", "value": "55"})
	return tags
}

func (x KczyHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x KczyHandler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x KczyHandler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x KczyHandler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x KczyHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x KczyHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(kczyTagUrl, x.parsePageNumber(page), tagName))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var pager = model.Pager{Limit: 20, Page: x.parsePageNumber(page), List: make([]model.Video, 0)}

	var result = gjson.ParseBytes(buff)

	pager.Total = int(result.Get("total").Int())
	pager.Pages = int(result.Get("pagecount").Int())
	pager.Page = int(result.Get("page").Int())
	pager.Limit = int(result.Get("limit").Int())
	result.Get("list").ForEach(func(key, value gjson.Result) bool {
		pager.List = append(pager.List, model.Video{
			Id:         value.Get("vod_id").String(),
			Name:       value.Get("vod_name").String(),
			Thumb:      value.Get("vod_pic").String(),
			Intro:      value.Get("vod_blurb").String(),
			Resolution: value.Get("vod_remarks").String(),
		})
		return true
	})

	pager.List = x.handleVideoListThumb(bfzyDetailUrl, pager.List)

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x KczyHandler) _search(keyword, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(kczySearchUrl, x.parsePageNumber(page), keyword))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var pager = model.Pager{Limit: 20, Page: x.parsePageNumber(page)}

	var result = gjson.ParseBytes(buff)

	pager.Total = int(result.Get("total").Int())
	pager.Pages = int(result.Get("pagecount").Int())
	pager.Page = int(result.Get("page").Int())
	pager.Limit = int(result.Get("limit").Int())
	result.Get("list").ForEach(func(key, value gjson.Result) bool {
		pager.List = append(pager.List, model.Video{
			Id:         value.Get("vod_id").String(),
			Name:       value.Get("vod_name").String(),
			Thumb:      value.Get("vod_pic").String(),
			Intro:      value.Get("vod_blurb").String(),
			Resolution: value.Get("vod_remarks").String(),
		})
		return true
	})

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x KczyHandler) parseVidTypeId(str string) (vid, tid string, err error) {
	var tmpList = strings.Split(str, "-")
	if len(tmpList) != 2 {
		return "", "", errors.New("请求参数错误")
	}
	vid = tmpList[0]
	tid = tmpList[1]
	return vid, tid, nil
}

func (x KczyHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(kczyDetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var video = model.Video{Id: id, Links: make([]model.Link, 0)}
	var result = gjson.ParseBytes(buff)
	if result.Get("total").Int() > 0 {
		result.Get("list").ForEach(func(key, value gjson.Result) bool {
			video.Name = value.Get("vod_name").String()
			video.Thumb = value.Get("vod_pic").String()
			video.Intro = value.Get("vod_content").String()
			video.Actors = value.Get("vod_actor").String()
			for idx, tmpUrl := range strings.Split(value.Get("vod_play_url").String(), "#") {
				var tmpList = strings.Split(tmpUrl, "$")
				if len(tmpList) == 2 {
					video.Links = append(video.Links, model.Link{
						Id:    fmt.Sprintf("%s-%d", video.Id, idx),
						Name:  tmpList[0],
						Url:   tmpList[1],
						Group: "bfzym3u8",
					})
				}
			}
			return true
		})
	}

	return model.NewSuccess(video)
}

func (x KczyHandler) _source(pid, vid string) interface{} {
	tmpVid, tmpTid, err := x.parseVidTypeId(pid)
	if err != nil {
		return model.NewError(err.Error())
	}
	buff, err := x.httpClient.Get(fmt.Sprintf(kczyDetailUrl, tmpVid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var source = model.Source{Id: pid, Vid: vid}
	var result = gjson.ParseBytes(buff)

	log.Println("[NNN]", result.Get("total").Int())

	if result.Get("total").Int() > 0 {
		var tmpNid = util.ParseNumber(tmpTid)
		result.Get("list").ForEach(func(key, value gjson.Result) bool {
			source.Name = value.Get("vod_name").String()
			source.Thumb = value.Get("vod_pic").String()
			for idx, tmpUrl := range strings.Split(value.Get("vod_play_url").String(), "#") {
				if idx == tmpNid {
					var tmpList = strings.Split(tmpUrl, "$")
					if len(tmpList) == 2 {
						source.Url = tmpList[1]
						source.Source = tmpList[1]
					}
				}
			}
			return true
		})
	}
	source.Type = x.parseVideoType(source.Source)
	if len(source.Url) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(source)
}

func (x KczyHandler) UpdateHeader(header map[string]string) error {
	return nil
}

func (x KczyHandler) HoldCookie() error {
	return nil
}
