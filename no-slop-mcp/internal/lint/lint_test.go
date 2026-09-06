// SPDX-License-Identifier: 0BSD
package lint

import (
	"testing"
)

func rules(fs []Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Rule]++
	}
	return m
}

func TestBannedChars(t *testing.T) {
	fs := rules(Check("Great work 🚀 — it went → well.", KindProse))
	if fs["emoji"] != 1 {
		t.Fatalf("want emoji finding, got %v", fs)
	}
	if fs["dash"] == 0 {
		t.Fatalf("want dash finding, got %v", fs)
	}
}

func TestArrows(t *testing.T) {
	fs := rules(Check("a -> b is fine\nuse → instead", KindProse))
	if fs["emoji"] != 1 {
		t.Fatalf("unicode arrow should be flagged, ascii arrow fine: %v", fs)
	}
}

func TestCommentsAndDocs(t *testing.T) {
	src := "// see the `config` value; it matters\nx := 1; y := 2"
	fs := rules(Check(src, KindComment))
	if fs["backtick-comment"] == 0 {
		t.Fatalf("want backtick-comment finding: %v", fs)
	}
	if fs["semicolon"] == 0 {
		t.Fatalf("want semicolon finding in comment kind: %v", fs)
	}
	// prose allows a single semicolon
	fs = rules(Check("one; two", KindProse))
	if fs["semicolon"] != 0 {
		t.Fatalf("single semicolon in prose should pass: %v", fs)
	}
}

func TestBannedPhrases(t *testing.T) {
	text := "Furthermore, we can leverage this to delve into the topic. It's important to note that."
	fs := Check(text, KindProse)
	m := rules(fs)
	if m["banned-phrase"] < 3 {
		t.Fatalf("want >=3 banned-phrase findings, got %v", fs)
	}
}

func TestExclusionZones(t *testing.T) {
	text := "```\nleverage the delve furthermore\n```\nclean line"
	fs := Check(text, KindProse)
	for _, f := range fs {
		if f.Rule == "banned-phrase" {
			t.Fatalf("fenced code should be excluded: %+v", f)
		}
	}
}

func TestHeadings(t *testing.T) {
	text := "# The Hidden Cost of Serialization\nbody\n# Firmware Corruption After Power Loss\nbody"
	fs := rules(Check(text, KindDoc))
	if fs["heading-drama"] != 1 {
		t.Fatalf("want 1 dramatic heading, got %v", fs)
	}
}

func TestContrastingParallelism(t *testing.T) {
	text := "It's not fast, it's slow. It's not cheap, it's costly. It's not old, it's new."
	fs := rules(Check(text, KindProse))
	if fs["not-x-but-y"] == 0 {
		t.Fatalf("want parallelism finding: %v", fs)
	}
}

func TestCleanText(t *testing.T) {
	text := "Apple quoted $755 for the repair in 2018. The failed part was a 50-cent filter.\n\nThe shop fixed it in 45 minutes."
	if fs := Check(text, KindProse); len(fs) != 0 {
		t.Fatalf("clean text flagged: %+v", fs)
	}
}

func TestEmojiVariation(t *testing.T) {
	fs := rules(Check("done ✅ ok ➜ end", KindProse))
	if fs["emoji"] == 0 {
		t.Fatal("check emoji/dingbat detection")
	}
}

func TestKindForPath(t *testing.T) {
	cases := map[string]Kind{
		"README.md": KindDoc, "a.rst": KindDoc, "x.go": KindComment,
		"y.py": KindComment, "z.vue": KindDoc, "noext": KindProse,
	}
	for p, want := range cases {
		if got := KindForPath(p); got != want {
			t.Fatalf("%s: want %s got %s", p, want, got)
		}
	}
}

func TestCheckDiff(t *testing.T) {
	diff := `@@ -1,3 +1,4 @@
 context
-removed — line
+added — bad line
+another clean line
`
	fs := CheckDiff(diff, KindDoc)
	if len(fs) == 0 {
		t.Fatal("want findings on added emdash line")
	}
	for _, f := range fs {
		if f.Line != 2 {
			t.Fatalf("finding should map to new line 2, got %+v", f)
		}
	}
	if fs := CheckDiff("@@ -1,1 +1,1 @@\n-old\n+new clean line\n", KindProse); len(fs) != 0 {
		t.Fatalf("clean diff flagged: %+v", fs)
	}
	if fs := CheckDiff("", KindProse); fs != nil {
		t.Fatal("empty diff should return nil")
	}
}
