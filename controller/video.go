package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/airplayTV/api/handler"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/goWebsocket"
	"github.com/skip2/go-qrcode"
	"github.com/spf13/cast"
	"github.com/zc310/headers"
	"log"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

type VideoController struct {
	WssManager *goWebsocket.WebsocketManager
}

func (x VideoController) Provider(ctx *gin.Context) {
	type ProviderItem struct {
		Name string      `json:"name"`
		Sort int         `json:"sort"`
		Tags interface{} `json:"tags"`
	}
	var providers = make([]ProviderItem, 0)
	var wg sync.WaitGroup
	for tmpName, tmpValue := range x.getSourceMap(ctx) {
		wg.Add(1)
		go func(tmpName string, sort int, h model.IVideo) {
			providers = append(providers, ProviderItem{
				Name: tmpName,
				Sort: sort,
				Tags: h.TagList(),
			})
			wg.Done()
		}(tmpName, tmpValue.Sort, tmpValue.Handler)
	}
	wg.Wait()
	slices.SortFunc(providers, func(a, b ProviderItem) int {
		return a.Sort - b.Sort
	})
	ctx.JSON(http.StatusOK, model.NewSuccess(providers))
}

func (x VideoController) Search(ctx *gin.Context) {
	h, ok := x.getSourceMap(ctx)[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	var cacheKey = fmt.Sprintf("Search::%s_%s_%s", ctx.Query("_source"), ctx.Query("keyword"), ctx.Query("page"))
	data, err := globalCache.Get(context.Background(), cacheKey)
	if err == nil {
		ctx.Header("Hit-Cache", "true")
		x.response(ctx, data)
		return
	}
	var resp = h.Handler.Search(ctx.Query("keyword"), ctx.Query("page"))
	switch resp.(type) {
	case model.Success:
		_ = globalCache.Set(context.Background(), cacheKey, resp, store.WithExpiration(time.Hour*2))
	}
	x.response(ctx, resp)
}

func (x VideoController) SearchV2(ctx *gin.Context) {
	var keyword = ctx.Query("keyword")
	var page = ctx.Query("page")

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	var tmpSourceList = x.getSourceMap(ctx)
	var sourceLength = len(tmpSourceList)
	var ch = make(chan interface{}, sourceLength)
	for tmpSourceName, h := range tmpSourceList {
		go func(name string, handler model.IVideo) {
			var resp interface{}
			defer func() {
				if err := recover(); err != nil {
					log.Println("[SearchV2.Error]", name, err)
				}
				ch <- gin.H{"source": name, "data": resp}
			}()
			if handler.Option().Disable { // 废了
				resp = model.NewError("数据源异常")
			} else if handler.Option().Searchable == false { // 不支持搜
				resp = model.NewError("不支持搜索")
			} else {
				resp = handler.Search(keyword, page)
			}
		}(tmpSourceName, h.Handler)
	}
	for i := 0; i < sourceLength; i++ {
		ctx.SSEvent("update", goWebsocket.ToJson(<-ch))
	}

	ctx.SSEvent("finish", model.NewSuccess(nil))
}

func (x VideoController) VideoList(ctx *gin.Context) {
	h, ok := x.getSourceMap(ctx)[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	var cacheKey = fmt.Sprintf("VideoList::%s_%s_%s", ctx.Query("_source"), ctx.Query("tag"), ctx.Query("page"))
	data, err := globalCache.Get(context.Background(), cacheKey)
	if err == nil {
		ctx.Header("Hit-Cache", "true")
		x.response(ctx, data)
		return
	}
	var resp = h.Handler.VideoList(ctx.Query("tag"), ctx.Query("page"))
	switch resp.(type) {
	case model.Success:
		_ = globalCache.Set(context.Background(), cacheKey, resp, store.WithExpiration(time.Hour*2))
	}
	x.response(ctx, resp)
}

func (x VideoController) Detail(ctx *gin.Context) {
	h, ok := x.getSourceMap(ctx)[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	var cacheKey = fmt.Sprintf("Detail::%s_%s", ctx.Query("_source"), ctx.Query("id"))
	data, err := globalCache.Get(context.Background(), cacheKey)
	if err == nil {
		ctx.Header("Hit-Cache", "true")
		x.response(ctx, data)
		return
	}
	var resp = h.Handler.Detail(ctx.Query("id"))
	switch resp.(type) {
	case model.Success:
		_ = globalCache.Set(context.Background(), cacheKey, resp, store.WithExpiration(time.Hour*24*7))
	}
	x.response(ctx, resp)
}

func (x VideoController) Source(ctx *gin.Context) {
	h, ok := x.getSourceMap(ctx)[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	var p = cast.ToBool(ctx.Query("_m3u8p"))
	var cacheKey = fmt.Sprintf("Source::%s_%s_%s_%s", ctx.Query("_source"), ctx.Query("pid"), ctx.Query("vid"), ctx.Query("_m3u8p"))
	data, err := globalCache.Get(context.Background(), cacheKey)
	if err == nil && !slices.Contains(noCacheSourceList, h.Handler.Name()) {
		ctx.Header("Hit-Cache", "true")
		x.response(ctx, data)
		return
	}
	var resp = h.Handler.Source(ctx.Query("pid"), ctx.Query("vid"))
	switch resp.(type) {
	case model.Success:
		resp = x.m3u8pHandler(p, resp)
		_ = globalCache.Set(context.Background(), cacheKey, resp, store.WithExpiration(time.Hour*1))
	}
	x.response(ctx, resp)
}

func (x VideoController) m3u8pHandler(m3u8p bool, resp interface{}) interface{} {
	if !m3u8p {
		return resp
	}
	switch resp.(type) {
	case model.Success:
		var tmpResp = resp.(model.Success)
		switch tmpResp.Data.(type) {
		case model.Source:
			var tmpSource = tmpResp.Data.(model.Source)
			tmpSource.Url = fmt.Sprintf("%s?url=%s", handler.ApiM3U8ProxyUrl, util.EncodeComponentUrl(tmpSource.Url))
			tmpResp.Data = tmpSource
			resp = tmpResp
		}
	}
	return resp
}

func (x VideoController) Airplay(ctx *gin.Context) {
	h, ok := x.getSourceMap(ctx)[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	x.response(ctx, h.Handler.Airplay(
		ctx.Query("pid"),
		ctx.Query("vid"),
	))
}

func (x VideoController) Control(ctx *gin.Context) {
	var post model.Control
	if err := ctx.ShouldBindBodyWithJSON(&post); err != nil {
		x.response(ctx, model.NewError("参数解析失败"))
		return
	}

	x.WssManager.SendToGroup(post.Group, x.WssManager.ToBytes(post))

	x.response(ctx, model.NewSuccess(nil))
}

func (x VideoController) M3u8p(ctx *gin.Context) {
	var tmpUrl = util.DecodeComponentUrl(ctx.Query("url"))
	parsed, err := url.Parse(tmpUrl)
	if err != nil {
		x.response(ctx, model.NewError("URL地址错误"))
		return
	}
	if len(parsed.Host) == 0 {
		x.response(ctx, model.NewError("URL地址错误"))
		return
	}
	//if !slices.Contains(model.M3u8ProxyHosts, parsed.Host) {
	//	x.response(ctx, model.NewError("不支持的代理地址："+parsed.Host))
	//	return
	//}
	var httpClient = util.HttpClient{}
	header, buff, err := httpClient.GetResponse(tmpUrl)
	if err != nil {
		x.response(ctx, model.NewError("请求失败："+err.Error()))
		return
	}
	for k, v := range header {
		ctx.Header(k, v[0])
	}
	// 跨域
	ctx.Header(headers.AccessControlAllowOrigin, "*")
	ctx.Header(headers.AccessControlAllowHeaders, "*")
	ctx.Header(headers.AccessControlAllowMethods, "*")
	ctx.Header(headers.AccessControlExposeHeaders, "Content-Length,Hit-Cache")

	ctx.DataFromReader(http.StatusOK, -1, header.Get(headers.ContentType), bytes.NewReader(buff), nil)
}

func (x VideoController) SetCookie(ctx *gin.Context) {
	h, ok := x.getSourceMap(ctx)[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}

	var tmpHeaders = map[string]string{
		headers.Cookie:    ctx.PostForm("cookie"),
		headers.UserAgent: ctx.PostForm("user-agent"),
	}
	var err = h.Handler.(handler.CzzyHandler).UpdateHeader(tmpHeaders)
	if err != nil {
		x.response(ctx, model.NewError(err.Error()))
	} else {
		x.response(ctx, model.NewSuccess("cookie已设置"))
	}
}

func (x VideoController) QrCode(ctx *gin.Context) {
	tmpUrl, err := url.QueryUnescape(ctx.Query("url"))
	if err != nil {
		x.response(ctx, model.NewError("参数错误"))
		return
	}
	//_, err = url.Parse(tmpUrl)
	//if err != nil {
	//	x.response(ctx, model.NewError("参数错误"))
	//	return
	//}

	var png []byte
	png, err = qrcode.Encode(tmpUrl, qrcode.High, 256)
	if err != nil {
		x.response(ctx, model.NewError("二维码生成失败："+err.Error()))
		return
	}

	x.response(ctx, model.NewSuccess(gin.H{
		"base64": "data:image/jpg;base64," + base64.StdEncoding.EncodeToString(png),
	}))
}

func (x VideoController) response(ctx *gin.Context, resp interface{}) {
	switch resp.(type) {
	case model.Success:
		ctx.JSON(http.StatusOK, resp)
	case model.Error:
		ctx.JSON(http.StatusOK, resp)
	default:
		log.Println("[resp]", util.ToString(resp))
		ctx.JSON(http.StatusInternalServerError, model.NewError("接口返回数据格式不支持"))
	}
}

func (x VideoController) CheckNetwork(ctx *gin.Context) {
	queryUrl, err := url.QueryUnescape(ctx.Query("url"))
	if err != nil {
		x.response(ctx, model.NewError("参数错误"))
		return
	}
	if util.ParseUrlHost(queryUrl) == "" {
		x.response(ctx, model.NewError("参数错误"))
		return
	}

	type Resp struct {
		Host string `json:"host"`
		Ip   string `json:"ip"`
		Addr string `json:"addr"`
		Url  string `json:"url"`
	}

	var resolvedUrlAddr = make([]Resp, 0)
	playList, err := util.ParsePlayUrlList(queryUrl)
	if err != nil {
		x.response(ctx, model.NewError("播放文件处理失败："+err.Error()))
		return
	}
	for _, tmpUrl := range playList {
		parsed, err := url.Parse(tmpUrl)
		if err != nil {
			continue
		}
		addrs, err := net.LookupHost(parsed.Hostname())
		if err != nil {
			continue
		}
		var region = util.IpAddress(addrs[0])
		resolvedUrlAddr = append(resolvedUrlAddr, Resp{
			Host: parsed.Hostname(),
			Ip:   addrs[0],
			Addr: strings.TrimSpace(fmt.Sprintf("%s %s", region.Country, region.Province)),
			Url:  tmpUrl,
		})
	}

	x.response(ctx, model.NewSuccess(gin.H{
		"url":      queryUrl,
		"resolved": resolvedUrlAddr,
	}))
}

func (x VideoController) SourceStat(ctx *gin.Context) {
	var qTime = ctx.DefaultQuery("time", time.Now().Format("2006010215"))

	var p = filepath.Join(util.AppPath(), fmt.Sprintf("cache/stat/source-stat-%s.json", qTime))
	var resolutionList []model.VideoResolution

	err := json.Unmarshal(util.ReadFile(p), &resolutionList)
	if err != nil {
		x.response(ctx, model.NewError("暂无数据："+err.Error()))
		return
	}
	x.response(ctx, model.NewSuccess(resolutionList))
}
