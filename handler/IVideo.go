package handler

import "github.com/airplayTV/api/model"

type IVideo interface {
	Init(options interface{}) IVideo
	Name() string
	Option() model.CmsZyOption
	TagList() interface{}
	VideoList(tag, page string) interface{}
	Search(keyword, page string) interface{}
	Detail(id string) interface{}
	Source(pid, vid string) interface{}
	Airplay(pid, vid string) interface{}
	UpdateHeader(h map[string]string) error
	HoldCookie() error
}
