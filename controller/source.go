package controller

import (
	"github.com/airplayTV/api/handler"
	"github.com/airplayTV/api/model"
	"github.com/gin-gonic/gin"
	"slices"
)

type SourceHandler struct {
	Sort    int
	Handler handler.IVideo
}

var sourceMap map[string]SourceHandler

var sourceModeListMap = make(map[string]map[string]SourceHandler)

// 不缓存播放数据的源
var noCacheSourceList = []string{
	handler.SubbHandler{}.Name(),     // wangchuanxin.top 第三次请求缓存数据就不能播放
	handler.NaifeiMeHandler{}.Name(), // ki-mi.vip解析压根不能缓存
}

func init() {
	sourceMap = map[string]SourceHandler{
		handler.CzzyHandler{}.Name(): {Sort: 1, Handler: handler.CzzyHandler{}.Init(model.CmsZyOption{
			Id:         "czzy",
			Name:       handler.CzzyHandler{}.Name(),
			Searchable: true,
		})},
		//handler.SubbHandler{}.Name():     {Sort: 2, Handler: handler.SubbHandler{}.Init(nil)}, // 限制国内IP访问
		//handler.YingshiHandler{}.Name():  {Sort: 3, Handler: handler.YingshiHandler{}.Init(nil)}, // api挂了
		handler.MaYiHandler{}.Name(): {Sort: 4, Handler: handler.MaYiHandler{}.Init(model.CmsZyOption{
			Id:         "mayi",
			Name:       handler.MaYiHandler{}.Name(),
			Searchable: true,
		})},
		handler.NaifeiMeHandler{}.Name(): {Sort: 5, Handler: handler.NaifeiMeHandler{}.Init(model.CmsZyOption{
			Id:         "naifeigc",
			Name:       handler.NaifeiMeHandler{}.Name(),
			Searchable: true,
		})},
		//handler.MeiYiDaHandler{}.Name():  {Sort: 6, Handler: handler.MeiYiDaHandler{}.Init(nil)}, // 不可达
		//handler.Huawei8Handler{}.Name():  {Sort: 7, Handler: handler.Huawei8Handler{}.Init()},
		//handler.Huawei8ApiHandler{}.Name(): {Sort: 8, Handler: handler.Huawei8ApiHandler{}.Init()},
		//handler.BfzyHandler{}.Name():       {Sort: 9, Handler: handler.BfzyHandler{}.Init()},
		//handler.KczyHandler{}.Name():       {Sort: 10, Handler: handler.KczyHandler{}.Init()},
	}

	var idx = 20
	for _, tmpConfig := range cmsApiConfig {
		if tmpConfig.Disable {
			continue
		}
		idx += 1
		var h = handler.CmsZyHandler{}.Init(model.CmsZyOption{
			Name:       tmpConfig.Name,
			Api:        tmpConfig.Api,
			Id:         tmpConfig.Id,
			Disable:    tmpConfig.Disable,
			Searchable: tmpConfig.Searchable,
		})
		sourceMap[tmpConfig.Name] = struct {
			Sort    int
			Handler handler.IVideo
		}{Sort: idx, Handler: h}
	}

	// 根据 mode 分类
	for tmpMode, tmpList := range sourceModeMap {
		tmpV, ok := sourceModeListMap[tmpMode]
		if !ok {
			sourceModeListMap[tmpMode] = make(map[string]SourceHandler)
			tmpV = sourceModeListMap[tmpMode]
		}
		for tmpName, tmpValue := range sourceMap {
			if slices.Contains(tmpList, tmpName) {
				tmpV[tmpName] = tmpValue
			}
		}
		sourceModeListMap[tmpMode] = tmpV
	}
}

func (x VideoController) getSourceMap(ctx *gin.Context) map[string]SourceHandler {
	var mode = ctx.GetHeader("x-source-mode")
	if len(mode) == 0 {
		mode = ctx.Query("_mode")
	}
	if v, ok := sourceModeListMap[mode]; ok {
		return v
	}
	if mode == "aptv-all" {
		return sourceMap
	}
	return sourceModeListMap["default"]
}
