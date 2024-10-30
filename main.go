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
	if err := newRouterWS(newRouterApi(app)).Run(":8082"); err != nil {
		log.Fatalln(err)
	}
}

func newRouterApi(app *gin.Engine) *gin.Engine {
	app.Static("/m3u8", "./m3u8/")

	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "PATCH", "GET", "POST", "HEAD"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	var videoController = new(controller.VideoController)
	app.GET("/api/video/provider", videoController.Provider) // 来源
	app.GET("/api/video/search", videoController.Search)     // 视频搜索
	app.GET("/api/video/list", videoController.VideoList)    // 视频列表（根据来源-TAG确定）
	app.GET("/api/video/detail", videoController.Detail)     // 视频详情
	app.GET("/api/video/source", videoController.Source)     // 视频播放源
	app.GET("/api/video/airplay", videoController.Airplay)   // 远程遥控功能

	return app
}

func newRouterWS(app *gin.Engine) *gin.Engine {
	websocketController := new(controller.WebsocketController)

	var ws = goWebsocket.NewWebsocketManager(true)
	ws.On("/", websocketController.Index)
	app.GET("/api/wss", func(ctx *gin.Context) {
		ws.Handler(ctx.Writer, ctx.Request, nil)
	})

	return app
}
