// SPDX-License-Identifier: 0BSD
package issues

import (
	"regexp"
	"sort"
	"strings"
)

// References maps project and spec terms to their canonical source.
// Terms that appear in issue text are linked to these so readers do not
// have to know the ecosystem jargon (BCP 47, LXMF, KISS, ...).
var References = map[string]string{
	"BCP 47":           "https://www.rfc-editor.org/rfc/rfc5646",
	"ISO 639-1":        "https://www.loc.gov/standards/iso639-2/php/code_list.php",
	"Reticulum":        "https://reticulum.network/",
	"LXMF":             "https://github.com/markqvist/LXMF",
	"LXST":             "https://github.com/markqvist/LXST",
	"LXMFy":            "https://github.com/quad4-infra/lxmfy",
	"NomadNet":         "https://github.com/markqvist/NomadNet",
	"Micron":           "https://github.com/markqvist/Micron",
	"RNode":            "https://unsigned.io/rnode/",
	"KISS":             "https://en.wikipedia.org/wiki/KISS_(TNC)",
	"propagation node": "https://github.com/markqvist/LXMF",
	"Sideband":         "https://github.com/markqvist/Sideband",
	"Landlock":         "https://landlock.io/",
	"CSP":              "https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CSP",
	"CSRF":             "https://owasp.org/www-community/attacks/csrf",
	"WebSocket":        "https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API",
	"msgpack":          "https://msgpack.org/",
	"X25519":           "https://cr.yp.to/ecdh.html",
	"Ed25519":          "https://ed25519.cr.yp.to/",
	"BLE":              "https://www.bluetooth.com/specifications/specs/",
	"Termux":           "https://termux.dev/",
	"AppImage":         "https://appimage.org/",
	"Flatpak":          "https://flatpak.org/",
	"SideQuest":        "https://sidequestvr.com/",
}

// refRe caches per-term matchers, built lazily by LinkReferences.
var refRe = func() map[string]*regexp.Regexp {
	m := map[string]*regexp.Regexp{}
	for term := range References {
		m[term] = regexp.MustCompile(`\b` + regexp.QuoteMeta(term) + `\b`)
	}
	return m
}()

// refTerms lists terms longest first so "ISO 639-1" wins over any
// shorter overlapping key.
var refTerms = func() []string {
	out := make([]string, 0, len(References))
	for t := range References {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}()

var mdLinkRe = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)

// LinkReferences rewrites the first plain-text occurrence of each known
// term into a markdown link to its canonical reference. Each term is
// linked at most once per document. Occurrences inside fenced code
// blocks, inside existing markdown links, or inside URLs are left alone,
// and a term already used as link text anywhere is skipped.
func LinkReferences(body string) string {
	done := map[string]bool{}
	for _, term := range refTerms {
		if strings.Contains(body, "["+term+"](") {
			done[term] = true
		}
	}
	var out []string
	fenced := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			out = append(out, line)
			continue
		}
		if !fenced {
			line = linkLine(line, done)
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func linkLine(line string, done map[string]bool) string {
	for _, term := range refTerms {
		if done[term] {
			continue
		}
		re := refRe[term]
		for _, loc := range re.FindAllStringIndex(line, -1) {
			s, e := loc[0], loc[1]
			if linkedOrCoded(line, s, e) {
				continue
			}
			line = line[:s] + "[" + line[s:e] + "](" + References[term] + ")" + line[e:]
			done[term] = true
			break
		}
	}
	return line
}

// linkedOrCoded reports whether the span at [s,e) is already inside a
// markdown link, an inline code span, or a bare URL.
func linkedOrCoded(line string, s, e int) bool {
	// inside [text](url)
	for _, m := range mdLinkRe.FindAllStringIndex(line, -1) {
		if s >= m[0] && e <= m[1] {
			return true
		}
	}
	// inside `code`
	if strings.Count(line[:s], "`")%2 == 1 {
		return true
	}
	// part of a URL: preceded by :// or surrounded by non-space URL chars
	if s >= 3 && line[s-3:s] == "://" {
		return true
	}
	if s > 0 && (line[s-1] == '/' || line[s-1] == '(') && strings.Contains(line[:s], "http") {
		return true
	}
	return false
}
