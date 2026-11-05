package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

// ---- CSP: satır içi script'ler için HASH tabanlı izin listesi ----
//
// NEDEN HASH, NEDEN NONCE DEĞİL:
//
// İlk uygulamada istek başına rastgele nonce kullanılmıştı. Ancak herkese açık
// sayfalar `Cache-Control: public, max-age=60, s-maxage=300` ile CDN'de
// önbellekleniyor. Önbelleğe alınmış yanıt tüm kullanıcılara AYNI nonce ile
// servis edilir; bu da nonce'un tekilliğini bozar (CSP'de nonce yanıt başına
// tekil olmalıdır). Saldırgan kendi aldığı yanıttan nonce'u öğrenip başka
// kullanıcılara servis edilecek yanıt için kullanabilir.
//
// Hash yaklaşımı bu sorunu kökten çözer:
//   - HTML bayt-aynı kalır (nonce eklenmez) → önbellek güvenli,
//   - politika yalnız o yanıtta GERÇEKTEN bulunan satır içi script'lerin
//     sha256 özetlerini listeler.
//
// NEDEN MIDDLEWARE DÜZEYİNDE:
//
// Önce her render fonksiyonu (s.render, s.renderPublic, renderPartial, AMP,
// akış) kendi çıktısından hash üretip başlığı yazıyordu. Uygulamada bunun
// kırılgan olduğu görüldü: aynı yanıtı birden fazla yol yazabildiğinde
// (templ panel + sayfa kabuğu) ve satır içi script içeriği istek verisine
// göre değiştiğinde başlık ile gövde ayrışabiliyordu — sonuç: tarayıcıda
// "Executing inline script violates..." hataları.
//
// withHTMLCSP tüm HTML yanıtını tamponlar ve hash'leri YAZILAN NİHAİ
// baytlardan üretir; başlık ile gövde tanım gereği tutarlıdır.

// cspBufferLimit tamponlanacak en büyük HTML yanıtı. Aşılırsa yanıt
// tamponlanmadan akıtılır (bellek koruması); CSP o durumda hash'siz kalır.
const cspBufferLimit = 8 << 20 // 8 MB

// withHTMLCSP HTML yanıtlarını tamponlayıp CSP'yi nihai gövdeden üretir.
func (s *Server) withHTMLCSP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bw := &cspWriter{w: w, raporURL: s.cspRaporURL()}
		next.ServeHTTP(bw, r)
		bw.finish()
	})
}

type cspWriter struct {
	w        http.ResponseWriter
	raporURL string
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

// durumIlet durum kodunu alttaki yazıcıya bir kez geçirir.
// Bu olmadan tamponlama sırasında bildirilen 204/302/404 gibi kodlar
// kaybolur ve her yanıt 200 döner.
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

// Flush tamponlamayı bırakıp alttaki yazıcıya devreder (SSE, uzun yoklama).
func (c *cspWriter) Flush() {
	if !c.akis {
		_ = c.bosalt()
		c.akis = true
	}
	if f, ok := c.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (c *cspWriter) Unwrap() http.ResponseWriter { return c.w }

// bosalt biriken tamponu alttaki yazıcıya yazar.
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
		c.w.Header().Set("Content-Security-Policy",
			buildCSP(cspScriptHashes(govde), cspStyleHashes(govde), collabURL(), c.raporURL))
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

// vendorStyleHashes: çalışma anında JS'in enjekte ettiği, İÇERİĞİ SABİT
// stil bloklarının hash'leri.
// TipTap/ProseMirror editörü kendi temel yerleşim CSS'ini
// `<style data-tiptap-style>` olarak sayfaya ekler (paket içinde sabit metin).
// Bu blok sunucu HTML'inde olmadığı için cspStyleHashes onu göremez; hash'i
// burada sabit olarak verilir. İçerik değişirse (paket yükseltmesi) hash
// tutmaz ve tarayıcı bloğu engeller — `e2e/tests/csp_style.test.js` bunu
// yakalar. Aynı kurallar `static/prosemirror.css` içinde de durduğu için
// engellense bile editör yerleşimi bozulmaz.
//
// Yeni bir sabit enjeksiyon eklenirse: hash'i buraya ekle.
var vendorStyleHashes = []string{
	"'sha256-PlumsSlvJ7vvWzjqibGAYKq92O3y/4JTxWWsWJvyUYA='", // prosemirror-view temel CSS
}

// cspStyleHashes HTML'deki satır içi <style> bloklarının sha256 özetlerini
// döndürür (CSP biçiminde: "'sha256-<base64>'").
//
// Neden: sayfa başlığındaki `<style>{{.SiteBg}}</style>`, `.DesignCSS`,
// `.AdminCSS` ve şablon içi stil blokları **veritabanından gelen** değerlerle
// kurulur. CSP'de `style-src 'unsafe-inline'` varken, HTML enjeksiyonu
// yapabilen bir saldırgan `<style>` bloğuyla CSS sızdırma (ör.
// `input[value^="a"]{background:url(//saldirgan)`) yapabilirdi.
// Hash ile yalnız sunucunun ürettiği baytlar izinli olur; enjekte edilen
// blok tarayıcıda çalışmaz.
//
// `style="..."` ÖZNİTELİKLERİ ayrı direktifle (`style-src-attr`) yönetilir;
// oradaki 'unsafe-inline' bilinçli olarak duruyor (aşağıdaki buildCSP yorumu).
func cspStyleHashes(html string) []string {
	if indexFold(html, "<style") < 0 {
		return nil
	}
	gorulen := map[string]bool{}
	var out []string

	rest := html
	for {
		i := indexFold(rest, "<style")
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

// cspScriptHashes HTML'deki satır içi <script> bloklarının sha256 özetlerini
// döndürür (CSP biçiminde: "sha256-<base64>").
//
// Atlananlar:
//   - src= taşıyan (harici) script'ler — 'self' zaten izinli,
//   - veri blokları (application/json, ld+json, text/template, text/plain) —
//     yürütülmezler, CSP bunlara uygulanmaz.
func cspScriptHashes(html string) []string {
	if !strings.Contains(html, "<script") {
		return nil
	}
	gorulen := map[string]bool{}
	var out []string

	rest := html
	for {
		i := indexFold(rest, "<script")
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

// indexFold büyük/küçük harf duyarsız strings.Index — ASCII güvenli.
//
// DİKKAT: `strings.ToLower` ile arama YAPILMAZ. Türkçe metinde ToLower bayt
// uzunluğunu değiştirebilir (ör. 'İ' U+0130 → "i̇" iki kod noktası), bu da
// küçük harfli metinden alınan indeksin orijinal metinde kaymasına yol açar.
// Bu hata, satır içi script hash'lerinin gövdeyle uyuşmaması olarak ortaya
// çıktı (tarayıcıda "Executing inline script violates...").
// Burada karşılaştırma bayt bayt ve boyut koruyucu yapılır.
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

// buildCSP zorunlu politikayı üretir.
//
// script-src: 'self' + sayfadaki satır içi script'lerin hash'leri + AMP/GTM.
// 'unsafe-inline' ve 'unsafe-eval' YOK.
//
// style-src-elem: 'self' + sayfadaki satır içi <style> bloklarının hash'leri.
// 'unsafe-inline' YOK → enjekte edilen <style> çalışmaz (CSS ile veri
// sızdırma kapalı).
//
// style-src-attr'de 'unsafe-inline' BİLİNÇLİ olarak duruyor: sayfalarda
// yüzlerce `style="..."` özniteliği var (≈1.500 sabit + ≈70 dinamik) ve
// öznitelikler hash ile izinlenemez — CSP'de öznitelik hash'i yoktur, yalnız
// 'unsafe-hashes' vardır ve o da 'unsafe-inline'a çok yakındır. Stil tabanlı
// XSS riski script'e göre çok düşük olduğundan bilinçli erteleme; kapatmak
// özniteliklerin sınıfa taşınmasını gerektirir (ayrı iş).
func buildCSP(scriptHashes, styleHashes []string, collabURL, reportURL string) string {
	scriptSrc := "script-src 'self'"
	for _, h := range scriptHashes {
		scriptSrc += " " + h
	}
	// AMP sayfaları kendi çalışma zamanını cdn.ampproject.org'dan yükler.
	scriptSrc += " https://www.googletagmanager.com https://cdn.ampproject.org"
	// Cloudflare Web Analytics beacon'ı tünelin ucunda sayfaya enjekte edilir;
	// kaynak listesinde olmadığı için her istekte engelleniyordu. Üretim CSP
	// raporlarında sürekli görünen ihlal buydu:
	//   script-src-elem blocked=https://static.cloudflareinsights.com
	// Beacon ölçümü cloudflareinsights.com/cdn-cgi/rum adresine POST eder.
	scriptSrc += " https://static.cloudflareinsights.com"

	styleSrc := "style-src 'self'"
	for _, h := range styleHashes {
		styleSrc += " " + h
	}
	for _, h := range vendorStyleHashes {
		styleSrc += " " + h
	}
	styleSrc += " https://fonts.googleapis.com"

	connectSrc := "connect-src 'self' https://www.google-analytics.com https://region1.google-analytics.com https://cloudflareinsights.com"
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
		"https://www.google.com https://www.googletagmanager.com https://cdn.ampproject.org https://search.google.com; " +
		"frame-ancestors 'self'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"object-src 'none'" + reportDirektifi(reportURL)
}

// reportDirektifi CSP raporlama direktifini üretir.
// report-uri: CSP3'te "deprecated" ama tüm tarayıcılarda çalışır.
// report-to: yeni biçim; grup adı "csp" ve Report-To başlığıyla eşleşir.
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
