package research

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Brave Brave Search API sağlayıcısıdır.
type Brave struct {
	apiKey string
	client *http.Client
	logger *slog.Logger
}

func (b *Brave) Name() string { return "brave" }

type braveResp struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Age         string `json:"age"`
			Profile     struct {
				Name string `json:"name"`
			} `json:"profile"`
		} `json:"results"`
	} `json:"web"`
}

func (b *Brave) Search(ctx context.Context, query string, count int) ([]Result, error) {
	if b.apiKey == "" {
		return nil, ErrDisabled
	}
	if count <= 0 || count > 20 {
		count = 8
	}
	u, _ := url.Parse("https://api.search.brave.com/res/v1/web/search")
	q := u.Query()
	q.Set("q", query)
	q.Set("count", strconv.Itoa(count))
	q.Set("country", "TR")
	q.Set("search_lang", "tr")
	q.Set("text_decorations", "false")
	q.Set("spellcheck", "false")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", b.apiKey)
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Brave Search %d: %s", resp.StatusCode, clip(string(raw), 200))
	}
	var br braveResp
	if err := json.Unmarshal(raw, &br); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(br.Web.Results))
	for _, r := range br.Web.Results {
		if strings.TrimSpace(r.URL) == "" {
			continue
		}
		item := Result{
			Title:       strings.TrimSpace(r.Title),
			URL:         r.URL,
			Description: strings.TrimSpace(r.Description),
		}
		if r.Profile.Name != "" {
			item.Publisher = r.Profile.Name
		}
		// Brave "age" alanı göreli zaman ("2 days ago") döndürür; kesin tarih
		// uydurmamak için yalnızca açıklamada tutulur, PublishedAt boş kalır.
		if r.Age != "" && item.Description == "" {
			item.Description = r.Age
		}
		out = append(out, item)
	}
	return out, nil
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
