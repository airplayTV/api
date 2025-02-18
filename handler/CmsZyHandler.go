package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"log"
	"net/url"
	"strings"
)

// 采集源
// https://bgm.tv/m/topic/group/406236
// https://woodchen.ink/archives/1207
// https://www.x-lsp.com/category/资源采集

type CmsZyHandler struct {
	Handler
}

func (x CmsZyHandler) Init(options interface{}) IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.UserAgent, useragent)

	var o = options.(model.CmsZyOption)
	x.option = model.CmsZyOption{Name: o.Name, Api: o.Api, Id: o.Id}

	return x
}

func (x CmsZyHandler) Name() string {
	return x.option.GetName()
}

func (x CmsZyHandler) getApiUrl() string {
	tmpUrl, err := url.Parse(x.option.GetApi())
	if err != nil {
		return ""
	}
	if len(tmpUrl.Scheme) > 0 && len(tmpUrl.Host) > 0 {
		return fmt.Sprintf("%s://%s/%s", tmpUrl.Scheme, tmpUrl.Host, strings.Trim(tmpUrl.Path, "/"))
	}
	return ""
}

func (x CmsZyHandler) TagList() interface{} {
	var key = fmt.Sprintf("tag-list-%s", x.option.GetName())

	data, err := handlerCache.Get(context.Background(), key)
	if err == nil {
		return x.formatTags(data.(map[string]string))
	}
	var tmpTags map[string]string
	log.Println("[req]", x.option.GetName(), x.getApiUrl())
	buff, err := x.httpClient.Get(x.getApiUrl())
	if err != nil {
		tmpTags = map[string]string{}
		tmpTags["全部"] = ""

		log.Println("[ResolveTagError]", err.Error())
		return x.formatTags(tmpTags)
	}
	tmpTags = make(map[string]string)
	tmpTags["全部"] = ""
	var result = gjson.ParseBytes(buff)
	result.Get("class").ForEach(func(key, value gjson.Result) bool {
		tmpTags[value.Get("type_name").String()] = value.Get("type_id").String()
		return true
	})

	if err = handlerCache.Set(context.Background(), key, tmpTags); err != nil {
		log.Println("[CacheSetError]", err.Error())
	}

	return x.formatTags(tmpTags)
}

func (x CmsZyHandler) formatTags(tags map[string]string) []model.KV1 {
	var result = make([]model.KV1, 0)
	for k, v := range tags {
		result = append(result, model.KV1{
			Name:  k,
			Value: v,
		})
	}
	return result
}

func (x CmsZyHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x CmsZyHandler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x CmsZyHandler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x CmsZyHandler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x CmsZyHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x CmsZyHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf("%s/?ac=list&pg=%d&t=%s", x.getApiUrl(), x.parsePageNumber(page), tagName))
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

	pager.List = x.handleVideoListThumb(fmt.Sprintf("%s/?ac=detail&ids=%%s", x.getApiUrl()), pager.List)

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x CmsZyHandler) _search(keyword, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf("%s/?ac=list&pg=%d&t=&wd=%s", x.getApiUrl(), x.parsePageNumber(page), keyword))
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

	pager.List = x.handleVideoListThumb(fmt.Sprintf("%s/?ac=detail&ids=%%s", x.getApiUrl()), pager.List)

	return model.NewSuccess(pager)
}

func (x CmsZyHandler) parseVidTypeId(str string) (vid, tid string, err error) {
	var tmpList = strings.Split(str, "-")
	if len(tmpList) != 2 {
		return "", "", errors.New("请求参数错误")
	}
	vid = tmpList[0]
	tid = tmpList[1]
	return vid, tid, nil
}

func (x CmsZyHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf("%s/?ac=detail&ids=%s", x.getApiUrl(), id))
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
						Group: x.option.GetName(),
					})
				}
			}
			return true
		})
	}

	return model.NewSuccess(video)
}

func (x CmsZyHandler) _source(pid, vid string) interface{} {
	tmpVid, tmpTid, err := x.parseVidTypeId(pid)
	if err != nil {
		return model.NewError(err.Error())
	}
	buff, err := x.httpClient.Get(fmt.Sprintf("%s/?ac=detail&ids=%s", x.getApiUrl(), tmpVid))
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

func (x CmsZyHandler) UpdateHeader(header map[string]string) error {
	return nil
}

func (x CmsZyHandler) HoldCookie() error {
	return nil
}
