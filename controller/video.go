package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/airplayTV/api/handler"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/gin-gonic/gin"
	"github.com/go-http-utils/headers"
	"github.com/lixiang4u/goWebsocket"
	"github.com/skip2/go-qrcode"
	"github.com/spf13/cast"
	"log"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

type VideoController struct {
	WssManager *goWebsocket.WebsocketManager
}

func (x VideoController) Provider(ctx *gin.Context) {
	x.LogVisitor(ctx.ClientIP())

	var providers = x.sortedSourceList(x.getSourceMap(ctx))

	ctx.JSON(http.StatusOK, model.NewSuccess(providers))
}

func (x VideoController) sortedSourceList(sourceMap map[string]model.SourceHandler) []model.ProviderItem {
	var providers = make([]model.ProviderItem, 0)
	var wg sync.WaitGroup
	for tmpName, tmpValue := range sourceMap {
		wg.Add(1)
		go func(tmpName string, sort int, h model.IVideo) {
			providers = append(providers, model.ProviderItem{
				Name: tmpName,
				Sort: sort,
				Tags: h.TagList(),
			})
			wg.Done()
		}(tmpName, tmpValue.Sort, tmpValue.Handler)
	}
	wg.Wait()

	// 排序啊
	var key = fmt.Sprintf("app-source-stat::provider")
	var resp = model.WithCache(key, store.WithExpiration(time.Hour*2), func() interface{} {
		var date = model.VideoResolution{}.MaxDate()
		return model.VideoResolution{}.List(date)
	}).([]model.VideoResolution)
	var sortMap = make(map[string]int)
	for i, resolution := range resp {
		sortMap[resolution.Source] = i
	}
	slices.SortFunc(providers, func(a, b model.ProviderItem) int {
		v1, ok1 := sortMap[a.Name]
		v2, ok2 := sortMap[b.Name]
		if !ok1 || !ok2 {
			return a.Sort - b.Sort
		}
		return v1 - v2
	})
	for i, _ := range providers {
		providers[i].Sort = i + 1
	}

	return providers
}

func (x VideoController) parseSourceHandler(ctx *gin.Context) (model.SourceHandler, error) {
	h, ok := x.getSourceMap(ctx)[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		return h, errors.New("数据源错误")
	}
	if h.Handler.Option().Disable {
		return h, errors.New("数据源错误")
	}
	return h, nil
}

func (x VideoController) Search(ctx *gin.Context) {
	x.LogVisitor(ctx.ClientIP())

	h, err := x.parseSourceHandler(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, model.NewError(err.Error()))
		return
	}
	if !h.Handler.Option().Searchable {
		ctx.JSON(http.StatusOK, model.NewError("当前源不支持搜索"))
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

func (x VideoController) getSourceSortMap(sourceMap map[string]model.SourceHandler) map[string]int {
	var sourceSortMap = make(map[string]int)
	for _, item := range x.sortedSourceList(sourceMap) {
		sourceSortMap[item.Name] = item.Sort
	}
	return sourceSortMap
}

func (x VideoController) SearchV2(ctx *gin.Context) {
	x.LogVisitor(ctx.ClientIP())

	var keyword = ctx.Query("keyword")
	var page = ctx.Query("page")

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	var tmpSourceList = x.getSourceMap(ctx)
	var sourceSortMap = x.getSourceSortMap(tmpSourceList)
	var sourceLength = len(tmpSourceList)
	var ch = make(chan interface{}, sourceLength)
	for tmpSourceName, h := range tmpSourceList {
		go func(name string, handler model.IVideo) {
			var resp interface{}
			defer func() {
				if err := recover(); err != nil {
					log.Println("[SearchV2.Error]", name, err)
				}
				tmpSort, ok := sourceSortMap[name]
				if !ok {
					tmpSort = 99999
				}
				ch <- gin.H{"source": name, "data": resp, "sort": tmpSort}
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
	x.LogVisitor(ctx.ClientIP())

	h, err := x.parseSourceHandler(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, model.NewError(err.Error()))
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
	x.LogVisitor(ctx.ClientIP())

	h, err := x.parseSourceHandler(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, model.NewError(err.Error()))
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
	x.LogVisitor(ctx.ClientIP())

	h, err := x.parseSourceHandler(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, model.NewError(err.Error()))
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
		go func(data interface{}) {
			defer func() { _ = recover() }()
			switch tmpSource := data.(type) {
			case model.Source:
				log.Println("[VideoSourceUrl]", goWebsocket.ToJson(map[string]interface{}{
					"_source": ctx.Query("_source"),
					"vid":     tmpSource.Vid,
					"pid":     tmpSource.Id,
					"name":    tmpSource.Name,
					"source":  tmpSource.Source,
				}))
			}
		}(resp.(model.Success).Data)
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
	h, err := x.parseSourceHandler(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, model.NewError(err.Error()))
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
	x.LogVisitor(ctx.ClientIP())

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

	ctx.Header("X-Source-Url", tmpUrl)

	var httpClient = x.resolveHttpClientCtx(tmpUrl)
	header, buff, err := httpClient.GetResponse(util.Http2HttpsUrl(tmpUrl))
	if err != nil {
		x.response(ctx, model.NewError("请求失败："+err.Error()))
		return
	}
	buff, err = util.FormatM3u8Url(buff, util.Http2HttpsUrl(tmpUrl), x.resolveUrlRedirect)
	if err != nil {
		x.response(ctx, model.NewError("文件处理异常："+err.Error()))
		return
	}

	for k, v := range header {
		if k == headers.ContentLength {
			continue
		}
		ctx.Header(k, v[0])
	}
	// 跨域
	ctx.Header(headers.AccessControlAllowOrigin, "*")
	ctx.Header(headers.AccessControlAllowHeaders, "*")
	ctx.Header(headers.AccessControlAllowMethods, "*")
	ctx.Header(headers.AccessControlExposeHeaders, "Content-Length,Hit-Cache")
	ctx.Header(headers.CacheControl, "no-cache, no-store")

	ctx.DataFromReader(http.StatusOK, -1, header.Get(headers.ContentType), bytes.NewReader(buff), nil)
}

func (x VideoController) resolveHttpClientCtx(tmpUrl string) util.HttpClient {
	var httpClient = util.HttpClient{}
	//log.Println("[util.ParseHost(tmpUrl)]", util.ParseHost(tmpUrl))
	switch util.ParseHost(tmpUrl) {
	case "media.oss-internal.novipnoad.net":
		httpClient.AddHeader(headers.Origin, "https://player.novipnoad.net")
		httpClient.AddHeader(headers.UserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
		if strings.Contains(tmpUrl, "media.oss-internal.novipnoad.net/ts/") {
			httpClient.AddHeader(headers.Accept, "*/*")
			httpClient.AddHeader(headers.Origin, "https://media.oss-internal.novipnoad.net")
		}
	}

	return httpClient
}

func (x VideoController) resolveUrlRedirect(tmpUrl string) string {
	switch util.ParseHost(tmpUrl) {
	case "media.oss-internal.novipnoad.net":
		tmpUrl = fmt.Sprintf("%s?url=%s", handler.ApiRedirectUrl, util.EncodeComponentUrl(tmpUrl))
	}
	return tmpUrl
}

func (x VideoController) ProxyRedirect(ctx *gin.Context) {
	x.LogVisitor(ctx.ClientIP())

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

	ctx.Header("X-Source-Url", tmpUrl)

	var httpClient = x.resolveHttpClientCtx(tmpUrl)
	location, err := httpClient.Location(tmpUrl)
	if err != nil {
		x.response(ctx, model.NewError(err.Error()))
		return
	}

	//log.Println("[location]", location)

	ctx.Redirect(307, location)
	return
}

func (x VideoController) SetCookie(ctx *gin.Context) {
	h, err := x.parseSourceHandler(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, model.NewError(err.Error()))
		return
	}

	var tmpHeaders = map[string]string{
		headers.Cookie:    ctx.PostForm("cookie"),
		headers.UserAgent: ctx.PostForm("user-agent"),
	}
	err = h.Handler.(handler.CzzyHandler).UpdateHeader(tmpHeaders)
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
	x.LogVisitor(ctx.ClientIP())

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
	x.LogVisitor(ctx.ClientIP())

	var key = fmt.Sprintf("app-source-stat::source-stat")
	var resp = model.WithCache(key, store.WithExpiration(time.Minute*2), func() interface{} {
		var date = model.VideoResolution{}.MaxDate()
		return model.VideoResolution{}.List(date)
	})
	if resp == nil || len(resp.([]model.VideoResolution)) <= 0 {
		x.response(ctx, model.NewError("暂无数据"))
		return
	}
	x.response(ctx, model.NewSuccess(resp))
}

func (x VideoController) LogVisitor(ip string) {
	var key = fmt.Sprintf("app-visitor-log::%s", ip)
	model.WithCache(key, store.WithExpiration(time.Duration(util.TodayLeftSeconds()*int64(time.Second))), func() interface{} {
		if err := (model.Visitor{}).CreateOrUpdate(ip); err != nil {
			log.Println("[Visitor.CreateOrUpdate]", err.Error())
		}
		return true
	})
}

func (x VideoController) TotalVisitor() int64 {
	var key = fmt.Sprintf("app-visitor-count::visitor-count")
	return model.WithCache(key, store.WithExpiration(time.Minute*30), func() interface{} {
		return (model.Visitor{}).Total()
	}).(int64)
}
