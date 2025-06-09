package controller

import (
	"github.com/lixiang4u/goWebsocket"
)

type WebsocketController struct {
}

func (x WebsocketController) Index(data goWebsocket.EventCtx) bool {

	return true
}
