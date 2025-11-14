package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/lixiang4u/goWebsocket"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
)

type NoVipHandler struct {
	Handler
}

func (x NoVipHandler) Init(options interface{}) model.IVideo {
	x.httpClient = util.HttpClient{
		//ProxyUrl: "http://127.0.0.1:1080",
	}
	x.httpClient.AddHeader(headers.UserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
	x.httpClient.AddHeader("referer", noVipHost)
	x.httpClient.AddHeader("upgrade-insecure-requests", "1") // 搜索会检测

	x.option = options.(model.CmsZyOption)

	return x
}

func (x NoVipHandler) Name() string {
	return "NO视频"
}

func (x NoVipHandler) Option() model.CmsZyOption {
	return x.option
}

func (x NoVipHandler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "movie"})
	tags = append(tags, gin.H{"name": "剧集", "value": "tv"})
	tags = append(tags, gin.H{"name": "综艺", "value": "shows"})
	tags = append(tags, gin.H{"name": "音乐", "value": "music"})
	tags = append(tags, gin.H{"name": "短片", "value": "short"})
	return tags
}

func (x NoVipHandler) VideoList(tag, page string) interface{} {
	var key = fmt.Sprintf("novip-video-list::%s_%s_%s", x.Name(), tag, page)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._videoList(tag, page)
	})
}

func (x NoVipHandler) Search(keyword, page string) interface{} {
	var key = fmt.Sprintf("novip-video-search::%s_%s_%s", x.Name(), keyword, page)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._search(keyword, page)
	})
}

func (x NoVipHandler) Detail(id string) interface{} {
	var key = fmt.Sprintf("novip-video-detail::%s_%s", x.Name(), id)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*6), func() interface{} {
		return x._detail(id)
	})
}

func (x NoVipHandler) Source(pid, vid string) interface{} {
	var key = fmt.Sprintf("novip-video-source::%s_%s_%s", x.Name(), pid, vid)
	return model.WithSuccessCache(key, store.WithExpiration(time.Hour*2), func() interface{} {
		return x._source(pid, vid)
	})
}

func (x NoVipHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x NoVipHandler) _tagUrl(tagName, page string) string {
	if len(tagName) <= 0 {
		tagName = "movie"
	}
	var p = x.parsePageNumber(page)
	if p == 1 {
		return fmt.Sprintf(noVipTagUrl, tagName)
	} else {
		return fmt.Sprintf("%s/page/%s/", strings.TrimRight(fmt.Sprintf(noVipTagUrl, tagName), "/"), page)
	}
}

func (x NoVipHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(x._tagUrl(tagName, page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var pager = model.Pager{Limit: 16, Page: x.parsePageNumber(page), List: make([]model.Video, 0)}

	doc.Find(".video-listing-content .video-item").Each(func(i int, selection *goquery.Selection) {
		var tmpUrl = selection.Find(".item-thumbnail a").AttrOr("href", "")
		pager.List = append(pager.List, model.Video{
			Id:    x.simpleRegEx(tmpUrl, `(\d+).html`),
			Name:  selection.Find(".item-head").Text(),
			Thumb: selection.Find(".item-thumbnail img").AttrOr("data-original", ""),
			Url:   tmpUrl,
		})
	})

	doc.Find(".wp-pagenavi a").Each(func(i int, selection *goquery.Selection) {
		tmpHref, _ := selection.Attr("href")
		n := x.parsePageNumber(x.simpleRegEx(tmpHref, `/page/(\d+)`))
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
		if n >= pager.Pages {
			pager.Pages = n
		}
	})
	var tmpLastPage = cast.ToInt(doc.Find(".wp-pagenavi .last").Text())
	if tmpLastPage > pager.Pages {
		pager.Pages = tmpLastPage
	}

	pager.Page, _ = strconv.Atoi(doc.Find(".wp-pagenavi .current").Text())

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x NoVipHandler) _search(keyword, page string) interface{} {
	var pager = model.Pager{Limit: 16, Page: x.parsePageNumber(page)}
	buff, err := x.httpClient.Get(fmt.Sprintf(noVipSearchUrl, pager.Page, url.QueryEscape(keyword)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("文档解析失败：" + err.Error())
	}

	doc.Find(".search-listing-content .video-item").Each(func(i int, selection *goquery.Selection) {
		var tmpUrl = selection.Find(".item-thumbnail a").AttrOr("href", "")
		pager.List = append(pager.List, model.Video{
			Id:    x.simpleRegEx(tmpUrl, `(\d+).html`),
			Name:  selection.Find(".item-head").Text(),
			Thumb: selection.Find(".item-thumbnail img").AttrOr("data-original", ""),
			Url:   tmpUrl,
		})
	})

	doc.Find(".wp-pagenavi a").Each(func(i int, selection *goquery.Selection) {
		tmpHref, _ := selection.Attr("href")
		n := x.parsePageNumber(x.simpleRegEx(tmpHref, `/page/(\d+)`))
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
		if n >= pager.Pages {
			pager.Pages = n
		}
	})
	var tmpLastPage = cast.ToInt(doc.Find(".wp-pagenavi .last").Text())
	if tmpLastPage > pager.Pages {
		pager.Pages = tmpLastPage
	}

	pager.Page, _ = strconv.Atoi(doc.Find(".wp-pagenavi .current").Text())

	return model.NewSuccess(pager)
}

func (x NoVipHandler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(noVipDetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id, Url: fmt.Sprintf(noVipDetailUrl, id)}
	doc.Find(".multilink-table-wrap a").Each(func(i int, selection *goquery.Selection) {
		tmpHref, _ := selection.Attr("href")
		video.Links = append(video.Links, model.Link{
			Id:    strings.TrimSpace(selection.AttrOr("data-vid", "")),
			Name:  strings.TrimSpace(selection.Text()),
			Url:   tmpHref,
			Group: "资源1",
		})
	})
	if len(video.Links) <= 0 {
		playJson, _ := x.parsePlayInfo(string(buff))
		video.Links = append(video.Links, model.Link{
			Id:    playJson.Get("vid").String(),
			Name:  "HD",
			Url:   "",
			Group: "资源1",
		})

	}

	video.Thumb = x.simpleRegEx(string(buff), `<meta property="og:image" content="(\S+)">`)
	video.Name = doc.Find(".single-video-view .entry-title").Text()

	{
		tmpText, _ := doc.Find(".single-video-view .item-content p").Eq(0).Html()
		var pattern = `(\<style\>\S+\<\/style\>)` // 匹配一个或多个数字
		tmpText = regexp.MustCompile(pattern).ReplaceAllString(tmpText, "")
		pattern = `(\<script\>\S+\<\/script\>)` // 匹配一个或多个数字
		tmpText = regexp.MustCompile(pattern).ReplaceAllString(tmpText, "")
		video.Intro = tmpText
	}

	if len(video.Name) <= 0 {
		return model.NewError("获取数据失败")
	}

	return model.NewSuccess(video)
}

func (x NoVipHandler) parsePlayInfo(htmlContent string) (resp gjson.Result, err error) {
	var playJsonText = x.simpleRegEx(htmlContent, `window\.playInfo=(\S+?);`)
	if len(playJsonText) <= 0 {
		err = errors.New("没有播放信息")
		return
	}
	playJsonText = strings.ReplaceAll(playJsonText, `vid:`, `"vid":`)
	playJsonText = strings.ReplaceAll(playJsonText, `pkey:`, `"pkey":`)
	resp = gjson.Parse(playJsonText)
	if !resp.Get("pkey").Exists() {
		//return model.NewError("解析失败：没有pkey2")
		err = errors.New("没有pkey")
		return
	}
	return resp, nil
}

func (x NoVipHandler) _source(pid, vid string) interface{} {
	var source = model.Source{Id: pid, Vid: vid}

	buff, err := x.httpClient.Get(fmt.Sprintf(noVipPlayUrl, vid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var ref = strings.ReplaceAll(doc.Find("#cancel-comment-reply-link").AttrOr("href", ""), "#respond", "")

	source.Name = doc.Find(".entry-title").Text()

	playJson, err := x.parsePlayInfo(string(buff))
	if err != nil {
		return model.NewError("解析失败：" + err.Error())
	}

	buff, err = x.httpClient.Get(fmt.Sprintf("https://player.novipnoad.net/v1/?url=%s&pkey=%s&ref=%s", pid, playJson.Get("pkey").String(), ref))
	if err != nil {
		return model.NewError("解析失败：" + err.Error())
	}

	var device = x.simpleRegEx(string(buff), `params\['device'\] = '(\S+?)';`)
	if len(device) <= 0 {
		return model.NewError("解析失败：没有device")
	}

	var vKeyHandler = strings.TrimSpace(x.simpleRegEx(string(buff), `function __\(\) \{([\S\s]+?)\}[\r\n]`))
	if len(vKeyHandler) <= 0 {
		return model.NewError("解析失败：没有vkey")
	}

	var matchedVKeyValues = x.simpleRegExList(x.fuckVKey(vKeyHandler), `{"ckey":"(\S+)","ref":"(\S+)","ip":"(\S+)","time":"(\S+)"}`)
	if len(matchedVKeyValues) < 5 {
		log.Println("[matchedVKeyValues]", goWebsocket.ToJson(matchedVKeyValues))
		return model.NewError("解析失败：vkey异常")
	}

	buff, err = x.httpClient.Get(fmt.Sprintf("https://player.novipnoad.net/v1/player.php?id=%s&device=%s", pid, device))
	if err != nil {
		return model.NewError("解析失败：" + err.Error())
	}

	// const jsapi = 'https://enc-vod.oss-internal.novipnoad.net/ftn/1762780536.js';
	var tmpJsApi = strings.TrimSpace(x.simpleRegEx(string(buff), `jsapi = '(\S+)';`))
	if len(tmpJsApi) <= 0 {
		return model.NewError("解析失败：没有jsapi")
	}

	buff, err = x.httpClient.Get(fmt.Sprintf(
		"%s?ckey=%s&ref=%s&ip=%s&time=%s",
		tmpJsApi,
		strings.ToUpper(matchedVKeyValues[1]),
		url.QueryEscape(matchedVKeyValues[2]),
		matchedVKeyValues[3],
		matchedVKeyValues[4],
	))
	if err != nil {
		return model.NewError("解析失败：" + err.Error())
	}

	var encryptedRC4 = x.simpleRegEx(string(buff), `videoUrl=JSON.decrypt\("(\S+?)"\);`)
	if len(encryptedRC4) <= 0 {
		return model.NewError("解析失败：没有加密数据")
	}
	encryptedRC4 = x.decryptNoVipVideoUrl(encryptedRC4)
	//log.Println("[decryptNoVipVideoUrl]", encryptedRC4)
	var decryptJson = gjson.Parse(encryptedRC4)
	if !decryptJson.Get("quality").IsArray() {
		return model.NewError("解析失败：" + encryptedRC4)
	}

	source.Source = decryptJson.Get("quality").Array()[0].Get("url").String()
	source.Url = source.Source
	// 视频类型问题处理
	source.Type = x.parseVideoType(source.Source)

	if len(source.Source) == 0 {
		return model.NewError("播放地址处理失败")
	}

	return model.NewSuccess(source)
}

func (x NoVipHandler) fuckVKey(vKeyHandler string) string {
	vKeyHandler = strings.Replace(vKeyHandler, "eval(function", "return (function", 1)
	vKeyHandler = fmt.Sprintf(`function __() { %s; }`, vKeyHandler)

	//log.Println("[vKeyHandler.E]", vKeyHandler)

	vm := goja.New()
	_, err := vm.RunString(vKeyHandler)
	if err != nil {
		log.Println("[LoadGojaError]", err.Error())
		return ""
	}

	var decode func() string
	err = vm.ExportTo(vm.Get("__"), &decode)
	if err != nil {
		log.Println("[ExportGojaFnError]", err.Error())
		return ""
	}

	return decode()
}

func (x NoVipHandler) decryptNoVipVideoUrl(encodedText string) string {
	// 1. Base64解码输入数据
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedText)
	if err != nil {
		return ""
	}
	// 2. RC4密钥调度算法 - 初始化置换表
	var sbox = [256]byte{
		99, 207, 215, 98, 55, 168, 56, 85, 13, 160, 50, 134, 15, 147, 28, 197, 19, 123, 12, 174, 33, 145, 142, 60, 183,
		29, 136, 218, 144, 124, 209, 46, 81, 155, 3, 121, 23, 34, 10, 73, 234, 65, 80, 248, 20, 163, 38, 118, 201, 230,
		68, 240, 156, 205, 151, 14, 105, 61, 0, 138, 122, 216, 59, 112, 176, 200, 67, 188, 250, 120, 44, 178, 53, 165, 94,
		170, 25, 93, 253, 154, 157, 117, 107, 11, 125, 40, 148, 126, 251, 71, 9, 17, 177, 181, 179, 135, 133, 39, 62, 64,
		210, 6, 225, 36, 45, 49, 79, 8, 95, 116, 32, 175, 54, 162, 192, 100, 227, 141, 229, 115, 51, 129, 128, 101, 194,
		96, 152, 198, 91, 226, 220, 254, 103, 213, 191, 223, 52, 202, 150, 231, 211, 221, 239, 219, 206, 190, 127, 166,
		131, 143, 173, 238, 158, 72, 119, 87, 189, 195, 47, 235, 102, 146, 245, 186, 16, 104, 42, 222, 233, 199, 66, 212,
		180, 77, 97, 24, 241, 228, 110, 21, 108, 247, 35, 30, 75, 4, 246, 130, 78, 86, 184, 84, 89, 92, 153, 76, 113, 161,
		58, 139, 114, 244, 159, 2, 203, 37, 252, 27, 149, 1, 26, 255, 140, 57, 5, 171, 232, 41, 237, 187, 69, 214, 185,
		111, 22, 48, 243, 249, 196, 88, 137, 82, 43, 164, 74, 172, 182, 90, 70, 169, 204, 208, 31, 193, 217, 224, 106, 109,
		7, 83, 236, 18, 132, 242, 63, 167,
	}
	// 3. RC4伪随机数生成
	var i = 0
	var j = 0
	var result = make([]byte, len(decodedBytes))
	for idx := 0; idx < len(decodedBytes); idx++ {
		// 更新状态变量
		i = (i + 1) % 256
		j = (j + int(sbox[i])) % 256
		// 交换sbox中的两个值
		sbox[i], sbox[j] = sbox[j], sbox[i]
		// 生成密钥流字节
		keyByte := sbox[(int(sbox[i])+int(sbox[j]))%256]
		// 异或解密
		result[idx] = decodedBytes[idx] ^ keyByte
	}
	return string(result)
}

func (x NoVipHandler) UpdateHeader(header map[string]string) error {
	if header == nil {
		return errors.New("header数据不能为空")
	}
	for key, value := range header {
		x.httpClient.AddHeader(key, value)
	}

	// 请求数据并检测Cookie是否可用
	var resp = x.VideoList("movie", "1")
	switch resp.(type) {
	case model.Success:
		// 如果可用则设置到当前上下文的http请求头
		_ = util.SaveHttpHeader(x.Name(), header)
		return nil
	default:
		log.Println("[ERR]", goWebsocket.ToJson(resp))
		return errors.New("cookie无效")
	}
}

func (x NoVipHandler) HoldCookie() error {
	switch r := x.Search("我的", "1").(type) {
	case model.Success:
		return nil
	case model.Error:
		return errors.New(r.Msg)
	}
	return errors.New("未知错误")
}
