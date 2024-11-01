package model

type Control struct {
	Source   string      `json:"_source"`
	ClientId string      `json:"client_id"` // 要发送的目的客户端ID，不指定则发送到组
	Group    string      `json:"group"`     // 要发送的目组
	Event    string      `json:"event"`     // 具体遥控器的事件
	Value    interface{} `json:"value"`     //  事件附加的数据
}
