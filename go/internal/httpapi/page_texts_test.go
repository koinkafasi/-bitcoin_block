package httpapi

import (
	"testing"

	"theblock/site/internal/repo"
)

func TestResolvePageTextsOncelikSirasi(t *testing.T) {
	defs := []pageTextDef{
		{Key: "kategori.baslik", Default: "{kategori} Haberleri"},
		{Key: "kategori.intro", Default: "varsayılan intro"},
	}
	rows := []repo.PageText{
		// Genel kategori, varsayılan dil.
		{PageKey: "kategori", TextKey: "kategori.baslik", Locale: "tr", Value: "Genel Başlık"},
		// Kategoriye özel, varsayılan dil.
		{PageKey: "kategori:ekonomi", TextKey: "kategori.baslik", Locale: "tr", Value: "Ekonomi Özel"},
		// Kategoriye özel, istenen dil (en üst öncelik).
		{PageKey: "kategori:ekonomi", TextKey: "kategori.intro", Locale: "en", Value: "Economy intro"},
	}
	got := resolvePageTexts("kategori:ekonomi", "kategori", rows, "en", "tr", defs,
		map[string]string{"kategori": "Ekonomi"})

	// İstenen dilde intro var → o kazanır.
	if got["kategori.intro"] != "Economy intro" {
		t.Errorf("intro: %q", got["kategori.intro"])
	}
	// İstenen dilde başlık yok → varsayılan dildeki kategoriye özel değer.
	if got["kategori.baslik"] != "Ekonomi Özel" {
		t.Errorf("baslik: %q", got["kategori.baslik"])
	}
}

func TestResolvePageTextsTabanSayfaVeYerTutucu(t *testing.T) {
	defs := []pageTextDef{{Key: "kategori.baslik", Default: "{kategori} Haberleri"}}
	rows := []repo.PageText{
		{PageKey: "kategori", TextKey: "kategori.baslik", Locale: "tr", Value: "{kategori} Gündemi"},
	}
	got := resolvePageTexts("kategori:spor", "kategori", rows, "tr", "tr", defs,
		map[string]string{"kategori": "Spor"})
	if got["kategori.baslik"] != "Spor Gündemi" {
		t.Errorf("taban sayfa + yer tutucu: %q", got["kategori.baslik"])
	}
}

func TestResolvePageTextsBosDegerYokSayilir(t *testing.T) {
	defs := []pageTextDef{{Key: "article.editor", Default: "Editör"}}
	rows := []repo.PageText{
		{PageKey: "article", TextKey: "article.editor", Locale: "tr", Value: "   "},
	}
	got := resolvePageTexts("article", "article", rows, "tr", "tr", defs, nil)
	if got["article.editor"] != "Editör" {
		t.Errorf("boş override yok sayılmalı: %q", got["article.editor"])
	}
}

func TestResolvePageTextsKodVarsayilani(t *testing.T) {
	defs := pageTextDefsFor("article")
	got := resolvePageTexts("article", "article", nil, "tr", "tr", defs,
		map[string]string{"dk": "5", "sayi": "1234"})
	if got["article.okuma_suresi"] != "5 dk okuma" {
		t.Errorf("okuma: %q", got["article.okuma_suresi"])
	}
	if got["article.goruntulenme"] != "1234 görüntülenme" {
		t.Errorf("görüntülenme: %q", got["article.goruntulenme"])
	}
	if got["article.ai_ozet"] != "AI ÖZET" {
		t.Errorf("ai özet: %q", got["article.ai_ozet"])
	}
}

func TestPageTextDefsSayfaTuru(t *testing.T) {
	if defs := pageTextDefsFor("home"); len(defs) != 0 {
		t.Errorf("ana sayfada şablon metni olmamalı, %d bulundu", len(defs))
	}
	if defs := pageTextDefsFor("kategori:ekonomi"); len(defs) == 0 {
		t.Error("kategoriye özel sayfa da kategori metinlerini içermeli")
	}
	defs := categoryTextDefs("ekonomi")
	for _, d := range defs {
		if d.Key == "kategori.intro" && d.Default == "" {
			t.Error("kategori intro varsayılanı hazır açıklama olmalı")
		}
	}
}

func TestPageTextBaseKey(t *testing.T) {
	cases := map[string]string{
		"kategori:ekonomi": "kategori",
		"kategori":         "kategori",
		"article":          "article",
		"home":             "home",
	}
	for in, want := range cases {
		if got := pageTextBaseKey(in); got != want {
			t.Errorf("%s → %s, beklenen %s", in, got, want)
		}
	}
}
