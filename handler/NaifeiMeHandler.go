package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/dop251/goja"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"net/url"
	"path/filepath"
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
	var pager = model.Pager{Limit: 10, Page: x.parsePageNumber(page), List: make([]model.Video, 0)}

	buff, err := x.httpClient.Get(fmt.Sprintf(netflixgcSearchUrl, keyword, pager.Page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc.Find(".row-right .search-box").Each(func(i int, selection *goquery.Selection) {
		pager.List = append(pager.List, model.Video{
			Id:         x.simpleRegEx(selection.Find(".public-list-exp").AttrOr("href", ""), `(\d+)`),
			Name:       strings.TrimSpace(selection.Find(".thumb-txt").Text()),
			Thumb:      selection.Find(".gen-movie-img").AttrOr("data-src", ""),
			Intro:      strings.TrimSpace(selection.Find(".thumb-blurb").Text()),
			Url:        "",
			Actors:     "",
			Tag:        selection.Find(".public-list-prb").Text(),
			Resolution: "",
			Links:      nil,
		})
	})

	var matches = x.simpleRegExList(doc.Find(".pages .page-tip").Text(), `共(\d+)条数据,当前(\d+)\/(\d+)页`)
	if len(matches) == 4 {
		pager.Total = x.parsePageNumber(matches[1])
		pager.Page = x.parsePageNumber(matches[2])
		pager.Pages = x.parsePageNumber(matches[3])
	} else {
		doc.Find(".row-right .page-link").Each(func(i int, selection *goquery.Selection) {
			var p = x.parsePageNumber(x.simpleRegEx(selection.AttrOr("href", ""), `-(\d+)---.html`))
			if p >= pager.Pages {
				pager.Pages = p
				pager.Total = pager.Pages * pager.Limit
			}
		})
	}

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
		if strings.Contains(tmpHtml, "badge") {
			tmpHtml = x.simpleRegEx(tmpHtml, `i>(\S+)<span class="badge"`)
		} else {
			tmpHtml = selection.Text()
		}
		tmpHtml = strings.TrimSpace(strings.ReplaceAll(tmpHtml, "&nbsp;", ""))
		groupMap = append(groupMap, tmpHtml)
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

	source.Source, err = x.parseNetflixGCEncryptedUrl(gjson.Parse(playerAAA))
	if err != nil {
		return model.NewError("解析加密数据失败：" + err.Error())
	}

	source.Type = x.parseVideoType(source.Source)
	source.Url = source.Source

	return model.NewSuccess(source)
}

func (x NaifeiMeHandler) parseNetflixGCEncryptedUrl(playerAAAJson gjson.Result) (string, error) {
	var tmpParse = ""
	var tmpUrl = playerAAAJson.Get("url").String()
	var tmpServer = playerAAAJson.Get("server").String()
	var tmpFrom = playerAAAJson.Get("from").String()
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
	playerConfig, playerList, _, serverList, err := x.getPlayerConfig()
	if err != nil {
		return "", err
	}
	if serverList.Get(tmpServer).Exists() {
		tmpServer = serverList.Get(tmpServer).Get("des").String()
	}
	if playerList.Get(tmpFrom).Exists() {
		if playerList.Get(tmpFrom).Get("ps").String() == "1" {
			if playerList.Get(tmpFrom).Get("parse").String() == "" {
				tmpParse = playerConfig.Get("parse").String()
			} else {
				tmpParse = playerList.Get(tmpFrom).Get("parse").String()
			}
			tmpFrom = "parse"
		}
	}
	//var tmpParseUrl = fmt.Sprintf("%s/static/player/%s.js", strings.TrimRight(netflixgcHost, "/"), tmpFrom)
	//log.Println("[tmpParseUrl]", tmpParseUrl)
	var tmpYul = fmt.Sprintf("%s%s", tmpParse, tmpUrl)
	buff, err := x.httpClient.Get(tmpYul)
	if err != nil {
		return "", err
	}
	var findConfig = x.simpleRegEx(string(buff), `let ConFig = ([\s\S]*?),box`)
	// 解密如下数据
	//log.Println("[UID]", gjson.Parse(findConfig).Get("config").Get("uid").String())
	//log.Println("[url]", gjson.Parse(findConfig).Get("url").String())
	tmpUrl, err = x.fuckNotGmCrypto(
		gjson.Parse(findConfig).Get("config").Get("uid").String(),
		gjson.Parse(findConfig).Get("url").String(),
	)
	if err != nil {
		return "", err
	}

	return tmpUrl, nil
}

func (x NaifeiMeHandler) getPlayerConfig() (gjson.Result, gjson.Result, gjson.Result, gjson.Result, error) {
	var g gjson.Result
	buff, err := x.httpClient.Get(fmt.Sprintf("https://www.netflixgc.com/static/js/playerconfig.js?t=%s", time.Now().Format("20060102")))
	if err != nil {
		return g, g, g, g, err
	}
	var playerConfig = x.simpleRegEx(string(buff), `MacPlayerConfig=(\S+);`)
	var matches = x.simpleRegExList(string(buff), `MacPlayerConfig.player_list=(\S+),MacPlayerConfig.downer_list=(\S+),MacPlayerConfig.server_list=(\S+);`)
	if len(matches) != 4 {
		return g, g, g, g, errors.New("匹配异常")
	}
	return gjson.Parse(playerConfig), gjson.Parse(matches[1]), gjson.Parse(matches[2]), gjson.Parse(matches[3]), nil
}

func (x NaifeiMeHandler) fuckNotGmCrypto(uid, data string) (string, error) {
	var scriptBuff = append(util.ReadFile(filepath.Join(util.AppPath(), "file/NotGm.js")))
	vm := goja.New()
	_, err := vm.RunString(string(scriptBuff))
	if err != nil {
		//log.Println("[LoadGojaError]", err.Error())
		return "", err
	}
	var fuckCryptoDecode func(uid, data string) string
	err = vm.ExportTo(vm.Get("fuckCryptoDecode"), &fuckCryptoDecode)
	if err != nil {
		//log.Println("[ExportGojaFnError]", err.Error())
		return "", err
	}
	var result = fuckCryptoDecode(uid, data)
	return result, nil
}

func (x NaifeiMeHandler) UpdateHeader(header map[string]string) error {
	if header == nil {
		return errors.New("header数据不能为空")
	}
	var tmpHttpClient = util.HttpClient{}
	tmpHttpClient.SetHeaders(x.httpClient.GetHeaders())
	for key, value := range header {
		tmpHttpClient.AddHeader(key, value)
	}

	// 请求数据并检测Cookie是否可用
	switch x.Search("我的", "1").(type) {
	case model.Success:
		// 如果可用则设置到当前上下文的http请求头
		x.httpClient.SetHeaders(tmpHttpClient.GetHeaders())

		_ = util.SaveHttpHeader(x.Name(), tmpHttpClient.GetHeaders())

		return nil
	default:
		return errors.New("cookie无效")
	}
}

func (x NaifeiMeHandler) HoldCookie() error {
	switch r := x.Search("我的", "1").(type) {
	case model.Success:
		return nil
	case model.Error:
		return errors.New(r.Msg)
	}
	return errors.New("未知错误")
}

// https://www.netflixgc.com/static/player/parse.js
