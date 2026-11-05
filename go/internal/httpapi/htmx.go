package httpapi

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

// IsHTMX isteğin HTMX tarafından mı gönderildiğini söyler.
// HTMX her isteğe HX-Request: true başlığı ekler.
//
// BOOST HARİÇ. `hx-boost` ile yapılan gezinme de HX-Request gönderir ve
// ayrıca HX-Boosted: true ekler. Boost bir SAYFA gezinmesidir; sunucudan
// tam sayfa bekler (yb. hx-select=".admin-main"). Boost'u da "parça isteği"
// saymak, panele `#medya-panel` gibi bir fragment döndürüyordu — o zaman
// hedefte .admin-main bulunmuyor ve swap BOŞ kalıyordu (ölçüm: belge aynı
// kalıyor ama içerik tamamen boşalıyordu).
//
// Bu ayrım kritik: parça istekleri (hx-get + hx-target) eskisi gibi parça
// alır, boost'lu gezinme tam sayfa alır ve yalnız .admin-main'i çizer.
func IsHTMX(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Boosted")), "true") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true")
}

// renderPartial yalnızca ilgili fragment'i basar. Admin yanıtları
// önbelleğe alınmamalı; HTMX/HTML ayrımı için Vary de eklenir.
func (s *Server) renderPartial(w http.ResponseWriter, r *http.Request, name string, data *tmplData) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		s.logger.Error("partial basılamadı", "template", name, "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
	}
}

// htmxRedirect HTMX isteğinde tarayıcıyı tam sayfa yönlendirir; normal
// istekte false döner ve çağıran flashRedirect kullanır.
func (s *Server) htmxRedirect(w http.ResponseWriter, r *http.Request, target string) bool {
	if !IsHTMX(r) {
		return false
	}
	w.Header().Set("HX-Redirect", target)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
	return true
}

// writeHTMXTrigger, HX-Trigger başlığıyla tarayıcıya özel bir olay gönderir.
// AI gibi JSON sonucu alan aksiyonlarda DOM'u değiştirmeden alanları
// güncellemek için kullanılır (hx-swap="none").
func (s *Server) writeHTMXTrigger(w http.ResponseWriter, event string, payload any) {
	b, err := marshalJSONASCII(map[string]any{event: payload})
	if err != nil {
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", b)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}

// marshalJSONASCII, json.Marshal çıktısındaki ASCII dışı karakterleri
// \uXXXX kaçışlarına çevirir. HX-Trigger bir HTTP başlığıdır; başlıklar
// bayt düzeyinde ISO-8859-1 gibi ele alındığından ham UTF-8 taşımak
// Türkçe karakterleri tarayıcıda bozar ("Türkçe" → "TÃ¼rkÃ§e"). Başlık
// yalnızca ASCII taşırsa JSON kaçışları istemcide çözülünce metin
// birebir geri gelir (Türkçe, emoji ve astral karakterler dahil).
func marshalJSONASCII(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.Grow(len(b))
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r < utf8.RuneSelf {
			// #nosec G115 -- r < utf8.RuneSelf (128) koşulu byte aralığını
			// garanti eder; dönüşüm taşamaz.
			sb.WriteByte(byte(r))
			i += size
			continue
		}
		if r > 0xFFFF {
			// UTF-16 vekil çifti (ör. emoji).
			r -= 0x10000
			fmt.Fprintf(&sb, `\u%04x\u%04x`, 0xD800+(r>>10), 0xDC00+(r&0x3FF))
		} else {
			fmt.Fprintf(&sb, `\u%04x`, r)
		}
		i += size
	}
	return sb.String(), nil
}

// writeHTMXStatus küçük bir durum parçası basar (flash + içerik).
// Tuval/canvas editörlerinde tam panel yerine yalnızca durum alanını
// güncellemek için kullanılır (ör. Sayfa Tasarımcısı manuel Kaydet).
func (s *Server) writeHTMXStatus(w http.ResponseWriter, text, kind string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request")
	w.WriteHeader(http.StatusOK)
	tmpl := `<div class="flash %s">%s</div>`
	kind = template.HTMLEscapeString(kind)
	if kind == "" {
		kind = "info"
	}
	text = template.HTMLEscapeString(text)
	_, _ = fmt.Fprintf(w, tmpl, kind, text)
}

// withHTMXCurrentQuery, HTMX aksiyonlarından sonra listeyi kullanıcının
// gördüğü filtrelerle yeniden kurabilmek için HX-Current-URL'in query
// parametrelerini isteğe taşır.
func withHTMXCurrentQuery(r *http.Request) *http.Request {
	raw := strings.TrimSpace(r.Header.Get("HX-Current-URL"))
	if raw == "" {
		return r
	}
	u, err := url.Parse(raw)
	if err != nil {
		return r
	}
	r2 := r.Clone(r.Context())
	u2 := *r2.URL
	u2.RawQuery = u.RawQuery
	r2.URL = &u2
	return r2
}
