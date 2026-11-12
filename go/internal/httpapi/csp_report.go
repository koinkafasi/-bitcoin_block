package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	cspRaporMaxGovde   = 16 << 10 // 16 KB
	cspRaporPencere    = time.Minute
	cspRaporIPAzami    = 60  // IP başına dakikada rapor
	cspRaporTekilAzami = 200 // dakikada loglanacak tekil ihlal
)

type cspRaporDurum struct {
	mu       sync.Mutex
	ipSayac  map[string]*denemeSayaci
	tekil    map[string]time.Time
	loglanan int
	pencere  time.Time
}

var cspRapor = cspRaporDurum{
	ipSayac: map[string]*denemeSayaci{},
	tekil:   map[string]time.Time{},
}

func (s *Server) handleCSPReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ip := clientIP(r)
	if !cspRaporIzinli(ip) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ct := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	if ct != "application/csp-report" && ct != "application/reports+json" && ct != "application/json" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	govde, err := io.ReadAll(io.LimitReader(r.Body, cspRaporMaxGovde))
	if err != nil || len(govde) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for _, ihlal := range cspIhlalleriniCoz(ct, govde) {
		s.cspIhlalLogla(ihlal)
	}
	w.WriteHeader(http.StatusNoContent)
}

// cspIhlal raporlanan tek ihlalin süzülmüş hâli.
type cspIhlal struct {
	Direktif   string
	Engellenen string // şema://host ya da 'inline' / 'eval'
	Belge      string // yol
	Kip        string // enforce | report
}

// cspIhlalleriniCoz iki rapor biçimini ortak yapıya çevirir.
func cspIhlalleriniCoz(ct string, govde []byte) []cspIhlal {
	var out []cspIhlal

	if ct == "application/reports+json" {
		var liste []struct {
			Type string `json:"type"`
			Body struct {
				DocumentURI        string `json:"documentURL"`
				BlockedURL         string `json:"blockedURL"`
				EffectiveDirective string `json:"effectiveDirective"`
				Disposition        string `json:"disposition"`
			} `json:"body"`
		}
		if err := json.Unmarshal(govde, &liste); err != nil {
			return nil
		}
		for _, it := range liste {
			if it.Type != "" && it.Type != "csp-violation" {
				continue
			}
			out = append(out, cspIhlal{
				Direktif:   temizDirektif(it.Body.EffectiveDirective),
				Engellenen: sadelestirURL(it.Body.BlockedURL),
				Belge:      sadelestirYol(it.Body.DocumentURI),
				Kip:        temizKip(it.Body.Disposition),
			})
		}
		return out
	}

	var tek struct {
		Rapor struct {
			DocumentURI        string `json:"document-uri"`
			BlockedURI         string `json:"blocked-uri"`
			ViolatedDirective  string `json:"violated-directive"`
			EffectiveDirective string `json:"effective-directive"`
			Disposition        string `json:"disposition"`
		} `json:"csp-report"`
	}
	if err := json.Unmarshal(govde, &tek); err != nil {
		return nil
	}
	d := tek.Rapor.EffectiveDirective
	if d == "" {
		d = tek.Rapor.ViolatedDirective
	}
	out = append(out, cspIhlal{
		Direktif:   temizDirektif(d),
		Engellenen: sadelestirURL(tek.Rapor.BlockedURI),
		Belge:      sadelestirYol(tek.Rapor.DocumentURI),
		Kip:        temizKip(tek.Rapor.Disposition),
	})
	return out
}

// temizDirektif direktif adını beyaz listeye indirger.
func temizDirektif(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	// "script-src 'self' ..." gelebilir; ilk kelime yeter.
	if i := strings.IndexAny(d, " \t"); i > 0 {
		d = d[:i]
	}
	switch d {
	case "script-src", "script-src-elem", "script-src-attr",
		"style-src", "style-src-elem", "style-src-attr",
		"img-src", "font-src", "connect-src", "frame-src",
		"object-src", "media-src", "worker-src", "base-uri",
		"form-action", "frame-ancestors", "default-src":
		return d
	}
	if d == "" {
		return "bilinmeyen"
	}
	return "diger"
}

func temizKip(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "report":
		return "report"
	default:
		return "enforce"
	}
}

// sadelestirURL yalnız şema + host bırakır (yol/sorgu atılır; PII sızmasın).
func sadelestirURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Tarayıcılar satır içi ihlallerde "inline"/"eval"/"data" gibi kısa
	// belirteçler gönderir; bunlar olduğu gibi kalsın.
	low := strings.ToLower(raw)
	switch low {
	case "inline", "eval", "data", "blob", "filesystem", "wasm-eval":
		return low
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "diger"
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
}

// sadelestirYol belge adresinden yalnız yolu bırakır.
func sadelestirYol(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	p := u.Path
	if p == "" {
		p = "/"
	}
	if len(p) > 120 {
		p = p[:120]
	}
	return p
}

// cspIhlalLogla tek ihlali (gürültü sınırlamasıyla) günlüğe yazar.
func (s *Server) cspIhlalLogla(ihlal cspIhlal) {
	anahtar := ihlal.Direktif + "|" + ihlal.Engellenen + "|" + ihlal.Belge

	cspRapor.mu.Lock()
	simdi := time.Now()
	if simdi.Sub(cspRapor.pencere) > cspRaporPencere {
		cspRapor.pencere = simdi
		cspRapor.loglanan = 0
		cspRapor.tekil = map[string]time.Time{}
	}
	if t, ok := cspRapor.tekil[anahtar]; ok && simdi.Sub(t) < cspRaporPencere {
		cspRapor.mu.Unlock()
		return
	}
	if cspRapor.loglanan >= cspRaporTekilAzami {
		cspRapor.mu.Unlock()
		return
	}
	cspRapor.tekil[anahtar] = simdi
	cspRapor.loglanan++
	cspRapor.mu.Unlock()

	s.logger.Warn("csp ihlali",
		"directive", ihlal.Direktif,
		"blocked", ihlal.Engellenen,
		"document", ihlal.Belge,
		"mode", ihlal.Kip,
	)
}

// cspRaporIzinli IP başına dakikalık rapor sınırı.
func cspRaporIzinli(ip string) bool {
	simdi := time.Now()
	cspRapor.mu.Lock()
	defer cspRapor.mu.Unlock()

	c := cspRapor.ipSayac[ip]
	if c == nil || simdi.Sub(c.ilk) > cspRaporPencere {
		c = &denemeSayaci{ilk: simdi}
		cspRapor.ipSayac[ip] = c
	}
	if c.n >= cspRaporIPAzami {
		return false
	}
	c.n++
	if len(cspRapor.ipSayac) > 2000 {
		for k, v := range cspRapor.ipSayac {
			if simdi.Sub(v.ilk) > cspRaporPencere {
				delete(cspRapor.ipSayac, k)
			}
		}
	}
	return true
}

// cspRaporURL mutlak rapor adresi üretir (report-uri mutlak olmalı).
func (s *Server) cspRaporURL() string {
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	if base == "" {
		return "/csp-rapor"
	}
	return base + "/csp-rapor"
}
