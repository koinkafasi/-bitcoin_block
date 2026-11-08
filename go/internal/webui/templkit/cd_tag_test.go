package templkit

import (
	"context"
	"strings"
	"testing"
)

// Etiket sayfası (/tag/{slug}) YAPI sözleşmesi.
//
// ÖLÇÜM (coindesk.com/tag/bitcoin, 8 Kasım 2026, 1440×1000, gerçek Chrome):
//
//	banner  : sol metin max 680 (h1 36/44 **w400** capitalize, spot 18/27),
//	          sağ görsel 692×389
//	<hr>    : tam genişlik
//	12 kolon: sol 8 (metin 524 + görsel 327×185, başlık 16/24 **w400**,
//	          saat 12/16, aralarda hr), sağ 4 ("Latest Crypto News" paneli)
//
// NİYE TEST: önceki sürüm etiket sayfasını BÖLÜM şablonuyla basıyordu —
// banner yok, h1 w700, 6+3+3 ızgara. CoinDesk'te iki aile tamamen farklı.

func etiketCiz(t *testing.T, v CDTagPage) string {
	t.Helper()
	var sb strings.Builder
	if err := CDTagPageView(v).Render(context.Background(), &sb); err != nil {
		t.Fatalf("etiket sayfası çizilemedi: %v", err)
	}
	return sb.String()
}

func ornekEtiket() CDTagPage {
	k := CDStory{Title: "Başlık", Href: "/h/1", Image: "/kg/" + strings.Repeat("b", 32),
		Time: "2 saat önce", Spot: "Spot metni"}
	return CDTagPage{
		Shell:       CDShell{SiteName: "bitcosar"},
		H1:          "Bitcoin Haberleri",
		Spot:        "Bitcoin etiketli son kripto haberleri ve analizler.",
		BannerImage: "/kg/" + strings.Repeat("b", 32),
		BannerHref:  "/h/0",
		Stories:     []CDStory{k, k, k},
		Latest:      []CDStory{k, k},
	}
}

// Banner yapısı: metin + görsel yan yana, ardından tam genişlik ayırıcı.
func TestEtiketBanner(t *testing.T) {
	h := etiketCiz(t, ornekEtiket())
	for _, sinif := range []string{"cd-tag-banner", "cd-tag-banner__metin", "cd-tag-banner__gorsel", "cd-tag-hr"} {
		if !strings.Contains(h, sinif) {
			t.Errorf("banner bileşeni eksik: .%s", sinif)
		}
	}
	if !strings.Contains(h, "Bitcoin Haberleri") {
		t.Error("etiket H1'i basılmamış")
	}
}

// Görsel yoksa banner tek kolon kalmalı (uydurma görsel basılmaz).
func TestEtiketBannerGorselsiz(t *testing.T) {
	v := ornekEtiket()
	v.BannerImage = ""
	h := etiketCiz(t, v)
	if strings.Contains(h, "cd-tag-banner__gorsel") {
		t.Error("banner görseli yokken görsel sütunu basılmış")
	}
	if !strings.Contains(h, "cd-tag-banner__metin") {
		t.Error("metin sütunu kaybolmuş")
	}
}

// 8/4 ızgara: sol liste, sağ "Son Kripto Haberler" paneli.
func TestEtiketIzgarasi(t *testing.T) {
	h := etiketCiz(t, ornekEtiket())
	for _, sinif := range []string{"cd-tag-izgara", "cd-tag-liste", "cd-tag-yan", "cd-tag-latest"} {
		if !strings.Contains(h, sinif) {
			t.Errorf("ızgara bileşeni eksik: .%s", sinif)
		}
	}
	if !strings.Contains(h, "Son Kripto Haberler") {
		t.Error("sağ panel başlığı yok")
	}
	// Bölüm sayfasının ızgarası etiket sayfasında KULLANILMAMALI.
	if strings.Contains(h, "cd-kat-izgara") {
		t.Error("etiket sayfası bölüm ızgarasını (cd-kat-izgara) kullanıyor")
	}
}

// Satırlar: küçük başlık (16/24 w400 sınıfı), saat, görsel, aralarda hr.
func TestEtiketSatirlari(t *testing.T) {
	h := etiketCiz(t, ornekEtiket())
	for _, sinif := range []string{"cd-tag-satir", "cd-tag-satir__metin", "cd-tag-satir__saat", "cd-tag-satir__gorsel", "cd-tag-satir__cizgi"} {
		if !strings.Contains(h, sinif) {
			t.Errorf("satır bileşeni eksik: .%s", sinif)
		}
	}
	if !strings.Contains(h, "327px") {
		t.Error("satır görselinin sizes'ında 327px yok")
	}
}

// Banner görseli 692 px yuvasını kullanmalı.
func TestEtiketBannerYuvasi(t *testing.T) {
	h := etiketCiz(t, ornekEtiket())
	if !strings.Contains(h, "692px") {
		t.Error("banner görselinin sizes'ında 692px yok (YuvaTagBanner kullanılmıyor)")
	}
}

// Boş etikette dürüst boş durum.
func TestEtiketBosDurum(t *testing.T) {
	v := CDTagPage{Shell: CDShell{SiteName: "bitcosar"}, H1: "Boş Haberleri"}
	h := etiketCiz(t, v)
	if !strings.Contains(h, "henüz haber yok") {
		t.Error("boş etikette dürüst boş durum metni yok")
	}
}
