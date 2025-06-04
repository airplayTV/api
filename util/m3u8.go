package util

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/grafov/m3u8"
	"golang.org/x/exp/slices"
	"net/url"
	"path"
	"strings"
)

// FormatM3u8Url m3u8地址格式化（修正地址问题）
func FormatM3u8Url(data []byte, sourceUrl string) ([]byte, error) {
	host := ParseUrlHost(sourceUrl)
	if host == "" {
		return nil, errors.New("sourceUrl地址错误")
	}
	playList, listType, err := m3u8.DecodeFrom(bytes.NewBuffer(data), true)
	if err != nil {
		return nil, err
	}
	switch listType {
	case m3u8.MEDIA:
		mediapl := playList.(*m3u8.MediaPlaylist)
		if mediapl.Key != nil {
			mediapl.Key.URI = fixUrlHost(mediapl.Key.URI, sourceUrl)
		}
		for idx, val := range mediapl.Segments {
			if val == nil {
				continue
			}
			mediapl.Segments[idx].URI = fixUrlHost(mediapl.Segments[idx].URI, sourceUrl)
			// 导致播放文件中出现两次加密数据播放器解析失败
			val.Key = nil
		}
	case m3u8.MASTER:
		masterpl := playList.(*m3u8.MasterPlaylist)
		for idx, val := range masterpl.Variants {
			if val == nil {
				continue
			}
			masterpl.Variants[idx].URI = fixUrlHost(masterpl.Variants[idx].URI, sourceUrl)
		}
	}
	return playList.Encode().Bytes(), nil
}

func fixUrlHost(tmpUrl, sourceUrl string) string {
	// 如果地址空则返回
	if len(tmpUrl) == 0 {
		return ""
	}
	// 如果带"/"，则直接拼接返回
	if strings.HasPrefix(tmpUrl, "/") {
		return fmt.Sprintf("%s/%s", ParseUrlHost(sourceUrl), strings.TrimLeft(tmpUrl, "/"))
	}
	// 如果地址完整则返回
	parsedUrl, err := url.Parse(tmpUrl)
	if err == nil && slices.Contains([]string{"http", "https"}, parsedUrl.Scheme) {
		return tmpUrl
	}
	parsedUrl, err = url.Parse(sourceUrl)
	if err != nil {
		return "" // 这是个异常，源地址异常
	}
	//不带"/"，则需要从 sourceUrl 开始拼接
	var p = strings.TrimLeft(path.Join(path.Dir(parsedUrl.Path), tmpUrl), "/")

	return fmt.Sprintf("%s://%s/%s", parsedUrl.Scheme, parsedUrl.Host, p)
}

func ParseM3u8Url(data []byte) (retUrlList []string, err error) {
	retUrlList = make([]string, 0)
	playList, listType, err := m3u8.DecodeFrom(bytes.NewBuffer(data), true)
	if err != nil {
		return nil, err
	}
	switch listType {
	case m3u8.MEDIA:
		mediapl := playList.(*m3u8.MediaPlaylist)
		if mediapl.Key != nil {
			retUrlList = append(retUrlList, mediapl.Key.URI)
		}
		for idx, val := range mediapl.Segments {
			if val == nil {
				continue
			}
			retUrlList = append(retUrlList, mediapl.Segments[idx].URI)
		}
	case m3u8.MASTER:
		masterpl := playList.(*m3u8.MasterPlaylist)
		for idx, val := range masterpl.Variants {
			if val == nil {
				continue
			}
			retUrlList = append(retUrlList, masterpl.Variants[idx].URI)
		}
	}
	return retUrlList, nil
}
