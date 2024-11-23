package controller

import (
	"github.com/airplayTV/api/handler"
	"github.com/airplayTV/api/model"
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
	var resp interface{}
	switch h.(type) {
	case handler.CzzyHandler:
		resp = h.(handler.CzzyHandler).Search("我的", "1")
	}

	switch resp.(type) {
	case model.Success:
		log.Println("[HoldCookieSuccess]", h.Name(), resp.(model.Success).Code)
		break
	case model.Error:
		log.Println("[HoldCookieError]", h.Name(), resp.(model.Error))
		break
	}

}
