// SPDX-License-Identifier: 0BSD
// Package scaffold emits MeshChatX-convention file templates for
// agents working on the codebase: Svelte 5 features and components,
// backend managers, WS handlers, plugins, and oracle tests.
package scaffold

import (
	"fmt"
	"regexp"
	"strings"
)

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,48}$`)

// File is a generated path plus content.
type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Note    string `json:"note,omitempty"`
}

// Kinds lists valid scaffold kinds.
var Kinds = []string{
	"svelte-feature", "svelte-component", "backend-manager",
	"ws-handler", "plugin", "oracle-test",
}

// Generate returns the file set for a scaffold kind.
func Generate(kind, name string) ([]File, error) {
	if !nameRe.MatchString(name) {
		return nil, fmt.Errorf("name must be lowercase identifier (got %q); use letters, digits, - or _", name)
	}
	switch kind {
	case "svelte-feature":
		return svelteFeature(name), nil
	case "svelte-component":
		return svelteComponent(name), nil
	case "backend-manager":
		return backendManager(name), nil
	case "ws-handler":
		return wsHandler(name), nil
	case "plugin":
		return plugin(name), nil
	case "oracle-test":
		return oracleTest(name), nil
	}
	return nil, fmt.Errorf("unknown kind %q; valid: %s", kind, strings.Join(Kinds, ", "))
}

// Checks returns the verification commands for a surface.
func Checks(surface string) (string, error) {
	m := map[string]string{
		"svelte":   "pnpm run svelte-check\npnpm run format:check:svelte\npnpm exec vitest run tests/frontend/<Name>.test.js\npnpm exec vitest run tests/frontend/featureModuleOwnership.test.js",
		"backend":  "uv run pytest tests/backend/test_<name>.py -q --tb=short",
		"plugin":   "uv run pytest tests/backend/test_plugin_manager.py tests/backend/test_plugin_permissions.py tests/backend/test_plugin_signature.py tests/backend/test_plugin_integrity.py tests/backend/test_plugin_python_runtime.py tests/backend/test_plugin_security.py -q --tb=short",
		"ws":       "uv run pytest tests/backend/ -k ws -q --tb=short\n# check the WS mutator denylist for new mutating types",
		"docs":     "task docs:check  # or the docs lint target; keep prose clean per no-ai-slop rules",
		"i18n":     "verify keys exist in all locale JSON files; user-visible strings use t()",
		"electron": "pnpm exec vitest run tests/electron/ -q",
		"landlock": "uv run pytest tests/backend/test_landlock_integration_surfaces.py -q --tb=short",
	}
	if s, ok := m[surface]; ok {
		return s, nil
	}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return "", fmt.Errorf("unknown surface %q; valid: %s", surface, strings.Join(keys, ", "))
}

func pascal(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func svelteFeature(name string) []File {
	P := pascal(name)
	return []File{
		{Path: fmt.Sprintf("meshchatx/src/frontend/features/%s/index.ts", name), Content: fmt.Sprintf(`// SPDX-License-Identifier: 0BSD
import { registerFeature } from "../../js/registries/featureRegistry.js";

export function register%sFeature() {
    registerFeature({
        id: %q,
        routes: [
            {
                name: %q,
                path: "/%s",
                mount: "svelte",
                load: () => import("./%sPage.svelte"),
            },
        ],
    });
}
`, P, name, name, name, P), Note: "Call register" + P + "Feature() from features/registerAllFeatures.ts after registerCoreContributions."},
		{Path: fmt.Sprintf("meshchatx/src/frontend/features/%s/%sPage.svelte", name, P), Content: fmt.Sprintf(`<!-- SPDX-License-Identifier: 0BSD -->
<script lang="ts">
    import { t } from "svelte-i18n";
    // API via window.api kernel clients. Toasts via ToastUtils.
</script>

<section>
    <h1>{$t("%s.title")}</h1>
</section>
`, name), Note: fmt.Sprintf("Add %s.title to every locale JSON. Runes only: no export let.", name)},
		{Path: fmt.Sprintf("tests/frontend/%sPage.test.js", name), Content: fmt.Sprintf(`// SPDX-License-Identifier: 0BSD
import { describe, it, expect } from "vitest";

describe("%s feature", () => {
    it("registers its route", () => {
        // import register%sFeature and assert routeRegistry contains /%s
    });
});
`, name, P, name)},
	}
}

func svelteComponent(name string) []File {
	P := pascal(name)
	return []File{
		{Path: fmt.Sprintf("meshchatx/src/frontend/ui/svelte/%s.svelte", P), Content: fmt.Sprintf(`<!-- SPDX-License-Identifier: 0BSD -->
<script lang="ts">
    interface Props {
        label: string;
    }
    let { label }: Props = $props();
</script>

<span class="%s">{label}</span>
`, name), Note: "Svelte 5 runes only. ui/svelte is for shared primitives; feature-locked UI goes in features/<id>/."},
	}
}

func backendManager(name string) []File {
	P := pascal(name)
	return []File{
		{Path: fmt.Sprintf("meshchatx/src/backend/%s_manager.py", name), Content: fmt.Sprintf(`# SPDX-License-Identifier: 0BSD
"""%s manager.

Owns %s state inside IdentityContext. Must release destinations,
timers, and DB handles on identity switch.
"""


class %sManager:
    def __init__(self, identity_context):
        self._ctx = identity_context

    def teardown(self):
        """Release destinations, timers, and handles. Called on identity switch."""
        pass
`, P, name, P), Note: "Keep identity-scoped state inside IdentityContext; implement teardown for identity switch. New mutating HTTP routes need CSRF; new WS mutators go through the denylist check."},
	}
}

func wsHandler(name string) []File {
	return []File{
		{Path: fmt.Sprintf("meshchatx/src/backend/ws/handlers_%s.py", name), Content: fmt.Sprintf(`# SPDX-License-Identifier: 0BSD
"""WebSocket handlers for %s."""


def register_%s_handlers(registry, ctx):
    async def handle_%s_get(ws, msg):
        # read-only queries only in this file or mark mutating types
        return {"ok": True}

    registry.register("%s.get", handle_%s_get)
`, name, name, name, name, name), Note: "Mutating WS types must pass the mutator denylist; read .agents/skills/auth-csrf-ws-security/SKILL.md first."},
	}
}

func plugin(name string) []File {
	return []File{
		{Path: fmt.Sprintf("plugins/%s/plugin.json", name), Content: fmt.Sprintf(`{
    "id": %q,
    "name": %q,
    "version": "0.1.0",
    "permissions": [],
    "managers": [],
    "hooks": [],
    "entry": "main.py",
    "runtime": "python"
}
`, name, name), Note: "Nothing is available unless declared in the manifest and granted by the user. New hooks go in KNOWN_HOOKS, new managers in KNOWN_MANAGERS (plugin_permissions.py)."},
		{Path: fmt.Sprintf("plugins/%s/main.py", name), Content: fmt.Sprintf(`# SPDX-License-Identifier: 0BSD
"""%s plugin entry point."""


def on_load(ctx):
    pass
`, name)},
	}
}

func oracleTest(name string) []File {
	return []File{
		{Path: fmt.Sprintf("tests/backend/test_%s_oracle.py", name), Content: fmt.Sprintf(`# SPDX-License-Identifier: 0BSD
"""Oracle tests for %s.

An oracle predicts the outcome independently of the code under test.
Do not ship tests that only assert no crash.
"""


def test_%s_rejects_invalid():
    # invariant in one sentence:
    # expected: invalid input raises ValueError
    pass
`, name, name), Note: "See .agents/skills/test-oracles/SKILL.md for oracle types."},
	}
}
