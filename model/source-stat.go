package model

type VideoResolution struct {
	Source string `json:"source"` // 视频源
	Name   string `json:"name"`   // 视频名
	Url    string `json:"url"`    // 视频地址
	TsUrl  string `json:"ts_url"` // 视频播放的TS地址
	Width  int    `json:"width"`  //视频 resolution 宽度
	Height int    `json:"height"` //视频 resolution 高度
	Time   string `json:"time"`   //测试时间
	Err    string `json:"err"`    // 错误原因
}
