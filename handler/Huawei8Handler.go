package handler

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"github.com/zc310/headers"
	"strconv"
	"strings"
)

type Huawei8Handler struct {
	Handler
}

func (x Huawei8Handler) Init(options interface{}) model.IVideo {
	x.httpClient = util.HttpClient{}
	x.httpClient.AddHeader(headers.UserAgent, useragent)
	x.httpClient.AddHeader(headers.Origin, huawei8Host)
	x.httpClient.AddHeader(headers.Referer, huawei8Host)
	return x
}

func (x Huawei8Handler) Name() string {
	return "华为吧"
}

func (x Huawei8Handler) Option() model.CmsZyOption {
	return x.option
}

func (x Huawei8Handler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "电影", "value": "20"})
	tags = append(tags, gin.H{"name": "电视剧", "value": "60"})
	tags = append(tags, gin.H{"name": "综艺", "value": "82"})
	tags = append(tags, gin.H{"name": "动漫", "value": "80"})
	tags = append(tags, gin.H{"name": "纪录片", "value": "86"})
	tags = append(tags, gin.H{"name": "短剧", "value": "120"})
	return tags
}

func (x Huawei8Handler) defaultThumb() string {
	return "https://iph.href.lu/360x528?text=HW8&fg=bcbcbc&bg=eeeeee"
}

func (x Huawei8Handler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x Huawei8Handler) Search(keyword, page string) interface{} {
	return x._search(keyword, page)
}

func (x Huawei8Handler) Detail(id string) interface{} {
	return x._detail(id)
}

func (x Huawei8Handler) Source(pid, vid string) interface{} {
	return x._source(pid, vid)
}

func (x Huawei8Handler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x Huawei8Handler) _videoList(tagName, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(huawei8TagUrl, tagName, x.parsePageNumber(page)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var pager = model.Pager{Limit: 50, Page: x.parsePageNumber(page), List: make([]model.Video, 0)}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc.Find(".xing_vb ul li").Each(func(i int, selection *goquery.Selection) {
		if selection.Find(".xing_vb4").Size() > 0 {
			name := x.splitHW8VideoTitle(selection.Find(".xing_vb4 a").Text())
			tmpUrl := selection.Find(".xing_vb4 a").AttrOr("href", "")
			tag := selection.Find(".xing_vb5").Text()
			pager.List = append(pager.List, model.Video{
				Id:         x.simpleRegEx(tmpUrl, `(\d+).html`),
				Name:       strings.TrimSpace(name),
				Thumb:      x.defaultThumb(),
				Url:        tmpUrl,
				Tag:        tag,
				Resolution: "",
			})
		}

	})

	doc.Find(".xing_vb .page_info .page_link").Each(func(i int, selection *goquery.Selection) {
		n := x.parsePageNumber(x.simpleRegEx(selection.AttrOr("href", ""), `/page/(\d+)`))
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
		if n >= pager.Pages {
			pager.Pages = n
		}
	})

	pager.Page, _ = strconv.Atoi(x.simpleRegEx(doc.Find(".page_tip").Text(), `当前(\d+)\/`))

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x Huawei8Handler) _search(keyword, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(huawei8SearchUrl, x.parsePageNumber(page), keyword))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	var pager = model.Pager{Limit: 50, Page: x.parsePageNumber(page)}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	doc.Find(".xing_vb ul li").Each(func(i int, selection *goquery.Selection) {
		if selection.Find(".xing_vb4").Size() > 0 {
			name := x.splitHW8VideoTitle(selection.Find(".xing_vb4 a").Text())
			tmpUrl := selection.Find(".xing_vb4 a").AttrOr("href", "")
			tag := selection.Find(".xing_vb5").Text()
			pager.List = append(pager.List, model.Video{
				Id:         x.simpleRegEx(tmpUrl, `(\d+).html`),
				Name:       strings.TrimSpace(name),
				Thumb:      x.defaultThumb(),
				Url:        tmpUrl,
				Tag:        tag,
				Resolution: "",
			})
		}

	})

	doc.Find(".xing_vb .page_info .page_link").Each(func(i int, selection *goquery.Selection) {
		n := x.parsePageNumber(x.simpleRegEx(selection.AttrOr("href", ""), `/page/(\d+)`))
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
		if n >= pager.Pages {
			pager.Pages = n
		}
	})

	pager.Page, _ = strconv.Atoi(x.simpleRegEx(doc.Find(".page_tip").Text(), `当前(\d+)\/`))

	if len(pager.List) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(pager)
}

func (x Huawei8Handler) parseVidTypeId(str string) (vid, tid string, err error) {
	var tmpList = strings.Split(str, "-")
	if len(tmpList) != 2 {
		return "", "", errors.New("请求参数错误")
	}
	vid = tmpList[0]
	tid = tmpList[1]
	return vid, tid, nil
}

func (x Huawei8Handler) _detail(id string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(huawei8DetailUrl, id))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var video = model.Video{Id: id}
	doc.Find(".vodplayinfo li input").Each(func(i int, selection *goquery.Selection) {
		tmpHref := selection.AttrOr("value", "")
		video.Links = append(video.Links, model.Link{
			Id:    fmt.Sprintf("%d", i),
			Name:  fmt.Sprintf("%d", i),
			Url:   tmpHref,
			Group: "HW8",
		})
	})

	video.Name = doc.Find(".vodInfo .vodh h2").Text()
	video.Thumb = util.FillUrlHost(fmt.Sprintf(huawei8DetailUrl, id), doc.Find(".vod .vodImg img").AttrOr("src", ""))
	video.Intro = strings.TrimSpace(doc.Find(".ibox .vodplayinfo").Eq(0).Text())

	return model.NewSuccess(video)

}

func (x Huawei8Handler) _source(pid, vid string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(huawei8PlayUrl, vid))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var source = model.Source{Id: pid, Vid: vid}
	//log.Println("[X]", len(string(buff)))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buff)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	var tmpPid = util.ParseNumber(pid)
	doc.Find(".vodplayinfo li input").Each(func(i int, selection *goquery.Selection) {
		if tmpPid == i {
			source.Url = selection.AttrOr("value", "")
		}
	})
	source.Name = doc.Find(".vodInfo .vodh h2").Text()
	source.Thumb = util.FillUrlHost(fmt.Sprintf(huawei8DetailUrl, vid), doc.Find(".vod .vodImg img").AttrOr("src", ""))
	source.Source = fmt.Sprintf(huawei8DetailUrl, vid)
	source.Type = x.parseVideoType(source.Url)

	if len(source.Url) == 0 {
		return model.NewError("暂无数据")
	}

	return model.NewSuccess(source)
}

func (x Huawei8Handler) splitHW8VideoTitle(name string) string {
	var lst = strings.Split(name, " ")
	if len(lst) == 0 {
		return ""
	}
	return lst[0]
}

func (x Huawei8Handler) UpdateHeader(header map[string]string) error {
	return nil
}

func (x Huawei8Handler) HoldCookie() error {
	return nil
}
