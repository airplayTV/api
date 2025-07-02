package handler

import (
	"bytes"
	"encoding/hex"
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
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type CzzyHandler struct {
	Handler
}

func (x CzzyHandler) Init(options interface{}) model.IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.UserAgent, useragent)
	x.httpClient.AddHeader(headers.Referer, czzyHost)

	x.option = options.(model.CmsZyOption)

	return x
}

func (x CzzyHandler) Name() string {
	return "厂长资源"
}

func (x CzzyHandler) Option() model.CmsZyOption {
	return x.option
}

func (x CzzyHandler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "movie_bt"})
	tags = append(tags, gin.H{"name": "国产剧", "value": "gcj"})
	tags = append(tags, gin.H{"name": "美剧", "value": "meijutt"})
	tags = append(tags, gin.H{"name": "韩剧", "value": "hanjutv"})
	tags = append(tags, gin.H{"name": "番剧", "value": "fanju"})
	tags = append(tags, gin.H{"name": "最新电影", "value": "zuixindianying"})
	tags = append(tags, gin.H{"name": "豆瓣Top250", "value": "dbtop250"})
	tags = append(tags, gin.H{"name": "高分影视", "value": "gaofenyingshi"})
	return tags
}

func (x CzzyHandler) VideoList(tag, page string) interface{} {
	var key = fmt.Sprintf("czzy-video-list::%s_%s_%s", x.Name(), tag, page)
	return model.WithCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._videoList(tag, page)
	})
}

func (x CzzyHandler) Search(keyword, page string) interface{} {
	var key = fmt.Sprintf("czzy-video-search::%s_%s_%s", x.Name(), keyword, page)
	return model.WithCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._search(keyword, page)
	})
}

func (x CzzyHandler) Detail(id string) interface{} {
	var key = fmt.Sprintf("czzy-video-detail::%s_%s", x.Name(), id)
	return model.WithCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._detail(id)
	})
}

func (x CzzyHandler) Source(pid, vid string) interface{} {
	var key = fmt.Sprintf("czzy-video-source::%s_%s_%s", x.Name(), pid, vid)
	return model.WithCache(key, store.WithExpiration(time.Hour*2), func() interface{} {
		return x._source(pid, vid)
	})
}

func (x CzzyHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x CzzyHandler) _tagUrl(tagName, page string) string {
	var p = x.parsePageNumber(page)
	if p == 1 {
		return fmt.Sprintf("%s/%s", strings.TrimRight(czzyHost, "/"), tagName)
	} else {
		return fmt.Sprintf(czzyTagUrl, tagName, x.parsePageNumber(page))
	}
}

func (x CzzyHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(x._tagUrl(tagName, page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var pager = model.Pager{Limit: 25, Page: x.parsePageNumber(page), List: make([]model.Video, 0)}

	doc.Find(".mi_cont .mi_ne_kd ul li").Each(func(i int, selection *goquery.Selection) {
		name := selection.Find(".dytit a").Text()
		tmpUrl, _ := selection.Find(".dytit a").Attr("href")
		thumb, _ := selection.Find("img.thumb").Attr("data-original")
		tag := selection.Find(".nostag").Text()
		actors := selection.Find(".inzhuy").Text()
		resolution := selection.Find(".hdinfo span").Text()

		pager.List = append(pager.List, model.Video{
			Id:         x.simpleRegEx(tmpUrl, `(\d+).html`),
			Name:       name,
			Thumb:      thumb,
			Url:        tmpUrl,
			Actors:     strings.TrimSpace(actors),
			Tag:        tag,
			Resolution: resolution,
		})
	})

	doc.Find(".pagenavi_txt a").Each(func(i int, selection *goquery.Selection) {
		tmpHref, _ := selection.Attr("href")
		n := x.parsePageNumber(x.simpleRegEx(tmpHref, `/page/(\d+)`))
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
		if n >= pager.Pages {
			pager.Pages = n
		}
	})

	pager.Page, _ = strconv.Atoi(doc.Find(".pagenavi_txt .current").Text())

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x CzzyHandler) _search(keyword, page string) interface{} {
	var pager = model.Pager{Limit: 20, Page: x.parsePageNumber(page)}
	buff, err := x.httpClient.Get(fmt.Sprintf(czzySearchUrl, keyword, pager.Page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("文档解析失败：" + err.Error())
	}
	if bytes.Contains(buff, []byte("challenge-error-text")) {
		return model.NewError("challenge失败")
	}
	doc.Find(".search_list ul li").Each(func(i int, selection *goquery.Selection) {
		name := selection.Find(".dytit a").Text()
		tmpUrl, _ := selection.Find(".dytit a").Attr("href")
		thumb := selection.Find("img").AttrOr("src", "")
		tag := selection.Find(".nostag").Text()
		actors := selection.Find(".inzhuy").Text()
		pager.List = append(pager.List, model.Video{
			Id:     x.simpleRegEx(tmpUrl, `(\d+)`),
			Name:   name,
			Thumb:  thumb,
			Url:    tmpUrl,
			Actors: strings.TrimSpace(actors),
			Tag:    tag,
		})
	})
	doc.Find(".pagenavi_txt a").Each(func(i int, selection *goquery.Selection) {
		var p = x.parsePageNumber(selection.Text())
		if className, ok := selection.Attr("class"); ok && className == "current" {
			pager.Page = p
		}
		if p >= pager.Pages {
			pager.Pages = p
		}
	})
	pager.Total = cast.ToInt(doc.Find(".mi_ne_kd .dy_tit_big span").Eq(0).Text())

	return model.NewSuccess(pager)
}

func (x CzzyHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(czzyDetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id}
	doc.Find(".paly_list_btn a").Each(func(i int, selection *goquery.Selection) {
		tmpHref, _ := selection.Attr("href")
		video.Links = append(video.Links, model.Link{
			Id:    x.simpleRegEx(tmpHref, `/v_play/(\S+).html`),
			Name:  strings.ReplaceAll(selection.Text(), "厂长", ""),
			Url:   tmpHref,
			Group: "资源1",
		})
	})

	video.Thumb, _ = doc.Find(".dyxingq .dyimg img").Attr("src")
	video.Name = doc.Find(".dyxingq .moviedteail_tt h1").Text()
	video.Intro = strings.TrimSpace(doc.Find(".yp_context").Text())
	if len(video.Name) <= 0 {
		return model.NewError("获取数据失败")
	}

	return model.NewSuccess(video)
}

func (x CzzyHandler) _source(pid, vid string) interface{} {
	var source = model.Source{Id: pid, Vid: vid}
	buff, err := x.httpClient.Get(fmt.Sprintf(czzyPlayUrl, pid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	source.Name = doc.Find(".pclist .jujiinfo h3").Text()

	if bytes.Contains(buff, []byte("md5.AES.decrypt")) && bytes.Contains(buff, []byte("decrypted.toString(md5.enc.Utf8")) {
		// 从html加密数据中解析播放地址
		var encryptedLine = x._findEncryptedLine(string(buff))
		if len(encryptedLine) == 0 {
			return model.NewError("获取数据失败：无解析数据")
		}

		tmpSource, err := x._parseVideoSource(pid, encryptedLine)
		if err != nil {
			return model.NewError(err.Error())
		}
		source.Source = tmpSource.Source
		source.Type = tmpSource.Type
	} else if doc.Find(".videoplay iframe").Length() > 0 {
		// 解析另一种iframe嵌套的视频
		iframeUrl, _ := doc.Find(".videoplay iframe").Attr("src")
		log.Println("[iframeUrl]", iframeUrl)
		frameContent, err := x.getIframeContent(iframeUrl)
		if err != nil {
			return model.NewError(err.Error())
		}
		var encryptResultV2 = x.simpleRegEx(frameContent, `var result_v2 = {"data":"(\S+?)"`)
		var findV3Rand = x.simpleRegEx(frameContent, `var rand = "(\S+)";`)
		var findV3Player = x.simpleRegEx(frameContent, `var player = "(\S+)";`)
		var tmpPlayUrl = x.simpleRegEx(frameContent, `const mysvg = '(\S+)';`)
		if len(tmpPlayUrl) > 0 {
			source.Source = tmpPlayUrl
		} else if len(encryptResultV2) > 0 {
			source.Source = x.parseEncryptedResultV2ToUrl(encryptResultV2)
		} else if len(findV3Rand) > 0 && len(findV3Player) > 0 {
			source.Source = x.parseEncryptedResultV3ToUrl(findV3Rand, findV3Player)
		} else {
			return model.NewError("未知解析逻辑1")
		}
	} else {
		return model.NewError("未知解析逻辑")
	}

	if len(source.Source) == 0 {
		return model.NewError("播放地址解析失败")
	}

	// 视频类型问题处理
	source.Type = x.parseVideoType(source.Source)
	source.Url = x.handleM3u8pUrl(source.Source)

	if len(source.Source) == 0 {
		return model.NewError("播放地址处理失败")
	}

	return model.NewSuccess(source)
}

func (x CzzyHandler) _findEncryptedLine(htmlContent string) string {
	var findLine = ""
	tmpList := strings.Split(htmlContent, "\n")
	for _, line := range tmpList {
		if strings.Contains(line, "md5.AES.decrypt") {
			findLine = line
			break
		}
	}
	return findLine
}

func (x CzzyHandler) _parseVideoSource(id, js string) (model.Source, error) {
	var source = model.Source{}
	tmpList := strings.Split(strings.TrimSpace(js), ";")

	var data = ""
	var key = ""
	var iv = ""
	for index, str := range tmpList {
		if index == 0 {
			regex := regexp.MustCompile(`"\S+"`)
			data = strings.Trim(regex.FindString(str), `"`)
			continue
		}
		if index == 1 {
			regex := regexp.MustCompile(`"(\S+)"`)
			matchList := regex.FindStringSubmatch(str)
			if len(matchList) > 0 {
				key = matchList[len(matchList)-1]
			}
			continue
		}
		if index == 2 {
			regex := regexp.MustCompile(`\((\S+)\)`)
			matchList := regex.FindStringSubmatch(str)
			if len(matchList) > 0 {
				iv = matchList[len(matchList)-1]
			}
			continue
		}
	}

	log.Println(fmt.Sprintf("[parsing] key: %s, iv: %s", key, iv))

	if key == "" && data == "" {
		return source, errors.New("解析失败")
	}
	bs, err := util.DecryptByAes([]byte(key), []byte(iv), data)
	if err != nil {
		return source, errors.New("解密失败")
	}
	//log.Println("[解析数据]", string(bs))
	source.Source = x.simpleRegEx(string(bs), `video: {url: "(\S+?)",`)
	source.Type = x.simpleRegEx(string(bs), `,type:"(\S+?)",`)
	if len(source.Source) == 0 {
		return source, errors.New("解析失败")
	}

	return source, nil
}

func (x CzzyHandler) getIframeContent(iframeUrl string) (string, error) {
	x.httpClient.AddHeader("referer", czzyHost)
	x.httpClient.AddHeader("sec-fetch-dest", "iframe")
	x.httpClient.AddHeader("sec-fetch-mode", "navigate")
	buff, err := x.httpClient.Get(iframeUrl)
	if err != nil {
		return "", err
	}
	return string(buff), nil
}

func (x CzzyHandler) parseEncryptedResultV2ToUrl(resultV2 string) string {
	// htoStr
	var chars = strings.Split(resultV2, "")
	slices.Reverse(chars)
	var sb = strings.Builder{}
	var tmpStr = ""
	var buf []byte
	var err error
	for i := 0; i < len(chars); i += 2 {
		tmpStr = chars[i] + chars[i+1]
		buf, err = hex.DecodeString(tmpStr)
		if err != nil {
			log.Println("[decodeHexError]", err.Error())
			break
		}
		sb.Write(buf)
	}
	// decodeStr
	var tmpUrl = sb.String()
	var tmpA = (len(tmpUrl) - 7) / 2
	return fmt.Sprintf("%s%s", tmpUrl[0:tmpA], tmpUrl[tmpA+7:])
}

func (x CzzyHandler) parseEncryptedResultV3ToUrl(rand, player string) string {
	buff, err := util.DecryptByAes([]byte("VFBTzdujpR9FWBhe"), []byte(rand), player)
	if err != nil {
		log.Println("[DecryptAesError]", err.Error())
		return ""
	} else {
		var result = gjson.ParseBytes(buff)
		return result.Get("url").String()
	}
}

func (x CzzyHandler) UpdateHeader(header map[string]string) error {
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

func (x CzzyHandler) HoldCookie() error {
	switch r := x.Search("我的", "1").(type) {
	case model.Success:
		return nil
	case model.Error:
		return errors.New(r.Msg)
	}
	return errors.New("未知错误")
}
