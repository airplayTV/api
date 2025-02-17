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

type Huawei8ApiHandler struct {
	Handler
}

func (x Huawei8ApiHandler) Init() IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.UserAgent, useragent)
	x.httpClient.AddHeader(headers.Origin, huawei8Host)
	x.httpClient.AddHeader(headers.Referer, huawei8Host)
	return x
}

func (x Huawei8ApiHandler) Name() string {
	return "华为吧api"
}

func (x Huawei8ApiHandler) TagList() interface{} {
	// tag列表：https://bfzyapi.com/api.php/provide/vod/?ac=list&pg=1&t=1
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "20"})
	tags = append(tags, gin.H{"name": "电视剧", "value": "60"})
	tags = append(tags, gin.H{"name": "综艺", "value": "82"})
	tags = append(tags, gin.H{"name": "动漫", "value": "80"})
	tags = append(tags, gin.H{"name": "纪录片", "value": "86"})
	tags = append(tags, gin.H{"name": "短剧", "value": "120"})
	tags = append(tags, gin.H{"name": "其它", "value": "58"})
	return tags
}

func (x Huawei8ApiHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x Huawei8ApiHandler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x Huawei8ApiHandler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x Huawei8ApiHandler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x Huawei8ApiHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x Huawei8ApiHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(huawei8ApiTagUrl, x.parsePageNumber(page), tagName))
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

	pager.List = x.handleVideoListThumb(huawei8ApiDetailUrl, pager.List)

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x Huawei8ApiHandler) _search(keyword, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(huawei8ApiSearchUrl, x.parsePageNumber(page), keyword))
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

func (x Huawei8ApiHandler) parseVidTypeId(str string) (vid, tid string, err error) {
	var tmpList = strings.Split(str, "-")
	if len(tmpList) != 2 {
		return "", "", errors.New("请求参数错误")
	}
	vid = tmpList[0]
	tid = tmpList[1]
	return vid, tid, nil
}

func (x Huawei8ApiHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(huawei8ApiDetailUrl, id))
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
						Group: "华为吧",
					})
				}
			}
			return true
		})
	}

	return model.NewSuccess(video)
}

func (x Huawei8ApiHandler) _source(pid, vid string) interface{} {
	tmpVid, tmpTid, err := x.parseVidTypeId(pid)
	if err != nil {
		return model.NewError(err.Error())
	}
	buff, err := x.httpClient.Get(fmt.Sprintf(huawei8ApiDetailUrl, tmpVid))
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

func (x Huawei8ApiHandler) UpdateHeader(header map[string]string) error {
	return nil
}

func (x Huawei8ApiHandler) HoldCookie() error {
	return nil
}
