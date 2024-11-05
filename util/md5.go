package util

import (
	"crypto/md5"
	"fmt"
	"io"
)

func StringMd5(s string) string {
	var md5Hash = md5.New()
	_, err := io.WriteString(md5Hash, s)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", md5Hash.Sum(nil))
}
