package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"theblock/site/internal/repo"
)

type girisCanliYanit struct {
	Sa      int64 `json:"sa"`      // sunucu saati (UNIX)
	Cekildi int64 `json:"cekildi"` // fiyatın çekildiği an
	Hazir   bool  `json:"hazir"`

	BTC girisCanliFiyat `json:"btc"`
	ETH girisCanliFiyat `json:"eth"`
	SOL girisCanliFiyat `json:"sol"`

	Yol      string `json:"yol"`
	Alan     string `json:"alan"`
	Hacim    string `json:"hacim"`
	Dominans string `json:"dominans"`
	Endeks   string `json:"endeks"`
	EndeksUp bool   `json:"endeks_up"`

	FNG *girisCanliGosterge `json:"fng"`
	RSI *girisCanliGosterge `json:"rsi"`

	Serit []girisCanliSerit `json:"serit"`
}

type girisCanliFiyat struct {
	Var     bool   `json:"var"`
	Fiyat   string `json:"fiyat"`
	Degisim string `json:"degisim"`
	Up      bool   `json:"up"`
}

type girisCanliGosterge struct {
	Var    bool   `json:"var"`
	Deger  int    `json:"deger"`
	Etiket string `json:"etiket"`
	Up     bool   `json:"up"`
	Dun    int    `json:"dun,omitempty"`
	Hafta  int    `json:"hafta,omitempty"`
}

type girisCanliSerit struct {
	Sym     string `json:"sym"`
	Fiyat   string `json:"fiyat"`
	Degisim string `json:"degisim"`
	Up      bool   `json:"up"`
}

// handleCDGirisCanli hero + şerit verisini JSON döndürür.
func (s *Server) handleCDGirisCanli(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	g := s.cdGirisOlustur(ctx)

	y := girisCanliYanit{
		Sa:       time.Now().Unix(),
		Cekildi:  g.Cekildi,
		Hazir:    g.Hazir,
		BTC:      girisCanliFiyat{Var: g.BTCFiyat != "", Fiyat: g.BTCFiyat, Degisim: g.BTCDegisim, Up: g.BTCUp},
		ETH:      girisCanliFiyat{Var: g.ETHFiyat != "", Fiyat: g.ETHFiyat, Degisim: g.ETHDegisim, Up: g.ETHUp},
		SOL:      girisCanliFiyat{Var: g.SOLFiyat != "", Fiyat: g.SOLFiyat, Degisim: g.SOLDegisim, Up: g.SOLUp},
		Yol:      g.BTCYol,
		Alan:     g.BTCAlan,
		Hacim:    g.Hacim,
		Dominans: g.Dominans,
		Endeks:   g.Endeks,
		EndeksUp: g.EndeksUp,
	}
	if g.FNGVar {
		y.FNG = &girisCanliGosterge{
			Var: true, Deger: g.FNGDeger, Etiket: g.FNGEtiket,
			Up: g.FNGUp, Dun: g.FNGDun, Hafta: g.FNGHafta,
		}
	}
	if g.RSIVar {
		y.RSI = &girisCanliGosterge{
			Var: true, Deger: g.RSI, Etiket: g.RSIEtiket, Up: g.RSIUp,
		}
	}
	for _, f := range repo.Fiyatlar(ctx, s.pool) {
		if len(y.Serit) >= 7 {
			break
		}
		if f.Price <= 0 {
			continue
		}
		y.Serit = append(y.Serit, girisCanliSerit{
			Sym:     f.Sym,
			Fiyat:   repo.FiyatBicim(f.Price),
			Degisim: repo.DegisimBicim(f.Change),
			Up:      f.Change >= 0,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// ASLA önbelleğe alınmaz: tarayıcı/ara katman her isteği sunucuya
	// ulaştırsın. (Sunucu tarafında zaten DB önbelleği var.)
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(y); err != nil {
		s.logger.Error("canlı giriş JSON", "error", err)
	}
}
