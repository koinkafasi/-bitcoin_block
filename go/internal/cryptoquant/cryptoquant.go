// Package cryptoquant CryptoQuant API istemcisidir (zincir-üstü veri).
//
// NEDEN AYRI PAKET: fiyat/piyasa değeri CoinGecko'dan gelir (250+ varlık);
// CryptoQuant ise yalnız zincir-üstü analitik sağlar ve yalnız birkaç varlık
// için veri döndürür. İkisini karıştırmamak için bu paket kendi tiplerini
// tanımlar; httpapi katmanı ikisini birleştirir.
//
// ÖLÇÜLEN SINIRLAR (8 Kasım 2026, gerçek anahtar):
//   - price-ohlcv: yalnız btc, eth, xrp, trx (diğerleri 404)
//   - market-indicator/mvrv ve /sopr: yalnız btc
//   - exchange-flows/*: `exchange=` parametresi ZORUNLU; btc + eth çalışır
//   - flow-indicator/exchange-whale-ratio: btc
//
// Anahtar yoksa istemci "kapalı" davranır: Var() false döner ve hiçbir istek
// atılmaz. Sayfalar veri yoksa modülü hiç basmaz (uydurma değer yazılmaz).
package cryptoquant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// VarsayilanBase API kökü.
//
// DİKKAT: `GET /v1/` 404 döner — kök uç yok, doğrudan metrik yolları çağrılır.
const VarsayilanBase = "https://api.cryptoquant.com/v1"

// Istemci CryptoQuant HTTP istemcisi (eşzamanlı kullanıma uygundur).
type Istemci struct {
	anahtar string
	base    string
	hc      *http.Client
}

// Yeni istemci kurar. Anahtar boşsa istemci kapalıdır.
func Yeni(anahtar string) *Istemci {
	return &Istemci{
		anahtar: strings.TrimSpace(anahtar),
		base:    VarsayilanBase,
		// Zaman aşımı kısa: sayfa render'ı API'yi bekleyip kilitlenmemeli.
		hc: &http.Client{Timeout: 8 * time.Second},
	}
}

// Var anahtar tanımlı mı.
func (c *Istemci) Var() bool { return c != nil && c.anahtar != "" }

// zarf API yanıt zarfı: {"status":{...},"result":{"data":[...]}}.
type zarf[T any] struct {
	Status struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
	Result struct {
		Data []T `json:"data"`
	} `json:"result"`
}

// CevapHatasi API'nin hata yanıtı.
type CevapHatasi struct {
	Kod      int
	Mesaj    string
	Yol      string
	Aciklama string
}

func (e *CevapHatasi) Error() string {
	return fmt.Sprintf("cryptoquant %s: %d %s %s", e.Yol, e.Kod, e.Mesaj, e.Aciklama)
}

// al tek bir metriği okur ve zarfı çözer.
func (c *Istemci) al(ctx context.Context, yol string, hedef any) error {
	if !c.Var() {
		return fmt.Errorf("cryptoquant: anahtar tanımlı değil")
	}
	istek, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+yol, nil)
	if err != nil {
		return err
	}
	istek.Header.Set("Authorization", "Bearer "+c.anahtar)
	istek.Header.Set("Accept", "application/json")

	yanit, err := c.hc.Do(istek)
	if err != nil {
		return err
	}
	defer func() { _ = yanit.Body.Close() }()

	govde, err := io.ReadAll(io.LimitReader(yanit.Body, 8<<20))
	if err != nil {
		return err
	}
	if yanit.StatusCode != http.StatusOK {
		var h struct {
			Status struct {
				Mesaj    string `json:"message"`
				Aciklama string `json:"description"`
			} `json:"status"`
		}
		_ = json.Unmarshal(govde, &h)
		return &CevapHatasi{
			Kod:      yanit.StatusCode,
			Mesaj:    h.Status.Mesaj,
			Aciklama: h.Status.Aciklama,
			Yol:      yol,
		}
	}
	return json.Unmarshal(govde, hedef)
}

// Bar günlük OHLCV.
type Bar struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// MVRV noktası.
type MVRV struct {
	Date string  `json:"date"`
	MVRV float64 `json:"mvrv"`
}

// SOPR noktası (ham + düzeltilmiş + kısa/uzun vade).
type SOPR struct {
	Date    string  `json:"date"`
	SOPR    float64 `json:"sopr"`
	ASOPR   float64 `json:"a_sopr"`
	STHSOPR float64 `json:"sth_sopr"`
	LTHSOPR float64 `json:"lth_sopr"`
}

// Akis borsa akış metriği. Metriğe göre alan dolar:
//   - inflow / netflow → netflow_total ya da inflow_total
//   - reserve          → reserve, reserve_usd
//   - inflow           → inflow_total, inflow_top10, inflow_top10_ratio
type Akis struct {
	Date         string  `json:"date"`
	NetflowTotal float64 `json:"netflow_total"`
	InflowTotal  float64 `json:"inflow_total"`
	InflowTop10  float64 `json:"inflow_top10"`
	Reserve      float64 `json:"reserve"`
	ReserveUSD   float64 `json:"reserve_usd"`
}

// BalinaOrani borsa balina oranı noktası.
type BalinaOrani struct {
	Date string  `json:"date"`
	Oran float64 `json:"exchange_whale_ratio"`
}

// DestekliFiyat fiyat OHLCV'si olan varlıklar (ölçüldü).
var DestekliFiyat = map[string]bool{"btc": true, "eth": true, "xrp": true, "trx": true}

// limitSinir API'nin makul üst sınırı.
func limitSinir(limit int) int {
	if limit <= 0 {
		return 30
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

// Fiyat günlük OHLCV. Dönen dilim ESKİDEN YENİYE sıralıdır (grafik için).
func (c *Istemci) Fiyat(ctx context.Context, coin string, limit int) ([]Bar, error) {
	coin = strings.ToLower(strings.TrimSpace(coin))
	var z zarf[Bar]
	yol := "/" + url.PathEscape(coin) + "/market-data/price-ohlcv?window=day&limit=" + strconv.Itoa(limitSinir(limit))
	if err := c.al(ctx, yol, &z); err != nil {
		return nil, err
	}
	return tersBar(z.Result.Data), nil
}

// MVRV piyasa değeri / gerçekleşen değer (yalnız BTC).
func (c *Istemci) MVRV(ctx context.Context, limit int) ([]MVRV, error) {
	var z zarf[MVRV]
	yol := "/btc/market-indicator/mvrv?window=day&limit=" + strconv.Itoa(limitSinir(limit))
	if err := c.al(ctx, yol, &z); err != nil {
		return nil, err
	}
	d := z.Result.Data
	for i, j := 0, len(d)-1; i < j; i, j = i+1, j-1 {
		d[i], d[j] = d[j], d[i]
	}
	return d, nil
}

// SOPR harcanan çıktı kâr oranı (yalnız BTC).
func (c *Istemci) SOPR(ctx context.Context, limit int) ([]SOPR, error) {
	var z zarf[SOPR]
	yol := "/btc/market-indicator/sopr?window=day&limit=" + strconv.Itoa(limitSinir(limit))
	if err := c.al(ctx, yol, &z); err != nil {
		return nil, err
	}
	d := z.Result.Data
	for i, j := 0, len(d)-1; i < j; i, j = i+1, j-1 {
		d[i], d[j] = d[j], d[i]
	}
	return d, nil
}

// BorsaAkis borsa akış metriği. metrik: inflow | netflow | reserve.
//
// `exchange` ZORUNLU (API 400 döner). Varsayılan "binance": en likit borsa ve
// CoinDesk'in akış grafiklerinde referans olarak kullandığı borsa.
func (c *Istemci) BorsaAkis(ctx context.Context, coin, metrik, exchange string, limit int) ([]Akis, error) {
	coin = strings.ToLower(strings.TrimSpace(coin))
	switch metrik {
	case "inflow", "netflow", "reserve":
	default:
		return nil, fmt.Errorf("cryptoquant: geçersiz metrik %q", metrik)
	}
	if strings.TrimSpace(exchange) == "" {
		exchange = "binance"
	}
	var z zarf[Akis]
	yol := "/" + url.PathEscape(coin) + "/exchange-flows/" + metrik +
		"?exchange=" + url.QueryEscape(exchange) +
		"&window=day&limit=" + strconv.Itoa(limitSinir(limit))
	if err := c.al(ctx, yol, &z); err != nil {
		return nil, err
	}
	d := z.Result.Data
	for i, j := 0, len(d)-1; i < j; i, j = i+1, j-1 {
		d[i], d[j] = d[j], d[i]
	}
	return d, nil
}

// BorsaBalinaOrani borsa balina oranı (yalnız BTC).
func (c *Istemci) BorsaBalinaOrani(ctx context.Context, exchange string, limit int) ([]BalinaOrani, error) {
	if strings.TrimSpace(exchange) == "" {
		exchange = "binance"
	}
	var z zarf[BalinaOrani]
	yol := "/btc/flow-indicator/exchange-whale-ratio?exchange=" + url.QueryEscape(exchange) +
		"&window=day&limit=" + strconv.Itoa(limitSinir(limit))
	if err := c.al(ctx, yol, &z); err != nil {
		return nil, err
	}
	d := z.Result.Data
	for i, j := 0, len(d)-1; i < j; i, j = i+1, j-1 {
		d[i], d[j] = d[j], d[i]
	}
	return d, nil
}

func tersBar(d []Bar) []Bar {
	for i, j := 0, len(d)-1; i < j; i, j = i+1, j-1 {
		d[i], d[j] = d[j], d[i]
	}
	return d
}
