// SPDX-License-Identifier: 0BSD
package issues

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Field is one input of an issue template, mirroring the GitHub issue
// forms in .github/ISSUE_TEMPLATE of the target repo.
type Field struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Hint     string `json:"hint,omitempty"`
}

// Template is an issue form: title prefix, fixed labels, and the
// ordered fields that become ### heading sections in the body.
type Template struct {
	Kind        string   `json:"kind"`
	TitlePrefix string   `json:"title_prefix"`
	Labels      []string `json:"labels"`
	Fields      []Field  `json:"fields"`
}

var osOptions = "Windows 11, Windows 10, Linux, macOS, Android, All platforms, Other"
var installOptions = "Windows installer / portable, Linux AppImage / deb / Flatpak, macOS app, Android APK, pip / uv / source checkout, Docker, Other"

// Templates are the supported issue kinds, matching the repo's issue
// forms plus an optional context section seen on filed issues.
var Templates = map[string]Template{
	"bug": {
		Kind:        "bug",
		TitlePrefix: "[Bug]: ",
		Labels:      []string{"bug"},
		Fields: []Field{
			{ID: "version", Label: "MeshChatX version", Required: true, Hint: "for example 4.8.6, or N/A"},
			{ID: "os", Label: "Operating system", Required: true, Hint: osOptions},
			{ID: "os_other", Label: "OS details (if Other)", Hint: "distribution, build, or device model"},
			{ID: "install", Label: "How did you install MeshChatX?", Required: true, Hint: installOptions},
			{ID: "description", Label: "What happened?", Required: true, Hint: "the bug and what you expected instead, in prose"},
			{ID: "reproduction", Label: "Steps to reproduce", Required: true, Hint: "numbered steps that reliably trigger it"},
			{ID: "logs", Label: "Logs or error output", Hint: "console output or log lines, fenced code blocks allowed"},
			{ID: "context", Label: "Additional context"},
		},
	},
	"feature": {
		Kind:        "feature",
		TitlePrefix: "[Feature]: ",
		Labels:      []string{"enhancement"},
		Fields: []Field{
			{ID: "version", Label: "MeshChatX version", Hint: "if relevant, or N/A"},
			{ID: "os", Label: "Operating system", Required: true, Hint: osOptions},
			{ID: "os_other", Label: "OS details (if Other)"},
			{ID: "problem", Label: "Problem or use case", Required: true, Hint: "what you are trying to do and what is missing today"},
			{ID: "proposal", Label: "Proposed solution", Required: true, Hint: "how you would like this to work"},
			{ID: "alternatives", Label: "Alternatives considered"},
			{ID: "context", Label: "Additional context"},
		},
	},
}

// Kinds lists the template kinds, sorted for stable output.
func Kinds() []string {
	out := make([]string, 0, len(Templates))
	for k := range Templates {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var wsRun = regexp.MustCompile(`[ \t]{2,}`)

// CleanProse normalizes free text to the house style used in filed
// issues: plain sentences without em dashes, en dashes, semicolons, or
// inline code backticks. Fenced code blocks are left untouched so logs
// and reproduction output keep their formatting.
func CleanProse(s string) string {
	var out []string
	fenced := false
	for line := range strings.SplitSeq(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			out = append(out, line)
			continue
		}
		if fenced {
			out = append(out, line)
			continue
		}
		line = strings.ReplaceAll(line, "—", ", ")  // em dash
		line = strings.ReplaceAll(line, "–", "-")   // en dash
		line = strings.ReplaceAll(line, "; ", ". ") // semicolon before text
		line = strings.ReplaceAll(line, ";", ",")
		line = strings.ReplaceAll(line, "`", "")
		line = wsRun.ReplaceAllString(line, " ")
		line = strings.ReplaceAll(line, ", ,", ",")
		out = append(out, strings.TrimRight(line, " \t"))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// Title builds the final issue title for kind, adding the template
// prefix when missing.
func Title(kind, title string) (string, error) {
	t, ok := Templates[kind]
	if !ok {
		return "", fmt.Errorf("unknown issue kind %q; want one of %s", kind, strings.Join(Kinds(), ", "))
	}
	title = CleanProse(title)
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	if !strings.HasPrefix(title, t.TitlePrefix) {
		title = t.TitlePrefix + title
	}
	return title, nil
}

// BuildBody renders values into the markdown body GitHub produces for
// issue forms: each field becomes a ### Label section. Missing required
// fields are an error. Unknown keys are an error so typos do not drop
// content silently.
func BuildBody(kind string, values map[string]string) (string, error) {
	t, ok := Templates[kind]
	if !ok {
		return "", fmt.Errorf("unknown issue kind %q; want one of %s", kind, strings.Join(Kinds(), ", "))
	}
	known := map[string]bool{}
	for _, f := range t.Fields {
		known[f.ID] = true
	}
	for k := range values {
		if !known[k] {
			return "", fmt.Errorf("unknown field %q for kind %s", k, kind)
		}
	}
	var b strings.Builder
	var missing []string
	for _, f := range t.Fields {
		v := CleanProse(values[f.ID])
		if v == "" {
			if f.Required {
				missing = append(missing, f.ID)
			}
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", f.Label, v)
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}
