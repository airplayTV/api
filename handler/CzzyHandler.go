package handler

import (
	"fmt"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/gin-gonic/gin"
)

type CzzyHandler struct {
	Handler
}

func (x CzzyHandler) Init() IVideo {
	x.httpClient = util.HttpClient{}
	return x
}

func (x CzzyHandler) Name() string {
	return "czzy"
}

func (x CzzyHandler) TagList() interface{} {
	var tags = make([]gin.H, 0)
	tags = append(tags, gin.H{"name": "dbtop250", "value": "dbtop250"})
	tags = append(tags, gin.H{"name": "movie_bt", "value": "movie_bt"})
	tags = append(tags, gin.H{"name": "gaofenyingshi", "value": "gaofenyingshi"})
	tags = append(tags, gin.H{"name": "zuixindianying", "value": "zuixindianying"})
	tags = append(tags, gin.H{"name": "gcj", "value": "gcj"})
	tags = append(tags, gin.H{"name": "meijutt", "value": "meijutt"})
	tags = append(tags, gin.H{"name": "hanjutv", "value": "hanjutv"})
	tags = append(tags, gin.H{"name": "fanju", "value": "fanju"})
	return model.NewSuccess(tags)
}

func (x CzzyHandler) VideoList(tag, page string) interface{} {
	return x._videoList(tag, page)
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

//

func (x CzzyHandler) _tagList() interface{} {
	return nil
}

func (x CzzyHandler) _videoList(tag, page string) interface{} {
	buff, err := x.httpClient.Get(fmt.Sprintf(czzyTagUrl, tag, x.parsePageNumber(page)))
	if err != nil {
		return model.NewError("获取数据失败：" + err.Error())
	}

	return model.NewSuccess(gin.H{"data": string(buff)})
}
