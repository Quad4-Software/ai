// SPDX-License-Identifier: 0BSD
// Minimal HTML to plain text conversion for rendered wiki content.
// Stdlib has no HTML parser; this is a tag scanner tuned for parser
// output: block elements become newlines, script and style bodies are
// dropped, entities are unescaped, and whitespace is normalized.
package mediawiki

import (
	"html"
	"strings"
)

// blockTags map to a newline boundary in text output.
var blockTags = map[string]bool{
	"p": true, "div": true, "br": true, "hr": true,
	"ul": true, "ol": true, "li": true, "dl": true, "dt": true, "dd": true,
	"table": true, "tr": true, "td": true, "th": true, "caption": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"blockquote": true, "pre": true, "section": true, "article": true,
	"header": true, "footer": true, "figure": true, "figcaption": true,
}

// dropTags discard their entire contents.
var dropTags = map[string]bool{
	"script": true, "style": true, "head": true, "template": true,
}

// stripHTML converts rendered HTML to readable plain text.
func stripHTML(s string) string {
	var out strings.Builder
	out.Grow(len(s) / 2)
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '<' {
			// comment or tag
			if strings.HasPrefix(s[i:], "<!--") {
				end := strings.Index(s[i+4:], "-->")
				if end < 0 {
					break
				}
				i += 4 + end + 3
				continue
			}
			end := strings.IndexByte(s[i:], '>')
			if end < 0 {
				break
			}
			tag := tagName(s[i+1 : i+end])
			selfClose := strings.HasSuffix(strings.TrimSpace(s[i+1:i+end]), "/")
			if dropTags[tag] && !strings.HasPrefix(s[i+1:], "/") && !selfClose {
				// skip to matching close tag
				close := "</" + tag + ">"
				j := strings.Index(strings.ToLower(s[i+end+1:]), close)
				if j < 0 {
					i = len(s)
					continue
				}
				i += end + 1 + j + len(close)
				continue
			}
			if blockTags[tag] {
				out.WriteByte('\n')
			}
			i += end + 1
			continue
		}
		if c == '&' {
			if semi := strings.IndexByte(s[i:], ';'); semi > 0 && semi < 12 {
				ent := s[i : i+semi+1]
				if dec := html.UnescapeString(ent); dec != ent {
					out.WriteString(dec)
					i += semi + 1
					continue
				}
			}
		}
		out.WriteByte(c)
		i++
	}
	return normalizeSpace(out.String())
}

// tagName lowercases and trims a raw tag like "div class=x" or "/p".
func tagName(raw string) string {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "/"))
	if i := strings.IndexAny(raw, " \t\n/>"); i >= 0 {
		raw = raw[:i]
	}
	return strings.ToLower(raw)
}

// normalizeSpace trims lines, collapses horizontal whitespace, and
// caps blank runs at one empty line.
func normalizeSpace(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blank := false
	for _, ln := range lines {
		ln = strings.Join(strings.Fields(ln), " ")
		if ln == "" {
			if blank {
				continue
			}
			blank = true
		} else {
			blank = false
		}
		out = append(out, ln)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
