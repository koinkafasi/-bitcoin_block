// Komut: ceviri-doldur — yayındaki İNGİLİZCE içerikleri Türkçeye çevirir.
//
// NEDEN: kripto portalı Türkçe yayın yapıyor (locale='tr') ama bot hattı
// kaynağın İngilizce başlığını olduğu gibi yazıyor. Ölçüm (6 Kasım 2026):
// yayındaki 133 içeriğin **126**'sının başlığında tek bir Türkçe harf yok
// (Decrypt 36, CryptoBriefing 30, CoinDesk 25, The Block 20, CryptoSlate 10).
// Okuyucu Türkçe bir siteye girip İngilizce başlık görüyordu.
//
// AKIŞ: AI'dan başlık + spot + paragraflar + SEO + özet istenir (otomatik
// yazım hattının `news` istemiyle aynı kurallar). Kalite kapısı geçerse
// `repo.SetContentYazim` ile yazılır — o yol SÜRÜM KORUMALI: bu arada editör
// içeriği kaydettiyse AI sonucu onun değişikliğini EZMEZ. Durum, kategori,
// görsel, yazar ve kaynak DEĞİŞMEZ.
//
// Kullanım:
//
//	ceviri-doldur -kuru            # yalnız kaç içerik bekliyor, AI çağrısı yok
//	ceviri-doldur -maks 20         # 20 içerik
//	ceviri-doldur -maks 200        # hepsi (uzun sürer: içerik başına 10-40 sn)
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/repo"
)

const (
	// eszamanli: sağlayıcı 429 verdiği için 2'nin üstüne çıkılmıyor.
	eszamanli = 2
	// kapiMinParagraf / kapiMinKarakter: kalite kapısı. Tek paragraflık
	// çeviri yayındaki metni İYİLEŞTİRMEZ, yalnız dilini değiştirir; bu
	// yüzden en az iki paragraf isteniyor.
	kapiMinParagraf = 2
	kapiMinKarakter = 300
)

// turkceHarf Türkçeye özgü harfler. Başlıkta hiçbiri yoksa başlık Türkçe
// değildir (İngilizce kaynak). Sezgi kaba ama ölçümle doğrulandı: 6 Kasım
// 2026'da 126 İngilizce başlığın 126'sında bu harfler yoktu.
const turkceHarf = "çğıöşüÇĞİÖŞÜ"

type aday struct {
	ID      int64
	Title   string
	Spot    string
	Govde   string
	Surum   int64
	Kaynak  string
	BolumAd string
}

func main() {
	maks := flag.Int("maks", 40, "tek koşuda işlenecek en fazla içerik")
	kuru := flag.Bool("kuru", false, "yalnız bekleyen sayısını yazdır (AI çağrısı yok)")
	kaynak := flag.String("kaynak", "", "yalnız bu kaynaktan gelenler (ör. Decrypt)")
	siteID := flag.Int64("site", 1, "site kimliği")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
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

	// Tek eşzamanlı koşu: iki süreç aynı satırları seçerse aynı içerik için
	// iki AI çağrısı yapılır (sürüm koruması veriyi bozmaz ama maliyeti
	// ikiye katlar). Zamanlayıcı ile elle koşu çakışabildiği için
	// danışma kilidi alınır; alınamazsa sessizce çıkılır.
	var kilitVar bool
	if err := pool.QueryRow(ctx, `SELECT pg_try_advisory_lock(4242)`).Scan(&kilitVar); err != nil || !kilitVar {
		fmt.Println("başka bir ceviri-doldur koşusu var; çıkılıyor")
		return
	}
	defer pool.Exec(context.Background(), `SELECT pg_advisory_unlock(4242)`)

	bekleyen, err := bekleyenSayisi(ctx, pool, *siteID, *kaynak)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sorgu:", err)
		os.Exit(1)
	}
	fmt.Printf("Türkçeleştirilecek yayında içerik: %d\n", bekleyen)
	if *kuru || bekleyen == 0 {
		return
	}

	liste, err := adaylariOku(ctx, pool, *siteID, *kaynak, *maks)
	if err != nil {
		fmt.Fprintln(os.Stderr, "liste:", err)
		os.Exit(1)
	}

	aiclient := ai.NewClient(logger)
	if !aiclient.Enabled() {
		fmt.Fprintln(os.Stderr, "gerçek AI sağlayıcısı yok: OMNIROUTE_API_KEY tanımlayın")
		os.Exit(1)
	}

	bas := time.Now()
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		yazilan  int
		elenecek int
		hata     int
	)
	sinir := make(chan struct{}, eszamanli)
	for _, a := range liste {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sinir <- struct{}{}
		go func(a aday) {
			defer wg.Done()
			defer func() { <-sinir }()
			y, err := ceviriUret(ctx, aiclient, a)
			if err != nil {
				mu.Lock()
				hata++
				mu.Unlock()
				logger.Warn("çeviri üretilemedi", "içerik", a.ID, "error", err)
				return
			}
			if eksik, neden := kapi(y); eksik {
				mu.Lock()
				elenecek++
				mu.Unlock()
				logger.Warn("kalite kapısı", "içerik", a.ID, "neden", neden)
				return
			}
			if err := repo.SetContentYazim(ctx, pool, a.ID, a.Surum, y); err != nil {
				mu.Lock()
				if errors.Is(err, repo.ErrConflict) {
					elenecek++
				} else {
					hata++
				}
				mu.Unlock()
				logger.Warn("içeriğe yazılamadı", "içerik", a.ID, "error", err)
				return
			}
			mu.Lock()
			yazilan++
			mu.Unlock()
			logger.Info("Türkçeleştirildi", "içerik", a.ID, "başlık", y.Title)
		}(a)
	}
	wg.Wait()

	fmt.Printf("bitti: %d çevrildi, %d kapıda elendi, %d hata, %s\n",
		yazilan, elenecek, hata, time.Since(bas).Round(time.Second))
	sonra, _ := bekleyenSayisi(ctx, pool, *siteID, *kaynak)
	fmt.Printf("kalan: %d\n", sonra)
}

func bekleyenSayisi(ctx context.Context, pool *pgxpool.Pool, siteID int64, kaynak string) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM contents
		 WHERE site_id = $1 AND status = 'yayinda' AND locale = 'tr'
		   AND translation_of IS NULL
		   AND COALESCE(title,'') <> ''
		   AND title !~ '[çğıöşüÇĞİÖŞÜ]'
		   AND detail <> '[]'::jsonb
		   AND ($2 = '' OR source_name = $2)`, siteID, kaynak).Scan(&n)
	return n, err
}

func adaylariOku(ctx context.Context, pool *pgxpool.Pool, siteID int64, kaynak string, maks int) ([]aday, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, COALESCE(title,''), COALESCE(spot,''), detail::text, version,
		       COALESCE(source_name,''), COALESCE(source_url,'')
		  FROM contents
		 WHERE site_id = $1 AND status = 'yayinda' AND locale = 'tr'
		   AND translation_of IS NULL
		   AND COALESCE(title,'') <> ''
		   AND title !~ '[çğıöşüÇĞİÖŞÜ]'
		   AND detail <> '[]'::jsonb
		   AND ($2 = '' OR source_name = $2)
		 ORDER BY published_at DESC NULLS LAST, id DESC
		 LIMIT $3`, siteID, kaynak, maks)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []aday
	for rows.Next() {
		var a aday
		var detay, url string
		if err := rows.Scan(&a.ID, &a.Title, &a.Spot, &detay, &a.Surum, &a.Kaynak, &url); err != nil {
			return nil, err
		}
		a.Govde = detayMetni(detay)
		if strings.TrimSpace(a.Govde) == "" {
			a.Govde = a.Spot
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func detayMetni(ham string) string {
	var bloklar []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(ham), &bloklar); err != nil {
		return ""
	}
	var parcalar []string
	for _, b := range bloklar {
		if t := strings.TrimSpace(b.Text); t != "" {
			parcalar = append(parcalar, t)
		}
	}
	return strings.Join(parcalar, "\n\n")
}

func ceviriUret(ctx context.Context, aiclient *ai.Client, a aday) (repo.YazimSonucu, error) {
	govde := a.Govde
	if r := []rune(govde); len(r) > 7000 {
		govde = string(r[:7000])
	}
	sistem := `Sen Türkçe yayın yapan bir kripto haber sitesinin editörüsün.
Yalnızca JSON döndür.
Kurallar:
- Kaynak metin İNGİLİZCE olabilir; çıktı TÜRKÇE olacak.
- Kaynakta olmayan isim, sayı, tarih, alıntı, neden veya sonuç UYDURMA.
- paragraphs: 3 ile 8 paragraf, her paragraf 2-4 cümle. İlk paragraf 5N1K ile başlasın.
- Kaynak site/ajans adını ("...'a göre", "...com") YAZMA; haberi kendi metnimiz gibi yaz.
- HTML etiketi, emoji, markdown, "devamını oku" kalıntısı kullanma. Düz metin.
- spot: 110-170 karakter, tek cümle.
- ai_summary: 2 veya 3 cümle, 40-70 kelime.
- seo_title: 30-60 karakter. meta_description: 110-155 karakter.
- Başlık: kaynağın başlığını birebir çevirme; Türkçe haber başlığı yaz, tıklama tuzağı yok.
Şu JSON şemasını döndür:
{"title":"","spot":"","ai_summary":"","paragraphs":[""],"seo_title":"","meta_description":""}`
	kullanici := fmt.Sprintf("Başlık: %s\n\nSpot: %s\n\nHaber metni:\n%s", a.Title, a.Spot, govde)

	resp, err := aiclient.CompleteStrict(ctx, ai.Request{
		Messages: []ai.Msg{
			{Role: ai.RoleSystem, Content: sistem},
			{Role: ai.RoleUser, Content: kullanici},
		},
		JSONMode: true,
		// Muhakeme tabanlı modeller bütçenin bir kısmını düşünmeye harcıyor;
		// 1800 kesiliyordu (ölçüm: finish_reason=length + boş içerik).
		MaxTokens:   4000,
		Temperature: 0.3,
	})
	if err != nil {
		return repo.YazimSonucu{}, err
	}
	return ceviriJSON(resp.Text)
}

func ceviriJSON(ham string) (repo.YazimSonucu, error) {
	s := strings.TrimSpace(ham)
	if strings.HasPrefix(s, "```") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
	}
	bas, son := strings.IndexByte(s, '{'), strings.LastIndexByte(s, '}')
	if bas < 0 || son <= bas {
		return repo.YazimSonucu{}, errors.New("JSON nesnesi bulunamadı")
	}
	var m struct {
		Title    string   `json:"title"`
		Spot     string   `json:"spot"`
		Ozet     string   `json:"ai_summary"`
		Paragraf []string `json:"paragraphs"`
		SeoTitle string   `json:"seo_title"`
		SeoDesc  string   `json:"meta_description"`
		Body     string   `json:"body"`
	}
	if err := json.Unmarshal([]byte(s[bas:son+1]), &m); err != nil {
		return repo.YazimSonucu{}, err
	}
	paragraflar := m.Paragraf
	if len(paragraflar) == 0 && strings.TrimSpace(m.Body) != "" {
		for _, p := range strings.Split(m.Body, "\n") {
			if p = strings.TrimSpace(p); p != "" {
				paragraflar = append(paragraflar, p)
			}
		}
	}
	temiz := make([]string, 0, len(paragraflar))
	for _, p := range paragraflar {
		if p = strings.Join(strings.Fields(p), " "); p != "" {
			temiz = append(temiz, p)
		}
	}
	return repo.YazimSonucu{
		Title:    strings.Join(strings.Fields(m.Title), " "),
		Spot:     strings.Join(strings.Fields(m.Spot), " "),
		Ozet:     strings.Join(strings.Fields(m.Ozet), " "),
		Paragraf: temiz,
		SeoTitle: strings.Join(strings.Fields(m.SeoTitle), " "),
		SeoDesc:  strings.Join(strings.Fields(m.SeoDesc), " "),
	}, nil
}

// kapi kalite kapısı: yayındaki metni daha kötüye çevirmeyiz.
func kapi(y repo.YazimSonucu) (bool, string) {
	if y.Title == "" {
		return true, "başlık boş"
	}
	if !strings.ContainsAny(y.Title, turkceHarf) {
		return true, "başlık hâlâ Türkçe değil"
	}
	if len(y.Paragraf) < kapiMinParagraf {
		return true, fmt.Sprintf("%d paragraf (en az %d)", len(y.Paragraf), kapiMinParagraf)
	}
	karakter := 0
	for _, p := range y.Paragraf {
		karakter += len([]rune(p))
	}
	if karakter < kapiMinKarakter {
		return true, fmt.Sprintf("%d karakter (en az %d)", karakter, kapiMinKarakter)
	}
	return false, ""
}
