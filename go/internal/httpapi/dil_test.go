package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIcerikDiliDususu: içerik yoksa varsayılan dile düşülür ve bu
// "düşüş" olarak işaretlenir (noindex + kanonik için).
func TestIcerikDiliDususu(t *testing.T) {
	src := okuDosya(t, "dil_yardimcilari.go")

	// Fonksiyon var ve iki değer döndürüyor.
	assert.Contains(t, src, "func (s *Server) icerikDili(")
	i := strings.Index(src, "func (s *Server) icerikDili(")
	son := strings.Index(src[i:], "\n}")
	govde := src[i : i+son]
	assert.Contains(t, govde, "yerelIcerikVar(",
		"içerik varlığı kontrol edilmeli")
	assert.Contains(t, govde, "return varsayilan, true",
		"içerik yoksa düşüş işaretlenmeli")
	assert.Contains(t, govde, "return istenen, false")

	// Kullanım: home + kategori düşüşü canonical/noindex üretmeli.
	pub := okuDosya(t, "public_handlers.go")
	assert.Contains(t, pub, "icerikLocale, dusus := s.icerikDili(ctx, site.ID, lang)",
		"ana sayfa düşüşü hesaplamalı")
	assert.Contains(t, pub, `d.Robots = "noindex,follow"`,
		"düşüşte noindex verilmeli")
	assert.Contains(t, pub, "d.Canonical = strings.TrimRight(s.absBase(r), \"/\") + homeURL(s.i18n.Default())",
		"düşüşte canonical varsayılan dile verilmeli (kopya dizinlenmesin)")
}

func TestHreflangYalnizIcerikliDiller(t *testing.T) {
	src := okuDosya(t, "dil_yardimcilari.go")
	i := strings.Index(src, "func (s *Server) yerelHreflang(")
	require.Greater(t, i, 0, "yerelHreflang bulunamadı")
	son := strings.Index(src[i:], "\n}")
	govde := src[i : i+son]

	assert.Contains(t, govde, "yerelIcerikVar(ctx, siteID, l.Code)",
		"yalnız içeriği olan diller listelenmeli")
	assert.Contains(t, govde, `Code: "x-default"`,
		"x-default her zaman eklenmeli")
	assert.Contains(t, govde, "if len(out) < 2",
		"tek dilde hreflang anlamsız, boş dönmeli")

	// Şablon: hreflang public_head'te TEK yerden basılmalı.
	head := okuDosya(t, "../webui/templates/partials.html")
	assert.Contains(t, head, `hreflang="{{.Code}}"`,
		"public_head hreflang basmalı")

	// Makale şablonundaki eski ikinci blok kaldırılmış olmalı (çift etiket).
	pub := okuDosya(t, "../webui/templates/site_public.html")
	assert.NotContains(t, pub, ".Article.Hreflangs",
		"hreflang tek yerden (public_head) basılmalı, çift etiket olmamalı")
}

// TestSiteAdiYerellesir: /en başlığı Türkçe slogan taşımamalı.
func TestSiteAdiYerellesir(t *testing.T) {
	src := okuDosya(t, "tmpl.go")
	assert.Contains(t, src, `bundle.T(loc, "site.name")`)
	assert.Contains(t, src, `s.localeText(loc, "site.suffix", "Güncel Haber Portalı")`)

	// Çeviri ATANMALI: yalnız anahtarın sorgulanması yetmez; mutasyonda
	// `siteAdi = v` yerine `_ = v` yazılınca test yeşil kalıyordu.
	i := strings.Index(src, `bundle.T(loc, "site.name")`)
	require.Greater(t, i, 0)
	son := strings.Index(src[i:], "\n\t}")
	require.Greater(t, son, 0)
	blok := src[i : i+son]
	assert.Contains(t, blok, "siteAdi = v",
		"yerelleştirilmiş ad siteAdi'ye atanmalı")

	// Karşılaştırma ORİJİNAL adla yapılmalı: yerelleştirilmiş adla
	// karşılaştırınca "theblock.com.tr — TheBlok" çıkıyordu.
	assert.Contains(t, src, "title != s.siteName",
		"site adı karşılaştırması orijinal adla olmalı")

	art := okuDosya(t, "cd_article.go")
	bas := strings.Index(art, "cdMakaleBaslik(c)")
	require.Greater(t, bas, 0, "CD makale <title>'ı cdMakaleBaslik'tan gelmeli")
	assert.Contains(t, art[:bas], "Title:", "başlık alanı bu üreticiyle doldurulmalı")
}

// TestKategoriAdiCevirisi: kategori adları `category.<slug>` anahtarından.
func TestKategoriAdiCevirisi(t *testing.T) {
	src := okuDosya(t, "menu.go")

	assert.Contains(t, src, "func (s *Server) publicMenuLocale(")
	assert.Contains(t, src, "func (s *Server) navCategoriesLang(")
	assert.Contains(t, src, `"category." + slug`, "çeviri anahtarı kategori slug'ı taşımalı")
	assert.Contains(t, src, `"category." + cats[i].Slug`, "gezinme kategorileri de çevrilmeli")

	// Elle eklenmiş menü satırlarında slug boş olabilir; adresten türetilir.
	assert.Contains(t, src, `strings.Index(u, "/kategori/")`,
		"slug boşsa adresten türetilmeli (elle eklenen satırlar)")
}

// TestCeviriAyarAnahtarlari: panelden yönetilebilir olmalı.
func TestCeviriAyarAnahtarlari(t *testing.T) {
	bot := okuDosya(t, "bot_handlers.go")
	assert.Contains(t, bot, `"ceviri_otomatik"`)
	assert.Contains(t, bot, `"ceviri_diller"`)
	assert.Contains(t, bot, `"ceviri_durum"`)
	assert.Contains(t, bot, "ceviri.AyarlariYaz")

	tpl := okuDosya(t, "../webui/templkit/trend.templ")
	assert.Contains(t, tpl, `name="ceviri_otomatik"`)
	assert.Contains(t, tpl, `name="ceviri_diller"`)

	main := okuDosya(t, "../../cmd/site/main.go")
	assert.Contains(t, main, "ceviri.Run(", "arka plan işi başlatılmalı")
}

func TestDilMigrationSozlesmesi(t *testing.T) {
	up := okuDosya(t, "../../migrations/0100_dil_ceviri_en.up.sql")

	for _, anahtar := range []string{"'all_in'", "'news_word'", "'site.name'", "'site.suffix'"} {
		assert.Contains(t, up, anahtar)
	}
	// Aynı anahtar hem tr hem en için olmalı.
	assert.Contains(t, up, "('tr', 'all_in'")
	assert.Contains(t, up, "('en', 'all_in'")
	// Kategori çevirileri de var.
	assert.Contains(t, up, "'category.gundem'")

	// Panelden düzenlenmiş çeviri EZİLMEMELİ.
	assert.Contains(t, up, "ON CONFLICT (locale, key) DO NOTHING")

	// Geri alma yalnız bu dosyanın sabit anahtarlarını siler.
	down := okuDosya(t, "../../migrations/0100_dil_ceviri_en.down.sql")
	assert.Contains(t, down, "DELETE FROM translations")
	assert.NotContains(t, down, "'category.%'",
		"kategori anahtarları elle düzenlenmiş olabilir, silinmemeli")
}

// TestDilSeciciYolKorur: dil değiştirmek makaleyi kaybettirmemeli.
func TestDilSeciciYolKorur(t *testing.T) {
	js := okuDosya(t, "../webui/static/app.js")

	i := strings.Index(js, "[data-lang-select]")
	require.Greater(t, i, 0)
	son := strings.Index(js[i:], "});")
	govde := js[i : i+son]

	// Eski davranış: her zaman dilin ana sayfasına gitmek.
	assert.NotContains(t, govde, `base === "/" ? "/" + lang + "/"`,
		"ana sayfaya zorlayan eski davranış kalkmalı")
	// Yol korunmalı: öneki soy, yenisini ekle.
	assert.Contains(t, govde, `yol.split("/")`,
		"yol location.pathname'den parçalanmalı (mutasyonda boş dizi olunca test yeşil kalıyordu)")
	assert.Contains(t, govde, "parcalar.shift()", "mevcut dil öneki soyulmalı")
	assert.Contains(t, govde, "location.search", "sorgu korunmalı")
	assert.Contains(t, govde, "location.hash", "çapa korunmalı")
}
