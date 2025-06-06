package main

import (
	"fmt"
	"github.com/airplayTV/api/controller"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/goWebsocket"
	"log"
	"net/http"
)

func main() {
	var app = gin.Default()
	app.Use(gin.Recovery())
	if err := newRouterApi(app).Run(":8082"); err != nil {
		log.Fatalln(err)
	}
}

func newRouterApi(app *gin.Engine) *gin.Engine {
	app.Static("/m3u8", "./m3u8/")

	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "PATCH", "GET", "POST", "HEAD"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length", "Hit-Cache"},
		AllowCredentials: true,
	}))

	var websocketController = new(controller.WebsocketController)
	var ws = goWebsocket.NewWebsocketManager(true)
	var videoController = controller.VideoController{WssManager: ws}

	// websocket
	ws.On("/", websocketController.Index)
	app.GET("/api/wss", func(ctx *gin.Context) {
		ws.Handler(ctx.Writer, ctx.Request, nil)
	})
	// api接口
	app.GET("/api/video/provider", UseRecovery(videoController.Provider)) // 来源
	app.GET("/api/video/search", UseRecovery(videoController.Search))     // 视频搜索
	app.GET("/api/video/list", UseRecovery(videoController.VideoList))    // 视频列表（根据来源-TAG确定）
	app.GET("/api/video/detail", UseRecovery(videoController.Detail))     // 视频详情
	app.GET("/api/video/source", UseRecovery(videoController.Source))     // 视频播放源
	app.POST("/api/video/control", UseRecovery(videoController.Control))  // 远程遥控功能
	app.GET("/api/m3u8p", UseRecovery(videoController.M3u8p))
	app.POST("/api/cookie", UseRecovery(videoController.SetCookie))               // 手动设置cookie用
	app.GET("/api/qrcode", UseRecovery(videoController.QrCode))                   // 根据文本生成二维码图
	app.GET("/api/m3u8/network-check", UseRecovery(videoController.CheckNetwork)) // 检测视频播放的网络

	app.GET("/api/sse/video/search", UseRecovery(videoController.SearchV2)) // 视频搜索SSE

	app.GET("/api/debug/http-ctx", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, model.NewSuccess(map[string]interface{}{
			"header": ctx.Request.Header,
		}))
	})

	return app
}

func UseRecovery(h func(ctx *gin.Context)) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Println(fmt.Sprintf("服务器异常：%s", util.ToString(gin.H{
					"method": ctx.Request.Method,
					"path":   ctx.Request.URL.Path,
					"ip":     ctx.RemoteIP(),
					"ips":    ctx.ClientIP(),
					"err":    err,
				})))
				ctx.JSON(http.StatusOK, model.NewError("服务器异常", 500))
			}
		}()
		h(ctx)
	}
}
