package controller

import (
	"context"
	"fmt"
	"github.com/airplayTV/api/handler"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strings"
	"time"
)

var sourceMap = map[string]handler.IVideo{
	handler.CzzyHandler{}.Name(): handler.CzzyHandler{}.Init(),
	handler.SubbHandler{}.Name(): handler.SubbHandler{}.Init(),
}

type VideoController struct {
}

func (x VideoController) Provider(ctx *gin.Context) {
	var providers = make([]string, 0)
	for s, _ := range sourceMap {
		providers = append(providers, s)
	}
	ctx.JSON(http.StatusOK, model.NewSuccess(providers))
}

func (x VideoController) Search(ctx *gin.Context) {
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	x.response(ctx, h.Search(
		ctx.Query("keyword"),
		ctx.Query("page"),
	))
}

func (x VideoController) TagList(ctx *gin.Context) {
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	x.response(ctx, h.TagList())
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
		x.response(ctx, data)
		return
	}
	var resp = h.VideoList(ctx.Query("tag"), ctx.Query("page"))
	switch resp.(type) {
	case model.Success:
		_ = globalCache.Set(context.Background(), cacheKey, resp, store.WithExpiration(time.Hour*1))
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
		x.response(ctx, data)
		return
	}
	var resp = h.Detail(ctx.Query("id"))
	switch resp.(type) {
	case model.Success:
		_ = globalCache.Set(context.Background(), cacheKey, resp, store.WithExpiration(time.Hour*10))
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
	if err == nil {
		x.response(ctx, data)
		return
	}
	var resp = h.Source(ctx.Query("pid"), ctx.Query("vid"))
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
	x.response(ctx, h.Airplay(
		ctx.Query("pid"),
		ctx.Query("vid"),
	))
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
