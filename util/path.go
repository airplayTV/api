package util

import (
	"os"
	"path/filepath"
)

func AppPath() string {
	p, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return p
}
