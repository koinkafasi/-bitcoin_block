// Command seed — tasarım incelemesi için gerçekçi haber + görsel verisi yükler.
//
// Kategori setini haberci tipi bir portala göre düzenler, halka açık haber RSS
// kaynaklarını çeker, görselleri (media:content / media:thumbnail / og:image)
// indirip medya deposuna yazar ve içerikleri "yayında" olarak ekler.
// Kaynak adı ve haber URL'si içerik kaydında saklanır (atıf).
//
// Kullanım:
//
//	STORAGE_DIR=/home/alper/theblock/storage go run ./cmd/seed -per-feed 18
//
// Bayraklar:
//
//	-per-feed 18    her kaynaktan alınacak azami haber sayısı
//	-workers 6      eşzamanlı indirme işçisi
//	-only <slug>    yalnızca verilen kategori slug'ını doldur
//	-flags          manşet/son dakika bayraklarını yeniden dağıt
package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"html"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/config"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- kategori kataloğu (haberci tipi portal yapısı) ----

type catDef struct {
	Name   string
	Slug   string
	Icon   string
	Sort   int
	Header bool
	// FromSlug: bu kategori önceden bu slug ile varsa yerinde yeniden
	// adlandırılır (bağlantılar ve içerikler korunur).
	FromSlug string
}

// catDefs haberpi.com canlı menüsüyle birebir hizalanmıştır.
// Üst menü ilk 6, kalanı "Diğer" açılır menüsünde görünür
// (her iki sitede de aynı kural).
var catDefs = []catDef{
	{"Gündem", "gundem", "news", 1, true, "guncel"},
	{"Siyaset", "siyaset", "vote", 2, true, ""},
	{"Ekonomi", "ekonomi", "coins", 3, true, ""},
	{"Dünya", "dunya", "globe", 4, true, ""},
	{"Spor", "spor", "ball", 5, true, ""},
	{"Yaşam", "yasam", "coffee", 6, true, ""},
	{"Teknoloji", "teknoloji", "device", 7, true, ""},
	{"Eğitim-kültür", "egitim-kultur", "book", 8, true, "egitim"},
	{"Asayiş", "asayis", "shield", 9, true, ""},
	{"İlçeler", "ilceler", "building", 10, true, ""},
	{"Sağlık", "saglik", "health", 11, true, ""},
	{"Bilim", "bilim", "flask", 12, true, ""},
	{"Sanat", "sanat", "palette", 13, true, ""},
	{"Turizm", "turizm", "plane", 14, true, ""},
	{"Diğer", "diger", "grid", 99, false, ""},
	// theblock'a özgü, haberci menüsünde yok: menüyü bozmamak için
	// başlıkta gösterilmez (URL'leri ve içerikleri çalışır durumda kalır).
	{"Oyun", "oyun", "game", 90, false, ""},
}

// hierarchyExtras: migration 0054 ile eklenen alt kategoriler. Seed tekrar
// çalıştığında boş kategoriler silinmesin diye keep listesine eklenir.
var hierarchyExtras = []string{
	"turkiye-ekonomisi", "dunya-ekonomisi", "kripto-para", "tarim", "sanayi-uretim",
	"bankacilik", "hisse-senetleri", "yatirim-fonlari", "banka-gise-kurlari", "kap-haberleri",
	"futbol", "basketbol", "atletizm", "dunya-kupasi", "golf", "gures", "extreme-sporlar", "diger-sporlar",
	"afet-kaza", "analiz", "guvenlik", "ozgun", "toplum", "ramazan",
	"afrika", "amerika", "asya", "avrupa", "ortadogu",
	"donanim", "internet", "mobil-teknolojiler", "robotik", "teknoloji-haberleri", "uzay", "yazilim",
	"e-spor", "mobil-oyunlar", "oyun-haberleri", "oyun-incelemeleri",
	"aktuel", "genclik", "kadin", "kisisel-gelisim", "doga-kesif", "gezi-rotalari",
	"sehir-rehberi", "seyahat-rehberi", "yeme-icme", "hayvanlar-alemi", "hobi", "mizah",
	"derleme-secki", "dusunce-fikir", "edebiyat", "kitap", "medya-dijital-kultur",
	"mekan", "muzik", "portreler", "sinema", "tarih", "mimari",
}

// otherCatDefs migration 0056'daki GZT alt kategorileri (fresh seed için).
type otherCatDef struct {
	Name, Slug, Parent string
	Sort               int
}

var otherCatDefs = []otherCatDef{
	{"Afrika", "afrika", "dunya", 1}, {"Amerika", "amerika", "dunya", 2},
	{"Asya", "asya", "dunya", 3}, {"Avrupa", "avrupa", "dunya", 4},
	{"Orta Doğu", "ortadogu", "dunya", 5},
	{"Donanım", "donanim", "teknoloji", 1}, {"İnternet", "internet", "teknoloji", 2},
	{"Mobil Teknolojiler", "mobil-teknolojiler", "teknoloji", 3},
	{"Robotik", "robotik", "teknoloji", 4}, {"Teknoloji Haberleri", "teknoloji-haberleri", "teknoloji", 5},
	{"Uzay", "uzay", "teknoloji", 6}, {"Yazılım", "yazilim", "teknoloji", 7},
	{"E-spor", "e-spor", "teknoloji", 8}, {"Mobil Oyunlar", "mobil-oyunlar", "teknoloji", 9},
	{"Oyun Haberleri", "oyun-haberleri", "teknoloji", 10},
	{"Oyun İncelemeleri", "oyun-incelemeleri", "teknoloji", 11},
	{"Aktüel", "aktuel", "yasam", 1}, {"Gençlik", "genclik", "yasam", 2},
	{"Kadın", "kadin", "yasam", 3}, {"Kişisel Gelişim", "kisisel-gelisim", "yasam", 4},
	{"Doğa & Keşif", "doga-kesif", "yasam", 5}, {"Gezi Rotaları", "gezi-rotalari", "yasam", 6},
	{"Şehir Rehberi", "sehir-rehberi", "yasam", 7}, {"Seyahat Rehberi", "seyahat-rehberi", "yasam", 8},
	{"Yeme-İçme", "yeme-icme", "yasam", 9}, {"Hayvanlar Alemi", "hayvanlar-alemi", "yasam", 10},
	{"Hobi", "hobi", "yasam", 11}, {"Mizah", "mizah", "yasam", 12},
	{"Derleme-Seçki", "derleme-secki", "egitim-kultur", 1},
	{"Düşünce-Fikir", "dusunce-fikir", "egitim-kultur", 2},
	{"Edebiyat", "edebiyat", "egitim-kultur", 3}, {"Kitap", "kitap", "egitim-kultur", 4},
	{"Medya-Dijital Kültür", "medya-dijital-kultur", "egitim-kultur", 5},
	{"Mekan", "mekan", "egitim-kultur", 6}, {"Müzik", "muzik", "egitim-kultur", 7},
	{"Portreler", "portreler", "egitim-kultur", 8}, {"Sinema", "sinema", "egitim-kultur", 9},
	{"Tarih", "tarih", "egitim-kultur", 10}, {"Mimari", "mimari", "egitim-kultur", 11},
}

// ---- haber kaynakları ----

type feedDef struct {
	Name   string
	URL    string
	Cat    string
	Source string
	// Keywords verilirse yalnızca başlık/özet içinde bu kelimelerden biri
	// geçen haberler alınır (kategoriye özel kaynak bulunamadığında).
	Keywords []string
}

var feedDefs = []feedDef{
	// --- Güncel ---
	{"Anadolu Ajansı Güncel", "https://www.aa.com.tr/tr/rss/default?cat=guncel", "guncel", "Anadolu Ajansı", nil},
	{"Sözcü Gündem", "https://www.sozcu.com.tr/feeds-rss-category-gundem", "guncel", "Sözcü", nil},
	{"Hürriyet Gündem", "https://www.hurriyet.com.tr/rss/gundem", "guncel", "Hürriyet", nil},
	// --- Siyaset ---
	{"Hürriyet Siyaset", "https://www.hurriyet.com.tr/rss/siyaset", "siyaset", "Hürriyet", nil},
	// --- Ekonomi ---
	{"Anadolu Ajansı Ekonomi", "https://www.aa.com.tr/tr/rss/default?cat=ekonomi", "ekonomi", "Anadolu Ajansı", nil},
	{"Sözcü Ekonomi", "https://www.sozcu.com.tr/feeds-rss-category-ekonomi", "ekonomi", "Sözcü", nil},
	// --- Yaşam ---
	{"Sözcü Yaşam", "https://www.sozcu.com.tr/feeds-rss-category-yasam", "yasam", "Sözcü", nil},
	// --- Asayiş (kategoriye özel feed yok → anahtar kelime süzgeci) ---
	{"Hürriyet Gündem / asayiş", "https://www.hurriyet.com.tr/rss/gundem", "asayis", "Hürriyet", []string{
		"asayiş", "asayis", "polis", "savcı", "savci", "operasyon", "kaza", "kontrol",
		"deprem", "değme", "yangın", "yangin", "cinayet", "darp", "gözaltı", "gozalti",
		"trafik", "hırsızlık", "hirsizlik", "uyuşturucu", "uyusturucu", "güvenlik",
		"emniyet", "jandarma", "itfaiye", "sel baskını", "sel",
	}},
	// --- İlçeler (yerel/bölgesel içerik süzgeci) ---
	{"Hürriyet Yaşam / ilçeler", "https://www.hurriyet.com.tr/rss/yasam", "ilceler", "Hürriyet", []string{
		"ilçe", "ilce", "belediye", "kaymakamlık", "kaymakamlik", "köy", "köyü",
		"mahalle", "belediyesi", "vali", "muhtar", "bölge", "ilçesi", "ilçeler",
	}},
	// --- Spor ---
	{"Anadolu Ajansı Spor", "https://www.aa.com.tr/tr/rss/default?cat=spor", "spor", "Anadolu Ajansı", nil},
	{"Sözcü Spor", "https://www.sozcu.com.tr/feeds-rss-category-spor", "spor", "Sözcü", nil},
	{"Hürriyet Spor", "https://www.hurriyet.com.tr/rss/spor", "spor", "Hürriyet", nil},
	// --- Eğitim-kültür ---
	{"Hürriyet Eğitim", "https://www.hurriyet.com.tr/rss/egitim", "egitim-kultur", "Hürriyet", nil},
	{"Anadolu Ajansı Eğitim", "https://www.aa.com.tr/tr/rss/default?cat=egitim", "egitim-kultur", "Anadolu Ajansı", nil},
	// --- Teknoloji ---
	{"Webrazzi", "https://webrazzi.com/feed/", "teknoloji", "Webrazzi", nil},
	{"ShiftDelete", "https://shiftdelete.net/feed/", "teknoloji", "ShiftDelete", nil},
	// --- Sağlık ---
	{"Anadolu Ajansı Sağlık", "https://www.aa.com.tr/tr/rss/default?cat=saglik", "saglik", "Anadolu Ajansı", nil},
	{"Sözcü Sağlık", "https://www.sozcu.com.tr/feeds-rss-category-saglik", "saglik", "Sözcü", nil},
	// --- Bilim ---
	{"Anadolu Ajansı Bilim", "https://www.aa.com.tr/tr/rss/default?cat=bilim-teknoloji", "bilim", "Anadolu Ajansı", nil},
	// --- Sanat ---
	{"Sözcü Yaşam / sanat", "https://www.sozcu.com.tr/feeds-rss-category-yasam", "sanat", "Sözcü", []string{
		"sanat", "kültür", "kultur", "tiyatro", "sinema", "film", "sergi", "sergi",
		"müzik", "muzik", "konser", "sahne", "kitap", "yazar", "sculpture", "resim",
		"müze", "muze", "tarih", "efsane",
	}},
	// --- Turizm ---
	{"Sözcü Yaşam / turizm", "https://www.sozcu.com.tr/feeds-rss-category-yasam", "turizm", "Sözcü", []string{
		"turizm", "turist", "otel", "tatil", "seyahat", "havaalan", "liman",
		"plaj", "kayak", "pansiyon", "kongre", "festival", "rota", " gezi",
	}},
	// --- Diğer ---
	{"Euronews Türkiye", "https://tr.euronews.com/rss", "diger", "Euronews", nil},
	// --- theblock'a özgü (menüde yok) ---
	{"IGN Türkiye", "https://tr.ign.com/rss", "oyun", "IGN", nil},
	{"Anadolu Ajansı Dünya", "https://www.aa.com.tr/tr/rss/default?cat=dunya", "dunya", "Anadolu Ajansı", nil},
}

// ---- RSS ----

type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel *struct {
		Title string    `xml:"title"`
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
	// Atom biçimli kaynaklar (bazı teknoloji/oyun feed'leri)
	Entries []rssItem `xml:"entry"`
}

// linkField hem RSS (<link>metin</link>) hem Atom (<link href="..."/>)
// biçimini tek alanda toplar.
type linkField struct {
	Text string `xml:",chardata"`
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type rssItem struct {
	Title       string    `xml:"title"`
	Link        linkField `xml:"link"`
	Description string    `xml:"description"`
	Summary     string    `xml:"summary"`
	PubDate     string    `xml:"pubDate"`
	Published   string    `xml:"published"`
	Updated     string    `xml:"updated"`
	// AA örneği: gövde metninde düz <image>URL</image>
	Image     string `xml:"image"`
	Enclosure struct {
		URL  string `xml:"url,attr"`
		Type string `xml:"type,attr"`
	} `xml:"enclosure"`
	MediaContent []struct {
		URL    string `xml:"url,attr"`
		Medium string `xml:"medium,attr"`
	} `xml:"media>content"`
	MediaThumb []struct {
		URL string `xml:"url,attr"`
	} `xml:"media>thumbnail"`
}

// itemURL haber bağlantısını RSS/Atom farkını gözetmeden döndürür.
func (it rssItem) itemURL() string {
	if t := strings.TrimSpace(it.Link.Text); t != "" {
		return t
	}
	if h := strings.TrimSpace(it.Link.Href); h != "" {
		return h
	}
	return ""
}

// itemText özet metnini RSS/Atom farkını gözetmeden döndürür.
func (it rssItem) itemText() string {
	if d := strings.TrimSpace(it.Description); d != "" {
		return d
	}
	return strings.TrimSpace(it.Summary)
}

// itemDate yayın tarihini RSS/Atom farkını gözetmeden döndürür.
func (it rssItem) itemDate() string {
	for _, d := range []string{it.PubDate, it.Published, it.Updated} {
		if strings.TrimSpace(d) != "" {
			return d
		}
	}
	return ""
}

// items tüm biçimlerden haber listesini döndürür.
func (f rssFeed) items() []rssItem {
	if f.Channel != nil && len(f.Channel.Items) > 0 {
		return f.Channel.Items
	}
	return f.Entries
}

const ua = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124 Safari/537.36"

var client = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 8,
		TLSClientConfig:     nil,
	},
}

var (
	reHTMLTags    = regexp.MustCompile(`(?s)<[^>]+>`)
	reSpaces      = regexp.MustCompile(`\s+`)
	reOgImage     = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']`)
	reOgImageAlt  = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`)
	reOgDesc      = regexp.MustCompile(`(?i)<meta[^>]+(?:name|property)=["'](?:og:description|description)["'][^>]+content=["']([^"']*)["']`)
	reTitleSuffix = regexp.MustCompile(`\s+[-–|]\s+[^-–|]{2,40}$`)
)

func main() {
	perFeed := flag.Int("per-feed", 18, "her kaynaktan alınacak azami haber")
	workers := flag.Int("workers", 6, "eşzamanlı işçi")
	only := flag.String("only", "", "yalnızca bu kategori slug'ı")
	flagsOnly := flag.Bool("flags", false, "yalnızca manşet/son dakika bayraklarını dağıt")
	withTags := flag.Bool("tags", true, "başlıktan anahtar kelime etiketleri üret (alaka motoru)")
	galleryN := flag.Int("galeri", 8, "görselli haberlerden üretilecek foto/video galeri içeriği (0=kapalı)")
	flag.Parse()

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL boş")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("havuz: %v", err)
	}
	defer pool.Close()

	if err := ensureCategories(ctx, pool); err != nil {
		log.Fatalf("kategoriler: %v", err)
	}
	catID, err := categoryMap(ctx, pool)
	if err != nil {
		log.Fatalf("kategori haritası: %v", err)
	}

	var authorID int64 = 1
	if err := pool.QueryRow(ctx, `SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&authorID); err != nil {
		log.Fatalf("yazar: %v", err)
	}

	if *flagsOnly {
		if err := assignFlags(ctx, pool); err != nil {
			log.Fatalf("bayraklar: %v", err)
		}
		log.Println("bayraklar güncellendi")
		return
	}

	// iş kuyruğu
	type job struct {
		item  rssItem
		feed  feedDef
		catID int64
	}
	var jobs []job
	for _, f := range feedDefs {
		if *only != "" && f.Cat != *only {
			continue
		}
		items, err := fetchFeed(f.URL)
		if err != nil {
			log.Printf("!! %s: %v", f.Name, err)
			continue
		}
		cid, ok := catID[f.Cat]
		if !ok {
			log.Printf("!! bilinmeyen kategori %q", f.Cat)
			continue
		}
		n := 0
		for _, it := range items {
			title := cleanTitle(it.Title)
			link := it.itemURL()
			if title == "" || link == "" {
				continue
			}
			// Anahtar kelime süzgeci: kategoriye özel kaynak yoksa
			// (Asayiş, İlçeler, Sanat, Turizm) ilgili haberleri seçer.
			if len(f.Keywords) > 0 && !matchesKeywords(title+" "+cleanText(it.itemText()), f.Keywords) {
				continue
			}
			jobs = append(jobs, job{
				item: rssItem{
					Title:        title,
					Link:         linkField{Text: link},
					Description:  it.itemText(),
					PubDate:      it.itemDate(),
					Image:        it.Image,
					MediaContent: it.MediaContent,
					MediaThumb:   it.MediaThumb,
					Enclosure:    it.Enclosure,
				},
				feed: f, catID: cid,
			})
			n++
			if n >= *perFeed {
				break
			}
		}
		log.Printf("   %-26s %d haber", f.Name, n)
	}

	var wg sync.WaitGroup
	ch := make(chan job)
	var mu sync.Mutex
	added, skipped := 0, 0
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range ch {
				ok, err := seedOne(ctx, pool, authorID, j.catID, j.feed, j.item, *withTags)
				mu.Lock()
				if err != nil {
					log.Printf("   x %s — %v", trunc(j.item.Title, 48), err)
				} else if ok {
					added++
				} else {
					skipped++
				}
				mu.Unlock()
			}
		}()
	}
	for _, j := range jobs {
		ch <- j
	}
	close(ch)
	wg.Wait()

	log.Printf("== eklendi: %d, atlandı: %d ==", added, skipped)

	// Foto/Video galeri içeriği üret (blokların boş kalmasın).
	if *galleryN > 0 {
		if n, err := seedGalleries(ctx, pool, authorID, *galleryN); err != nil {
			log.Printf("!! galeri: %v", err)
		} else if n > 0 {
			log.Printf("== galeri içeriği: %d ==", n)
		}
	}

	// Etiketsiz kalan yayındaki içerikleri de etiketle (alaka motoru için).
	if *withTags {
		if n, err := backfillTags(ctx, pool, 5); err != nil {
			log.Printf("!! etiketleme: %v", err)
		} else if n > 0 {
			log.Printf("== etiketlenen içerik: %d ==", n)
		}
	}

	if err := assignFlags(ctx, pool); err != nil {
		log.Fatalf("bayraklar: %v", err)
	}
	report(ctx, pool)
}

// ---- kategori yönetimi ----

func ensureCategories(ctx context.Context, pool *pgxpool.Pool) error {
	for _, c := range catDefs {
		// Önceki slug'dan gelen kategori varsa yerinde yeniden adlandır
		// (içerikler ve menü bağlantıları korunur).
		if c.FromSlug != "" && c.FromSlug != c.Slug {
			var oldID int64
			if err := pool.QueryRow(ctx,
				`SELECT id FROM categories WHERE site_id=1 AND slug=$1`, c.FromSlug).Scan(&oldID); err == nil {
				if _, err := pool.Exec(ctx,
					`UPDATE categories SET slug=$2, name=$3, in_header=$4, in_menu=true,
					    in_footer=true, status='yayinda', sort_order=$5, icon=$6
					 WHERE id=$1`,
					oldID, c.Slug, c.Name, c.Header, c.Sort, c.Icon); err == nil {
					log.Printf("   ~ kategori yeniden adlandırıldı: %s -> %s", c.FromSlug, c.Slug)
				}
			}
		}
		var id int64
		err := pool.QueryRow(ctx, `SELECT id FROM categories WHERE site_id=1 AND slug=$1`, c.Slug).Scan(&id)
		if err != nil {
			if _, err := pool.Exec(ctx, `
				INSERT INTO categories (name, slug, in_header, in_menu, in_footer, status, sort_order, icon, site_id)
				VALUES ($1,$2,$3,true,true,'yayinda',$4,$5,1)`,
				c.Name, c.Slug, c.Header, c.Sort, c.Icon); err != nil {
				return fmt.Errorf("%s: %w", c.Slug, err)
			}
			log.Printf("   + kategori eklendi: %s", c.Name)
			continue
		}
		if _, err := pool.Exec(ctx, `
			UPDATE categories SET name=$2, in_header=$3, in_menu=true, in_footer=true,
				status='yayinda', sort_order=$4, icon=$5 WHERE id=$1`,
			id, c.Name, c.Header, c.Sort, c.Icon); err != nil {
			return fmt.Errorf("%s güncelleme: %w", c.Slug, err)
		}
	}
	// katalogda olmayan, içeriği olmayan kategorileri temizle
	keep := make([]string, 0, len(catDefs))
	for _, c := range catDefs {
		keep = append(keep, c.Slug)
	}
	keep = append(keep, hierarchyExtras...)
	rows, err := pool.Query(ctx, `SELECT id, slug FROM categories WHERE site_id=1`)
	if err != nil {
		return err
	}
	var drop []int64
	for rows.Next() {
		var id int64
		var slug string
		if rows.Scan(&id, &slug) != nil {
			continue
		}
		found := false
		for _, k := range keep {
			if k == slug {
				found = true
				break
			}
		}
		if found {
			continue
		}
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM contents WHERE category_id=$1`, id).Scan(&n); err != nil {
			// Sayim basarisizsa "bos" varsayimi kategoriyi siler: veri kaybi.
			// Hatayi yutmak yerine o kategoroyu atla ve gorunur uyar.
			log.Printf("   ! kategori #%d sayilamadi, silme atlandi: %v", id, err)
			continue
		}
		if n == 0 {
			drop = append(drop, id)
		}
	}
	rows.Close()
	for _, id := range drop {
		if _, err := pool.Exec(ctx, `DELETE FROM categories WHERE id=$1`, id); err != nil {
			log.Printf("   ! kategori #%d silinemedi: %v", id, err)
			continue
		}
		log.Printf("   - boş kategori silindi: #%d", id)
	}
	if err := applyCategoryHierarchy(ctx, pool); err != nil {
		return err
	}
	return applyCategoryTaxonomy(ctx, pool)
}

// applyCategoryTaxonomy 0058 ile aynı menü görünürlüğü/sıralamasını uygular.
func applyCategoryTaxonomy(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		UPDATE categories SET sort_order = CASE slug
			WHEN 'gundem' THEN 1 WHEN 'siyaset' THEN 2 WHEN 'ekonomi' THEN 3 WHEN 'dunya' THEN 4
			WHEN 'spor' THEN 5 WHEN 'yasam' THEN 6 WHEN 'teknoloji' THEN 7
			WHEN 'egitim-kultur' THEN 8 WHEN 'diger' THEN 99 ELSE sort_order END,
			in_header = CASE WHEN slug='diger' THEN FALSE ELSE TRUE END,
			in_menu = TRUE, in_footer = TRUE
		WHERE parent_id IS NULL`)
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `UPDATE categories SET in_menu=FALSE, in_footer=FALSE WHERE parent_id IS NOT NULL`); err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		UPDATE categories SET in_menu=TRUE, in_footer=TRUE, sort_order = CASE slug
			WHEN 'asayis' THEN 1 WHEN 'ilceler' THEN 2 WHEN 'afet-kaza' THEN 3 WHEN 'analiz' THEN 4
			WHEN 'turkiye-ekonomisi' THEN 1 WHEN 'dunya-ekonomisi' THEN 2 WHEN 'piyasalar' THEN 3
			WHEN 'borsa' THEN 4 WHEN 'doviz' THEN 5 WHEN 'altin' THEN 6 WHEN 'kripto-para' THEN 7
			WHEN 'emtia' THEN 8 WHEN 'faiz' THEN 9 WHEN 'enflasyon' THEN 10 WHEN 'bankacilik' THEN 11
			WHEN 'finans' THEN 12 WHEN 'is-dunyasi' THEN 13 WHEN 'enerji' THEN 14
			WHEN 'otomotiv' THEN 15 WHEN 'emlak' THEN 16 WHEN 'tarim' THEN 17 WHEN 'sanayi-uretim' THEN 18
			WHEN 'saglik' THEN 1 WHEN 'turizm' THEN 2 WHEN 'aktuel' THEN 3 WHEN 'kadin' THEN 4
			WHEN 'kisisel-gelisim' THEN 5 WHEN 'yeme-icme' THEN 6 WHEN 'doga-kesif' THEN 7
			WHEN 'gezi-rotalari' THEN 8 WHEN 'hobi' THEN 9 WHEN 'mizah' THEN 10
			WHEN 'futbol' THEN 1 WHEN 'basketbol' THEN 2 WHEN 'dunya-kupasi' THEN 3
			WHEN 'atletizm' THEN 4 WHEN 'diger-sporlar' THEN 5 WHEN 'golf' THEN 6
			WHEN 'gures' THEN 7 WHEN 'extreme-sporlar' THEN 8
			WHEN 'sanat' THEN 1 WHEN 'edebiyat' THEN 2 WHEN 'kitap' THEN 3 WHEN 'sinema' THEN 4
			WHEN 'tarih' THEN 5 WHEN 'mimari' THEN 6 WHEN 'muzik' THEN 7 WHEN 'dusunce-fikir' THEN 8
			WHEN 'bilim' THEN 1 WHEN 'oyun' THEN 2 WHEN 'donanim' THEN 3 WHEN 'internet' THEN 4
			WHEN 'mobil-teknolojiler' THEN 5 WHEN 'yazilim' THEN 6 WHEN 'uzay' THEN 7
			WHEN 'robotik' THEN 8 WHEN 'e-spor' THEN 9 WHEN 'oyun-haberleri' THEN 10
			WHEN 'afrika' THEN 1 WHEN 'amerika' THEN 2 WHEN 'asya' THEN 3 WHEN 'avrupa' THEN 4
			WHEN 'ortadogu' THEN 5 ELSE sort_order END
		WHERE parent_id IS NOT NULL AND (
			slug IN (
				'asayis','ilceler','afet-kaza','analiz',
				'turkiye-ekonomisi','dunya-ekonomisi','piyasalar','borsa','doviz','altin',
				'kripto-para','emtia','faiz','enflasyon','bankacilik','finans','is-dunyasi',
				'enerji','otomotiv','emlak','tarim','sanayi-uretim',
				'saglik','turizm','aktuel','kadin','kisisel-gelisim','yeme-icme',
				'doga-kesif','gezi-rotalari','hobi','mizah',
				'futbol','basketbol','dunya-kupasi','atletizm','diger-sporlar','golf','gures','extreme-sporlar',
				'sanat','edebiyat','kitap','sinema','tarih','mimari','muzik','dusunce-fikir',
				'bilim','oyun','donanim','internet','mobil-teknolojiler','yazilim','uzay','robotik',
				'e-spor','oyun-haberleri','afrika','amerika','asya','avrupa','ortadogu'
			)
			OR EXISTS (SELECT 1 FROM contents x WHERE x.category_id = categories.id AND x.status='yayinda')
		)`)
	return err
}

// applyCategoryHierarchy GZT tarzı alt kategori setini ve ana menü
// sadeleştirmesini idempotent biçimde uygular (migration 0054 ile aynı).
func applyCategoryHierarchy(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `UPDATE categories SET name='Gündem' WHERE slug='guncel'`); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `UPDATE categories SET slug='gundem' WHERE slug='guncel' AND NOT EXISTS (SELECT 1 FROM categories WHERE slug='gundem')`); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
		SELECT p.site_id, v.name, v.slug, p.id, FALSE, TRUE, TRUE, 'yayinda', v.sort, v.icon
		FROM categories p
		CROSS JOIN (VALUES
			('Türkiye Ekonomisi','turkiye-ekonomisi',13,'chart-line'),
			('Dünya Ekonomisi','dunya-ekonomisi',14,'globe'),
			('Kripto Para','kripto-para',15,'currency-bitcoin'),
			('Tarım','tarim',16,'plant'),
			('Sanayi & Üretim','sanayi-uretim',17,'building-factory'),
			('Bankacılık','bankacilik',18,'building-bank'),
			('Hisse Senetleri','hisse-senetleri',19,'chart-candle'),
			('Yatırım Fonları','yatirim-fonlari',20,'briefcase'),
			('Banka Gişe Kurları','banka-gise-kurlari',21,'cash'),
			('KAP Haberleri','kap-haberleri',22,'news')
		) AS v(name, slug, sort, icon)
		WHERE p.slug='ekonomi'
		ON CONFLICT (slug) DO NOTHING`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
		SELECT p.site_id, v.name, v.slug, p.id, FALSE, TRUE, TRUE, 'yayinda', v.sort, v.icon
		FROM categories p
		CROSS JOIN (VALUES
			('Futbol','futbol',1,'ball-football'),
			('Basketbol','basketbol',2,'ball-basketball'),
			('Atletizm','atletizm',3,'run'),
			('Dünya Kupası','dunya-kupasi',4,'trophy'),
			('Golf','golf',5,'golf'),
			('Güreş','gures',6,'karate'),
			('Ekstrem Sporlar','extreme-sporlar',7,'bike'),
			('Diğer Sporlar','diger-sporlar',8,'ball')
		) AS v(name, slug, sort, icon)
		WHERE p.slug='spor'
		ON CONFLICT (slug) DO NOTHING`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
		SELECT p.site_id, v.name, v.slug, p.id, FALSE, TRUE, TRUE, 'yayinda', v.sort, v.icon
		FROM categories p
		CROSS JOIN (VALUES
			('Afet & Kaza','afet-kaza',13,'alert-triangle'),
			('Analiz','analiz',14,'chart-line'),
			('Güvenlik','guvenlik',15,'shield'),
			('Özgün','ozgun',16,'star'),
			('Toplum','toplum',17,'users'),
			('Ramazan','ramazan',18,'moon')
		) AS v(name, slug, sort, icon)
		WHERE p.slug='gundem'
		ON CONFLICT (slug) DO NOTHING`)
	if err != nil {
		return err
	}
	for _, c := range otherCatDefs {
		if _, err := pool.Exec(ctx, `
			INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
			SELECT p.site_id, $1, $2, p.id, FALSE, TRUE, TRUE, 'yayinda', $3, 'news'
			FROM categories p WHERE p.slug = $4
			ON CONFLICT (slug) DO NOTHING`, c.Name, c.Slug, c.Sort, c.Parent); err != nil {
			return err
		}
	}
	_, err = pool.Exec(ctx, `
		UPDATE categories c
		SET parent_id = p.id, in_header = FALSE, in_menu = TRUE
		FROM categories p, (VALUES
			('asayis','gundem'),('ilceler','gundem'),('bilim','teknoloji'),
			('oyun','teknoloji'),('saglik','yasam'),('turizm','yasam'),('sanat','egitim-kultur')
		) AS m(child_slug, parent_slug)
		WHERE c.slug=m.child_slug AND p.slug=m.parent_slug AND p.site_id=c.site_id`)
	return err
}

func categoryMap(ctx context.Context, pool *pgxpool.Pool) (map[string]int64, error) {
	rows, err := pool.Query(ctx, `SELECT id, slug FROM categories WHERE site_id=1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]int64{}
	for rows.Next() {
		var id int64
		var slug string
		if rows.Scan(&id, &slug) == nil {
			m[slug] = id
		}
	}
	return m, rows.Err()
}

// foldTR metni Türkçe karakterlerden bağımsız hale getirir
// (asayiş → asayis, İstanbul → istanbul).
func foldTR(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'ı', 'İ':
			b.WriteRune('i')
		case 'ş', 'Ş':
			b.WriteRune('s')
		case 'ğ', 'Ğ':
			b.WriteRune('g')
		case 'ü', 'Ü':
			b.WriteRune('u')
		case 'ö', 'Ö':
			b.WriteRune('o')
		case 'ç', 'Ç':
			b.WriteRune('c')
		case '\u0307': // birleşen nokta (İ'nin ToLower çıktısı)
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// matchesKeywords metinde anahtar kelimelerden en az birini arar.
// HEM metin HEM anahtar kelimeler katlanır; aksi halde "asayiş" ile
// "asayis" eşleşmez ve süzgeç hiçbir şey seçmez.
func matchesKeywords(text string, words []string) bool {
	folded := foldTR(text)
	for _, w := range words {
		kw := foldTR(w)
		if kw != "" && strings.Contains(folded, kw) {
			return true
		}
	}
	return false
}

// ---- tek haber ekleme ----

func seedOne(ctx context.Context, pool *pgxpool.Pool, authorID, catID int64, f feedDef, it rssItem, withTags bool) (bool, error) {
	link := it.itemURL()
	if link == "" {
		return false, fmt.Errorf("bağlantı yok")
	}
	h := sha1.Sum([]byte(link))
	hash := hex.EncodeToString(h[:])

	var exists int
	if err := pool.QueryRow(ctx, `SELECT 1 FROM crawled_items WHERE url_hash=$1`, hash).Scan(&exists); err == nil {
		// Anahtar kelime süzgeçli "sözde kategori" (Asayiş, İlçeler, Sanat,
		// Turizm) için haber zaten başka bir kategoride kayıtlı olabilir
		// (ör. aynı Hürriyet gündem haberi Güncel'de). Kopya üretmek yerine
		// kategoriyi devralır: her haber tek kategoride kalır, sözde
		// kategori de gerçek içerik kazanır.
		if len(f.Keywords) > 0 {
			_, err := reassignCategory(ctx, pool, link, catID)
			return false, err
		}
		return false, nil
	}

	// görsel: önce RSS etiketi, yoksa haber sayfasından og:image
	imgURL := pickImage(it)
	spot := cleanText(it.itemText())
	if imgURL == "" || spot == "" {
		ogImg, ogDesc := scrapePage(link)
		if imgURL == "" {
			imgURL = ogImg
		}
		if spot == "" {
			spot = cleanText(ogDesc)
		}
	}
	if len([]rune(spot)) > 280 {
		spot = string([]rune(spot)[:280])
	}

	var coverID *int64
	if imgURL != "" {
		if data, ext, err := downloadImage(imgURL); err == nil {
			stored, err := platform.SaveMedia(data, "seed"+ext)
			if err == nil {
				w, hgt := imageDims(data, ext)
				size := int64(len(data))
				m := &repo.Media{
					SiteID: 1, UserID: authorID, Kind: "image",
					Filename: "seed" + ext, Path: stored,
					Mime: mimeOf(ext), SizeBytes: &size,
					Alt: &it.Title, Source: "internet",
				}
				if w > 0 {
					m.Width, m.Height = &w, &hgt
				}
				if id, err := repo.CreateMedia(ctx, pool, m); err == nil {
					coverID = &id
				}
			}
		}
	}

	// gövde: RSS özeti (veya og:description) tek paragraflık blok
	body := spot
	if body == "" {
		body = it.Title
	}
	blocks := []map[string]any{{"type": "p", "text": body}}
	detail, _ := json.Marshal(blocks)

	now := time.Now()
	pub := now.Add(-time.Duration(len(it.Title)%240) * time.Minute)
	if t := parseDate(it.PubDate); !t.IsZero() {
		pub = t
	}
	if pub.After(now) {
		pub = now
	}

	slug := platform.Slugify(it.Title)
	if slug == "" {
		slug = "haber"
	}
	slug = slug + "-" + hash[:6]

	seodesc := spot
	if len([]rune(seodesc)) > 180 {
		seodesc = string([]rune(seodesc)[:180])
	}
	c := &repo.Content{
		TypeKey:       "haber",
		Status:        "yayinda",
		Title:         it.Title,
		Spot:          spot,
		Detail:        detail,
		Slug:          &slug,
		CategoryID:    &catID,
		AuthorID:      authorID,
		CoverMediaID:  coverID,
		SourceURL:     &link,
		SourceName:    &f.Source,
		CommentStatus: "acik",
		SeoTitle:      &it.Title,
		SeoDesc:       &seodesc,
		CanonicalURL:  &link,
		PublishAt:     &pub,
		PublishedAt:   &pub,
		Locale:        "tr",
		SiteID:        1,
		AiScore:       60,
	}
	newID, err := repo.CreateContent(ctx, pool, c)
	if err != nil {
		return false, err
	}
	if withTags {
		// Başlıktan anahtar kelimeler: alaka motorunun "etiket örtüşmesi"
		// bileşeni bu etiketlerle beslenir.
		_, _ = repo.AutoTagFromTitle(ctx, pool, newID, it.Title, 5)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO crawled_items (feed_id, url, url_hash) VALUES (NULL,$1,$2)
		ON CONFLICT DO NOTHING`, link, hash); err != nil {
		log.Printf("   ! crawled_items: %v", err)
	}
	return true, nil
}

// pseudoCats: içeriği başka bir kategoriden devralabilen, anahtar kelime
// süzgeciyle dolan kategoriler. Bir sözde kategori başka bir sözde
// kategoriden haber çalmaz (yoksa her çalıştırmada yer değiştirirler).
var pseudoCats = map[string]bool{
	"asayis": true, "ilceler": true, "sanat": true, "turizm": true,
}

// reassignCategory haber zaten kayıtlıysa (aynı kaynak bağlantı) kategorisini
// istenen kategoriye taşır. Kart kategorisi değişen haber, kategori
// manşeti (is_dm) olma niteliğini kaybeder; sonraki assignFlags düzeltir.
func reassignCategory(ctx context.Context, pool *pgxpool.Pool, link string, catID int64) (bool, error) {
	var id, cur int64
	if err := pool.QueryRow(ctx, `
		SELECT id, COALESCE(category_id, 0) FROM contents
		 WHERE site_id=1 AND source_url=$1 ORDER BY id LIMIT 1`, link).Scan(&id, &cur); err != nil {
		return false, nil
	}
	if cur == catID {
		return false, nil
	}
	var curSlug string
	_ = pool.QueryRow(ctx, `SELECT COALESCE(slug,'') FROM categories WHERE id=$1`, cur).Scan(&curSlug)
	if pseudoCats[curSlug] {
		return false, nil
	}
	if _, err := pool.Exec(ctx,
		`UPDATE contents SET category_id=$2, is_dm=false WHERE id=$1`, id, catID); err != nil {
		return false, err
	}
	return true, nil
}

// ---- manşet / son dakika bayrakları ----

func assignFlags(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`UPDATE contents SET is_am=false, is_sd=false, is_dm=false WHERE site_id=1 AND status='yayinda'`,
		// slider + üst manşet: en yeni 8
		`UPDATE contents SET is_am=true WHERE id IN (
			SELECT id FROM contents WHERE site_id=1 AND status='yayinda'
			ORDER BY published_at DESC NULLS LAST LIMIT 8)`,
		// son dakika: manşetten sonraki en yeni 12
		`UPDATE contents SET is_sd=true WHERE id IN (
			SELECT id FROM contents WHERE site_id=1 AND status='yayinda' AND is_am=false
			ORDER BY published_at DESC NULLS LAST LIMIT 12)`,
		// kategori manşeti: her kategoriden en yeni 1 (manşet/son dakika dışı)
		`UPDATE contents SET is_dm=true WHERE id IN (
			SELECT DISTINCT ON (category_id) id FROM contents
			WHERE site_id=1 AND status='yayinda' AND is_am=false AND is_sd=false
			ORDER BY category_id, published_at DESC NULLS LAST)`,
	}
	for _, q := range stmts {
		if _, err := pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// ---- RSS / görsel yardımcıları ----

func fetchFeed(rawURL string) ([]rssItem, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml;q=0.9, */*;q=0.8")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var f rssFeed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("xml: %w", err)
	}
	return f.items(), nil
}

func pickImage(it rssItem) string {
	for _, m := range it.MediaContent {
		if m.URL != "" && isImageURL(m.URL) {
			return m.URL
		}
	}
	for _, m := range it.MediaThumb {
		if m.URL != "" && isImageURL(m.URL) {
			return m.URL
		}
	}
	if it.Enclosure.URL != "" && isImageURL(it.Enclosure.URL) {
		return it.Enclosure.URL
	}
	// AA örneği: <image>URL</image>
	if s := strings.TrimSpace(it.Image); s != "" && isImageURL(s) {
		return s
	}
	return ""
}

func isImageURL(u string) bool {
	l := strings.ToLower(u)
	return strings.HasPrefix(l, "http") &&
		(strings.Contains(l, ".jpg") || strings.Contains(l, ".jpeg") ||
			strings.Contains(l, ".png") || strings.Contains(l, ".webp") ||
			strings.Contains(l, ".avif") || strings.Contains(l, ".gif"))
}

func scrapePage(pageURL string) (img, desc string) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", ""
	}
	s := string(body)
	if m := reOgImage.FindStringSubmatch(s); m != nil {
		img = m[1]
	} else if m := reOgImageAlt.FindStringSubmatch(s); m != nil {
		img = m[1]
	}
	if m := reOgDesc.FindStringSubmatch(s); m != nil {
		desc = m[1]
	}
	if img != "" && strings.HasPrefix(img, "//") {
		img = "https:" + img
	}
	return img, desc
}

func downloadImage(u string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Referer", req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "image/") {
		return nil, "", fmt.Errorf("beklenmeyen içerik türü %s", ct)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return nil, "", err
	}
	if len(data) < 2048 {
		return nil, "", fmt.Errorf("dosya çok küçük (%d bayt)", len(data))
	}
	return data, extOf(u, data), nil
}

func extOf(u string, data []byte) string {
	l := strings.ToLower(u)
	for _, e := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif"} {
		if strings.Contains(l, e) {
			if e == ".jpeg" {
				return ".jpg"
			}
			return e
		}
	}
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8}):
		return ".jpg"
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}):
		return ".png"
	case bytes.HasPrefix(data, []byte("GIF8")):
		return ".gif"
	case len(data) > 12 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return ".webp"
	}
	return ".jpg"
}

func mimeOf(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".avif":
		return "image/avif"
	default:
		return "image/jpeg"
	}
}

func imageDims(data []byte, ext string) (int, int) {
	if ext == ".webp" {
		return webpDims(data)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

// webpDims VP8X/VP8L/VP8 başlıklarından boyutu okur (stdlib webp desteklemez).
func webpDims(b []byte) (int, int) {
	if len(b) < 30 || !bytes.Equal(b[0:4], []byte("RIFF")) {
		return 0, 0
	}
	switch string(b[12:16]) {
	case "VP8X":
		w := 1 + int(b[24]) + int(b[25])<<8 + int(b[26])<<16
		h := 1 + int(b[27]) + int(b[28])<<8 + int(b[29])<<16
		return w, h
	case "VP8L":
		v := uint32(b[21]) | uint32(b[22])<<8 | uint32(b[23])<<16 | uint32(b[24])<<24
		return int(v&0x3FFF) + 1, int((v>>14)&0x3FFF) + 1
	case "VP8 ":
		w := int(b[26]) | int(b[27])<<8 | int(b[28])<<16
		h := int(b[29]) | int(b[30])<<8 | int(b[31])<<16
		return w & 0x3FFF, h & 0x3FFF
	}
	return 0, 0
}

// ---- metin yardımcıları ----

func cleanTitle(s string) string {
	s = cleanText(s)
	s = reTitleSuffix.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func cleanText(s string) string {
	s = html.UnescapeString(s)
	if reHTMLTags.MatchString(s) {
		s = reHTMLTags.ReplaceAllString(s, " ")
	}
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, l := range []string{time.RFC1123Z, time.RFC1123, time.RFC3339, time.RFC822Z,
		"Mon, 02 Jan 2006 15:04:05 -0700", "2006-01-02T15:04:05Z07:00", "Mon, 02 Jan 2006 15:04:05 MST"} {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// seedGalleries görselli yayındaki haberlerden foto galeri ve video içeriği
// türetir; "Foto Galeri" / "Video Galeri" blokları böylece gerçek veriyle
// dolar. Demo amaçlıdır: içerik aynı kaynağın görseli olan haberlerden
// kopyalanır, kaynak bağlantısı korunur.
func seedGalleries(ctx context.Context, pool *pgxpool.Pool, authorID int64, total int) (int, error) {
	var existing int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM contents WHERE site_id=1
		  AND type_key IN ('foto_galeri','video')`).Scan(&existing); err != nil {
		return 0, err
	}
	if existing > 0 {
		return 0, nil // zaten var, tekrar üretme
	}

	src, err := repo.ListVisualNews(ctx, pool, 1, total)
	if err != nil {
		return 0, err
	}

	n := 0
	for i, c := range src {
		// tek sayılar foto galeri, çift sayılar video
		kind, label := "foto_galeri", "Foto Galeri"
		if i%2 == 1 {
			kind, label = "video", "Video Galeri"
		}
		title := c.Title
		spot := c.Spot
		detail := c.Detail

		h := sha1.Sum([]byte(fmt.Sprintf("galeri:%d:%s", c.ID, kind)))
		slug := platform.Slugify(title) + "-galeri-" + hex.EncodeToString(h[:])[:6]

		pub := time.Now().Add(-time.Duration(i) * time.Minute)
		if c.PublishedAt != nil {
			pub = c.PublishedAt.Add(-time.Duration(i) * time.Minute)
		}
		g := &repo.Content{
			TypeKey:       kind,
			Status:        "yayinda",
			Title:         title,
			Spot:          spot,
			Detail:        detail,
			Slug:          &slug,
			CategoryID:    c.CategoryID,
			AuthorID:      authorID,
			CoverMediaID:  c.CoverMediaID,
			SourceURL:     c.SourceURL,
			SourceName:    c.SourceName,
			CommentStatus: "kapali",
			PublishAt:     &pub,
			PublishedAt:   &pub,
			Locale:        "tr",
			SiteID:        1,
			AiScore:       60,
		}
		_ = label
		if _, err := repo.CreateContent(ctx, pool, g); err != nil {
			continue
		}
		n++
	}
	return n, nil
}

// backfillTags etiketi olmayan yayındaki içerikleri başlıktan etiketler.
func backfillTags(ctx context.Context, pool *pgxpool.Pool, perItem int) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title FROM contents c
		WHERE c.site_id = 1 AND c.status = 'yayinda'
		  AND NOT EXISTS (SELECT 1 FROM content_tags ct WHERE ct.content_id = c.id)`)
	if err != nil {
		return 0, err
	}
	type item struct {
		id    int64
		title string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.title); err != nil {
			continue
		}
		items = append(items, it)
	}
	rows.Close()
	n := 0
	for _, it := range items {
		if _, err := repo.AutoTagFromTitle(ctx, pool, it.id, it.title, perItem); err == nil {
			n++
		}
	}
	return n, nil
}

// ---- rapor ----

func report(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		SELECT c.slug, c.name, count(k.id), count(k.cover_media_id)
		FROM categories c
		LEFT JOIN contents k ON k.category_id = c.id AND k.status='yayinda'
		WHERE c.site_id=1
		GROUP BY c.slug, c.name, c.sort_order
		ORDER BY c.sort_order`)
	if err != nil {
		log.Printf("rapor: %v", err)
		return
	}
	defer rows.Close()
	fmt.Println("\n--- kategori / içerik / görsel ---")
	for rows.Next() {
		var slug, name string
		var total, withImg int
		if err := rows.Scan(&slug, &name, &total, &withImg); err != nil {
			log.Printf("   ! özet satırı okunamadı: %v", err)
			continue
		}
		fmt.Printf("  %-14s %-14s %3d haber  %3d görselli\n", slug, name, total, withImg)
	}
	var tot, img int
	if err := pool.QueryRow(ctx, `SELECT count(*), count(cover_media_id) FROM contents WHERE site_id=1 AND status='yayinda'`).Scan(&tot, &img); err != nil {
		log.Printf("   ! toplam sayılamadı: %v", err)
		return
	}
	fmt.Printf("\nTOPLAM: %d yayında haber, %d görselli\n", tot, img)
}
