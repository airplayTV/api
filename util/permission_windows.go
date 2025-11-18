//go:build windows

package util

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

type TokenElevation struct {
	TokenIsElevated uint32
}

func WindowsAdmin() bool {
	var tokenHandle syscall.Token
	currentProcess, _ := syscall.GetCurrentProcess()

	err := syscall.OpenProcessToken(currentProcess, syscall.TOKEN_QUERY, &tokenHandle)
	if err != nil {
		return false
	}
	defer func() { _ = tokenHandle.Close() }()

	var tokenElevation TokenElevation
	var returnedLen uint32

	err = syscall.GetTokenInformation(
		tokenHandle,
		syscall.TokenElevation,
		(*byte)(unsafe.Pointer(&tokenElevation)),
		uint32(unsafe.Sizeof(tokenElevation)),
		&returnedLen,
	)

	return err == nil && tokenElevation.TokenIsElevated != 0
}

func UpdateHosts(line string) error {
	// line ---> 127.0.0.1	www.google.com
	var hostsFile = "C:\\Windows\\System32\\drivers\\etc\\hosts"
	var hostsBuf = ReadFile(hostsFile)
	if hostsBuf == nil {
		return errors.New("读取hosts文件失败")
	}
	var hostsContent = string(hostsBuf)
	if !strings.Contains(hostsContent, line) {
		hostsContent = strings.TrimSpace(hostsContent)
		hostsContent = fmt.Sprintf("%s\r\n%s", hostsContent, line)
	}
	if err := WriteFile(hostsFile, []byte(hostsContent)); err != nil {
		return err // 可能需要管理员权限
	}
	return nil
}
