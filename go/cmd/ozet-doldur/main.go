// Komut: ozet-doldur — yayındaki içeriklerin AI özetini (contents.ai_summary)
// üretir.
//
// NEDEN AYRI KOMUT: makale sayfasındaki "Özet" kutusu (tasarım spec §10.2.2)
// yalnız ai_summary DOLUYSA basılıyor. Ölçüm (6 Kasım 2026): yayındaki 126
// içeriğin 126'sında alan boş — kutu hiç görünmüyordu. Otomatik yazım hattı
// (Faz 194) yalnız TASLAKLARA özet ekliyor; yayındaki metni yeniden yazmak
// editör kararıdır, bu yüzden burada yalnız ÖZET alanı doldurulur: başlık,
// spot, gövde ve durum DEĞİŞMEZ.
//
// Kullanım:
//
//	ozet-doldur                 # özeti boş yayındaki içerikler, en çok 200
//	ozet-doldur -maks 50        # daha küçük parti
//	ozet-doldur -kuru           # yalnız say, AI çağrısı yapma
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
)

// eszamanli: okuyucu sağlayıcı 429 verdiği için 2'nin üstüne çıkmıyoruz.
const eszamanli = 2

type aday struct {
	ID    int64
	Title string
	Spot  string
	Govde string
}

func main() {
	maks := flag.Int("maks", 200, "tek koşuda işlenecek en fazla içerik")
	kuru := flag.Bool("kuru", false, "yalnız bekleyen sayısını yazdır")
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

	bekleyen, err := bekleyenSayisi(ctx, pool, *siteID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sorgu:", err)
		os.Exit(1)
	}
	fmt.Printf("özeti boş yayındaki içerik: %d\n", bekleyen)
	if *kuru || bekleyen == 0 {
		return
	}

	liste, err := adaylariOku(ctx, pool, *siteID, *maks)
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
		wg      sync.WaitGroup
		mu      sync.Mutex
		yazilan int
		hata    int
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
			ozet, err := ozetUretDene(ctx, aiclient, a)
			if err != nil {
				mu.Lock()
				hata++
				mu.Unlock()
				logger.Warn("özet üretilemedi", "içerik", a.ID, "error", err)
				return
			}
			if err := ozetYaz(ctx, pool, a.ID, ozet); err != nil {
				mu.Lock()
				hata++
				mu.Unlock()
				logger.Warn("özet yazılamadı", "içerik", a.ID, "error", err)
				return
			}
			mu.Lock()
			yazilan++
			mu.Unlock()
		}(a)
	}
	wg.Wait()

	fmt.Printf("bitti: %d özet yazıldı, %d hata, %s\n", yazilan, hata, time.Since(bas).Round(time.Second))
	sonra, _ := bekleyenSayisi(ctx, pool, *siteID)
	fmt.Printf("özeti boş kalan: %d\n", sonra)
}

func bekleyenSayisi(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM contents
		 WHERE site_id = $1 AND status = 'yayinda'
		   AND (ai_summary IS NULL OR btrim(ai_summary) = '')
		   AND title IS NOT NULL AND btrim(title) <> ''`, siteID).Scan(&n)
	return n, err
}

func adaylariOku(ctx context.Context, pool *pgxpool.Pool, siteID int64, maks int) ([]aday, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, COALESCE(title,''), COALESCE(spot,''), COALESCE(detail::text,'[]')
		  FROM contents
		 WHERE site_id = $1 AND status = 'yayinda'
		   AND (ai_summary IS NULL OR btrim(ai_summary) = '')
		   AND title IS NOT NULL AND btrim(title) <> ''
		 ORDER BY published_at DESC NULLS LAST, id DESC
		 LIMIT $2`, siteID, maks)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []aday
	for rows.Next() {
		var a aday
		var detay string
		if err := rows.Scan(&a.ID, &a.Title, &a.Spot, &detay); err != nil {
			return nil, err
		}
		a.Govde = detayMetni(detay)
		out = append(out, a)
	}
	return out, rows.Err()
}

// detayMetni JSONB blok dizisinden düz metin üretir.
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

func ozetUret(ctx context.Context, aiclient *ai.Client, a aday) (string, error) {
	govde := a.Govde
	if r := []rune(govde); len(r) > 4000 {
		govde = string(r[:4000])
	}
	sistem := `Sen Türkçe yayın yapan bir kripto haber sitesinin editörüsün.
Yalnızca JSON döndür: {"ai_summary":"..."}
Kurallar:
- Özet TÜRKÇE olacak; kaynak metin İngilizce olabilir.
- Tam 2 veya 3 cümle, 40-70 kelime. Makale sayfasındaki "Özet" kutusuna basılır.
- Yalnızca metindeki bilgiyi kullan; yeni isim, sayı, tarih, iddia EKLEME.
- İlk cümle ne olduğunu, son cümle neden önemli olduğunu söylesin.
- Kaynak site/ajans adını yazma. Tıklama tuzağı, "bu yazıda" kalıbı yok.
- Düz metin: markdown, HTML, madde işareti kullanma.`
	kullanici := fmt.Sprintf("Başlık: %s\n\nSpot: %s\n\nHaber metni:\n%s", a.Title, a.Spot, govde)
	resp, err := aiclient.CompleteStrict(ctx, ai.Request{
		Messages: []ai.Msg{
			{Role: ai.RoleSystem, Content: sistem},
			{Role: ai.RoleUser, Content: kullanici},
		},
		JSONMode: true,
		// 500 token YETMEDİ (ölçüm 6 Kasım 2026): 133/133 çağrı
		// `completion_tokens=500` ile kesildi ve "JSON nesnesi bulunamadı"
		// hatası verdi. Yanıt kesildiğinde özet yazılmıyor; bütçe bol tutulur.
		MaxTokens:   1600,
		Temperature: 0.3,
	})
	if err != nil {
		return "", err
	}
	return ozetJSON(resp.Text)
}

func ozetJSON(ham string) (string, error) {
	s := strings.TrimSpace(ham)
	if strings.HasPrefix(s, "```") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
	}
	bas, son := strings.IndexByte(s, '{'), strings.LastIndexByte(s, '}')
	if bas < 0 || son <= bas {
		return "", errors.New("JSON nesnesi bulunamadı")
	}
	var m struct {
		Ozet string `json:"ai_summary"`
	}
	if err := json.Unmarshal([]byte(s[bas:son+1]), &m); err != nil {
		return "", err
	}
	ozet := strings.Join(strings.Fields(m.Ozet), " ")
	if len([]rune(ozet)) < 60 {
		return "", errors.New("özet çok kısa")
	}
	return ozet, nil
}

// ozetUretDene özet üretimini bütçe artırarak bir kez yineler.
//
// Neden: sağlayıcı aynı isteme bazen kesik yanıt döndürüyor (ölçüm: 133
// çağrının tamamı 500 token bütçesinde kesildi). İkinci deneme daha geniş
// bütçeyle yapılır; ikisi de olmazsa içerik atlanır ve bir sonraki koşuda
// yeniden denenir (alan boş kalır, uydurma özet yazılmaz).
func ozetUretDene(ctx context.Context, aiclient *ai.Client, a aday) (string, error) {
	ozet, err := ozetUret(ctx, aiclient, a)
	if err == nil {
		return ozet, nil
	}
	return ozetUret(ctx, aiclient, a)
}

// ozetYaz yalnız BOŞ alanı doldurur: bu arada editör özet yazdıysa ezilmez.
func ozetYaz(ctx context.Context, pool *pgxpool.Pool, id int64, ozet string) error {
	komut, err := pool.Exec(ctx, `
		UPDATE contents SET ai_summary = $2, updated_at = now()
		 WHERE id = $1 AND (ai_summary IS NULL OR btrim(ai_summary) = '')`, id, ozet)
	if err != nil {
		return err
	}
	if komut.RowsAffected() == 0 {
		return errors.New("alan bu arada dolmuş")
	}
	return nil
}
