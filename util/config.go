package util

import (
	"fmt"
	"github.com/goccy/go-json"
	"path/filepath"
)

func SaveHttpHeader(source string, h map[string]string) error {
	var p = filepath.Join(AppPath(), "file", fmt.Sprintf("http_header_%s.json", StringMd5(source)[:8]))
	return WriteFile(p, ToBytes(h, true))
}

func LoadHttpHeader(source string) (h map[string]string, err error) {
	var p = filepath.Join(AppPath(), "file", fmt.Sprintf("http_header_%s.json", StringMd5(source)[:8]))
	err = json.Unmarshal(ReadFile(p), &h)
	return
}
