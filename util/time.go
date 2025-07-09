package util

import (
	"github.com/spf13/cast"
	"time"
)

var loc *time.Location

func TimeLocation() *time.Location {
	if loc == nil {
		loc = time.Now().Location()
	}
	return loc
}

func NowTime(layout string) string {
	// layout 2006-01-02 15:04:05
	return time.Now().Format(layout)
}

func NowUnixTime() int64 {
	return time.Now().Unix()
}

func NowUnixMilliTime() int64 {
	return time.Now().UnixMilli()
}

func NowUnixNanoTime() int64 {
	return time.Now().UnixNano()
}

func FormatDateTime(timestamp int64, layout ...string) string {
	// layout 2006-01-02 15:04:05
	if len(layout) <= 0 {
		return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
	}
	return time.Unix(timestamp, 0).Format(layout[0])
}

func TodayLeftSeconds() int {
	now := time.Now()
	// 基于当前时区构造下一时间 精确到纳秒级
	endOfDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()).Add(-time.Nanosecond)
	// 计算时间差并转秒
	return int(endOfDay.Sub(now).Seconds())
}

// NowDateEndUnixTime 今天结束时的 unixtime 时间
func NowDateEndUnixTime() int64 {
	var now = time.Now()
	// 基于当前时区构造下一时间 精确到纳秒级
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()).Add(-time.Nanosecond).Unix()
}

// NowDateStartUnixTime 今天开始时的 unixtime 时间
func NowDateStartUnixTime() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
}

func NowDateNumber(days ...int) int {
	if len(days) > 0 {
		return cast.ToInt(time.Now().AddDate(0, 0, days[0]).Format("20060102"))
	}
	return cast.ToInt(time.Now().Format("20060102"))
}

func NowMonthStartUnixTime() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Unix()
}
