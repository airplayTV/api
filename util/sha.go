package util

import (
	"crypto/sha1"
	"fmt"
)

func Sha1Simple(data []byte) string {
	hash := sha1.Sum(data)
	return fmt.Sprintf("%x", hash)
}
