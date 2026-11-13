package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

// cspBufferLimit tamponlanacak en büyük HTML yanıtı. Aşılırsa yanıt
// tamponlanmadan akıtılır (bellek koruması); CSP o durumda hash'siz kalır.
const cspBufferLimit = 8 << 20 // 8 MB

// withHTMLCSP HTML yanıtlarını tamponlayıp CSP'yi nihai gövdeden üretir.
func (s *Server) withHTMLCSP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bw := &cspWriter{w: w, raporURL: s.cspRaporURL(), hashes: s.cspHashes}
		next.ServeHTTP(bw, r)
		bw.finish()
	})
}

type cspWriter struct {
	w        http.ResponseWriter
	raporURL string
	hashes   *cspHashCache
	buf      strings.Builder
	status   int  // handler'ın bildirdiği durum kodu
	hdrYazil bool // WriteHeader çağrıldı mı
	iletildi bool // durum kodu alttaki yazıcıya geçti mi
	akis     bool // tamponlama bırakıldı (SSE / büyük yanıt)
}

func (c *cspWriter) Header() http.Header { return c.w.Header() }

func (c *cspWriter) WriteHeader(code int) {
	if c.hdrYazil {
		return
	}
	c.hdrYazil = true
	c.status = code
	// Akış yanıtları (SSE) tamponlanamaz: hemen geçir.
	if strings.HasPrefix(c.w.Header().Get("Content-Type"), "text/event-stream") {
		c.akis = true
		c.durumIlet(code)
	}
}

func (c *cspWriter) durumIlet(code int) {
	if c.iletildi {
		return
	}
	c.iletildi = true
	c.w.WriteHeader(code)
}

// durumKesin handler hiç WriteHeader çağırmadıysa 200 kabul et.
func (c *cspWriter) durumKesin() int {
	if c.hdrYazil && c.status != 0 {
		return c.status
	}
	return http.StatusOK
}

func (c *cspWriter) Write(p []byte) (int, error) {
	if c.akis {
		return c.w.Write(p)
	}
	if !c.hdrYazil {
		// Varsayılan durum 200; Content-Type henüz belli değil.
		c.WriteHeader(http.StatusOK)
		if c.akis {
			return c.w.Write(p)
		}
	}
	if c.buf.Len()+len(p) > cspBufferLimit {
		// Çok büyük yanıt: tamponu bırak, olduğu gibi akıt.
		if err := c.bosalt(); err != nil {
			return 0, err
		}
		c.akis = true
		return c.w.Write(p)
	}
	return c.buf.Write(p)
}

func (c *cspWriter) Flush() {
	ct := c.w.Header().Get("Content-Type")
	htmlMi := ct == "" || strings.HasPrefix(ct, "text/html")
	if !c.akis && htmlMi {
		// HTML: gövde CSP hash'leri için tamponda kalmalı.
		return
	}
	if !c.akis {
		_ = c.bosalt()
		c.akis = true
	}
	if f, ok := c.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (c *cspWriter) Unwrap() http.ResponseWriter { return c.w }

func (c *cspWriter) cspBasligi(govde string) string {
	if c.hashes != nil {
		if v, ok := c.hashes.Get(govde); ok {
			return v
		}
	}
	v := buildCSP(cspScriptHashes(govde), cspStyleHashes(govde), collabURL(), c.raporURL)
	if c.hashes != nil {
		c.hashes.Set(govde, v)
	}
	return v
}

func (c *cspWriter) bosalt() error {
	c.durumIlet(c.durumKesin())
	if c.buf.Len() == 0 {
		return nil
	}
	_, err := c.w.Write([]byte(c.buf.String()))
	c.buf.Reset()
	return err
}

// finish yanıt bittiğinde CSP'yi nihai gövdeden üretip tamponu boşaltır.
func (c *cspWriter) finish() {
	if c.akis {
		return
	}
	govde := c.buf.String()
	if strings.HasPrefix(c.w.Header().Get("Content-Type"), "text/html") ||
		(c.w.Header().Get("Content-Type") == "" && strings.Contains(strings.ToLower(govde[:min(400, len(govde))]), "<html")) {
		c.w.Header().Set("Content-Security-Policy", c.cspBasligi(govde))
	}
	if c.buf.Len() == 0 {
		// Gövdesiz yanıt (204, 304, boş 302): durum kodu yine iletilmeli.
		c.durumIlet(c.durumKesin())
		return
	}
	_ = c.bosalt()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var vendorStyleHashes = []string{
	"'sha256-PlumsSlvJ7vvWzjqibGAYKq92O3y/4JTxWWsWJvyUYA='", // prosemirror-view temel CSS
}

func cspStyleHashes(html string) []string {
	if indexFold(html, "<style") < 0 {
		return nil
	}
	gorulen := map[string]bool{}
	var out []string

	rest := html
	for {
		// Yorum içindeki "<style" etiket değildir (bkz. htmlYorumAtla).
		yeni, i := htmlYorumAtla(rest, "<style")
		rest = yeni
		if i < 0 {
			break
		}
		rest = rest[i:]
		gt := strings.IndexByte(rest, '>')
		if gt < 0 {
			break
		}
		govde := rest[gt+1:]

		k := indexFold(govde, "</style")
		if k < 0 {
			break
		}
		icerik := govde[:k]
		rest = govde[k:]

		// Tarayıcı stil verisinde satır sonlarını LF'e indirger.
		normal := strings.ReplaceAll(strings.ReplaceAll(icerik, "\r\n", "\n"), "\r", "\n")
		if strings.TrimSpace(normal) == "" {
			continue
		}
		sum := sha256.Sum256([]byte(normal))
		h := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
		if !gorulen[h] {
			gorulen[h] = true
			out = append(out, h)
		}
	}
	return out
}

func cspScriptHashes(html string) []string {
	if indexFold(html, "<script") < 0 {
		return nil
	}
	gorulen := map[string]bool{}
	var out []string

	rest := html
	for {
		// HTML YORUMU içindeki "<script" etiket DEĞİLDİR (bkz. htmlYorumAtla).
		yeni, i := htmlYorumAtla(rest, "<script")
		rest = yeni
		if i < 0 {
			break
		}
		rest = rest[i:]
		gt := strings.IndexByte(rest, '>')
		if gt < 0 {
			break
		}
		tag := rest[:gt]
		govde := rest[gt+1:]

		low := strings.ToLower(tag)
		if strings.Contains(low, "src=") || veriBloguMu(low) {
			rest = govde
			continue
		}

		k := indexFold(govde, "</script")
		if k < 0 {
			break
		}
		icerik := govde[:k]
		rest = govde[k:]

		// Tarayıcı script verisinde satır sonlarını LF'e indirger (CRLF/CR → LF).
		normal := strings.ReplaceAll(strings.ReplaceAll(icerik, "\r\n", "\n"), "\r", "\n")
		if strings.TrimSpace(normal) == "" {
			continue
		}
		sum := sha256.Sum256([]byte(normal))
		h := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
		if !gorulen[h] {
			gorulen[h] = true
			out = append(out, h)
		}
	}
	return out
}

// veriBloguMu yürütülmeyen script türlerini tanır.
func veriBloguMu(lowTag string) bool {
	for _, t := range []string{
		"application/json", "application/ld+json",
		"text/template", "text/plain", "text/x-template",
	} {
		if strings.Contains(lowTag, `type="`+t+`"`) || strings.Contains(lowTag, `type='`+t+`'`) ||
			strings.Contains(lowTag, "type="+t) {
			return true
		}
	}
	return false
}

func indexFold(s, alt string) int {
	n, m := len(s), len(alt)
	if m == 0 {
		return 0
	}
	if m > n {
		return -1
	}
	for i := 0; i+m <= n; i++ {
		eslesti := true
		for j := 0; j < m; j++ {
			a := s[i+j]
			b := alt[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				eslesti = false
				break
			}
		}
		if eslesti {
			return i
		}
	}
	return -1
}

func buildCSP(scriptHashes, styleHashes []string, collabURL, reportURL string) string {
	scriptSrc := "script-src 'self'"
	for _, h := range scriptHashes {
		scriptSrc += " " + h
	}
	// AMP sayfaları kendi çalışma zamanını cdn.ampproject.org'dan yükler.
	scriptSrc += " https://www.googletagmanager.com https://cdn.ampproject.org"
	scriptSrc += " https://static.cloudflareinsights.com"

	// REKLAM AĞLARI (13 Kasım 2026): bu alan adları eksik olduğu için
	// `/static/cd-entegrasyon.js` tarafından eklenen betikler tarayıcıda
	// CSP ile BLOKLANIYORDU ve reklamlar hiç yüklenmiyordu (canlı konsol
	// ölçümü: coinzillatag.com/lib/display.js, /lib/sticky.js ve
	// appsha1.cointraffic.io/js "violates CSP directive" hatası).
	// Kapsam DAR tutuldu: yalnız bu üç ağ + AdSense'in zorunlu uçları.
	//   · Coinzilla : coinzillatag.com (betikler) +
	//                  request-global.czilladx.com (ASIL REKLAM İÇERİĞİ —
	//                  ölçüldü 13 Kasım 2026: betikler yüklendiği hâlde
	//                  reklam gelmiyordu çünkü asıl istek bu alan adına
	//                  gidiyor ve connect-src'de yoktu)
	//   · Cointraffic: appsha1.cointraffic.io (betik) +
	//                  apps.adsgravity.io (izleyici/ölçüm uç noktası)
	//   · AdSense    : pagead2/adservice/tpc/googletagservices,
	//                  googleads.g.doubleclick.net, securepubads
	scriptSrc += " https://coinzillatag.com https://appsha1.cointraffic.io" +
		" https://pagead2.googlesyndication.com https://adservice.google.com" +
		" https://tpc.googlesyndication.com https://www.googletagservices.com" +
		" https://securepubads.g.doubleclick.net https://fundingchoicesmessages.google.com"

	styleSrc := "style-src 'self'"
	for _, h := range styleHashes {
		styleSrc += " " + h
	}
	for _, h := range vendorStyleHashes {
		styleSrc += " " + h
	}
	styleSrc += " https://fonts.googleapis.com"

	connectSrc := "connect-src 'self' https://www.google-analytics.com https://region1.google-analytics.com https://cloudflareinsights.com" +
		// Reklam ağlarının XHR/beacon uçları (ölçüm, kapatma, kalite).
		// request-global.czilladx.com REKLAMIN KENDİSİNİ serve eder —
		// eksik olduğu için reklam hiç gelmiyordu (canlı konsol, 13 Kasım).
		" https://coinzillatag.com https://request-global.czilladx.com" +
		" https://appsha1.cointraffic.io https://apps.adsgravity.io" +
		" https://pagead2.googlesyndication.com https://googleads.g.doubleclick.net" +
		" https://ep1.adtrafficquality.google https://ep2.adtrafficquality.google" +
		" https://securepubads.g.doubleclick.net"
	if o := websocketOrigin(collabURL); o != "" {
		connectSrc += " " + o
	}

	return "default-src 'self'; " +
		scriptSrc + "; " +
		styleSrc + "; " +
		"style-src-attr 'unsafe-inline'; " +
		"img-src 'self' data: https:; " +
		"font-src 'self' https://fonts.gstatic.com; " +
		connectSrc + "; " +
		"frame-src 'self' https://www.youtube.com https://www.youtube-nocookie.com " +
		"https://www.google.com https://www.googletagmanager.com https://cdn.ampproject.org https://search.google.com " +
		// Reklam iframe'leri (AdSense + Coinzilla)
		"https://googleads.g.doubleclick.net https://tpc.googlesyndication.com " +
		"https://securepubads.g.doubleclick.net https://coinzillatag.com " +
		// Cointraffic izleyicisi KENDI iframe'ini de basiyor (olcum 13 Kasim
		// 2026, canli konsol): "Framing https://apps.adsgravity.io/ violates
		// frame-src". Alan adi connect-src'de vardi, frame-src'de yoktu —
		// reklami engellemiyordu ama her sayfa yuklemesinde ihlal raporu
		// uretiyor ve olcum verisi kaybediyordu.
		"https://request-global.czilladx.com https://apps.adsgravity.io; " +
		"frame-ancestors 'self'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"object-src 'none'" + reportDirektifi(reportURL)
}

func reportDirektifi(reportURL string) string {
	if reportURL == "" {
		return ""
	}
	return "; report-uri " + reportURL + "; report-to csp"
}

// websocketOrigin env'deki collab adresini CSP connect-src girdisine çevirir.
// http(s) verilirse ws(s) karşılığına da izin verilir.
func websocketOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return "ws://" + u.Host
	case "https":
		return "wss://" + u.Host
	case "ws", "wss":
		return strings.ToLower(u.Scheme) + "://" + u.Host
	}
	return ""
}
