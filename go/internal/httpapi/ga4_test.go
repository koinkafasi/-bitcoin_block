package httpapi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidGA4MeasurementKabulEdilenler(t *testing.T) {
	cases := map[string]string{
		"G-XK4YN7T0PE":   "G-XK4YN7T0PE",
		"  G-ABCD1234 ":  "G-ABCD1234",
		"g-xk4yn7t0pe":   "G-XK4YN7T0PE", // küçük harf girilirse düzeltilir
		"G-1234567890AB": "G-1234567890AB",
	}
	for in, want := range cases {
		assert.Equal(t, want, validGA4Measurement(in), "girdi: %q", in)
	}
}

func TestValidGA4MeasurementReddedilenler(t *testing.T) {
	for _, in := range []string{
		"",
		"   ",
		"UA-12345-1",               // eski Universal Analytics
		"GTM-ABC123",               // Tag Manager konteyneri, ölçüm kimliği değil
		"G-",                       // gövdesiz
		"G-AB",                     // çok kısa
		"G-XK4YN7T0PE\"></script>", // enjeksiyon denemesi
		"G-XK4 YN7",                // boşluk
		"G-XK4YN7T0PE;alert(1)",
		"561087489", // mülk kimliği, ölçüm kimliği değil
	} {
		assert.Equal(t, "", validGA4Measurement(in), "girdi: %q reddedilmeliydi", in)
	}
}

func TestPublicHeadOlcumKimligiYoksaKodBasmaz(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData() // GA4Measurement boş
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "public_head", d))
	out := buf.String()
	assert.NotContains(t, out, "googletagmanager.com", "ayar boşken izleme kodu basılmamalı")
	assert.NotContains(t, out, "gtag(")
}

func TestPublicHeadOlcumKimligiVarsaRizaKapisiBasar(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.GA4Measurement = "G-XK4YN7T0PE"
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "public_head", d))
	out := buf.String()

	assert.NotContains(t, out, "googletagmanager.com",
		"rıza alınmadan analitik script sunucudan basılmamalı")
	assert.NotContains(t, out, "gtag(",
		"gtag çağrısı sayfaya gömülmemeli")

	assert.Contains(t, out, `data-ga4="G-XK4YN7T0PE"`,
		"ölçüm kimliği betiğe veri özniteliğiyle geçmeli")
	assert.Contains(t, out, "defer", "onay betiği sayfayı bloklamamalı")
}

func TestNormalizeGA4PropertyIDBiciminiTemizler(t *testing.T) {
	cases := map[string]string{
		"561087489":            "561087489",
		" 561087489 ":          "561087489",
		"properties/561087489": "561087489", // API biçimi yapıştırılırsa
		"":                     "",
		"abc":                  "",
		"properties/":          "",
		"561087489/extra":      "",
	}
	for in, want := range cases {
		assert.Equal(t, want, normalizeGA4PropertyID(in), "girdi: %q", in)
	}
}
