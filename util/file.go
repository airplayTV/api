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

func WriteFile(filename string, data []byte) error {
	fi, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = fi.Close() }()
	_, err = fi.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func MkdirAll(filename string) error {
	return os.MkdirAll(filepath.Dir(filename), 0644)
}
