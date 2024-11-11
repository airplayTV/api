package handler

import (
	"errors"
	"fmt"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"strings"
	"time"
)

type NaifeiMeHandler struct {
	Handler
}

func (x NaifeiMeHandler) Init() IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.UserAgent, useragent)
	x.httpClient.AddHeader(headers.Origin, yingshiHost)
	x.httpClient.AddHeader(headers.Referer, yingshiHost)
	return x
}

func (x NaifeiMeHandler) Name() string {
	return "奈飞工厂"
}

func (x NaifeiMeHandler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "2"})
	tags = append(tags, gin.H{"name": "电视剧", "value": "1"})
	tags = append(tags, gin.H{"name": "综艺", "value": "3"})
	tags = append(tags, gin.H{"name": "动漫", "value": "4"})
	tags = append(tags, gin.H{"name": "纪录片", "value": "5"})
	return tags
}

func (x NaifeiMeHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x NaifeiMeHandler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x NaifeiMeHandler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x NaifeiMeHandler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x NaifeiMeHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x NaifeiMeHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.requestUrlBypassSafeLineChallenge(netflixgcHost)
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var httpClient = x.httpClient.Clone()
	buff, err = httpClient.Get(netflixgcEcScriptUrl)
	if err != nil {
		return model.NewError("获取加密数据失败")
	}
	var uid = x.simpleRegEx(string(buff), `"Uid": "(\S+)"`)
	if len(uid) == 0 {
		return model.NewError("获取加密数据失败2")
	}
	var ts = time.Now().Unix()
	var params = fmt.Sprintf(
		"type=%s&class=&area=&lang=&version=&state=&letter=&page=%d&time=%d&key=%s",
		tagName,
		x.parsePageNumber(page),
		ts,
		util.StringMd5(fmt.Sprintf("DS%d%s", ts, uid)),
	)
	httpClient.AddHeader(headers.ContentType, "application/x-www-form-urlencoded; charset=UTF-8")
	buff, err = httpClient.Post(netflixgcTagUrl, params)
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var pager = model.Pager{Limit: 40, Page: x.parsePageNumber(page), List: make([]model.Video, 0)}
	var result = gjson.ParseBytes(buff)

	pager.Total = int(result.Get("total").Int())
	pager.Pages = int(result.Get("pagecount").Int())
	pager.Page = int(result.Get("page").Int())
	result.Get("list").ForEach(func(key, value gjson.Result) bool {
		pager.List = append(pager.List, model.Video{
			Id:    value.Get("vod_id").String(),
			Name:  value.Get("vod_name").String(),
			Thumb: value.Get("vod_pic").String(),
			Intro: value.Get("vod_blurb").String(),
		})
		return true
	})

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x NaifeiMeHandler) _search(keyword, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(yingshiSearchUrl, keyword, x.parsePageNumber(page)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var pager = model.Pager{Limit: 20, Page: x.parsePageNumber(page)}

	var result = gjson.ParseBytes(buff)

	pager.Total = int(result.Get("data").Get("Total").Int())
	pager.Pages = int(result.Get("data").Get("TotalPageCount").Int())
	pager.Page = int(result.Get("data").Get("Page").Int())
	pager.Limit = int(result.Get("data").Get("Limit").Int())
	result.Get("data").Get("List").ForEach(func(key, value gjson.Result) bool {
		pager.List = append(pager.List, model.Video{
			Id:    fmt.Sprintf("%s-%s", value.Get("vod_id").String(), value.Get("type_id").String()),
			Name:  value.Get("vod_name").String(),
			Thumb: value.Get("vod_pic").String(),
			Intro: value.Get("vod_blurb").String(),
			Url:   fmt.Sprintf(yingshiDetailUrl, value.Get("vod_id").String(), value.Get("type_id").String()),
		})
		return true
	})

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x NaifeiMeHandler) parseVidTypeId(str string) (vid, tid string, err error) {
	var tmpList = strings.Split(str, "-")
	if len(tmpList) != 2 {
		return "", "", errors.New("请求参数错误")
	}
	vid = tmpList[0]
	tid = tmpList[1]
	return vid, tid, nil
}

func (x NaifeiMeHandler) _detail(id string) interface{} {
	vid, tid, err := x.parseVidTypeId(id)
	if err != nil {
		return model.NewError(err.Error())
	}

	buff, err := x.httpClient.Get(fmt.Sprintf(yingshiDetailUrl, vid, tid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id, Links: make([]model.Link, 0)}

	var result = gjson.ParseBytes(buff)

	video.Name = result.Get("data").Get("vod_name").String()
	video.Thumb = result.Get("data").Get("vod_pic").String()
	video.Intro = result.Get("data").Get("vod_content").String()
	video.Url = fmt.Sprintf(yingshiDetailUrl, vid, tid)
	video.Actors = result.Get("data").Get("vod_content").String()
	result.Get("data").Get("vod_sources").ForEach(func(key, value gjson.Result) bool {
		var tmpSourceId = value.Get("source_id").String()
		var tmpGroup = value.Get("source_name").String()
		if value.Get("vod_play_list").Get("url_count").Int() > 0 {
			value.Get("vod_play_list").Get("urls").ForEach(func(key, value gjson.Result) bool {
				video.Links = append(video.Links, model.Link{
					Id:    fmt.Sprintf("%s-%s", tmpSourceId, value.Get("nid").String()),
					Name:  value.Get("name").String(),
					Url:   value.Get("url").String(),
					Group: tmpGroup,
				})
				return true
			})
		}
		return true
	})

	return model.NewSuccess(video)
}

func (x NaifeiMeHandler) _source(pid, vid string) interface{} {
	tmpVid, tmpTid, err := x.parseVidTypeId(vid)
	if err != nil {
		return model.NewError(err.Error())
	}
	tmpSourceId, tmpNid, err := x.parseVidTypeId(pid)
	if err != nil {
		return model.NewError(err.Error())
	}
	buff, err := x.httpClient.Get(fmt.Sprintf(yingshiDetailUrl, tmpVid, tmpTid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var source = model.Source{Id: pid, Vid: vid}

	var result = gjson.ParseBytes(buff)

	source.Name = result.Get("data").Get("vod_name").String()
	source.Thumb = result.Get("data").Get("vod_pic").String()
	result.Get("data").Get("vod_sources").ForEach(func(key, value gjson.Result) bool {
		if value.Get("source_id").String() == tmpSourceId && value.Get("vod_play_list").Get("url_count").Int() > 0 {
			value.Get("vod_play_list").Get("urls").ForEach(func(key, value gjson.Result) bool {
				if tmpNid == value.Get("nid").String() {
					source.Url = value.Get("url").String()
					source.Source = value.Get("url").String()
					source.Type = x.parseVideoType(source.Source)
				}
				return true
			})
		}
		return true
	})

	if len(source.Url) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(source)
}
