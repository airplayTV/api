package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/goWebsocket"
	"github.com/mitchellh/mapstructure"
)

var AppSocket = goWebsocket.NewWebsocketManager(true)

type WebsocketController struct {
}

func (x WebsocketController) Index(data goWebsocket.EventCtx) bool {
	//log.Println("[Index]", goWebsocket.ToJson(data))
	return true
}

func (x WebsocketController) Connect(data goWebsocket.EventCtx) bool {
	AppSocket.Send(data.From, goWebsocket.EventCtx{
		Event: data.Event,
		Data:  gin.H{"code": 200, "msg": "socket已连接", "client_id": data.From},
	})
	return true
}

func (x WebsocketController) JoinGroup(data goWebsocket.EventCtx) bool {
	type Req struct {
		Group string `json:"group"`
	}
	var req Req
	_ = mapstructure.Decode(data.Data, &req)
	if len(req.Group) <= 0 {
		return false
	}
	AppSocket.JoinGroup(data.From, req.Group)
	AppSocket.Send(data.From, goWebsocket.EventCtx{
		Event: data.Event,
		Data:  gin.H{"code": 200, "msg": "已加入分组", "client_id": data.From, "group": req.Group},
	})
	return true
}

func (x WebsocketController) SendToGroup(data goWebsocket.EventCtx) bool {
	type Req struct {
		Event string `json:"event"`
		From  string `json:"from"`
		Group string `json:"group"`
	}
	var req Req
	_ = mapstructure.Decode(data.Data, &req)
	if len(req.Group) <= 0 {
		return false
	}
	AppSocket.SendToGroup(req.Group, data.Data)

	return true
}
