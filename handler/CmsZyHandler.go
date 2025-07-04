package handler

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// 采集源
// https://bgm.tv/m/topic/group/406236
// https://woodchen.ink/archives/1207
// https://www.x-lsp.com/category/资源采集
// https://github.com/hd9211/Tvbox1/blob/5634bf904d19c102dc98481741ba578528ea7aa0/zy.json#L92

type CmsZyHandler struct {
	Handler
}

func (x CmsZyHandler) Init(options interface{}) model.IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.UserAgent, useragent)

	x.option = options.(model.CmsZyOption)
	return x
}

func (x CmsZyHandler) Name() string {
	return x.option.GetName()
}

func (x CmsZyHandler) Option() model.CmsZyOption {
	return x.option
}

func (x CmsZyHandler) VideoList(tag, page string) interface{} {
	var key = fmt.Sprintf("cms-video-list::%s_%s_%s", x.Name(), tag, page)
	return model.WithCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._videoList(tag, page)
	})
}

func (x CmsZyHandler) Search(keyword, page string) interface{} {
	switch x.Name() {
	case "红牛资源":
		var tmpCache = model.GetSetCache("cms-hongniu-search-sleep-5s", store.WithExpiration(time.Second*5))
		if tmpCache {
			return model.NewError("该源限制5s内连续搜索")
		}
	}

	var key = fmt.Sprintf("cms-video-search::%s_%s_%s", x.Name(), keyword, page)
	return model.WithCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._search(keyword, page)
	})
}

func (x CmsZyHandler) Detail(id string) interface{} {
	var key = fmt.Sprintf("cms-video-detail::%s_%s", x.Name(), id)
	return model.WithCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._detail(id)
	})
}

func (x CmsZyHandler) Source(pid, vid string) interface{} {
	var key = fmt.Sprintf("cms-video-source::%s_%s_%s", x.Name(), pid, vid)
	return model.WithCache(key, store.WithExpiration(time.Hour*2), func() interface{} {
		return x._source(pid, vid)
	})
}

func (x CmsZyHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x CmsZyHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf("%s?ac=list&pg=%d&t=%s", x.getApiUrl(), x.parsePageNumber(page), tagName))
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
			Intro:      strings.TrimSpace(value.Get("vod_blurb").String()),
			Resolution: value.Get("vod_remarks").String(),
			UpdatedAt:  value.Get("vod_time").String(),
		})
		return true
	})

	pager.List = x.handleVideoListThumb(fmt.Sprintf("%s?ac=detail&ids=%%s", x.getApiUrl()), pager.List)

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x CmsZyHandler) _search(keyword, page string) interface{} {
	switch x.Name() {
	case "红牛资源":
		return x.hongniuSearch(keyword, page)
	default:
		return x.apiSearch(keyword, page)
	}
}

func (x CmsZyHandler) hongniuSearch(keyword, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf("%s/index.php/vod/search/page/%d/wd/%s.html?ac=detail", util.ParseUrlHost(x.option.Host), x.parsePageNumber(page), keyword))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("解析数据失败：" + err.Error())
	}

	var pager = model.Pager{Limit: 51, Page: x.parsePageNumber(page), List: make([]model.Video, 0)}

	var matches = x.simpleRegExList(doc.Find(".pages .page_tip").Text(), `共(\d+)条数据,当前(\d+)/(\d+)页`)
	if len(matches) > 3 {
		pager.Total = cast.ToInt(matches[1])
		pager.Pages = cast.ToInt(matches[3])
		pager.Page = cast.ToInt(matches[2])
	}

	doc.Find(".xing_vb ul li").Each(func(i int, selection *goquery.Selection) {
		var tmpId = x.simpleRegEx(selection.Find("a").AttrOr("href", ""), `id/(\S+).html`)
		if len(tmpId) <= 0 {
			return
		}
		pager.List = append(pager.List, model.Video{
			Id:         tmpId,
			Name:       selection.Find("a").Text(),
			Thumb:      "",
			Intro:      "",
			Url:        "",
			Actors:     "",
			Tag:        selection.Find(".xing_vb5").Text(),
			Resolution: "",
			UpdatedAt:  selection.Find(".xing_vb7").Text(),
			Links:      nil,
		})
	})
	log.Println("[A]", len(pager.List))

	return model.NewSuccess(pager)
}

func (x CmsZyHandler) apiSearch(keyword, page string) interface{} {
	var reqUrl = fmt.Sprintf("%s/?ac=list&pg=%d&t=&wd=%s", x.getApiUrl(), x.parsePageNumber(page), url.QueryEscape(keyword))
	buff, err := x.httpClient.Get(reqUrl)
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
			Intro:      strings.TrimSpace(value.Get("vod_blurb").String()),
			Resolution: value.Get("vod_remarks").String(),
			UpdatedAt:  value.Get("vod_time").String(),
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
	buff, err := x.httpClient.Get(fmt.Sprintf("%s?ac=detail&ids=%s", x.getApiUrl(), id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var video = model.Video{Id: id, Links: make([]model.Link, 0)}
	var result = gjson.ParseBytes(buff)
	if result.Get("total").Int() > 0 {
		result.Get("list").ForEach(func(key, value gjson.Result) bool {
			video.Name = value.Get("vod_name").String()
			video.Thumb = value.Get("vod_pic").String()
			video.Intro = util.HtmlToText(value.Get("vod_content").String())
			video.Actors = value.Get("vod_actor").String()
			video.UpdatedAt = value.Get("vod_time").String()
			video.Links, _ = x.parseSourceList(
				x.option.GetName(),
				value.Get("vod_play_from").String(),
				value.Get("vod_play_note").String(),
				value.Get("vod_play_url").String(),
				"",
			)
			return true
		})
	}

	return model.NewSuccess(video)
}

func (x CmsZyHandler) _source(pid, vid string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf("%s?ac=detail&ids=%s", x.getApiUrl(), vid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var source = model.Source{Id: pid, Vid: vid}
	var result = gjson.ParseBytes(buff)

	if result.Get("total").Int() > 0 {
		result.Get("list").ForEach(func(key, value gjson.Result) bool {
			source.Name = value.Get("vod_name").String()
			source.Thumb = value.Get("vod_pic").String()
			_, tmpLink := x.parseSourceList(
				x.option.GetName(),
				value.Get("vod_play_from").String(),
				value.Get("vod_play_note").String(),
				value.Get("vod_play_url").String(),
				pid,
			)
			source.Url = tmpLink.Url
			source.Source = tmpLink.Url

			return true
		})
	}
	source.Type = x.parseVideoType(source.Source)
	if len(source.Url) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(source)
}

func (x CmsZyHandler) parseSourceSplit(vodPlayNote string) string {
	if len(vodPlayNote) == 0 {
		return "#"
	}
	return vodPlayNote
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
	var tagCacheFile = filepath.Join(util.AppPath(), fmt.Sprintf("cache/tags/%s.json", x.option.GetId()))
	var buff = util.ReadFile(tagCacheFile)
	var tmpTags = map[string]string{"全部": ""}
	var result = gjson.ParseBytes(buff)
	result.Get("class").ForEach(func(key, value gjson.Result) bool {
		tmpTags[value.Get("type_name").String()] = value.Get("type_id").String()
		return true
	})
	go x.saveTagListLocal(tagCacheFile)

	return x.formatTags(tmpTags)
}

func (x CmsZyHandler) saveTagListLocal(filename string) {
	stat, err := os.Stat(filepath.Dir(filename))
	if err != nil {
		if err = os.MkdirAll(filepath.Dir(filename), 0644); err != nil {
			log.Println("[目录创建失败]", err.Error())
			return
		}
	}
	stat, err = os.Stat(filename)
	if err == nil {
		if time.Now().Unix()-stat.ModTime().Unix() <= 86400*2 {
			return
		}
	}
	_ = os.Remove(filename)

	log.Println("[req]", x.option.GetName(), x.option.GetApi())
	buff, err := x.httpClient.Get(x.option.GetApi())
	if err != nil {
		_ = util.WriteFile(fmt.Sprintf("%s.error", filename), []byte(err.Error()))
		return
	}
	var result = gjson.ParseBytes(buff)
	if len(result.Get("class").Array()) > 0 {
		_ = util.WriteFile(filename, buff)
	} else {
		_ = util.WriteFile(fmt.Sprintf("%s.error", filename), buff)
	}
}

func (x CmsZyHandler) formatTags(tags map[string]string) []model.KV1 {
	var result = make([]model.KV1, 0)
	for k, v := range tags {
		result = append(result, model.KV1{
			Name:  k,
			Value: v,
		})
	}
	slices.SortFunc(result, func(a, b model.KV1) int {
		return strings.Compare(a.Value, b.Value)
	})
	return result
}

func (x CmsZyHandler) parseSourceList(sourceName, vodPlayFrom, vodPlayNote, vodPlayUrl, pid string) ([]model.Link, model.Link) {
	var sourceNameList = []string{sourceName}
	var sourceList = []string{vodPlayUrl}
	if len(vodPlayNote) > 0 {
		sourceNameList = strings.Split(vodPlayFrom, vodPlayNote)
		sourceList = strings.Split(vodPlayUrl, vodPlayNote)
	}

	var links = make([]model.Link, 0)
	var link model.Link

	for i, tmpString := range sourceList {
		tmpString = strings.Trim(tmpString, "#")
		for j, tmpSource := range strings.Split(tmpString, "#") {
			var tmpList = strings.Split(tmpSource, "$")
			var tmpPid = fmt.Sprintf("%d-%d", i, j)
			if len(tmpList) == 1 {
				tmpList = []string{"HD", tmpList[0]}
			}
			links = append(links, model.Link{
				Id:    tmpPid,
				Name:  tmpList[0],
				Url:   tmpList[1],
				Group: sourceNameList[i],
			})
			if tmpPid == pid {
				link = model.Link{
					Id:    tmpPid,
					Name:  tmpList[0],
					Url:   tmpList[1],
					Group: sourceNameList[i],
				}
			}
		}
	}

	return links, link
}

func (x CmsZyHandler) UpdateHeader(header map[string]string) error {
	return nil
}

func (x CmsZyHandler) HoldCookie() error {
	return nil
}
