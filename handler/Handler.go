package handler

import (
	"github.com/airplayTV/api/util"
	"strconv"
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
