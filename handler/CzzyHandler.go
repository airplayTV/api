package handler

import (
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
)

type CzzyHandler struct {
	httpClient util.HttpClient
}

func (x CzzyHandler) Init() IVideo {
	x.httpClient = util.HttpClient{}
	return x
}

func (x CzzyHandler) Name() string {
	return "czzy"
}

func (x CzzyHandler) TagList() interface{} {
	return model.NewSuccess(gin.H{
		"name": x.Name(),
		"tags": x.Name(),
	})
}

func (x CzzyHandler) VideoList(tag, page string) interface{} {
	return gin.H{}
}

func (x CzzyHandler) Search(keyword, page string) interface{} {
	return gin.H{}
}

func (x CzzyHandler) Detail(id string) interface{} {
	return gin.H{}
}

func (x CzzyHandler) Source(pid, vid string) interface{} {
	return gin.H{}
}

func (x CzzyHandler) Airplay(pid, vid string) interface{} {
	return gin.H{}
}
