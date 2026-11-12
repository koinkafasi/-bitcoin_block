// Package scheduler periyodik işleri çalıştırır: planlanmış yayın, çöp temizliği,
// bayat oturum temizliği.
package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/auth"
	"theblock/site/internal/live"
	"theblock/site/internal/mailer"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
	"theblock/site/internal/searchsvc"
)

// Run verilen aralıkta tik koyarak periyodik işleri yürütür.
func Run(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 30 * time.Second
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	logger.Info("scheduler başladı", "her", every.String())

	for {
		select {
		case <-ctx.Done():
			logger.Info("scheduler durdu")
			return
		case <-ticker.C:
			if err := tick(ctx, pool, logger); err != nil {
				logger.Warn("scheduler işi hata", "error", err)
			}
		}
	}
}

// tick tüm periyodik görevleri sırayla çalıştırır.
func tick(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	// 1) Planlanmış yayın: publish_at gelmiş kayıtları yayına geçir.
	rows, err := pool.Query(ctx, `
		UPDATE contents SET status = 'yayinda',
		       published_at = COALESCE(published_at, now()),
		       version = version + 1,
		       updated_at = now()
		WHERE status = 'planlanmis' AND publish_at IS NOT NULL AND publish_at <= now()
		RETURNING id, site_id, version, COALESCE(slug, '')`)
	if err != nil {
		return err
	}
	var promoted []int64
	var events []live.Event
	for rows.Next() {
		var id, siteID int64
		var version int
		var slug string
		if err := rows.Scan(&id, &siteID, &version, &slug); err == nil {
			promoted = append(promoted, id)
			events = append(events, live.Event{
				SiteID: siteID, ContentID: id, Version: version, Status: "yayinda",
				Slug: slug, Kind: "content", At: time.Now().Unix(),
			})
		}
	}
	rows.Close()
	for _, ev := range events {
		live.Default.Publish(ev)
	}
	if len(promoted) > 0 {
		indexPublished(ctx, pool, logger, promoted)
		// Faz 38: zamanlanmış yayınlar IndexNow'a da bildirilir (auto_index
		// ayarı açıkken) — Google News/haber sitemap tazeliği için.
		submitIndexNowPromoted(ctx, pool, logger, promoted)
	}

	// 2) Çöp (cop) kayıtları 30 gün sonra kalıcı silinsin.
	if _, err := pool.Exec(ctx,
		`DELETE FROM contents WHERE status = 'cop' AND updated_at < now() - interval '30 days'`); err != nil {
		return err
	}

	// 3) Bayat oturumları temizle.
	_, _ = auth.CleanupExpired(ctx, pool)

	// 4) Üye giriş hız sınırı denemelerini temizle (1 günden eski).
	if err := repo.PurgeLoginAttempts(ctx, pool); err != nil {
		logger.Warn("giriş denemeleri temizlenemedi", "error", err)
	}

	// 5) Süresi geçmiş şifre sıfırlama/onay kayıtlarını temizle.
	if _, err := pool.Exec(ctx,
		`DELETE FROM password_resets WHERE expires_at < now() - interval '7 days'`); err != nil {
		logger.Warn("şifre sıfırlama kayıtları temizlenemedi", "error", err)
	}

	// 6) Abonelik yaşam döngüsü: bitişe 3 gün kala hatırlatma, süresi
	// dolanları ücretsize düşürme (günlük tek bildirim).
	if ids, err := repo.ExpiringSoon(ctx, pool, 3); err == nil {
		for _, id := range ids {
			if repo.HasNotificationToday(ctx, pool, id, "uyelik-hatirlatma") {
				continue
			}
			name := "aboneliğiniz"
			if t, ok := repo.UserTier(ctx, pool, id); ok {
				name = t.Name + " aboneliğiniz"
			}
			_ = repo.AddNotification(ctx, pool, id, "uyelik-hatirlatma",
				name+" 3 gün içinde sona eriyor",
				"Kesintisiz erişim için aboneliğinizi yenileyin.", "/abonelik")
			if u, err := repo.GetUserByID(ctx, pool, id); err == nil {
				_ = mailer.Send(ctx, pool, u.Email, "Aboneliğiniz yakında sona eriyor",
					"<p>Sayın "+u.Name+",</p><p>"+name+" 3 gün içinde sona eriyor. "+
						"Kesintisiz erişim için <a href=\"/abonelik\">aboneliğinizi yenileyin</a>.</p>")
			}
		}
	} else {
		logger.Warn("abonelik hatırlatma listesi alınamadı", "error", err)
	}
	if ids, err := repo.ExpireSubscriptions(ctx, pool); err == nil {
		for _, id := range ids {
			_ = repo.AddNotification(ctx, pool, id, "uyelik-bitti",
				"Aboneliğiniz sona erdi",
				"Özel içeriklere erişim kısıtlandı. Yenilemek için paketleri inceleyin.", "/abonelik")
			if u, err := repo.GetUserByID(ctx, pool, id); err == nil {
				_ = mailer.Send(ctx, pool, u.Email, "Aboneliğiniz sona erdi",
					"<p>Sayın "+u.Name+",</p><p>Aboneliğiniz sona erdi; özel içeriklere erişim kısıtlandı. "+
						"<a href=\"/abonelik\">Paketleri inceleyip</a> yeniden abone olabilirsiniz.</p>")
			}
		}
	} else {
		logger.Warn("süresi dolan abonelikler düşürülemedi", "error", err)
	}
	// 7) Ödeme mutabakatı: 24 saatten eski "pending" kayıtları kapat.
	if n, err := repo.ReconcilePending(ctx, pool, 24); err == nil && n > 0 {
		logger.Info("ödeme mutabakatı: bekleyen ödemeler kapatıldı", "adet", n)
	}

	if err := repo.PurgeServiceCache(ctx, pool, 7); err != nil {
		logger.Warn("servis önbelleği temizlenemedi", "error", err)
	}
	return nil
}

func indexPublished(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, ids []int64) {
	sc := searchsvc.NewClient(os.Getenv("SEARCH_SVC_URL"))
	if sc == nil {
		return
	}
	docs := make([]searchsvc.Doc, 0, len(ids))
	for _, id := range ids {
		c, err := repo.GetContent(ctx, pool, id)
		if err != nil || c.Status != "yayinda" {
			continue
		}
		tags, _ := repo.TagsOf(ctx, pool, id)
		docs = append(docs, searchsvc.Doc{
			ID:       strconv.FormatInt(id, 10),
			Title:    c.Title,
			Spot:     c.Spot,
			Body:     blocksPlainText(c.Detail),
			Tags:     tags,
			URL:      slugURL(c.DisplaySlug(), id),
			Category: catSlugOrEmpty(c.CategorySlug),
		})
	}
	if len(docs) == 0 {
		return
	}
	if _, err := sc.Index(ctx, docs); err != nil {
		logger.Warn("scheduler indeksleme hatası", "adet", len(docs), "error", err)
	}
}

// blocksPlainText gövde bloklarından aranabilir düz metin çıkarır.
func blocksPlainText(detail []byte) string {
	var bl []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(detail, &bl); err != nil {
		return strings.TrimSpace(string(detail))
	}
	var parts []string
	for _, b := range bl {
		if s := strings.TrimSpace(b.Text); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// slugURL site içi makale yolu üretir (slug--id; render aşamasında tam URL
// DB'e göre kurulur — burada yalnız tutarlı kimlik için).
func slugURL(slug string, id int64) string {
	s := strings.TrimSpace(slug)
	if s == "" {
		s = "icerik"
	}
	return "/" + s + "-" + strconv.FormatInt(id, 10)
}

func catSlugOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// seoSetting scheduler tarafından seo bölümü ayarını okur (JSON dizge).
func seoSetting(ctx context.Context, pool *pgxpool.Pool, key string) string {
	b, err := repo.GetSetting(ctx, pool, "seo", key)
	if err != nil {
		return ""
	}
	var v string
	if json.Unmarshal(b, &v) != nil {
		return strings.Trim(string(b), `"`)
	}
	return v
}

func submitIndexNowPromoted(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, ids []int64) {
	if seoSetting(ctx, pool, "auto_index") != "1" {
		return
	}
	key := seoSetting(ctx, pool, "indexnow_key")
	if key == "" {
		return
	}
	endpoint := seoSetting(ctx, pool, "indexnow_endpoint")
	host := strings.TrimRight(seoSetting(ctx, pool, "indexnow_domain"), "/")
	if host == "" {
		host = "theblock.com.tr"
	}
	keyLoc := "https://" + host + "/" + key + ".txt"

	urls := make([]string, 0, len(ids))
	var siteID int64 // indexing_log FK → sites(id); 0 değil, içeriğin sitesi (Faz 38)
	for _, id := range ids {
		c, err := repo.GetContent(ctx, pool, id)
		if err != nil || c.Status != "yayinda" {
			continue
		}
		if siteID == 0 && c.SiteID > 0 {
			siteID = c.SiteID
		}
		slug := c.DisplaySlug()
		if slug == "" {
			slug = "icerik"
		}
		urls = append(urls, "https://"+host+"/"+slug+"-"+strconv.FormatInt(id, 10))
	}
	if len(urls) == 0 {
		return
	}
	go func() {
		bg := context.Background()
		code, err := platform.SubmitIndexNow(bg, endpoint, host, key, keyLoc, urls)
		for _, u := range urls {
			if err != nil {
				_ = repo.LogIndexing(bg, pool, siteID, "indexnow_sched", u, "hata", err.Error())
			} else {
				_ = repo.LogIndexing(bg, pool, siteID, "indexnow_sched", u, "ok", "HTTP "+strconv.Itoa(code))
			}
		}
		if err != nil {
			logger.Warn("scheduler indexnow hata", "adet", len(urls), "error", err)
		}
	}()
}

var _ = repo.ErrNotFound
