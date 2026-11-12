package youtube

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Oge tek bir YouTube içeriği.
type Oge struct {
	VideoID  string
	Tur      string // "video" | "short" | "stream"
	Baslik   string
	Aciklama string
	Kanal    string
	Gorsel   string
	URL      string
	SureSn   int
	Izlenme  int64
	// YayinAt sıfır olabilir (canlı yayınlarda tarih yok).
	YayinAt time.Time
}

// Sonuc tek bir çekme işleminin çıktısı.
type Sonuc struct {
	Videolar []Oge
	Shorts   []Oge
	Yayinlar []Oge
	// CanliVideoID şu an canlı yayında olan video (yoksa boş).
	CanliVideoID string
}

var (
	kanalIDKalibi = regexp.MustCompile(`^UC[A-Za-z0-9_-]{20,26}$`)
	handleKalibi  = regexp.MustCompile(`^[A-Za-z0-9._-]{2,64}$`)
)

// GecerliKanalID kanal kimliğinin biçimini doğrular.
func GecerliKanalID(s string) bool { return kanalIDKalibi.MatchString(strings.TrimSpace(s)) }

// GecerliHandle kanal handle'ının biçimini doğrular (@ işareti olmadan).
func GecerliHandle(s string) bool { return handleKalibi.MatchString(strings.TrimSpace(s)) }

// Client çekme istemcisi. HTTP alanı testlerde değiştirilebilir.
type Client struct {
	HTTP    *http.Client
	KanalID string
	Handle  string
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (c *Client) Cek(ctx context.Context) (Sonuc, error) {
	var s Sonuc
	var errler []error

	if GecerliKanalID(c.KanalID) {
		rssURL := "https://www.youtube.com/feeds/videos.xml?channel_id=" + url.QueryEscape(c.KanalID)
		html, err := c.get(ctx, rssURL)
		if err != nil {
			errler = append(errler, fmt.Errorf("rss: %w", err))
		} else {
			oge, err := RSSOku(html)
			if err != nil {
				errler = append(errler, fmt.Errorf("rss ayrıştırma: %w", err))
			}
			for _, o := range oge {
				if o.Tur == "short" {
					s.Shorts = append(s.Shorts, o)
				} else {
					s.Videolar = append(s.Videolar, o)
				}
			}
		}
	} else {
		errler = append(errler, errors.New("kanal kimliği geçersiz"))
	}

	if GecerliHandle(c.Handle) {
		base := "https://www.youtube.com/@" + c.Handle
		html, err := c.get(ctx, base+"/streams")
		if err != nil {
			errler = append(errler, fmt.Errorf("streams: %w", err))
		} else {
			kayitlar := YayinlarOku(html)
			for _, k := range kayitlar {
				s.Yayinlar = append(s.Yayinlar, Oge{
					VideoID: k.VideoID,
					Tur:     "stream",
					Baslik:  k.Baslik,
					Kanal:   c.Handle,
					Gorsel:  GorselURL(k.VideoID, "stream"),
					URL:     "https://www.youtube.com/watch?v=" + k.VideoID,
				})
			}
		}
		liveHTML, err := c.get(ctx, base+"/live")
		if err != nil {
			errler = append(errler, fmt.Errorf("live: %w", err))
		} else {
			s.CanliVideoID = CanliVideoID(liveHTML)
		}
	} else {
		errler = append(errler, errors.New("kanal handle geçersiz"))
	}

	if len(s.Videolar) == 0 && len(s.Shorts) == 0 && len(s.Yayinlar) == 0 {
		return s, errors.Join(errler...)
	}
	return s, nil
}

func (c *Client) get(ctx context.Context, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	// YouTube tr-TR içerik ve Türkçe başlık döndürsün.
	req.Header.Set("Accept-Language", "tr-TR,tr;q=0.9,en;q=0.5")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36")
	resp, err := c.http().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	// Sayfalar büyük (≈1,4 MB); sınır koyuyoruz ki bozuk yanıt belleği şişirmesin.
	b, err := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
	return string(b), err
}

// ---- RSS ----

type rssFeed struct {
	Title   string `xml:"title"`
	Entries []struct {
		VideoID   string `xml:"videoId"`
		Title     string `xml:"title"`
		Published string `xml:"published"`
		Link      struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Group struct {
			Description string `xml:"description"`
			Thumbnail   struct {
				URL string `xml:"url,attr"`
			} `xml:"thumbnail"`
			Community struct {
				Statistics struct {
					Views int64 `xml:"views,attr"`
				} `xml:"statistics"`
			} `xml:"community"`
			Duration struct {
				Seconds int `xml:"seconds,attr"`
			} `xml:"duration"`
		} `xml:"group"`
	} `xml:"entry"`
}

// RSSOku kanal RSS'ini ayrıştırır; türü bağlantı yolundan belirler.
func RSSOku(xmlMetin string) ([]Oge, error) {
	var f rssFeed
	if err := xml.Unmarshal([]byte(xmlMetin), &f); err != nil {
		return nil, err
	}
	out := make([]Oge, 0, len(f.Entries))
	for _, e := range f.Entries {
		id := strings.TrimSpace(e.VideoID)
		if len(id) != 11 {
			continue
		}
		tur := "video"
		if strings.Contains(e.Link.Href, "/shorts/") {
			tur = "short"
		}
		var yayin time.Time
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(e.Published)); err == nil {
			yayin = t
		}
		gorsel := e.Group.Thumbnail.URL
		if gorsel == "" {
			gorsel = GorselURL(id, tur)
		}
		out = append(out, Oge{
			VideoID:  id,
			Tur:      tur,
			Baslik:   strings.TrimSpace(e.Title),
			Aciklama: strings.TrimSpace(e.Group.Description),
			Kanal:    strings.TrimSpace(f.Title),
			Gorsel:   gorsel,
			URL:      "https://www.youtube.com/watch?v=" + id,
			SureSn:   e.Group.Duration.Seconds,
			Izlenme:  e.Group.Community.Statistics.Views,
			YayinAt:  yayin,
		})
	}
	return out, nil
}

// ---- /streams sayfası ----

// YayinKaydi /streams sayfasından çıkarılan tek yayın.
type YayinKaydi struct {
	VideoID string
	Baslik  string
}

func YayinlarOku(html string) []YayinKaydi {
	var out []YayinKaydi
	gorulen := map[string]bool{}
	for _, m := range kilitler(html) {
		id, _ := m["contentId"].(string)
		if len(id) != 11 || gorulen[id] {
			continue
		}
		baslik := metinYolu(m, "metadata", "lockupMetadataViewModel", "title", "content")
		if baslik == "" {
			baslik = metinYolu(m, "metadata", "lockupMetadataViewModel", "title", "simpleText")
		}
		if baslik == "" {
			continue
		}
		gorulen[id] = true
		out = append(out, YayinKaydi{VideoID: id, Baslik: baslik})
		if len(out) >= 12 {
			break
		}
	}
	return out
}

// kilitler ytInitialData içindeki `lockupViewModel` nesnelerini toplar.
func kilitler(html string) []map[string]any {
	veri := initialData(html)
	if veri == nil {
		return nil
	}
	var out []map[string]any
	var yuru func(any)
	yuru = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if _, ok := t["contentId"].(string); ok {
				if _, varMi := t["metadata"]; varMi {
					out = append(out, t)
				}
			}
			for _, alt := range t {
				yuru(alt)
			}
		case []any:
			for _, alt := range t {
				yuru(alt)
			}
		}
	}
	yuru(veri)
	return out
}

// initialData sayfadaki `var ytInitialData = {…};` JSON'unu çözer.
func initialData(html string) any {
	const isaret = "ytInitialData = "
	i := strings.Index(html, isaret)
	if i < 0 {
		return nil
	}
	bas := i + len(isaret)
	son := jsonSonu(html, bas)
	if son <= bas {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(html[bas:son]), &v); err != nil {
		return nil
	}
	return v
}

// jsonSonu metin içinde bas'tan başlayan JSON nesnesinin bitiş indeksini
// bulur (dizge ve kaçış karakterlerini atlayarak süslü parantez sayar).
func jsonSonu(s string, bas int) int {
	derinlik := 0
	dizgede := false
	kacis := false
	for i := bas; i < len(s); i++ {
		c := s[i]
		if dizgede {
			switch {
			case kacis:
				kacis = false
			case c == '\\':
				kacis = true
			case c == '"':
				dizgede = false
			}
			continue
		}
		switch c {
		case '"':
			dizgede = true
		case '{':
			derinlik++
		case '}':
			derinlik--
			if derinlik == 0 {
				return i + 1
			}
		}
	}
	return -1
}

// metinYolu iç içe map'lerde verilen yolu izleyip son metni döndürür.
func metinYolu(v any, yol ...string) string {
	simdi := v
	for _, k := range yol {
		m, ok := simdi.(map[string]any)
		if !ok {
			return ""
		}
		simdi = m[k]
	}
	s, _ := simdi.(string)
	return strings.TrimSpace(s)
}

// ---- /live sayfası ----

var kanonikKalibi = regexp.MustCompile(`<link rel="canonical" href="([^"]+)"`)

func CanliVideoID(html string) string {
	if !strings.Contains(html, `"isLive":true`) {
		return ""
	}
	m := kanonikKalibi.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	u, err := url.Parse(m[1])
	if err != nil || u.Host != "www.youtube.com" {
		return ""
	}
	if u.Path == "/watch" {
		if id := u.Query().Get("v"); len(id) == 11 {
			return id
		}
	}
	// /live/<id> biçimi
	if strings.HasPrefix(u.Path, "/live/") && len(strings.TrimPrefix(u.Path, "/live/")) == 11 {
		return strings.TrimPrefix(u.Path, "/live/")
	}
	return ""
}

// ---- yardımcılar ----

func GorselURL(id, tur string) string {
	if id == "" {
		return ""
	}
	if tur == "short" {
		return "https://i.ytimg.com/vi/" + id + "/oar2.jpg"
	}
	return "https://i.ytimg.com/vi/" + id + "/maxresdefault.jpg"
}

func GorselYedekZincir(id, tur string) string {
	// Boş kimlik: adres ÜRETİLMEZ. Aksi hâlde "…/vi//hqdefault.jpg" gibi
	// bozuk bir adres basılıyordu (test yakaladı, 11 Kasım 2026).
	if id == "" {
		return ""
	}
	taban := "https://i.ytimg.com/vi/" + id + "/"
	if tur == "short" {
		return taban + "maxresdefault.jpg," + taban + "hqdefault.jpg"
	}
	return taban + "sddefault.jpg," + taban + "hqdefault.jpg"
}

func GorselSrcset(id, tur string) string {
	if id == "" || tur == "short" {
		return ""
	}
	return "https://i.ytimg.com/vi/" + id + "/hqdefault.jpg 480w, " +
		"https://i.ytimg.com/vi/" + id + "/sddefault.jpg 640w, " +
		"https://i.ytimg.com/vi/" + id + "/maxresdefault.jpg 1280w"
}

// BuyukGorselURL video sayfası kahraman görseli için yüksek çözünürlük.
func BuyukGorselURL(id string) string {
	if id == "" {
		return ""
	}
	return "https://i.ytimg.com/vi/" + id + "/maxresdefault.jpg"
}

// Sure yaz "1:02:03" / "12:34" biçiminde süre metni üretir.
func Sure(sn int) string {
	if sn <= 0 {
		return ""
	}
	s := sn % 60
	d := (sn / 60) % 60
	sa := sn / 3600
	if sa > 0 {
		return strconv.Itoa(sa) + ":" + iki(d) + ":" + iki(s)
	}
	return strconv.Itoa(d) + ":" + iki(s)
}

func iki(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// OnizlemeURL kart üstüne gelince oynatılan KISA hareketli önizleme.
//
// YouTube'un kendi arayüzünde hover'da gösterdiği dosya: 6 saniyelik,
// sessiz, animasyonlu WebP. Neden bunu seçtik:
//
//   · <video> gerekmiyor — tek <img> kaynağı değişiyor, DOM'a eleman
//     eklenmiyor, ses/otomatik oynatma izni sorunu yok.
//   · Dosya küçük (~100-300 KB) ve TEK istek; gerçek videoda parça
//     parça indirme + çözücü maliyeti olurdu.
//   · İstek YALNIZ hover'da yapılır; liste sayfasının ilk yüklemesine
//     hiçbir ağırlık eklemez.
//
// Uç nokta YouTube'un belgelenmiş API'si DEĞİL. Bu yüzden çağıran taraf
// yüklenememeyi sessizce karşılamalı (statik kapakta kalır) — kart
// bozulmaz, yalnız hareket olmaz.
func OnizlemeURL(id string) string {
	if id == "" {
		return ""
	}
	return "https://i.ytimg.com/an_webp/" + id + "/mqdefault_6s.webp"
}
