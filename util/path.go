package util

import (
	"net/url"
)

func FillUrlHost(hostUrl, pathUrl string) string {
	u2, err := url.Parse(pathUrl)
	if err != nil {
		return pathUrl
	}
	if len(u2.Scheme) > 0 && len(u2.Host) > 0 {
		return pathUrl
	}
	u1, err := url.Parse(hostUrl)
	if err != nil {
		return pathUrl
	}
	if len(u1.Host) == 0 {
		return pathUrl
	}
	u2.Scheme = u1.Scheme
	u2.Host = u1.Host
	return u2.String()
}
