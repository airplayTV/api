package controller

import (
	"github.com/airplayTV/api/handler"
	"log"
	"time"
)

func init() {
	go func() {
		log.Println("[TimerStart]")
		var t = time.NewTicker(time.Second * 60 * 5)
		for range t.C {
			for _, h := range sourceMap {
				go holdCookie(h.Handler)
			}
		}
	}()
}

func holdCookie(h handler.IVideo) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("[HoldCookieError]", h.Name(), err)
		}
	}()

	if err := h.HoldCookie(); err != nil {
		log.Println("[HoldCookieError]", h.Name(), err.Error())
	} else {
		log.Println("[HoldCookieSuccess]", h.Name())
	}
}
