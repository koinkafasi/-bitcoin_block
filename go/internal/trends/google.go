package trends

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const userAgent = "theblock-trends/1.0 (+https://bitcosar.com)"

// GoogleTrendsProvider Google Trends Daily Search Trends RSS adaptörü.
type GoogleTrendsProvider struct {
	Client *http.Client
}

func (p *GoogleTrendsProvider) Kind() string { return "google_trends" }

func (p *GoogleTrendsProvider) Fetch(ctx context.Context, src Source) (Result, error) {
	body, etag, modified, notModified, err := fetchFeed(ctx, p.Client, src)
	if err != nil {
		return Result{}, err
	}
	if notModified {
		return Result{NotModified: true, ETag: etag, LastModified: modified}, nil
	}
	terms, err := ParseGoogleTrendsRSS(body)
	if err != nil {
		return Result{}, err
	}
	return Result{Terms: terms, ETag: etag, LastModified: modified}, nil
}

type googleTrendsFeed struct {
	Channel struct {
		Items []struct {
			Title         string `xml:"title"`
			ApproxTraffic string `xml:"approx_traffic"`
			Picture       string `xml:"picture"`
			PictureSource string `xml:"picture_source"`
			PubDate       string `xml:"pubDate"`
			News          []struct {
				Title   string `xml:"news_item_title"`
				Snippet string `xml:"news_item_snippet"`
				URL     string `xml:"news_item_url"`
				Picture string `xml:"news_item_picture"`
				Source  string `xml:"news_item_source"`
			} `xml:"news_item"`
		} `xml:"item"`
	} `xml:"channel"`
}

// ParseGoogleTrendsRSS ham XML'i yapılandırılmış terimlere çevirir.
// Google Trends'in `ht:` önekli alanları encoding/xml'de yerel adla eşleşir.
func ParseGoogleTrendsRSS(body []byte) ([]Term, error) {
	var feed googleTrendsFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("google trends XML: %w", err)
	}
	out := make([]Term, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		term := strings.TrimSpace(it.Title)
		if term == "" {
			continue
		}
		t := Term{
			Term:          term,
			ApproxTraffic: ParseApproxTraffic(it.ApproxTraffic),
			Picture:       strings.TrimSpace(it.Picture),
			PictureSource: strings.TrimSpace(it.PictureSource),
			PublishedAt:   parseFeedDate(it.PubDate),
		}
		for _, n := range it.News {
			title := strings.TrimSpace(n.Title)
			url := strings.TrimSpace(n.URL)
			if title == "" && url == "" {
				continue
			}
			source := strings.TrimSpace(n.Source)
			if source == "" {
				source = t.PictureSource
			}
			t.News = append(t.News, News{
				Title:   title,
				URL:     url,
				Source:  source,
				Picture: strings.TrimSpace(n.Picture),
				Snippet: stripMarkup(n.Snippet),
			})
		}
		out = append(out, t)
	}
	return out, nil
}

// GoogleNewsProvider Google News RSS adaptörü (haber yoğunluğu sinyali).
type GoogleNewsProvider struct {
	Client *http.Client
}

func (p *GoogleNewsProvider) Kind() string { return "google_news" }

func (p *GoogleNewsProvider) Fetch(ctx context.Context, src Source) (Result, error) {
	return fetchGenericRSS(ctx, p.Client, src, "google_news")
}

// RSSProvider genel RSS/Atom adaptörü (ajans ve özel kaynaklar).
type RSSProvider struct {
	Client *http.Client
}

func (p *RSSProvider) Kind() string { return "rss" }

func (p *RSSProvider) Fetch(ctx context.Context, src Source) (Result, error) {
	return fetchGenericRSS(ctx, p.Client, src, "rss")
}

// DisabledProvider yapılandırılmamış/ücretli adaptörler için kapalı kapıdır.
// Varsayılan olarak hiçbir dış istek yapmaz; panelde açıklama gösterilir.
type DisabledProvider struct{ Name string }

func (p *DisabledProvider) Kind() string { return p.Name }

func (p *DisabledProvider) Fetch(context.Context, Source) (Result, error) {
	return Result{}, errors.New("sağlayıcı etkin değil: bu kaynak türü için API anahtarı ve ödeme ayarı gerekli")
}

// ProviderFor kaynak türüne göre adaptör üretir.
func ProviderFor(kind string, client *http.Client) Provider {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "google_trends", "trends":
		return &GoogleTrendsProvider{Client: client}
	case "google_news":
		return &GoogleNewsProvider{Client: client}
	case "rss":
		return &RSSProvider{Client: client}
	case "dataforseo", "serpapi", "paid":
		return &DisabledProvider{Name: strings.ToLower(strings.TrimSpace(kind))}
	default:
		return &RSSProvider{Client: client}
	}
}

func fetchGenericRSS(ctx context.Context, client *http.Client, src Source, kind string) (Result, error) {
	body, etag, modified, notModified, err := fetchFeed(ctx, client, src)
	if err != nil {
		return Result{}, err
	}
	if notModified {
		return Result{NotModified: true, ETag: etag, LastModified: modified}, nil
	}
	var feed struct {
		Channel struct {
			Items []struct {
				Title   string `xml:"title"`
				Link    string `xml:"link"`
				Desc    string `xml:"description"`
				PubDate string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
		Entries []struct {
			Title   string `xml:"title"`
			Link    string `xml:"link"`
			Summary string `xml:"summary"`
			Updated string `xml:"updated"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(body, &feed); err != nil {
		return Result{}, fmt.Errorf("%s XML: %w", kind, err)
	}
	terms := make([]Term, 0, len(feed.Channel.Items)+len(feed.Entries))
	for _, it := range feed.Channel.Items {
		title := strings.TrimSpace(it.Title)
		link := strings.TrimSpace(it.Link)
		if title == "" {
			continue
		}
		snippet := stripMarkup(it.Desc)
		terms = append(terms, Term{
			Term:        title,
			PublishedAt: parseFeedDate(it.PubDate),
			News:        []News{{Title: title, URL: link, Snippet: snippet, Source: src.Name}},
		})
	}
	for _, it := range feed.Entries {
		title := strings.TrimSpace(it.Title)
		link := strings.TrimSpace(it.Link)
		if title == "" {
			continue
		}
		terms = append(terms, Term{
			Term:        title,
			PublishedAt: parseFeedDate(it.Updated),
			News:        []News{{Title: title, URL: link, Snippet: stripMarkup(it.Summary), Source: src.Name}},
		})
	}
	return Result{Terms: terms, ETag: etag, LastModified: modified}, nil
}

// fetchFeed ortak HTTP katmanı: ETag/If-Modified-Since, boyut sınırı ve
// 304 desteği. Dönen etag/modified, 304 olsa bile kaynakta saklanabilsin.
func fetchFeed(ctx context.Context, client *http.Client, src Source) (body []byte, etag, modified string, notModified bool, err error) {
	if strings.TrimSpace(src.URL) == "" {
		return nil, "", "", false, errors.New("kaynak URL boş")
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, "", "", false, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml;q=0.9, */*;q=0.5")
	if src.ETag != "" {
		req.Header.Set("If-None-Match", src.ETag)
	}
	if src.LastModified != "" {
		req.Header.Set("If-Modified-Since", src.LastModified)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", "", false, err
	}
	defer resp.Body.Close()
	etag = resp.Header.Get("ETag")
	modified = resp.Header.Get("Last-Modified")
	if resp.StatusCode == http.StatusNotModified {
		return nil, etag, modified, true, nil
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, etag, modified, false, fmt.Errorf("feed HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	body, err = io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, etag, modified, false, err
	}
	return body, etag, modified, false, nil
}

func parseFeedDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		time.RFC1123Z, time.RFC1123, time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 -0700", "2006-01-02T15:04:05Z07:00",
		"Mon, 2 Jan 2006 15:04:05 -0700",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// stripMarkup RSS açıklamalarındaki HTML etiketlerini temizler.
func stripMarkup(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
