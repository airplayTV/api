package handler

import (
	"github.com/airplayTV/api/util"
	"regexp"
	"strconv"
	"strings"
)

type Handler struct {
	httpClient util.HttpClient
}

func (x Handler) parsePageNumber(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 1
	}
	if n <= 0 {
		return 1
	}
	return n
}

func (x Handler) simpleRegEx(plainText, regex string) string {
	//regEx := regexp.MustCompile(`(\d+)`)
	regEx := regexp.MustCompile(regex)
	tmpList := regEx.FindStringSubmatch(plainText)
	if len(tmpList) < 2 {
		return ""
	}
	return tmpList[1]
}

func (x Handler) parseVideoType(sourceUrl string) string {
	if strings.Contains(sourceUrl, ".m3u8") {
		return sourceTypeHLS
	}

	return ""
}
