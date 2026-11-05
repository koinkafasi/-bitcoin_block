// Package migrations P0 çekirdek şema SQL dosyalarını embed eder.
// Kural: migration SAHİBİ Go'dur (spec 03 §3) — şema go/migrations altında, uygulama Go cmd/migrate ile.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
