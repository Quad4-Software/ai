// SPDX-License-Identifier: 0BSD
package templates

import (
	_ "embed"
)

//go:embed page.mu
var Page string

//go:embed color.mu
var Color string

//go:embed form.mu
var Form string

//go:embed table.mu
var Table string

//go:embed minimal.mu
var Minimal string

//go:embed reference.txt
var Reference string

// ByName maps template names to embedded starter sources.
var ByName = map[string]string{
	"page":    Page,
	"color":   Color,
	"form":    Form,
	"table":   Table,
	"minimal": Minimal,
}
