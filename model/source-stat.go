package model

type VideoResolution struct {
	Source string // 视频源
	Name   string // 视频名
	Url    string // 视频地址
	TsUrl  string // 视频播放的TS地址
	Width  int    //视频 resolution 宽度
	Height int    //视频 resolution 高度
	Time   string //测试时间
	Err    string // 错误原因
}
