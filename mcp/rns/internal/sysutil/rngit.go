// SPDX-License-Identifier: 0BSD
// Package sysutil runs read-only local Reticulum utilities
// (rnstatus, rnpath, rnid, rnprobe) with strict argument validation,
// timeouts, and bounded output.
package sysutil

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var (
	rngitSubRe = regexp.MustCompile(`^[a-z]{2,20}$`)
	rnsURLRe   = regexp.MustCompile(`^rns://[0-9a-fA-F]{32}(?:/[A-Za-z0-9._\-]{1,64}){1,3}$`)
	// safeArgRe matches RNS URLs, simple local paths/identifiers, and long flags.
	safeArgRe = regexp.MustCompile(`^[A-Za-z0-9._/:@\-]{1,512}$`)
)

var rngitSubs = []string{
	"create", "release", "fork", "mirror", "sync", "perms",
	"work", "info", "ls", "log", "show", "verify", "install",
}

var boolFlags = []string{
	"--bare", "--public", "--private", "--wheel", "--no-wheel",
	"--json", "--force", "--all", "--tags", "--user", "--global",
}

// Rngit runs a validated rngit subcommand with safe arguments.
// It accepts one RNS URL and a small set of extra flags or simple targets.
func (r *Runner) Rngit(ctx context.Context, sub, urlArg string, extra []string) (string, error) {
	sub = strings.ToLower(strings.TrimSpace(sub))
	if !rngitSubRe.MatchString(sub) || !slices.Contains(rngitSubs, sub) {
		return "", fmt.Errorf("invalid or disallowed rngit subcommand %q; allowed: %s", sub, strings.Join(rngitSubs, ", "))
	}
	args := []string{sub}
	if urlArg != "" {
		if !rnsURLRe.MatchString(urlArg) {
			return "", fmt.Errorf("invalid rns_url %q; expected rns://<32 or 64 hex hash>/<group>/<repo>", urlArg)
		}
		args = append(args, urlArg)
	}
	for _, e := range extra {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.HasPrefix(e, "-") {
			if !slices.Contains(boolFlags, e) {
				return "", fmt.Errorf("disallowed flag %q; allowed: %s", e, strings.Join(boolFlags, ", "))
			}
			args = append(args, e)
			continue
		}
		if !safeArgRe.MatchString(e) || strings.Contains(e, "..") {
			return "", fmt.Errorf("disallowed argument %q", e)
		}
		args = append(args, e)
	}
	return r.exec(ctx, "rngit", args...)
}
