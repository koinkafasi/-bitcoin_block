// Package research gerçek web araması ve kaynak çekme adaptörlerini içerir.
// Sağlayıcı yoksa Disabled döner; sahte kaynak üretilmez.
package research

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ErrDisabled gerçek araştırma sağlayıcısı yapılandırılmadığında döner.
var ErrDisabled = errors.New("araştırma sağlayıcısı yapılandırılmadı")

// Result tek bir arama sonucudur. Content, güvenli şekilde çekilen sayfa
// metnidir; yalnız modele kanıt olarak verilir, kullanıcıya snippet gösterilir.
type Result struct {
	Title       string
	URL         string
	Description string
	Publisher   string
	PublishedAt *time.Time
	Content     string
}

// Provider web araması yapan sağlayıcıdır.
type Provider interface {
	Name() string
	Search(ctx context.Context, query string, count int) ([]Result, error)
}

// Disabled araştırma kapalıyken kullanılan sağlayıcıdır.
type Disabled struct{}

func (Disabled) Name() string { return "disabled" }
func (Disabled) Search(context.Context, string, int) ([]Result, error) {
	return nil, ErrDisabled
}

// FromEnv BRAVE_SEARCH_API_KEY varsa Brave, yoksa Disabled döndürür.
func FromEnv(logger *slog.Logger) Provider {
	key := strings.TrimSpace(os.Getenv("BRAVE_SEARCH_API_KEY"))
	if key == "" {
		return Disabled{}
	}
	return &Brave{apiKey: key, client: defaultClient(), logger: logger}
}

// Collect birden çok sorguyu çalıştırır, URL'ye göre tekilleştirir ve
// gerekiyorsa ilk sonuçların sayfa metnini güvenli biçimde çeker.
func Collect(ctx context.Context, p Provider, queries []string, max int, fetchPages bool) ([]Result, error) {
	if p == nil {
		return nil, ErrDisabled
	}
	if max <= 0 || max > 12 {
		max = 8
	}
	seen := map[string]bool{}
	var out []Result
	var firstErr error
	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		items, err := p.Search(ctx, q, max)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, it := range items {
			u, uerr := url.Parse(it.URL)
			if uerr != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
				continue
			}
			if seen[it.URL] {
				continue
			}
			seen[it.URL] = true
			if it.Publisher == "" {
				if u, err := url.Parse(it.URL); err == nil {
					it.Publisher = strings.TrimPrefix(u.Hostname(), "www.")
				}
			}
			out = append(out, it)
			if len(out) >= max {
				break
			}
		}
		if len(out) >= max {
			break
		}
	}
	if len(out) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, nil
	}
	if fetchPages {
		fetchTop(ctx, out, 4)
	}
	return out, nil
}

func fetchTop(ctx context.Context, items []Result, n int) {
	if n > len(items) {
		n = len(items)
	}
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			text, err := FetchText(ctx, items[i].URL)
			if err == nil {
				items[i].Content = text
			}
		}(i)
	}
	wg.Wait()
}
