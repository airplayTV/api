package util

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"net/url"
)

func EncodeComponentUrl(prettyUrl string) string {
	return base64.StdEncoding.EncodeToString([]byte(
		ToString(gin.H{"url": url.QueryEscape(prettyUrl)}),
	))
}

func DecodeComponentUrl(encodedUrl string) string {
	buff, err := base64.StdEncoding.DecodeString(encodedUrl)
	if err != nil {
		return ""
	}
	var decodedJson = gjson.ParseBytes(buff)
	unescape, err := url.QueryUnescape(decodedJson.Get("url").String())
	if err != nil {
		return ""
	}
	return unescape
}

func GBKToUTF8(buff []byte) ([]byte, error) {
	bytes, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), buff)
	return bytes, err
}
