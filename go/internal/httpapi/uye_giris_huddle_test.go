package httpapi

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUyeGirisHuddleKabugu(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.MetaTitle = "Giriş Yap"
	d.UyeMode = "giris"
	d.UyeRedirect = "hesabim"
	d.UyeKayitAcik = true

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "uye-giris", d))
	out := buf.String()

	assert.Contains(t, out, `class="auth"`)
	assert.Contains(t, out, "data-auth")
	assert.Contains(t, out, `data-mood="idle"`)
	assert.Contains(t, out, "data-huddle")
	assert.Equal(t, 4, strings.Count(out, `class="mate mate--`),
		"dört karakter: kırmızı, mürekkep, amber, mercan")
	assert.Contains(t, out, `id="password"`)
	assert.Contains(t, out, "data-peek")
	assert.Contains(t, out, `aria-pressed="false"`)
	assert.Contains(t, out, `action="/giris"`)
	assert.Contains(t, out, `href="/kayit?redirect=hesabim"`)
	assert.Contains(t, out, "data-live", "canlı bölge olmalı")
	assert.Contains(t, out, "Caps Lock")
}

func TestUyeGirisSunucuHatasindaYanlisDurum(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.UyeMode = "giris"
	d.UyeError = "E-posta veya şifre hatalı."

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "uye-giris", d))
	out := buf.String()

	assert.Contains(t, out, `data-mood="wrong"`)
	assert.Contains(t, out, "E-posta veya şifre hatalı.")
}

func TestUyeGirisGoogleYalnizYapilandirilinca(t *testing.T) {
	tmpl := parseTestTemplates(t)

	kapali := headerData()
	kapali.UyeMode = "giris"
	var b1 bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&b1, "uye-giris", kapali))
	assert.NotContains(t, b1.String(), "giris/google",
		"kimlik bilgisi yokken Google düğmesi görünmemeli")

	acik := headerData()
	acik.UyeMode = "giris"
	acik.GoogleEnabled = true
	acik.UyeRedirect = "abonelik"
	var b2 bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&b2, "uye-giris", acik))
	out := b2.String()
	assert.Contains(t, out, "Google ile devam et")
	assert.Contains(t, out, `href="/giris/google?redirect=abonelik"`)
}

func TestUyeKayitKapaliHalaKabukIcinde(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.UyeMode = "kayit-kapali"

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "uye-kayit", d))
	out := buf.String()

	assert.Contains(t, out, "data-auth", "kayıt kapalı da aynı kabuğu kullanmalı")
	assert.Contains(t, out, "kayıtları şu anda kapalı")
	assert.NotContains(t, out, `action="/kayit"`)
}

func TestUyeGirisVarliklariBagli(t *testing.T) {
	for _, ad := range []string{"uye-giris.css", "uye-giris.js"} {
		_, err := os.Stat(filepath.Join("..", "webui", "static", ad))
		require.NoError(t, err, "%s gömülü statik klasörde olmalı", ad)
	}
	ham, err := os.ReadFile(filepath.Join("..", "webui", "templates", "site_uye.html"))
	require.NoError(t, err)
	kaynak := string(ham)
	assert.Contains(t, kaynak, `{{asset "uye-giris.css"}}`)
	assert.Contains(t, kaynak, `{{asset "uye-giris.js"}}`)
}

func TestUyeGirisKayitKapaliykenBaglantiYok(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.UyeMode = "giris"
	d.UyeKayitAcik = false

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "uye-giris", d))
	assert.NotContains(t, buf.String(), "/kayit?redirect=",
		"kayıt kapalıyken giriş sayfası ölü bağlantı vermemeli")
}

func TestUyeSifreAkisiAyniKabukta(t *testing.T) {
	tmpl := parseTestTemplates(t)
	cases := []struct {
		sablon, mod, baslik string
	}{
		{"uye-sifremi-unuttum", "sifremi-unuttum", "Şifremi unuttum"},
		{"uye-sifre-sifirla", "sifre-sifirla", "Yeni şifre belirle"},
		{"uye-sifre-tamam", "sifre-tamam", "Şifreniz güncellendi"},
		{"uye-onay", "onay", "E-posta doğrulanamadı"},
	}
	for _, c := range cases {
		d := headerData()
		d.UyeMode = c.mod
		var buf bytes.Buffer
		require.NoError(t, tmpl.ExecuteTemplate(&buf, c.sablon, d), c.sablon)
		out := buf.String()
		assert.Contains(t, out, "data-auth", "%s kabuğu kullanmalı", c.sablon)
		assert.Equal(t, 4, strings.Count(out, `class="mate mate--`), c.sablon)
		assert.Contains(t, out, c.baslik, c.sablon)
		assert.Contains(t, out, `class="acta"`, "%s tek ana düğme", c.sablon)
	}
}

func TestUyeOnayIkiDurum(t *testing.T) {
	tmpl := parseTestTemplates(t)

	basarili := headerData()
	basarili.UyeMode = "onay"
	basarili.UyeOK = true
	basarili.UyeInfo = "E-posta adresiniz doğrulandı. Teşekkürler!"
	var b1 bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&b1, "uye-onay", basarili))
	out1 := b1.String()
	assert.Contains(t, out1, "E-posta doğrulandı")
	assert.NotContains(t, out1, "panel__ok--hata")
	assert.Contains(t, out1, `href="/hesabim"`)

	hatali := headerData()
	hatali.UyeMode = "onay"
	hatali.UyeInfo = "Bağlantı geçersiz veya süresi dolmuş."
	var b2 bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&b2, "uye-onay", hatali))
	out2 := b2.String()
	assert.Contains(t, out2, "panel__ok--hata",
		"geçersiz bağlantıda yeşil tik çizilmemeli")
	assert.Contains(t, out2, "E-posta doğrulanamadı")
	assert.Contains(t, out2, `href="/giris"`)
	assert.NotContains(t, out2, `class="panel__ok" aria-hidden="true"`)
}

func TestUyeSifreSifirlaAlanlari(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.UyeMode = "sifre-sifirla"
	d.UyeToken = "tok123"

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "uye-sifre-sifirla", d))
	out := buf.String()

	assert.Contains(t, out, `name="token" value="tok123"`)
	assert.Contains(t, out, `id="password"`)
	assert.Contains(t, out, `id="password2"`)
	assert.Contains(t, out, `data-err="password2"`)
	assert.Contains(t, out, `aria-controls="password"`, "göz düğmesi bağlı olmalı")
}

func TestGoogleDonusAnahtari(t *testing.T) {
	cases := map[string]string{
		"hesabim":              "hesabim",
		"abonelik":             "abonelik",
		"":                     "",
		"https://kotu.example": "",
		"//kotu.example":       "",
		"admin":                "",
	}
	for in, want := range cases {
		assert.Equal(t, want, googleDonusAnahtari(in), "girdi: %q", in)
	}
}
