package templkit

import (
	"context"
	"strings"
	"testing"
)

// Kategori/bölüm sayfasının YAPI sözleşmesi.
//
// ÖLÇÜM (coindesk.com/markets, 8 Kasım 2026, 1440×1000, gerçek Chrome):
//
//	H1 kutusu : h1 36/44 w700, `border-b strong`, mb 24 (toplam 61 px)
//	12 kolon  : gap 16
//	  sol 9   : lead YATAY (metin 319 / görsel 686×386) + altında border-t
//	            alt ızgara 4 kolon × 239 px (görsel 239×135 üstte, h3 20/26)
//	  sağ 3   : GÖRSELSİZ ray, 6 kalem, h2 16/24 w500 + saat 14/22
//	"More From X" : 1032 px ORTALANMIŞ, border-t strong, 14/21 w500
//	haber satırı  : metin 681 + görsel 327×185, h2 24/30 w500, aralarda hr
//
// NİYE TEST: önceki sürüm ana sayfanın `.cd-kat-one` ızgarasını (6+3+3)
// kullanıyordu. Ölçülen sonuç: lead 681 px (olması gereken 1029), "Son
// Haberler" listesi 1029×749 px DİKEY kartlar (olması gereken 1032×185
// YATAY satırlar), sağ ray ve "More From" etiketi hiç yoktu. Bu testler
// o yanlış düzenin geri gelmesini engeller.

func kategoriCiz(t *testing.T, v CDSectionPage) string {
	t.Helper()
	var sb strings.Builder
	if err := CDSectionPageView(v).Render(context.Background(), &sb); err != nil {
		t.Fatalf("kategori sayfası çizilemedi: %v", err)
	}
	return sb.String()
}

func ornekKategori() CDSectionPage {
	k := CDStory{Title: "Başlık", Href: "/h/1", Image: "/kg/" + strings.Repeat("a", 32),
		Time: "2 gün önce"}
	return CDSectionPage{
		Shell:      CDShell{SiteName: "bitcosar"},
		Section:    "Finans",
		MoreBaslik: "Finans bölümünden daha fazlası",
		Lead:       k,
		Alt:        []CDStory{k, k, k, k},
		Ray:        []CDStory{k, k, k, k, k, k},
		Stories:    []CDStory{k, k, k},
	}
}

// Öne çıkan alan CoinDesk'in 12 kolonlu ızgarası olmalı: sol 9 / sağ 3.
// `.cd-hero` ve eski `.cd-kat-one` KULLANILMAMALI.
func TestKategoriIzgarasi(t *testing.T) {
	h := kategoriCiz(t, ornekKategori())
	if strings.Contains(h, `class="cd-hero"`) {
		t.Error("kategori sayfası .cd-hero kullanıyor (ana sayfa düzeni)")
	}
	if !strings.Contains(h, "cd-kat-izgara") {
		t.Error("12 kolonlu ızgara (.cd-kat-izgara) yok")
	}
	if !strings.Contains(h, "cd-kat-sol") {
		t.Error("sol 9 kolonluk blok (.cd-kat-sol) yok")
	}
	if strings.Contains(h, "cd-kat-one") {
		t.Error("eski 6+3+3 ızgarası (.cd-kat-one) kullanılmış — CoinDesk'te yok")
	}
	if strings.Contains(h, "cd-kat-bant") {
		t.Error("eski dikey liste bandı (.cd-kat-bant) kullanılmış — CoinDesk'te yok")
	}
}

// Lead YATAY olmalı (metin + görsel yan yana), dikey kart değil.
func TestKategoriLeadYatay(t *testing.T) {
	h := kategoriCiz(t, ornekKategori())
	for _, sinif := range []string{"cd-kat-lead", "cd-kat-lead__metin", "cd-kat-lead__gorsel"} {
		if !strings.Contains(h, sinif) {
			t.Errorf("lead bileşeni eksik: .%s", sinif)
		}
	}
	// Lead'in altında ayırıcı (CoinDesk: border-t mt-6 mb-2).
	if !strings.Contains(h, "cd-kat-lead__cizgi") {
		t.Error("lead altındaki ayırıcı (.cd-kat-lead__cizgi) yok")
	}
}

// Alt 4'lü ızgara ve GÖRSELSİZ sağ ray bulunmalı.
func TestKategoriAltVeRay(t *testing.T) {
	h := kategoriCiz(t, ornekKategori())
	if !strings.Contains(h, "cd-kat-alt") {
		t.Error("alt 4'lü ızgara (.cd-kat-alt) yok")
	}
	if !strings.Contains(h, "cd-kat-ray") {
		t.Error("sağ ray (.cd-kat-ray) yok")
	}
	if !strings.Contains(h, "cd-kat-ray__kalem") {
		t.Error("ray kalemi (.cd-kat-ray__kalem) yok")
	}
	// Ray GÖRSELSİZ: ray bloğu içinde <img> olmamalı.
	i := strings.Index(h, "cd-kat-ray")
	if i >= 0 {
		j := strings.Index(h[i:], "</aside>")
		if j > 0 && strings.Contains(h[i:i+j], "<img") {
			t.Error("sağ ray görsel basıyor; CoinDesk rayı GÖRSELSİZ (ölçüldü)")
		}
	}
}

// "More From X" etiketi ve 1032 px'lik ortalanmış satır listesi.
func TestKategoriMoreFromVeSatirlar(t *testing.T) {
	h := kategoriCiz(t, ornekKategori())
	if !strings.Contains(h, "cd-kat-more") {
		t.Error(`"More From" etiketi (.cd-kat-more) yok`)
	}
	if !strings.Contains(h, "Finans bölümünden daha fazlası") {
		t.Error("More From etiket metni basılmamış")
	}
	if !strings.Contains(h, "cd-kat-genis") {
		t.Error("1032 px ortalanmış kapsayıcı (.cd-kat-genis) yok")
	}
	if !strings.Contains(h, "cd-kat-satir") {
		t.Error("haber satırı (.cd-kat-satir) yok")
	}
	if !strings.Contains(h, "cd-kat-satir__cizgi") {
		t.Error("satırlar arası ayırıcı yok")
	}
}

// Kartlar yeni yuvaları kullanmalı — sizes ölçülen genişlikleri taşımalı
// (lead görseli 686, alt kart 239, satır görseli 327).
func TestKategoriYuvalariKullaniliyor(t *testing.T) {
	h := kategoriCiz(t, ornekKategori())
	for _, px := range []string{"686px", "239px", "327px"} {
		if !strings.Contains(h, px) {
			t.Errorf("sizes içinde %s yok (yeni kategori yuvası kullanılmıyor)", px)
		}
	}
	if strings.Contains(h, "1094px") {
		t.Error("kategori sayfasında hero lead yuvası (1094px) kullanılmış")
	}
}

// H1 bandı var, altında AÇIKLAMA PARAGRAFI yok.
//
// Ölçüldü: ne /markets ne /tr/business'te H1 altı açıklama var.
func TestKategoriH1AltiAciklamaYok(t *testing.T) {
	v := ornekKategori()
	v.Desc = "Bu metin <meta description>'da kalmalı, sayfada görünmemeli."
	h := kategoriCiz(t, v)
	if !strings.Contains(h, "cd-section-title") {
		t.Fatal("H1 bandı (.cd-section-title) yok")
	}
	i := strings.Index(h, "cd-section-title")
	j := strings.Index(h[i:], "</header>")
	if j > 0 && strings.Contains(h[i:i+j], "görünmemeli") {
		t.Error("H1 bandında açıklama paragrafı basılmış; ölçümde yok")
	}
}

// Sayfalama açık DÜĞME olmalı, sonsuz kaydırma değil.
func TestKategoriSayfalamaDugme(t *testing.T) {
	v := ornekKategori()
	v.NextHref = "/business?s=2"
	v.PrevHref = "/business"
	h := kategoriCiz(t, v)
	if !strings.Contains(h, "cd-daha__btn") {
		t.Error("sayfalama düğmesi (.cd-daha__btn) yok")
	}
	if !strings.Contains(h, `rel="next"`) {
		t.Error(`sayfalama bağlantısında rel="next" yok`)
	}
	if !strings.Contains(h, `rel="prev"`) {
		t.Error(`2. sayfada rel="prev" geri bağlantısı yok`)
	}
}

// Boş bölümde dürüst boş durum basılmalı, uydurma kart değil.
func TestKategoriBosDurum(t *testing.T) {
	v := CDSectionPage{Shell: CDShell{SiteName: "bitcosar"}, Section: "Boş"}
	h := kategoriCiz(t, v)
	if !strings.Contains(h, "henüz haber yok") {
		t.Error("boş bölümde dürüst boş durum metni yok")
	}
}
