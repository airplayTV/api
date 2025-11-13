package handler

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"slices"
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
	x.httpClient = util.HttpClient{ProxyUrl: "http://127.0.0.1:1080"}
	x.httpClient.AddHeader(headers.UserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36")
	x.httpClient.AddHeader("cookie", "cf_clearance=LNH4OSaz5flx9ZSaDo_0Do.pVBd9spmMBh6cBGrERnk-1760846173-1.2.1.1-tSRT.BqPoD2_ugXBJr_LB0OIOrXUoCr3kNEs_E1i0helAhYmek7e2FgKQGRvEZmEEsKel9SmES5uoGD2VGnsNvWwifhaq8IkiZp2hfuRrFdb3m1Rpa7IuHUl21Ed4OaKsz31nj.Gbqz0OaMuXFIQKsLCinqpmvv0XUgD25vtCvN6dLKhHTxMYu_PvqgEfFNEsiIJ0SGvxsbb6y6of8DumYj20qFaI4qhY8FcpCOzYoQ")
	x.httpClient.AddHeader("upgrade-insecure-requests", "1")

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
		name := selection.Find(".item-head").Text()
		tmpUrl, _ := selection.Find(".item-thumbnail a").Attr("href")
		thumb, _ := selection.Find("item-thumbnail img").Attr("data-original")

		pager.List = append(pager.List, model.Video{
			Id:    x.simpleRegEx(tmpUrl, `(\d+).html`),
			Name:  name,
			Thumb: thumb,
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
		name := selection.Find(".item-head").Text()
		tmpUrl, _ := selection.Find(".item-thumbnail a").Attr("href")
		thumb, _ := selection.Find("item-thumbnail img").Attr("data-original")

		pager.List = append(pager.List, model.Video{
			Id:    x.simpleRegEx(tmpUrl, `(\d+).html`),
			Name:  name,
			Thumb: thumb,
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
		video.Links = append(video.Links, model.Link{
			Id:    "E0",
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
	log.Println("[ref]", ref)

	var pKey = x.simpleRegEx(string(buff), `{pkey:"(\S+?)"};`)
	log.Println("[pKey]", pKey)

	buff, err = x.httpClient.Get(fmt.Sprintf("https://player.novipnoad.net/v1/?url=%s&pkey=%s&ref=/anime/%s.html", pid, pKey, vid))
	if err != nil {
		return model.NewError("解析失败1：" + err.Error())
	}
	_ = util.WriteFile("D:\\repo\\github.com\\airplayTV\\api\\_debug\\ddddddddd-iframe.html", buff)

	var device = x.simpleRegEx(string(buff), `params\['device'\] = '(\S+?)';`)
	if len(device) <= 0 {
		return model.NewError("解析失败2：" + err.Error())
	}

	var vKeyHandler = strings.TrimSpace(x.simpleRegEx(string(buff), `function __\(\) \{([\S\s]+?)\}[\r\n]`))
	if len(vKeyHandler) <= 0 {
		return model.NewError("解析失败3：" + err.Error())
	}
	//var tmpVKeyText = x.fuckVKey(vKeyHandler)
	//log.Println("[vKeyHandler]", tmpVKeyText)

	// 2025/11/13 16:21:26 [vKeyHandler] window.sessionStorage.setItem('vkey','{"ckey":"58bc6bfdb129f942542523084e81990d23599f57","ref":"/anime/150044.html","ip":"49.65.131.79","time":"1763022082"}');
	var matchedVKeyValues = x.simpleRegExList(x.fuckVKey(vKeyHandler), `{"ckey":"(\S+)","ref":"(\S+)","ip":"(\S+)","time":"(\S+)"}`)
	if len(matchedVKeyValues) < 5 {
		log.Println("[matchedVKeyValues]", goWebsocket.ToJson(matchedVKeyValues))
		return model.NewError("解析失败4：" + err.Error())
	}

	buff, err = x.httpClient.Get(fmt.Sprintf(
		"https://enc-vod.oss-internal.novipnoad.net/ftn/1748101977.js?ckey=%s&ref=%s&ip=%s&time=%s",
		strings.ToUpper(matchedVKeyValues[1]),
		url.QueryEscape(matchedVKeyValues[2]),
		matchedVKeyValues[3],
		matchedVKeyValues[4],
	))
	if err != nil {
		return model.NewError("解析失败4：" + err.Error())
	}

	// var videoUrl=JSON.decrypt("MTnMsYiESx7n6oSEyGuyBgJKb74kOK95u1jt4cB9/ISmZ7EV/MuekYCpUlJ9CbM3NIPFUZ/w3l5vfcos4ckU5J0+RY8HgPu9ArOKJWvVUKyp1/pdPmM7RTkzMmwoMKYza9xzmO4IAiEvBsG5aWrKFPyQWoNk8/2DB9ribgJAHY7RHlRBqHuY2qvUwTEb/hgDnq8zenOLdE41m29WTQsiB6ZlOuIR8awbQRD8LCgybdYwEc3ieHDHRrV1RsH+Wl45DZNiup1YCaxSZQb/nPNHuz3V/DXRaerBM0Ts2II0Z/Htgw2ikKhFokPFhRY/OUWuEjJv8msyppEdeabB37kA6qOo66V/Gu7tgk3ngxZnJpWTukFq2rxsM//dtsqpU/228n80b0QVmCk9I2riDz8J4GYTy8N/PJ3Xm4aXGLcypALtycEaRWrDnfY/enpE5w10LrJkTnmRw3neEF7Ztx8ZDkvzmx2r8wyJTU4LbgLvkrMlhPKVMkIwIgqpdNNBiI/YUA==");

	_ = util.WriteFile("D:\\repo\\github.com\\airplayTV\\api\\_debug\\ddddddddd-player-js.html", buff)
	/////////////// TODO FIXME

	source.Name = doc.Find(".pclist .jujiinfo h3").Text()

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

func (x NoVipHandler) _findEncryptedLine(htmlContent string) string {
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

const NoVipVkeyBaseChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+/"

func (x NoVipHandler) baseConvert(numberStr string, fromBase, toBase int) string {
	if fromBase < 2 || fromBase > len(NoVipVkeyBaseChars) || toBase < 2 || toBase > len(NoVipVkeyBaseChars) {
		return "0"
	}

	var fromChars = NoVipVkeyBaseChars[:fromBase]
	var toChars = NoVipVkeyBaseChars[:toBase]

	// 将输入字符串从源进制转换为十进制
	var decimalValue = 0
	for i := 0; i < len(numberStr); i++ {
		char := numberStr[i]
		var digitValue = strings.Index(fromChars, string(char))
		if digitValue == -1 {
			continue
		}
		var power = len(numberStr) - 1 - i
		var multiplier = 1
		for j := 0; j < power; j++ {
			multiplier *= fromBase
		}
		decimalValue += digitValue * multiplier
	}

	// 处理零值情况
	if decimalValue == 0 {
		return "0"
	}

	// 将十进制值转换为目标进制
	var result = ""
	var tempValue = decimalValue
	for tempValue > 0 {
		remainder := tempValue % toBase
		result = string(toChars[remainder]) + result
		tempValue = tempValue / toBase
	}

	return result
}

func (x NoVipHandler) parseVKeySession(encodedStr string, param1, param2, delimiterIndex int, replaceChars string, offset int) string {
	var chunkSize = param1 >> 1
	_ = chunkSize

	var decoded = ""
	var i = 0
	for i < len(encodedStr) {
		var chunk = ""
		// 收集直到遇到分隔符的字符
		for i < len(encodedStr) && encodedStr[i] != replaceChars[delimiterIndex] {
			chunk += string(encodedStr[i])
			i++
		}

		// 跳过空块
		if chunk == "" {
			i++
			continue
		}

		// 将字符替换为对应的数字
		for j := 0; j < len(replaceChars); j++ {
			oldChar := string(replaceChars[j])
			newChar := strconv.Itoa(j)
			chunk = strings.ReplaceAll(chunk, oldChar, newChar)
		}

		// 转换进制并获取字符代码
		var converted = x.baseConvert(chunk, delimiterIndex, 12)
		//charCode, err := strconv.Atoi(converted)
		//if err != nil {
		//	log.Printf("转换错误: %v, 原始字符串: %s", err, chunk)
		//	i++
		//	continue
		//}

		var charCode = cast.ToInt(converted) - offset

		//charCode -= offset
		// 确保字符代码在有效范围内
		if charCode >= 0 && charCode <= 1114111 { // Unicode最大码点
			decoded += string(rune(charCode))
		} else {
			log.Printf("无效字符代码: %d", charCode)
		}
		i++ // 跳过分隔符
	}

	// 改进的URL解码逻辑
	if decoded == "" {
		return ""
	}

	// 处理可能的URL编码
	decoded = strings.ReplaceAll(decoded, "+", " ")

	// 使用Go标准库进行URL解码
	finalDecoded, err := url.QueryUnescape(decoded)
	if err != nil {
		log.Printf("URL解码错误: %v, 使用原始字符串", err)
		return decoded
	}
	return finalDecoded
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

func (x NoVipHandler) _parseVideoSource(id, js string) (model.Source, error) {
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

func (x NoVipHandler) getIframeContent(iframeUrl string) (string, error) {
	x.httpClient.AddHeader("referer", czzyHost)
	x.httpClient.AddHeader("sec-fetch-dest", "iframe")
	x.httpClient.AddHeader("sec-fetch-mode", "navigate")
	buff, err := x.httpClient.Get(iframeUrl)
	if err != nil {
		return "", err
	}
	return string(buff), nil
}

func (x NoVipHandler) parseEncryptedResultV2ToUrl(resultV2 string) string {
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

func (x NoVipHandler) parseEncryptedResultV3ToUrl(rand, player string) string {
	buff, err := util.DecryptByAes([]byte("VFBTzdujpR9FWBhe"), []byte(rand), player)
	if err != nil {
		log.Println("[DecryptAesError]", err.Error())
		return ""
	} else {
		var result = gjson.ParseBytes(buff)
		return result.Get("url").String()
	}
}

func (x NoVipHandler) UpdateHeader(header map[string]string) error {
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

func (x NoVipHandler) HoldCookie() error {
	switch r := x.Search("我的", "1").(type) {
	case model.Success:
		return nil
	case model.Error:
		return errors.New(r.Msg)
	}
	return errors.New("未知错误")
}
