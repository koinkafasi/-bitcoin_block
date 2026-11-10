// Command backfill — tek seferlik veri doldurma işleri.
//
// Kullanım:
//
//	DATABASE_URL=... go run ./cmd/backfill -is=skor [-dry]
//	DATABASE_URL=... go run ./cmd/backfill -is=kategori [-dry]
//	DATABASE_URL=... go run ./cmd/backfill -is=mukerrer [-dry]
//	DATABASE_URL=... go run ./cmd/backfill -is=oncelik [-dry]
//	DATABASE_URL=... go run ./cmd/backfill -is=gorsel [-dry]
//	DATABASE_URL=... go run ./cmd/backfill -is=gorsel-kaldir [-dry]
//
// Neden ayrı komut: bu işler bot hattının kodunu (scoring/classify) yeniden
// kullanır ama istek yolunda çalışmaz. Tabloyu sayfa sayfa gezer, her sayfayı
// tek işlemde yazar; `-dry` ile hiçbir şey yazmadan yalnız rapor üretir.
//
// `-is=mukerrer` yıkıcı DEĞİLDİR: mükerrer taslaklar silinmez, `arsiv`
// durumuna alınır (geri alınabilir; kullanıcı onayı bu davranışa verildi).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/classify"
	"theblock/site/internal/quality"
	"theblock/site/internal/repo"
	"theblock/site/internal/scoring"
)

func main() {
	is := flag.String("is", "", "iş: skor | kategori | mukerrer | oncelik")
	dry := flag.Bool("dry", false, "hiçbir şey yazma, yalnız raporla")
	batch := flag.Int("batch", 500, "sayfa boyutu")
	flag.Parse()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL boş (deploy/.env)")
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

	var runErr error
	switch *is {
	case "skor":
		runErr = backfillScores(ctx, pool, *dry, *batch)
	case "kategori":
		runErr = backfillCategories(ctx, pool, *dry, *batch)
	case "mukerrer":
		runErr = mergeDuplicates(ctx, pool, *dry, *batch)
	case "oncelik":
		runErr = backfillOncelik(ctx, pool, *dry, *batch)
	case "gorsel":
		runErr = backfillGorseller(ctx, pool, *dry, *batch)
	case "gorsel-kaldir":
		runErr = backfillGorselKaldir(ctx, pool, *dry, *batch)
	default:
		fmt.Fprintln(os.Stderr, "kullanım: -is=skor|kategori|mukerrer|oncelik|gorsel|gorsel-kaldir [-dry]")
		os.Exit(1)
	}
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "HATA: %v\n", runErr)
		os.Exit(1)
	}
}

// probe taslak satırının doldurma için gereken alanlarıdır.
type probe struct {
	ID        int64
	Title     string
	Spot      string
	SourceURL *string
	CoverID   *int64
	CatID     *int64
	SiteID    int64
}

// backfillScores ai_score=0 olan İÇERİĞİ scoring.Relevance ile doldurur.
//
// Önce yalnız taslaklara bakıyordu ve bu sessiz bir tuzaktı: yayındaki 492
// içeriğin 488'inde ai_score=0 kalmıştı. Kategorilerde eşik şu an 0 olduğu
// için kimse fark etmiyordu, ama panelden homepage_min_score ya da min_score
// açıldığı an o 488 haber ana sayfadan kaybolurdu
// (contents.go: homepage_min_score <= c.ai_score).
//
// Yalnız 0 olanlar doldurulur; editörün elle girdiği puan korunur.
//
// Etiket sayısı ayrı sorguyla toplanır (tek sorguda JOIN + GROUP BY büyük
// tabloda gereksiz maliyet); kapak ve kaynak alanları satırda zaten var.
func backfillScores(ctx context.Context, pool *pgxpool.Pool, dry bool, batch int) error {
	total, updated := 0, 0
	lastID := int64(0)
	dist := map[int]int{}
	for {
		rows, err := pool.Query(ctx, `
			SELECT c.id, c.title, COALESCE(c.spot,''), c.source_url, c.cover_media_id,
			       c.category_id, c.site_id,
			       (SELECT count(*) FROM content_tags t WHERE t.content_id = c.id)
			FROM contents c
			WHERE c.ai_score = 0 AND c.id > $1
			ORDER BY c.id
			LIMIT $2`, lastID, batch)
		if err != nil {
			return err
		}
		type row struct {
			probe
			tags int
		}
		var page []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.Title, &r.Spot, &r.SourceURL, &r.CoverID, &r.CatID, &r.SiteID, &r.tags); err != nil {
				rows.Close()
				return err
			}
			page = append(page, r)
		}
		rows.Close()
		if len(page) == 0 {
			break
		}
		for _, r := range page {
			lastID = r.ID
			score, _ := scoring.Relevance(scoring.Input{
				Title:     r.Title,
				Spot:      r.Spot,
				TagCount:  r.tags,
				HasCover:  r.CoverID != nil,
				HasSource: r.SourceURL != nil && *r.SourceURL != "",
			})
			total++
			dist[score/10*10]++
			if dry {
				continue
			}
			if _, err := pool.Exec(ctx, `UPDATE contents SET ai_score = $1 WHERE id = $2 AND ai_score = 0`, score, r.ID); err != nil {
				return err
			}
			updated++
		}
	}
	fmt.Printf("skor: %d içerik tarandı, %d güncellendi (dry=%v)\n", total, updated, dry)
	for b := 0; b <= 100; b += 10 {
		if dist[b] > 0 {
			ust := b + 9
			if b >= 100 {
				ust = 100 // puan 100'de kırpılıyor, bant tek değer
			}
			fmt.Printf("  %3d-%3d : %d\n", b, ust, dist[b])
		}
	}
	return nil
}

// backfillCategories kategori atamasını classifier ile yeniden hesaplar.
//
// Yalnız `Gündem`de duran taslaklara dokunur: elle atanmış özel kategoriler
// (GZT Spor gibi kaynak bazlı atamalar) korunur. Böylece bot hattındaki yeni
// sınıflandırma geçmişe de uygulanır ama editör kararları ezilmez.
func backfillCategories(ctx context.Context, pool *pgxpool.Pool, dry bool, batch int) error {
	ids, err := categoryIDs(ctx, pool)
	if err != nil {
		return err
	}
	gundem := ids["gundem"]
	feeds, err := feedCategorySlugs(ctx, pool)
	if err != nil {
		return err
	}
	total, moved := 0, 0
	dist := map[string]int{}
	samples := map[string][]string{}
	lastID := int64(0)
	for {
		rows, err := pool.Query(ctx, `
			SELECT c.id, c.title, COALESCE(c.spot,''), COALESCE(c.source_name,'')
			FROM contents c
			WHERE c.status = 'taslak' AND c.category_id = $1 AND c.id > $2
			ORDER BY c.id LIMIT $3`, gundem, lastID, batch)
		if err != nil {
			return err
		}
		type row struct {
			ID     int64
			Title  string
			Spot   string
			Source string
		}
		var page []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.Title, &r.Spot, &r.Source); err != nil {
				rows.Close()
				return err
			}
			page = append(page, r)
		}
		rows.Close()
		if len(page) == 0 {
			break
		}
		for _, r := range page {
			lastID = r.ID
			total++
			slug := classify.Slug(classify.Doc{Title: r.Title, Spot: r.Spot}, feeds[r.Source])
			dist[slug]++
			if samples[slug] == nil {
				samples[slug] = []string{}
			}
			if len(samples[slug]) < 3 && slug != "gundem" {
				samples[slug] = append(samples[slug], r.Title)
			}
			if slug == "gundem" || dry {
				continue
			}
			id, ok := ids[slug]
			if !ok {
				continue
			}
			if _, err := pool.Exec(ctx, `UPDATE contents SET category_id = $1 WHERE id = $2 AND category_id = $3`, id, r.ID, gundem); err != nil {
				return err
			}
			moved++
		}
	}
	fmt.Printf("kategori: %d taslak tarandı, %d taşındı (dry=%v)\n", total, moved, dry)
	for slug, n := range dist {
		fmt.Printf("  %-16s %d\n", slug, n)
	}
	fmt.Println("--- örnekler ---")
	for slug, titles := range samples {
		for _, t := range titles {
			fmt.Printf("  %-14s %s\n", slug, truncateTitle(t, 90))
		}
	}
	return nil
}

func truncateTitle(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// mergeDuplicates aynı başlıklı taslak gruplarını `arsiv` durumuna alır.
//
// Grup içinden tutulan kayıt: en yüksek ai_score, sonra en eski. Diğerleri
// arşivlenir — silinmez, hiçbir veri kaybolmaz; `UPDATE contents SET
// status='taslak' WHERE status='arsiv'` ile geri alınabilir.
//
// Eşleştirme `quality.CheckDuplicates` ile yapılır (bot hattıyla aynı karar),
// ama aday kümesi grup içidir: SQL yalnız aynı normalize başlığa sahip
// grupları getirir, kesin kararı Go verir.
func mergeDuplicates(ctx context.Context, pool *pgxpool.Pool, dry bool, batch int) error {
	groups, err := duplicateGroups(ctx, pool, batch)
	if err != nil {
		return err
	}
	archived, kept := 0, 0
	for _, g := range groups {
		if len(g.members) < 2 {
			continue
		}
		keep := g.members[0]
		kept++
		for _, m := range g.members[1:] {
			// Kesin karar: başlık+spot shingle benzerliği yeterli değilse
			// (ör. aynı başlık, tamamen farklı spot) dokunma.
			rep := quality.CheckDuplicates(
				strings.TrimSpace(m.Title+" "+m.Spot),
				[]quality.Candidate{{Ref: strconv.FormatInt(keep.ID, 10), Title: keep.Title, Text: strings.TrimSpace(keep.Title + " " + keep.Spot)}},
			)
			if rep.OK() {
				continue
			}
			if dry {
				archived++
				continue
			}
			tag, err := pool.Exec(ctx, `UPDATE contents SET status = 'arsiv' WHERE id = $1 AND status = 'taslak'`, m.ID)
			if err != nil {
				return err
			}
			if tag.RowsAffected() > 0 {
				archived++
			}
		}
	}
	fmt.Printf("mükerrer: %d grup, %d kayıt tutuldu, %d kayıt arşivlendi (dry=%v)\n", len(groups), kept, archived, dry)
	return nil
}

type dupMember struct {
	ID    int64
	Title string
	Spot  string
	Score int
}
type dupGroup struct{ members []dupMember }

// duplicateGroups aynı normalize başlıklı taslak gruplarını, her grupta
// "tutulacak kayıt" başta olacak şekilde döndürür.
func duplicateGroups(ctx context.Context, pool *pgxpool.Pool, batch int) ([]dupGroup, error) {
	rows, err := pool.Query(ctx, `
		SELECT lower(btrim(title)) AS t
		FROM contents
		WHERE status = 'taslak'
		GROUP BY 1
		HAVING count(*) > 1
		ORDER BY count(*) DESC
		LIMIT $1`, batch*10)
	if err != nil {
		return nil, err
	}
	var titles []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return nil, err
		}
		titles = append(titles, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []dupGroup
	for _, t := range titles {
		mrows, err := pool.Query(ctx, `
			SELECT id, title, COALESCE(spot,''), ai_score
			FROM contents
			WHERE status = 'taslak' AND lower(btrim(title)) = $1
			ORDER BY ai_score DESC, length(COALESCE(spot,'')) DESC, created_at ASC, id ASC`, t)
		if err != nil {
			return nil, err
		}
		g := dupGroup{}
		for mrows.Next() {
			var m dupMember
			if err := mrows.Scan(&m.ID, &m.Title, &m.Spot, &m.Score); err != nil {
				mrows.Close()
				return nil, err
			}
			g.members = append(g.members, m)
		}
		mrows.Close()
		if len(g.members) > 1 {
			out = append(out, g)
		}
	}
	return out, nil
}

// categoryIDs slug → id haritası.
func categoryIDs(ctx context.Context, pool *pgxpool.Pool) (map[string]int64, error) {
	rows, err := pool.Query(ctx, `SELECT slug, id FROM categories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var slug string
		var id int64
		if err := rows.Scan(&slug, &id); err != nil {
			return nil, err
		}
		out[slug] = id
	}
	return out, rows.Err()
}

// feedCategorySlugs feed adı → kategori slug haritası (feed'in kendi ataması).
func feedCategorySlugs(ctx context.Context, pool *pgxpool.Pool) (map[string]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT f.name, COALESCE(c.slug,'')
		FROM feeds f LEFT JOIN categories c ON c.id = f.category_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, slug string
		if err := rows.Scan(&name, &slug); err != nil {
			return nil, err
		}
		out[name] = slug
	}
	return out, rows.Err()
}

// backfillOncelik güven ve trend eksenlerini yazar, sonra ortaya çıkan
// öncelik dağılımını raporlar.
//
// Tazelik ve öncelik SÜTUNA YAZILMAZ; sorgu anında hesaplanır. Bu yüzden
// burada "öncelik güncellendi" diye bir sayı yoktur — yalnız girdi eksenleri
// yazılır ve sonuç dağılımı ölçülür.
func backfillOncelik(ctx context.Context, pool *pgxpool.Pool, dry bool, batch int) error {
	sites, err := repo.ListSites(ctx, pool)
	if err != nil {
		return err
	}
	for _, st := range sites {
		fmt.Printf("site %d (%s)\n", st.ID, st.Domain)

		if dry {
			fmt.Println("  güven: kuru koşum, yazılmadı")
		} else {
			n, err := repo.RefreshGuvenScores(ctx, pool, st.ID)
			if err != nil {
				return fmt.Errorf("güven: %w", err)
			}
			fmt.Printf("  güven: %d satır güncellendi\n", n)
		}

		r, err := repo.RefreshTrendScores(ctx, pool, st.ID, batch, dry)
		if err != nil {
			return fmt.Errorf("trend: %w", err)
		}
		fmt.Printf("  trend: %d taze terim · %d taslak tarandı · %d eşleşti (%%%.1f) · %d yazılacak · %s\n",
			r.Terim, r.Taranan, r.Eslesen,
			100*float64(r.Eslesen)/float64(max(r.Taranan, 1)), r.Yazilan, r.Sure.Round(time.Millisecond))
		for b := 0; b <= 100; b += 10 {
			if r.Dagilim[b] > 0 {
				ust := b + 9
				if b >= 100 {
					ust = 100 // puan 100'de kırpılıyor, bant tek değer
				}
				fmt.Printf("    trend %3d-%3d : %d\n", b, ust, r.Dagilim[b])
			}
		}

		if err := raporOncelik(ctx, pool, st.ID, dry); err != nil {
			return err
		}
	}
	return nil
}

// raporOncelik öncelik dağılımını SQL'den okur. Kuru koşumda trend sütunu
// henüz yazılmadığı için dağılım mevcut değerlere göre çıkar; bu da
// "yazmadan önce ne vardı" sorusunun cevabıdır.
func raporOncelik(ctx context.Context, pool *pgxpool.Pool, siteID int64, dry bool) error {
	q := fmt.Sprintf(`
		SELECT FLOOR((%s)/10)*10 AS bant, count(*)
		  FROM contents c
		 WHERE c.site_id = $1 AND c.status = 'taslak'
		 GROUP BY 1 ORDER BY 1`, scoring.OncelikSQL("c"))
	rows, err := pool.Query(ctx, q, siteID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type satir struct {
		bant int
		n    int64
	}
	var dagilim []satir
	var toplam int64
	for rows.Next() {
		var s satir
		if err := rows.Scan(&s.bant, &s.n); err != nil {
			return err
		}
		dagilim = append(dagilim, s)
		toplam += s.n
	}
	if err := rows.Err(); err != nil {
		return err
	}
	etiket := "öncelik dağılımı"
	if dry {
		etiket += " (YAZILMADAN ÖNCE)"
	}
	fmt.Printf("  %s — %d taslak\n", etiket, toplam)
	var enBuyuk int64
	for _, s := range dagilim {
		if s.n > enBuyuk {
			enBuyuk = s.n
		}
		fmt.Printf("    %3d-%3d : %6d  (%%%.1f)\n", s.bant, s.bant+9, s.n,
			100*float64(s.n)/float64(max(toplam, 1)))
	}
	if toplam > 0 {
		fmt.Printf("  en kalabalık bant: %%%.1f  (eski ai_score %%87,9 idi)\n",
			100*float64(enBuyuk)/float64(toplam))
	}
	return nil
}
