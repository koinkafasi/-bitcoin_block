package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// RunTrends günlük kategori trendlerini hesaplar (spec 06 K-B / 01 §L3).
// Son günlerin yayınlanan içerik etiketlerinden kategori×etiket frekansı çıkarır
// ve `trends` tablosuna günlük satır yazar (upsert).
func RunTrends(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 6 * time.Hour
	}
	logger.Info("trend üretici başladı", "her", every.String())
	refreshTrends(ctx, pool, logger)

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("trend üretici durdu")
			return
		case <-ticker.C:
			refreshTrends(ctx, pool, logger)
		}
	}
}

func refreshTrends(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	sites, err := repo.ListSites(ctx, pool)
	if err != nil {
		logger.Warn("trend: site listesi alınamadı", "error", err)
		return
	}
	total := 0
	for _, st := range sites {
		rows, err := repo.CategoryTagCounts(ctx, pool, st.ID, "tr", 7, 12)
		if err != nil {
			logger.Warn("trend: etiket sayımı", "site", st.Domain, "error", err)
			continue
		}
		for _, r := range rows {
			t := repo.CatTrend{SiteID: st.ID, CategoryID: r.CategoryID, Tag: r.Tag,
				PostCount: r.PostCount, Lang: "tr", Source: "otomatik"}
			if err := repo.UpsertCatTrend(ctx, pool, t); err != nil {
				logger.Warn("trend: yazılamadı", "tag", r.Tag, "error", err)
				continue
			}
			total++
		}
	}
	if total > 0 {
		logger.Info("trend güncellendi", "satir", total)
	}
}

// ---- Günlük podcast (spec 06 K-E) ----

// RunPodcast her gün dünkü özeti sese çevirir (espeak-ng + ffmpeg → MP3).
func RunPodcast(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	logger.Info("podcast üretici başladı")
	build := func(day time.Time) {
		sites, err := repo.ListSites(ctx, pool)
		if err != nil {
			logger.Warn("podcast: site listesi alınamadı", "error", err)
			return
		}
		for _, st := range sites {
			dg, err := repo.DigestByDate(ctx, pool, st.ID, day, "tr", "gunluk")
			if err != nil || dg == nil {
				dg, err = repo.BuildDailyDigest(ctx, pool, st.ID, day, "tr")
				if err != nil {
					continue
				}
			}
			if dg.AudioPath != nil && *dg.AudioPath != "" {
				continue
			}
			items, _ := repo.RecentPublishedOn(ctx, pool, st.ID, "tr", day)
			if len(items) == 0 {
				continue
			}
			var sb strings.Builder
			sb.WriteString(dg.Title + ". " + dg.Summary + " ")
			for i, it := range items {
				sb.WriteString(fmt.Sprintf("%d. %s. ", i+1, it.Title))
			}
			name := fmt.Sprintf("ozet-%s-%s", day.Format("20060102"), "tr")
			rel, err := platform.SynthesizeSpeech(sb.String(), "tr",
				filepath.Join(platform.StorageDir(), "media", "audio"), name)
			if err != nil {
				logger.Warn("podcast sentezlenemedi", "site", st.Domain, "error", err)
				continue
			}
			stored, err := platform.SaveMediaFile(filepath.Join(platform.StorageDir(), "media", "audio", rel), rel)
			if err != nil {
				logger.Warn("podcast kaydedilemedi", "error", err)
				continue
			}
			dg.AudioPath = &stored
			if err := repo.SetDigestAudio(ctx, pool, *dg); err != nil {
				logger.Warn("podcast bağlanamadı", "error", err)
				continue
			}
			logger.Info("podcast üretildi", "site", st.Domain, "tarih", day.Format("2006-01-02"), "dosya", stored)
		}
	}
	// hemen bir kez dünkü günü doldur, sonra günde bir (03:20)
	build(time.Now().AddDate(0, 0, -1))
	last := time.Now()
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if now.Hour() >= 3 && now.Sub(last) > 20*time.Hour {
				build(time.Now().AddDate(0, 0, -1))
				last = now
			}
		}
	}
}
