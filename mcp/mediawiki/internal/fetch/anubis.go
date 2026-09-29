// SPDX-License-Identifier: 0BSD
// Native solver for Anubis challenges. The challenge page embeds the
// puzzle as a JSON script tag; supported methods are the sha256
// proof-of-work (fast and legacy slow) and metarefresh (timed
// redirect). Interactive methods such as preact are left to a remote
// browser solver.
package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// anubisAPI is the fixed Anubis API path suffix under the base prefix.
const anubisAPI = "/.within.website/x/cmd/anubis/api/"

// anubisCookie is the auth cookie Anubis issues on a passed challenge.
const anubisCookie = "techaro.lol-anubis"

var (
	anubisChallengeRe = regexp.MustCompile(`(?s)<script[^>]*id="anubis_challenge"[^>]*>(.*?)</script>`)
	anubisPrefixRe    = regexp.MustCompile(`(?s)<script[^>]*id="anubis_base_prefix"[^>]*>(.*?)</script>`)
)

// errUnsolvable marks challenge methods a native solver cannot do.
var errUnsolvable = errors.New("anubis method needs a browser")

// anubisPage is the anubis_challenge script JSON shape.
type anubisPage struct {
	Rules struct {
		Algorithm  string `json:"algorithm"`
		Difficulty int    `json:"difficulty"`
	} `json:"rules"`
	Challenge struct {
		ID         string    `json:"id"`
		Method     string    `json:"method"`
		RandomData string    `json:"randomData"`
		IssuedAt   time.Time `json:"issuedAt"`
	} `json:"challenge"`
}

// solveAnubis extracts the challenge from an interstitial page, solves
// it, and submits pass-challenge so the auth cookie lands in the jar.
// It returns the algorithm name for Via reporting.
func (f *Fetcher) solveAnubis(ctx context.Context, page *Result, origURL string) (string, error) {
	var ap anubisPage
	m := anubisChallengeRe.FindSubmatch(page.Body)
	if m == nil {
		return "", fmt.Errorf("anubis page without anubis_challenge JSON")
	}
	if err := json.Unmarshal(m[1], &ap); err != nil {
		return "", fmt.Errorf("anubis challenge JSON: %w", err)
	}
	if ap.Challenge.ID == "" || ap.Challenge.RandomData == "" {
		return "", fmt.Errorf("anubis challenge missing id or randomData")
	}
	algo := ap.Rules.Algorithm
	if algo == "" {
		algo = ap.Challenge.Method
	}

	q := url.Values{}
	q.Set("id", ap.Challenge.ID)
	q.Set("redir", origURL)
	t0 := time.Now()
	switch algo {
	case "fast", "slow":
		nonce, sum, err := solvePoW(ctx, ap.Challenge.RandomData, ap.Rules.Difficulty)
		if err != nil {
			return "", err
		}
		q.Set("response", sum)
		q.Set("nonce", strconv.FormatUint(nonce, 10))
		q.Set("elapsedTime", strconv.FormatFloat(float64(time.Since(t0).Milliseconds()), 'f', -1, 64))
	case "metarefresh":
		// The server insists the client waited IssuedAt +
		// difficulty*800ms before passing.
		wait := time.Until(ap.Challenge.IssuedAt.Add(time.Duration(ap.Rules.Difficulty) * 800 * time.Millisecond))
		if wait > 0 {
			t := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				t.Stop()
				return "", ctx.Err()
			case <-t.C:
			}
		}
		q.Set("challenge", ap.Challenge.RandomData)
	case "":
		return "", fmt.Errorf("%w: none declared", errUnsolvable)
	default:
		return "", fmt.Errorf("%w: %s", errUnsolvable, algo)
	}

	u, err := url.Parse(origURL)
	if err != nil {
		return "", err
	}
	pass := *u
	pass.Path = anubisPrefix(page.Body) + anubisAPI + "pass-challenge"
	pass.RawQuery = q.Encode()
	pass.Fragment = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pass.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", f.userAgent())
	resp, err := f.hcNoRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("anubis pass-challenge: %w", f.cleanErr(err))
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	for _, c := range f.jar.Cookies(u) {
		if c.Name == anubisCookie {
			return algo, nil
		}
	}
	return "", fmt.Errorf("anubis pass-challenge returned %s without issuing %s", resp.Status, anubisCookie)
}

// anubisPrefix reads the anubis_base_prefix JSON script, which holds a
// quoted string for deployments mounted under a subpath.
func anubisPrefix(body []byte) string {
	m := anubisPrefixRe.FindSubmatch(body)
	if m == nil {
		return ""
	}
	var s string
	if json.Unmarshal(m[1], &s) != nil {
		return ""
	}
	return strings.TrimSuffix(s, "/")
}

// solvePoW finds a nonce where sha256(randomData + nonce) hex-encoded
// has difficulty leading zero digits, matching the Anubis fast
// algorithm. Work is spread over a few goroutines and bounded by ctx.
func solvePoW(ctx context.Context, data string, difficulty int) (uint64, string, error) {
	if difficulty < 0 || difficulty > 64 {
		return 0, "", fmt.Errorf("anubis difficulty %d out of range", difficulty)
	}
	workers := min(runtime.NumCPU(), 8)
	var found atomic.Uint64
	var done atomic.Bool
	var sum [sha256.Size]byte // winner writes before setting done

	var wg atomic.Int64
	wg.Add(int64(workers))
	fin := make(chan struct{})
	for w := range workers {
		go func(step uint64) {
			defer func() {
				if wg.Add(-1) == 0 {
					close(fin)
				}
			}()
			for nonce := step; !done.Load(); nonce += uint64(workers) {
				if nonce&4095 == 0 && ctx.Err() != nil {
					return
				}
				h := sha256.Sum256([]byte(data + strconv.FormatUint(nonce, 10)))
				if zeroPrefix(h[:], difficulty) {
					if done.CompareAndSwap(false, true) {
						sum = h
						found.Store(nonce)
					}
					return
				}
			}
		}(uint64(w)) // #nosec G115 -- w is a loop index in [0, 8)
	}
	select {
	case <-fin:
	case <-ctx.Done():
		done.Store(true)
		<-fin
		return 0, "", fmt.Errorf("anubis pow difficulty %d: %w", difficulty, ctx.Err())
	}
	return found.Load(), hex.EncodeToString(sum[:]), nil
}

// zeroPrefix reports whether sum has n leading zero hex nibbles.
func zeroPrefix(sum []byte, n int) bool {
	for i := 0; i < n/2; i++ {
		if sum[i] != 0 {
			return false
		}
	}
	return n%2 == 0 || sum[n/2]>>4 == 0
}
