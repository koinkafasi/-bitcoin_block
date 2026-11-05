package httpapi

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui/templkit"
)

// Künye alanları panelden kaydedilmeliydi: kod onları settings "kunye"
// bölümünden okuyordu ama yazacak hiçbir arayüz yoktu. Bu testler
// kaydetmenin alan adlarını ve "boş bırakma = iletişim yedeğine dön"
// davranışını kilitler.

// toplananKunye sahte bir settings yazıcısı: hangi anahtara ne yazıldığını
// tutar, böylece DB olmadan kaydetme sözleşmesi doğrulanabilir.
func toplananKunye() (func(section, key string, v any), map[string]any) {
	yazilan := map[string]any{}
	return func(section, key string, v any) {
		yazilan[section+"/"+key] = v
	}, yazilan
}

func TestKunyeAyarKaydetme(t *testing.T) {
	form := url.Values{
		"kunye_ticaret_unvani":        {"TheBlock Medya A.Ş."},
		"kunye_tuzel_kisi_temsilcisi": {"Alper Ayyıldız"},
		"kunye_yayinci":               {"TheBlock"},
		"kunye_sorumlu_yim":           {"  Alper Ayyıldız  "}, // boşluklar kırpılmalı
		"kunye_uets":                  {"1234567890"},
		"kunye_yer_saglayici_unvan":   {"Cloudflare, Inc."},
	}
	r := httptest.NewRequest("POST", "/admin/ayarlar", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	set, yazilan := toplananKunye()
	s := &Server{}
	s.saveKunyeSettings(r, set)

	assert.Equal(t, "TheBlock Medya A.Ş.", yazilan["kunye/ticaret_unvani"])
	assert.Equal(t, "Alper Ayyıldız", yazilan["kunye/sorumlu_yim"], "baş/son boşluk kırpılmalı")
	assert.Equal(t, "Cloudflare, Inc.", yazilan["kunye/yer_saglayici_unvan"])
	// Formda olmayan alan yazılmamalı: başka bir sekme kaydedilirken
	// künyenin sessizce silinmesini bu kural engeller.
	_, var_mi := yazilan["kunye/telefon"]
	assert.False(t, var_mi, "formda olmayan künye alanı yazılmamalı")
}

// Telefon/e-posta/adres bilerek BOŞ gönderilir: kaydetme bunları boş dize
// olarak yazmalı (anahtarı silmemeli), böylece canlıda iletişim ayarından
// gelen yedek yeniden devreye girer.
func TestKunyeAyarBosDegerYedekBirakir(t *testing.T) {
	form := url.Values{"kunye_telefon": {"   "}}
	r := httptest.NewRequest("POST", "/admin/ayarlar", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	set, yazilan := toplananKunye()
	(&Server{}).saveKunyeSettings(r, set)

	deger, ok := yazilan["kunye/telefon"]
	require.True(t, ok, "alan formda varsa yazılmalı")
	assert.Equal(t, "", deger, "boşluk kırpılıp boş dize yazılmalı")
}

// Panel formu, kayıtlı değeri ve varsa iletişim yedeğini göstermeli.
// Ayrıca on alanın hepsi tek bir yerde listelenmeli ki yeni bir zorunlu
// alan eklendiğinde panel sessizce eksik kalmasın.
func TestKunyePaneliOnAlaniGosterir(t *testing.T) {
	sc := &settingsData{
		Kunye:      bikKunyeAlanlari(),
		KunyeEksik: []string{"UETS"},
		KunyeYedek: map[string]string{"telefon": "+90 212 000 00 00"},
	}
	for i := range sc.Kunye {
		if sc.Kunye[i].Key == "ticaret_unvani" {
			sc.Kunye[i].Value = "TheBlock Medya A.Ş."
		}
	}

	v := ayarlarViewFrom(sc, nil, nil)
	require.Len(t, v.Kunye, 10, "tebliğ on zorunlu alan istiyor")

	var bulunan templkit.AyarKunye
	for _, a := range v.Kunye {
		if a.Key == "ticaret_unvani" {
			bulunan = a
		}
	}
	assert.Equal(t, "TheBlock Medya A.Ş.", bulunan.Value)
	assert.Equal(t, "Yayın Sahibi Ticaret Unvanı", bulunan.Label)

	for _, a := range v.Kunye {
		if a.Key == "telefon" {
			assert.Equal(t, "+90 212 000 00 00", a.Yedek,
				"iletişim yedeği alan altında gösterilmeli")
		}
	}
	assert.Equal(t, []string{"UETS"}, v.KunyeEksik)
}
