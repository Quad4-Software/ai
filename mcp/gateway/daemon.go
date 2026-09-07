// SPDX-License-Identifier: 0BSD
// Shared-daemon transport for gateway: one long-lived process owns
// all child servers on a unix socket, and lightweight --attach clients
// bridge stdio to it so every MCP window shares one gateway.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Quad4-Software/ai/mcp/gateway/internal/mcp"
)

// defaultSocket returns the shared-daemon socket path, preferring the
// per-user runtime dir so the socket is not world-visible in /tmp.
func defaultSocket() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "gateway.sock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("gateway-%d.sock", os.Getuid()))
}

// runDaemon serves srv on a unix socket until SIGTERM/SIGINT. The
// socket is owner-only and removed on exit.
func runDaemon(ctx context.Context, sock string, srv *mcp.Server) error {
	if err := os.Remove(sock); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	l, err := net.Listen("unix", sock)
	if err != nil {
		return err
	}
	defer os.Remove(sock) // #nosec G104 -- best effort cleanup
	if err := os.Chmod(sock, 0o600); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(os.Stderr, "gateway: daemon listening on %s\n", sock)
	return srv.ServeListener(ctx, l)
}

// attach bridges this process's stdio to the daemon socket, starting
// the daemon first if it is not running.
func attach(sock string) error {
	conn, err := dialDaemon(sock)
	if err != nil {
		return err
	}
	defer conn.Close()
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(conn, os.Stdin)
		if uc, ok := conn.(*net.UnixConn); ok {
			uc.CloseWrite() // #nosec G104 -- best effort half-close
		}
		done <- err
	}()
	_, copyErr := io.Copy(os.Stdout, conn)
	select {
	case err := <-done:
		if copyErr != nil {
			return copyErr
		}
		return err
	default:
		return copyErr
	}
}

// dialDaemon connects to the socket, spawning the daemon detached and
// retrying briefly when it is not up yet.
func dialDaemon(sock string) (net.Conn, error) {
	conn, err := net.Dial("unix", sock)
	if err == nil {
		return conn, nil
	}
	dialErr := err
	if err := spawnDaemon(sock); err != nil {
		return nil, fmt.Errorf("dial %s: %w (daemon spawn: %v)", sock, dialErr, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.Dial("unix", sock)
		if err == nil {
			return conn, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("dial %s: daemon did not come up", sock)
}

// spawnDaemon starts this binary in --listen mode, detached from the
// client session so it outlives any single window.
func spawnDaemon(sock string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	log, err := os.OpenFile(sock+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304 -- path derived from configured socket
	if err != nil {
		return err
	}
	defer log.Close()
	cmd := exec.Command(exe, "--daemon", "--socket", sock) // #nosec G204 -- fixed argv, own binary
	cmd.Stdout, cmd.Stderr = log, log
	cmd.Stdin = nil
	cmd.SysProcAttr = detachSysProcAttr()
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release() // #nosec G104 -- intentional orphan
}
