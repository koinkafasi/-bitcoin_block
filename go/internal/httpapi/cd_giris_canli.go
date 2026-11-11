package httpapi

// Canlı piyasa ucu: GET /canli/giris
//
// NEDEN VAR (kullanıcı isteği, 11 Kasım 2026: "verileri canlı olarak
// çalışır istiyorum"): ana sayfa önbellekli HTML olarak servis ediliyor ve
// şerit sunucuda basılıyor. Sayfa açık kaldıkça fiyatlar DONUYORDU.
//
// AKIŞ: tarayıcı 30 sn'de bir BURAYA sorar. Burası CoinGecko'ya değil,
// repo önbelleğine bakar (fiyat 60 sn, grafik 10 dk). Yani:
//   - ziyaretçi sayısı ne olursa olsun upstream istek sayısı sabit
//     (ölçüldü: CoinGecko ücretsiz katmanı 6 istek/30 sn'de 429 veriyor),
//   - ekrandaki fiyat en fazla ~60 sn eski olur.
//
// "CANLI" ADI: uç canlı fiyat akışı değil, önbellekli fiyatın en taze
// hâlini verir. Yaş (`cekildi`) yanıtta var; arayüz "az önce" yazar.
// Okuyucuya olmayan bir gerçek zamanlılık vaat etmiyoruz.

import (
	"encoding/json"
	"net/http"
	"time"

	"theblock/site/internal/repo"
)

// girisCanliYanit tarayıcıya giden JSON.
//
// Alan adları KISA ve sabit: JS bunları doğrudan okur. Türkçe metin
// sunucuda biçimlenmiş gelir (fiyat/para birimi kuralı tek yerde kalsın).
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
	// Şerit: fiyat önbelleği zaten bellekte; ek maliyet yok. Yalnız
	// kripto kalemleri — BIST fiyatları ayrı kaynaktan (Yahoo) geliyor ve
	// bu uçta güncellenmiyor, eski değer yazmak yanıltıcı olurdu.
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
