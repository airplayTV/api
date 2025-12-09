package handler

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/zc310/headers"
	"log"
	"net/url"
	"regexp"
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
	return "CATV"
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
	buff, err := x.requestUrlBypassCheck(fmt.Sprintf(catvDetailUrl, util.DecodeComponentUrl(id)))
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
		if !slices.Contains([]string{"xl0", "xl4"}, lineId) {
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

	//// 干，妈的不稳定

	buff, err := x.httpClient.Get(util.DecodeComponentUrl(pid))
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

func (x CATVHandler) _findEncryptedLine(htmlContent string) string {
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

func (x CATVHandler) _parseVideoSource(id, js string) (model.Source, error) {
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

func (x CATVHandler) requestUrlBypassCheck(requestUrl string) ([]byte, error) {
	buff, err := x.httpClient.Get(requestUrl)
	if err != nil {
		return nil, err
	}
	return x.bypassHuadongCheck(requestUrl, string(buff))
}

func (x CATVHandler) bypassHuadongCheck(requestUrl, respHtml string) ([]byte, error) {
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

func (x CATVHandler) bypassStringToHex(str string) string {
	var codeList []string
	for _, r := range []rune(str) {
		codeList = append(codeList, fmt.Sprintf("%d", int(r)+1))
	}
	return strings.Join(codeList, "")
}

func (x CATVHandler) UpdateHeader(header map[string]string) error {
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

func (x CATVHandler) HoldCookie() error {
	switch r := x.Search("我的", "1").(type) {
	case model.Success:
		return nil
	case model.Error:
		return errors.New(r.Msg)
	}
	return errors.New("未知错误")
}
