// SPDX-License-Identifier: 0BSD
// Anti-bot and WAF interstitial detection. classify() inspects a
// response and names the protection vendor and challenge style so the
// fetcher can pick a solver, or so the user gets an actionable error.
package fetch

import (
	"net/http"
	"strings"
)

// Protection describes an anti-bot interstitial found in a response.
type Protection struct {
	// Vendor names the product: cloudflare, anubis, ddos-guard,
	// sucuri, aws-waf, akamai, or generic.
	Vendor string `json:"vendor"`
	// Kind is managed-challenge, js-challenge, pow, or block. A block
	// is a deny rule, not a solvable challenge.
	Kind string `json:"kind"`
	// Detail is a short human-readable hint, for example the
	// Cloudflare error code.
	Detail string `json:"detail,omitempty"`
}

// sniffSize bounds how much of a body classify inspects.
const sniffSize = 128 << 10

// classify returns a zero Protection when the response looks like a
// normal page rather than an interstitial. Anubis challenge pages are
// served with HTTP 200, so body markers are checked before status.
func classify(status int, h http.Header, body []byte) Protection {
	sniff := body
	if len(sniff) > sniffSize {
		sniff = sniff[:sniffSize]
	}
	b := strings.ToLower(string(sniff))
	server := strings.ToLower(h.Get("Server"))
	cookies := strings.ToLower(strings.Join(h.Values("Set-Cookie"), ";"))

	// Anubis embeds the challenge as JSON in a script tag and serves
	// the challenge page with status 200, so it is checked first and
	// without a status gate.
	if strings.Contains(b, `id="anubis_challenge"`) ||
		strings.Contains(b, "/.within.website/x/cmd/anubis") ||
		strings.Contains(cookies, "techaro.lol-anubis") {
		return Protection{Vendor: "anubis", Kind: "pow"}
	}

	if strings.Contains(strings.ToLower(h.Get("cf-mitigated")), "challenge") {
		return Protection{Vendor: "cloudflare", Kind: "managed-challenge", Detail: "cf-mitigated: challenge"}
	}
	cfBody := strings.Contains(b, "challenges.cloudflare.com") ||
		strings.Contains(b, "just a moment") ||
		strings.Contains(b, "cf-chl") ||
		strings.Contains(b, "cf-challenge")
	if strings.Contains(server, "cloudflare") && cfBody {
		kind := "js-challenge"
		if code := cfErrorCode(b); code != "" {
			return Protection{Vendor: "cloudflare", Kind: "block", Detail: "error code " + code}
		}
		return Protection{Vendor: "cloudflare", Kind: kind}
	}
	// Cloudflare deny pages can lack the challenges script but carry
	// the error code marker.
	if strings.Contains(server, "cloudflare") {
		if code := cfErrorCode(b); code != "" {
			return Protection{Vendor: "cloudflare", Kind: "block", Detail: "error code " + code}
		}
	}

	if strings.Contains(server, "ddos-guard") ||
		strings.Contains(b, "ddos-guard") ||
		strings.Contains(cookies, "__ddg") {
		return Protection{Vendor: "ddos-guard", Kind: "js-challenge"}
	}
	if h.Get("x-sucuri-id") != "" || h.Get("x-sucuri-block") != "" ||
		(strings.Contains(b, "sucuri") && strings.Contains(b, "access denied")) {
		return Protection{Vendor: "sucuri", Kind: "block"}
	}
	if h.Get("x-amzn-waf-action") != "" ||
		strings.Contains(b, "aws-waf-token") || strings.Contains(b, "awswaf") {
		return Protection{Vendor: "aws-waf", Kind: "js-challenge"}
	}
	if strings.Contains(cookies, "_abck") || strings.Contains(cookies, "bm_sz") ||
		(strings.Contains(b, "akamai") && strings.Contains(b, "reference #")) {
		return Protection{Vendor: "akamai", Kind: "js-challenge"}
	}

	// Weak generic signal: an error status plus challenge vocabulary
	// in a small HTML body.
	if status == http.StatusUnauthorized || status == http.StatusForbidden ||
		status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable {
		for _, hint := range []string{
			"checking your browser", "verify you are human", "are you a robot",
			"challenge-platform", "pow_challenge", "proof of work",
			"press & hold", "press and hold",
		} {
			if strings.Contains(b, hint) {
				return Protection{Vendor: "generic", Kind: "js-challenge", Detail: hint}
			}
		}
	}
	return Protection{}
}

// cfErrorCode pulls the numeric Cloudflare error (1020, 1010, ...)
// out of a deny page body.
func cfErrorCode(b string) string {
	_, after, ok := strings.Cut(b, "error code:")
	if !ok {
		return ""
	}
	s := after
	s = strings.TrimLeft(s, " ")
	var code strings.Builder
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		code.WriteRune(r)
	}
	return code.String()
}
