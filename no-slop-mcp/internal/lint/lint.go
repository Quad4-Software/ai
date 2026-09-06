// SPDX-License-Identifier: 0BSD
// Package lint checks prose against the no-AI-slop rule set and the
// MeshChatX style rules: no emojis, no emoji arrows, no emdashes,
// restrained semicolons and backticks, no backticks in code comments.
package lint

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Kind controls which checks apply.
type Kind string

const (
	KindProse   Kind = "prose"   // general prose
	KindDoc     Kind = "doc"     // docs/markdown: semicolons banned
	KindComment Kind = "comment" // code comments: semicolons and backticks banned
)

var docExts = map[string]bool{
	".md": true, ".markdown": true, ".rst": true, ".txt": true,
	".adoc": true, ".mdx": true, ".html": true, ".vue": true, ".svelte": true,
}

var codeExts = map[string]bool{
	".go": true, ".py": true, ".js": true, ".ts": true, ".jsx": true,
	".tsx": true, ".c": true, ".h": true, ".cpp": true, ".rs": true,
	".java": true, ".sh": true, ".css": true, ".json": true, ".yaml": true,
	".yml": true, ".toml": true, ".ini": true, ".cfg": true,
}

// KindForPath guesses the lint kind from a filename extension.
func KindForPath(path string) Kind {
	i := strings.LastIndexByte(path, '.')
	if i < 0 {
		return KindProse
	}
	ext := strings.ToLower(path[i:])
	if docExts[ext] {
		return KindDoc
	}
	if codeExts[ext] {
		return KindComment
	}
	return KindProse
}

// CheckDiff lints added lines of a unified diff. Returned findings use
// the new-file line numbers.
func CheckDiff(diff string, kind Kind) []Finding {
	var added []string
	var lineNums []int
	newLine := 0
	hunkRe := regexp.MustCompile(`@@ -\d+(?:,\d+)? \+(\d+)`)
	for line := range strings.SplitSeq(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "@@"):
			if m := hunkRe.FindStringSubmatch(line); m != nil {
				fmt.Sscanf(m[1], "%d", &newLine) // #nosec G104 -- error tolerated; empty/default is handled downstream
			}
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		case strings.HasPrefix(line, "+"):
			added = append(added, line[1:])
			lineNums = append(lineNums, newLine)
			newLine++
		case strings.HasPrefix(line, "-"):
			// removed lines do not advance new-line counter
		default:
			if newLine > 0 {
				newLine++
			}
		}
	}
	if len(added) == 0 {
		return nil
	}
	findings := Check(strings.Join(added, "\n"), kind)
	for i := range findings {
		if findings[i].Line > 0 && findings[i].Line <= len(lineNums) {
			findings[i].Line = lineNums[findings[i].Line-1]
		}
	}
	return findings
}

// Finding is one rule violation.
type Finding struct {
	Rule    string `json:"rule"`
	Line    int    `json:"line"`
	Match   string `json:"match"`
	Message string `json:"message"`
}

// Check lints text. kind may be "prose", "doc", or "comment".
func Check(text string, kind Kind) []Finding {
	var out []Finding
	out = append(out, checkChars(text, kind)...)
	out = append(out, checkPhrases(text)...)
	out = append(out, checkHeadings(text)...)
	out = append(out, checkStructure(text)...)
	return out
}

func isEmoji(r rune) bool {
	return unicode.Is(unicode.So, r) ||
		(r >= 0x1F000 && r <= 0x1FAFF) ||
		(r >= 0x2190 && r <= 0x21FF) || // arrows
		(r >= 0x2B00 && r <= 0x2BFF) || // stars, arrows
		(r >= 0x2700 && r <= 0x27BF) || // dingbats incl. decorative arrows
		r == 0xFE0F || r == 0x3030 || r == 0x303D
}

func isCommentLine(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") ||
		strings.HasPrefix(t, "--") || strings.HasPrefix(t, "*") ||
		strings.HasPrefix(t, "/*")
}

func checkChars(text string, kind Kind) []Finding {
	var out []Finding
	semiLines := 0
	totalBackticks := strings.Count(text, "`")
	for i, line := range strings.Split(text, "\n") {
		ln := i + 1
		// emojis and decorative arrows
		for _, r := range line {
			if isEmoji(r) {
				out = append(out, Finding{Rule: "emoji", Line: ln, Match: string(r),
					Message: "emojis and decorative unicode arrows are banned; use words"})
				break
			}
		}
		// emdashes, endashes, double hyphens used as dashes
		for _, m := range []struct{ pat, name string }{
			{"—", "em-dash"}, {"–", "en-dash"}, {" -- ", "double-hyphen"},
		} {
			if strings.Contains(line, m.pat) {
				out = append(out, Finding{Rule: "dash", Line: ln, Match: m.name,
					Message: "dashes are banned; use a period, comma, colon, or restructure"})
			}
		}
		if c := strings.Count(line, ";"); c > 0 {
			semiLines++
			if kind != KindProse || c > 1 {
				out = append(out, Finding{Rule: "semicolon", Line: ln, Match: ";",
					Message: "semicolons are banned in comments and docs; use a period or comma"})
			}
		}
		if strings.Contains(line, "`") {
			if kind == KindComment || isCommentLine(line) {
				out = append(out, Finding{Rule: "backtick-comment", Line: ln, Match: "`",
					Message: "no backticks in code comments; use plain words or quoted identifiers"})
			}
		}
	}
	if kind == KindProse && semiLines > 3 {
		out = append(out, Finding{Rule: "semicolon-density", Match: fmt.Sprintf("%d lines use semicolons", semiLines),
			Message: "semicolon density is an AI tell; prefer periods and commas"})
	}
	if kind != KindComment && totalBackticks > 12 {
		out = append(out, Finding{Rule: "backtick-density", Match: fmt.Sprintf("%d backticks", totalBackticks),
			Message: "too many inline backticks; use fenced blocks for code, plain words for short names"})
	}
	return out
}

// banned phrases: lowercase substring -> suggestion.
var bannedPhrases = []struct{ pat, msg string }{
	// intensifiers
	{"extremely", "intensifier; prove it with a number or cut it"},
	{"significantly", "intensifier; state the number instead"},
	{"dramatically", "intensifier; state the change instead"},
	{"exceptionally", "intensifier; cut it"},
	{"incredibly", "intensifier; cut it"},
	{"remarkably", "intensifier; cut it"},
	{"literally", "intensifier; cut it"},
	// filler phrases
	{"in today's world", "filler opener; open on the fact"},
	{"in today's fast-paced", "filler opener; open on the fact"},
	{"in today's digital age", "filler opener; open on the fact"},
	{"it's important to note", "filler; state the fact"},
	{"it is important to note", "filler; state the fact"},
	{"when it comes to", "filler; name the subject"},
	{"at the end of the day", "filler; cut it"},
	{"in the realm of", "filler; say in or within"},
	{"it goes without saying", "filler; then do not say it"},
	{"look no further", "marketing filler; cut it"},
	{"our team of experts", "marketing filler; name who and how many"},
	{"in a world where", "filler opener; cut it"},
	{"let's dive in", "filler; start on the content"},
	{"here's the thing", "rhetorical filler; state the thing"},
	{"here's the kicker", "rhetorical filler; state the fact"},
	// AI transitions
	{"furthermore", "AI transition; use also, and, or but"},
	{"moreover", "AI transition; use also or and"},
	{"notwithstanding", "AI transition; use despite or still"},
	{"that being said", "AI transition; use however or but"},
	{"at its core", "AI transition; use essentially or rewrite"},
	{"in essence", "AI transition; rewrite"},
	{"it is worth noting", "AI transition; state the fact"},
	{"it should be noted", "AI transition; state the fact"},
	{"in the landscape of", "AI transition; say in or within"},
	{"to put it simply", "AI transition; use in short or rewrite"},
	{"in conclusion", "AI conclusion; end on the strongest fact"},
	{"to sum up", "AI conclusion; end on the strongest fact"},
	{"all things considered", "AI conclusion; cut it"},
	// AI verbs
	{"delve", "AI verb; use explore or examine"},
	{"leverage", "AI verb; use use or apply"},
	{"utilize", "AI verb; use use"},
	{"utilise", "AI verb; use use"},
	{"facilitate", "AI verb; use help or enable"},
	{"foster", "AI verb; use encourage or support"},
	{"bolster", "AI verb; use strengthen or support"},
	{"underscore", "AI verb; use highlight or stress"},
	{"unveil", "AI verb; use reveal or show"},
	{"streamline", "AI verb; use simplify"},
	{"endeavour", "AI verb; use try"},
	{"ascertain", "AI verb; use find out"},
	{"elucidate", "AI verb; use explain"},
	// academic tells
	{"shed light on", "academic tell; use clarify or explain"},
	{"pave the way for", "academic tell; use enable or allow"},
	{"a myriad of", "academic tell; use many or a count"},
	{"a plethora of", "academic tell; use many or a count"},
	{"paramount", "academic tell; use essential or critical"},
	{"pertaining to", "academic tell; use about"},
	{"prior to", "academic tell; use before"},
	{"subsequent to", "academic tell; use after"},
	{"in light of", "academic tell; use because of"},
	{"with respect to", "academic tell; use about"},
	{"in terms of", "academic tell; use about or for"},
	{"the fact that", "academic tell; rewrite the sentence"},
	// weasel words
	{"may potentially", "weasel; commit or cut"},
	{"can potentially", "weasel; commit or cut"},
	{"might be able to", "weasel; commit or cut"},
	{"can help to", "weasel; commit or cut"},
	{"helps ensure", "weasel; commit or cut"},
	{"it is widely acknowledged", "weasel; cite the source or cut"},
	// inflated symbolism
	{"a stark reminder", "inflated symbolism; state the fact"},
	{"serves as a testament", "inflated symbolism; state the fact"},
	{"a testament to", "inflated symbolism; state the fact"},
	{"watershed moment", "inflated symbolism; state the fact"},
	{"deeply rooted", "inflated symbolism; state the fact"},
	{"an unwavering commitment", "inflated symbolism; state the fact"},
	{"a beacon of", "metaphorical noun; state the fact"},
	{"a tapestry of", "metaphorical noun; state the fact"},
	{"a symphony of", "metaphorical noun; state the fact"},
	{"a valuable insight", "inflated symbolism; state the insight"},
	{"indelible mark", "inflated symbolism; state the fact"},
	{"a significant role in shaping", "inflated symbolism; state what changed"},
	// hallucinated markup
	{"oaicite", "AI citation artifact; remove it"},
	{"contentreference", "AI citation artifact; remove it"},
	{"grok_card", "AI citation artifact; remove it"},
	{"attributableindex", "AI citation artifact; remove it"},
	{"turn0search", "AI citation artifact; remove it"},
	// research narration
	{"could not be located", "research narration; silently omit"},
	{"was not found", "research narration; silently omit"},
	{"no record was found", "research narration; silently omit"},
	{"is not available", "research narration; silently omit"},
	{"as of this writing", "research narration; silently omit"},
	// synthetic enthusiasm
	{"game-changer", "marketing hype; state what changed"},
	{"cutting-edge", "AI adjective; name the version or date"},
	{"groundbreaking", "AI adjective; state the fact"},
	{"revolutionary", "AI adjective; state the fact"},
	{"seamless", "AI adjective; state what the user does"},
	{"robust", "AI adjective; say what it survives"},
	{"comprehensive", "AI adjective; list what it covers"},
	{"pivotal", "AI adjective; say what depends on it"},
}

// sentence-start bans
var sentenceBans = []struct{ pat, msg string }{
	{"whether you're", "never start a sentence with Whether you're"},
	{"it's not just", "contrasting parallelism; say what it is"},
	{"imagine a world", "filler opener; cut it"},
}

var exclam = regexp.MustCompile(`!`)
var notButIs = regexp.MustCompile(`(?i)\bit'?s not\b[^.]{1,80}\bit'?s\b`)
var dramaticHead = regexp.MustCompile(`(?i)^the\s+\w+\s+(trap|killer|cost|problem|danger|secret|lie|myth|truth)\b|^the\s+(hidden|silent|secret|real|true)\s+\w+|^why\s+\w+.*(destroy|kill|hurt|fail)`)

func checkPhrases(text string) []Finding {
	var out []Finding
	lines := strings.Split(text, "\n")
	low := strings.ToLower(text)
	for _, bp := range bannedPhrases {
		idx := 0
		for {
			i := strings.Index(low[idx:], bp.pat)
			if i < 0 {
				break
			}
			line := lineOf(text, idx+i)
			// skip inside direct quotes and code fences
			if !inQuotedOrCode(lines, line) {
				out = append(out, Finding{Rule: "banned-phrase", Line: line, Match: bp.pat, Message: bp.msg})
			}
			idx += i + len(bp.pat)
			if len(out) > 200 {
				return out
			}
		}
	}
	for i, line := range lines {
		l := strings.ToLower(strings.TrimSpace(line))
		for _, sb := range sentenceBans {
			if strings.HasPrefix(l, sb.pat) || strings.Contains(l, ". "+sb.pat) {
				out = append(out, Finding{Rule: "banned-opener", Line: i + 1, Match: sb.pat, Message: sb.msg})
			}
		}
	}
	if c := len(exclam.FindAllString(text, -1)); c > 0 {
		out = append(out, Finding{Rule: "exclamation", Match: fmt.Sprintf("%d exclamation marks", c),
			Message: "no synthetic enthusiasm; let the evidence carry the weight"})
	}
	return out
}

func checkHeadings(text string) []Finding {
	var out []Finding
	for i, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "#") {
			continue
		}
		h := strings.TrimSpace(strings.TrimLeft(t, "#"))
		l := strings.ToLower(h)
		if strings.Contains(h, "(") {
			out = append(out, Finding{Rule: "heading-paren", Line: i + 1, Match: h,
				Message: "no parenthetical clarifications in headings"})
		}
		if dramaticHead.MatchString(l) {
			out = append(out, Finding{Rule: "heading-drama", Line: i + 1, Match: h,
				Message: "heading teases instead of naming the contents; name the subject"})
		}
		for _, vague := range []string{"broader pattern", "broader implications", "wider context", "larger trend", "industry-wide impact"} {
			if strings.Contains(l, vague) {
				out = append(out, Finding{Rule: "heading-vague", Line: i + 1, Match: h,
					Message: "vague analytical heading; name the subject, not the abstraction"})
			}
		}
	}
	return out
}

var sentenceEnd = regexp.MustCompile(`[^.!?\n][.!?](?:\s|$)`)

func checkStructure(text string) []Finding {
	var out []Finding
	// contrasting parallelism density
	if n := len(notButIs.FindAllString(strings.ToLower(text), -1)); n > 2 {
		out = append(out, Finding{Rule: "not-x-but-y", Match: fmt.Sprintf("%d occurrences", n),
			Message: "more than two 'it is not X, it is Y' constructions is an AI tell"})
	}
	// hedging markers per paragraph
	hedges := []string{"may ", "might ", "could ", "potentially", "probably", "generally", "likely", "it seems", "it appears", "arguably"}
	for para := range strings.SplitSeq(text, "\n\n") {
		l := strings.ToLower(para)
		count := 0
		for _, h := range hedges {
			count += strings.Count(l, h)
		}
		if count > 3 {
			line := lineOf(text, strings.Index(text, para))
			out = append(out, Finding{Rule: "hedging", Line: line, Match: fmt.Sprintf("%d hedges in one paragraph", count),
				Message: "more than 3 hedging markers in a paragraph; commit or cut"})
		}
	}
	// sentence length variance: flag flat blocks over ~500 words
	words := len(strings.Fields(text))
	if words >= 300 {
		var lens []int
		start := 0
		for _, loc := range sentenceEnd.FindAllStringIndex(text, -1) {
			lens = append(lens, len(strings.Fields(text[start:loc[0]+1])))
			start = loc[1]
		}
		if len(lens) >= 10 {
			short, long := false, false
			for _, l := range lens {
				if l < 8 {
					short = true
				}
				if l > 30 {
					long = true
				}
			}
			if !short || !long {
				out = append(out, Finding{Rule: "burstiness", Match: "uniform sentence lengths",
					Message: "no sentence under 8 words or over 30; human writing varies"})
			}
		}
	}
	return out
}

func lineOf(text string, offset int) int {
	if offset < 0 {
		return 0
	}
	return strings.Count(text[:offset], "\n") + 1
}

// inQuotedOrCode reports whether a 1-based line sits inside a fenced
// code block or the line wraps the match in double quotes.
func inQuotedOrCode(lines []string, line int) bool {
	inFence := false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") {
			inFence = !inFence
		}
		if i+1 == line {
			return inFence
		}
	}
	return false
}
