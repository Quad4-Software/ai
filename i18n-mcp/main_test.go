// SPDX-License-Identifier: 0BSD
package main

import (
	"encoding/json"
	"testing"
)

func TestFlatten(t *testing.T) {
	raw := `{"a":{"b":{"c":"x"},"d":"y"},"e":1}`
	var m map[string]any
	json.Unmarshal([]byte(raw), &m)
	out := map[string]string{}
	flatten(m, "", out)
	if out["a.b.c"] != "x" || out["a.d"] != "y" || out["e"] != "1" {
		t.Fatalf("flatten: %v", out)
	}
}

func TestHardcodedRegexes(t *testing.T) {
	if m := tmplText.FindStringSubmatch(`<p>Save changes</p>`); m == nil || m[1] != "Save changes" {
		t.Fatal("template text not detected")
	}
	if tmplText.MatchString(`<p>{{ $t('x.y') }}</p>`) {
		t.Fatal("i18n binding should not match")
	}
	if m := attrLit.FindStringSubmatch(`<input placeholder="Enter name">`); m == nil {
		t.Fatal("attr literal not detected")
	}
	if !i18nCall.MatchString(`{{ $t('a.b') }}`) || !i18nCall.MatchString(`t("k")`) {
		t.Fatal("i18n call not detected")
	}
}
