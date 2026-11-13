// Komut: ghost-aktar — ghost.bitcosar.com (Coin Hit) dışa aktarmasından
// SEÇİLMİŞ yazıları bitcosar.com'a taslak olarak alır.
//
// NEDEN SEÇEREK: dışa aktarmada 14.116 yayın var ama ölçüm (13 Kasım
// 2026) gövdelerin medyanını 46 KELİME buldu — %99,4'ü 300 kelimenin
// altında, yani başlık+spot kırpıntısı. Bunları toplu almak sitenin
// %77'sini ince içeriğe çevirirdi ve Google'ın site GENELİ kalite
// değerlendirmesini aşağı çekerdi. 600 kelime ve üzeri, başlığında
// İngilizce kalıntısı olmayan, yayındaki yazı sayısı: 50.
//
// NEDEN YENİDEN YAZDIRILMIYOR: bu 50 yazı zaten uzun ve yapılı (824
// adet h2, yazı başına ~16 alt başlık). AI'ya yeniden yazdırmak onları
// İYİLEŞTİRMEZ, kısaltır. Üretim hattı KISA RSS parçalarını makaleye
// çevirmek için; burada gereken SEO alanları ve görsel taşıma.
//
// Kullanım:
//
//	ghost-aktar -dosya adaylar.json -kuru     # yalnız ölç, yazma
//	ghost-aktar -dosya adaylar.json           # taslak olarak aktar
//	ghost-aktar -dosya adaylar.json -maks 5   # küçük parti ile dene
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// aday çıkarma betiğinin ürettiği kayıt (ghost_aday_cikar.py).
type aday struct {
	GhostID      string   `json:"ghost_id"`
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Excerpt      string   `json:"excerpt"`
	HTML         string   `json:"html"`
	Plaintext    string   `json:"plaintext"`
	Kelime       int      `json:"kelime"`
	FeatureImage string   `json:"feature_image"`
	PublishedAt  string   `json:"published_at"`
	Tags         []string `json:"tags"`
	GhostURL     string   `json:"ghost_url"`
}

// etiketKategori Ghost etiketini site kategorisine eşler.
//
// Eşlenmeyen etiket içeriği DÜŞÜRMEZ, yalnız varsayılan kategoriye
// koyar: kategori bilgisi editörde düzeltilebilir, kaybolan yazı
// düzeltilemez.
var etiketKategori = map[string]string{
	"bitcoin":    "markets",
	"ethereum":   "markets",
	"piyasa":     "markets",
	"stablecoin": "business",
	"defi":       "tech",
	"regülasyon": "policy",
	"regulasyon": "policy",
	"analiz":     "news-analysis",
	"gündem":     "news-analysis",
	"gundem":     "news-analysis",
	"son dakika": "news-analysis",
	"eğitim":     "learn",
	"egitim":     "learn",
	"kriptopedi": "kriptopedi",
	"rehber":     "learn",
}

const varsayilanKategori = "news-analysis"

func main() {
	dosya := flag.String("dosya", "", "aday JSON dosyası (zorunlu)")
	kuru := flag.Bool("kuru", false, "yalnız ölç ve raporla; veritabanına YAZMA")
	maks := flag.Int("maks", 0, "en fazla bu kadar yazı (0 = hepsi)")
	siteID := flag.Int64("site", 1, "site kimliği")
	yazarID := flag.Int64("yazar", 0, "yazar kimliği (0 = ilk yönetici)")
	flag.Parse()

	if *dosya == "" {
		fmt.Fprintln(os.Stderr, "-dosya zorunlu")
		os.Exit(2)
	}

	// Yol operatörün kendi verdiği `-dosya` bayrağından geliyor; bu
	// komutu çalıştırabilen kişi zaten DATABASE_URL'e sahip, yani
	// dosya sistemine erişimi bu bayrakla genişlemiyor. Web isteğinden
	// gelen bir yol olsaydı durum başka olurdu.
	// #nosec G703 -- yol operatörün verdiği `-dosya` BAYRAĞINDAN geliyor,
	// web isteğinden değil. Bu komutu çalıştırabilen kişinin zaten
	// DATABASE_URL'i var; dosya erişimi bu bayrakla genişlemiyor.
	ham, err := os.ReadFile(filepath.Clean(*dosya))
	if err != nil {
		fmt.Fprintln(os.Stderr, "dosya okunamadı:", err)
		os.Exit(1)
	}
	var adaylar []aday
	if err := json.Unmarshal(ham, &adaylar); err != nil {
		fmt.Fprintln(os.Stderr, "JSON okunamadı:", err)
		os.Exit(1)
	}
	if *maks > 0 && len(adaylar) > *maks {
		adaylar = adaylar[:*maks]
	}
	fmt.Printf("aday: %d\n\n", len(adaylar))

	// 1) DÖNÜŞÜM ÖLÇÜMÜ — veritabanına dokunmadan.
	rapor := olcVeDonustur(adaylar)
	rapor.yazdir()

	if *kuru {
		fmt.Println("\n(kuru koşu — veritabanına yazılmadı)")
		return
	}
	if rapor.kayipli > 0 {
		fmt.Fprintf(os.Stderr,
			"\nDURDURULDU: %d yazıda metin kaybı var. Önce dönüşümü düzeltin.\n",
			rapor.kayipli)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL tanımlı değil")
		os.Exit(1)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "veritabanı:", err)
		os.Exit(1)
	}
	defer pool.Close()

	kategoriler, err := kategoriHaritasi(ctx, pool, *siteID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kategoriler:", err)
		os.Exit(1)
	}

	yazar := *yazarID
	if yazar == 0 {
		if yazar, err = ilkYonetici(ctx, pool); err != nil {
			fmt.Fprintln(os.Stderr, "yazar bulunamadı:", err)
			os.Exit(1)
		}
	}

	var yonlendirme []string
	basarili := 0
	for _, a := range adaylar {
		if ctx.Err() != nil {
			break
		}
		detail, err := platform.HTMLToTipTap(a.HTML)
		if err != nil {
			fmt.Printf("  ATLANDI %-50s dönüşüm: %v\n", kisalt(a.Title, 50), err)
			continue
		}
		katSlug := kategoriSec(a.Tags)
		katID, varsa := kategoriler[katSlug]
		if !varsa {
			katID = kategoriler[varsayilanKategori]
		}

		c := &repo.Content{
			TypeKey:  "makale",
			Status:   "taslak",
			Title:    a.Title,
			Spot:     ozet(a),
			Detail:   detail,
			SiteID:   *siteID,
			AuthorID: yazar,
			Locale:   "tr",
			// Kapak görseli ŞİMDİLİK kaynak adresinden gösteriliyor;
			// medya kütüphanesine alınması ayrı adım (gorsel-aktar).
			ImageURL: a.FeatureImage,
		}
		if katID > 0 {
			c.CategoryID = &katID
		}
		if s := slugSec(a.Slug); s != "" {
			c.Slug = &s
		}

		id, err := repo.CreateContent(ctx, pool, c)
		if err != nil {
			fmt.Printf("  HATA    %-50s %v\n", kisalt(a.Title, 50), err)
			continue
		}
		basarili++
		yonlendirme = append(yonlendirme, fmt.Sprintf("%s\t%d", a.GhostURL, id))
		fmt.Printf("  ✓ %6d  %-52s  [%s]\n", id, kisalt(a.Title, 52), katSlug)
	}

	fmt.Printf("\naktarılan: %d / %d\n", basarili, len(adaylar))

	if len(yonlendirme) > 0 {
		yol := filepath.Clean(*dosya) + ".yonlendirme.tsv"
		// #nosec G703 -- çıktı, operatörün verdiği girdi dosyasının YANINA
		// yazılıyor; hedefi seçen yine operatörün kendi bayrağı.
		if err := os.WriteFile(yol, []byte(strings.Join(yonlendirme, "\n")+"\n"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, "yönlendirme yazılamadı:", err)
		} else {
			fmt.Println("yönlendirme haritası:", yol)
		}
	}
}

// ---- dönüşüm ölçümü ----

type donusumRaporu struct {
	toplam   int
	paragraf int
	baslik   int
	liste    int
	gorsel   int
	alinti   int
	kayipli  int
	kayiplar []string
}

// olcVeDonustur her adayı çevirir ve METİN KAYBI olup olmadığını
// ölçer.
//
// NEDEN KAYIP ÖLÇÜMÜ: dönüşüm hatası sessizdir — bozuk bir ayrıştırma
// yazının yarısını düşürür ve bunu ancak okuyucu fark eder. Ghost'un
// kendi `plaintext` alanı elimizde olduğu için karşılaştırma bedava.
func olcVeDonustur(adaylar []aday) donusumRaporu {
	var r donusumRaporu
	for _, a := range adaylar {
		ham, err := platform.HTMLToTipTap(a.HTML)
		if err != nil {
			r.kayipli++
			r.kayiplar = append(r.kayiplar, a.Title+" (dönüşüm hatası)")
			continue
		}
		var d platform.TTNode
		if json.Unmarshal(ham, &d) != nil {
			r.kayipli++
			continue
		}
		r.toplam++
		say(&r, d.Content)

		// Kelime karşılaştırması: Ghost'un plaintext'i referans.
		cikan := len(strings.Fields(tumMetin(d.Content)))
		beklenen := len(strings.Fields(a.Plaintext))
		if beklenen > 0 && cikan < beklenen*85/100 {
			r.kayipli++
			r.kayiplar = append(r.kayiplar,
				fmt.Sprintf("%s (%d -> %d kelime)", kisalt(a.Title, 44), beklenen, cikan))
		}
	}
	return r
}

func say(r *donusumRaporu, ns []platform.TTNode) {
	for _, n := range ns {
		switch n.Type {
		case "paragraph":
			r.paragraf++
		case "heading":
			r.baslik++
		case "bulletList", "orderedList":
			r.liste++
		case "image":
			r.gorsel++
		case "blockquote":
			r.alinti++
		}
		say(r, n.Content)
	}
}

func tumMetin(ns []platform.TTNode) string {
	var sb strings.Builder
	for _, n := range ns {
		if n.Text != "" {
			sb.WriteString(n.Text)
			sb.WriteByte(' ')
		}
		sb.WriteString(tumMetin(n.Content))
	}
	return sb.String()
}

func (r donusumRaporu) yazdir() {
	fmt.Println("=== dönüşüm ölçümü ===")
	fmt.Printf("  çevrilen yazı : %d\n", r.toplam)
	fmt.Printf("  paragraf      : %d\n", r.paragraf)
	fmt.Printf("  ALT BAŞLIK    : %d\n", r.baslik)
	fmt.Printf("  liste         : %d\n", r.liste)
	fmt.Printf("  alıntı        : %d\n", r.alinti)
	fmt.Printf("  görsel        : %d\n", r.gorsel)
	if r.kayipli == 0 {
		fmt.Println("  metin kaybı   : yok")
		return
	}
	fmt.Printf("  METİN KAYBI   : %d yazı\n", r.kayipli)
	for i, k := range r.kayiplar {
		if i >= 10 {
			fmt.Printf("    ... ve %d tane daha\n", len(r.kayiplar)-10)
			break
		}
		fmt.Println("    -", k)
	}
}

// ---- yardımcılar ----

func kategoriSec(etiketler []string) string {
	// Sıra bağımsız olsun diye alfabetik: aynı etiket kümesi her
	// koşuda aynı kategoriyi versin.
	es := append([]string(nil), etiketler...)
	sort.Strings(es)
	for _, e := range es {
		if k, ok := etiketKategori[strings.ToLower(strings.TrimSpace(e))]; ok {
			return k
		}
	}
	return varsayilanKategori
}

func ozet(a aday) string {
	s := strings.TrimSpace(a.Excerpt)
	if s == "" {
		s = strings.TrimSpace(a.Plaintext)
	}
	return kisalt(strings.Join(strings.Fields(s), " "), 240)
}

// slugSec Ghost slug'ını korur: eski adres ile yeni adres arasındaki
// bağı insan gözüyle izlenebilir kılar.
func slugSec(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) > 180 {
		s = s[:180]
	}
	return s
}

func kisalt(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func kategoriHaritasi(ctx context.Context, pool *pgxpool.Pool, siteID int64) (map[string]int64, error) {
	rows, err := pool.Query(ctx,
		`SELECT slug, id FROM categories WHERE site_id=$1 OR $1=0`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var s string
		var id int64
		if err := rows.Scan(&s, &id); err != nil {
			return nil, err
		}
		out[s] = id
	}
	return out, rows.Err()
}

func ilkYonetici(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx,
		`SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&id)
	return id, err
}
