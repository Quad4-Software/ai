// SPDX-License-Identifier: 0BSD
//go:build !unix

package main

import "syscall"

// detachSysProcAttr is a no-op on platforms without session support.
func detachSysProcAttr() *syscall.SysProcAttr {
	return nil
}
