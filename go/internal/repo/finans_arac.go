package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"theblock/site/internal/cryptoicon"
)

// KurTablosu çevirici için para birimi kurları (1 birim = X TRY).
type KurTablosu struct {
	Kurlar  map[string]float64 `json:"kurlar"`
	Adlar   map[string]string  `json:"adlar"`
	Updated time.Time          `json:"updated"`
}

// kurAdları çeviricide gösterilen Türkçe adlar.
var kurAdları = map[string]string{
	"TRY": "Türk Lirası",
	"USD": "Amerikan Doları",
	"EUR": "Euro",
	"GBP": "İngiliz Sterlini",
	"JPY": "Japon Yeni",
	"AUD": "Avustralya Doları",
	"CAD": "Kanada Doları",
	"CHF": "İsviçre Frangı",
	"DKK": "Danimarka Kronu",
	"NOK": "Norveç Kronu",
	"SEK": "İsveç Kronu",
	"SAR": "Suudi Riyali",
	"AED": "BAE Dirhemi",
	"RUB": "Rus Rublesi",
	"CNY": "Çin Yuanı",
	"KWD": "Kuveyt Dinarı",
	"QAR": "Katar Riyali",
	"KRW": "Güney Kore Wonu",
	"INR": "Hindistan Rupisi",
	"MXN": "Meksika Pesosu",
	"BRL": "Brezilya Reali",
	"ZAR": "Güney Afrika Randı",
	"PLN": "Polonya Zlotisi",
	"CZK": "Çek Korunası",
	"HUF": "Macar Forinti",
	"RON": "Rumen Leyi",
	"NZD": "Yeni Zelanda Doları",
	"SGD": "Singapur Doları",
	"HKD": "Hong Kong Doları",
	"ILS": "İsrail Şekeli",
}

// KurSiralama çevirici listelerinin varsayılan sırası (referans düzeni).
var KurSiralama = []string{
	"USD", "EUR", "TRY", "GBP", "JPY", "AUD", "CAD", "CHF", "DKK", "NOK", "SEK", "SAR",
}

// KurSembolleri tablodaki kodları referans sırasıyla döndürür; tabloda
// olmayan kodlar atlanır, tablodaki fazlalar sona eklenir.
func (k KurTablosu) Kodlar() []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(k.Kurlar))
	for _, c := range KurSiralama {
		if r, ok := k.Kurlar[c]; ok && r > 0 {
			out = append(out, c)
			seen[c] = true
		}
	}
	// Kalanlar alfabetik (referans dışı para birimleri de seçilebilsin).
	var kalan []string
	for c, r := range k.Kurlar {
		if !seen[c] && r > 0 {
			kalan = append(kalan, c)
		}
	}
	// basit sıralama (küçük liste)
	for i := 1; i < len(kalan); i++ {
		for j := i; j > 0 && kalan[j-1] > kalan[j]; j-- {
			kalan[j-1], kalan[j] = kalan[j], kalan[j-1]
		}
	}
	return append(out, kalan...)
}

// Ad para biriminin görünen adını verir (yoksa kodun kendisi).
func (k KurTablosu) Ad(code string) string {
	if a, ok := k.Adlar[code]; ok && a != "" {
		return a
	}
	if a, ok := kurAdları[code]; ok {
		return a
	}
	return code
}

// Cevir amount birimini from'dan to'ya çevirir (TRY üzerinden).
func (k KurTablosu) Cevir(amount float64, from, to string) (float64, bool) {
	f, ok1 := k.Kurlar[strings.ToUpper(from)]
	t, ok2 := k.Kurlar[strings.ToUpper(to)]
	if !ok1 || !ok2 || f <= 0 || t <= 0 {
		return 0, false
	}
	return amount * f / t, true
}

// kurlarCacheKey önbellek anahtarı.
const kurlarCacheKey = "ana"

// truncgilKur truncgil v4 para birimi satırı.
type truncgilKur struct {
	Selling float64 `json:"Selling"`
	Buying  float64 `json:"Buying"`
	Type    string  `json:"Type"`
}

// deger gösterilecek değeri seçer (satış varsa satış).
func (k truncgilKur) deger() (float64, bool) {
	if k.Selling > 0 {
		return k.Selling, true
	}
	if k.Buying > 0 {
		return k.Buying, true
	}
	return 0, false
}

func fetchKurTablosuKaynak(ctx context.Context) (map[string]truncgilKur, error) {
	raw, err := httpGetJSONUA("https://finans.truncgil.com/v4/today.json", 12*time.Second)
	if err != nil {
		return nil, err
	}
	var ham map[string]json.RawMessage
	if err := json.Unmarshal(raw, &ham); err != nil {
		return nil, err
	}
	out := make(map[string]truncgilKur, len(ham))
	for code, v := range ham {
		var k truncgilKur
		if json.Unmarshal(v, &k) != nil {
			continue
		}
		if k.Type == "" {
			continue
		}
		out[strings.ToUpper(code)] = k
	}
	return out, nil
}

// RefreshKurlar truncgil'den tüm para birimlerini çekip önbelleğe yazar.
func RefreshKurlar(ctx context.Context, pool DB) error {
	m, err := fetchKurTablosuKaynak(ctx)
	if err != nil || len(m) == 0 {
		if err == nil {
			err = fmt.Errorf("kurlar: boş yanıt")
		}
		return err
	}
	kurlar := map[string]float64{"TRY": 1}
	for code, k := range m {
		if !strings.EqualFold(k.Type, "Currency") {
			continue
		}
		v, ok := k.deger()
		if !ok || v <= 0 {
			continue
		}
		code = strings.ToUpper(code)
		// JPY 100 birim olarak gelirse 1 birime indir (bugün 1 birim geliyor).
		if code == "JPY" && v > 1 {
			v /= 100
		}
		kurlar[code] = v
	}
	if len(kurlar) < 3 {
		return fmt.Errorf("kurlar: yalnız %d para birimi alındı", len(kurlar))
	}
	adlar := make(map[string]string, len(kurlar))
	for c := range kurlar {
		adlar[c] = kurAdları[c]
	}
	b, err := json.Marshal(KurTablosu{Kurlar: kurlar, Adlar: adlar, Updated: time.Now()})
	if err != nil {
		return err
	}
	return CacheSet(ctx, pool, "kurlar", kurlarCacheKey, b)
}

// KurlarOf önbellekteki kur tablosunu döndürür.
func KurlarOf(ctx context.Context, pool DB) KurTablosu {
	var kt KurTablosu
	if b, err := CacheGet(ctx, pool, "kurlar", kurlarCacheKey); err == nil {
		_ = json.Unmarshal(b, &kt)
	}
	if kt.Kurlar == nil {
		kt.Kurlar = map[string]float64{}
	}
	if kt.Adlar == nil {
		kt.Adlar = map[string]string{}
	}
	return kt
}

// ---- Kripto para (Kripto Para modülü) ----

type KriptoRow struct {
	Symbol    string  `json:"symbol"`     // BTC
	Name      string  `json:"name"`       // Bitcoin
	Slug      string  `json:"slug"`       // bitcoin (kanonik adres)
	Image     string  `json:"image"`      // ikon URL
	Value     float64 `json:"value"`      // fiyat (TRY)
	Delta     float64 `json:"delta"`      // 24s %
	Delta7    float64 `json:"delta7"`     // 7g %
	Volume    float64 `json:"volume"`     // 24s hacim (TRY)
	MarketCap float64 `json:"market_cap"` // piyasa değeri (TRY)
}

// KriptoData kripto para önbelleği.
type KriptoData struct {
	Rows    []KriptoRow `json:"rows"`
	Updated time.Time   `json:"updated"`
}

const kriptoMarkets = "https://api.coingecko.com/api/v3/coins/markets?vs_currency=try&order=market_cap_desc&per_page=100&page=1&price_change_percentage=24h,7d"

// kriptoSlug CoinGecko kimliğinden adres parçası üretir (kanonik adres).
func kriptoSlug(id, symbol string) string {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		return strings.ToLower(symbol)
	}
	return id
}

func RefreshKripto(ctx context.Context, pool DB) error {
	raw, err := httpGetJSONUA(kriptoMarkets, 15*time.Second)
	if err != nil {
		return err
	}
	var arr []struct {
		ID        string  `json:"id"`
		Symbol    string  `json:"symbol"`
		Name      string  `json:"name"`
		Price     float64 `json:"current_price"`
		Chg24     float64 `json:"price_change_percentage_24h"`
		Chg7      float64 `json:"price_change_percentage_7d_in_currency"`
		Volume    float64 `json:"total_volume"`
		MarketCap float64 `json:"market_cap"`
	}
	if err := json.Unmarshal(raw, &arr); err != nil {
		return err
	}
	rows := make([]KriptoRow, 0, len(arr))
	for _, c := range arr {
		if c.Price <= 0 || c.Symbol == "" {
			continue
		}
		rows = append(rows, KriptoRow{
			Symbol:    strings.ToUpper(c.Symbol),
			Name:      c.Name,
			Slug:      kriptoSlug(c.ID, c.Symbol),
			Image:     cryptoicon.URL(c.Symbol),
			Value:     c.Price,
			Delta:     c.Chg24,
			Delta7:    c.Chg7,
			Volume:    c.Volume,
			MarketCap: c.MarketCap,
		})
	}
	if len(rows) == 0 {
		return fmt.Errorf("kripto: boş liste")
	}
	b, err := json.Marshal(KriptoData{Rows: rows, Updated: time.Now()})
	if err != nil {
		return err
	}
	return CacheSet(ctx, pool, "kripto", kriptoCacheKey, b)
}

// kriptoCacheKey kripto önbellek anahtarı.
const kriptoCacheKey = "ana"

func KriptoOf(ctx context.Context, pool DB) KriptoData {
	var kd KriptoData
	if b, err := CacheGet(ctx, pool, "kripto", kriptoCacheKey); err == nil {
		_ = json.Unmarshal(b, &kd)
	}
	for i := range kd.Rows {
		kd.Rows[i].Image = cryptoicon.URL(kd.Rows[i].Symbol)
	}
	return kd
}
