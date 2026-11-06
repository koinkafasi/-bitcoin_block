package templkit

import (
	"context"
	"strings"
	"testing"
)

// Kategori/bölüm sayfasının YAPI sözleşmesi (ölçüm: spec §10.3).
//
// CoinDesk `/tr/business` ölçüldü: kapsayıcı 1432, içerik 1384, öne çıkan
// alan **12 kolonlu** ızgara (gap 16; lead 6 kolon = 689px, diğerleri
// 3 kolon = 327px), liste bandı 12'nin 9'u (1032px), sayfalama açık
// düğme. **Hero YOK.**
//
// NİYE TEST: kategori sayfası `.cd-hero`'yu kullanıyordu ve o ızgaraya
// yalnız İKİ çocuk veriliyordu. Ölçülen sonuç: lead 290px, yan kartlar
// 596px — manşet yan kartlardan DAR, hiyerarşi tersine dönmüş, ve bir
// 290px kolon boş. Bu testler o düzenin geri gelmesini engeller.

func kategoriCiz(t *testing.T, v CDSectionPage) string {
	t.Helper()
	var sb strings.Builder
	if err := CDSectionPageView(v).Render(context.Background(), &sb); err != nil {
		t.Fatalf("kategori sayfası çizilemedi: %v", err)
	}
	return sb.String()
}

func ornekKategori() CDSectionPage {
	k := CDStory{Title: "Başlık", Href: "/h/1", Image: "/kg/" + strings.Repeat("a", 32)}
	return CDSectionPage{
		Shell:   CDShell{SiteName: "bitcosar"},
		Section: "Finans",
		Lead:    k,
		Side:    []CDStory{k, k},
		Stories: []CDStory{k, k, k},
	}
}

// Öne çıkan alan 12 kolonlu ızgara olmalı, hero OLMAMALI.
func TestKategoriHeroKullanmaz(t *testing.T) {
	h := kategoriCiz(t, ornekKategori())
	if strings.Contains(h, `class="cd-hero"`) {
		t.Error(`kategori sayfası .cd-hero kullanıyor; ölçümde manşet 290px'lik en dar kolona düşüyordu (spec §10.3.1)`)
	}
	if strings.Contains(h, "cd-hero__side") {
		t.Error("kategori sayfasında .cd-hero__side olmamalı")
	}
	if !strings.Contains(h, "cd-kat-one") {
		t.Error("öne çıkan 12 kolonlu ızgara (.cd-kat-one) yok")
	}
}

// Liste bandı 9 kolonluk sarmalayıcı içinde olmalı.
func TestKategoriListeBandi(t *testing.T) {
	h := kategoriCiz(t, ornekKategori())
	if !strings.Contains(h, "cd-kat-bant") {
		t.Error("liste bandı sarmalayıcısı (.cd-kat-bant) yok")
	}
	if !strings.Contains(h, "cd-kat-liste") {
		t.Error("liste kolonu (.cd-kat-liste) yok")
	}
	// Eski 3 kolonlu ızgara kullanılmamalı: CoinDesk 12 kolon kullanıyor.
	if strings.Contains(h, "cd-grid--3") {
		t.Error("kategori listesi .cd-grid--3 kullanıyor; 12 kolonlu ızgara olmalı")
	}
}

// Kartlar kategori yuvalarını kullanmalı — sizes ölçülen genişlikleri
// taşımalı (lead 689, kart 327).
func TestKategoriYuvalariKullaniliyor(t *testing.T) {
	h := kategoriCiz(t, ornekKategori())
	if !strings.Contains(h, "689px") {
		t.Error("öne çıkan kartın sizes'ında 689px yok (YuvaKatLead kullanılmıyor)")
	}
	if !strings.Contains(h, "327px") {
		t.Error("kategori kartının sizes'ında 327px yok (YuvaKatKart kullanılmıyor)")
	}
	// Hero yuvaları kategori sayfasında kullanılmamalı.
	if strings.Contains(h, "1094px") {
		t.Error("kategori sayfasında hero lead yuvası (1094px) kullanılmış")
	}
}

// H1 bandı var, altında AÇIKLAMA PARAGRAFI yok.
//
// Ölçüldü: ne /markets ne /tr/business'te H1 altı açıklama var. Eski
// spec §8 "tek satır açıklama" diyordu — yanlıştı.
func TestKategoriH1AltiAciklamaYok(t *testing.T) {
	v := ornekKategori()
	v.Desc = "Bu metin <meta description>'da kalmalı, sayfada görünmemeli."
	h := kategoriCiz(t, v)
	if !strings.Contains(h, "cd-section-title") {
		t.Fatal("H1 bandı (.cd-section-title) yok")
	}
	// Açıklama yalnız meta'da olmalı, başlık bandının içinde değil.
	i := strings.Index(h, "cd-section-title")
	j := strings.Index(h[i:], "</header>")
	if j > 0 && strings.Contains(h[i:i+j], "görünmemeli") {
		t.Error("H1 bandında açıklama paragrafı basılmış; ölçümde yok (spec §10.3)")
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
	// Geri bağlantısı erişilebilirlik için kalmalı.
	if !strings.Contains(h, `rel="prev"`) {
		t.Error(`2. sayfada rel="prev" geri bağlantısı yok`)
	}
}

// Boş bölümde dürüst boş durum basılmalı, uydurma kart değil.
func TestKategoriBosDurum(t *testing.T) {
	v := ornekKategori()
	v.Stories = nil
	h := kategoriCiz(t, v)
	if !strings.Contains(h, "henüz haber yok") {
		t.Error("boş bölümde dürüst boş durum metni yok")
	}
}
