package handler

type IVideo interface {
	Init() IVideo
	Name() string
	TagList() interface{}
	VideoList(tag, page string) interface{}
	Search(keyword, page string) interface{}
	Detail(id string) interface{}
	Source(pid, vid string) interface{}
	Airplay(pid, vid string) interface{}
	UpdateHeader(h map[string]string) error
	HoldCookie() error
}
