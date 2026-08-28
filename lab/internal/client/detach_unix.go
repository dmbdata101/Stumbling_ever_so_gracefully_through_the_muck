//go:build unix

package client

import "syscall"

// detach puts labd in its own session so it survives the terminal that
// happened to start it.
func detach() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }
