package controller

import (
	"github.com/airplayTV/api/handler"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strings"
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
	x.response(ctx, h.VideoList(
		ctx.Query("tag"),
		ctx.Query("page"),
	))
}

func (x VideoController) Detail(ctx *gin.Context) {
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	x.response(ctx, h.Detail(
		ctx.Query("id"),
	))
}

func (x VideoController) Source(ctx *gin.Context) {
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, model.NewError("数据源错误"))
		return
	}
	x.response(ctx, h.Source(
		ctx.Query("pid"),
		ctx.Query("vid"),
	))
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
