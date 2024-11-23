package handler

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"log"
	"strings"
)

type MaYiHandler struct {
	Handler
}

func (x MaYiHandler) Init() IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.UserAgent, useragent)
	x.httpClient.AddHeader(headers.Origin, mayiHost)
	x.httpClient.AddHeader(headers.Referer, mayiHost)
	return x
}

func (x MaYiHandler) Name() string {
	return "蚂蚁影视"
}

func (x MaYiHandler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "1"})
	tags = append(tags, gin.H{"name": "电视剧", "value": "2"})
	tags = append(tags, gin.H{"name": "综艺", "value": "3"})
	tags = append(tags, gin.H{"name": "动漫", "value": "4"})
	return tags
}

func (x MaYiHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x MaYiHandler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x MaYiHandler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x MaYiHandler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x MaYiHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x MaYiHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(mayiTagUrl, tagName, x.parsePageNumber(page)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	log.Println("[request]", fmt.Sprintf(mayiTagUrl, tagName, x.parsePageNumber(page)))
	log.Println("[html]", string(buff))

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var pager = model.Pager{Limit: 36, Page: x.parsePageNumber(page), List: make([]model.Video, 0)}

	doc.Find(".stui-vodlist .stui-vodlist__box").Each(func(i int, selection *goquery.Selection) {
		name := selection.Find(".title a").Text()
		tmpUrl, _ := selection.Find(".title a").Attr("href")
		thumb, _ := selection.Find(".stui-vodlist__thumb").Attr("data-original")
		tag := selection.Find(".stui-vodlist__thumb .pic-text").Text()
		actors := selection.Find("p.text").Text()
		resolution := selection.Find(".stui-vodlist__thumb .pic-text").Text()

		pager.List = append(pager.List, model.Video{
			Id:         x.simpleRegEx(tmpUrl, `(\d+)`),
			Name:       name,
			Thumb:      thumb,
			Url:        tmpUrl,
			Actors:     strings.TrimSpace(actors),
			Tag:        tag,
			Resolution: resolution,
		})
	})

	var pages = doc.Find(".stui-page .num").Text()
	var matches = x.simpleRegExList(pages, `(\d+)/(\d+)`)
	if len(matches) == 3 {
		pager.Pages = x.parsePageNumber(matches[2])
		pager.Page = x.parsePageNumber(matches[1])
		pager.Total = pager.Limit * pager.Pages
	} else {
		pager.Pages = 1
		pager.Page = 1
		pager.Total = pager.Limit
	}

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x MaYiHandler) _search(keyword, page string) interface{} {
	var pager = model.Pager{Limit: 10, Page: x.parsePageNumber(page)}
	buff, err := x.httpClient.Get(fmt.Sprintf(mayiSearchUrl, keyword, pager.Page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		log.Println("[文档解析失败]", err.Error())
		return pager
	}
	doc.Find(".col-lg-wide-75 .stui-vodlist__media li").Each(func(i int, selection *goquery.Selection) {
		name := selection.Find(".title a").Text()
		tmpUrl, _ := selection.Find(".title a").Attr("href")
		thumb, _ := selection.Find(".v-thumb").Attr("data-original")
		tag := selection.Find(".pic-text").Text()
		pager.List = append(pager.List, model.Video{
			Id:    x.simpleRegEx(tmpUrl, `(\d+)`),
			Name:  name,
			Thumb: thumb,
			Url:   tmpUrl,
			Tag:   tag,
		})
	})

	var pages = doc.Find(".stui-page .num").Text()
	var matches = x.simpleRegExList(pages, `(\d+)/(\d+)`)
	if len(matches) == 3 {
		pager.Pages = x.parsePageNumber(matches[2])
		pager.Page = x.parsePageNumber(matches[1])
		pager.Total = pager.Limit * pager.Pages
	} else {
		pager.Pages = 1
		pager.Page = 1
		pager.Total = pager.Limit
	}

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x MaYiHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(mayiDetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id}

	var groupMap = make(map[string]string)
	doc.Find(".play_source .title").Each(func(i int, selection *goquery.Selection) {
		groupMap[selection.AttrOr("data-mid", "")] = strings.TrimSpace(selection.Text())
	})
	doc.Find(".play_source_list .play_source_list_item").Each(func(i int, selection *goquery.Selection) {
		var tmpGroup = fmt.Sprintf("group_%s", selection.AttrOr("data-mid", ""))
		if v, ok := groupMap[selection.AttrOr("data-mid", "")]; ok {
			tmpGroup = strings.TrimSpace(v)
		}
		selection.Find("li a").Each(func(i int, selection *goquery.Selection) {
			video.Links = append(video.Links, model.Link{
				Id:    x.simpleRegEx(selection.AttrOr("href", ""), `/vodplay/(\d+-\d+-\d+).html`),
				Name:  strings.TrimSpace(selection.Text()),
				Url:   selection.AttrOr("href", ""),
				Group: tmpGroup,
			})
		})
	})

	video.Thumb = doc.Find(".col-md-wide-75 .lazyload").AttrOr("data-original", "")
	video.Name = doc.Find(".col-md-wide-75 .picture").AttrOr("title", "")
	video.Intro = strings.TrimSpace(doc.Find(".col-md-wide-75 .detail-content").Text())
	video.Tag = strings.TrimSpace(doc.Find(".col-md-wide-75 .pic-text").Text())

	return model.NewSuccess(video)
}

func (x MaYiHandler) _source(pid, vid string) interface{} {
	var source = model.Source{Id: pid, Vid: vid}
	buff, err := x.httpClient.Get(fmt.Sprintf(mayiPlayUrl, pid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	source.Name = doc.Find(".stui-player__detail .title").Text()

	var playerDataStr = x.simpleRegEx(string(buff), `var player_data=(\S+)</script>`)
	var jsonResult = gjson.Parse(playerDataStr)
	var encryptedUrl = jsonResult.Get("url").String()

	//log.Println("[playerDataStr]", playerDataStr)
	//log.Println("[encryptedUrl]", encryptedUrl)

	buff, err = x.httpClient.Get(fmt.Sprintf(mayiParseUrl, encryptedUrl))
	if err != nil {
		return model.NewError("获取解析数据失败：" + err.Error())
	}

	source.Source = x.simpleRegEx(string(buff), `var video_url = '(\S+)';`)
	source.Type = x.parseVideoType(source.Source)
	source.Url = source.Source

	if len(source.Source) == 0 {
		return model.NewError("播放地址处理失败")
	}

	return model.NewSuccess(source)
}

func (x MaYiHandler) _getPlayInfo(pid string) interface{} {
	return nil
}

func (x MaYiHandler) UpdateHeader(header map[string]string) error {
	return nil
}

func (x MaYiHandler) HoldCookie() error {
	return nil
}
