package repo

import (
	"context"
	"errors"
	"io"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"theblock/site/internal/platform"
)

const (
	kaynakGovdeMaxBayt  = 2 << 20
	kaynakGovdeZamanAsi = 40 * time.Second
	// kaynakGovdeMaxParagraf: fazlası AI istemini şişirir, maliyeti artırır.
	kaynakGovdeMaxParagraf = 16
	// kaynakGovdeMaxRuna: tek makale için üst sınır (AI girdisi).
	kaynakGovdeMaxRuna = 9000
	// kaynakGovdeMinRuna: bundan kısa satır gövde paragrafı sayılmaz.
	kaynakGovdeMinRuna = 80
)

// KaynakGovde çekilen makale gövdesidir.
type KaynakGovde struct {
	Paragraflar []string
	// Gorsel: makalenin kendi kapak görseli (mutlak http/https) veya "".
	Gorsel string
}

var (
	// markdownBaglantiRE [metin](adres) → metin
	markdownBaglantiRE = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	// markdownVurguRE **kalın** ve _italik_ işaretleri
	markdownVurguRE = regexp.MustCompile(`\*\*|__`)
	// fiyatEtiketiRE: gövdeye gömülen canlı fiyat rozetleri ("BTC$76,125.31")
	fiyatEtiketiRE = regexp.MustCompile(`\b[A-Z]{2,10}\$[\d.,]+`)
	// markdownGorselRE ![alt](adres)
	markdownGorselRE = regexp.MustCompile(`^!\[[^\]]*\]\(([^)]+)\)`)
)

// govdeBitisIsaretleri gövdenin bittiği satırlar: kaynak sitenin kendi
// yasal notu, ilgili haber listesi ve çerez metni.
var govdeBitisIsaretleri = []string{
	"note: the views expressed",
	"read more:",
	"view full report",
	"when you visit any website",
	"this information might be about you",
	"sign up for free",
	"already have an account",
	"disclaimer",
}

func KaynakGovdeCek(ctx context.Context, makaleURL string) (KaynakGovde, error) {
	makaleURL = strings.TrimSpace(makaleURL)
	if makaleURL == "" {
		return KaynakGovde{}, errors.New("boş makale adresi")
	}
	u, err := neturl.Parse(makaleURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return KaynakGovde{}, errors.New("geçersiz makale adresi")
	}
	if err := platform.ValidateOutboundURL(u); err != nil {
		return KaynakGovde{}, err
	}

	ctx, iptal := context.WithTimeout(ctx, kaynakGovdeZamanAsi)
	defer iptal()

	okuURL := "https://r.jina.ai/" + makaleURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, okuURL, nil)
	if err != nil {
		return KaynakGovde{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; bitcosar-bot/1.0)")
	req.Header.Set("Accept", "text/plain")
	client := platform.SafeFetchClient(kaynakGovdeZamanAsi)
	resp, err := client.Do(req)
	if err != nil {
		return KaynakGovde{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return KaynakGovde{}, errors.New("okuyucu HTTP " + resp.Status)
	}
	ham, err := io.ReadAll(io.LimitReader(resp.Body, kaynakGovdeMaxBayt))
	if err != nil {
		return KaynakGovde{}, err
	}
	g := govdeAyikla(string(ham))
	if len(g.Paragraflar) == 0 {
		return KaynakGovde{}, errors.New("gövde paragrafı bulunamadı")
	}
	return g, nil
}

func govdeAyikla(ham string) KaynakGovde {
	var g KaynakGovde
	icerik := ham
	if i := strings.Index(ham, "Markdown Content:"); i >= 0 {
		icerik = ham[i+len("Markdown Content:"):]
	}

	basladi := false
	toplam := 0
	for _, satir := range strings.Split(icerik, "\n") {
		s := strings.TrimSpace(satir)
		if s == "" {
			continue
		}
		// Kapak görseli: gövde başlamadan önceki ilk görsel.
		if m := markdownGorselRE.FindStringSubmatch(s); m != nil && g.Gorsel == "" {
			if adres := strings.TrimSpace(m[1]); strings.HasPrefix(adres, "http") {
				g.Gorsel = adres
			}
			continue
		}
		kucuk := strings.ToLower(s)
		for _, isaret := range govdeBitisIsaretleri {
			if strings.Contains(kucuk, isaret) {
				return g
			}
		}
		// Gövde, künye/spot satırından sonra başlar. Okuyucu çıktısında bu
		// satır "## ..." biçiminde geliyor; ilk paragrafı ondan sonra al.
		if strings.HasPrefix(s, "## ") {
			basladi = true
			continue
		}
		if !basladi {
			continue
		}
		// Gezinme/liste/kod satırları: gövde değil.
		if strings.HasPrefix(s, "*") || strings.HasPrefix(s, "|") ||
			strings.HasPrefix(s, "#") || strings.HasPrefix(s, ">") ||
			strings.HasPrefix(s, "[") || strings.HasPrefix(s, "![") {
			continue
		}
		if siraliListeRE.MatchString(s) {
			continue
		}
		temiz := markdownTemizle(s)
		if len([]rune(temiz)) < kaynakGovdeMinRuna {
			continue
		}
		g.Paragraflar = append(g.Paragraflar, temiz)
		toplam += len([]rune(temiz))
		if len(g.Paragraflar) >= kaynakGovdeMaxParagraf || toplam >= kaynakGovdeMaxRuna {
			break
		}
	}
	return g
}

// siraliListeRE "1.   metin" biçimindeki ilgili haber listelerini yakalar.
var siraliListeRE = regexp.MustCompile(`^\d+\.\s`)

// markdownTemizle markdown süslerini ve gömülü fiyat rozetlerini temizler.
func markdownTemizle(s string) string {
	s = markdownGorselRE.ReplaceAllString(s, "")
	s = markdownBaglantiRE.ReplaceAllString(s, "$1")
	s = markdownVurguRE.ReplaceAllString(s, "")
	s = fiyatEtiketiRE.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "`", "")
	return strings.Join(strings.Fields(s), " ")
}

const kaynakGovdeTurLimiti = 8

func govdeGerekli(kind string) bool {
	return strings.EqualFold(strings.TrimSpace(kind), "bing")
}

func feedOku(ctx context.Context, f *Feed) ([]RSSItem, error) {
	if strings.EqualFold(strings.TrimSpace(f.Kind), "bing") {
		return FetchBingLinkFeed(ctx, f.URL, siteHostSorgudan(f.URL))
	}
	return FetchRSS(f.URL)
}

// siteHostSorgudan Bing sorgusundaki `site:ornek.com/yol` alan adını çıkarır.
// Bulunamazsa boş döner: o durumda alan adı süzgeci uygulanmaz.
func siteHostSorgudan(raw string) string {
	u, err := neturl.Parse(raw)
	if err != nil {
		return ""
	}
	q := u.Query().Get("q")
	i := strings.Index(strings.ToLower(q), "site:")
	if i < 0 {
		return ""
	}
	alan := strings.Fields(q[i+len("site:"):])
	if len(alan) == 0 {
		return ""
	}
	host := alan[0]
	if j := strings.IndexByte(host, '/'); j >= 0 {
		host = host[:j]
	}
	return strings.ToLower(strings.TrimSpace(host))
}
