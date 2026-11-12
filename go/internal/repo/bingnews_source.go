package repo

import (
	"context"
	"errors"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
)

const bingNewsMaxBodyBytes = 4 << 20

func FetchBingNews(ctx context.Context, rawURL string) ([]RSSItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")
	req.Header.Set("Accept-Language", "tr-TR,tr;q=0.9,en;q=0.7")
	resp, err := htmlHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Bing News HTTP " + resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, bingNewsMaxBodyBytes))
	if err != nil {
		return nil, err
	}
	items, err := parseRSS(string(body))
	if err != nil {
		return nil, err
	}
	out := make([]RSSItem, 0, len(items))
	for _, it := range items {
		link := msnArticleURL(it.Link)
		if link == "" {
			continue
		}
		it.Link = link
		it.Section = msnSection(link)
		it.Title = truncateRunes(strings.TrimSpace(it.Title), 300)
		it.Description = truncateRunes(strings.TrimSpace(it.Description), 600)
		if it.Title == "" {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

func msnArticleURL(raw string) string {
	u, err := neturl.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	target := u.Query().Get("url")
	if target == "" {
		target = raw
	}
	tu, err := neturl.Parse(target)
	if err != nil {
		return ""
	}
	host := strings.ToLower(tu.Host)
	if host != "msn.com" && !strings.HasSuffix(host, ".msn.com") {
		return ""
	}
	tu.Scheme = "https"
	tu.Fragment = ""
	return tu.String()
}

func msnSection(raw string) string {
	u, err := neturl.Parse(raw)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, p := range parts {
		if strings.EqualFold(p, "tr-tr") && i+1 < len(parts) {
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

const kesifAzamiYas = 180 * 24 * time.Hour

func FetchBingLinkFeed(ctx context.Context, rawURL, izinliHost string) ([]RSSItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := htmlHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Bing News HTTP " + resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, bingNewsMaxBodyBytes))
	if err != nil {
		return nil, err
	}
	items, err := parseRSS(string(body))
	if err != nil {
		return nil, err
	}
	host := strings.ToLower(strings.TrimSpace(izinliHost))
	out := make([]RSSItem, 0, len(items))
	for _, it := range items {
		link := apiclickHedef(it.Link)
		if link == "" {
			continue
		}
		if host != "" {
			u, perr := neturl.Parse(link)
			if perr != nil {
				continue
			}
			h := strings.ToLower(u.Host)
			if h != host && !strings.HasSuffix(h, "."+host) {
				continue
			}
		}
		if !it.PubDate.IsZero() && time.Since(it.PubDate) > kesifAzamiYas {
			continue
		}
		if it.PubDate.IsZero() {
			it.PubDate = time.Now()
		}
		it.Link = link
		it.Section = yolBolum(link)
		it.Title = truncateRunes(strings.TrimSpace(it.Title), 300)
		it.Description = truncateRunes(strings.TrimSpace(it.Description), 600)
		if it.Title == "" || it.Link == "" {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

// apiclickHedef Bing apiclick yönlendirmesinden gerçek hedef adresi çıkarır.
// Yönlendirme değilse adresin kendisi döner.
func apiclickHedef(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := neturl.Parse(raw)
	if err != nil {
		return ""
	}
	target := u.Query().Get("url")
	if target == "" {
		target = raw
	}
	tu, err := neturl.Parse(target)
	if err != nil || (tu.Scheme != "http" && tu.Scheme != "https") || tu.Host == "" {
		return ""
	}
	return tu.String()
}

// yolBolum adresin ilk yol parçasını bölüm adı sayar
// (…/opinion/2026/10/31/x → "opinion").
func yolBolum(raw string) string {
	u, err := neturl.Parse(raw)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}
