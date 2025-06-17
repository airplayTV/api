package model

//import "github.com/airplayTV/api/handler"

var M3u8ProxyHosts = []string{
	"yundunm.nowm3.xyz:2083",
}

type IVideo interface {
	Init(options interface{}) IVideo
	Name() string
	Option() CmsZyOption
	TagList() interface{}
	VideoList(tag, page string) interface{}
	Search(keyword, page string) interface{}
	Detail(id string) interface{}
	Source(pid, vid string) interface{}
	Airplay(pid, vid string) interface{}
	UpdateHeader(h map[string]string) error
	HoldCookie() error
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
