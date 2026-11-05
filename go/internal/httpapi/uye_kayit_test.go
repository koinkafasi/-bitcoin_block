package httpapi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUyeKayitKapaliSablonu(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.MetaTitle = "Kayıt Ol"
	d.UyeMode = "kayit-kapali"
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "uye-kayit", d))
	out := buf.String()
	assert.Contains(t, out, "kayıtları şu anda kapalı")
	assert.NotContains(t, out, `action="/kayit"`)
	assert.Contains(t, out, "Giriş Yap")
}

func TestUyeKayitAcikFormu(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.MetaTitle = "Kayıt Ol"
	d.UyeMode = "kayit"
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "uye-kayit", d))
	assert.Contains(t, buf.String(), `action="/kayit"`)
}
