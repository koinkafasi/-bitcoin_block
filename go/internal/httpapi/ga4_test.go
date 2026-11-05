package httpapi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GA4 ölçüm kodu (gtag.js) entegrasyonu.
//
// Durum tespiti (28 Ekim 2026): canlı sitede hiçbir analytics kodu yoktu —
// ne G- ölçüm kimliği, ne GTM, ne gtag.js, ne eski UA-. GA4 mülkü açıldı
// (561087489 / G-XK4YN7T0PE) ama veri akması için sayfaya kod gerekiyor.
//
// İki kural:
//  1. Ölçüm kimliği BOŞSA sayfa hiçbir üçüncü taraf kodu basmaz. Varsayılan
//     izleme yok; yönetici bilinçli olarak açar.
//  2. Kimlik doğrulanmadan script etiketine yazılmaz. Ayar alanı panelden
//     geldiği için serbest metin; biçimi tutmayan değer sessizce yok sayılır.

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

// Ölçüm kimliği tanımlıyken bile sunucu gtag.js BASMAZ: kimlik yalnızca
// cerez.js'e data-ga4 ile geçer, script'i ziyaretçi kabul ederse betik
// istemci tarafında ekler.
//
// Bu test daha önce tam tersini doğruluyordu (kimlik varsa gtag.js basılsın).
// Davranış, Kişisel Verileri Koruma Kurulu'nun 08.08.2024 tarihli 2024/1361
// sayılı kararı nedeniyle değişti: kesinlikle gerekli olmayan bir analitik
// çerez için açık rıza aranır ve sitenin İLK GİRİŞİNDE rızayı alacak bir
// mekanizma bulunmaması ihlal sayıldı. Google Consent Mode yeterli değil;
// o yöntemde gtag.js yine yüklenir ve ziyaretçinin IP adresi Google'a
// aktarılır (6698 s.K. m.9).
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

	// Betiğin ADI burada doğrulanamaz: bu şablon testinde asset() sahte ve
	// boş dize döndürüyor, yani src="" basılıyor. cerez.js'in gerçekten
	// yüklendiği, asset() gerçek olan entegrasyon testinde kontrol ediliyor
	// (ucuncu_taraf_integration_test.go). Burada değişmez olan şey, ölçüm
	// kimliğinin script etiketine veri özniteliği olarak geçmesi.
	assert.Contains(t, out, `data-ga4="G-XK4YN7T0PE"`,
		"ölçüm kimliği betiğe veri özniteliğiyle geçmeli")
	assert.Contains(t, out, "defer", "onay betiği sayfayı bloklamamalı")
}

// Panel tarafı: mülk kimliği tek kaynaktan okunur. Geçmişte aynı değer
// trend/ga4_property ve adsense/ga_property olarak iki ayrı ayarda
// tutuluyordu; biri doldurulup diğeri unutulduğunda panelin bir yarısı
// "yapılandırılmadı" diyordu.
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
