package repo

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// gazeteMansetKaynakURL kaynak sayfa.
const gazeteMansetKaynakURL = "https://www.gazeteoku.com/"

// gazeteMansetTimeout: sayfa ~213 KB ve 0.44 sn'de dönüyor; 15 sn bol.
const gazeteMansetTimeout = 15 * time.Second

var (
	// Kapak görseli deseni. Gruplar: 1) CDN yol öneki  2-4) tarih  5) slug.
	gazeteKapakRe = regexp.MustCompile(
		`https://i\.gazeteoku\.com/(\d+)/\d+/\d+/storage/files/images/(\d{4})/(\d{2})/(\d{2})/([a-z0-9-]+?)-\d{4}-\d{2}-\d{2}-[a-z0-9]+\.jpg`)
	// Boyut parçasını değiştirmek için (CDN vekili).
	gazeteBoyRe = regexp.MustCompile(`^(https://i\.gazeteoku\.com/\d+)/\d+/\d+/`)
)

var gazeteSiraListesi = []string{
	"hurriyet", "sabah", "nefes", "sozcu", "cumhuriyet", "milliyet", "posta",
	"takvim", "turkiye-gazetesi", "yeni-safak", "karar", "turkgun", "fanatik",
	"fotomac", "aksam", "dunya", "birgun", "evrensel", "aydinlik-gazetesi",
	"yenicag", "yeni-akit", "yeni-asya", "milli-gazete", "korkusuz", "oksijen",
	"agos", "milat", "dirilis-postasi", "yenibirlik", "tavir-gazetesi",
	"daily-sabah", "hurriyet-daily-news",
}

// gazeteAdlari slug → görünen ad.
var gazeteAdlari = map[string]string{
	"hurriyet": "Hürriyet", "sabah": "Sabah", "nefes": "Nefes",
	"sozcu": "Sözcü", "cumhuriyet": "Cumhuriyet", "milliyet": "Milliyet",
	"posta": "Posta", "takvim": "Takvim", "turkiye-gazetesi": "Türkiye Gazetesi",
	"yeni-safak": "Yeni Şafak", "karar": "Karar", "turkgun": "Türkgün",
	"fanatik": "Fanatik", "fotomac": "Fotomaç", "aksam": "Akşam",
	"dunya": "Dünya", "birgun": "BirGün", "evrensel": "Evrensel",
	"aydinlik-gazetesi": "Aydınlık", "yenicag": "Yeniçağ",
	"yeni-akit": "Yeni Akit", "yeni-asya": "Yeni Asya",
	"milli-gazete": "Millî Gazete", "korkusuz": "Korkusuz",
	"oksijen": "Oksijen", "agos": "Agos", "milat": "Milat",
	"dirilis-postasi": "Diriliş Postası", "yenibirlik": "Yeni Birlik",
	"tavir-gazetesi": "Tavır", "daily-sabah": "Daily Sabah",
	"hurriyet-daily-news": "Hürriyet Daily News",
}

// gazeteSiraOf bir gazetenin sırasını döndürür; bilinmeyen gazete sona gider.
func gazeteSiraOf(slug string) int {
	for i, s := range gazeteSiraListesi {
		if s == slug {
			return i
		}
	}
	return len(gazeteSiraListesi)
}

// gazeteAdiOf slug için görünen ad; bilinmeyen slug başlık biçimine çevrilir.
func gazeteAdiOf(slug string) string {
	if ad, ok := gazeteAdlari[slug]; ok {
		return ad
	}
	parca := strings.Split(slug, "-")
	for i, p := range parca {
		if p == "" {
			continue
		}
		parca[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parca, " ")
}

// GazeteKaynakURL kaynak sayfanın adresidir (test/panel için dışa açık).
func GazeteKaynakURL() string { return gazeteMansetKaynakURL }

func gazeteKapakCoz(ham []byte) []gazeteKapak {
	s := string(ham)
	eslesme := gazeteKapakRe.FindAllStringSubmatch(s, -1)
	if len(eslesme) == 0 {
		return nil
	}
	gorulen := map[string]bool{}
	var out []gazeteKapak
	for _, m := range eslesme {
		slug := m[5]
		if gorulen[slug] {
			continue
		}
		gorulen[slug] = true
		out = append(out, gazeteKapak{
			Slug:   slug,
			Ad:     gazeteAdiOf(slug),
			Tarih:  m[2] + "-" + m[3] + "-" + m[4],
			Gorsel: gazeteBoyRe.ReplaceAllString(m[0], "${1}/900/1310/"),
			Kucuk:  gazeteBoyRe.ReplaceAllString(m[0], "${1}/300/437/"),
			Sira:   gazeteSiraOf(slug),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sira < out[j].Sira })
	return out
}

// gazeteKapak ayrıştırılmış tek kapak (test edilebilirlik için iç tip).
type gazeteKapak struct {
	Slug   string
	Ad     string
	Tarih  string // YYYY-MM-DD (görselin yolundan)
	Gorsel string // 900x1310
	Kucuk  string // 300x437
	Sira   int
}

// GazeteMansetCoz test/araç kullanımı için dışa açık sarmalayıcı.
func GazeteMansetCoz(ham []byte) []gazeteKapak { return gazeteKapakCoz(ham) }

func gazeteMansetTaze(ctx context.Context, pool DB, siteID int64) bool {
	var sonTarih *string
	var sonDeneme *time.Time
	_ = pool.QueryRow(ctx, `
		SELECT max(page_date)::text, max(updated_at)
		FROM newspaper_pages WHERE site_id=$1`, siteID).Scan(&sonTarih, &sonDeneme)
	if sonTarih != nil && *sonTarih == time.Now().Format("2006-01-02") {
		return true
	}
	// Bugünün kapakları yoksa (kaynak henüz yayınlamadı) 50 dakikada bir
	// yeniden denenir.
	if sonDeneme != nil && time.Since(*sonDeneme) < 50*time.Minute {
		return true
	}
	return false
}

func RefreshGazeteMansetleri(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	return refreshGazeteMansetleri(ctx, pool, false)
}

// FetchGazeteMansetleri panelden "kaynaktan çek" için zorunlu (force) çekim.
// Taze kontrolü atlanır: editör düğmeye bastıysa sonucu bekliyor.
func FetchGazeteMansetleri(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	return refreshGazeteMansetleri(ctx, pool, true)
}

func refreshGazeteMansetleri(ctx context.Context, pool *pgxpool.Pool, zorla bool) (int, error) {
	siteler, err := ListSites(ctx, pool)
	if err != nil {
		return 0, err
	}
	if len(siteler) == 0 {
		return 0, nil
	}

	// Ağ turu bir kez: tüm siteler aynı kaynağı kullanır.
	var cekilecek bool
	for _, site := range siteler {
		if zorla || !gazeteMansetTaze(ctx, pool, site.ID) {
			cekilecek = true
			break
		}
	}
	if !cekilecek {
		return 0, nil
	}

	ham, err := httpGetTarayici(ctx, gazeteMansetKaynakURL, gazeteMansetTimeout)
	if err != nil {
		return 0, fmt.Errorf("gazete manşet kaynağı: %w", err)
	}
	kapaklar := gazeteKapakCoz(ham)
	if len(kapaklar) == 0 {
		return 0, fmt.Errorf("gazete manşet kaynağında kapak bulunamadı (%d bayt)", len(ham))
	}

	toplam := 0
	for _, site := range siteler {
		if !zorla && gazeteMansetTaze(ctx, pool, site.ID) {
			continue
		}
		for _, k := range kapaklar {
			p := &NewspaperPage{
				SiteID:   site.ID,
				Name:     k.Ad,
				Slug:     k.Slug,
				PageDate: k.Tarih,
				ImageURL: k.Gorsel,
				ThumbURL: k.Kucuk,
				Sort:     k.Sira,
			}
			if err := UpsertNewspaperPage(ctx, pool, p); err != nil {
				return toplam, err
			}
			toplam++
		}
		// Tablo sonsuz büyümesin: 8 günden eski kapaklar silinir.
		_ = PruneNewspaperPages(ctx, pool, site.ID, time.Now().AddDate(0, 0, -8).Format("2006-01-02"))
	}
	return toplam, nil
}
