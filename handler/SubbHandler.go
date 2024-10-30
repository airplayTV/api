package handler

import (
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
)

type SubbHandler struct {
	Handler
}

func (x SubbHandler) Init() IVideo {
	x.httpClient = util.HttpClient{}
	return x
}

func (x SubbHandler) Name() string {
	return "subaibaiys"
}

func (x SubbHandler) TagList() interface{} {
	return gin.H{
		"name": x.Name(),
	}
}

func (x SubbHandler) VideoList(tag, page string) interface{} {
	return gin.H{}
}

func (x SubbHandler) Search(keyword, page string) interface{} {
	return gin.H{}
}

func (x SubbHandler) Detail(id string) interface{} {
	return gin.H{}
}

func (x SubbHandler) Source(pid, vid string) interface{} {
	return gin.H{}
}

func (x SubbHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}
