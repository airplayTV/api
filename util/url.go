package util

import (
	"fmt"
	"net/url"
)

func ParseUrlHost(tmpUrl string) (host string) {
	tmpUrl2, err := url.Parse(tmpUrl)
	if err != nil {
		return
	}
	if tmpUrl2.Host == "" {
		return
	}
	return fmt.Sprintf("%s://%s", tmpUrl2.Scheme, tmpUrl2.Host)
}

func Http2HttpsUrl(tmpUrl string) string {
	parsed, err := url.Parse(tmpUrl)
	if err != nil {
		return ""
	}
	parsed.Scheme = "https"
	return parsed.String()
}
