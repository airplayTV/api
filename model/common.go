package model

//import "github.com/airplayTV/api/handler"

var apiHost = "https://airplay-api.artools.cc"

var M3u8ProxyHosts = []string{
	"yundunm.nowm3.xyz:2083",
}

type SourceHandler struct {
	Sort    int
	Handler IVideo
}

var appSourceMap map[string]SourceHandler

func AppSourceMap(value ...map[string]SourceHandler) map[string]SourceHandler {
	if len(value) >= 1 {
		appSourceMap = value[0]
	}
	return appSourceMap
}

func ApiHost() string {
	return apiHost
}
