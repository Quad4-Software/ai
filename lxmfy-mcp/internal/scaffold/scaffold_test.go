// SPDX-License-Identifier: 0BSD
package scaffold

import (
	"strings"
	"testing"
)

func TestGenerateBot(t *testing.T) {
	for _, tmpl := range Templates {
		files, err := GenerateBot(tmpl, "mybot")
		if err != nil {
			t.Fatalf("template %s: %v", tmpl, err)
		}
		if len(files) == 0 {
			t.Fatalf("template %s produced no files", tmpl)
		}
		if tmpl == "cogtest" && !strings.Contains(files[0].Content, "CogTestBot") {
			t.Fatalf("cogtest should use CogTestBot class: %s", files[0].Content)
		}
		if tmpl == "rrc" && !strings.Contains(files[0].Content, "RRCBot") {
			t.Fatalf("rrc should use RRCBot class: %s", files[0].Content)
		}
	}
	_, err := GenerateBot("nope", "mybot")
	if err == nil {
		t.Fatal("unknown template should error")
	}
	_, err = GenerateBot("minimal", "9bad")
	if err == nil {
		t.Fatal("bad name should error")
	}
}

func TestGenerateCog(t *testing.T) {
	files, err := GenerateCog("utility")
	if err != nil {
		t.Fatalf("cog: %v", err)
	}
	if len(files) != 1 || !strings.Contains(files[0].Content, "uptime") {
		t.Fatalf("unexpected cog output: %+v", files)
	}
}

func TestDiagnoseBot(t *testing.T) {
	out, err := DiagnoseBot("from lxmfy import LXMFBot\nbot = LXMFBot(name='x')\nbot.run()\n")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if !strings.Contains(out, "admins") {
		t.Fatalf("expected admin warning: %s", out)
	}
}

func TestTests(t *testing.T) {
	if !strings.Contains(Tests(), "manifold") {
		t.Fatal("Tests() should mention manifold")
	}
}
