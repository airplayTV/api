package util

import (
	"io"
	"io/fs"
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

func WriteFile(filename string, data []byte) error {
	fi, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC, fs.ModePerm)
	if err != nil {
		return err
	}
	_, err = fi.Write(data)
	if err != nil {
		return err
	}
	return nil
}