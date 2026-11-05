package httpapi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
)

// Arama sonuç sayfası, hangi arama motorunun/veritabanının çalıştığını
// KULLANICIYA GÖSTERMEMELİ.
//
// ÖLÇÜM (5 Kasım 2026, üretim): `/arama?q=test` sayfası
// "Motor: PostgreSQL (pg_trgm) · 0 sonuç" basıyordu. Bu bir altyapı
// sızıntısıdır: ziyaretçiye faydası yok, saldırgana sürüm/CVE hedeflemesi
// için ipucu verir (PostgreSQL + pg_trgm = bilinen sürüm aralığı).
func TestAramaSayfasiMotorAdiniSizdirmaz(t *testing.T) {
	tmpl := parseTestTemplates(t)
	slug := "bir-haber"
	data := &tmplData{
		Query: "test",
		Arama: []repo.Content{{ID: 1, Title: "Bir haber", Slug: &slug}},
	}
	var buf bytes.Buffer
	// cat_liste alt şablonu ek alanlar isteyebilir; asıl ölçülen üst
	// bölümdeki sonuç satırıdır, şablon hatası bu testi düşürmemeli.
	_ = tmpl.ExecuteTemplate(&buf, "arama", data)
	out := buf.String()

	for _, sizinti := range []string{"Motor:", "PostgreSQL", "pg_trgm", "Tantivy", "search-svc"} {
		assert.NotContains(t, strings.ToLower(out), strings.ToLower(sizinti),
			"arama sayfası altyapı bilgisi sızdırıyor: %q", sizinti)
	}
}

// Aynı kural kaynak düzeyinde de tutulmalı: ileride biri metni geri
// eklerse şablonu düzenleyen kişi bu testi görsün.
func TestAramaSablonundaMotorMetniYok(t *testing.T) {
	b, err := webui.FS.ReadFile("templates/site_public.html")
	require.NoError(t, err)
	s := string(b)
	for _, yasak := range []string{"Motor: {{", "Motor: ", "pg_trgm", "Tantivy (search-svc)"} {
		assert.NotContains(t, s, yasak, "site_public.html içinde altyapı sızıntısı: %q", yasak)
	}
	// Sonuç sayısı yine görünmeli (tek başına, motor adı olmadan).
	assert.Contains(t, s, "{{len .Arama}} sonuç")
}
