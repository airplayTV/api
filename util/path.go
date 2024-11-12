package util

import (
	"io"
	"os"
	"path/filepath"
)

func AppPath() string {
	p, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return p
}

func ReadFile(filename string) []byte {
	fi, err := os.Open(filename)
	if err != nil {
		return nil
	}
	buff, err := io.ReadAll(fi)
	if err != nil {
		return nil
	}
	return buff
}
