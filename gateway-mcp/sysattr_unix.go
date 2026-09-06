// SPDX-License-Identifier: 0BSD
//go:build unix

package main

import "syscall"

// detachSysProcAttr puts the daemon in its own session so it outlives
// the client that spawned it.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
