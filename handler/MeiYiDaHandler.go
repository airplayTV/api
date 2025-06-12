package handler

import (
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/dop251/goja"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

type MeiYiDaHandler struct {
	Handler
}

func (x MeiYiDaHandler) Init(options interface{}) IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.UserAgent, useragent)
	x.httpClient.AddHeader(headers.Origin, mayiHost)
	x.httpClient.AddHeader(headers.Referer, mayiHost)
	return x
}

func (x MeiYiDaHandler) Name() string {
	return "美益达影视"
}

func (x MeiYiDaHandler) Option() model.CmsZyOption {
	return x.option
}

func (x MeiYiDaHandler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "1"})
	tags = append(tags, gin.H{"name": "电视剧", "value": "2"})
	tags = append(tags, gin.H{"name": "综艺", "value": "3"})
	tags = append(tags, gin.H{"name": "动漫", "value": "4"})
	tags = append(tags, gin.H{"name": "短剧", "value": "5"})
	return tags
}

func (x MeiYiDaHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x MeiYiDaHandler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x MeiYiDaHandler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x MeiYiDaHandler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x MeiYiDaHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x MeiYiDaHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(meiyidaTagUrl, tagName, x.parsePageNumber(page)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var pager = model.Pager{Limit: 72, Page: x.parsePageNumber(page), List: make([]model.Video, 0)}

	doc.Find(".module .module-item").Each(func(i int, selection *goquery.Selection) {
		name := selection.Find(".module-poster-item-title").Text()
		tmpUrl, _ := selection.Attr("href")
		thumb, _ := selection.Find(".lazyload").Attr("data-original")
		tag := selection.Find(".module-item-note").Text()

		pager.List = append(pager.List, model.Video{
			Id:    x.simpleRegEx(tmpUrl, `(\d+)`),
			Name:  name,
			Thumb: thumb,
			Url:   tmpUrl,
			Tag:   tag,
		})
	})

	var maxPage = 0
	doc.Find("#page .page-link").Each(func(i int, selection *goquery.Selection) {
		tmpUrl, _ := selection.Attr("href")
		tmpNumber := x.parsePageNumber(x.simpleRegEx(tmpUrl, `--------(\d+)---.html`))
		if tmpNumber >= maxPage {
			maxPage = tmpNumber
		}
	})
	pager.Total = pager.Limit*maxPage + 1

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x MeiYiDaHandler) _search(keyword, page string) interface{} {
	var pager = model.Pager{Limit: 10, Page: x.parsePageNumber(page)}
	buff, err := x.httpClient.Get(fmt.Sprintf(meiyidaSearchUrl, keyword, pager.Page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		log.Println("[文档解析失败]", err.Error())
		return pager
	}
	//log.Println("[SSSSS]", string(buff))
	doc.Find(".module .module-item").Each(func(i int, selection *goquery.Selection) {
		name := selection.Find(".module-card-item-title").Text()
		tmpUrl, _ := selection.Find(".module-card-item-title a").Attr("href")
		thumb, _ := selection.Find(".lazyload").Attr("data-original")
		tag := selection.Find(".module-item-note").Text()
		//intro := selection.Find(".module-info-item-content").Text()

		pager.List = append(pager.List, model.Video{
			Id:         x.simpleRegEx(tmpUrl, `(\d+)`),
			Name:       strings.TrimSpace(name),
			Thumb:      thumb,
			Url:        tmpUrl,
			Tag:        tag,
			Resolution: tag,
		})
	})

	var maxPage = 0
	doc.Find("#page .page-link").Each(func(i int, selection *goquery.Selection) {
		tmpUrl, _ := selection.Attr("href")
		tmpNumber := x.parsePageNumber(x.simpleRegEx(tmpUrl, `--------(\d+)---.html`))
		if tmpNumber >= maxPage {
			maxPage = tmpNumber
		}
	})
	pager.Total = pager.Limit*maxPage + 1

	return model.NewSuccess(pager)
}

func (x MeiYiDaHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(meiyidaDetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	log.Println("[DOC]", string(buff))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id}

	var tmpSelection = doc.Find(".module .module-main")
	{
		video.Name = tmpSelection.Find(".module-info-heading h1").Text()
		video.Intro = strings.TrimSpace(tmpSelection.Find(".module-info-introduction-content").Text())
		video.Thumb, _ = tmpSelection.Find(".module-item-cover .lazyload").Attr("data-original")
		video.Url = fmt.Sprintf(meiyidaDetailUrl, id)
		video.Tag = strings.TrimSpace(tmpSelection.Find(".module-info-item-content").Text())
	}

	var groupList = make([]string, 0)
	doc.Find("#y-playList .tab-item").Each(func(i int, selection *goquery.Selection) {
		groupList = append(groupList, strings.TrimSpace(selection.AttrOr("data-dropdown-value", "")))
	})
	doc.Find(".module .module-list.his-tab-list").Each(func(i int, selection *goquery.Selection) {
		var tmpGroup = groupList[i]
		selection.Find(".module-play-list .module-play-list-link").Each(func(j int, selection *goquery.Selection) {
			tmpUrl := selection.AttrOr("href", "")
			video.Links = append(video.Links, model.Link{
				Id:    x.simpleRegEx(tmpUrl, `(\d+-\d+-\d+)`),
				Name:  selection.Find("span").Text(),
				Url:   tmpUrl,
				Group: tmpGroup,
			})
		})
	})

	return model.NewSuccess(video)
}

func (x MeiYiDaHandler) _source(pid, vid string) interface{} {
	var source = model.Source{Id: pid, Vid: vid}
	h, buff, err := x.httpClient.GetResponse(fmt.Sprintf(meiyidaPlayUrl, pid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var findJson = x.simpleRegEx(string(buff), `player_aaaa=(\{[\s\S]*?\})</script>`)
	var result = gjson.Parse(findJson)
	source.Url = result.Get("url").String()
	source.Name = doc.Find(".module-player-side .module-info-heading a").AttrOr("title", "")
	source.Thumb = x.simpleRegEx(string(buff), `vod_image='(\S+)',vod_url`)

	var _type = result.Get("encrypt").Int()
	switch _type {
	case 1:
		source.Url = url.QueryEscape(source.Url)
		break
	case 2:
		tmpBuff, _ := base64.StdEncoding.DecodeString(source.Url)
		source.Url = url.QueryEscape(string(tmpBuff))
		break
	default:
		break
	}

	source.Source = result.Get("url").String()
	source.Url = x.handleEncryptUrl(fmt.Sprintf("%s/player/?type=%d&url=%s", meiyidaHost, _type, source.Url), result, h)
	source.Type = x.parseVideoType(source.Url)

	if len(source.Url) == 0 {
		return model.NewError("解析加密数据失败：" + err.Error())
	}

	return model.NewSuccess(source)
}

func (x MeiYiDaHandler) UpdateHeader(header map[string]string) error {
	return nil
}

func (x MeiYiDaHandler) HoldCookie() error {
	return nil
}

func (x MeiYiDaHandler) handleEncryptUrl(playFrameUrl string, playerAAA gjson.Result, header http.Header) string {

	var parse = ""
	var playServer = playerAAA.Get("server").String()
	var playFrom = playerAAA.Get("from").String()
	var playUrl = playerAAA.Get("url").String()
	if playServer == "no" {
		playServer = ""
	}

	// 获取配置
	b, err := x.httpClient.Get(fmt.Sprintf("%s/static/js/playerconfig.js?t=20250106", meiyidaHost))
	if err != nil {
		log.Println("[内容获取失败]", err.Error())
		return ""
	}
	var jsText = string(b)
	var findPlayerConfig = x.simpleRegEx(jsText, `MacPlayerConfig=(\S+);`)
	var findPlayerList = x.simpleRegEx(jsText, `MacPlayerConfig.player_list=(\S+),MacPlayerConfig.downer_list`)
	var findServerList = x.simpleRegEx(jsText, `MacPlayerConfig.server_list=(\S+);`)

	var playerConfigJson = gjson.Parse(findPlayerConfig)
	var playerListJson = gjson.Parse(findPlayerList)
	var serverListJson = gjson.Parse(findServerList)

	serverListJson.ForEach(func(key, value gjson.Result) bool {
		if playServer == key.String() {
			playServer = value.Get("des").String()
		}
		return true
	})

	playerListJson.ForEach(func(key, value gjson.Result) bool {
		if playFrom == key.String() {
			if value.Get("ps").String() == "1" {
				parse = value.Get("parse").String()
				if value.Get("parse").String() == "" {
					parse = playerConfigJson.Get("parse").String()
				}
				playFrom = "parse"
			}
		}
		return true
	})

	// MacPlayer.Parse + MacPlayer.PlayUrl
	var reqUrl = fmt.Sprintf("%s/%s%s", strings.TrimRight(meiyidaHost, "/"), strings.TrimLeft(parse, "/"), playUrl)

	// 需要带Cookie
	x.httpClient.AddHeader(headers.Cookie, header.Get("Set-Cookie"))

	// 获取配置
	b, err = x.httpClient.Get(reqUrl)
	if err != nil {
		log.Println("[内容获取失败]", err.Error())
		return ""
	}
	//var findConfig = util.SimpleRegEx(string(b), `var config = ([\S\s]+)YKQ.start();`)
	var findConfig = x.simpleRegEx(string(b), `var config = (\{[\s\S]*?\})`)
	var configJson = gjson.Parse(findConfig)
	if !configJson.Get("url").Exists() {
		log.Println("[config.parse.error]")
		return ""
	}

	log.Println("[config.url]", configJson.Get("url"), configJson.Get("id"))

	// key来源：https://meiyd11.com/player/js/setting.js?v=4
	//// https://meiyd11.com/static/js/playerconfig.js?t=20240923
	return x.fuckRc4(configJson.Get("url").String(), "202205051426239465", 1)
}

func (x MeiYiDaHandler) fuckRc4(data, key string, t int) string {
	var scriptBuff = append(
		util.ReadFile(filepath.Join(util.AppPath(), "file/base64-polyfill.js")),
		util.ReadFile(filepath.Join(util.AppPath(), "file/fuck-crypto-bridge-myd.js"))...,
	)
	vm := goja.New()
	_, err := vm.RunString(string(scriptBuff))

	if err != nil {
		log.Println("[LoadGojaError]", err.Error())
		return ""
	}

	var rc4Decode func(string, string, int) string
	err = vm.ExportTo(vm.Get("rc4Decode"), &rc4Decode)
	if err != nil {
		log.Println("[ExportGojaFnError]", err.Error())
		return ""
	}

	return rc4Decode(data, key, t)
}
