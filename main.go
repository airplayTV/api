package main

import (
	"fmt"
	"github.com/airplayTV/api/controller"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/task"
	"github.com/airplayTV/api/util"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/goWebsocket"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func init() {
	log.SetOutput(&lumberjack.Logger{
		Filename: filepath.Join(util.AppPath(), "app.log"),
		MaxSize:  100, // megabytes
		//MaxBackups: 3,
		MaxAge: 90, //days
		//Compress: true, // disabled by default
	})
}

func main() {
	go task.NewSourceStat().Run()
	go task.NewHoldCookie().Run()

	var port = portUse(8082)
	var app = gin.Default()
	app.Use(gin.Recovery())
	app = newRouterApi(app) // start api
	//app = newRouterWeb(app) // start frontend
	//go serveHtml(port)
	if err := app.Run(fmt.Sprintf(":%d", port)); err != nil {
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
	var videoController = controller.VideoController{WssManager: controller.AppSocket}
	var homeController = controller.HomeController{}

	// websocket
	controller.AppSocket.On(goWebsocket.Event(goWebsocket.EventConnect).String(), websocketController.Connect)
	controller.AppSocket.On("join-group", websocketController.JoinGroup)
	controller.AppSocket.On("send-to-group", websocketController.SendToGroup)
	app.GET("/api/wss", func(ctx *gin.Context) {
		controller.AppSocket.Handler(ctx.Writer, ctx.Request, nil)
	})

	// api接口
	app.GET("/", UseRecovery(homeController.Index))                       // 来源
	app.GET("/api/video/provider", UseRecovery(videoController.Provider)) // 来源
	app.GET("/api/video/search", UseRecovery(videoController.Search))     // 视频搜索
	app.GET("/api/video/list", UseRecovery(videoController.VideoList))    // 视频列表（根据来源-TAG确定）
	app.GET("/api/video/detail", UseRecovery(videoController.Detail))     // 视频详情
	app.GET("/api/video/source", UseRecovery(videoController.Source))     // 视频播放源
	app.POST("/api/video/control", UseRecovery(videoController.Control))  // 远程遥控功能
	app.GET("/api/m3u8p", UseRecovery(videoController.M3u8p))
	app.GET("/api/redirect", UseRecovery(videoController.ProxyRedirect))
	app.POST("/api/cookie", UseRecovery(videoController.SetCookie))               // 手动设置cookie用
	app.GET("/api/qrcode", UseRecovery(videoController.QrCode))                   // 根据文本生成二维码图
	app.GET("/api/m3u8/network-check", UseRecovery(videoController.CheckNetwork)) // 检测视频播放的网络
	app.GET("/api/sse/video/search", UseRecovery(videoController.SearchV2))       // 视频搜索SSE
	app.GET("/api/source/stat", UseRecovery(videoController.SourceStat))          // 源统计数据
	app.Any("/api/debug", UseRecovery(homeController.Debug))

	//app.GET("/api/debug/http-ctx", func(ctx *gin.Context) {
	//	ctx.JSON(http.StatusOK, model.NewSuccess(map[string]interface{}{
	//		"header": ctx.Request.Header,
	//	}))
	//})
	//app.GET("/api/debug/pprof", func(ctx *gin.Context) {
	//	ctx.JSON(http.StatusOK, model.NewSuccess(map[string]interface{}{
	//		"header": ctx.Request.Header,
	//	}))
	//})

	return app
}

func UseRecovery(h func(ctx *gin.Context)) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Println("[Err]", err)
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

func portUse(port int) int {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return portUse(port + 1)
	}
	defer func() { _ = listener.Close() }()
	return port
}

func newRouterWeb(app *gin.Engine) *gin.Engine {
	app.StaticFS("assets/", gin.Dir(filepath.Join(util.AppPath(), "dist/assets/"), true))
	app.NoRoute(func(ctx *gin.Context) {
		ctx.File("./dist/index.html")
	})
	return app
}

func serveHtml(port int) {
	time.Sleep(time.Second * 1 / 2)
	var osName = strings.ToLower(runtime.GOOS)
	switch osName {
	case "windows":
		cmd := exec.Command("cmd", "/c", "start", fmt.Sprintf("http://127.0.0.1:%d", port))
		_ = cmd.Run()
	case "darwin":
		cmd := exec.Command("open", fmt.Sprintf("http://127.0.0.1:%d", port))
		_ = cmd.Run()
	}
}
