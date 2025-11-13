package handler

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/lixiang4u/goWebsocket"
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
	x.httpClient = util.HttpClient{
		//ProxyUrl: "http://127.0.0.1:1080",
	}
	x.httpClient.AddHeader(headers.UserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")
	x.httpClient.AddHeader("cookie", "cf_clearance=OSdZb9fQRcPWVgQ9a9SAWj57OrXT1GXYgxBbWObbonU-1760530853-1.2.1.1-Hg79Wb2wWc.X6Oie20G278A7xEpPrdO7EhVyLkTrfimMNqOJ2TpK9QEwcahMEKos7jhHrO3qg0SMqZwVu_T9jNHUUGINWCzkvHQ3uIua.3qfp9txgWfPLUuEfzQk9nI_3ZBzPhv6HkgF7Ov_c0TXfYGpHDJQxnH2EY_fGJ8n096_Ll3Ej2YCoPCKgtUa5MTFC7.RKsav.xQv1VJpGatWKqcKygpIYCIr_hN50iTk3Y0")
	x.httpClient.AddHeader("referer", noVipHost)
	//x.httpClient.AddHeader("upgrade-insecure-requests", "1")

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
	//log.Println("[ref]", ref)

	source.Name = doc.Find(".entry-title").Text()

	var pKey = x.simpleRegEx(string(buff), `{pkey:"(\S+?)"};`)
	if len(pKey) <= 0 {
		return model.NewError("解析失败：没有pkey")
	}
	//log.Println("[pKey]", pKey)

	buff, err = x.httpClient.Get(fmt.Sprintf("https://player.novipnoad.net/v1/?url=%s&pkey=%s&ref=%s", pid, pKey, ref))
	if err != nil {
		return model.NewError("解析失败：" + err.Error())
	}
	//_ = util.WriteFile("D:\\repo\\github.com\\airplayTV\\api\\_debug\\ddddddddd-iframe.html", buff)

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

	buff, err = x.httpClient.Get(fmt.Sprintf(
		"https://enc-vod.oss-internal.novipnoad.net/ftn/1748101977.js?ckey=%s&ref=%s&ip=%s&time=%s",
		strings.ToUpper(matchedVKeyValues[1]),
		url.QueryEscape(matchedVKeyValues[2]),
		matchedVKeyValues[3],
		matchedVKeyValues[4],
	))
	if err != nil {
		return model.NewError("解析失败2：" + err.Error())
	}

	var encryptedRC4 = x.simpleRegEx(string(buff), `videoUrl=JSON.decrypt\("(\S+?)"\);`)
	if len(encryptedRC4) <= 0 {
		return model.NewError("解析失败：没有加密数据")
	}
	encryptedRC4 = x.decryptNoVipVideoUrl(encryptedRC4)
	//log.Println("[decryptNoVipVideoUrl]", encryptedRC4)
	var decryptJson = gjson.Parse(encryptedRC4)
	if !decryptJson.Get("quality").IsArray() {
		return model.NewError("解析失败3：" + encryptedRC4)
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
