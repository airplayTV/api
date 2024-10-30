package handler

import "github.com/gin-gonic/gin"

type IVideo interface {
	Name() string
	TagList() gin.H
	VideoList(tag, page string) gin.H
	Search(keyword, page string) gin.H
	Detail(id string) gin.H
	Source(pid, vid string) gin.H
}
