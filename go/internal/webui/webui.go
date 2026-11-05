// Package webui site + panel şablonlarını ve statik varlıkları embed eder.
package webui

import "embed"

//go:embed templates static
var FS embed.FS
