package main

import (
	"github.com/airplayTV/api/controller"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/goWebsocket"
	"log"
)

func main() {
	var app = gin.Default()
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
	app.GET("/api/video/provider", videoController.Provider) // 来源
	app.GET("/api/video/search", videoController.Search)     // 视频搜索
	app.GET("/api/video/list", videoController.VideoList)    // 视频列表（根据来源-TAG确定）
	app.GET("/api/video/detail", videoController.Detail)     // 视频详情
	app.GET("/api/video/source", videoController.Source)     // 视频播放源
	app.POST("/api/video/control", videoController.Control)  // 远程遥控功能
	app.GET("/api/m3u8p", videoController.M3u8p)
	app.POST("/api/cookie", videoController.SetCookie) // 手动设置cookie用

	app.GET("/api/sse/video/search", videoController.SearchV2) // 视频搜索SSE

	return app
}
