package handler

import "github.com/gin-gonic/gin"

type SubbHandler struct {
}

func (x SubbHandler) Name() string {
	return "subaibaiys"
}
func (x SubbHandler) TagList() gin.H {
	return gin.H{
		"name": x.Name(),
	}
}
func (x SubbHandler) VideoList(tag, page string) gin.H {
	return gin.H{}
}
func (x SubbHandler) Search(keyword, page string) gin.H {
	return gin.H{}
}
func (x SubbHandler) Detail(id string) gin.H {
	return gin.H{}
}
func (x SubbHandler) Source(pid, vid string) gin.H {
	return gin.H{}
}
