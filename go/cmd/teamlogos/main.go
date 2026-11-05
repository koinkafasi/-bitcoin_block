// Command teamlogos — takım logolarını kendi altyapımıza indirir.
//
// Kullanım:
//
//	DATABASE_URL=... go run ./cmd/teamlogos         # eksikleri indir
//	DATABASE_URL=... go run ./cmd/teamlogos -force  # hepsini yeniden indir
//	DATABASE_URL=... go run ./cmd/teamlogos -liste  # yalnız rapor
//
// Neden ayrı komut: indirme ağ işidir, istek yolunda yapılmaz. Site çalışırken
// scheduler zaten eksikleri indirir; bu komut ilk kurulumu (veya logo
// değişimini) tek seferde yapmak ve sonucu görmek için var.
//
// `-force` diskteki dosyaları siler ve yeniden indirir. Kaynak URL'ler
// önbellekteki puan durumu ve fikstür verisinden okunur; yani indirme için
// ESPN/TheSportsDB'ye bir kez daha gidilir (puan/fikstür verisi yenilenir).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

func main() {
	force := flag.Bool("force", false, "diskteki logoları sil, yeniden indir")
	liste := flag.Bool("liste", false, "yalnız diskteki logoları listele")
	flag.Parse()

	if *liste {
		rapor()
		return
	}
	if *force {
		if err := os.RemoveAll(platform.TeamLogoDir()); err != nil {
			fmt.Fprintf(os.Stderr, "logo dizini silinemedi: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("diskteki logolar silindi")
	}

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL boş (deploy/.env)")
		os.Exit(1)
	}
	// STORAGE_DIR tanımsızsa depolama kökü çalışma dizinine göre çözülür ve
	// servisin kullandığı dizinden farklı olur: logolar diske yazılır, önbellek
	// yerel URL'e çevrilir, ama site 404 döner (28 Ekim 2026'da tam bu oldu).
	// Sessiz yanlış dizin yerine açık hata.
	if os.Getenv("STORAGE_DIR") == "" {
		fmt.Fprintf(os.Stderr,
			"STORAGE_DIR tanımsız. Servisin kullandığı dizini verin, ör:\n"+
				"  STORAGE_DIR=/home/alper/theblock/storage go run ./cmd/teamlogos\n"+
				"(şu an varsayılan: %s)\n", platform.TeamLogoDir())
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bağlantı: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Puan durumu: RefreshStandings içinde mirrorStandingBadges çağrılıyor.
	if err := repo.RefreshStandings(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "puan durumu: %v\n", err)
	}
	// Fikstür: RefreshFixtures tüm liglerin haftalarını gezer ve rozetleri indirir.
	if err := repo.RefreshFixtures(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "fikstür: %v\n", err)
	}
	// Önbellekte dış URL taşımaya devam eden rozetleri yerelleştir (kaynağı
	// yeniden çağırmadan).
	if n, err := repo.LocalizeCachedBadges(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "önbellek rozetleri: %v\n", err)
	} else if n > 0 {
		fmt.Printf("önbellek: %d kayıt yerelleştirildi\n", n)
	}
	rapor()
	kalanRaporu(ctx, pool)
}

// rapor diskteki logo sayısını ve boyutunu yazar.
func rapor() {
	dir := platform.TeamLogoDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("logo dizini yok (%s)\n", dir)
		return
	}
	var n int
	var bytes int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		n++
		bytes += info.Size()
	}
	fmt.Printf("logo: %d dosya, %.1f MB, dizin: %s\n", n, float64(bytes)/(1<<20), dir)
}

// kalanRaporu önbellekte hâlâ dış URL taşıyan rozet sayısını yazar.
//
// Akamai soğuk edge'de 202 döndüğü için tek çalıştırmada hepsi inmez; komut
// kaldığı yerden devam eder (diskte olan atlanır). Bu satır ilerlemeyi görünür
// kılar: birkaç çalıştırmada sıfıra iner, arada scheduler de tamamlar.
func kalanRaporu(ctx context.Context, pool *pgxpool.Pool) {
	var dis int
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT jsonb_array_elements(data->'rows')->>'badge' AS b
			FROM services_cache WHERE kind='puan'
			UNION ALL
			SELECT jsonb_array_elements(data->'matches')->>'homeBadge'
			FROM services_cache WHERE kind='fikstur-r'
			UNION ALL
			SELECT jsonb_array_elements(data->'matches')->>'awayBadge'
			FROM services_cache WHERE kind='fikstur-r'
		) s
		WHERE coalesce(b,'') <> '' AND b NOT LIKE '/takim-logo/%'`).Scan(&dis)
	if err != nil {
		return
	}
	if dis == 0 {
		fmt.Println("kalan dis rozet: 0 - hepsi yerel")
		return
	}
	fmt.Printf("kalan dış rozet: %d (komutu tekrar çalıştırın; scheduler de tamamlar)\n", dis)
}
