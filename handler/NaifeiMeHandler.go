package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"log"
	"net/url"
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
	return "奈飞工厂[烂站王中王]"
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
	buff, err := x.requestUrlBypassSafeLineChallenge(fmt.Sprintf(netflixgcDetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id}
	video.Name = doc.Find(".vod-detail .slide-info-title").Text()
	video.Thumb = doc.Find(".vod-detail .detail-pic .lazy").AttrOr("data-src", "")
	video.Intro = strings.TrimSpace(doc.Find("#height_limit").Text())

	var groupMap = make([]string, 0)
	doc.Find(".nav-swiper a").Each(func(i int, selection *goquery.Selection) {
		tmpHtml, _ := selection.Html()
		tmpHtml = strings.ReplaceAll(x.simpleRegEx(tmpHtml, `i>(\S+)<span class="badge"`), "&nbsp;", "")
		groupMap = append(groupMap, strings.TrimSpace(tmpHtml))
	})

	doc.Find(".anthology-list-box .anthology-list-play").Each(func(i int, selection *goquery.Selection) {
		if i >= len(groupMap) {
			return
		}
		var tmpGroup = groupMap[i]
		selection.Find(".box a").Each(func(i int, selection *goquery.Selection) {
			video.Links = append(video.Links, model.Link{
				Id:    x.simpleRegEx(selection.AttrOr("href", ""), `/play/(\d+-\d+-\d+).html`),
				Name:  strings.TrimSpace(selection.Text()),
				Url:   selection.AttrOr("href", ""),
				Group: tmpGroup,
			})
		})
	})

	return model.NewSuccess(video)
}

func (x NaifeiMeHandler) _source(pid, vid string) interface{} {
	buff, err := x.requestUrlBypassSafeLineChallenge(fmt.Sprintf(netflixgcPlayUrl, pid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var source = model.Source{Id: pid, Vid: vid}
	source.Name = doc.Find(".plist-body .player-title-link").Text()
	source.Thumb = doc.Find(".player-vod-no1 .lazy").AttrOr("data-src", "")

	var playerAAA = x.simpleRegEx(string(buff), `var player_aaaa=(\{[\s\S]*?\})</script>`)
	log.Println("[playerAAA]", playerAAA)

	x.parseNetflixGCEncryptedUrl(gjson.Parse(playerAAA))

	if len(source.Url) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(source)
}

func (x NaifeiMeHandler) parseNetflixGCEncryptedUrl(playerAAAJson gjson.Result) {
	var tmpUrl = playerAAAJson.Get("url").String()
	var tmpServer = playerAAAJson.Get("server").String()
	if playerAAAJson.Get("encrypt").String() == "1" {
		tmpUrl, _ = url.QueryUnescape(tmpUrl)
	}
	if playerAAAJson.Get("encrypt").String() == "2" {
		decodeString, err := base64.StdEncoding.DecodeString(tmpUrl)
		if err == nil {
			tmpUrl, _ = url.QueryUnescape(string(decodeString))
		}
	}
	if tmpServer == "no" {
		tmpServer = ""
	}

	log.Println("[tmpUrl]", tmpUrl)
}
