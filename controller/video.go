package controller

import (
	"github.com/airplayTV/api/handler"
	"github.com/airplayTV/api/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

var sourceMap = map[string]handler.IVideo{
	handler.CzzyHandler{}.Name(): handler.CzzyHandler{},
	handler.SubbHandler{}.Name(): handler.SubbHandler{},
}

type VideoController struct {
}

func (x VideoController) Provider(ctx *gin.Context) {
	var providers interface{}
	ctx.JSON(http.StatusOK, model.NewSuccess(providers))
}
func (x VideoController) Search(ctx *gin.Context) {

}
func (x VideoController) TagList(ctx *gin.Context) {
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, gin.H{"err": "source err"})
		return
	}
	ctx.JSON(http.StatusOK, h.TagList())
}
func (x VideoController) VideoList(ctx *gin.Context) {
}
func (x VideoController) Detail(ctx *gin.Context) {

}
func (x VideoController) Source(ctx *gin.Context) {

}
func (x VideoController) Airplay(ctx *gin.Context) {
	h, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
	if !ok {
		ctx.JSON(http.StatusOK, gin.H{"err": "source err"})
		return
	}
	ctx.JSON(http.StatusOK, h.TagList())

}
func X() {

}

//func (x VideoController) checkVideoHandler(ctx *gin.Context) (handler.IVideo, bool) {
//	v, ok := sourceMap[strings.TrimSpace(ctx.Query("_source"))]
//
//	log.Println("[_source]", strings.TrimSpace(ctx.Query("_source")))
//	log.Println("[sourceMap]", sourceMap)
//
//	return v, ok
//}

func NewVideo(ctx *gin.Context) {
	//
	//
	//
	//
	//switch  {
	//case handler.CzzyHandler{}.Name():
	//	return handler.CzzyHandler{}
	//}
}
