package httpapi

import (
	"bytes"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderRedirectYalnizBilinenHedefler(t *testing.T) {
	cases := map[string]string{
		"abonelik":             "/abonelik",
		"hesabim":              "/hesabim",
		"":                     "/",
		"https://kotu.example": "/",
		"//kotu.example":       "/",
		"javascript:alert(1)":  "/",
	}
	for in, want := range cases {
		assert.Equal(t, want, readerRedirect(in), "girdi: %q", in)
	}
}

func TestBotTrapHoneypotVeZamanTuzagi(t *testing.T) {
	r := httptest.NewRequest("POST", "/giris", strings.NewReader(url.Values{
		"website": []string{"http://spam.example"},
	}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())
	assert.False(t, testServer().botTrapOK(r), "honeypot doluysa bot sayılmalı")

	r2 := httptest.NewRequest("POST", "/giris", nil)
	assert.True(t, testServer().botTrapOK(r2), "boş form geçmeli")
}

func TestMemberInitial(t *testing.T) {
	assert.Equal(t, "A", memberInitial("alper"))
	assert.Equal(t, "A", memberInitial("  Alper Yılmaz "))
	assert.Equal(t, "Ç", memberInitial("çetin"))
	assert.Equal(t, "?", memberInitial("   "))
}

func testServer() *Server { return &Server{} }

func headerData() *tmplData {
	return &tmplData{
		SiteName: "Test", SiteSlogan: "Slogan", HomeURL: "/", LangBase: "",
		Header: &HeaderConfig{
			Model: "model1", Skin: "auto", Width: "normal",
			ShowSearch: true, ShowTheme: true, ShowMenu: true,
		},
		Footer: &FooterConfig{Layout: "sade", LogoPos: "center"},
	}
}

func TestHeaderUyeSimgesiVarsayilanGorunur(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "site_header", headerData()))
	out := buf.String()
	assert.Contains(t, out, "member-btn")
	assert.Contains(t, out, "Üye girişi")
	assert.NotContains(t, out, "member-avatar")
}

func TestHeaderUyeSimgesiGirisliAvatar(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.MemberLoggedIn = true
	d.MemberName = "Ayşe"
	d.MemberInitial = "A"
	d.MemberURL = "/hesabim"
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "site_header", d))
	out := buf.String()
	assert.Contains(t, out, "member-avatar")
	assert.Contains(t, out, ">A</span>")
	assert.Contains(t, out, `href="/hesabim"`)
	assert.NotContains(t, out, "Üye girişi")
}

func TestHeaderUyeSimgesiKapatilabilir(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.Header.HideMember = true
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "site_header", d))
	assert.NotContains(t, buf.String(), "member-btn")
}

func TestUyeGirisSablonu(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.MetaTitle = "Giriş Yap"
	d.UyeMode = "giris"
	d.UyeError = "Hatalı giriş"
	d.UyeFormTS = "2026-01-01T00:00:00Z"
	d.GoogleEnabled = true
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "uye-giris", d))
	out := buf.String()
	assert.Contains(t, out, `action="/giris"`)
	assert.Contains(t, out, "Hatalı giriş")
	assert.Contains(t, out, "Google ile devam et")
	assert.Contains(t, out, "Şifremi unuttum")
	assert.Contains(t, out, "name=\"website\"")
}

func TestUyeKayitSablonuKVKKOnayi(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.MetaTitle = "Kayıt Ol"
	d.UyeMode = "kayit"
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "uye-kayit", d))
	out := buf.String()
	assert.Contains(t, out, `action="/kayit"`)
	assert.Contains(t, out, `name="terms"`)
	assert.Contains(t, out, "Kullanım Koşulları")
	assert.Contains(t, out, "KVKK")
}
