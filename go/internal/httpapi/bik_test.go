package httpapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func strPtrBik(s string) *string { return &s }

// Kaynak standardı (Tebliğ): imzalı yazıda fikir işçisi, yoksa içeriğin
// kaynağı (ajans/bülten/iktibas), hiçbiri yoksa haber merkezi.
func TestBikArticleAuthorKaynakSirasi(t *testing.T) {
	yazar := &repo.PersonRef{ID: 7, Name: "Ahmet Yılmaz"}

	t.Run("fikir işçisi önceliklidir", func(t *testing.T) {
		c := repo.Content{SourceName: strPtrBik("AA")}
		assert.Equal(t, "Ahmet Yılmaz", bikArticleAuthor(c, yazar),
			"imza varsa ajans adı değil yazar adı yazılmalı")
	})

	t.Run("yazar yoksa kaynak adı", func(t *testing.T) {
		c := repo.Content{SourceName: strPtrBik("AA")}
		assert.Equal(t, "AA", bikArticleAuthor(c, nil))
	})

	t.Run("iktibas kaynağı aynen geçer", func(t *testing.T) {
		c := repo.Content{SourceName: strPtrBik("www.bik.gov.tr")}
		assert.Equal(t, "www.bik.gov.tr", bikArticleAuthor(c, nil))
	})

	t.Run("hiçbiri yoksa haber merkezi", func(t *testing.T) {
		assert.Equal(t, "HABER MERKEZI", bikArticleAuthor(repo.Content{}, nil))
		assert.Equal(t, "HABER MERKEZI", bikArticleAuthor(repo.Content{SourceName: strPtrBik("   ")}, nil))
	})

	t.Run("harf biçimi korunur", func(t *testing.T) {
		y := &repo.PersonRef{Name: "İlknur Çetin"}
		assert.Equal(t, "İlknur Çetin", bikArticleAuthor(repo.Content{}, y))
	})
}

func TestBuildBikMetaAlanlari(t *testing.T) {
	pub := time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC)
	upd := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	c := repo.Content{
		Title:        "Borsa günü nasıl kapandı",
		CategoryName: strPtrBik("Ekonomi"),
		PublishedAt:  &pub,
		UpdatedAt:    upd,
		SourceName:   strPtrBik("AA"),
	}
	m := buildBikMeta(c, nil, "https://theblock.com.tr/borsa-gunu-1")

	assert.Equal(t, "Borsa günü nasıl kapandı", m.Title)
	assert.Equal(t, "https://theblock.com.tr/borsa-gunu-1", m.URL)
	assert.Equal(t, "Ekonomi", m.Section)
	assert.Equal(t, "AA", m.Author)

	assert.Equal(t, "2026-10-20T09:30:00Z", m.Published)
	assert.Equal(t, "2026-10-21T12:00:00Z", m.Modified)
	assert.NotContains(t, m.Published, "+")
	assert.NotContains(t, m.Modified, "+")
}

// Güncelleme yayından önce görünüyorsa (saat kayması, toplu içe aktarma)
// dateModified < datePublished mantıksız olur.
func TestBuildBikMetaGuncellemeYayindanOnceOlamaz(t *testing.T) {
	pub := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	upd := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	m := buildBikMeta(repo.Content{PublishedAt: &pub, UpdatedAt: upd}, nil, "x")
	assert.Equal(t, m.Published, m.Modified)
}

// Yanlış değer basmaktansa hiç basmamak yeğdir: boş alanlar şablonda
// atlanabilsin diye boş dönmeli.
func TestBuildBikMetaBosAlanlar(t *testing.T) {
	m := buildBikMeta(repo.Content{Title: "  "}, nil, "")
	assert.Empty(t, m.Title)
	assert.Empty(t, m.URL)
	assert.Empty(t, m.Section)
	assert.Empty(t, m.Published)
	assert.Equal(t, "HABER MERKEZI", m.Author, "kaynak her zaman yazılmalı")
}

func TestBikKunyeHTMLTumKimlikleriBasar(t *testing.T) {
	alanlar := bikKunyeAlanlari()
	require.Len(t, alanlar, 10, "tebliğ on zorunlu alan sayıyor")

	out := string(bikKunyeHTML(alanlar)) // hepsi boş
	for _, a := range alanlar {
		assert.Contains(t, out, `id="`+a.ID+`"`, "%s kimliği basılmalı", a.ID)
	}
	assert.Contains(t, out, "—", "boş alan görünür biçimde işaretlenmeli")
}

func TestBikKunyeHTMLDegerKacirir(t *testing.T) {
	alanlar := []bikKunyeAlan{{
		ID: "bik-kunye-yayinci", Label: "Yayıncı",
		Value: `<script>alert(1)</script>`,
	}}
	out := string(bikKunyeHTML(alanlar))
	assert.NotContains(t, out, "<script>", "ayar değeri HTML olarak yorumlanmamalı")
	assert.Contains(t, out, "&lt;script&gt;")
}

func TestBikKunyeEksiklerBosAlanlariSayar(t *testing.T) {
	alanlar := bikKunyeAlanlari()
	assert.Len(t, bikKunyeEksikler(alanlar), 10, "hiçbiri doldurulmadı")

	alanlar[0].Value = "Örnek Medya A.Ş."
	eksik := bikKunyeEksikler(alanlar)
	assert.Len(t, eksik, 9)
	// Tam etiketle karşılaştır: "Yer Sağlayıcı Ticaret Unvanı" etiketi de
	// "Ticaret Unvanı" dizesini içeriyor ve o alan hâlâ boş.
	assert.NotContains(t, eksik, "Yayın Sahibi Ticaret Unvanı",
		"dolu alan eksik listesinde olmamalı")
	assert.Contains(t, eksik, "Yer Sağlayıcı Ticaret Unvanı",
		"boş kalan benzer adlı alan eksik sayılmalı")
}
