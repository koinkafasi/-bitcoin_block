package repo

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"

	"theblock/site/internal/platform"
)

// HTML kaynak adaptörü: RSS yayınlamayan sitelerin ana sayfa/section
// listelerinden haber bağlantılarını çıkarır. Yeni bağlantıların ilk birkaç
// tanesi için makale sayfasındaki JSON-LD/meta özetiyle zenginleştirme yapılır.

const (
	htmlMaxBodyBytes   = 4 << 20
	htmlDetailBudget   = 6
	htmlMaxItemsPerRun = 30
)

var (
	htmlAnchorRE       = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a>`)
	htmlHrefRE         = regexp.MustCompile(`(?is)\bhref\s*=\s*["']([^"']+)["']`)
	htmlScriptStyleRE  = regexp.MustCompile(`(?is)<script\b.*?</script>|<style\b.*?</style>`)
	htmlTagRE          = regexp.MustCompile(`(?is)<[^>]+>`)
	htmlLDJSONRE       = regexp.MustCompile(`(?is)<script\b[^>]*type\s*=\s*["']?application/ld\+json["']?[^>]*>(.*?)</script>`)
	htmlMetaRE         = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	htmlAttrRE         = regexp.MustCompile(`(?is)([a-z:-]+)\s*=\s*["']([^"']*)["']`)
	htmlTitleRE        = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	htmlArticleIDRE    = regexp.MustCompile(`\d{5,}`)
	htmlRelativeTimeRE = regexp.MustCompile(`(?i)\s+(az önce|dün|bugün|\d+\s+(dakika|saat|gün|hafta|ay|yıl)\s+önce)\s*$`)
	htmlLiveTimeRE     = regexp.MustCompile(`(?i)^bugün\s+\d{1,2}:\d{2}`)
)

var htmlHTTPClient = platform.SafeFetchClient(20 * time.Second)

type htmlArticleMeta struct {
	Title       string
	Description string
	Section     string
	Published   time.Time
}

// ScanHTMLFeed HTML kaynağını tarar; yalnızca daha önce alınmamış yeni
// bağlantıları döndürür. İlk birkaç yeni bağlantı için makale özeti ve tarih
// bilgisi çekilir; bu sayede AI taslakları yalnızca başlığa mahkûm kalmaz.
func ScanHTMLFeed(ctx context.Context, pool *pgxpool.Pool, siteID int64, f Feed) ([]RSSItem, error) {
	listing, err := FetchHTMLListing(ctx, f.URL)
	if err != nil {
		return nil, err
	}
	if len(listing) == 0 {
		return nil, errors.New("HTML kaynağında haber bağlantısı bulunamadı")
	}
	out := make([]RSSItem, 0, htmlMaxItemsPerRun)
	detailBudget := htmlDetailBudget
	for _, it := range listing {
		if len(out) >= htmlMaxItemsPerRun {
			break
		}
		h := sha1.Sum([]byte(it.Link))
		var exists bool
		_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ai_source_items WHERE url_hash=$1)`,
			hex.EncodeToString(h[:])).Scan(&exists)
		if exists {
			continue
		}
		if detailBudget > 0 {
			detailBudget--
			if meta, err := FetchHTMLArticle(ctx, it.Link); err == nil {
				it.Enriched = true
				if meta.Title != "" {
					it.Title = meta.Title
				}
				if meta.Description != "" {
					it.Description = meta.Description
				}
				if !meta.Published.IsZero() {
					it.PubDate = meta.Published
				}
				if meta.Section != "" {
					it.Section = meta.Section
				}
			}
		}
		out = append(out, it)
	}
	return out, nil
}

// FetchHTMLListing bir HTML sayfasındaki haber bağlantılarını çıkarır.
func FetchHTMLListing(ctx context.Context, pageURL string) ([]RSSItem, error) {
	body, err := fetchHTMLDocument(ctx, pageURL, "text/html,application/xhtml+xml")
	if err != nil {
		return nil, err
	}
	return parseHTMLListing(pageURL, string(body)), nil
}

func parseHTMLListing(pageURL, doc string) []RSSItem {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}
	baseHost := normalizeHTMLHost(base.Host)
	items := make([]RSSItem, 0, htmlMaxItemsPerRun)
	byURL := map[string]int{}
	byTitle := map[string]bool{}
	for _, m := range htmlAnchorRE.FindAllStringSubmatch(doc, -1) {
		h := htmlHrefRE.FindStringSubmatch(m[1])
		if len(h) < 2 {
			continue
		}
		href := strings.TrimSpace(html.UnescapeString(h[1]))
		if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(strings.ToLower(href), "javascript:") {
			continue
		}
		u, err := base.Parse(href)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			continue
		}
		if normalizeHTMLHost(u.Host) != baseHost {
			continue
		}
		u.Fragment = ""
		stripTrackingParams(u)
		if !looksLikeArticleURL(u) {
			continue
		}
		title := cleanHTMLTitle(m[2])
		if !validListingTitle(title) {
			continue
		}
		key := u.String()
		description := title
		if len([]rune(description)) >= 60 {
			description = truncateRunes(description, 600)
		} else {
			description = ""
		}
		if idx, ok := byURL[key]; ok {
			if len([]rune(description)) > len([]rune(items[idx].Description)) && description != items[idx].Title {
				items[idx].Description = description
			}
			continue
		}
		titleKey := normalizeNewsTitle(title)
		if byTitle[titleKey] {
			continue
		}
		byTitle[titleKey] = true
		byURL[key] = len(items)
		items = append(items, RSSItem{Title: title, Link: key, Description: description})
		if len(items) >= htmlMaxItemsPerRun*2 {
			break
		}
	}
	return items
}

// FetchHTMLArticle makale sayfasındaki NewsArticle JSON-LD'sini, ardından
// Open Graph/meta açıklamasını okur.
func FetchHTMLArticle(ctx context.Context, articleURL string) (htmlArticleMeta, error) {
	body, err := fetchHTMLDocument(ctx, articleURL, "text/html,application/xhtml+xml")
	if err != nil {
		return htmlArticleMeta{}, err
	}
	return parseHTMLArticleMeta(string(body)), nil
}

func parseHTMLArticleMeta(doc string) htmlArticleMeta {
	var meta htmlArticleMeta
	if node := firstJSONLDArticle(doc); node != nil {
		meta.Title = cleanHTMLText(jsonLDString(node["headline"]))
		meta.Description = cleanHTMLText(jsonLDString(node["description"]))
		meta.Section = cleanHTMLText(jsonLDString(node["articleSection"]))
		meta.Published = parseRSSDate(jsonLDString(node["datePublished"]))
	}
	metas := map[string]string{}
	for _, tag := range htmlMetaRE.FindAllString(doc, -1) {
		attrs := map[string]string{}
		for _, a := range htmlAttrRE.FindAllStringSubmatch(tag, -1) {
			attrs[strings.ToLower(a[1])] = html.UnescapeString(a[2])
		}
		key := strings.ToLower(attrs["property"])
		if key == "" {
			key = strings.ToLower(attrs["name"])
		}
		if key != "" {
			metas[key] = attrs["content"]
		}
	}
	if meta.Title == "" {
		meta.Title = cleanHTMLText(metas["og:title"])
	}
	if meta.Description == "" {
		meta.Description = cleanHTMLText(metas["og:description"])
	}
	if meta.Description == "" {
		meta.Description = cleanHTMLText(metas["description"])
	}
	if meta.Section == "" {
		meta.Section = cleanHTMLText(metas["article:section"])
	}
	if meta.Section == "" {
		meta.Section = cleanHTMLText(metas["articlesection"])
	}
	if isGenericSection(meta.Section) {
		if sec := breadcrumbSection(doc); sec != "" {
			meta.Section = sec
		}
	}
	if meta.Published.IsZero() {
		meta.Published = parseRSSDate(metas["article:published_time"])
	}
	if meta.Title == "" {
		if m := htmlTitleRE.FindStringSubmatch(doc); len(m) == 2 {
			meta.Title = cleanHTMLText(m[1])
		}
	}
	meta.Title = truncateRunes(meta.Title, 220)
	meta.Title = cleanHTMLTitle(meta.Title)
	meta.Description = truncateRunes(meta.Description, 600)
	return meta
}

func firstJSONLDArticle(doc string) map[string]any {
	for _, m := range htmlLDJSONRE.FindAllStringSubmatch(doc, -1) {
		var v any
		if err := json.Unmarshal([]byte(strings.TrimSpace(m[1])), &v); err != nil {
			continue
		}
		if node := findJSONLDArticle(v, 0); node != nil {
			return node
		}
	}
	return nil
}

func findJSONLDArticle(v any, depth int) map[string]any {
	if depth > 8 {
		return nil
	}
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			if node := findJSONLDArticle(item, depth+1); node != nil {
				return node
			}
		}
	case map[string]any:
		if jsonLDTypeIsArticle(x["@type"]) {
			return x
		}
		if graph, ok := x["@graph"]; ok {
			if node := findJSONLDArticle(graph, depth+1); node != nil {
				return node
			}
		}
	}
	return nil
}

func isGenericSection(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "news", "haber", "haberler", "genel", "general", "default":
		return true
	}
	return false
}

func breadcrumbSection(doc string) string {
	for _, m := range htmlLDJSONRE.FindAllStringSubmatch(doc, -1) {
		var v any
		if err := json.Unmarshal([]byte(strings.TrimSpace(m[1])), &v); err != nil {
			continue
		}
		if sec := findBreadcrumbSection(v, 0); sec != "" {
			return sec
		}
	}
	return ""
}

func findBreadcrumbSection(v any, depth int) string {
	if depth > 8 {
		return ""
	}
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			if sec := findBreadcrumbSection(item, depth+1); sec != "" {
				return sec
			}
		}
	case map[string]any:
		if jsonLDTypeIsBreadcrumb(x["@type"]) {
			items, _ := x["itemListElement"].([]any)
			fallback := ""
			for i, raw := range items {
				m, _ := raw.(map[string]any)
				name, id := breadcrumbItem(m)
				if name == "" {
					continue
				}
				lowerID := strings.ToLower(id)
				if strings.Contains(lowerID, "/kategori/") || strings.Contains(lowerID, "/category/") ||
					strings.Contains(lowerID, "/kategori-") {
					return name
				}
				if i > 0 && i < len(items)-1 && !isHomeSection(name) {
					fallback = name
				}
			}
			if fallback != "" {
				return fallback
			}
		}
		if graph, ok := x["@graph"]; ok {
			return findBreadcrumbSection(graph, depth+1)
		}
	}
	return ""
}

func breadcrumbItem(m map[string]any) (string, string) {
	name := jsonLDString(m["name"])
	id := jsonLDString(m["@id"])
	if item, ok := m["item"].(map[string]any); ok {
		if name == "" {
			name = jsonLDString(item["name"])
		}
		if id == "" {
			id = jsonLDString(item["@id"])
		}
	}
	return name, id
}

func isHomeSection(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "anasayfa", "ana sayfa", "home", "startseite":
		return true
	}
	return false
}

func jsonLDTypeIsBreadcrumb(v any) bool {
	switch x := v.(type) {
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "BreadcrumbList")
	case []any:
		for _, item := range x {
			if s, ok := item.(string); ok && strings.EqualFold(strings.TrimSpace(s), "BreadcrumbList") {
				return true
			}
		}
	}
	return false
}

func jsonLDTypeIsArticle(v any) bool {
	var types []string
	switch x := v.(type) {
	case string:
		types = append(types, x)
	case []any:
		for _, item := range x {
			if s, ok := item.(string); ok {
				types = append(types, s)
			}
		}
	}
	for _, t := range types {
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "newsarticle", "reportagenewsarticle", "analysisnewsarticle", "article", "liveblogposting", "blogposting":
			return true
		}
	}
	return false
}

func jsonLDString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		for _, item := range x {
			if s := jsonLDString(item); s != "" {
				return s
			}
		}
	}
	return ""
}

func fetchHTMLDocument(ctx context.Context, rawURL, accept string) ([]byte, error) {
	// Adres panelden (AI Kaynak / HTML akış) girer: SSRF korumalı istemci
	// kullanılır. Loopback/özel/metadata adresleri ve 80/443 dışı portlar
	// reddedilir.
	u, perr := url.Parse(rawURL)
	if perr != nil {
		return nil, perr
	}
	if err := platform.ValidateOutboundURL(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", accept+",*/*;q=0.8")
	req.Header.Set("Accept-Language", "tr-TR,tr;q=0.9,en;q=0.7")
	resp, err := htmlHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("HTML kaynağı HTTP " + resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, htmlMaxBodyBytes))
	if err != nil {
		return nil, err
	}
	return decodeHTMLCharset(body), nil
}

func decodeHTMLCharset(body []byte) []byte {
	if utf8.Valid(body) {
		return body
	}
	sample := body
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	low := strings.ToLower(string(sample))
	if strings.Contains(low, "windows-1254") || strings.Contains(low, "iso-8859-9") || strings.Contains(low, "latin5") {
		if out, err := charmap.Windows1254.NewDecoder().Bytes(body); err == nil {
			return out
		}
	}
	return body
}

func cleanHTMLText(s string) string {
	s = htmlScriptStyleRE.ReplaceAllString(s, " ")
	s = htmlTagRE.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

func cleanHTMLTitle(s string) string {
	s = cleanHTMLText(s)
	return strings.TrimSpace(htmlRelativeTimeRE.ReplaceAllString(s, ""))
}

func normalizeHTMLHost(host string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(host)), "www.")
}

func stripTrackingParams(u *url.URL) {
	if u.RawQuery == "" {
		return
	}
	q := u.Query()
	for key := range q {
		lk := strings.ToLower(key)
		if strings.HasPrefix(lk, "utm_") || lk == "fbclid" || lk == "gclid" || lk == "yclid" || lk == "ref" {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
}

func looksLikeArticleURL(u *url.URL) bool {
	path := strings.TrimSuffix(u.Path, "/")
	if path == "" || path == "/" {
		return false
	}
	low := strings.ToLower(path)
	for _, bad := range []string{
		"/etiket/", "/kategori/", "/yazar/", "/author/", "/tag/", "/galeri/",
		"/foto-galeri/", "/video/", "/canli/", "/arama/", "/search/", "/page/",
		"/hakkimizda", "/iletisim", "/gizlilik", "/kunye", "/reklam", "/abone",
		"/uyelik", "/login", "/giris", "/rss", "/feed", "/sitemap", ".xml", ".json",
		"/forum/", "/konu/", "/uye/",
	} {
		if strings.Contains(low, bad) {
			return false
		}
	}
	seg := low[strings.LastIndex(low, "/")+1:]
	if seg == "" || seg == "index.html" || seg == "index.php" {
		return false
	}
	if htmlArticleIDRE.MatchString(seg) || strings.Count(seg, "-") >= 3 {
		return true
	}
	return strings.Contains(low, "/haber/") || strings.Contains(low, "/news/")
}

func validListingTitle(title string) bool {
	n := len([]rune(title))
	if n < 30 || n > 220 || strings.Count(title, " ") < 2 {
		return false
	}
	low := strings.ToLower(title)
	for _, bad := range []string{
		"devamını oku", "haberin devamı", "tüm hakları", "çerez politikası",
		"gizlilik politikası", "abone ol", "giriş yap", "reklam ver",
	} {
		if low == bad || strings.HasPrefix(low, bad+" ") {
			return false
		}
	}
	if strings.HasPrefix(strings.ToUpper(title), "CANLI ") {
		return false
	}
	if htmlLiveTimeRE.MatchString(title) {
		return false
	}
	return true
}

func normalizeNewsTitle(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSpace = false
		case r >= 'ç' && r <= 'ü' || r == 'ı' || r == 'ş' || r == 'ğ' || r == 'ö':
			b.WriteRune(r)
			lastSpace = false
		default:
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// sectionCategorySlug, HTML/JSON-LD bölüm adını site kategorisine eşler.
func sectionCategorySlug(section string) string {
	k := strings.ToLower(strings.TrimSpace(section))
	k = strings.NewReplacer(
		"İ", "i", "I", "i", "ı", "i",
		"Ş", "s", "ş", "s", "Ç", "c", "ç", "c",
		"Ğ", "g", "ğ", "g", "Ü", "u", "ü", "u", "Ö", "o", "ö", "o",
		"Â", "a", "â", "a", "Î", "i", "î", "i", "Û", "u", "û", "u",
	).Replace(section)
	k = strings.ToLower(k)
	switch {
	case strings.Contains(k, "gundem"), strings.Contains(k, "guncel"), strings.Contains(k, "asayis"), strings.Contains(k, "politika"):
		return "gundem"
	case strings.Contains(k, "ekonomi"), strings.Contains(k, "finans"), strings.Contains(k, "is dunyasi"),
		strings.Contains(k, "piyasa"), strings.Contains(k, "borsa"), strings.Contains(k, "kap haber"),
		strings.Contains(k, "hisse"), strings.Contains(k, "doviz"), strings.Contains(k, "altin"),
		strings.Contains(k, "emtia"), strings.Contains(k, "kripto"):
		return "ekonomi"
	case strings.Contains(k, "spor"):
		return "spor"
	case strings.Contains(k, "dunya"), strings.Contains(k, "dis haber"):
		return "dunya"
	case strings.Contains(k, "teknoloji"), strings.Contains(k, "bilim"):
		return "teknoloji"
	case strings.Contains(k, "sinema"), strings.Contains(k, "film"):
		return "sinema"
	case strings.Contains(k, "tarih"):
		return "tarih"
	case strings.Contains(k, "turizm"), strings.Contains(k, "seyahat"):
		return "turizm"
	case strings.Contains(k, "oyun"):
		return "oyun"
	case strings.Contains(k, "mimari"), strings.Contains(k, "emlak"):
		return "mimari"
	case strings.Contains(k, "kultur"), strings.Contains(k, "sanat"), strings.Contains(k, "egitim"):
		return "egitim-kultur"
	case strings.Contains(k, "saglik"), strings.Contains(k, "yasam"), strings.Contains(k, "magazin"):
		return "yasam"
	}
	return ""
}
