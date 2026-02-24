package handler

import (
	"errors"
	"fmt"
	"github.com/airplayTV/api/util"
	"github.com/lixiang4u/goWebsocket"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type XgCartoonHandler struct {
	Handler
}

func (x XgCartoonHandler) Init(options interface{}) model.IVideo {
	x.option = options.(model.CmsZyOption)
	x.httpClient = util.HttpClient{}
	return x
}

func (x XgCartoonHandler) Name() string {
	return "西瓜卡通"
}

func (x XgCartoonHandler) Option() model.CmsZyOption {
	return x.option
}

func (x XgCartoonHandler) TagList() interface{} {
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

func (x XgCartoonHandler) VideoList(tag, page string) interface{} {
	var key = fmt.Sprintf("xgct-video-list::%s_%s_%s", x.Name(), tag, page)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._videoList(tag, page)
	})
}

func (x XgCartoonHandler) Search(keyword, page string) interface{} {
	var key = fmt.Sprintf("xgct-video-search::%s_%s_%s", x.Name(), keyword, page)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._search(keyword, page)
	})
}

func (x XgCartoonHandler) Detail(id string) interface{} {
	var key = fmt.Sprintf("xgct-video-detail::%s_%s", x.Name(), id)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._detail(id)
	})
}

func (x XgCartoonHandler) Source(pid, vid string) interface{} {
	var key = fmt.Sprintf("xgct-video-source::%s_%s_%s", x.Name(), pid, vid)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*2), func() interface{} {
		return x._source(pid, vid)
	})
}

func (x XgCartoonHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x XgCartoonHandler) _tagUrl(tagName string, page, limit int) string {
	var params = strings.Split(tagName, ",")
	if len(params) < 4 {
		params = []string{"*", "*", "*", "*"}
	}
	return fmt.Sprintf(xgctTagUrl, params[0], params[1], params[3], page, limit)
}

func (x XgCartoonHandler) _videoList(tagName, page string) interface{} {
	var pager = model.Pager{Limit: 36, Page: x.parsePageNumber(page)}

	buff, err := x.httpClient.Get(x._tagUrl(tagName, pager.Page, pager.Limit))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var result = gjson.ParseBytes(buff)
	if !result.Get("items").IsArray() {
		return model.NewError("没有解析到数据")
	}
	if result.Get("next").Exists() {
		pager.Pages = pager.Page + 1
	}
	pager.Total = pager.Pages * pager.Limit

	result.Get("items").ForEach(func(key, value gjson.Result) bool {
		log.Println("【name】", value.Get("name").String())
		pager.List = append(pager.List, model.Video{
			Id:         value.Get("cartoon_id").String(),
			Name:       value.Get("name").String(),
			Thumb:      fmt.Sprintf("https://static-a.xgcartoon.com/cover/%s?w=300&h=256&q=100", value.Get("topic_img").String()),
			Intro:      "",
			Resolution: value.Get("region_name").String(),
			UpdatedAt:  value.Get("vod_time").String(),
		})
		return true
	})

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x XgCartoonHandler) _search(keyword, page string) interface{} {
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

func (x XgCartoonHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(xgctDetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id, Url: fmt.Sprintf(noVipDetailUrl, id)}
	doc.Find(".detail-right__volumes .chapter-box").Each(func(i int, selection *goquery.Selection) {
		video.Links = append(video.Links, model.Link{
			Id:    x.simpleRegEx(selection.AttrOr("href", ""), `&chapter_id=(\S+)`),
			Name:  strings.TrimSpace(selection.AttrOr("title", "")),
			Url:   selection.AttrOr("href", ""),
			Group: "西瓜动漫",
		})
	})
	if len(video.Links) <= 0 {
		return model.NewError("获取播放数据失败")
	}

	video.Thumb = strings.TrimSpace(doc.Find(".detail-sider amp-img[layout=responsive]").AttrOr("src", ""))
	video.Name = strings.TrimSpace(doc.Find(".detail-right__title h1").Text())
	video.Intro = strings.TrimSpace(doc.Find(".detail-right__desc p").Text())

	if len(video.Name) <= 0 {
		return model.NewError("获取数据失败")
	}

	return model.NewSuccess(video)
}

func (x XgCartoonHandler) _source(pid, vid string) interface{} {
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

func (x XgCartoonHandler) UpdateHeader(header map[string]string) error {
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

func (x XgCartoonHandler) HoldCookie() error {
	var resp = x._videoList("movie", "1")
	switch r := resp.(type) {
	case model.Success:
		return nil
	case model.Error:
		return errors.New(r.Msg)
	}
	return errors.New("未知错误")
}
