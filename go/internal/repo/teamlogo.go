package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"theblock/site/internal/platform"
)

// TeamLogoURL kendi logo ucunun yolunu döndürür.
func TeamLogoURL(key string) string {
	if key == "" {
		return ""
	}
	return "/takim-logo/" + key + ".png"
}

// espnLogoID ESPN logo URL'sindeki sayısal takım kimliğini çıkarır.
// Örnek: https://a.espncdn.com/i/teamlogos/soccer/500/432.png → "432"
var espnLogoIDRe = regexp.MustCompile(`/teamlogos/[a-z]+/[0-9]+/([0-9]+)\.png`)

func TeamLogoKey(badgeURL, teamName string) string {
	if id := espnLogoIDRe.FindStringSubmatch(badgeURL); len(id) == 2 {
		return "e-" + id[1]
	}
	if s := slugKey(teamName); s != "" {
		return "t-" + s
	}
	return ""
}

// slugKey takım adını logo anahtarına çevirir: küçük harf, Türkçe karakter
// sadeleştirme, tire ayırıcı, en çok 40 karakter.
func slugKey(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch r {
		case 'ı', 'i', 'î':
			r = 'i'
		case 'ş', 's':
			r = 's'
		case 'ğ', 'g':
			r = 'g'
		case 'ü', 'û', 'u':
			r = 'u'
		case 'ö', 'o':
			r = 'o'
		case 'ç', 'c':
			r = 'c'
		case 'â', 'a':
			r = 'a'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 40 {
		out = strings.TrimRight(out[:40], "-")
	}
	return out
}

const (
	// teamLogoMaxBytes tek logonun üst sınırı. ESPN 500px PNG'leri 15–290 KB;
	// 2 MB rahat bir tavan ve disk doldurma riskini sınırlar.
	teamLogoMaxBytes = 2 << 20
	// teamLogoTimeout indirme zaman aşımı.
	teamLogoTimeout = 12 * time.Second
)

var teamLogoHTTP = platform.SafeFetchClient(teamLogoTimeout)

// mirrorTeamLogo logoyu gerekirse indirir ve yerel URL'ini döndürür.
// Başarısız olursa "" döner; çağıran dış URL'i korur.
func mirrorTeamLogo(ctx context.Context, key, srcURL string) string {
	if key == "" || srcURL == "" {
		return ""
	}
	if !platform.ValidTeamLogoKey(key) {
		return ""
	}
	// Zaten diskte varsa ağa hiç çıkmıyoruz: sayfa yenilemesi indirme tetiklemez.
	if platform.HasTeamLogo(key) {
		return TeamLogoURL(key)
	}
	data, err := downloadPNG(ctx, srcURL)
	if err != nil {
		return ""
	}
	if err := platform.SaveTeamLogo(key, data); err != nil {
		return ""
	}
	return TeamLogoURL(key)
}

func mirrorSourceURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	switch u.Host {
	case "a.espncdn.com", "a1.espncdn.com", "a2.espncdn.com", "a3.espncdn.com", "a4.espncdn.com":
		u.Host = "cdn.espn.com"
		return u.String()
	}
	return raw
}

func downloadPNG(ctx context.Context, srcURL string) ([]byte, error) {
	var sonErr error
	bekleme := time.Second
	for deneme := 0; deneme < 4; deneme++ {
		if deneme > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(bekleme):
			}
			bekleme *= 2
		}
		data, err := downloadPNGOnce(ctx, srcURL)
		if err == nil {
			return data, nil
		}
		sonErr = err
		if !yenidenDenenebilir(err) {
			return nil, err
		}
	}
	return nil, sonErr
}

func yenidenDenenebilir(err error) bool {
	var se statusErr
	if errors.As(err, &se) {
		switch int(se) {
		case http.StatusAccepted, http.StatusTooManyRequests,
			http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
		return false
	}
	// PNG olmayan gövde (boş 202, hata sayfası) ve ağ hataları geçici sayılır.
	return true
}

// downloadPNGOnce tek indirme denemesidir.
func downloadPNGOnce(ctx context.Context, srcURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, teamLogoTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mirrorSourceURL(srcURL), nil)
	if err != nil {
		return nil, err
	}
	// Bazı CDN'ler boş/tanınmayan User-Agent'ı reddediyor.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "image/png,image/*;q=0.8")
	resp, err := teamLogoHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errStatus(resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, teamLogoMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if !isPNG(data) {
		return nil, errors.New("gövde PNG değil (boş ya da hata sayfası)")
	}
	return data, nil
}

func isPNG(data []byte) bool {
	return len(data) > 512 && bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
}

type statusErr int

func (e statusErr) Error() string { return "logo HTTP " + http.StatusText(int(e)) }

func errStatus(code int) error { return statusErr(code) }

const logoPace = 150 * time.Millisecond

func mirrorStandingBadges(ctx context.Context, rows []StandingRow) {
	for i := range rows {
		if rows[i].Badge == "" || strings.HasPrefix(rows[i].Badge, "/takim-logo/") {
			continue
		}
		key := TeamLogoKey(rows[i].Badge, rows[i].Team)
		if key == "" || platform.HasTeamLogo(key) {
			if key != "" {
				rows[i].Badge = TeamLogoURL(key)
			}
			continue
		}
		if url := mirrorTeamLogo(ctx, key, rows[i].Badge); url != "" {
			rows[i].Badge = url
		}
		sleepCtx(ctx, logoPace)
	}
}

// mirrorFixtureBadges fikstür rozetlerini yerelleştirir.
func mirrorFixtureBadges(ctx context.Context, fr *FixtureRound) {
	for i := range fr.Matches {
		m := &fr.Matches[i]
		if m.HomeBadge != "" && !strings.HasPrefix(m.HomeBadge, "/takim-logo/") {
			if url := mirrorTeamLogo(ctx, TeamLogoKey(m.HomeBadge, m.Home), m.HomeBadge); url != "" {
				m.HomeBadge = url
			}
			sleepCtx(ctx, logoPace)
		}
		if m.AwayBadge != "" && !strings.HasPrefix(m.AwayBadge, "/takim-logo/") {
			if url := mirrorTeamLogo(ctx, TeamLogoKey(m.AwayBadge, m.Away), m.AwayBadge); url != "" {
				m.AwayBadge = url
			}
			sleepCtx(ctx, logoPace)
		}
	}
}

// sleepCtx bağlam iptal edilirse beklemeden döner.
func sleepCtx(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func LocalizeCachedBadges(ctx context.Context, pool DB) (int, error) {
	degisen := 0
	for _, kind := range []string{"puan", "fikstur-r"} {
		rows, err := CacheList(ctx, pool, kind)
		if err != nil {
			return degisen, err
		}
		for _, row := range rows {
			var yeni []byte
			if kind == "puan" {
				var sd StandingsData
				if json.Unmarshal(row.Data, &sd) != nil || len(sd.Rows) == 0 {
					continue
				}
				once := sayBadge(sd.Rows)
				mirrorStandingBadges(ctx, sd.Rows)
				if sayBadge(sd.Rows) == once {
					continue
				}
				yeni, _ = json.Marshal(sd)
			} else {
				var fr FixtureRound
				if json.Unmarshal(row.Data, &fr) != nil || len(fr.Matches) == 0 {
					continue
				}
				once := sayFixtureBadge(fr)
				mirrorFixtureBadges(ctx, &fr)
				if sayFixtureBadge(fr) == once {
					continue
				}
				yeni, _ = json.Marshal(fr)
			}
			if yeni == nil {
				continue
			}
			if err := CacheSet(ctx, pool, kind, row.Key, yeni); err != nil {
				return degisen, err
			}
			degisen++
		}
	}
	return degisen, nil
}

// sayBadge yerel rozet sayısını döndürür (değişim ölçütü).
func sayBadge(rows []StandingRow) int {
	n := 0
	for _, r := range rows {
		if strings.HasPrefix(r.Badge, "/takim-logo/") {
			n++
		}
	}
	return n
}

func sayFixtureBadge(fr FixtureRound) int {
	n := 0
	for _, m := range fr.Matches {
		if strings.HasPrefix(m.HomeBadge, "/takim-logo/") {
			n++
		}
		if strings.HasPrefix(m.AwayBadge, "/takim-logo/") {
			n++
		}
	}
	return n
}
