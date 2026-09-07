// SPDX-License-Identifier: 0BSD
// Package sysutil runs read-only local Reticulum utilities
// (rnstatus, rnpath, rnid, rnprobe) with strict argument validation,
// timeouts, and bounded output.
package sysutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	execTimeout = 20 * time.Second
	maxOutput   = 256 << 10
)

var (
	hashRe = regexp.MustCompile(`^[0-9a-fA-F]{32}$|^[0-9a-fA-F]{64}$`)
	nameRe = regexp.MustCompile(`^[a-zA-Z0-9._\- ]{1,128}$`)
	hopsRe = regexp.MustCompile(`^[0-9]{1,3}$`)
)

// Runner executes a validated command and returns combined output.
type Runner struct {
	// execFn is replaceable in tests.
	execFn func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func New() *Runner { return &Runner{execFn: run} }

func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("%s not found on PATH (install RNS utilities via pipx)", name)
	}
	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...) // #nosec G204 -- fixed argv, no shell, command allowlisted
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	out := buf.Bytes()
	if len(out) > maxOutput {
		out = append(out[:maxOutput], []byte("\n[output truncated]")...)
	}
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// Status runs rnstatus. all maps to -a (verbose stats).
func (r *Runner) Status(ctx context.Context, all bool) (string, error) {
	args := []string{}
	if all {
		args = append(args, "-a")
	}
	return r.exec(ctx, "rnstatus", args...)
}

// PathTable runs rnpath -t, optionally filtered by max hops.
func (r *Runner) PathTable(ctx context.Context, maxHops string) (string, error) {
	args := []string{"-t"}
	if maxHops != "" {
		if !hopsRe.MatchString(maxHops) {
			return "", fmt.Errorf("invalid max_hops %q", maxHops)
		}
		args = append(args, "-m", maxHops)
	}
	return r.exec(ctx, "rnpath", args...)
}

// PathLookup runs rnpath -w 15 <destination>.
func (r *Runner) PathLookup(ctx context.Context, destination string) (string, error) {
	if !hashRe.MatchString(destination) {
		return "", fmt.Errorf("destination must be a 32 or 64 char hex hash, got %q", destination)
	}
	return r.exec(ctx, "rnpath", "-w", "15", destination)
}

// DestinationHash runs rnid -i <identity> -H <aspects>.
func (r *Runner) DestinationHash(ctx context.Context, identity, aspects string) (string, error) {
	if !hashRe.MatchString(identity) {
		return "", fmt.Errorf("identity must be a 32 or 64 char hex hash, got %q", identity)
	}
	if !nameRe.MatchString(aspects) {
		return "", fmt.Errorf("invalid aspects %q", aspects)
	}
	return r.exec(ctx, "rnid", "-i", identity, "-H", aspects)
}

// Utilities is the allowlist for Help.
var Utilities = []string{
	"rnsd", "rnstatus", "rnpath", "rnprobe", "rnid",
	"rncp", "rnx", "rnsh", "rngit", "rnodeconf", "nomadnet",
}

// Help runs <utility> --help for any allowlisted RNS utility.
func (r *Runner) Help(ctx context.Context, utility string) (string, error) {
	if slices.Contains(Utilities, utility) {
		return r.exec(ctx, utility, "--help")
	}
	return "", fmt.Errorf("unknown or disallowed utility %q; allowed: %s", utility, strings.Join(Utilities, ", "))
}

// Probe runs rnprobe <app_name> <destination>.
func (r *Runner) Probe(ctx context.Context, appName, destination string) (string, error) {
	if !nameRe.MatchString(appName) {
		return "", fmt.Errorf("invalid app_name %q", appName)
	}
	if !hashRe.MatchString(destination) {
		return "", fmt.Errorf("destination must be a 32 or 64 char hex hash, got %q", destination)
	}
	return r.exec(ctx, "rnprobe", appName, destination)
}

func (r *Runner) exec(ctx context.Context, name string, args ...string) (string, error) {
	out, err := r.execFn(ctx, name, args...)
	s := string(bytes.TrimSpace(out))
	if s == "" {
		if err != nil {
			return "", err
		}
		return "(no output)", nil
	}
	return s, nil
}
