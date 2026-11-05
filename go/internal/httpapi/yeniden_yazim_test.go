package httpapi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// Faz 204 — yeniden yazım paneli şablonu.
//
// Panelin işi eski/yeni metni YAN YANA göstermek ve onay/red düğmelerini
// yalnız uygun durumda basmak. Şablon hatası bu ekranı tamamen düşürürdü;
// veritabanı gerektirmeyen bu test yapıyı kilitler.
func TestYenidenYazimPanelSablonu(t *testing.T) {
	tmpl := parseTestTemplates(t)

	data := &tmplData{Yeniden: &yenidenData{
		AdaySayisi: 982,
		PilotAdet:  "10",
		Toplam:     2,
		Ozet: map[string]int{
			repo.YenidenBekliyor: 0, repo.YenidenKuyrukta: 0,
			repo.YenidenHazir: 1, repo.YenidenHata: 1,
		},
		Satirlar: []yenidenView{
			{
				ID: 7, ContentID: 71, Baslik: "Hizmet üretim endeksi arttı",
				Locale: "tr", Mod: repo.YenidenModArastirmali, ModEtiket: "Araştırmalı",
				Durum: repo.YenidenHazir, DurumEtiket: "Onay bekliyor",
				DurumSinif: "hazir", EskiParagrafN: 1, YeniParagrafN: 4,
				EskiKarakter: 120, YeniKarakter: 480,
				EskiParagraf: []string{"Tek paragraf."},
				YeniParagraf: []string{"Birinci paragraf.", "İkinci paragraf.", "Üçüncü paragraf.", "Dördüncü paragraf."},
				Uyarilar:     []string{"gövde 480 karakter; haber standardı 700."},
				Kaynaklar: []repo.YenidenKaynak{
					{Ref: "S1", Baslik: "TÜİK: Hizmet üretim endeksi arttı", URL: "https://ornek.com/haber", Yayinci: "Örnek"},
				},
				Olusturma: "03.11.2026 10:15",
			},
			{
				ID: 8, ContentID: 72, Baslik: "Kısa kalan öneri",
				Locale: "en", Durum: repo.YenidenHata, DurumEtiket: "Hata",
				DurumSinif: "hata", EskiParagrafN: 1, EskiKarakter: 100,
				Hata: "kalite: gövde 1 paragraf; en az 2 gerekir",
			},
		},
	}}

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "yeniden_yazim_panel", data))
	out := buf.String()

	assert.Contains(t, out, "AI ile yeniden yazım")
	assert.Contains(t, out, "Aday içerik")
	assert.Contains(t, out, "982")
	assert.Contains(t, out, "Pilotu başlat")
	assert.Contains(t, out, "Hizmet üretim endeksi arttı")
	assert.Contains(t, out, "Onay bekliyor")
	assert.Contains(t, out, "kalite: gövde 1 paragraf")
	assert.Contains(t, out, "Araştırmalı")
	assert.Contains(t, out, `name="mod"`)
	assert.Contains(t, out, `action="/admin/yeniden-yazim/7/onayla"`)
	assert.NotContains(t, out, `action="/admin/yeniden-yazim/8/onayla"`,
		"hata durumundaki öneride onay düğmesi olmamalı")
	// Emoji yasak (proje kuralı: TestAdminSablonlariEmojisiz).
	assert.NotContains(t, out, "👉")
}

// Detay ekranı: eski ve yeni metin birlikte, uyarılar görünür.
func TestYenidenYazimDetaySablonu(t *testing.T) {
	tmpl := parseTestTemplates(t)
	v := &yenidenView{
		ID: 9, ContentID: 90, Baslik: "Detay başlığı", Locale: "en",
		Durum: repo.YenidenHazir, DurumEtiket: "Onay bekliyor", DurumSinif: "hazir",
		Surum: 4, Olusturma: "03.11.2026 10:20",
		EskiSpot: "Eski spot", YeniSpot: "Yeni spot",
		EskiParagraf:  []string{"Eski gövde paragrafı."},
		YeniParagraf:  []string{"New first paragraph.", "New second paragraph."},
		EskiParagrafN: 1, YeniParagrafN: 2, EskiKarakter: 30, YeniKarakter: 400,
		EskiSeoBaslik: "Eski SEO", YeniSeoBaslik: "New SEO",
		Uyarilar: []string{"gövde 400 karakter; haber standardı 700."},
		Provider: "omniroute", Model: "auto/best-chat", TokenGirdi: 300, TokenCikti: 1800,
		Mod: repo.YenidenModArastirmali, ModEtiket: "Araştırmalı",
		Kaynaklar: []repo.YenidenKaynak{
			{Ref: "S1", Baslik: "Kaynak başlığı", URL: "https://ornek.com/kaynak", Yayinci: "Örnek Yayın", YayinTari: "2026-11-03"},
		},
	}
	data := &tmplData{Yeniden: &yenidenData{Detay: v, Ozet: map[string]int{}, AdaySayisi: 982}}

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "yeniden_yazim_panel", data))
	out := buf.String()

	assert.Contains(t, out, "Şu anki metin")
	assert.Contains(t, out, "AI önerisi")
	assert.Contains(t, out, "Eski gövde paragrafı.")
	assert.Contains(t, out, "New first paragraph.")
	assert.Contains(t, out, "Onayla ve yayına yaz")
	assert.Contains(t, out, "Reddet")
	assert.Contains(t, out, "300 girdi / 1800 çıktı token")
	assert.Contains(t, out, "Kullanılan kaynaklar (1)")
	assert.Contains(t, out, "https://ornek.com/kaynak")
	assert.Contains(t, out, "Örnek Yayın")
}
