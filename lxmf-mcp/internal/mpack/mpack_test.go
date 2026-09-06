// SPDX-License-Identifier: 0BSD
package mpack

import (
	"bytes"
	"testing"
)

func TestDecodeMap(t *testing.T) {
	// {bin(2)\xde\xad: [1.5, "hi", nil, 0xc0-blessed], "k": 7}
	// hand-encoded: fixmap(2), bin8 key, fixarray(3) [float64, str, nil], fixstr k, fixint 7
	data := []byte{
		0x82,
		0xc4, 0x02, 0xde, 0xad,
		0x93,
		0xcb, 0x3f, 0xf8, 0, 0, 0, 0, 0, 0, // 1.5
		0xa2, 'h', 'i',
		0xc0,
		0xa1, 'k',
		0x07,
	}
	v, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	m, ok := v.(map[any]any)
	if !ok || len(m) != 2 {
		t.Fatalf("map: %v", v)
	}
	var found []any
	for k, val := range m {
		if ks, ok := k.(string); ok && len(ks) == 2 && ks[0] == 0xde && ks[1] == 0xad {
			found = val.([]any)
		}
	}
	if len(found) != 3 || found[0].(float64) != 1.5 || found[1] != "hi" || found[2] != nil {
		t.Fatalf("array: %v", found)
	}
	if m["k"] != int64(7) {
		t.Fatalf("k: %v", m["k"])
	}
}
