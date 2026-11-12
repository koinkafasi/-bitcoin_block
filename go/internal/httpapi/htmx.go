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
