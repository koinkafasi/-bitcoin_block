package classify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugKaynakBolumuOnecelikli(t *testing.T) {
	// Bölüm bilgisi başlıktaki sözcüklerden daha güvenilir: yayıncının kendi
	// sınıflandırması.
	got := Slug(Doc{
		Title:    "Dolar kuru bugün ne kadar oldu?",
		Spot:     "Serbest piyasada dolar ve euro yükseldi.",
		Sections: []string{"Spor"},
	}, "gundem")
	assert.Equal(t, "spor", got)
}

func TestSlugAnahtarSozcuk(t *testing.T) {
	got := Slug(Doc{
		Title: "BIST 100 endeksinde sert düşüş",
		Spot:  "Borsa İstanbul'da satış baskısı arttı.",
	}, "gundem")
	assert.Equal(t, "borsa", got)
}

func TestSlugTurkceBuyukHarfDuyarli(t *testing.T) {
	// 'İ' ve 'I' Türkçe kurallarıyla çevrilmezse eşleşme kaçar.
	a := Slug(Doc{Title: "İSTANBUL'DA FUTBOL MAÇI"}, "")
	b := Slug(Doc{Title: "istanbul'da futbol maçı"}, "")
	assert.Equal(t, "futbol", a)
	assert.Equal(t, a, b)
}

func TestSlugFeedKategorisiYedek(t *testing.T) {
	got := Slug(Doc{Title: "Belediye duyurdu"}, "yasam")
	assert.Equal(t, "yasam", got)
}

func TestSlugFeedKategorisiAnahtarSozcuktenOnce(t *testing.T) {
	// Editörün atadığı kategori heuristiği ezer: "GZT Spor" akışındaki bir
	// haberin spotunda "dolar" geçse bile kategori spordur.
	got := Slug(Doc{Title: "Transfer bütçesi dolarla açıklandı"}, "spor")
	assert.Equal(t, "spor", got)
}

func TestSlugVarsayilanGundem(t *testing.T) {
	assert.Equal(t, "gundem", Slug(Doc{Title: "Kısa bir açıklama geldi"}, ""))
}

func TestSlugGundemFeedBolumuGundemDondurur(t *testing.T) {
	// Bölüm "Gündem" ise anahtar sözcük aşaması yine denenir; ikisi de yoksa
	// sonuç gundem olmalı, boş string değil.
	assert.Equal(t, "gundem", Slug(Doc{Title: "Açıklama", Sections: []string{"Gündem"}}, ""))
}

func TestSlugYanlisPozitifYok(t *testing.T) {
	// "spor" kelimesi geçmiyor; cümle spor haberi değil.
	assert.Equal(t, "gundem", Slug(Doc{Title: "Yeni spor salonu açıldı mı"}, ""))
}

func TestClassifyGerekce(t *testing.T) {
	slug, reason := Classify(Doc{Title: "Merkez Bankası faiz kararını açıkladı", Spot: "Politika faizi sabit kaldı."}, "")
	assert.Equal(t, "faiz", slug)
	assert.Contains(t, reason, "anahtar sözcük")

	slug, reason = Classify(Doc{Sections: []string{"Ekonomi"}}, "")
	assert.Equal(t, "ekonomi", slug)
	assert.Equal(t, "kaynak bölümü", reason)

	slug, reason = Classify(Doc{Title: "Duyuru"}, "spor")
	assert.Equal(t, "spor", slug)
	assert.Equal(t, "feed kategorisi", reason)
}

func TestSectionMapSluglariGecerli(t *testing.T) {
	// Bölüm tablosu yalnız gerçek kategori slug'larına işaret etmeli.
	valid := map[string]bool{}
	for _, s := range []string{
		"gundem", "ekonomi", "is-dunyasi", "finans", "piyasalar", "borsa", "doviz",
		"altin", "emtia", "faiz", "enflasyon", "kripto-para", "dunya", "spor",
		"futbol", "basketbol", "teknoloji", "bilim", "oyun", "e-spor", "yasam",
		"saglik", "egitim-kultur", "sanat", "sinema", "muzik", "kitap", "edebiyat",
		"tarih", "mimari", "turizm", "gezi-rotalari", "seyahat-rehberi", "otomotiv",
		"enerji", "emlak", "tarim", "bankacilik", "asayis", "aktuel", "toplum",
		"kadin", "genclik", "medya-dijital-kultur", "sehir-rehberi", "afrika",
		"amerika", "asya", "avrupa", "ortadogu", "siyaset", "afet-kaza",
	} {
		valid[s] = true
	}
	for section, slug := range sectionMap {
		assert.True(t, valid[slug], "bölüm %q bilinmeyen slug %q", section, slug)
	}
}
