package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"sort"
	"strings"
	"time"
)

const (
	binanceTickerURL = "https://api.binance.com/api/v3/ticker/24hr"
	binanceKlinesURL = "https://api.binance.com/api/v3/klines"
	binanceInfoURL   = "https://api.binance.com/api/v3/exchangeInfo"

	binanceTTL = 6 * time.Second

	binanceSembolTTL = 24 * time.Hour
)

// binanceFiyat tek sembolün canlı değeri.
type binanceFiyat struct {
	Fiyat   float64
	Degisim float64
}

func binanceSemboller(ctx context.Context, pool DB) map[string]bool {
	const kind, key = "fiyat", "binance-sembol"
	if b, err := CacheGet(ctx, pool, kind, key); err == nil && len(b) > 0 {
		var zarf struct {
			Cekildi int64    `json:"cekildi"`
			Sembol  []string `json:"sembol"`
		}
		if json.Unmarshal(b, &zarf) == nil && len(zarf.Sembol) > 0 {
			if time.Since(time.Unix(zarf.Cekildi, 0)) < binanceSembolTTL {
				out := make(map[string]bool, len(zarf.Sembol))
				for _, s := range zarf.Sembol {
					out[s] = true
				}
				return out
			}
		}
	}

	govde, err := binanceCekSinir(ctx, binanceInfoURL, 32<<20)
	if err != nil {
		return nil
	}
	var ham struct {
		Symbols []struct {
			Symbol     string `json:"symbol"`
			Status     string `json:"status"`
			QuoteAsset string `json:"quoteAsset"`
		} `json:"symbols"`
	}
	if json.Unmarshal(govde, &ham) != nil {
		return nil
	}
	liste := make([]string, 0, 600)
	for _, s := range ham.Symbols {
		if s.Status == "TRADING" && s.QuoteAsset == "USDT" {
			liste = append(liste, s.Symbol)
		}
	}
	if len(liste) == 0 {
		return nil
	}
	if b, err := json.Marshal(map[string]any{"cekildi": time.Now().Unix(), "sembol": liste}); err == nil {
		_ = CacheSet(ctx, pool, kind, key, b)
	}
	out := make(map[string]bool, len(liste))
	for _, s := range liste {
		out[s] = true
	}
	return out
}

func binanceFiyatlar(ctx context.Context, pool DB, syms []string) map[string]binanceFiyat {
	if len(syms) == 0 {
		return nil
	}
	gecerli := binanceSemboller(ctx, pool)

	// Parite listesi: BTC → BTCUSDT. USDT'nin kendisi atlanır.
	pariteler := make([]string, 0, len(syms))
	ters := make(map[string]string, len(syms)) // BTCUSDT → BTC
	for _, s := range syms {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s == "" || s == "USDT" {
			continue
		}
		p := s + "USDT"
		if gecerli != nil && !gecerli[p] {
			continue
		}
		pariteler = append(pariteler, p)
		ters[p] = s
	}
	if len(pariteler) == 0 {
		return nil
	}
	sort.Strings(pariteler)

	dizi, err := json.Marshal(pariteler)
	if err != nil {
		return nil
	}
	sorgu := neturl.Values{}
	sorgu.Set("symbols", string(dizi))
	url := binanceTickerURL + "?" + sorgu.Encode()
	govde, err := binanceCek(ctx, url)
	if err != nil {
		govde, err = binanceCek(ctx, binanceGuvenliSorgu(pariteler))
		if err != nil {
			return nil
		}
	}
	var ham []struct {
		Symbol             string `json:"symbol"`
		LastPrice          string `json:"lastPrice"`
		PriceChangePercent string `json:"priceChangePercent"`
	}
	if json.Unmarshal(govde, &ham) != nil {
		return nil
	}
	out := make(map[string]binanceFiyat, len(ham))
	for _, h := range ham {
		sym, ok := ters[h.Symbol]
		if !ok {
			continue
		}
		f, err1 := parseFloat(h.LastPrice)
		d, err2 := parseFloat(h.PriceChangePercent)
		if err1 != nil || err2 != nil {
			continue
		}
		out[sym] = binanceFiyat{Fiyat: f, Degisim: d}
	}
	return out
}

func binanceKlines(ctx context.Context, sembol string, adet int) ([]float64, error) {
	url := fmt.Sprintf("%s?symbol=%sUSDT&interval=1h&limit=%d", binanceKlinesURL, sembol, adet)
	govde, err := binanceCek(ctx, url)
	if err != nil {
		return nil, err
	}
	// Kline dizisi karışık tipli: [açılışZamanı, açılış, yüksek, düşük, KAPANIŞ, ...]
	var ham [][]json.RawMessage
	if err := json.Unmarshal(govde, &ham); err != nil {
		return nil, fmt.Errorf("binance klines: JSON: %w", err)
	}
	out := make([]float64, 0, len(ham))
	for _, k := range ham {
		if len(k) < 5 {
			continue
		}
		var kapanis float64
		if err := json.Unmarshal(k[4], &kapanis); err != nil || kapanis <= 0 {
			continue
		}
		out = append(out, kapanis)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("binance klines: boş")
	}
	return out, nil
}

func binanceGuvenliSorgu(pariteler []string) string {
	guvenli := map[string]bool{
		"BTCUSDT": true, "ETHUSDT": true, "BNBUSDT": true, "SOLUSDT": true,
		"XRPUSDT": true, "ADAUSDT": true, "DOGEUSDT": true, "AVAXUSDT": true,
		"LINKUSDT": true, "DOTUSDT": true, "LTCUSDT": true, "TRXUSDT": true,
		"SHIBUSDT": true, "UNIUSDT": true, "ATOMUSDT": true, "XLMUSDT": true,
		"NEARUSDT": true, "APTUSDT": true, "ARBUSDT": true, "OPUSDT": true,
		"FILUSDT": true, "INJUSDT": true, "SUIUSDT": true, "TONUSDT": true,
		"PEPEUSDT": true, "USDCUSDT": true, "WBTCUSDT": true, "ETCUSDT": true,
		"AAVEUSDT": true, "GRTUSDT": true, "ALGOUSDT": true, "VETUSDT": true,
	}
	alt := make([]string, 0, len(pariteler))
	for _, p := range pariteler {
		if guvenli[p] {
			alt = append(alt, p)
		}
	}
	if len(alt) == 0 {
		alt = []string{"BTCUSDT"}
	}
	dizi, _ := json.Marshal(alt)
	sorgu := neturl.Values{}
	sorgu.Set("symbols", string(dizi))
	return binanceTickerURL + "?" + sorgu.Encode()
}

// binanceCek ortak GET.
func binanceCek(ctx context.Context, url string) ([]byte, error) {
	return binanceCekSinir(ctx, url, 8<<20)
}

// binanceCekSinir gövde okuma sınırı verilebilen sürüm.
func binanceCekSinir(ctx context.Context, url string, sinir int64) ([]byte, error) {
	istek, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	istek.Header.Set("User-Agent", "bitcosar/1.0 (+https://bitcosar.com)")
	istek.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(istek)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, sinir))
}

// parseFloat boş/bozuk metni güvenle sayıya çevirir.
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("boş")
	}
	var f float64
	if _, err := fmt.Sscanf(s, "%g", &f); err != nil {
		return 0, err
	}
	return f, nil
}
