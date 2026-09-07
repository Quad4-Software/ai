// SPDX-License-Identifier: 0BSD
// Package community fetches community sources: rns.recipes interface
// directory and the Reticulum miraheze wiki.
package community

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Exported bounds so callers can keep schemas in sync.
const (
	DirectoryURL          = "https://directory.rns.recipes/api/directory/submitted"
	DirectoryDefaultLimit = 50
	DirectoryMaxLimit     = 200
)

// DirectoryEntry is one interface returned by directory.rns.recipes.
type DirectoryEntry struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	TypeName string `json:"typeName"`
	Network  string `json:"network"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Status   string `json:"status"`
}

// DirectorySearch queries the public rns.recipes interface directory.
func DirectorySearch(ctx context.Context, search, ifaceType, status string, limit int) (string, error) {
	if limit <= 0 || limit > DirectoryMaxLimit {
		limit = DirectoryDefaultLimit
	}
	q := url.Values{}
	q.Set("search", search)
	q.Set("type", ifaceType)
	q.Set("status", status)
	q.Set("limit", strconv.Itoa(limit))
	u := DirectoryURL + "?" + q.Encode()

	body, err := get(ctx, u)
	if err != nil {
		return "", err
	}
	var res struct {
		Data []DirectoryEntry `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		return "", fmt.Errorf("decode directory: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "found %d interface(s)\n", len(res.Data))
	for i, e := range res.Data {
		if i >= limit {
			fmt.Fprintf(&b, "... %d more\n", len(res.Data)-limit)
			break
		}
		fmt.Fprintf(&b, "%s [%s/%s] %s:%d status=%s (id=%d)\n", e.Name, e.Type, e.TypeName, e.Host, e.Port, e.Status, e.ID)
	}
	return b.String(), nil
}
