package handler

import "github.com/gin-gonic/gin"

type CzzyHandler struct {
}

func (x CzzyHandler) Name() string {
	return "czzy"
}
func (x CzzyHandler) TagList() gin.H {
	return gin.H{
		"name": x.Name(),
	}
}
func (x CzzyHandler) VideoList(tag, page string) gin.H {
	return gin.H{}
}
func (x CzzyHandler) Search(keyword, page string) gin.H {
	return gin.H{}
}
func (x CzzyHandler) Detail(id string) gin.H {
	return gin.H{}
}
func (x CzzyHandler) Source(pid, vid string) gin.H {
	return gin.H{}
}
