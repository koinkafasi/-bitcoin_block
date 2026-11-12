package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

type finansZaman struct {
	HM string `json:"hm"` // 15:04
	DT string `json:"dt"` // 29.10.2026 15:04
}

type finansFiyat struct {
	F float64 `json:"f"` // fiyat / değer
	D float64 `json:"d"` // yüzde değişim
}

type finansKripto struct {
	F  float64 `json:"f"`
	D  float64 `json:"d"`
	D7 float64 `json:"d7"`
	H  float64 `json:"h"`
	P  float64 `json:"p"`
}

type finansPiyasa struct {
	F float64 `json:"f"`
	D float64 `json:"d"`
	Y int     `json:"y"`
	O int     `json:"o,omitempty"` // ondalık
	B string  `json:"b,omitempty"` // birim (%, $)
}

type finansCanli struct {
	Ok     bool                    `json:"ok"`
	Guncel map[string]finansZaman  `json:"guncel,omitempty"`
	Kur    map[string]float64      `json:"kur,omitempty"`
	Hisse  map[string]finansFiyat  `json:"hisse,omitempty"`
	Kripto map[string]finansKripto `json:"kripto,omitempty"`
	Piyasa map[string]finansPiyasa `json:"piyasa,omitempty"`
	Servis map[string]finansFiyat  `json:"servis,omitempty"`
}

func finansZamanOf(t time.Time) finansZaman {
	if t.IsZero() {
		return finansZaman{}
	}
	return finansZaman{HM: t.Format("15:04"), DT: t.Format("02.01.2006 15:04")}
}

// finansTurleri ?tur=hisse,kripto,... süzgecini okur. Boş/geçersiz değerde
// tüm türler döner: istemci yanlış parametre yüzünden boş ekran görmez.
func finansTurleri(r *http.Request) map[string]bool {
	hepsi := []string{"kur", "hisse", "kripto", "piyasa", "servis"}
	raw := strings.TrimSpace(r.URL.Query().Get("tur"))
	if raw == "" {
		out := make(map[string]bool, len(hepsi))
		for _, t := range hepsi {
			out[t] = true
		}
		return out
	}
	out := map[string]bool{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		switch p {
		case "kur", "hisse", "kripto", "piyasa", "servis":
			out[p] = true
		}
	}
	if len(out) == 0 {
		for _, t := range hepsi {
			out[t] = true
		}
	}
	return out
}

// finansCanliVeri önbellekleri okuyup istemcinin beklediği kompakt yapıya
// çevirir. İstek yolunda ağa çıkılmaz; yalnız services_cache okunur.
func (s *Server) finansCanliVeri(ctx context.Context, istenen map[string]bool) finansCanli {
	out := finansCanli{Ok: true, Guncel: map[string]finansZaman{}}

	if istenen["kur"] {
		kt := repo.KurlarOf(ctx, s.pool)
		if len(kt.Kurlar) > 0 {
			out.Kur = make(map[string]float64, len(kt.Kurlar))
			for code, rate := range kt.Kurlar {
				if rate > 0 {
					out.Kur[strings.ToUpper(code)] = rate
				}
			}
			if len(out.Kur) > 0 {
				out.Guncel["kur"] = finansZamanOf(kt.Updated)
			}
		}
	}

	if istenen["hisse"] {
		hd := repo.HisselerOf(ctx, s.pool)
		if len(hd.Rows) > 0 {
			out.Hisse = make(map[string]finansFiyat, len(hd.Rows))
			for _, r := range hd.Rows {
				if r.Value > 0 && r.Code != "" {
					out.Hisse[r.Code] = finansFiyat{F: r.Value, D: r.Delta}
				}
			}
			if len(out.Hisse) > 0 {
				out.Guncel["hisse"] = finansZamanOf(hd.Updated)
			}
		}
	}

	if istenen["kripto"] {
		kd := repo.KriptoOf(ctx, s.pool)
		if len(kd.Rows) > 0 {
			out.Kripto = make(map[string]finansKripto, len(kd.Rows))
			for _, r := range kd.Rows {
				if r.Value > 0 && r.Symbol != "" {
					out.Kripto[r.Symbol] = finansKripto{
						F: r.Value, D: r.Delta, D7: r.Delta7,
						H: r.Volume, P: r.MarketCap,
					}
				}
			}
			if len(out.Kripto) > 0 {
				out.Guncel["kripto"] = finansZamanOf(kd.Updated)
			}
		}
	}

	if istenen["piyasa"] {
		ps := repo.PiyasaOf(ctx, s.pool)
		if len(ps.Kalemler) > 0 {
			out.Piyasa = make(map[string]finansPiyasa, len(ps.Kalemler))
			for _, k := range ps.Kalemler {
				if k.Anahtar == "" || k.Demo {
					continue
				}
				out.Piyasa[k.Anahtar] = finansPiyasa{
					F: k.Deger, D: k.Degisim, Y: k.Yon,
					O: k.Ondalik, B: k.Birim,
				}
			}
			if len(out.Piyasa) > 0 {
				out.Guncel["piyasa"] = finansZamanOf(ps.Guncel)
			}
		}
	}

	if istenen["servis"] {
		items, demo := servisKurlar(ctx, s.pool)
		if len(items) > 0 {
			out.Servis = make(map[string]finansFiyat, len(items))
			for _, k := range items {
				// Demo kurlar (USD/EUR/ALTIN) hiçbir zaman akışa girmez.
				if demo && (k.Name == "USD" || k.Name == "EUR" || k.Name == "ALTIN") {
					continue
				}
				if k.Value > 0 {
					out.Servis[k.Name] = finansFiyat{F: k.Value, D: k.Delta}
				}
			}
			if len(out.Servis) > 0 {
				if bd := repo.BorsaOf(ctx, s.pool); !bd.Updated.IsZero() {
					out.Guncel["servis"] = finansZamanOf(bd.Updated)
				}
			}
		}
	}

	return out
}

func servisKurlar(ctx context.Context, pool repo.DB) ([]repo.Kur, bool) {
	kur := map[string]repo.Kur{}
	demo := false

	for _, k := range repo.PiyasaOf(ctx, pool).Kalemler {
		if k.Demo || k.Deger <= 0 {
			continue
		}
		var ad string
		switch k.Anahtar {
		case "usd":
			ad = "USD"
		case "eur":
			ad = "EUR"
		case "gram-altin":
			ad = "ALTIN"
		case "bist100":
			ad = "BIST"
		}
		if ad != "" {
			kur[ad] = repo.Kur{Name: ad, Value: k.Deger, Delta: k.Degisim}
		}
	}

	// 2) Eski kur önbelleği — yalnız piyasa önbelleğinde eksik kalan kalemler
	//    için yedek. Demo değerler banda hiç girmez.
	kurEksik := false
	for _, ad := range []string{"USD", "EUR", "ALTIN"} {
		if _, varMi := kur[ad]; !varMi {
			kurEksik = true
			break
		}
	}
	if kb, err := repo.CacheGet(ctx, pool, "kur", "ana"); kurEksik && err == nil {
		var km map[string]any
		if json.Unmarshal(kb, &km) == nil {
			kurDemo, _ := km["demo"].(bool)
			if kurDemo {
				demo = true
			}
			// Eksik/boş değer 0,00 olarak basılmaz.
			for ad, anahtar := range map[string]string{"USD": "dolar", "EUR": "euro", "ALTIN": "altin"} {
				if _, varMi := kur[ad]; varMi || kurDemo {
					continue
				}
				if v := num(km[anahtar]); v > 0 {
					kur[ad] = repo.Kur{Name: ad, Value: v}
				}
			}
		}
	}

	// 3) Borsa önbelleği: BTC, ONS ve (piyasa yoksa) BIST.
	if bb, err := repo.CacheGet(ctx, pool, "borsa", "ana"); err == nil {
		var bd repo.BorsaData
		if json.Unmarshal(bb, &bd) == nil {
			for _, row := range bd.Rows {
				if row.Value <= 0 {
					continue
				}
				switch {
				case row.Kind == "endeks" && row.Code == "XU100":
					if _, ok := kur["BIST"]; !ok {
						kur["BIST"] = repo.Kur{Name: "BIST", Value: row.Value, Delta: row.Delta}
					}
				case row.Kind == "kripto" && row.Code == "BTC":
					kur["BTC"] = repo.Kur{Name: "BTC", Value: row.Value, Delta: row.Delta}
				case row.Kind == "altin_ons" && row.Code == "ONS":
					kur["ONS"] = repo.Kur{Name: "ONS", Value: row.Value, Delta: row.Delta}
				}
			}
		}
	}

	// Ekrandaki sıra sabittir: BIST, USD, EUR, ALTIN, BTC, ONS.
	out := make([]repo.Kur, 0, 6)
	for _, ad := range []string{"BIST", "USD", "EUR", "ALTIN", "BTC", "ONS"} {
		if k, ok := kur[ad]; ok && k.Value > 0 {
			out = append(out, k)
		}
	}
	return out, demo
}

// finansETag gövdenin içerik özetidir; değerler değişmediyse istemci 304 alır.
func finansETag(b []byte) string {
	h := sha256.Sum256(b)
	return `"` + hex.EncodeToString(h[:16]) + `"`
}

func finansETagEsles(header, etag string) bool {
	for _, p := range strings.Split(header, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "*" || p == etag || strings.TrimPrefix(p, "W/") == etag {
			return true
		}
	}
	return false
}

// handleFinansCanli GET /api/piyasa/canli
func (s *Server) handleFinansCanli(w http.ResponseWriter, r *http.Request) {
	veri := s.finansCanliVeri(r.Context(), finansTurleri(r))
	b, err := json.Marshal(veri)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false})
		return
	}
	etag := finansETag(b)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", etag)
	if finansETagEsles(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
