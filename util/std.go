package util

import (
	"log"
	"os"
)

func ExitMsg(msg string) {
	log.Println(msg)
	_, _ = os.Stdin.Read(make([]byte, 1))
	os.Exit(1)
}
