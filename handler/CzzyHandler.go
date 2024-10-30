package handler

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
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
	tags = append(tags, gin.H{"name": "dbtop250", "value": "dbtop250"})
	tags = append(tags, gin.H{"name": "movie_bt", "value": "movie_bt"})
	tags = append(tags, gin.H{"name": "gaofenyingshi", "value": "gaofenyingshi"})
	tags = append(tags, gin.H{"name": "zuixindianying", "value": "zuixindianying"})
	tags = append(tags, gin.H{"name": "gcj", "value": "gcj"})
	tags = append(tags, gin.H{"name": "meijutt", "value": "meijutt"})
	tags = append(tags, gin.H{"name": "hanjutv", "value": "hanjutv"})
	tags = append(tags, gin.H{"name": "fanju", "value": "fanju"})
	return model.NewSuccess(tags)
}

func (x CzzyHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
}

func (x CzzyHandler) Search(keyword, page string) interface{} {
	return gin.H{}
}

func (x CzzyHandler) Detail(id string) interface{} {
	return gin.H{}
}

func (x CzzyHandler) Source(pid, vid string) interface{} {
	return gin.H{}
}

func (x CzzyHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}

//

func (x CzzyHandler) _tagList() interface{} {
	return nil
}

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
		tmpList := strings.Split(tmpHref, "/")
		n, _ := strconv.Atoi(tmpList[len(tmpList)-1])
		if n*pager.Limit > pager.Total {
			pager.Total = n * pager.Limit
		}
	})

	pager.Page, _ = strconv.Atoi(doc.Find(".pagenavi_txt .current").Text())

	return model.NewSuccess(pager)
}
