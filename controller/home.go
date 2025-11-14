package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/goWebsocket"
	"io"
	"log"
	"net/http"
)

type HomeController struct {
}

func (x HomeController) Index(ctx *gin.Context) {
	ctx.String(http.StatusOK, "AirplayTV")
}

func (x HomeController) Debug(ctx *gin.Context) {
	buff, _ := io.ReadAll(ctx.Request.Body)
	log.Println("[debug]", goWebsocket.ToJson(map[string]interface{}{
		"uri":    ctx.Request.RequestURI,
		"header": ctx.Request.Header,
		"body":   string(buff),
	}))

	ctx.String(http.StatusOK, "debug")
}
