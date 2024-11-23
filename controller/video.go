package controller

import (
	"bytes"
	"context"
	"fmt"
	"github.com/airplayTV/api/handler"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lixiang4u/goWebsocket"
	"github.com/zc310/headers"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

var sourceMap = map[string]struct {
	Sort    int
	Handler handler.IVideo
}{
	handler.CzzyHandler{}.Name():     {Sort: 1, Handler: handler.CzzyHandler{}.Init()},
	handler.SubbHandler{}.Name():     {Sort: 2, Handler: handler.SubbHandler{}.Init()},
	handler.YingshiHandler{}.Name():  {Sort: 3, Handler: handler.YingshiHandler{}.Init()},
	handler.MaYiHandler{}.Name():     {Sort: 4, Handler: handler.MaYiHandler{}.Init()},
	handler.NaifeiMeHandler{}.Name(): {Sort: 5, Handler: handler.NaifeiMeHandler{}.Init()},
}

// 不缓存播放数据的源
var noCacheSourceList = []string{
	handler.SubbHandler{}.Name(),     // wangchuanxin.top 第三次请求缓存数据就不能播放
	handler.NaifeiMeHandler{}.Name(), // ki-mi.vip解析压根不能缓存
}

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
	for tmpName, tmpValue := range sourceMap {
		providers = append(providers, ProviderItem{
			Name: tmpName,
			Sort: tmpValue.Sort,
			Tags: tmpValue.Handler.TagList(),
		})
	}
	slices.SortFunc(providers, func(a, b ProviderItem) int {
		return a.Sort - b.Sort
	})
	ctx.JSON(http.StatusOK, model.NewSuccess(providers))
}

func (x VideoController) Search(ctx *gin.Context) {
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
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

func (x VideoController) VideoList(ctx *gin.Context) {
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
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
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
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
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	var cacheKey = fmt.Sprintf("Source::%s_%s_%s", ctx.Query("_source"), ctx.Query("pid"), ctx.Query("vid"))
	data, err := globalCache.Get(context.Background(), cacheKey)
	if err == nil && !slices.Contains(noCacheSourceList, h.Handler.Name()) {
		ctx.Header("Hit-Cache", "true")
		x.response(ctx, data)
		return
	}
	var resp = h.Handler.Source(ctx.Query("pid"), ctx.Query("vid"))
	switch resp.(type) {
	case model.Success:
		_ = globalCache.Set(context.Background(), cacheKey, resp, store.WithExpiration(time.Hour*1))
	}
	x.response(ctx, resp)
}

func (x VideoController) Airplay(ctx *gin.Context) {
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
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

	x.WssManager.SendToGroup(post.Group, websocket.TextMessage, x.WssManager.ToBytes(post))

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
	if !slices.Contains(model.M3u8ProxyHosts, parsed.Host) {
		x.response(ctx, model.NewError("不支持的代理地址："+parsed.Host))
		return
	}
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
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
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
