package handler

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/go-http-utils/headers"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/airplayTV/api/util"
	"github.com/lixiang4u/goWebsocket"

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
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "1"})
	tags = append(tags, gin.H{"name": "电视剧", "value": "2"})
	tags = append(tags, gin.H{"name": "动漫", "value": "4"})
	tags = append(tags, gin.H{"name": "综艺", "value": "3"})
	return tags
}

func (x JinpaiHandler) VideoList(tag, page string) interface{} {
	var key = fmt.Sprintf("jinpai-video-list::%s_%s_%s", x.Name(), tag, page)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._videoList(tag, page)
	})
}

func (x JinpaiHandler) Search(keyword, page string) interface{} {
	var key = fmt.Sprintf("jinpai-video-search::%s_%s_%s", x.Name(), keyword, page)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._search(keyword, page)
	})
}

func (x JinpaiHandler) Detail(id string) interface{} {
	var key = fmt.Sprintf("jinpai-video-detail::%s_%s", x.Name(), id)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._detail(id)
	})
}

func (x JinpaiHandler) Source(pid, vid string) interface{} {
	var key = fmt.Sprintf("jinpai-video-source::%s_%s_%s", x.Name(), pid, vid)
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

	buff, err := x.jinpaiSignHttpClientGet(fmt.Sprintf(jinpaiTagUrl, tagName, pager.Page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	result, err := x.parseVideoListJson(buff)
	if err != nil {
		return model.NewError(err.Error())
	}

	result.Get("list").ForEach(func(key, value gjson.Result) bool {
		pager.List = append(pager.List, model.Video{
			Id:         value.Get("vodId").String(),
			Name:       value.Get("vodName").String(),
			Thumb:      value.Get("vodPic").String(),
			Resolution: value.Get("vodVersion").String(),
			Tag:        value.Get("vodYear").String(),
			UpdatedAt:  value.Get("vodPubdate").String(),
		})
		return true
	})

	pager.Pages = cast.ToInt(result.Get("totalPage").String())
	pager.Total = cast.ToInt(result.Get("totalCount").String())

	if len(pager.List) <= 0 {
		return model.NewError("没有解析到数据")
	}

	return model.NewSuccess(pager)
}

func (x JinpaiHandler) _search(keyword, page string) interface{} {
	var pager = model.Pager{Limit: 12, Pages: 1, Page: x.parsePageNumber(page)}
	buff, err := x.jinpaiSignHttpClientGet(fmt.Sprintf(jinpaiSearchUrl, url.QueryEscape(keyword), pager.Page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var result = gjson.ParseBytes(buff)
	result = result.Get("data").Get("result")

	result.Get("list").ForEach(func(key, value gjson.Result) bool {
		pager.List = append(pager.List, model.Video{
			Id:         value.Get("vodId").String(),
			Name:       value.Get("vodName").String(),
			Thumb:      value.Get("vodPic").String(),
			Resolution: value.Get("vodVersion").String(),
			Tag:        value.Get("vodYear").String(),
			UpdatedAt:  value.Get("vodPubdate").String(),
		})
		return true
	})

	pager.Pages = cast.ToInt(result.Get("totalPage").String())
	pager.Total = cast.ToInt(result.Get("totalCount").String())

	if len(pager.List) <= 0 {
		return model.NewError("没有解析到数据")
	}

	return model.NewSuccess(pager)
}

func (x JinpaiHandler) _detail(id string) interface{} {
	buff, err := x.jinpaiSignHttpClientGet(fmt.Sprintf(jinpaiDetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id, Url: fmt.Sprintf(jinpaiDetailUrl, id), Links: make([]model.Link, 0)}
	var result = gjson.ParseBytes(buff)
	video.Name = result.Get("data").Get("vodName").String()
	video.Thumb = result.Get("data").Get("vodPic").String()
	video.Intro = util.HtmlToText(result.Get("data").Get("vodContent").String())
	result.Get("data").Get("episodeList").ForEach(func(key, value gjson.Result) bool {
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

func (x JinpaiHandler) parseVideoListJson(html []byte) (result gjson.Result, err error) {
	var scanner = bufio.NewScanner(bytes.NewReader(html))
	for scanner.Scan() {
		var tmpLine = strings.TrimSpace(scanner.Text())
		if !strings.Contains(tmpLine, "filerList") || !strings.Contains(tmpLine, "videoList") {
			continue
		}
		var tmpRegEx = fmt.Sprintf(`%s([\S\s]+)%s$`, regexp.QuoteMeta(`"videoList":`), regexp.QuoteMeta(`}]`))
		result = gjson.Parse(x.simpleRegEx(tmpLine, tmpRegEx)).Get("data")
	}
	if !result.Exists() {
		err = errors.New("数据解析失败1")
		return
	}
	return
}

func (x JinpaiHandler) jinpaiSignHttpClientGet(requestUrl string) (buff []byte, err error) {
	var ts = cast.ToString(time.Now().UnixMilli())
	var tmpHttp = x.httpClient.Clone()
	tmpHttp.AddHeader("Authorization", "")
	tmpHttp.AddHeader("Deviceid", uuid.New().String())
	tmpHttp.AddHeader("T", ts)
	tmpHttp.AddHeader("Client-Type", "1")
	tmpHttp.AddHeader(headers.Referer, jinpaiHost)

	tmpUrl, err := url.Parse(requestUrl)
	if err != nil {
		return
	}
	var values = tmpUrl.Query()
	var signKey = "cb808529bae6b6be45ecfab29a4889bc"
	if strings.Contains(requestUrl, "video/episode") {
		var tmpQuery = fmt.Sprintf(`clientType=1&id=%s&nid=%s&key=%s&t=%s`, values.Get("id"), values.Get("nid"), signKey, ts)
		tmpHttp.AddHeader("Sign", util.Sha1Simple([]byte(util.StringMd5(tmpQuery))))
	} else if strings.Contains(requestUrl, "video/detail") {
		var tmpQuery = fmt.Sprintf(`id=%s&key=%s&t=%s`, values.Get("id"), signKey, ts)
		tmpHttp.AddHeader("Sign", util.Sha1Simple([]byte(util.StringMd5(tmpQuery))))
	} else if strings.Contains(requestUrl, "video/searchByWord") {
		var tmpQuery = fmt.Sprintf(`keyword=%s&pageNum=%s&pageSize=%s&sourceCode=1&key=%s&t=%s`, values.Get("keyword"), values.Get("pageNum"), values.Get("pageSize"), signKey, ts)
		tmpHttp.AddHeader("Sign", util.Sha1Simple([]byte(util.StringMd5(tmpQuery))))
	} else if strings.Contains(requestUrl, "vod/show/id") {
		tmpHttp.AddHeader("RSC", "1")
	} else {
		err = errors.New("请求配置异常")
		return
	}

	return tmpHttp.Get(requestUrl)
}

func (x JinpaiHandler) _source(pid, vid string) interface{} {
	var source = model.Source{Id: pid, Vid: vid}

	buff, err := x.jinpaiSignHttpClientGet(fmt.Sprintf(jinpaiPlayUrl, vid, pid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var result = gjson.ParseBytes(buff)
	var resolution int64 = 0
	result.Get("data").Get("list").ForEach(func(key, value gjson.Result) bool {
		if value.Get("resolution").Int() >= resolution {
			resolution = value.Get("resolution").Int()
			source.Source = value.Get("url").String()
			source.Name = fmt.Sprintf("%s(%s)", value.Get("resolutionName").String(), value.Get("resolution").String())
		}
		return true
	})

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
