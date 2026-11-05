package httpapi

import (
	"context"
	"strings"

	"theblock/site/internal/repo"
)

// Yerelleştirme yardımcıları (Faz 189).
//
// Ölçülen sorun: 24 dil etkin, ama yalnız Türkçe içerik var. `/en` 200
// dönüyordu ve <html lang="en"> doğruydu; buna karşılık ana sayfada manşet
// YOKTU (TR'de 150 hero öğesi, EN'de 0) çünkü içerik sorguları
// `locale = $1` ile sıkı filtreleniyor. Sonuç: boş manşet alanı → tasarım
// kırık görünüyor, sayfa "İngilizce" olduğunu iddia ederken Türkçe
// başlıklar taşıyordu.

// yerelIcerikVar verilen dilde yayımlanmış içerik var mı?
//
// Yalnız varlık kontrolü: sayım gerekmiyor, LIMIT 1 yeterli.
func (s *Server) yerelIcerikVar(ctx context.Context, siteID int64, locale string) bool {
	if locale == "" {
		return false
	}
	var bir int
	err := s.pool.QueryRow(ctx,
		`SELECT 1 FROM contents WHERE site_id=$1 AND locale=$2 AND status='yayinda' LIMIT 1`,
		siteIDOr1(siteID), locale).Scan(&bir)
	return err == nil
}

// icerikDili içerik sorgularında kullanılacak dili verir.
//
// İstenen dilde içerik varsa o; yoksa varsayılan dil (tr). İkinci değer
// true ise "gösterilen içerik istenen dilden FARKLI" demektir: bu durumda
// sayfa noindex'lenir ve canonical varsayılan dilin adresine verilir, yoksa
// aynı Türkçe içerik 24 adreste kopya olarak dizinlenir.
func (s *Server) icerikDili(ctx context.Context, siteID int64, istenen string) (string, bool) {
	varsayilan := s.i18n.Default()
	if istenen == "" || istenen == varsayilan {
		return varsayilan, false
	}
	if s.yerelIcerikVar(ctx, siteID, istenen) {
		return istenen, false
	}
	return varsayilan, true
}

// yerelHreflang ana sayfa/bölüm sayfaları için alternatif dil etiketleri.
//
// Yalnız GERÇEKTEN içeriği olan diller listelenir: boş bir dil sayfasına
// hreflang vermek, arama motorunu boş sayfaya çağırmak olurdu. Kendi dil
// her zaman eklenir; x-default varsayılan dile verilir.
func (s *Server) yerelHreflang(ctx context.Context, absBase string, siteID int64, istenen string, yolBicimi func(locale string) string) []langLink {
	taban := strings.TrimRight(absBase, "/")
	kendi := istenen
	if kendi == "" {
		kendi = s.i18n.Default()
	}
	out := []langLink{{
		Code: kendi,
		URL:  taban + yolBicimi(kendi),
	}}
	for _, l := range s.i18n.Enabled() {
		if l.Code == kendi || l.Code == "tr" {
			continue
		}
		if !s.yerelIcerikVar(ctx, siteID, l.Code) {
			continue
		}
		out = append(out, langLink{Code: l.Code, URL: taban + yolBicimi(l.Code)})
	}
	// tr varsayılan dil: her zaman alternatif olarak listelenir (kendi dil
	// değilse) çünkü sayfa ona düşebiliyor.
	if kendi != s.i18n.Default() {
		out = append(out, langLink{Code: s.i18n.Default(), URL: taban + yolBicimi(s.i18n.Default())})
	}
	if len(out) < 2 {
		// Tek dil: hreflang anlamsız.
		return nil
	}
	out = append(out, langLink{Code: "x-default", URL: taban + yolBicimi(s.i18n.Default())})
	return out
}

// siteIDOr1 nil/0 site kimliğini 1'e çeker (repo katmanıyla aynı kural).
func siteIDOr1(id int64) int64 {
	if id <= 0 {
		return 1
	}
	return id
}

// categoryLangName kategori adının yerelleştirilmiş hâlini verir.
//
// Kategoriler tablosunda dil kolonu yok (ad tek: "Gündem"). Çeviri
// `translations` tablosunda `category.<slug>` anahtarıyla tutulur; yoksa
// veritabanı adı kullanılır. Böylece şema değişmeden bölüm başlıkları
// çevrilebiliyor.
func (s *Server) categoryLangName(ctx context.Context, locale, slug, yedek string) string {
	if s.i18n == nil || locale == "" || locale == "tr" || slug == "" {
		return yedek
	}
	anahtar := "category." + slug
	v := s.i18n.T(locale, anahtar)
	if v == anahtar || strings.TrimSpace(v) == "" {
		return yedek
	}
	return v
}

// sitemapDilleri içeriği olan dilleri döndürür (sitemap/hreflang için).
func (s *Server) sitemapDilleri(ctx context.Context, siteID int64) []string {
	var out []string
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT locale FROM contents WHERE site_id=$1 AND status='yayinda'`,
		siteIDOr1(siteID))
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err == nil && l != "" {
			out = append(out, l)
		}
	}
	return out
}

// contentSiteID yayımlanmış içerikten site kimliği (yardımcı).
func contentSiteID(c repo.Content) int64 {
	if c.SiteID > 0 {
		return c.SiteID
	}
	return 1
}

// localeText verilen anahtarın çevirisini, yoksa yedeği döndürür.
//
// `T` anahtar bulunamazsa anahtarın kendisini döndürüyor; bu, ekrana
// "site.suffix" yazılmasına yol açar. Bu yardımcı o durumu yedeğe çevirir.
func (s *Server) localeText(locale, anahtar, yedek string) string {
	if s.i18n == nil || locale == "" || locale == "tr" {
		return yedek
	}
	v := s.i18n.T(locale, anahtar)
	if v == anahtar || strings.TrimSpace(v) == "" {
		return yedek
	}
	return v
}

// kategoriIcerikDili kategori sayfası için kullanılacak dili verir.
//
// Global `icerikDili` yeterli değil: sitede İngilizce içerik olabilir ama o
// KATEGORİDE hiç olmayabilir (ölçüldü: /en varken /en/kategori/otomobil
// boştu). Bu yüzden kategori (ve alt kategorileri) bazında bakılır.
//
// İkinci değer düşüş bayrağıdır (canonical/noindex için).
func (s *Server) kategoriIcerikDili(ctx context.Context, siteID int64, slug, istenen string) (string, bool) {
	varsayilan := s.i18n.Default()
	if istenen == "" || istenen == varsayilan {
		return varsayilan, false
	}
	var bir int
	err := s.pool.QueryRow(ctx, `
		SELECT 1 FROM contents c
		JOIN categories kat ON kat.id = c.category_id
		WHERE c.site_id = $1 AND c.locale = $2 AND c.status = 'yayinda'
		  AND (kat.slug = $3
		       OR kat.parent_id = (SELECT id FROM categories WHERE slug = $3 LIMIT 1))
		LIMIT 1`, siteIDOr1(siteID), istenen, slug).Scan(&bir)
	if err == nil {
		return istenen, false
	}
	return varsayilan, true
}
