package task

import (
	"fmt"
	"github.com/airplayTV/api/model"
	"log"
	"runtime/debug"
	"time"
)

type HoldCookie struct {
}

func NewHoldCookie() *HoldCookie {
	return &HoldCookie{}
}

func (x HoldCookie) Run() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("[taskHandler.recover0]", err)
			log.Println("[taskHandler.recover0]", string(debug.Stack()))
		}
	}()

	var ticker = time.NewTicker(time.Minute * 5)
	x.taskHandler()
	for {
		select {
		case <-ticker.C:
			log.Println("[NewHoldCookie.Run]")
			x.taskHandler()
		}
	}

}

func (x HoldCookie) taskHandler() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("[taskHandler.recover2]", err)
			log.Println("[taskHandler.recover2]", string(debug.Stack()))
		}
	}()

	log.Println("[开始执行任务]")

	for _, h := range model.AppSourceMap() {
		if err := h.Handler.HoldCookie(); err != nil {
			log.Println("[HoldCookieError]", h.Handler.Name(), err.Error())
		} else {
			log.Println("[HoldCookieOk]", h.Handler.Name())
		}
	}

	log.Println(fmt.Sprintf("[完成任务] ok "))
}
