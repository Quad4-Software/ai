// SPDX-License-Identifier: 0BSD
package melovian

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// extensionIDRe is the registry id shape: lowercase slug, 2 to 48
// chars, segments separated by single hyphens.
var extensionIDRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ScaffoldFile is one generated file: a path relative to the extension
// root plus its full contents.
type ScaffoldFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ScaffoldExtension returns the source files for a new registry
// extension. It never writes to disk: the caller decides where the
// files land. Install the result server-side with install-dir.
func ScaffoldExtension(id, name, description, author string) ([]ScaffoldFile, error) {
	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}
	if len(id) < 2 || len(id) > 48 || !extensionIDRe.MatchString(id) {
		return nil, fmt.Errorf("invalid extension id %q; want 2-48 chars of [a-z0-9-] slug form, for example mood-lights", id)
	}
	if name == "" {
		name = id
	}
	if description == "" {
		description = "A Melovian extension."
	}
	if author == "" {
		author = "Unknown"
	}

	manifest := map[string]any{
		"id":            id,
		"name":          name,
		"version":       "1.0.0",
		"description":   description,
		"author":        author,
		"license":       "Apache-2.0",
		"minAppVersion": "0.1.0",
		"appTheme":      id,
		"styles":        []string{id + ".css"},
		"script":        "script.ts",
		"permissions": []string{
			"styles",
			"script",
			"theme",
			"track-decorations",
		},
		"settings": []map[string]any{
			{
				"key":     "enabled",
				"label":   "Enable " + name,
				"type":    "boolean",
				"default": true,
			},
		},
	}
	mj, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}

	script := fmt.Sprintf(`// SPDX-License-Identifier: 0BSD
// %s extension script. The sandbox has no DOM, network, or timers.
// Register track rules from settings; styling lives in %s.css.
declare const melovian: {
  settings(): Record<string, unknown>
  addTrackRule(rule: unknown): void
}

const settings = melovian.settings()
const enabled = settings["enabled"] !== false

if (enabled) {
  melovian.addTrackRule({
    match: {},
    decoration: {
      progressGradient: "linear-gradient(90deg, #6d8dff, #41e6c0)",
    },
  })
}
`, name, id)

	css := fmt.Sprintf(`/* %s theme layer. Gated on the appTheme attribute so it
   only applies while this extension is enabled. */
[data-extension-theme="%s"] .player-bar {
  --%s-accent: #6d8dff;
}
`, name, id, id)

	return []ScaffoldFile{
		{Path: "melovian-extension.json", Content: string(mj) + "\n"},
		{Path: "script.ts", Content: script},
		{Path: id + ".css", Content: css},
		{Path: "CHANGELOG.md", Content: "## 1.0.0\n\n- Initial release\n"},
	}, nil
}
