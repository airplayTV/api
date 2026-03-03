package handler

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/airplayTV/api/util"
	"github.com/lixiang4u/goWebsocket"

	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/gin-gonic/gin"
)

type JinpaiHandler struct {
	Handler
}

func (x JinpaiHandler) Init(options interface{}) model.IVideo {
	x.option = options.(model.CmsZyOption)
	x.httpClient = util.HttpClient{}
	return x
}

func (x JinpaiHandler) Name() string {
	// jpyy2.com
	// https://610pkea.com/
	return "金牌影院"
}

func (x JinpaiHandler) Option() model.CmsZyOption {
	return x.option
}

func (x JinpaiHandler) TagList() interface{} {
	var sep = "," // type,region,state,filter
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "全部", "value": strings.Join([]string{"*", "*", "*", "*"}, sep)})
	tags = append(tags, gin.H{"name": "国漫", "value": strings.Join([]string{"*", "cn", "*", "*"}, sep)})
	tags = append(tags, gin.H{"name": "日漫", "value": strings.Join([]string{"*", "jp", "*", "*"}, sep)})
	tags = append(tags, gin.H{"name": "少儿", "value": strings.Join([]string{"shaoer", "*", "*", "*"}, sep)})
	tags = append(tags, gin.H{"name": "儿童", "value": strings.Join([]string{"ertongxiang", "*", "*", "*"}, sep)})
	tags = append(tags, gin.H{"name": "亲子", "value": strings.Join([]string{"qinzi", "*", "*", "*"}, sep)})
	tags = append(tags, gin.H{"name": "音乐", "value": strings.Join([]string{"yinyue", "*", "*", "*"}, sep)})
	tags = append(tags, gin.H{"name": "高达", "value": strings.Join([]string{"gaoda", "*", "*", "*"}, sep)})
	return tags
}

func (x JinpaiHandler) VideoList(tag, page string) interface{} {
	var key = fmt.Sprintf("xgct-video-list::%s_%s_%s", x.Name(), tag, page)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._videoList(tag, page)
	})
}

func (x JinpaiHandler) Search(keyword, page string) interface{} {
	var key = fmt.Sprintf("xgct-video-search::%s_%s_%s", x.Name(), keyword, page)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._search(keyword, page)
	})
}

func (x JinpaiHandler) Detail(id string) interface{} {
	var key = fmt.Sprintf("xgct-video-detail::%s_%s", x.Name(), id)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._detail(id)
	})
}

func (x JinpaiHandler) Source(pid, vid string) interface{} {
	var key = fmt.Sprintf("xgct-video-source::%s_%s_%s", x.Name(), pid, vid)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*2), func() interface{} {
		return x._source(pid, vid)
	})
}

func (x JinpaiHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x JinpaiHandler) _videoList(tagName, page string) interface{} {
	var pager = model.Pager{Limit: 24, Page: x.parsePageNumber(page)}

	buff, err := x.httpClient.Get(fmt.Sprintf(jinpaiTagUrl, tagName, pager.Page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc.Find(".movie-ul .content-card a").Each(func(i int, selection *goquery.Selection) {
		pager.List = append(pager.List, model.Video{
			Id:         x.simpleRegEx(selection.AttrOr("href", ""), `/detail/(\d+)`),
			Name:       selection.Find(".card-info .title").Text(),
			Thumb:      selection.Find(".card-img img").AttrOr("src", ""),
			Url:        selection.AttrOr("href", ""),
			Actors:     selection.Find(".card-info .role span").Text(),
			Tag:        selection.Find(".info-tag").Text(),
			Resolution: selection.Find(".tag").Text(),
		})
	})

	doc.Find(".pagination > div > div").Each(func(i int, selection *goquery.Selection) {
		var n = x.parsePageNumber(selection.AttrOr("page", ""))
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
		if n >= pager.Pages {
			pager.Pages = n
		}
	})

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x JinpaiHandler) _search(keyword, page string) interface{} {
	var pager = model.Pager{Limit: 100, Pages: 1, Page: x.parsePageNumber(page)}
	buff, err := x.httpClient.Get(fmt.Sprintf(xgctSearchUrl, url.QueryEscape(keyword)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("文档解析失败：" + err.Error())
	}

	doc.Find(".search .topic-list .topic-list-box").Each(func(i int, selection *goquery.Selection) {
		pager.List = append(pager.List, model.Video{
			Id:    x.simpleRegEx(selection.Find(".topic-list-item").AttrOr("href", ""), `/detail/(\S+)`),
			Name:  strings.TrimSpace(selection.Find(".h3.mb12").Text()),
			Thumb: selection.Find("amp-img").AttrOr("src", ""),
			// https://cn.xgcartoon.com/detail/zhongshengdoushitianzun_dongtaimanhua4k-yuanmandongman
			Url: fmt.Sprintf("https://cn.xgcartoon.com/%s", strings.TrimLeft(selection.Find(".topic-list-item").AttrOr("href", ""), "/")),
		})
	})

	return model.NewSuccess(pager)
}

func (x JinpaiHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(jinpaiDetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	//doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	//if err != nil {
	//	return model.NewError("获取数据失败：" + err.Error())
	//}

	var video = model.Video{Id: id, Url: fmt.Sprintf(jinpaiDetailUrl, id), Links: make([]model.Link, 0)}

	result, err := x.parseVideoJson(buff, id)
	if err != nil {
		return model.NewError(err.Error())
	}

	video.Name = result.Get("vodName").String()
	video.Thumb = result.Get("vodPic").String()
	video.Intro = util.HtmlToText(result.Get("vodContent").String())
	result.Get("episodeList").ForEach(func(key, value gjson.Result) bool {
		video.Links = append(video.Links, model.Link{
			Id:    value.Get("nid").String(),
			Name:  value.Get("name").String(),
			Url:   value.Get("playUrl").String(),
			Group: "金牌影院播放器",
		})
		return true
	})

	if len(video.Name) <= 0 || len(video.Links) <= 0 {
		return model.NewError("获取数据失败")
	}

	return model.NewSuccess(video)
}

func (x JinpaiHandler) parseVideoJson(html []byte, vid string) (result gjson.Result, err error) {
	var scanner = bufio.NewScanner(bytes.NewReader(html))
	for scanner.Scan() {
		var tmpLine = strings.TrimSpace(scanner.Text())
		if !strings.Contains(tmpLine, "episodeList") || !strings.Contains(tmpLine, "searchParams") {
			continue
		}
		for _, tmpLine2 := range strings.Split(tmpLine, `<script>`) {
			if !strings.Contains(tmpLine2, "vodId") || !strings.Contains(tmpLine2, vid) {
				continue
			}
			tmpLine2 = strings.ReplaceAll(tmpLine2, `\"`, `"`)
			var prefix = x.simpleRegEx(tmpLine2, `self\.__next_f\.push\(\[1,"(\S+?):\[`)
			var suffix = `\n"])</script>`
			var tmpRegEx = fmt.Sprintf(`%s([\S\s]+)%s`, regexp.QuoteMeta(fmt.Sprintf("%s:", prefix)), regexp.QuoteMeta(suffix))
			result = gjson.Parse(x.simpleRegEx(tmpLine2, tmpRegEx))
		}
	}

	if !result.IsArray() || !result.Array()[3].Exists() {
		err = errors.New("数据解析失败1")
		return
	}
	result = result.Array()[3].Get("data").Get("data")
	if !result.Exists() {
		err = errors.New("数据解析失败2")
		return
	}

	return
}

func (x JinpaiHandler) _source(pid, vid string) interface{} {
	var source = model.Source{Id: pid, Vid: vid}

	buff, err := x.httpClient.Get(fmt.Sprintf(xgctPlayUrl, vid, pid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var iframe = doc.Find(".video-iframe iframe").AttrOr("src", "")
	log.Println("[iframe]", iframe)

	// https://pframe.xgcartoon.com/player.htm?vid=89ac1cfc-6fb3-4c1f-9602-d25fda7f151e&amp;autoplay=false
	// https://xgct-video.bzcdn.net/89ac1cfc-6fb3-4c1f-9602-d25fda7f151e/playlist.m3u8
	var guid = x.simpleRegEx(iframe, `vid=(\S+)&`)
	if len(guid) <= 0 {
		return model.NewError("没有解析到播放地址")
	}

	source.Name = strings.TrimSpace(doc.Find(".breadcrumb a.breadcrumb-item").Last().Text())
	source.Source = fmt.Sprintf("https://xgct-video.bzcdn.net/%s/playlist.m3u8", guid)
	source.Url = source.Source
	// 视频类型问题处理
	source.Type = x.parseVideoType(source.Source)

	if len(source.Source) == 0 {
		return model.NewError("播放地址处理失败")
	}

	return model.NewSuccess(source)
}

func (x JinpaiHandler) UpdateHeader(header map[string]string) error {
	if header == nil {
		return errors.New("header数据不能为空")
	}
	for key, value := range header {
		x.httpClient.AddHeader(key, value)
	}

	// 请求数据并检测Cookie是否可用
	var resp = x._videoList("movie", "1")
	switch resp.(type) {
	case model.Success:
		// 如果可用则设置到当前上下文的http请求头
		//_ = util.SaveHttpHeader(x.Name(), header)
		return nil
	default:
		log.Println("[UpdateHeaderErr]", x.Name(), goWebsocket.ToJson(resp))
		return errors.New("cookie无效")
	}
}

func (x JinpaiHandler) HoldCookie() error {
	var resp = x._videoList("movie", "1")
	switch r := resp.(type) {
	case model.Success:
		return nil
	case model.Error:
		return errors.New(r.Msg)
	}
	return errors.New("未知错误")
}
