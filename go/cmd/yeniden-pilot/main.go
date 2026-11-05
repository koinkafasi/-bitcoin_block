// Komut: yeniden-pilot — toplu AI yeniden yazım pilotunu çalıştırır ve ölçer.
//
// Neden ayrı komut: pilot ölçümü üretim sunucusunun çalışmasını
// gerektirmemeli. Sunucudaki işçi döngüleri (çeviri, otomatik yazım) bu
// komutta BAŞLAMAZ; yalnız yeniden yazım önerisi üretilir. Öneri yayına
// yazılmaz: `yeniden_yazim` tablosunda onay bekler.
//
// Kullanım:
//
//	yeniden-pilot -pilot 10          # 10 haber kökü → TR+EN öneri aç, işle
//	yeniden-pilot -pilot 0 -maks 20  # yalnız kuyruktakileri işle
//	yeniden-pilot -rapor             # sonuçları yazdır, AI çağrısı yapma
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/aieditor"
	"theblock/site/internal/repo"
)

func main() {
	pilot := flag.Int("pilot", 0, "açılacak haber kökü sayısı (TR+EN birlikte)")
	maks := flag.Int("maks", 40, "tek koşuda işlenecek en fazla öneri")
	rapor := flag.Bool("rapor", false, "yalnız rapor yazdır (AI çağrısı yok)")
	siteID := flag.Int64("site", 1, "site kimliği")
	mod := flag.String("mod", "arastirmali", "arastirmali | yeniden")
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

	if !*rapor && *pilot > 0 {
		n, err := repo.YenidenYazimPilot(ctx, pool, *siteID, *pilot, *mod)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pilot açılamadı:", err)
			os.Exit(1)
		}
		fmt.Printf("pilot: %d öneri açıldı (%d haber kökü)\n", n, *pilot)
	}

	if !*rapor {
		runPilot(ctx, pool, logger, *maks)
	}
	raporYaz(ctx, pool, *siteID)
}

// runPilot kuyruk boşalana kadar tur atar.
func runPilot(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, maks int) {
	aiclient := ai.NewClient(logger)
	if !aiclient.Enabled() {
		fmt.Fprintln(os.Stderr, "gerçek AI sağlayıcısı yok: OMNIROUTE_API_KEY tanımlayın")
		os.Exit(1)
	}
	bas := time.Now()
	islenen := 0
	for {
		if ctx.Err() != nil || islenen >= maks {
			return
		}
		once := time.Now()
		n := aieditor.YenidenBirTur(ctx, pool, aiclient, logger)
		if n == 0 {
			fmt.Printf("kuyruk boş (toplam %d öneri, %s)\n", islenen, time.Since(bas).Round(time.Second))
			return
		}
		islenen += n
		fmt.Printf("tur: %d öneri işlendi (%s)\n", n, time.Since(once).Round(time.Second))
	}
}

// raporYaz ölçüm tablosunu basar: kalite, süre, token.
func raporYaz(ctx context.Context, pool *pgxpool.Pool, siteID int64) {
	satirlar, err := repo.YenidenYazimListele(ctx, pool, siteID, "", "", 500)
	if err != nil {
		fmt.Fprintln(os.Stderr, "liste:", err)
		return
	}
	type dilToplam struct {
		adet, eskiKar, yeniKar, eskiPar, yeniPar, uyari int
	}
	toplam := map[string]*dilToplam{}
	durum := map[string]int{}
	fmt.Println()
	fmt.Printf("%-6s %-3s %-9s %6s %6s %5s %5s %5s %6s\n",
		"id", "dil", "durum", "eskiK", "yeniK", "eP", "yP", "uyar", "token")
	for _, y := range satirlar {
		durum[y.Status]++
		eskiP := repo.GovdeParagraflari(y.OldDetail)
		yeniP := repo.GovdeParagraflari(y.NewDetail)
		eskiK, yeniK := karakter(eskiP), karakter(yeniP)
		fmt.Printf("%-6d %-3s %-9s %6d %6d %5d %5d %5d %6d\n",
			y.ID, y.Locale, y.Status, eskiK, yeniK, len(eskiP), len(yeniP),
			len(y.Warnings), y.PromptTokens+y.CompletionTokens)
		if y.Status != repo.YenidenHazir && y.Status != repo.YenidenUygulandi {
			if y.Error != "" {
				fmt.Printf("       └─ %s\n", y.Error)
			}
			continue
		}
		t := toplam[y.Locale]
		if t == nil {
			t = &dilToplam{}
			toplam[y.Locale] = t
		}
		t.adet++
		t.eskiKar += eskiK
		t.yeniKar += yeniK
		t.eskiPar += len(eskiP)
		t.yeniPar += len(yeniP)
		t.uyari += len(y.Warnings)
	}

	fmt.Println()
	fmt.Println("== durum ==")
	diller := make([]string, 0, len(durum))
	for d := range durum {
		diller = append(diller, d)
	}
	sort.Strings(diller)
	for _, d := range diller {
		fmt.Printf("%-10s %d\n", d, durum[d])
	}

	fmt.Println()
	fmt.Println("== dil bazında ortalama ==")
	fmt.Printf("%-4s %5s %8s %8s %7s %7s %6s\n", "dil", "adet", "eskiK", "yeniK", "eP", "yP", "uyar")
	for _, dil := range []string{"tr", "en"} {
		t := toplam[dil]
		if t == nil || t.adet == 0 {
			fmt.Printf("%-4s %5d %8s %8s %7s %7s %6s\n", dil, 0, "-", "-", "-", "-", "-")
			continue
		}
		fmt.Printf("%-4s %5d %8.0f %8.0f %7.1f %7.1f %6.2f\n",
			dil, t.adet,
			float64(t.eskiKar)/float64(t.adet), float64(t.yeniKar)/float64(t.adet),
			float64(t.eskiPar)/float64(t.adet), float64(t.yeniPar)/float64(t.adet),
			float64(t.uyari)/float64(t.adet))
	}

	// Token ve süre: ai_usage_logs gerçek çağrı kayıtlarıdır.
	var cagri, girdi, cikti, gecikme int
	var maliyet float64
	err = pool.QueryRow(ctx, `
		SELECT count(*), COALESCE(sum(prompt_tokens),0), COALESCE(sum(completion_tokens),0),
		       COALESCE(sum(latency_ms),0), COALESCE(sum(cost_est),0)
		  FROM ai_usage_logs WHERE action = 'yeniden_yazim'`).
		Scan(&cagri, &girdi, &cikti, &gecikme, &maliyet)
	if err == nil && cagri > 0 {
		fmt.Println()
		fmt.Printf("== AI çağrıları ==\nçağrı %d · girdi %d token · çıktı %d token · tahmini maliyet $%.5f · ortalama gecikme %.1f sn\n",
			cagri, girdi, cikti, maliyet, float64(gecikme)/float64(cagri)/1000)
	}
}

func karakter(p []string) int {
	n := 0
	for _, s := range p {
		n += len([]rune(s))
	}
	return n
}
