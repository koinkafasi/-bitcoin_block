package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

func RunGSC(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 6 * time.Hour
	}
	logger.Info("GSC/GA4 tazeleyici başladı", "her", every.String())
	tryGSC(ctx, pool, logger)
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tryGSC(ctx, pool, logger)
		}
	}
}

func tryGSC(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	last := repo.GetTrendSetting(ctx, pool, "gsc_last_sync")
	if last != "" {
		if t, err := time.Parse(time.RFC3339, last); err == nil && time.Since(t) < 20*time.Hour {
			return
		}
	}
	sites, err := repo.ListSites(ctx, pool)
	if err != nil || len(sites) == 0 {
		return
	}
	n, err := SyncGSC(ctx, pool, logger, sites[0].ID)
	if err != nil {
		logger.Warn("GSC senkronu", "error", err)
	} else if n > 0 {
		logger.Info("GSC senkronu tamam", "satır", n)
	}
	active, err := SyncGA4(ctx, pool, logger)
	if err != nil {
		logger.Warn("GA4 senkronu", "error", err)
	} else if active > 0 {
		logger.Info("GA4 canlı özet", "aktif", active)
	}
}

// SyncGSC son 3 günün Search Console sorgularını çeker ve kaydeder.
func SyncGSC(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, siteID int64) (int, error) {
	property := strings.TrimSpace(repo.GetTrendSetting(ctx, pool, "gsc_property"))
	if property == "" {
		return 0, fmt.Errorf("Search Console özelliği tanımlı değil")
	}
	sa := settingString(ctx, pool, "seo", "gsc_service_account")
	if sa == "" {
		return 0, fmt.Errorf("GSC servis hesabı tanımlı değil (SEO > indeksleme)")
	}
	tok, err := platform.ServiceAccountToken(ctx, []byte(sa), "https://www.googleapis.com/auth/webmasters.readonly")
	if err != nil {
		return 0, fmt.Errorf("GSC token: %w", err)
	}
	base := "https://searchconsole.googleapis.com/webmasters/v3/sites/" +
		url.PathEscape(property) + "/searchAnalytics/query"
	total := 0
	for i := 1; i <= 3; i++ {
		day := time.Now().AddDate(0, 0, -i)
		ds := day.Format("2006-01-02")
		body := map[string]any{
			"startDate": ds, "endDate": ds,
			"dimensions": []string{"query", "page"},
			"rowLimit":   1000, "dataState": "final",
		}
		code, raw, err := platform.PostJSON(ctx, base, body, map[string]string{
			"Authorization": "Bearer " + tok,
		})
		if err != nil {
			return total, fmt.Errorf("GSC istek: %w", err)
		}
		if code != 200 {
			return total, fmt.Errorf("GSC HTTP %d: %s", code, truncateText(string(raw), 200))
		}
		var out struct {
			Rows []struct {
				Keys        []string `json:"keys"`
				Clicks      float64  `json:"clicks"`
				Impressions float64  `json:"impressions"`
				CTR         float64  `json:"ctr"`
				Position    float64  `json:"position"`
			} `json:"rows"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return total, fmt.Errorf("GSC yanıt: %w", err)
		}
		rows := make([]repo.GSCQueryRow, 0, len(out.Rows))
		for _, r := range out.Rows {
			q, page := "", ""
			if len(r.Keys) > 0 {
				q = r.Keys[0]
			}
			if len(r.Keys) > 1 {
				page = r.Keys[1]
			}
			rows = append(rows, repo.GSCQueryRow{
				Query: q, Page: page,
				Clicks: int(r.Clicks), Impressions: int(r.Impressions),
				CTR: r.CTR, Position: r.Position,
			})
		}
		if err := repo.SaveGSCQueries(ctx, pool, siteID, property, day, rows); err != nil {
			return total, fmt.Errorf("GSC kaydet: %w", err)
		}
		total += len(rows)
	}
	_ = repo.SetTrendSetting(ctx, pool, "gsc_last_sync", time.Now().Format(time.RFC3339))
	return total, nil
}

// GA4Summary panelde gösterilen canlı özettir.
type GA4Summary struct {
	Aktif    int       `json:"aktif"`
	Sayfalar []GA4Page `json:"sayfalar"`
	Guncel   string    `json:"guncel"`
}

// GA4Page canlı sayfa satırı.
type GA4Page struct {
	Sayfa string `json:"sayfa"`
	Aktif int    `json:"aktif"`
}

// SyncGA4 GA4 Realtime API'den aktif kullanıcı özeti çeker.
func SyncGA4(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) (int, error) {
	property := strings.TrimSpace(repo.GetTrendSetting(ctx, pool, "ga4_property"))
	if property == "" {
		return 0, fmt.Errorf("GA4 özellik kimliği tanımlı değil")
	}
	sa := settingString(ctx, pool, "seo", "gsc_service_account")
	if sa == "" {
		return 0, fmt.Errorf("GA4 servis hesabı tanımlı değil")
	}
	tok, err := platform.ServiceAccountToken(ctx, []byte(sa), "https://www.googleapis.com/auth/analytics.readonly")
	if err != nil {
		return 0, fmt.Errorf("GA4 token: %w", err)
	}
	api := "https://analyticsdata.googleapis.com/v1beta/properties/" + url.PathEscape(property) + ":runRealtimeReport"
	body := map[string]any{
		"metrics":    []map[string]string{{"name": "activeUsers"}},
		"dimensions": []map[string]string{{"name": "unifiedScreenName"}},
		"limit":      5,
	}
	code, raw, err := platform.PostJSON(ctx, api, body, map[string]string{"Authorization": "Bearer " + tok})
	if err != nil {
		return 0, fmt.Errorf("GA4 istek: %w", err)
	}
	if code != 200 {
		return 0, fmt.Errorf("GA4 HTTP %d: %s", code, truncateText(string(raw), 200))
	}
	var out struct {
		Rows []struct {
			DimensionValues []struct{ Value string } `json:"dimensionValues"`
			MetricValues    []struct{ Value string } `json:"metricValues"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, fmt.Errorf("GA4 yanıt: %w", err)
	}
	sum := GA4Summary{Guncel: time.Now().Format(time.RFC3339)}
	for _, r := range out.Rows {
		page, n := "", 0
		if len(r.DimensionValues) > 0 {
			page = r.DimensionValues[0].Value
		}
		if len(r.MetricValues) > 0 {
			n, _ = strconv.Atoi(r.MetricValues[0].Value)
		}
		if page == "" {
			continue
		}
		sum.Sayfalar = append(sum.Sayfalar, GA4Page{Sayfa: page, Aktif: n})
		sum.Aktif += n
	}
	b, _ := json.Marshal(sum)
	if err := repo.SetSetting(ctx, pool, "trend", "ga4_ozet", b); err != nil {
		return sum.Aktif, err
	}
	return sum.Aktif, nil
}

func settingString(ctx context.Context, pool *pgxpool.Pool, section, key string) string {
	b, err := repo.GetSetting(ctx, pool, section, key)
	if err != nil {
		return ""
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		return s
	}
	return strings.Trim(string(b), `"`)
}

func truncateText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
