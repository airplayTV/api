package handler

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type CzzyHandler struct {
	Handler
}

func (x CzzyHandler) Init() IVideo {
	x.httpClient = util.HttpClient{}
	return x
}

func (x CzzyHandler) Name() string {
	return "czzy"
}

func (x CzzyHandler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "豆瓣Top250", "value": "dbtop250"})
	tags = append(tags, gin.H{"name": "电影", "value": "movie_bt"})
	tags = append(tags, gin.H{"name": "高分影视", "value": "gaofenyingshi"})
	tags = append(tags, gin.H{"name": "最新电影", "value": "zuixindianying"})
	tags = append(tags, gin.H{"name": "国产剧", "value": "gcj"})
	tags = append(tags, gin.H{"name": "美剧", "value": "meijutt"})
	tags = append(tags, gin.H{"name": "韩剧", "value": "hanjutv"})
	tags = append(tags, gin.H{"name": "番剧", "value": "fanju"})
	return tags
}

func (x CzzyHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x CzzyHandler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x CzzyHandler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x CzzyHandler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x CzzyHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x CzzyHandler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(czzyTagUrl, tagName, x.parsePageNumber(page)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var pager = model.Pager{Limit: 25, Page: x.parsePageNumber(page)}

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
		n := x.parsePageNumber(x.simpleRegEx(tmpHref, `/page/(\d+)`))
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
		if n >= pager.Pages {
			pager.Pages = n
		}
	})

	pager.Page, _ = strconv.Atoi(doc.Find(".pagenavi_txt .current").Text())

	return model.NewSuccess(pager)
}

func (x CzzyHandler) _search(keyword, page string) interface{} {
	return model.NewError("尚未实现")
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
