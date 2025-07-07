package model

type VideoResolution struct {
	Id      int    `json:"id"`
	Date    int    `json:"date"`
	Source  string `json:"source"`  // 视频源
	Name    string `json:"name"`    // 视频名
	Vid     string `json:"vid"`     // 视频ID
	Pid     string `json:"pid"`     // 播放ID
	Url     string `json:"url"`     // 视频地址
	TsUrl   string `json:"ts_url"`  // 视频播放的TS地址
	Width   int    `json:"width"`   //视频 resolution 宽度
	Height  int    `json:"height"`  //视频 resolution 高度
	Time    string `json:"time"`    //测试时间
	Latency int64  `json:"latency"` //测试视频解析所用时间/不计算api请求（毫秒）
	Err     string `json:"err"`     // 错误原因
}

func (VideoResolution) TableName() string {
	return "source_stat"
}

func (x VideoResolution) Save(m VideoResolution) error {
	return DB().Table(x.TableName()).Create(&m).Error
}

func (x VideoResolution) SaveAll(m []VideoResolution) error {
	return DB().Table(x.TableName()).CreateInBatches(&m, 20).Error
}
