//go:build linux
// +build linux

package util

func WindowsAdmin() bool {
	return false
}

func UpdateHosts(line string) error {
	return nil
}
