package model

type Source struct {
	Id     string `json:"id"`     // 播放ID，Vid下的播放源
	Vid    string `json:"vid"`    // 电影/电视/综艺视频ID
	Name   string `json:"name"`   // 名称
	Thumb  string `json:"thumb"`  // 缩略图
	Url    string `json:"url"`    // 播放地址
	Type   string `json:"type"`   // 视频类型，hls....
	Source string `json:"source"` // 源播放地址
}
