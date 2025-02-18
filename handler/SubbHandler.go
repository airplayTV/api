package handler

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"github.com/zc310/headers"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type SubbHandler struct {
	Handler
}

func (x SubbHandler) Init(options interface{}) IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.Referer, subbHost)
	return x
}

func (x SubbHandler) Name() string {
	return "素白白影视"
}

func (x SubbHandler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "new-movie"})
	tags = append(tags, gin.H{"name": "高分影视", "value": "high-movie"})
	tags = append(tags, gin.H{"name": "最新电影", "value": "new-movie"})
	tags = append(tags, gin.H{"name": "香港经典", "value": "hongkong-movie"})
	tags = append(tags, gin.H{"name": "电视剧", "value": "tv-drama"})
	tags = append(tags, gin.H{"name": "欧美剧", "value": "american-drama"})
	tags = append(tags, gin.H{"name": "动漫剧", "value": "anime-drama"})
	return tags
}

func (x SubbHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x SubbHandler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x SubbHandler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x SubbHandler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x SubbHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x SubbHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.requestUrlBypassCheck(fmt.Sprintf(subbTagUrl, tagName, x.parsePageNumber(page)))
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
			Id:         x.simpleRegEx(tmpUrl, `(\d+)`),
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
		tmpList := strings.Split(tmpHref, "/")
		n, _ := strconv.Atoi(tmpList[len(tmpList)-1])
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
	})

	pager.Page, _ = strconv.Atoi(doc.Find(".pagenavi_txt .current").Text())

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x SubbHandler) _search(keyword, page string) interface{} {
	var pager = model.Pager{Limit: 25, Page: x.parsePageNumber(page)}
	buff, err := x.requestUrlBypassCheck(fmt.Sprintf(subbSearchUrl, keyword, pager.Page))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		log.Println("[文档解析失败]", err.Error())
		return pager
	}
	doc.Find(".search_list ul li").Each(func(i int, selection *goquery.Selection) {
		name := selection.Find(".dytit a").Text()
		tmpUrl, _ := selection.Find(".dytit a").Attr("href")
		thumb, _ := selection.Find("img.thumb").Attr("data-original")
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
	//pager.Page, _ = strconv.Atoi(doc.Find(".pagenavi_txt .current").Text())
	pager.Total = pager.Pages * pager.Limit

	return model.NewSuccess(pager)
}

func (x SubbHandler) _detail(id string) interface{} {
	buff, err := x.requestUrlBypassCheck(fmt.Sprintf(subbDetailUrl, id))
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

	return model.NewSuccess(video)
}

func (x SubbHandler) _source(pid, vid string) interface{} {
	var source = model.Source{Id: pid, Vid: vid}
	buff, err := x.requestUrlBypassCheck(fmt.Sprintf(subbPlayUrl, pid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	source.Name = doc.Find(".paycon .ptit a").Text()

	{
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
	}

	{
		// 解析另一种iframe嵌套的视频
		iframeUrl, _ := doc.Find(".videoplay iframe").Attr("src")
		if strings.TrimSpace(iframeUrl) != "" {
			log.Println("[iframeUrl]", iframeUrl)
		}
	}

	if len(source.Source) == 0 {
		return model.NewError("播放地址解析失败")
	}

	// 视频类型问题处理
	source.Type = x.parseVideoType(source.Source)
	source.Url = source.Source

	if len(source.Source) == 0 {
		return model.NewError("播放地址处理失败")
	}

	return model.NewSuccess(source)

}

func (x SubbHandler) _findEncryptedLine(htmlContent string) string {
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

func (x SubbHandler) _parseVideoSource(id, js string) (model.Source, error) {
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

func (x SubbHandler) requestUrlBypassCheck(requestUrl string) ([]byte, error) {
	buff, err := x.httpClient.Get(requestUrl)
	if err != nil {
		return nil, err
	}
	return x.bypassHuadongCheck(requestUrl, string(buff))
}

func (x SubbHandler) bypassHuadongCheck(requestUrl, respHtml string) ([]byte, error) {
	var findCheckUrl = x.simpleRegEx(respHtml, `src="(\S+)"></script>`)
	if !strings.Contains(findCheckUrl, "huadong") && !strings.Contains(respHtml, "人机身份验证") {
		return []byte(respHtml), nil
	}
	_, buff, err := x.httpClient.GetResponse(fmt.Sprintf("%s/%s", strings.TrimRight(subbHost, "/"), strings.TrimLeft(findCheckUrl, "/")))
	if err != nil {
		return nil, err
	}
	respHtml = string(buff)

	var kvList = x.simpleRegExList(respHtml, `,key="(\S+)",value="(\S+)";`)
	var urlList = x.simpleRegExList(respHtml, `c\.get\("(\S+)\?type=(\S+)&key=`)
	if len(kvList) < 3 || len(urlList) < 3 {
		return nil, errors.New("验证数据解析失败")
	}

	var checkUrl = fmt.Sprintf(
		"%s/%s?type=%s&key=%s&value=%s",
		strings.TrimRight(subbHost, "/"),
		strings.TrimLeft(urlList[1], "/"),
		urlList[2],
		kvList[1],
		util.StringMd5(x.bypassStringToHex(kvList[2])),
	)
	log.Println("[checkUrl]", checkUrl)

	header, buff, err := x.httpClient.GetResponse(checkUrl)
	if err != nil {
		return nil, err
	}

	// 修改请求头，全局生效
	x.httpClient.AddHeader(headers.Cookie, header.Get("Set-Cookie"))

	_, buff, err = x.httpClient.GetResponse(requestUrl)
	if err != nil {
		return nil, err
	}

	_ = util.SaveHttpHeader(x.Name(), x.httpClient.GetHeaders())

	return buff, nil
}

func (x SubbHandler) bypassStringToHex(str string) string {
	var codeList []string
	for _, r := range []rune(str) {
		codeList = append(codeList, fmt.Sprintf("%d", int(r)+1))
	}
	return strings.Join(codeList, "")
}

func (x SubbHandler) UpdateHeader(header map[string]string) error {
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

func (x SubbHandler) HoldCookie() error {
	switch r := x.Search("我的", "1").(type) {
	case model.Success:
		return nil
	case model.Error:
		return errors.New(r.Msg)
	}
	return errors.New("未知错误")
}
