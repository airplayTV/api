package controller

import (
	"github.com/gorilla/websocket"
	"github.com/lixiang4u/goWebsocket"
)

type WebsocketController struct {
}

func (x WebsocketController) Index(clientId string, ws *websocket.Conn, messageType int, data goWebsocket.EventProtocol) bool {

	return true
}
