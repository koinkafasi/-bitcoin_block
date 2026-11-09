package templkit

import (
	"context"
	"strings"
	"testing"
)

// Makale sayfası (/…/{yyyy}/{mm}/{dd}/{slug}) YAPI sözleşmesi.
//
// ÖLÇÜM (CoinDesk §10.2, gerçek Chrome 1440×1000):
//
//	kicker 16/32 → H1 36/44 w500 → deck 16/24 → "Yazan" + yazar →
//	tarih bloğu (y=488) → ÇEVİRİ ETİKETİ (y=510, 160×25) →
//	eylem pilleri (y=551) → kapak → Özet → gövde 22/30

func makaleCiz(t *testing.T, v CDArticleView) string {
	t.Helper()
	var sb strings.Builder
	if err := CDArticle(v).Render(context.Background(), &sb); err != nil {
		t.Fatalf("makale çizilemedi: %v", err)
	}
	return sb.String()
}

func ornekMakale() CDArticleView {
	return CDArticleView{
		Shell:        CDShell{SiteName: "bitcosar"},
		Title:        "Örnek Başlık — bitcosar",
		Desc:         "Örnek açıklama.",
		Canonical:    "https://ornek.test/policy/2026/11/05/orijinal-haber-f6eb90",
		Kicker:       "Politika",
		SectionName:  "Politika",
		SectionHref:  "/policy",
		Headline:     "Örnek Başlık",
		Spot:         "Örnek spot metni.",
		Author:       "Yazar Adı",
		PublishedAt:  "5 Kasım 2026 15:21 +03",
		PublishedISO: "2026-11-05T12:21:00Z",
		BodyHTML:     "<p>Gövde.</p>",
		// Paylaşım bağlantıları: handler'da GERÇEK değerlerle dolduruluyor
		// (bkz. cd_article.go). Örnekte boş bırakmak `href=""` üretiyordu;
		// o kusur testin kendisindeydi, sayfada değil (canlıda 0 boş href).
		ShareURL:      "https://ornek.test/policy/2026/11/05/orijinal-haber-f6eb90",
		ShareX:        "https://twitter.com/intent/tweet?url=x",
		ShareWhatsApp: "https://wa.me/?text=x",
		ShareLinkedIn: "https://www.linkedin.com/sharing/share-offsite/?url=x",
		Locale:        "tr",
		Dil:           CDMakaleDilTR(),
	}
}

// TestCDMakaleCeviriEtiketi — ÇEVİRİ ETİKETİ sözleşmesi (spec §10.2).
//
// NİYE TEST: etiket yanlış koşulda basılırsa orijinal haberler "çeviri"
// gibi görünür (okuyucu yanıltılır); hiç basılmazsa çeviri sayfası
// kaynağını saklar. İkisi de sessiz hatadır.
func TestCDMakaleCeviriEtiketi(t *testing.T) {
	v := ornekMakale()
	v.Dil = CDMakaleDilEN()
	v.CeviriEtiketi = "Translated by AI"
	v.CeviriHref = "/policy/2026/11/05/orijinal-haber-f6eb90"

	html := makaleCiz(t, v)
	if !strings.Contains(html, `class="cd-ceviri"`) {
		t.Fatalf("etiket basılmadı")
	}
	if !strings.Contains(html, "Translated by AI") {
		t.Errorf("etiket metni yok")
	}
	if !strings.Contains(html, `href="/policy/2026/11/05/orijinal-haber-f6eb90"`) {
		t.Errorf("orijinal bağlantısı yok")
	}
	if !strings.Contains(html, "translated into English by AI") {
		t.Errorf("aria-label yok")
	}

	// ETİKET TARİHTEN SONRA, EYLEM ŞERİDİNDEN ÖNCE (CoinDesk sırası).
	iTarih := strings.Index(html, "cd-article__tarih")
	iEtiket := strings.Index(html, "cd-ceviri")
	iAksiyon := strings.Index(html, "cd-aksiyon")
	if iTarih < 0 || iEtiket < 0 || iAksiyon < 0 {
		t.Fatalf("blok eksik (tarih=%d etiket=%d aksiyon=%d)", iTarih, iEtiket, iAksiyon)
	}
	if !(iTarih < iEtiket && iEtiket < iAksiyon) {
		t.Errorf("sıra yanlış: tarih=%d etiket=%d aksiyon=%d", iTarih, iEtiket, iAksiyon)
	}
}

// TestCDMakaleCeviriEtiketiYok orijinal içerikte etiket HİÇ basılmamalı.
func TestCDMakaleCeviriEtiketiYok(t *testing.T) {
	html := makaleCiz(t, ornekMakale())
	if strings.Contains(html, "cd-ceviri") {
		t.Errorf("çeviri olmayan içerikte etiket basılmış")
	}
	if strings.Contains(html, "AI tarafından çevrildi") {
		t.Errorf("çeviri metni basılmış")
	}
}

// TestCDMakaleCeviriBaglantiYok orijinal okunamıyorsa etiket yine görünür
// ama BAĞLANTISIZ (kırık href üretilmez).
func TestCDMakaleCeviriBaglantiYok(t *testing.T) {
	v := ornekMakale()
	v.CeviriEtiketi = "AI tarafından çevrildi"
	v.CeviriHref = ""

	html := makaleCiz(t, v)
	if !strings.Contains(html, `class="cd-ceviri"`) {
		t.Fatalf("etiket basılmadı")
	}
	if strings.Contains(html, `<a class="cd-ceviri"`) {
		t.Errorf("hedefsiz <a> basılmış")
	}
	if !strings.Contains(html, `<span class="cd-ceviri"`) {
		t.Errorf("span geri düşüşü yok")
	}
	// Etiket bloğunda boş href OLMAMALI. (Kabuğun arama düğmeleri
	// bilinçli olarak `href=""` taşır; JS varsayılanı iptal edip panel açar.)
	i := strings.Index(html, `class="cd-ceviri"`)
	if i < 0 {
		t.Fatalf("etiket yok")
	}
	if blok := html[i : i+240]; strings.Contains(blok, `href=""`) {
		t.Errorf("etiket bloğunda boş href: %s", blok)
	}
}

// TestCDMakaleDil metinlerin sayfa diline uyduğunu kilitler.
func TestCDMakaleDil(t *testing.T) {
	tr := CDMakaleDilIcin("tr")
	if tr.Yazan != "Yazan" || tr.Yayinlandi != "Yayınlandı" || tr.Guncellendi != "Güncellendi" {
		t.Errorf("TR iskelet yanlış: %+v", tr)
	}
	if tr.Ceviri != "AI tarafından çevrildi" {
		t.Errorf("TR etiket metni yanlış: %q", tr.Ceviri)
	}
	en := CDMakaleDilIcin("en")
	if en.Yazan != "By" || en.Yayinlandi != "Published" || en.Guncellendi != "Updated" {
		t.Errorf("EN iskelet yanlış: %+v", en)
	}
	if en.Ceviri != "Translated by AI" {
		t.Errorf("EN etiket metni yanlış: %q", en.Ceviri)
	}
	// Bilinmeyen/boş dil Türkçeye düşer (eski davranış korunur).
	for _, kod := range []string{"", "  ", "TR ", "xx"} {
		if got := CDMakaleDilIcin(kod); got.Yazan != "Yazan" {
			t.Errorf("%q → %q, Türkçeye düşmeliydi", kod, got.Yazan)
		}
	}
}

// TestCDMakaleIskeletDileDuyarli "Yazan/Yayınlandı" yerine sayfa dilinin
// metinleri basılmalı: İngilizce gövdeli sayfada Türkçe etiket kalmamalı.
func TestCDMakaleIskeletDileDuyarli(t *testing.T) {
	v := ornekMakale()
	v.Locale = "en"
	v.Dil = CDMakaleDilEN()
	html := makaleCiz(t, v)

	if !strings.Contains(html, ">By<") {
		t.Errorf("EN sayfada 'By' yok")
	}
	if !strings.Contains(html, "Published") {
		t.Errorf("EN sayfada 'Published' yok")
	}
	for _, turkce := range []string{">Yazan<", ">Yayınlandı"} {
		if strings.Contains(html, turkce) {
			t.Errorf("EN sayfada Türkçe iskelet kaldı: %s", turkce)
		}
	}
}
