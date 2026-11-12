package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func faqGovde(bloklar string) []byte {
	return []byte(`{"type":"doc","content":[` + bloklar + `]}`)
}

func h2(metin string) string {
	return `{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"` + metin + `"}]}`
}

func h3(metin string) string {
	return `{"type":"heading","attrs":{"level":3},"content":[{"type":"text","text":"` + metin + `"}]}`
}

func par(metin string) string {
	return `{"type":"paragraph","content":[{"type":"text","text":"` + metin + `"}]}`
}

func TestFaqCikarSSSBolumunuBulur(t *testing.T) {
	govde := faqGovde(strings.Join([]string{
		par("Giriş paragrafı."),
		h2("Bitcoin nasıl çalışır?"),
		par("Gövde metni."),
		h2("Sık sorulan sorular"),
		h3("Bitcoin'in sahibi kim?"),
		par("Bitcoin'in sahibi yoktur."),
		h3("En küçük birim nedir?"),
		par("Satoshi."),
		par("Bir bitcoin 100 milyon satoshiye bölünür."),
	}, ","))

	sorular := faqCikar(govde)
	require.Len(t, sorular, 2)
	assert.Equal(t, "Bitcoin'in sahibi kim?", sorular[0].Soru)
	assert.Equal(t, "Bitcoin'in sahibi yoktur.", sorular[0].Cevap)
	// Aynı sorunun birden çok paragrafı tek cevapta birleşir.
	assert.Equal(t, "Satoshi. Bir bitcoin 100 milyon satoshiye bölünür.", sorular[1].Cevap)
}

func TestFaqCikarBolumDisiBasliklariAlmaz(t *testing.T) {
	// SSS bölümünden ÖNCEKİ H3'ler soru değildir.
	govde := faqGovde(strings.Join([]string{
		h2("Nasıl çalışır?"),
		h3("Blok ve zincir"),
		par("Teknik açıklama."),
		h2("Sık sorulan sorular"),
		h3("Soru bir?"),
		par("Cevap bir."),
		h3("Soru iki?"),
		par("Cevap iki."),
	}, ","))

	sorular := faqCikar(govde)
	require.Len(t, sorular, 2)
	assert.Equal(t, "Soru bir?", sorular[0].Soru)
}

func TestFaqCikarUstBaslikBolumuKapatir(t *testing.T) {
	// SSS'den sonra gelen H2, bölümü kapatır: sonraki paragraflar
	// son sorunun cevabına eklenmez.
	govde := faqGovde(strings.Join([]string{
		h2("Sık sorulan sorular"),
		h3("Soru bir?"),
		par("Cevap bir."),
		h3("Soru iki?"),
		par("Cevap iki."),
		h2("Kaynaklar"),
		par("Bu paragraf cevaba karışmamalı."),
	}, ","))

	sorular := faqCikar(govde)
	require.Len(t, sorular, 2)
	assert.Equal(t, "Cevap iki.", sorular[1].Cevap)
}

func TestFaqCikarTekSoruSemaBasmaz(t *testing.T) {
	// Google en az iki soru bekler; tek soruluk bölüm için şema basmak
	// yapısal veri uyarısı üretir.
	govde := faqGovde(strings.Join([]string{
		h2("Sık sorulan sorular"),
		h3("Tek soru?"),
		par("Tek cevap."),
	}, ","))

	assert.Empty(t, faqCikar(govde))
	assert.Empty(t, string(faqJSONLD(nil)))
}

func TestFaqCikarCevapsizSoruAtlanir(t *testing.T) {
	govde := faqGovde(strings.Join([]string{
		h2("Sık sorulan sorular"),
		h3("Cevabı yazılmamış soru?"),
		h3("Soru bir?"),
		par("Cevap bir."),
		h3("Soru iki?"),
		par("Cevap iki."),
	}, ","))

	sorular := faqCikar(govde)
	require.Len(t, sorular, 2)
	assert.Equal(t, "Soru bir?", sorular[0].Soru)
}

func TestFaqCikarEskiBicimdeCalismaz(t *testing.T) {
	// Eski dizi biçiminde başlık düzeyi yok: soruyu cevaptan ayıramayız,
	// bu yüzden şema basmıyoruz (uydurmaktansa boş bırakmak doğru).
	eski := []byte(`[{"type":"p","text":"Sık sorulan sorular"},{"type":"p","text":"Soru?"}]`)
	assert.Empty(t, faqCikar(eski))
}

func TestFaqJSONLDYapisi(t *testing.T) {
	js := string(faqJSONLD([]faqSoru{
		{Soru: "Soru bir?", Cevap: "Cevap bir."},
		{Soru: "Soru iki?", Cevap: "Cevap iki."},
	}))
	assert.Contains(t, js, `"@type":"FAQPage"`)
	assert.Contains(t, js, `"@type":"Question"`)
	assert.Contains(t, js, `"@type":"Answer"`)
	assert.Contains(t, js, "Soru bir?")
	assert.True(t, strings.HasPrefix(js, `<script type="application/ld+json">`),
		"blok script etiketiyle sarılmalı")
}
