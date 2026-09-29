// SPDX-License-Identifier: 0BSD
package fetch

import (
	"net/http"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name   string
		status int
		header http.Header
		body   string
		vendor string
		kind   string
	}{
		{
			name:   "cloudflare managed challenge",
			status: 403,
			header: http.Header{
				"Server":       {"cloudflare"},
				"Cf-Mitigated": {"challenge"},
			},
			body:   `<html><title>Just a moment...</title>challenges.cloudflare.com</html>`,
			vendor: "cloudflare",
			kind:   "managed-challenge",
		},
		{
			name:   "cloudflare js challenge by body",
			status: 403,
			header: http.Header{"Server": {"cloudflare"}},
			body:   `<title>Just a moment...</title>`,
			vendor: "cloudflare",
			kind:   "js-challenge",
		},
		{
			name:   "cloudflare waf block",
			status: 403,
			header: http.Header{"Server": {"cloudflare"}},
			body:   `<html>Access denied error code: 1020</html>`,
			vendor: "cloudflare",
			kind:   "block",
		},
		{
			name:   "anubis challenge page is status 200",
			status: 200,
			header: http.Header{},
			body: `<html><script id="anubis_challenge" type="application/json">` +
				`{"challenge":{"id":"x","randomData":"y"},"rules":{"algorithm":"fast","difficulty":4}}` +
				`</script>Making sure you are not a bot</html>`,
			vendor: "anubis",
			kind:   "pow",
		},
		{
			name:   "ddos-guard by server header",
			status: 403,
			header: http.Header{"Server": {"ddos-guard"}},
			body:   `<html>Checking your browser</html>`,
			vendor: "ddos-guard",
			kind:   "js-challenge",
		},
		{
			name:   "sucuri deny",
			status: 403,
			header: http.Header{"X-Sucuri-Id": {"12345"}},
			body:   `<html>Access Denied - Sucuri Website Firewall</html>`,
			vendor: "sucuri",
			kind:   "block",
		},
		{
			name:   "aws waf",
			status: 405,
			header: http.Header{"X-Amzn-Waf-Action": {"challenge"}},
			body:   `<html>awswaf integration</html>`,
			vendor: "aws-waf",
			kind:   "js-challenge",
		},
		{
			name:   "generic js challenge",
			status: 503,
			header: http.Header{},
			body:   `<html>Checking your browser before continuing</html>`,
			vendor: "generic",
			kind:   "js-challenge",
		},
		{
			name:   "normal json is clean",
			status: 200,
			header: http.Header{"Content-Type": {"application/json"}},
			body:   `{"batchcomplete":true,"query":{"search":[]}}`,
			vendor: "",
		},
		{
			name:   "plain 404 is clean",
			status: 404,
			header: http.Header{},
			body:   `<html>not found</html>`,
			vendor: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := classify(tc.status, tc.header, []byte(tc.body))
			if p.Vendor != tc.vendor || p.Kind != tc.kind {
				t.Fatalf("got %+v, want vendor=%q kind=%q", p, tc.vendor, tc.kind)
			}
		})
	}
}

func TestCFErrorCode(t *testing.T) {
	if got := cfErrorCode("sorry error code: 1020 done"); got != "1020" {
		t.Fatalf("got %q", got)
	}
	if got := cfErrorCode("no code here"); got != "" {
		t.Fatalf("got %q", got)
	}
}
