package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type HomeController struct {
}

func (x HomeController) Index(ctx *gin.Context) {
	ctx.String(http.StatusOK, "AirplayTV")
}
