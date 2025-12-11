package handler

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	headers2 "github.com/go-http-utils/headers"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"log"
	"net/url"
	"slices"
	"strings"
)

type CATVHandler struct {
	Handler
}

func (x CATVHandler) Init(options interface{}) model.IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.Referer, subbHost)

	x.option = options.(model.CmsZyOption)
	return x
}

func (x CATVHandler) Name() string {
	return "CATV(解析)"
}

func (x CATVHandler) Option() model.CmsZyOption {
	return x.option
}

func (x CATVHandler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "movie"})
	tags = append(tags, gin.H{"name": "剧集", "value": "tv"})
	tags = append(tags, gin.H{"name": "动漫", "value": "dongman"})
	tags = append(tags, gin.H{"name": "综艺", "value": "zongyi"})
	//tags = append(tags, gin.H{"name": "小视频", "value": "music"})
	//tags = append(tags, gin.H{"name": "直播", "value": "zb"})
	return tags
}

func (x CATVHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x CATVHandler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x CATVHandler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x CATVHandler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x CATVHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x CATVHandler) _videoList(tagName, page string) interface{} {
	var p = x.parsePageNumber(page)
	buff, err := x.httpClient.Get(fmt.Sprintf(catvTagUrl, tagName, p, p))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var pager = model.Pager{Limit: 35, Page: p, List: make([]model.Video, 0)}

	doc.Find(".myui-panel-box .myui-vodlist li").Each(func(i int, selection *goquery.Selection) {
		var name = selection.Find(".title a").Text()
		var tmpUrl = selection.Find(".title a").AttrOr("href", "")
		var thumb = selection.Find(".myui-vodlist__thumb").AttrOr("data-original", "")
		var tag = selection.Find(".pic-tag").Text()
		var actors = selection.Find(".text-muted").Text()
		var resolution = selection.Find(".undefined").Text()
		pager.List = append(pager.List, model.Video{
			Id:         util.EncodeComponentUrl(tmpUrl),
			Name:       name,
			Thumb:      thumb,
			Url:        tmpUrl,
			Actors:     strings.TrimSpace(actors),
			Tag:        tag,
			Resolution: resolution,
		})
	})

	doc.Find(".myui-page a").Each(func(i int, selection *goquery.Selection) {
		var tmpHref = selection.AttrOr("href", "")
		var n = cast.ToInt(x.simpleRegEx(tmpHref, `_(\d+).html`))
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
	})

	pager.Page = cast.ToInt(doc.Find(".myui-page .on").Text())

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x CATVHandler) _search(keyword, page string) interface{} {
	var pager = model.Pager{Limit: 50, Page: x.parsePageNumber(page)}
	buff, err := x.httpClient.Get(fmt.Sprintf(catvSearchUrl, url.QueryEscape(keyword)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		log.Println("[文档解析失败]", err.Error())
		return pager
	}
	doc.Find("#searchList li").Each(func(i int, selection *goquery.Selection) {
		var name = selection.Find(".title a").Text()
		var tmpUrl = selection.Find(".title a").AttrOr("href", "")
		var thumb = selection.Find(".lazyload").AttrOr("data-original", "")
		var tag = selection.Find(".nostag").Text()
		var actors = selection.Find(".inzhuy").Text()
		pager.List = append(pager.List, model.Video{
			Id:     util.EncodeComponentUrl(tmpUrl),
			Name:   name,
			Thumb:  thumb,
			Url:    tmpUrl,
			Actors: strings.TrimSpace(actors),
			Tag:    tag,
		})
	})

	if len(pager.List) <= 0 {
		return model.NewError("暂无数据")
	}

	pager.Total = len(pager.List)
	pager.Pages = 1
	pager.Page = 1

	return model.NewSuccess(pager)
}

func (x CATVHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(catvDetailUrl, util.DecodeComponentUrl(id)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id}
	doc.Find("#xluu ul a").Each(func(i int, selection *goquery.Selection) {
		var tmpGroup = selection.Text()
		var lineId = selection.AttrOr("id", "")
		var jxUrl = selection.AttrOr("data-url", "")
		// xl0 线路暂未解密
		// xl4 解析地址：https://jx.nnxv.cn/tv.php?url=http://www.mgtv.com/b/297817/4050151.html?cxid=90f0zbamf
		if !slices.Contains([]string{ /** "xl0", **/ "xl4"}, lineId) {
			return
		}
		doc.Find("#playlist li").Each(func(j int, selection2 *goquery.Selection) {
			var sourceUrl = selection2.Find("a").AttrOr("href", "")
			video.Links = append(video.Links, model.Link{
				Id:    util.EncodeComponentUrl(fmt.Sprintf("%s%s", jxUrl, sourceUrl)),
				Name:  selection2.Find("a").Text(),
				Url:   selection2.Find("a").AttrOr("href", ""),
				Group: tmpGroup,
			})
		})
	})

	video.Name = doc.Find(".xzname").AttrOr("data-name", "")
	video.Thumb = doc.Find(".xzname").AttrOr("src", "")
	video.Intro = strings.TrimSpace(doc.Find(".myui-panel-box .sketch").Text())

	return model.NewSuccess(video)
}

func (x CATVHandler) _source(pid, vid string) interface{} {
	var source = model.Source{Id: pid, Vid: vid}
	var jxUrl = util.DecodeComponentUrl(pid)
	jxUrl = strings.ReplaceAll(jxUrl, "https://qt6.cn/?", "https://jx.xymp4.cc/?")
	jxUrl = strings.ReplaceAll(jxUrl, "https://jx.mmkv.cn/", "https://jx.nnxv.cn/")

	u, _ := url.Parse(jxUrl)
	resp, err := x.handleQiGeJieXi(u.Query().Get("url"))
	if err != nil {
		log.Println("[handleQiGeJieXiError]", err.Error())
		return nil
	}

	source.Source = resp.Get("url").String()
	// 视频类型问题处理
	source.Type = x.parseVideoType(source.Source)
	source.Url = source.Source

	if len(source.Source) == 0 {
		return model.NewError("播放地址处理失败")
	}

	return model.NewSuccess(source)
}

func (x CATVHandler) handleQiGeJieXi(sourceUrl string) (resp gjson.Result, err error) {
	var httpClient = x.httpClient.Clone()
	httpClient.AddHeader(headers2.ContentType, "application/x-www-form-urlencoded")
	httpClient.AddHeader(headers2.Origin, "https://jx.nnxv.cn")
	httpClient.AddHeader(headers2.Referer, fmt.Sprintf("https://jx.nnxv.cn/tv.html?url=%s", sourceUrl))
	httpClient.AddHeader(headers2.UserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 Edg/143.0.0.0")

	var form = url.Values{}
	form.Add("url", sourceUrl)
	form.Add("api_key", "qigejiexi")

	_, body, err := httpClient.PostResponse("https://jx.nnxv.cn/qigejiexiii.js", form.Encode())
	if err != nil {
		return
	}

	if bytes.Contains(body, []byte("502 Bad Gateway")) {
		err = errors.New("仅限国内ip请求解析/502 Bad Gateway")
		return
	}

	decrypt, err := util.AesDecrypt_QiGeJieXi(string(body), []byte("QRSTUVWXYZabcdefghijklmnopqrsutv"))
	if err != nil {
		return
	}

	return gjson.ParseBytes(decrypt), nil
}

func (x CATVHandler) UpdateHeader(header map[string]string) error {
	return nil
}

func (x CATVHandler) HoldCookie() error {
	return nil
}
