package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/repo"
)

// Kategori görünürlüğü (gorunum_kats) — kullanıcı isteği:
// "özelik olarak spor yazısı sayfalarında göstermek için bir de geliştirme
// ekle modüle seçenek, hangi yazı kategorilerinde yazılarda olması için".
//
// Kural: ayar boşsa blok her yerde; doluysa yalnız seçilen kategorilerin
// YAZILARINDA basılır. Ana sayfa/kategori sayfalarında kısıt uygulanmaz —
// orada düzeni kullanıcı ayrıca düzenler.

func yaziSayfasi(slug string) *tmplData {
	c := repo.Content{CategorySlug: &slug}
	return &tmplData{Article: &articleData{Content: c}}
}

func TestBlokKategoriGorunumuOkuma(t *testing.T) {
	assert.Nil(t, blokKategoriGorunumu(nil), "ayar yoksa kısıt yok")
	assert.Nil(t, blokKategoriGorunumu(map[string]any{}))
	assert.Nil(t, blokKategoriGorunumu(map[string]any{"gorunum_kats": ""}))
	assert.Nil(t, blokKategoriGorunumu(map[string]any{"gorunum_kats": "  ,  , "}))

	got := blokKategoriGorunumu(map[string]any{"gorunum_kats": "spor, futbol"})
	assert.Equal(t, []string{"spor", "futbol"}, got)
}

// Tırnaklı eski kayıtlar onarılmalı.
//
// Gerçek olay (28 Ekim 2026): panelde kategori seçildiğinde değer JSON'a
// `"spor"` (tırnaklar dahil) yazıldı. Sunucu `"spor"` okuduğu için yazının
// slug'ı `spor` ile eşleşmedi ve modül spor yazılarında GÖRÜNMEDİ; üstelik
// ekonomi yazılarında da gizlenmiyordu (hiçbir şeyle eşleşmediği için
// sonuç "hiç görünmemek" oldu).
func TestBlokKategoriGorunumuTirnakTemizler(t *testing.T) {
	assert.Equal(t, []string{"spor"},
		blokKategoriGorunumu(map[string]any{"gorunum_kats": `"spor"`}))
	assert.Equal(t, []string{"spor", "futbol"},
		blokKategoriGorunumu(map[string]any{"gorunum_kats": `"spor","futbol"`}))
	assert.Equal(t, []string{"spor"},
		blokKategoriGorunumu(map[string]any{"gorunum_kats": `'spor'`}))

	// Tırnaklı değer, filtre kararında da doğru eşleşmeli.
	opts := map[string]any{"gorunum_kats": `"spor"`}
	assert.True(t, blokYazidaGorunur(opts, yaziSayfasi("spor")))
	assert.False(t, blokYazidaGorunur(opts, yaziSayfasi("ekonomi")))
}

func TestBlokYazidaGorunurAyarYoksaHerYerde(t *testing.T) {
	assert.True(t, blokYazidaGorunur(nil, yaziSayfasi("ekonomi")))
	assert.True(t, blokYazidaGorunur(map[string]any{}, yaziSayfasi("ekonomi")))
}

func TestBlokYazidaGorunurKategoriEslesmesi(t *testing.T) {
	opts := map[string]any{"gorunum_kats": "spor,futbol"}

	// Spor yazısında görünür
	assert.True(t, blokYazidaGorunur(opts, yaziSayfasi("spor")))
	// Futbol yazısında görünür
	assert.True(t, blokYazidaGorunur(opts, yaziSayfasi("futbol")))
	// Ekonomi yazısında GÖRÜNMEZ
	assert.False(t, blokYazidaGorunur(opts, yaziSayfasi("ekonomi")))
}

func TestBlokYazidaGorunurBuyukKucukHarf(t *testing.T) {
	// Kategori slug'ları küçük harf ama ayar elle yazılmış olabilir.
	opts := map[string]any{"gorunum_kats": "Spor"}
	assert.True(t, blokYazidaGorunur(opts, yaziSayfasi("spor")))
}

func TestBlokYazidaGorunurYaziSayfasiDegilseKisitYok(t *testing.T) {
	opts := map[string]any{"gorunum_kats": "spor"}
	// Ana sayfa: d.Article nil → kısıt uygulanmaz
	assert.True(t, blokYazidaGorunur(opts, &tmplData{}))
	assert.True(t, blokYazidaGorunur(opts, nil))
}

func TestBlokYazidaGorunurKategorisizYazi(t *testing.T) {
	opts := map[string]any{"gorunum_kats": "spor"}
	// Kategorisi olmayan yazı: kısıt uygulanamaz → göster (sessizce gizlemek
	// editörü şaşırtırdı).
	empty := ""
	assert.True(t, blokYazidaGorunur(opts, &tmplData{Article: &articleData{
		Content: repo.Content{CategorySlug: &empty},
	}}))
	assert.True(t, blokYazidaGorunur(opts, &tmplData{Article: &articleData{
		Content: repo.Content{CategorySlug: nil},
	}}))
}
