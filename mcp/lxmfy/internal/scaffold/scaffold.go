// SPDX-License-Identifier: 0BSD
// Package scaffold emits LXMFy bot and cog templates and runs lightweight
// static diagnostics on bot source code.
package scaffold

import (
	"fmt"
	"regexp"
	"strings"
)

// File is a generated path plus content.
type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Note    string `json:"note,omitempty"`
}

// Templates lists the built-in bot template names supported by lxmfy create.
var Templates = []string{"minimal", "echo", "note", "reminder", "rrc", "cogtest"}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,48}$`)

var templateClasses = map[string]string{
	"echo":     "Echo",
	"note":     "Note",
	"reminder": "Reminder",
	"rrc":      "RRC",
	"cogtest":  "CogTest",
}

// GenerateBot returns the file set for a template and bot name.
func GenerateBot(template, name string) ([]File, error) {
	if !nameRe.MatchString(name) {
		return nil, fmt.Errorf("name must be lowercase identifier (got %q); use letters, digits, - or _", name)
	}
	switch template {
	case "minimal":
		return minimalBot(name), nil
	case "echo", "note", "reminder", "rrc", "cogtest":
		return templateBot(template, name), nil
	}
	return nil, fmt.Errorf("unknown template %q; valid: %s", template, strings.Join(Templates, ", "))
}

// GenerateCog returns a cogs/<name>.py extension file.
func GenerateCog(name string) ([]File, error) {
	if !nameRe.MatchString(name) {
		return nil, fmt.Errorf("name must be lowercase identifier (got %q); use letters, digits, - or _", name)
	}
	P := pascal(name)
	return []File{{
		Path: fmt.Sprintf("cogs/%s.py", name),
		Content: fmt.Sprintf(`# SPDX-License-Identifier: 0BSD
"""%s cog."""

import time

from lxmfy import Command


class %sCog:
    def __init__(self, bot):
        self.bot = bot
        self.start_time = time.time()

    @Command(name="uptime", description="Seconds since bot start")
    def uptime(self, ctx):
        ctx.reply(f"Uptime: {time.time() - self.start_time:.1f}s")


def setup(bot):
    bot.add_cog(%sCog(bot))
`, name, P, P),
		Note: "Load by setting cogs_enabled=True and cogs_dir='cogs' in your LXMFBot. The command is /uptime.",
	}}, nil
}

// Tests returns a summary of the LXMFy reliability suite.
func Tests() string {
	return `LXMFy test categories
- manifold:   NLP intent vector-space topology validation
- chaos:      bit-rot, storage corruption, SD card failure simulation
- temporal:   system clock jumps (±1 year) resilience
- leak:       memory, file descriptor, and thread tracking

Run with the upstream test runner:
  make test        # from the LXMFy repo
  make ci          # lint, typecheck, security, tests, build
  pytest tests/    # full suite

See the "Testing" section in the api-reference page for details.`
}

// DiagnoseBot statically checks bot Python source for common mistakes.
func DiagnoseBot(code string) (string, error) {
	var issues []string
	codeLow := strings.ToLower(code)

	if !strings.Contains(codeLow, "lxmfbot(") {
		issues = append(issues, "no LXMFBot(...) constructor found")
	}
	if !strings.Contains(codeLow, "bot.run()") {
		issues = append(issues, "bot.run() not found; the bot will not start")
	} else if !strings.Contains(code, `if __name__ == "__main__":`) {
		issues = append(issues, "guard bot.run() with if __name__ == '__main__'")
	}
	if !strings.Contains(codeLow, "admins") {
		issues = append(issues, "no admins configured; admin_only commands are unreachable and privileged commands are exposed")
	} else if strings.Contains(codeLow, "admins=set()") || strings.Contains(codeLow, "admins={}") {
		issues = append(issues, "admins is empty; admin_only commands are unreachable and anyone can trigger privileged commands")
	}
	if strings.Contains(codeLow, "landlock_enabled=false") {
		issues = append(issues, "Landlock LSM sandbox is disabled; only do this on trusted systems")
	}
	if strings.Contains(codeLow, "external_cogs_enabled=true") && !strings.Contains(codeLow, "external_cogs_sandbox_enabled=true") {
		issues = append(issues, "external cogs are enabled without a sandbox; set external_cogs_sandbox_enabled=true or disable")
	}
	if strings.Contains(codeLow, "threaded=true") {
		issues = append(issues, "threaded command found: do not touch RNS/LXMF/transport objects inside it; use ctx.reply() for results")
	}
	if strings.Contains(codeLow, "signature_verification_enabled=false") {
		issues = append(issues, "signature verification is disabled; messages are not cryptographically validated")
	}
	if strings.Contains(codeLow, "require_message_signatures=false") {
		issues = append(issues, "unsigned messages are accepted; consider enabling require_message_signatures")
	}

	if len(issues) == 0 {
		return "No obvious issues found. Verify functionality by running the bot in a test Reticulum instance.", nil
	}
	return "warnings:\n- " + strings.Join(issues, "\n- "), nil
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

func minimalBot(name string) []File {
	return []File{{
		Path: name + ".py",
		Content: fmt.Sprintf(`# SPDX-License-Identifier: 0BSD
"""%s: a minimal LXMFy bot."""

from lxmfy import LXMFBot

bot = LXMFBot(
    name=%q,
    announce=600,
    announce_immediately=True,
    command_prefix="/",
    storage_type="json",
    storage_path="data",
)


@bot.command(name="hello", description="Say hello")
def hello(ctx):
    ctx.reply(f"Hello, {ctx.sender}!")


if __name__ == "__main__":
    print(f"Starting {bot.config.name}")
    print(f"LXMF address: {bot.local.hash}")
    bot.run()
`, name, name),
		Note: "Add your admin hash with bot.config.admins.add('<hash>') before running.",
	}, {
		Path: "cogs/basic.py",
		Content: `# SPDX-License-Identifier: 0BSD
"""Example cog; commands here are auto-loaded when cogs_enabled=True."""

from lxmfy import Command


@Command(name="about", description="Show bot info")
def about(ctx):
    ctx.reply("Built with LXMFy")


def setup(bot):
    # Register commands or classes here
    pass
`,
		Note: "Rename the @Command or create a class to keep commands organized.",
	}}
}

func templateBot(kind, name string) []File {
	class := templateClasses[kind]
	if class == "" {
		class = pascal(kind)
	}
	return []File{{
		Path: name + ".py",
		Content: fmt.Sprintf(`# SPDX-License-Identifier: 0BSD
"""%s: %s template bot."""

from lxmfy.templates import %sBot

if __name__ == "__main__":
    bot = %sBot()
    # bot.config.name = %q
    bot.run()
`, name, kind, class, class, name),
		Note: fmt.Sprintf("Verify the %sBot class exists in your installed lxmfy version.", class),
	}}
}
