package httpapi

import (
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// Basın İlan Kurumu — İnternet Haber Sitelerinin Teknik Kurallarına Dair
// Tebliğ'in istediği meta etiketleri ve reklam işaretleri.
//
// Neden gerekiyor: site resmî ilan yayımlıyor (/resmi-ilanlar). Tebliğ, resmî
// ilan yayınlayan veya bekleme süresinde bulunan internet haber sitelerini
// kapsıyor. Kurum, uyumu otomatik teknik kontrolle ölçüyor; etiketler yoksa
// kontrol başarısız sayılıyor.
//
// Kaynak standardı (Tebliğ, "Kaynak standartları" maddesi) beş durum
// tanımlıyor:
//
//	Fikir işçisi        <meta name="articleAuthor" content="AHMET YILMAZ">
//	Haber ajansı        <meta name="articleAuthor" content="AA">
//	Bülten              <meta name="articleAuthor" content="BULTEN">
//	Haber merkezi       <meta name="articleAuthor" content="HABER MERKEZI">
//	Alıntı (iktibas)    <meta name="articleAuthor" content="www.kaynak.com">
//
// KRİTİK KURAL: değer, "haber veya köşe yazısı içeriğinde belirtilen kaynak
// bilgisiyle AYNI" olmak zorunda. Bu yüzden burada hiçbir büyük/küçük harf
// dönüşümü YAPILMAZ — sayfada "Ahmet Yılmaz" yazarken meta'ya "AHMET YILMAZ"
// koymak iki değeri ayrıştırır. Tebliğdeki örneklerin büyük harfli olması
// biçim zorunluluğu değil, örnek yazımıdır. (Ayrıca Türkçe'de i/İ dönüşümü
// ayrı bir tuzak; hiç dokunmamak en güvenlisi.)
const bikHaberMerkezi = "HABER MERKEZI"

// bikMeta bir içerik sayfası için tebliğin istediği meta değerleridir.
// Boş alanlar şablonda basılmaz: yanlış değer basmaktansa hiç basmamak
// yeğdir, çünkü Kurum içerikle meta'nın tutarlılığını da denetliyor.
type bikMeta struct {
	Title     string // <meta name="title">
	URL       string // <meta name="url"> — tam (mutlak) adres
	Section   string // <meta name="articleSection"> — kategori
	Published string // <meta name="datePublished"> — RFC3339
	Modified  string // <meta name="dateModified"> — RFC3339
	Author    string // <meta name="articleAuthor"> — kaynak
}

// buildBikMeta içerik ve yazar bilgisinden meta değerlerini üretir.
//
// canonical mutlak içerik adresidir; boşsa URL alanı boş bırakılır (göreli
// adres basmak tebliğin "tam URL" beklentisini karşılamaz).
func buildBikMeta(c repo.Content, author *repo.PersonRef, canonical string) bikMeta {
	m := bikMeta{
		Title:   strings.TrimSpace(c.Title),
		URL:     strings.TrimSpace(canonical),
		Section: strTrimPtr(c.CategoryName),
		Author:  bikArticleAuthor(c, author),
	}
	// Tarihler UTC olarak (…Z) yazılır. Yerel saat kullanılsaydı offset "+03:00"
	// olurdu ve html/template öznitelik içindeki "+" karakterini &#43; olarak
	// kaçırırdı. Tarayıcı bunu sorunsuz çözer, ama Kurum'un otomatik kontrolü
	// ham dize eşleştirmesi yapıyorsa tarih okunamaz görünür. "Z" bu riski
	// tamamen ortadan kaldırır ve RFC3339 açısından da kesindir.
	if c.PublishedAt != nil && !c.PublishedAt.IsZero() {
		m.Published = c.PublishedAt.UTC().Format(time.RFC3339)
	}
	if !c.UpdatedAt.IsZero() {
		m.Modified = c.UpdatedAt.UTC().Format(time.RFC3339)
	}
	// Güncelleme yayından önce görünüyorsa (saat kayması, toplu içe aktarma)
	// yayın zamanını kullan: dateModified < datePublished mantıksızdır.
	if m.Published != "" && c.PublishedAt != nil &&
		!c.UpdatedAt.IsZero() && c.UpdatedAt.Before(*c.PublishedAt) {
		m.Modified = m.Published
	}
	return m
}

// bikArticleAuthor kaynak değerini tebliğin standardına göre seçer.
//
// Sıra önemli: imzalı yazı varsa fikir işçisi adı esastır; yoksa içeriğin
// kaynağı (ajans adı, bülten veren kuruluş veya iktibas edilen site) yazılır;
// hiçbiri yoksa haber merkezi.
func bikArticleAuthor(c repo.Content, author *repo.PersonRef) string {
	if author != nil {
		if ad := strings.TrimSpace(author.Name); ad != "" {
			return ad
		}
	}
	if ad := strTrimPtr(c.SourceName); ad != "" {
		return ad
	}
	return bikHaberMerkezi
}

func strTrimPtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}
