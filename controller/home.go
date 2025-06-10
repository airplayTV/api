package controller

import (
	"github.com/airplayTV/api/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

type HomeController struct {
}

func (x HomeController) Index(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, model.NewSuccess(map[string]interface{}{
		"header": ctx.Request.Header,
	}))
}
