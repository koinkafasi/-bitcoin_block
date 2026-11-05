package repo

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"html"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/htmlindex"

	"theblock/site/internal/classify"
	"theblock/site/internal/platform"
	"theblock/site/internal/scoring"
)

// Feed ajans/RSS kaynağıdır.
type Feed struct {
	ID         int64
	SiteID     int64
	Name       string
	URL        string
	CategoryID *int64
	AuthorID   *int64
	Aktif      bool
	SortOrder  int
	Homepage   string
	Kind       string
	Trust      int
	Notes      string
	// Aggregator feed başka yayıncıların haberlerini topluyorsa true
	// (MSN, Google News). Bot sırasını ve mükerrer eşiğini etkiler.
	Aggregator    bool
	LastCheckedAt *time.Time
	LastStatus    string
	LastError     string
	LastItemCount int
}

// CreateFeed yeni feed ekler.
func CreateFeed(ctx context.Context, pool *pgxpool.Pool, f *Feed) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO feeds (site_id, name, url, category_id, author_id, aktif)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		f.SiteID, f.Name, f.URL, nullableInt(f.CategoryID), nullableInt64P(f.AuthorID), f.Aktif).Scan(&id)
	return id, err
}

// ListFeeds feed'leri döndürür.
func ListFeeds(ctx context.Context, pool *pgxpool.Pool) ([]Feed, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, name, url, category_id, author_id, aktif, sort_order,
		       COALESCE(homepage,''), COALESCE(kind,'rss'), COALESCE(trust,50), COALESCE(notes,''),
		       last_checked_at, COALESCE(last_status,''), COALESCE(last_error,''), COALESCE(last_item_count,0),
		       aggregator
		FROM feeds ORDER BY aggregator, trust DESC, sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Feed
	for rows.Next() {
		var f Feed
		if err := rows.Scan(&f.ID, &f.SiteID, &f.Name, &f.URL, &f.CategoryID, &f.AuthorID, &f.Aktif, &f.SortOrder,
			&f.Homepage, &f.Kind, &f.Trust, &f.Notes, &f.LastCheckedAt, &f.LastStatus, &f.LastError, &f.LastItemCount,
			&f.Aggregator); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// DeleteFeed feed siler.
func DeleteFeed(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM feeds WHERE id = $1`, id)
	return err
}

// SetFeedAktif toggle eder.
func SetFeedAktif(ctx context.Context, pool *pgxpool.Pool, id int64, aktif bool) error {
	_, err := pool.Exec(ctx, `UPDATE feeds SET aktif = $1 WHERE id = $2`, aktif, id)
	return err
}

// WordRule kelime değiştirme kuralıdır.
type WordRule struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// LoadWordRules bot/kelimeler ayarını okur.
func LoadWordRules(ctx context.Context, pool *pgxpool.Pool) ([]WordRule, error) {
	b, err := GetSetting(ctx, pool, "bot", "kelimeler")
	if err != nil {
		return nil, nil
	}
	var rules []WordRule
	if err := json.Unmarshal(b, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// SaveWordRules kuralları kaydeder.
func SaveWordRules(ctx context.Context, pool *pgxpool.Pool, rules []WordRule) error {
	b, _ := json.Marshal(rules)
	return SetSetting(ctx, pool, "bot", "kelimeler", b)
}

// ApplyReplacer kuralları metne uygular.
func ApplyReplacer(s string, rules []WordRule) string {
	for _, r := range rules {
		if r.From != "" {
			s = strings.ReplaceAll(s, r.From, r.To)
		}
	}
	return s
}

// RSSItem parse edilen feed öğesidir.
type RSSItem struct {
	Title       string
	Link        string
	Description string
	PubDate     time.Time
	Section     string
	Categories  []string
	Enriched    bool
	// Image kaynağın verdiği kapak görseli (mutlak URL). Boş olabilir.
	Image string
}

// FetchRSS URL'den RSS 2.0 / Atom / RDF öğelerini okur.
// Akış adresi panelden girilir → SSRF korumalı istemci kullanılır.
func FetchRSS(url string) ([]RSSItem, error) {
	u, perr := neturl.Parse(url)
	if perr != nil {
		return nil, perr
	}
	if err := platform.ValidateOutboundURL(u); err != nil {
		return nil, err
	}
	client := platform.SafeFetchClient(20 * time.Second)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Bazı haber siteleri bot User-Agent'ını 403 ile engelliyor; tarayıcı
	// başlıkları RSS erişimini güvenilir hale getiriyor.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")
	req.Header.Set("Accept-Language", "tr-TR,tr;q=0.9,en;q=0.7")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("feed HTTP " + resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	// Ölçüm (28 Ekim 2026): bot çekimlerinin %52'si hata veriyordu. Üç ayrı
	// kök neden: (1) feed yerine HTML sayfası dönen adresler, (2) RSS
	// gövdesindeki tanımsız HTML varlıkları (&nbsp;, &l…), (3) UTF-8
	// olmayan gövdeler (windows-1254). Üçü de burada, ayrıştırmadan önce
	// ele alınıyor; ham XML hatası yerine anlaşılır mesaj dönüyor.
	body, err = decodeCharset(body, contentType)
	if err != nil {
		return nil, err
	}
	if looksLikeHTML(contentType, body) {
		return nil, errors.New("adres RSS değil, HTML sayfası döndü — feed adresini düzeltin")
	}
	return parseRSS(string(body))
}

// looksLikeHTML sunucunun RSS yerine web sayfası döndürüp döndürmediğini
// söyler. Bu, yanlış yazılmış bir feed adresinin en sık belirtisidir ve
// "XML syntax error: attribute name without =" mesajından çok daha anlaşılır.
func looksLikeHTML(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		return true
	}
	head := strings.ToLower(string(body[:min(len(body), 512)]))
	head = strings.TrimSpace(head)
	return strings.HasPrefix(head, "<!doctype html") || strings.HasPrefix(head, "<html")
}

// decodeCharset gövdeyi UTF-8'e çevirir.
//
// Sıra: Content-Type charset → <?xml encoding=...?> → geçerlilik denetimi.
// Bildirilen kodlama UTF-8 olsa bile gövde geçersiz bayt içerebilir (Türkçe
// sitelerde sık); o durumda windows-1254'e düşülür.
func decodeCharset(body []byte, contentType string) ([]byte, error) {
	label := charsetParam(contentType)
	if label == "" {
		label = xmlEncodingDecl(body)
	}
	label = strings.ToLower(strings.TrimSpace(label))

	if label != "" && label != "utf-8" && label != "utf8" {
		if enc, err := htmlindex.Get(label); err == nil {
			out, derr := enc.NewDecoder().Bytes(body)
			if derr == nil {
				body = out
			}
		}
	}
	if utf8.Valid(body) {
		return body, nil
	}
	// Bildirilen kodlama yanlış: Türkçe haber sitelerinde yaygın olan
	// windows-1254 ile son bir deneme.
	if enc, err := htmlindex.Get("windows-1254"); err == nil {
		if out, derr := enc.NewDecoder().Bytes(body); derr == nil {
			return out, nil
		}
	}
	return body, nil
}

// charsetParam Content-Type başlığındaki charset değerini döndürür.
func charsetParam(contentType string) string {
	for _, part := range strings.Split(contentType, ";") {
		part = strings.TrimSpace(part)
		if len(part) > 8 && strings.EqualFold(part[:8], "charset=") {
			return strings.Trim(part[8:], `"' `)
		}
	}
	return ""
}

// xmlEncodingDecl XML bildirimindeki encoding değerini döndürür.
func xmlEncodingDecl(body []byte) string {
	head := string(body[:min(len(body), 200)])
	i := strings.Index(strings.ToLower(head), "encoding=")
	if i < 0 {
		return ""
	}
	rest := head[i+len("encoding="):]
	rest = strings.TrimLeft(rest, `"' `)
	end := strings.IndexAny(rest, `"'?> `)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// parseRSS basit RSS 2.0 / Atom / RDF ayrıştırıcı.
func parseRSS(doc string) ([]RSSItem, error) {
	var feed struct {
		Channel struct {
			Items []rssXMLItem `xml:"item"`
		} `xml:"channel"`
		// RDF (RSS 1.0) biçiminde öğeler doğrudan kökün altındadır ve tarih
		// dc:date alanındadır. DW Türkçe bu biçimi kullanıyor.
		RDFItems  []rssXMLItem `xml:"item"`
		AtomItems []struct {
			Title      string `xml:"title"`
			Link       string `xml:"link"`
			Content    string `xml:"content"`
			Summary    string `xml:"summary"`
			Updated    string `xml:"updated"`
			Categories []struct {
				Term  string `xml:"term,attr"`
				Label string `xml:"label,attr"`
			} `xml:"category"`
		} `xml:"entry"`
	}
	raw := sanitizeXML(doc)
	dec := xml.NewDecoder(strings.NewReader(raw))
	dec.Strict = false
	if err := dec.Decode(&feed); err != nil {
		return nil, err
	}
	var out []RSSItem
	for _, it := range append(feed.Channel.Items, feed.RDFItems...) {
		body := it.Desc
		if body == "" {
			body = it.Content
		}
		cats := make([]string, 0, len(it.Categories))
		for _, c := range it.Categories {
			if c = strings.TrimSpace(c); c != "" {
				cats = append(cats, c)
			}
		}
		date := it.PubDate
		if date == "" {
			date = it.DCDate
		}
		out = append(out, RSSItem{Title: strings.TrimSpace(it.Title), Link: strings.TrimSpace(it.Link),
			Description: stripHTML(body), PubDate: parseRSSDate(date), Categories: cats,
			Image: rssGorsel(it, body)})
	}
	for _, it := range feed.AtomItems {
		body := it.Summary
		if body == "" {
			body = it.Content
		}
		cats := make([]string, 0, len(it.Categories))
		for _, c := range it.Categories {
			raw := strings.TrimSpace(c.Term)
			if raw == "" {
				raw = strings.TrimSpace(c.Label)
			}
			if raw != "" {
				cats = append(cats, raw)
			}
		}
		out = append(out, RSSItem{Title: strings.TrimSpace(it.Title), Link: strings.TrimSpace(it.Link),
			Description: stripHTML(body), PubDate: parseRSSDate(it.Updated), Categories: cats,
			Image: rssHTMLGorsel(body)})
	}
	return out, nil
}

// rssGorsel bir RSS 2.0/RDF öğesinin kapak görselini bulur.
//
// Sıra bilinçli: media:thumbnail en küçük/kırpılmış olan, media:content ise
// yayın için hazırlanmış geniş görsel. Bu yüzden content → thumbnail →
// enclosure → gövde HTML'i sırası izlenir. Tüm adaylar http(s) olmalı;
// veri: URI'leri ve göreli yollar atılır (şablon bunları gösteremez).
func rssGorsel(it rssXMLItem, govde string) string {
	for _, m := range it.Medias {
		if u := gecerliGorselURL(m.URL); u != "" && (m.Medium == "" || m.Medium == "image") {
			return u
		}
	}
	for _, m := range it.Thumbnails {
		if u := gecerliGorselURL(m.URL); u != "" {
			return u
		}
	}
	for _, e := range it.Enclosures {
		if u := gecerliGorselURL(e.URL); u != "" && (e.Type == "" || strings.HasPrefix(e.Type, "image/")) {
			return u
		}
	}
	return rssHTMLGorsel(govde)
}

// rssHTMLGorsel gövde HTML'i içindeki ilk <img src> adresini döndürür.
func rssHTMLGorsel(govde string) string {
	lower := strings.ToLower(govde)
	for i := 0; ; {
		j := strings.Index(lower[i:], "<img")
		if j < 0 {
			return ""
		}
		j += i
		k := strings.Index(lower[j:], "src=")
		if k < 0 {
			return ""
		}
		k += j + len("src=")
		if k >= len(govde) {
			return ""
		}
		quote := govde[k]
		if quote != '"' && quote != '\'' {
			i = k
			continue
		}
		rest := govde[k+1:]
		end := strings.IndexByte(rest, quote)
		if end < 0 {
			return ""
		}
		if u := gecerliGorselURL(rest[:end]); u != "" {
			return u
		}
		i = k + 1 + end
	}
}

// gecerliGorselURL yalnız mutlak http(s) adreslerini kabul eder.
//
// NEDEN: feed'lerde `data:` URI'leri (base64) ve `//cdn...` gibi şemasız
// yollar var. data: URI'yi <img src> olarak basmak HTML'i megabaytlarca
// şişirir; şemasız yolu basmak sayfayı https'te karışık içerik hatasına
// düşürür. İkisi de sessiz bozulma; burada elenir.
func gecerliGorselURL(raw string) string {
	u := strings.TrimSpace(html.UnescapeString(raw))
	if u == "" {
		return ""
	}
	lu := strings.ToLower(u)
	if strings.HasPrefix(lu, "https://") || strings.HasPrefix(lu, "http://") {
		return u
	}
	return ""
}

func stripHTML(s string) string {
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
	ham := b.String()
	// Agregatör özetleri başlıkları KIRILMASIZ BOŞLUKLA yapıştırıyor:
	// "…zaferi olarak gördü&nbsp;&nbsp;Son DakikaBM'de … Euronews.com…"
	// (ölçüm: 1088 içerik). Sınırı çözmeden ÖNCE kesiyoruz; sonra
	// `boslukToparla` bu izi siler ve gövde anlamsız başlık yığını olarak
	// kaydedilirdi.
	if ilk, kes := yapistirilmisBasliklar(ham); kes {
		ham = ilk
	}
	// Varlıkları çöz: "&nbsp;" / "&#39;" düz metin olarak saklanıyordu
	// (ölçüm: 2405 içerikte sayısal varlık).
	return varlikCoz(ham)
}

func parseRSSDate(s string) time.Time {
	for _, layout := range []string{
		time.RFC1123Z, time.RFC1123, time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 -0700", "2006-01-02T15:04:05Z07:00",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// CrawlFeed feed'i çeker, yeni öğeleri taslak içerik yapar. İçe aktarılan sayı döner.
func CrawlFeed(ctx context.Context, pool *pgxpool.Pool, f *Feed, rules []WordRule, defaultAuthor int64) (imported int, lastErr error) {
	items, err := FetchRSS(f.URL)
	if err != nil {
		// Panelin "son durum" sütunu bu satırlardan besleniyor; çekim
		// sonucunu yazmamak, ölü feed'i panelde "hiç denenmemiş" gösteriyordu.
		_ = markFeedCheck(ctx, pool, f.ID, "hata", err.Error(), 0)
		return 0, err
	}
	// Kategori çözümü çekim başına bir kez okunur: her öğe için ayrı sorgu
	// atmak gereksiz yük olurdu.
	slugIDs, _ := categorySlugIDs(ctx, pool)
	count := 0
	for _, it := range items {
		if it.Link == "" || it.Title == "" {
			continue
		}
		if it.PubDate.IsZero() {
			it.PubDate = time.Now()
		}
		h := sha1.Sum([]byte(it.Link))
		hash := hex.EncodeToString(h[:])
		var exists int
		if err := pool.QueryRow(ctx, `SELECT 1 FROM crawled_items WHERE url_hash = $1`, hash).Scan(&exists); err == nil {
			continue // zaten çekildi
		}
		// içerik oluştur (taslak)
		title := ApplyReplacer(strings.TrimSpace(it.Title), rules)
		// Başlık da varlık taşıyabilir ("Mardin&#039;in …").
		title = varlikCoz(title)
		spot := ApplyReplacer(it.Description, rules)
		// Körlemesine 240 rune kesmek cümleyi ortadan bölüyordu (ölçüm:
		// yayındaki 94 spot noktalamasız bitiyordu). Cümle sınırına saygılı.
		spot = cumleGuvenliKirp(spot, 240)

		// Yakın-kopya kapısı: url_hash farklı olsa da aynı haber başka
		// kaynaktan gelmiş olabilir (bkz. bot_dedupe.go). Mükerrer ise
		// kaydetmeyiz ama url'yi "çekildi" diye işaretleriz; aksi hâlde her
		// turda yeniden denenir.
		//
		// Kaynak hiyerarşisi: kayıtlı kopya bir agregatörden geliyorsa ve
		// gelen öğe orijinal kaynaktansa orijinal kazanır — agregatör kopyası
		// arşivlenir, yeni içerik normal akışta kaydedilir.
		dup, action, dupID := duplicateDraft(ctx, pool, siteIDOr1(f.SiteID), title, spot, f.Aggregator)
		if dup && action != DedupeReplace {
			if _, err := pool.Exec(ctx, `INSERT INTO crawled_items (feed_id, url, url_hash) VALUES ($1,$2,$3)`,
				f.ID, it.Link, hash); err != nil {
				lastErr = err
			}
			continue
		}
		if dup && action == DedupeReplace && dupID > 0 {
			if _, err := pool.Exec(ctx, `UPDATE contents SET status='arsiv' WHERE id=$1 AND status='taslak'`, dupID); err != nil {
				lastErr = err
			}
		}

		// Gövde: RSS özeti tek blok hâlinde kaydediliyordu (ölçüm: yayındaki
		// 982/986 içerik TEK paragraf). Paragraflara bölününce okunur oluyor;
		// metin uydurulmuyor, yalnız cümle sınırlarından ayrılıyor.
		detail := paragrafBloklari(ApplyReplacer(it.Description, rules))
		// Alaka puanı içerik oluşturulurken hesaplanır: kategori yayın kapısı
		// (categories.min_score) bu alana bakar. Eskiden yalnızca panelden
		// hesaplandığı için taslakların tamamı 0 puanla duruyordu (B4).
		score, _ := scoring.Relevance(scoring.Input{
			Title:     title,
			Spot:      spot,
			TagCount:  len(it.Categories),
			HasSource: true,
		})
		// Kategori: kaynak bölümü → editörün feed ataması → anahtar sözcük.
		// Ayrıntı ve gerekçe: internal/classify (B5).
		//
		// KRİPTO PORTALI (bitcosar): bölümler CoinDesk taksonomisidir
		// (markets/business/tech/policy). Eski Türkçe eşleme (gundem/ekonomi)
		// kullanılırsa 179 içeriğin tamamı tek bölüme düşüyordu — ölçüldü.
		catID := f.CategoryID
		slug := classify.CoinDeskSection(it.Categories, title, spot)
		if id, ok := slugIDs[slug]; ok && (f.CategoryID == nil || *f.CategoryID != id) {
			v := id
			catID = &v
		}
		c := &Content{
			TypeKey:       "haber",
			Status:        "taslak",
			Title:         title,
			Spot:          spot,
			Detail:        detail,
			Slug:          strPtr(platform.Slugify(title) + "-" + hash[:6]),
			CategoryID:    catID,
			AuthorID:      feedAuthor(f.AuthorID, defaultAuthor),
			SourceURL:     strPtr(it.Link),
			SourceName:    strPtr(f.Name),
			CommentStatus: "acik",
			Locale:        "tr",
			AiScore:       score,
			// Kaynağın kendi kapak görseli (mutlak http/https). Boşsa kart
			// görselsiz basılır; uydurma görsel konmaz.
			ImageURL: it.Image,
		}
		nid, err := CreateContent(ctx, pool, c)
		if err != nil {
			lastErr = err
			continue
		}
		// Kaynağın GERÇEK etiketleri (RSS <category>). Hata içeriği
		// engellemez: etiket yazılamazsa haber yine yayına girer.
		if len(it.Categories) > 0 {
			_ = baglaEtiketler(ctx, pool, nid, it.Categories)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO crawled_items (feed_id, url, url_hash) VALUES ($1,$2,$3)`,
			f.ID, it.Link, hash); err != nil {
			continue
		}
		count++
		_ = nid
	}
	status, errMsg := "ok", ""
	if lastErr != nil {
		status, errMsg = "kismi", lastErr.Error()
	}
	_ = markFeedCheck(ctx, pool, f.ID, status, errMsg, count)
	return count, lastErr
}

// markFeedCheck feed'in son çekim sonucunu yazar. Hata yutulur: bu bir
// gözlemlenebilirlik kaydıdır, içerik akışını durdurmamalıdır.
func markFeedCheck(ctx context.Context, pool *pgxpool.Pool, feedID int64, status, errMsg string, count int) error {
	_, err := pool.Exec(ctx, `
		UPDATE feeds SET last_checked_at = now(), last_status = $2, last_error = $3, last_item_count = $4
		WHERE id = $1`, feedID, status, errMsg, count)
	return err
}

func feedAuthor(authorOpt *int64, def int64) int64 {
	if authorOpt != nil && *authorOpt != 0 {
		return *authorOpt
	}
	return def
}

// categorySlug kategori id'sinin slug'ını döndürür (id yoksa boş).
func categorySlug(ctx context.Context, pool *pgxpool.Pool, id *int64) (string, error) {
	if id == nil || *id == 0 {
		return "", nil
	}
	var slug string
	err := pool.QueryRow(ctx, `SELECT slug FROM categories WHERE id = $1`, *id).Scan(&slug)
	if err != nil {
		return "", err
	}
	return slug, nil
}

// categorySlugIDs slug → id haritası (çekim başına bir kez okunur).
func categorySlugIDs(ctx context.Context, pool *pgxpool.Pool) (map[string]int64, error) {
	rows, err := pool.Query(ctx, `SELECT slug, id FROM categories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64, 96)
	for rows.Next() {
		var slug string
		var id int64
		if err := rows.Scan(&slug, &id); err != nil {
			return nil, err
		}
		out[slug] = id
	}
	return out, rows.Err()
}

func strPtr(s string) *string { return &s }

// textToBlocksStmt repo içi blok üretimi (httpapi util'daki ile aynı kural).
// paragrafBloklari metni okunur paragraflara bölüp blok dizisi üretir.
//
// `textToBlocksStmt` yalnız "\n" üzerinde bölüyordu; RSS özetlerinde satır
// sonu olmadığı için tek paragraf çıkıyordu.
func paragrafBloklari(s string) []byte {
	parcalar := paragraflaraBol(s)
	blocks := make([]map[string]any, 0, len(parcalar))
	for _, p := range parcalar {
		if p = strings.TrimSpace(p); p != "" {
			blocks = append(blocks, map[string]any{"type": "p", "text": p})
		}
	}
	b, _ := json.Marshal(blocks)
	return b
}

func textToBlocksStmt(s string) []byte {
	blocks := []map[string]any{}
	for _, para := range strings.Split(s, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		blocks = append(blocks, map[string]any{"type": "p", "text": para})
	}
	if blocks == nil {
		blocks = []map[string]any{}
	}
	b, _ := json.Marshal(blocks)
	return b
}
