//go:build windows

package cli

import "syscall"

const (
	detachedProcess    = 0x00000008
	createNewProcGroup = 0x00000200
)

func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: detachedProcess | createNewProcGroup}
}
