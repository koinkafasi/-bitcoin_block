package httpapi

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/cryptoquant"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

const (
	zincirKind = "zincir"
	zincirTTL  = 10 * time.Minute
	zincirGun  = 90
)

// zincirPaket önbellekte saklanan ham seriler.
type zincirPaket struct {
	Updated time.Time                 `json:"updated"`
	Fiyat   []cryptoquant.Bar         `json:"fiyat,omitempty"`
	MVRV    []cryptoquant.MVRV        `json:"mvrv,omitempty"`
	SOPR    []cryptoquant.SOPR        `json:"sopr,omitempty"`
	Netflow []cryptoquant.Akis        `json:"netflow,omitempty"`
	Rezerv  []cryptoquant.Akis        `json:"rezerv,omitempty"`
	Balina  []cryptoquant.BalinaOrani `json:"balina,omitempty"`
}

// zincirVeri önbellekten okur; bayatsa CryptoQuant'tan paralel çeker.
func (s *Server) zincirVeri(ctx context.Context, coin string) zincirPaket {
	coin = strings.ToLower(strings.TrimSpace(coin))
	if coin == "" {
		return zincirPaket{}
	}
	if b, err := repo.CacheGet(ctx, s.pool, zincirKind, coin); err == nil && len(b) > 0 {
		var p zincirPaket
		if json.Unmarshal(b, &p) == nil && time.Since(p.Updated) < zincirTTL {
			return p
		}
	}

	ist := cryptoquant.Yeni(s.cfg.CryptoQuantKey)
	if !ist.Var() {
		return zincirPaket{}
	}

	var p zincirPaket
	var mu sync.Mutex
	var wg sync.WaitGroup
	ekle := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}

	if cryptoquant.DestekliFiyat[coin] {
		ekle(func() {
			if d, err := ist.Fiyat(ctx, coin, zincirGun); err == nil {
				mu.Lock()
				p.Fiyat = d
				mu.Unlock()
			} else {
				s.logger.Debug("cryptoquant fiyat", "coin", coin, "error", err)
			}
		})
		ekle(func() {
			if d, err := ist.BorsaAkis(ctx, coin, "netflow", "binance", zincirGun); err == nil {
				mu.Lock()
				p.Netflow = d
				mu.Unlock()
			}
		})
		ekle(func() {
			if d, err := ist.BorsaAkis(ctx, coin, "reserve", "binance", zincirGun); err == nil {
				mu.Lock()
				p.Rezerv = d
				mu.Unlock()
			}
		})
	}
	if coin == "btc" {
		ekle(func() {
			if d, err := ist.MVRV(ctx, zincirGun); err == nil {
				mu.Lock()
				p.MVRV = d
				mu.Unlock()
			}
		})
		ekle(func() {
			if d, err := ist.SOPR(ctx, zincirGun); err == nil {
				mu.Lock()
				p.SOPR = d
				mu.Unlock()
			}
		})
		ekle(func() {
			if d, err := ist.BorsaBalinaOrani(ctx, "binance", zincirGun); err == nil {
				mu.Lock()
				p.Balina = d
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	if len(p.Fiyat) == 0 && len(p.MVRV) == 0 && len(p.SOPR) == 0 &&
		len(p.Netflow) == 0 && len(p.Rezerv) == 0 && len(p.Balina) == 0 {
		return zincirPaket{}
	}
	p.Updated = time.Now()
	if b, err := json.Marshal(p); err == nil {
		if err := repo.CacheSet(ctx, s.pool, zincirKind, coin, b); err != nil {
			s.logger.Debug("cryptoquant önbellek yazma", "coin", coin, "error", err)
		}
	}
	return p
}

// cdFiyatGrafikKart OHLCV kapanışlarından fiyat grafiği üretir (son 30 gün).
func cdFiyatGrafikKart(bars []cryptoquant.Bar) *templkit.CDGrafikKart {
	if len(bars) < 2 {
		return nil
	}
	son := bars
	if len(son) > 30 {
		son = son[len(son)-30:]
	}
	deger := make([]float64, 0, len(son))
	for _, b := range son {
		if b.Close > 0 {
			deger = append(deger, b.Close)
		}
	}
	yol, alan, min, maks, ok := templkit.CDGrafikYolu(deger)
	if !ok {
		return nil
	}
	ilk, sonDeger := deger[0], deger[len(deger)-1]
	degisim := 0.0
	if ilk > 0 {
		degisim = (sonDeger - ilk) / ilk * 100
	}
	return &templkit.CDGrafikKart{
		Baslik:    "Fiyat",
		Deger:     repo.FiyatBicim(sonDeger),
		Degisim:   repo.DegisimBicim(degisim),
		Up:        degisim >= 0,
		Alt:       "Kapanış fiyatı · son " + strconv.Itoa(len(deger)) + " gün · CryptoQuant",
		Yol:       yol,
		Alan:      alan,
		Min:       repo.FiyatBicim(min),
		Maks:      repo.FiyatBicim(maks),
		EtiketSol: gunEtiketi(son[0].Date),
		EtiketSag: gunEtiketi(son[len(son)-1].Date),
	}
}

func cdMetrikKart(baslik, alt string, deger []float64, tarih []string, bicim func(float64) string) *templkit.CDGrafikKart {
	return cdMetrikKartTip(baslik, alt, deger, tarih, bicim, true)
}

// cdMetrikKartTip yuzde=false ise farkı MUTLAK olarak gösterir (net akış gibi
// işareti anlamlı serilerde yüzde değişim yanıltıcı olur).
func cdMetrikKartTip(baslik, alt string, deger []float64, tarih []string, bicim func(float64) string, yuzde bool) *templkit.CDGrafikKart {
	if len(deger) < 2 {
		return nil
	}
	son := deger[len(deger)-1]
	onceki := deger[len(deger)-2]
	fark := son - onceki
	degisim := 0.0
	if onceki != 0 {
		degisim = fark / abs(onceki) * 100
	}
	metin := repo.DegisimBicim(degisim)
	if !yuzde {
		isaret := "+"
		if fark < 0 {
			isaret = "−"
			fark = -fark
		}
		metin = isaret + bicim(fark)
	}
	kart := &templkit.CDGrafikKart{
		Baslik:  baslik,
		Deger:   bicim(son),
		Degisim: metin,
		Up:      fark >= 0,
		Alt:     alt,
		Kucuk:   true,
	}
	kart.Yol, kart.Alan, _, _, _ = templkit.CDGrafikYolu(deger)
	if len(tarih) == len(deger) {
		kart.EtiketSol = gunEtiketi(tarih[0])
		kart.EtiketSag = gunEtiketi(tarih[len(tarih)-1])
	}
	return kart
}

// cdCoinGrafikler /price/{coin} için fiyat grafiği + zincir-üstü kartlar.
func (s *Server) cdCoinGrafikler(ctx context.Context, sym, ad string) (*templkit.CDGrafikKart, []templkit.CDGrafikKart) {
	coin := strings.ToLower(strings.TrimSpace(sym))
	if coin == "" {
		return nil, nil
	}
	p := s.zincirVeri(ctx, coin)
	fiyat := cdFiyatGrafikKart(p.Fiyat)

	var kartlar []templkit.CDGrafikKart
	if len(p.MVRV) > 0 {
		d := make([]float64, 0, len(p.MVRV))
		t := make([]string, 0, len(p.MVRV))
		for _, n := range p.MVRV {
			d = append(d, n.MVRV)
			t = append(t, n.Date)
		}
		if k := cdMetrikKart("MVRV", "Piyasa değeri / gerçekleşen değer", d, t,
			func(v float64) string { return repo.SayiBicim(v, 2) }); k != nil {
			kartlar = append(kartlar, *k)
		}
	}
	if len(p.SOPR) > 0 {
		d := make([]float64, 0, len(p.SOPR))
		t := make([]string, 0, len(p.SOPR))
		for _, n := range p.SOPR {
			d = append(d, n.SOPR)
			t = append(t, n.Date)
		}
		if k := cdMetrikKart("SOPR", "Harcanan çıktı kâr oranı", d, t,
			func(v float64) string { return repo.SayiBicim(v, 3) }); k != nil {
			kartlar = append(kartlar, *k)
		}
	}
	if len(p.Netflow) > 0 {
		d := make([]float64, 0, len(p.Netflow))
		t := make([]string, 0, len(p.Netflow))
		for _, n := range p.Netflow {
			d = append(d, n.NetflowTotal)
			t = append(t, n.Date)
		}
		if k := cdMetrikKartTip("Borsa net akışı", "Binance · günlük net giriş (+) / çıkış (−)", d, t,
			func(v float64) string { return repo.SayiBicim(v, 0) + " " + strings.ToUpper(sym) }, false); k != nil {
			kartlar = append(kartlar, *k)
		}
	}
	if len(p.Rezerv) > 0 {
		d := make([]float64, 0, len(p.Rezerv))
		t := make([]string, 0, len(p.Rezerv))
		for _, n := range p.Rezerv {
			d = append(d, n.Reserve)
			t = append(t, n.Date)
		}
		if k := cdMetrikKart("Borsa rezervi", "Binance · borsada tutulan bakiye", d, t,
			func(v float64) string { return repo.SayiBicim(v, 0) + " " + strings.ToUpper(sym) }); k != nil {
			kartlar = append(kartlar, *k)
		}
	}
	if len(p.Balina) > 0 {
		d := make([]float64, 0, len(p.Balina))
		t := make([]string, 0, len(p.Balina))
		for _, n := range p.Balina {
			d = append(d, n.Oran)
			t = append(t, n.Date)
		}
		if k := cdMetrikKart("Balina oranı", "Binance · balina işlem payı", d, t,
			func(v float64) string { return repo.SayiBicim(v*100, 2) + "%" }); k != nil {
			kartlar = append(kartlar, *k)
		}
	}
	return fiyat, kartlar
}

// cdNabiz ana sayfa/markets "Piyasa Nabzı" modülü: BTC fiyat grafiği + kartlar.
func (s *Server) cdNabiz(ctx context.Context) (*templkit.CDGrafikKart, []templkit.CDGrafikKart) {
	p := s.zincirVeri(ctx, "btc")
	fiyat := cdFiyatGrafikKart(p.Fiyat)
	kartlar := make([]templkit.CDGrafikKart, 0, 4)
	if len(p.MVRV) > 0 {
		d, t := make([]float64, 0, len(p.MVRV)), make([]string, 0, len(p.MVRV))
		for _, n := range p.MVRV {
			d = append(d, n.MVRV)
			t = append(t, n.Date)
		}
		if k := cdMetrikKart("MVRV", "Piyasa / gerçekleşen değer", d, t,
			func(v float64) string { return repo.SayiBicim(v, 2) }); k != nil {
			kartlar = append(kartlar, *k)
		}
	}
	if len(p.SOPR) > 0 {
		d, t := make([]float64, 0, len(p.SOPR)), make([]string, 0, len(p.SOPR))
		for _, n := range p.SOPR {
			d = append(d, n.SOPR)
			t = append(t, n.Date)
		}
		if k := cdMetrikKart("SOPR", "Harcanan çıktı kâr oranı", d, t,
			func(v float64) string { return repo.SayiBicim(v, 3) }); k != nil {
			kartlar = append(kartlar, *k)
		}
	}
	if len(p.Netflow) > 0 {
		d, t := make([]float64, 0, len(p.Netflow)), make([]string, 0, len(p.Netflow))
		for _, n := range p.Netflow {
			d = append(d, n.NetflowTotal)
			t = append(t, n.Date)
		}
		if k := cdMetrikKartTip("Borsa net akışı", "Binance · BTC", d, t,
			func(v float64) string { return repo.SayiBicim(v, 0) + " BTC" }, false); k != nil {
			kartlar = append(kartlar, *k)
		}
	}
	if len(p.Rezerv) > 0 {
		d, t := make([]float64, 0, len(p.Rezerv)), make([]string, 0, len(p.Rezerv))
		for _, n := range p.Rezerv {
			d = append(d, n.Reserve)
			t = append(t, n.Date)
		}
		if k := cdMetrikKart("Borsa rezervi", "Binance · BTC", d, t,
			func(v float64) string { return repo.SayiBicim(v, 0) + " BTC" }); k != nil {
			kartlar = append(kartlar, *k)
		}
	}
	return fiyat, kartlar
}

// gunEtiketi "2026-11-07" → "7 Kas".
func gunEtiketi(iso string) string {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(iso))
	if err != nil {
		return iso
	}
	aylar := [...]string{"Oca", "Şub", "Mar", "Nis", "May", "Haz", "Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"}
	return strconv.Itoa(t.Day()) + " " + aylar[int(t.Month())-1]
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
