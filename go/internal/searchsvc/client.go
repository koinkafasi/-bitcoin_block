package searchsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client Rust/Tantivy search-svc istemcisidir (spec 03 P4).
// Servis yoksa/kapalıysa çağıran taraf SQL aramasına (pg_trgm) düşer.
type Client struct {
	BaseURL string
	http    *http.Client
}

// NewClient SEARCH_SVC_URL verilirse istemci üretir; verilmezse nil döner.
func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	return &Client{
		BaseURL: baseURL,
		http:    &http.Client{Timeout: 3 * time.Second},
	}
}

// Doc indekslenecek belgedir.
type Doc struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Spot     string   `json:"spot"`
	Tags     []string `json:"tags,omitempty"`
	URL      string   `json:"url"`
	Category string   `json:"category,omitempty"`
}

// Hit arama sonucudur.
type Hit struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	URL   string  `json:"url"`
	Score float32 `json:"score"`
}

type searchResp struct {
	Hits   []Hit `json:"hits"`
	Total  int   `json:"total"`
	TookMs int64 `json:"took_ms"`
}

// Healthy servis erişilebilir mi?
func (c *Client) Healthy(ctx context.Context) bool {
	if c == nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/healthz", nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Index belgeleri toplu indeksler.
func (c *Client) Index(ctx context.Context, docs []Doc) (int, error) {
	if c == nil || len(docs) == 0 {
		return 0, nil
	}
	body, err := json.Marshal(map[string]any{"docs": docs})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/index", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("search-svc %d", resp.StatusCode)
	}
	var out struct {
		Indexed int `json:"indexed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	return out.Indexed, nil
}

// Search sorguyu çalıştırır; hata durumunda çağıran taraf yedek yola düşer.
func (c *Client) Search(ctx context.Context, q string, limit int) ([]Hit, error) {
	if c == nil || strings.TrimSpace(q) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	u := fmt.Sprintf("%s/v1/search?q=%s&limit=%d", c.BaseURL, urlQueryEscape(q), limit)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search-svc %d", resp.StatusCode)
	}
	var out searchResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Hits, nil
}

// Stats indeks istatistikleri.
func (c *Client) Stats(ctx context.Context) (map[string]any, error) {
	if c == nil {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/v1/stats", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func urlQueryEscape(s string) string { return url.QueryEscape(s) }
