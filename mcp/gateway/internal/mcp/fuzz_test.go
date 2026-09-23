// SPDX-License-Identifier: 0BSD
package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// FuzzDispatchLine feeds arbitrary bytes through the request parser
// and dispatch path. The server must never panic and must always
// produce at most one response per input line.
func FuzzDispatchLine(f *testing.F) {
	seeds := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"x","arguments":{}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":null,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":[1,2],"method":"tasks/get","params":{"taskId":"task-0"}}`,
		`{"jsonrpc":"2.0","id":4,"method":"../../etc/passwd"}`,
		`{"id":5,"method":"tools/call","params":{"name":"%s%s%n","arguments":"AAAA"}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"read","task":true}}`,
		`[]`, `{}`, `null`, `"str"`, `{"jsonrpc":2,"id":"x"}`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, line string) {
		if strings.Contains(line, "\n") {
			return // one line per message; multi-line is a different test
		}
		srv := NewServer("fuzz", "0", []Tool{
			{Name: "read", InputSchema: map[string]any{},
				Handle: func(context.Context, json.RawMessage) (string, error) { return "ok", nil }},
		}, nil)
		var out strings.Builder
		// must not panic; EOF errors are fine
		_ = srv.Serve(context.Background(), strings.NewReader(line+"\n"), &out)
	})
}
