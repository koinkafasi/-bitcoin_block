package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---- Faz 145: iletişim bilgileri / yer tutucular -------------------

func TestContactDefaultsDomain(t *testing.T) {
	c := contactDefaults("https://Yeni-AlanAdi.com.tr/yol?x=1")
	assert.Equal(t, "yeni-alanadi.com.tr", c.Domain, "şema/yol/küçük harf sadeleşmeli")
	assert.Equal(t, "info@yeni-alanadi.com.tr", c.Info)
	assert.Equal(t, "haber@yeni-alanadi.com.tr", c.Haber)
	assert.Equal(t, "reklam@yeni-alanadi.com.tr", c.Reklam)
	assert.Equal(t, "telif@yeni-alanadi.com.tr", c.Telif)
	assert.Equal(t, "kariyer@yeni-alanadi.com.tr", c.Kariyer)
	assert.True(t, c.FormAktif, "form varsayılan açık")
}

func TestContactDefaultsBosDomain(t *testing.T) {
	c := contactDefaults("   ")
	assert.Equal(t, varsayilanDomain, c.Domain, "boş alan adı sabit varsayılana düşer")
	assert.Equal(t, "info@"+varsayilanDomain, c.Info)
}

func TestNormalizeContactConfigBosluklariDoldurur(t *testing.T) {
	c := normalizeContactConfig(ContactConfig{Domain: "ornek.com", Info: "  ozel@baska.com "})
	assert.Equal(t, "ornek.com", c.Domain)
	assert.Equal(t, "ozel@baska.com", c.Info, "özel adres korunur (kırpılır)")
	assert.Equal(t, "haber@ornek.com", c.Haber, "boş alan alan adından üretilir")
	assert.Equal(t, "ozel@baska.com", c.KVKK, "KVKK boşsa genel iletişime düşer")
}

func TestContactTokensYerTutucu(t *testing.T) {
	c := contactDefaults("bitcosar.com")
	c.Telefon = "+90 212 000 00 00"
	tok := contactTokens(c)
	assert.Equal(t, "bitcosar.com", tok["domain"])
	assert.Equal(t, "info@bitcosar.com", tok["eposta_info"])
	assert.Equal(t, "info@bitcosar.com", tok["eposta"], "{eposta} kısa takma ad")
	assert.Equal(t, "+90 212 000 00 00", tok["telefon"])

	in := "Bize {eposta_info} veya {eposta_reklam} adresinden yazın. Site: {domain}. Tel: {telefon}"
	out := pageTextVars(in, tok)
	assert.Contains(t, out, "info@bitcosar.com")
	assert.Contains(t, out, "reklam@bitcosar.com")
	assert.Contains(t, out, "bitcosar.com")
	assert.Contains(t, out, "+90 212 000 00 00")
	assert.NotContains(t, out, "{", "bilinen yer tutucu kalmamalı")
}

func TestContactTokensBilinmeyenYerTutucuKorunur(t *testing.T) {
	tok := contactTokens(contactDefaults("ornek.com"))
	out := pageTextVars("Bilinmeyen {yok_boyle} kalır", tok)
	assert.Contains(t, out, "{yok_boyle}")
}

func TestValidEmail(t *testing.T) {
	for _, s := range []string{"a@b.com", "info@theblock.com.tr", "x.y+z@a.co"} {
		assert.True(t, validEmail(s), s)
	}
	for _, s := range []string{"", "abc", "a@", "@b.com", "a b@c.com", "a@@b.com"} {
		assert.False(t, validEmail(s), s)
	}
}

func TestContactRateLimit(t *testing.T) {
	ip := "203.0.113.9"
	assert.False(t, contactRateLimited(ip), "ilk istek serbest")
	assert.True(t, contactRateLimited(ip), "aynı IP kısa sürede ikinci kez engellenir")
	assert.False(t, contactRateLimited("203.0.113.10"), "başka IP etkilenmez")
}

func TestContactTopicsBenzersiz(t *testing.T) {
	seen := map[string]bool{}
	for _, tp := range contactTopics() {
		assert.False(t, seen[tp.Key], "yinelenen konu: "+tp.Key)
		assert.NotEmpty(t, tp.Name)
		seen[tp.Key] = true
	}
	assert.True(t, seen["diger"], "'diger' konusu bulunmalı")
}

// applyContactTokensWith DB gerektirmeyen saf yol üzerinden sınanır.
func TestApplyContactTokensWith(t *testing.T) {
	c := contactDefaults("theblock.com.tr")
	got := applyContactTokensWith("Yazın: {eposta_info} · {domain}", c)
	assert.Equal(t, "Yazın: info@theblock.com.tr · theblock.com.tr", got)
	assert.Equal(t, "yer tutucu yok", applyContactTokensWith("yer tutucu yok", c))
}

func TestNormalizePageURLSayfaKoku(t *testing.T) {
	cases := map[string]string{
		"/sayfa/kvkk":       "/kvkk",
		"/en/sayfa/kunye":   "/en/kunye",
		"/kvkk":             "/kvkk",
		"/kategori/ekonomi": "/kategori/ekonomi",
		"":                  "",
	}
	for in, want := range cases {
		assert.Equal(t, want, normalizePageURL(in), in)
	}
}

func TestContactDomainChangeAdresleriTasir(t *testing.T) {
	old := contactDefaults("eski.com")
	// Bir adres bilinçli olarak başka alan adında (kurumsal sağlayıcı).
	old.KVKK = "kvkk@hukukfirmasi.com"
	neu := old
	neu.Domain = "yeni.com"
	got := shiftEmailsToDomain(neu, old.Domain)

	assert.Equal(t, "info@yeni.com", got.Info)
	assert.Equal(t, "haber@yeni.com", got.Haber)
	assert.Equal(t, "reklam@yeni.com", got.Reklam)
	assert.Equal(t, "kvkk@hukukfirmasi.com", got.KVKK, "başka alan adlı özel adres korunur")

	// Alan adı değişmediyse hiçbir şey taşınmaz.
	same := shiftEmailsToDomain(got, "yeni.com")
	assert.Equal(t, "info@yeni.com", same.Info)
}
