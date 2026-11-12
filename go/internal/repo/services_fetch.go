package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/platform"
)

// WeatherInfo hava durumu özetidir (önbellek prototipi).
type WeatherInfo struct {
	City     string  `json:"city"`
	Sicaklik float64 `json:"sicaklik"`
	Aciklama string  `json:"aciklama"`
	Demo     bool    `json:"demo"`
}

// Kur bilgi satırı (döviz bandı).
type Kur struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Delta float64 `json:"delta"`
}

const (
	openMeteo    = "https://api.open-meteo.com/v1/forecast?latitude=%v&longitude=%v&current=temperature_2m,weather_code,wind_speed_10m&timezone=auto"
	floatRates   = "https://www.floatrates.com/daily/%s.json"
	aladhanDaily = "https://api.aladhan.com/v1/timings/%d-%d-%d?latitude=%v&longitude=%v&method=13"
	aladhanMonth = "https://api.aladhan.com/v1/calendar/%d/%d?latitude=%v&longitude=%v&method=13"
)

func httpGetJSON(url string) ([]byte, error) {
	return httpGetJSONT(url, 9*time.Second)
}

func httpGetJSONT(url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

// RefreshWeather 81 ile hava çekip önbelleğe yazar; dışarı kapalıysa demo satırı.
func RefreshWeather(ctx context.Context, pool DB) (int, error) {
	if err := EnsureCities(ctx, pool); err != nil {
		return 0, err
	}
	cities, err := ListCities(ctx, pool)
	if err != nil {
		return 0, err
	}
	ok := 0
	for _, c := range cities {
		demo, _ := json.Marshal(WeatherInfo{City: c.Name, Sicaklik: 21.0, Aciklama: "hava verisi beklemede (demo)", Demo: true})
		_ = CacheSet(ctx, pool, "hava", strconv.Itoa(c.Plaka), demo)
		raw, err := httpGetJSONT(fmt.Sprintf(openMeteo, c.Lat, c.Lon), 8*time.Second)
		if err != nil {
			continue
		}
		// current.temperature_2m çıkar
		var cur struct {
			Current struct {
				Temp float64 `json:"temperature_2m"`
				Code int     `json:"weather_code"`
			} `json:"current"`
		}
		if json.Unmarshal(raw, &cur) == nil {
			info, _ := json.Marshal(WeatherInfo{City: c.Name, Sicaklik: cur.Current.Temp, Aciklama: weatherCode(cur.Current.Code)})
			_ = CacheSet(ctx, pool, "hava", strconv.Itoa(c.Plaka), info)
			ok++
		}
	}
	return ok, nil
}

func weatherCode(c int) string {
	switch {
	case c == 0:
		return "Açık"
	case c <= 3:
		return "Az bulutlu"
	case c <= 48:
		return "Bulutlu"
	case c <= 67 || c <= 82:
		return "Yağışlı"
	default:
		return "Kapalı"
	}
}

func RefreshForex(ctx context.Context, pool DB) error {
	out := map[string]float64{}
	for _, k := range PiyasaOf(ctx, pool).Kalemler {
		if k.Demo || k.Deger <= 0 {
			continue
		}
		switch k.Anahtar {
		case "usd":
			out["usd"] = k.Deger
		case "eur":
			out["eur"] = k.Deger
		case "gram-altin":
			out["altin"] = k.Deger
		}
	}
	// Yedek: floatrates (yalnız piyasa önbelleği henüz dolmamışsa).
	if out["usd"] <= 0 || out["eur"] <= 0 {
		for _, code := range []string{"usd", "eur"} {
			if out[code] > 0 {
				continue
			}
			if raw, err := httpGetJSON(fmt.Sprintf(floatRates, code)); err == nil {
				var m map[string]struct {
					Rate float64 `json:"rate"`
				}
				if json.Unmarshal(raw, &m) == nil {
					if t, ok := m["try"]; ok {
						out[code] = t.Rate
					}
				}
			}
		}
	}
	demo := out["usd"] <= 0 || out["eur"] <= 0
	row := map[string]any{"dolar": out["usd"], "euro": out["eur"], "altin": out["altin"], "demo": demo}
	b, _ := json.Marshal(row)
	return CacheSet(ctx, pool, "kur", "ana", b)
}

// NamazVakit map'si: "Imsak" → "HH:MM" vb.
type NamazVakit struct {
	Gun    int    `json:"gun"`
	Imsak  string `json:"imsak"`
	Gunes  string `json:"gunes"`
	Ogle   string `json:"ogle"`
	Ikindi string `json:"ikindi"`
	Aksam  string `json:"aksam"`
	Yatsi  string `json:"yatsi"`
	Il     string `json:"il"`
	Demo   bool   `json:"demo"`
}

// RefreshNamaz 81 il için bugünün vakitlerini çeker.
func RefreshNamaz(ctx context.Context, pool DB) (int, error) {
	if err := EnsureCities(ctx, pool); err != nil {
		return 0, err
	}
	cities, err := ListCities(ctx, pool)
	if err != nil {
		return 0, err
	}
	ok := 0
	d := time.Now()
	for _, c := range cities {
		// önce demo yaz (sayfa anında dolu), gerçek gelirse üzerine yazar
		demo, _ := json.Marshal(NamazVakit{Il: c.Name, Imsak: "05:12", Gunes: "06:40", Ogle: "12:50", Ikindi: "15:55", Aksam: "18:45", Yatsi: "20:05", Demo: true})
		_ = CacheSet(ctx, pool, "namaz", strconv.Itoa(c.Plaka), demo)
		raw, err := httpGetJSONT(fmt.Sprintf(aladhanDaily, d.Day(), int(d.Month()), d.Year(), c.Lat, c.Lon), 5*time.Second)
		if err != nil {
			continue
		}
		var resp struct {
			Data struct {
				Timings map[string]string `json:"timings"`
			} `json:"data"`
		}
		if json.Unmarshal(raw, &resp) == nil && len(resp.Data.Timings) > 0 {
			t := resp.Data.Timings
			trim := func(s string) string {
				if len(s) >= 5 {
					return s[:5]
				}
				return s
			}
			nv, _ := json.Marshal(NamazVakit{Il: c.Name,
				Imsak: trim(t["Imsak"]), Gunes: trim(t["Sunrise"]), Ogle: trim(t["Dhuhr"]),
				Ikindi: trim(t["Asr"]), Aksam: trim(t["Maghrib"]), Yatsi: trim(t["Isha"])})
			_ = CacheSet(ctx, pool, "namaz", strconv.Itoa(c.Plaka), nv)
			ok++
		} else {
			demo, _ := json.Marshal(NamazVakit{Il: c.Name, Demo: true})
			_ = CacheSet(ctx, pool, "namaz", strconv.Itoa(c.Plaka), demo)
		}
	}
	return ok, nil
}

// RefreshImsakiye belirli şehir için aylık takvimi önbelleğe yazar.
func RefreshImsakiye(ctx context.Context, pool DB, plaka int) error {
	c, err := GetCityByPlaka(ctx, pool, plaka)
	if err != nil {
		return err
	}
	d := time.Now()
	raw, err := httpGetJSONT(fmt.Sprintf(aladhanMonth, d.Year(), int(d.Month()), c.Lat, c.Lon), 8*time.Second)
	if err != nil {
		demo, _ := json.Marshal([]NamazVakit{{Gun: 1, Il: c.Name, Imsak: "05:12", Demo: true}})
		return CacheSet(ctx, pool, "imsakiye", strconv.Itoa(plaka), demo)
	}
	var resp struct {
		Data []struct {
			Timings map[string]string `json:"timings"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &resp) != nil || len(resp.Data) == 0 {
		return err
	}
	var list []NamazVakit
	for i, it := range resp.Data {
		t := it.Timings
		trim := func(s string) string {
			if len(s) >= 5 {
				return s[:5]
			}
			return s
		}
		nv := NamazVakit{Il: c.Name, Gun: i + 1,
			Imsak: trim(t["Imsak"]), Gunes: trim(t["Sunrise"]), Ogle: trim(t["Dhuhr"]),
			Ikindi: trim(t["Asr"]), Aksam: trim(t["Maghrib"]), Yatsi: trim(t["Isha"])}
		list = append(list, nv)
	}
	b, _ := json.Marshal(list)
	return CacheSet(ctx, pool, "imsakiye", strconv.Itoa(plaka), b)
}

// Eczane nöbetçi eczane kaydıdır.
type Eczane struct {
	Ilce  string `json:"ilce"`
	Ad    string `json:"ad"`
	Adres string `json:"adres"`
	Tel   string `json:"tel"`
}

// EczaneSehir bir şehrin nöbetçi eczane listesidir.
type EczaneSehir struct {
	Plaka int    `json:"plaka"`
	Sehir string `json:"sehir"`
	Demo  bool   `json:"demo"`
	Tarih string `json:"tarih"`
	// Kaynak: "ozel" | "eczaneler.org" | "demo". Sayfada hangi kaynağın
	// kullanıldığı görünsün diye saklanır.
	Kaynak string `json:"kaynak"`
	// Saat verinin çekildiği saat (HH:MM). "Boş" ilde kaynak sonradan
	// yayınlayabilir; 2 saatte bir yeniden denenir.
	Saat string   `json:"saat,omitempty"`
	Bos  bool     `json:"bos,omitempty"`
	List []Eczane `json:"list"`
}

func RefreshEczane(ctx context.Context, pool DB) (int, error) {
	if err := EnsureCities(ctx, pool); err != nil {
		return 0, err
	}
	src := ""
	if b, err := GetSetting(ctx, pool, "servis", "eczane_kaynak"); err == nil {
		src = strings.Trim(string(b), `"`)
	}
	// NosyAPI anahtarı (servis/eczane_api_key). Boşsa kaynak kapalı.
	apiAnahtar := ""
	if b, err := GetSetting(ctx, pool, "servis", "eczane_api_key"); err == nil {
		apiAnahtar = strings.Trim(string(b), `"`)
	}
	// Devre kesici: anahtar geçersizse her il için 25 sn beklemek yerine
	// ilk birkaç hatadan sonra API bu turda kapatılır.
	apiArdardaHata := 0
	apiKapali := false
	// eczaneler.org küme çıkış IP'sinden 403 alıyor; 81 il × 20 sn
	// beklememek için aynı devre kesici orada da var.
	orgArdardaHata := 0
	orgKapali := false
	cities, err := ListCities(ctx, pool)
	if err != nil {
		return 0, err
	}
	bugun := time.Now().Format("2006-01-02")
	ok := 0
	for _, c := range cities {
		if ctx.Err() != nil {
			//nolint:nilerr // bağlam iptal edildi: tur normal biçimde biter, hata değildir
			return ok, nil
		}
		anahtar := strconv.Itoa(c.Plaka)

		if b, err := CacheGet(ctx, pool, "eczane", anahtar); err == nil && len(b) > 0 {
			var mevcut EczaneSehir
			if json.Unmarshal(b, &mevcut) == nil && mevcut.Tarih == bugun {
				if !mevcut.Demo && !mevcut.Bos && len(mevcut.List) > 0 {
					ok++
					continue
				}
				if mevcut.Bos && saatIkiSaatIcinde(mevcut.Saat) {
					continue
				}
			}
		}

		demo := true
		var list []Eczane
		kaynak := ""

		// 1) Özel kaynak.
		if src != "" {
			u := strings.ReplaceAll(src, "{plaka}", strconv.Itoa(c.Plaka))
			u = strings.ReplaceAll(u, "{ilce}", "")
			if raw, err := httpGetJSONT(u, 8*time.Second); err == nil {
				var parsed struct {
					Ilce      string   `json:"ilce"`
					Eczaneler []Eczane `json:"eczaneler"`
				}
				if json.Unmarshal(raw, &parsed) == nil && len(parsed.Eczaneler) > 0 {
					list = eczaneTekrarsiz(parsed.Eczaneler)
					if list[0].Ilce == "" {
						for i := range list {
							list[i].Ilce = parsed.Ilce
						}
					}
					demo = false
					kaynak = "ozel"
				}
			}
		}

		if list == nil {
			if oda, ok := eczaneOdaKaynaklari[c.Plaka]; ok {
				if l, ad, err := oda(ctx); err == nil && len(l) > 0 {
					list = l
					demo = false
					kaynak = ad
				}
			}
		}

		apiHazir := false
		if list == nil && apiAnahtar != "" && !apiKapali && c.Slug != "" {
			l, hazir := eczaneAPISehir(ctx, pool, apiAnahtar, c.Slug, c.Plaka)
			if hazir {
				apiArdardaHata = 0
				apiHazir = true
				kaynak = eczaneRapidAPIKaynak
				if len(l) > 0 {
					list = l
					demo = false
				}
			} else {
				apiArdardaHata++
				if apiArdardaHata >= 3 {
					apiKapali = true
				}
			}
		}

		if list == nil && !apiHazir && !orgKapali && c.Slug != "" {
			if l, err := eczaneEczanelerOrg(ctx, c.Slug); err == nil {
				orgArdardaHata = 0
				kaynak = "eczaneler.org"
				if len(l) > 0 {
					list = l
					demo = false
				}
			} else {
				orgArdardaHata++
				if orgArdardaHata >= 3 {
					orgKapali = true
				}
			}
		}

		bos := false
		if demo {
			switch {
			case apiHazir:
				bos = true
				demo = false
				list = nil
				kaynak = eczaneRapidAPIKaynak
			case kaynak == "eczaneler.org":
				bos = true
				demo = false
				list = nil
			default:
				bos = true
				demo = false
				list = nil
				kaynak = "ulasilamadi"
			}
		} else {
			ok++
		}

		b, _ := json.Marshal(EczaneSehir{
			Plaka: c.Plaka, Sehir: c.Name, Demo: demo,
			Tarih: bugun, Saat: time.Now().Format("15:04"), Kaynak: kaynak,
			Bos: bos, List: list,
		})
		_ = CacheSet(ctx, pool, "eczane", anahtar, b)

		// Kaba olmamak için iller arasında kısa bekleme (yalnız ağ turunda).
		if kaynak == "eczaneler.org" {
			select {
			case <-ctx.Done():
				return ok, nil
			case <-time.After(400 * time.Millisecond):
			}
		}
	}
	return ok, nil
}

func EczaneOf(ctx context.Context, pool DB, plaka string) (EczaneSehir, error) {
	var es EczaneSehir
	b, err := CacheGet(ctx, pool, "eczane", plaka)
	if err != nil {
		return es, err
	}
	_ = json.Unmarshal(b, &es)
	es.List = eczaneTekrarsiz(es.List)
	return es, nil
}

// ---- Borsa (P3-5): BIST endeks/hisseler + kripto ----

// BorsaRow tek bir piyasa satırı (endeks, hisse veya kripto).
type BorsaRow struct {
	Kind  string  `json:"kind"`  // endeks | hisse | kripto
	Code  string  `json:"code"`  // XU100, THYAO, BTC
	Name  string  `json:"name"`  // BIST 100, Türk Hava Yolları, Bitcoin
	Value float64 `json:"value"` // fiyat
	Delta float64 `json:"delta"` // günlük % değişim
	Para  string  `json:"para"`  // TRY | USD
}

// BorsaData piyasa özeti (önbellekte saklanır).
type BorsaData struct {
	Rows    []BorsaRow `json:"rows"`
	Demo    bool       `json:"demo"`
	Updated time.Time  `json:"updated"`
}

const coingeckoMarkets = "https://api.coingecko.com/api/v3/coins/markets?vs_currency=try&order=market_cap_desc&per_page=8&page=1&ids=bitcoin,ethereum,ripple,binancecoin,solana,cardano,dogecoin,avalanche-2&price_change_percentage=24h"

// yahooChart Yahoo Finance grafik ucu (BIST için .IS sembolleri).
const yahooChart = "https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d"

var bistList = []struct {
	Code, Name, Kind string
}{
	{"XU100.IS", "BIST 100", "endeks"},
	{"THYAO.IS", "Türk Hava Yolları", "hisse"},
	{"ASELS.IS", "Aselsan", "hisse"},
	{"GARAN.IS", "Garanti BBVA", "hisse"},
	{"AKBNK.IS", "Akbank", "hisse"},
	{"EREGL.IS", "Ereğli Demir Çelik", "hisse"},
	{"TUPRS.IS", "Tüpraş", "hisse"},
	{"BIMAS.IS", "BİM", "hisse"},
	{"KCHOL.IS", "Koç Holding", "hisse"},
	{"SISE.IS", "Şişecam", "hisse"},
}

// httpGetJSONUA User-Agent gerektiren uçlar (Yahoo Finance) için.
func httpGetJSONUA(url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; theblock/1.0)")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

// RefreshBorsa BIST (Yahoo Finance) + kripto (CoinGecko) verisini önbelleğe yazar.
// Uçlardan biri erişilemezse elde olanla devam eder; hiç veri yoksa demo üretir.
func RefreshBorsa(ctx context.Context, pool DB) error {
	var rows []BorsaRow

	// kripto (TRY)
	if raw, err := httpGetJSONUA(coingeckoMarkets, 12*time.Second); err == nil {
		var arr []struct {
			Name    string  `json:"name"`
			Symbol  string  `json:"symbol"`
			Current float64 `json:"current_price"`
			Chg     float64 `json:"price_change_percentage_24h"`
		}
		if json.Unmarshal(raw, &arr) == nil {
			for _, c := range arr {
				if c.Current == 0 {
					continue
				}
				rows = append(rows, BorsaRow{Kind: "kripto", Code: strings.ToUpper(c.Symbol), Name: c.Name, Value: c.Current, Delta: c.Chg, Para: "TRY"})
			}
		}
	}

	// BIST endeks + hisseler (paralel — gecikmeyi azaltır)
	bistRows := make([]*BorsaRow, len(bistList))
	var wg sync.WaitGroup
	for i, b := range bistList {
		wg.Add(1)
		go func(i int, code, name, kind string) {
			defer wg.Done()
			raw, err := httpGetJSONUA(fmt.Sprintf(yahooChart, code), 8*time.Second)
			if err != nil {
				return
			}
			var resp struct {
				Chart struct {
					Result []struct {
						Meta struct {
							RegularMarketPrice float64 `json:"regularMarketPrice"`
							ChangePercent      float64 `json:"regularMarketChangePercent"`
						} `json:"meta"`
					} `json:"result"`
				} `json:"chart"`
			}
			if json.Unmarshal(raw, &resp) != nil || len(resp.Chart.Result) == 0 {
				return
			}
			m := resp.Chart.Result[0].Meta
			if m.RegularMarketPrice == 0 {
				return
			}
			row := BorsaRow{Kind: kind, Code: strings.TrimSuffix(code, ".IS"), Name: name, Value: m.RegularMarketPrice, Delta: m.ChangePercent, Para: "TRY"}
			bistRows[i] = &row
		}(i, b.Code, b.Name, b.Kind)
	}
	wg.Wait()
	for _, r := range bistRows {
		if r != nil {
			rows = append(rows, *r)
		}
	}

	// Ons altın (XAU) — Yahoo GC=F (COMEX altın vadeli, USD) — coinhit yaklaşımı
	if raw, err := httpGetJSONUA(fmt.Sprintf(yahooChart, "GC=F"), 8*time.Second); err == nil {
		var resp struct {
			Chart struct {
				Result []struct {
					Meta struct {
						RegularMarketPrice float64 `json:"regularMarketPrice"`
						ChangePercent      float64 `json:"regularMarketChangePercent"`
					} `json:"meta"`
				} `json:"result"`
			} `json:"chart"`
		}
		if json.Unmarshal(raw, &resp) == nil && len(resp.Chart.Result) > 0 {
			m := resp.Chart.Result[0].Meta
			if m.RegularMarketPrice > 0 {
				rows = append(rows, BorsaRow{Kind: "altin_ons", Code: "ONS", Name: "Ons Altın", Value: m.RegularMarketPrice, Delta: m.ChangePercent, Para: "USD"})
			}
		}
	}

	demo := len(rows) == 0
	if demo {
		rows = demoBorsa()
	}
	b, _ := json.Marshal(BorsaData{Rows: rows, Demo: demo, Updated: time.Now()})
	return CacheSet(ctx, pool, "borsa", "ana", b)
}

// BorsaOf önbellekteki piyasa özetini döndürür.
func BorsaOf(ctx context.Context, pool DB) BorsaData {
	var bd BorsaData
	if b, err := CacheGet(ctx, pool, "borsa", "ana"); err == nil {
		_ = json.Unmarshal(b, &bd)
	}
	return bd
}

func demoBorsa() []BorsaRow {
	return []BorsaRow{
		{Kind: "endeks", Code: "XU100", Name: "BIST 100", Value: 12218.49, Delta: -0.26, Para: "TRY"},
		{Kind: "hisse", Code: "THYAO", Name: "Türk Hava Yolları", Value: 283.0, Delta: -1.14, Para: "TRY"},
		{Kind: "kripto", Code: "BTC", Name: "Bitcoin", Value: 4118572, Delta: 0.15, Para: "TRY"},
	}
}

// HisseRow tek hisse satırı.
type HisseRow struct {
	Code  string  `json:"code"`  // THYAO
	Name  string  `json:"name"`  // Türk Hava Yolları
	Value float64 `json:"value"` // fiyat (TRY)
	Delta float64 `json:"delta"` // günlük % değişim
}

// HisseData hisse şeridi önbelleği.
type HisseData struct {
	Rows    []HisseRow `json:"rows"`
	Updated time.Time  `json:"updated"`
}

// HisseMeta hisse evrenindeki tek kayıt (kod + kısa ad).
type HisseMeta struct {
	Code string
	Name string
}

func HisseEvreni() []HisseMeta {
	out := make([]HisseMeta, len(bistEvreni))
	copy(out, bistEvreni)
	return out
}

// HisseSembolleri şeritte izlenen kodları döndürür (test/teşhis).
func HisseSembolleri() []string {
	out := make([]string, 0, len(bistEvreni))
	for _, h := range bistEvreni {
		out = append(out, h.Code)
	}
	return out
}

// hisseAd koddan görünen adı bulur (evrende yoksa kodun kendisi).
func hisseAd(code string) string {
	for _, h := range bistEvreni {
		if h.Code == code {
			return h.Name
		}
	}
	return code
}

const hisseSparkURL = "https://query1.finance.yahoo.com/v8/finance/spark?symbols=%s&range=1d&interval=1d"

var (
	hisseSparkBase  = hisseSparkURL
	hisseChunkSize  = 18
	hisseConcurrent = 4
)

// hisseSparkParça bir spark yanıtındaki tek sembolün fiyat/değişimini okur.
type hisseSparkParça struct {
	Close              []float64 `json:"close"`
	FulldayPrice       float64   `json:"fulldayPrice"`
	FulldayChangePct   *float64  `json:"fulldayChangePercent"`
	ChartPreviousClose float64   `json:"chartPreviousClose"`
}

func (p hisseSparkParça) fiyat() (float64, float64, bool) {
	fiyat := 0.0
	if len(p.Close) > 0 && p.Close[0] > 0 {
		fiyat = p.Close[0]
	} else if p.FulldayPrice > 0 {
		fiyat = p.FulldayPrice
	}
	if fiyat <= 0 {
		return 0, 0, false
	}
	if p.FulldayChangePct != nil {
		return fiyat, *p.FulldayChangePct, true
	}
	if p.ChartPreviousClose > 0 {
		return fiyat, (fiyat - p.ChartPreviousClose) / p.ChartPreviousClose * 100, true
	}
	return fiyat, 0, true
}

// fetchHisseChunk bir sembol grubunu tek istekte okur.
func fetchHisseChunk(codes []string, timeout time.Duration) map[string]HisseRow {
	if len(codes) == 0 {
		return nil
	}
	liste := make([]string, 0, len(codes))
	for _, c := range codes {
		liste = append(liste, c+".IS")
	}
	raw, err := httpGetJSONUA(fmt.Sprintf(hisseSparkBase, strings.Join(liste, ",")), timeout)
	if err != nil {
		return nil
	}
	var resp map[string]hisseSparkParça
	if json.Unmarshal(raw, &resp) != nil {
		return nil
	}
	out := make(map[string]HisseRow, len(resp))
	for sym, parca := range resp {
		code := strings.TrimSuffix(sym, ".IS")
		v, d, ok := parca.fiyat()
		if !ok {
			continue
		}
		out[code] = HisseRow{Code: code, Name: hisseAd(code), Value: v, Delta: d}
	}
	return out
}

func RefreshHisseler(ctx context.Context, pool DB) error {
	codes := make([]string, 0, len(bistEvreni))
	for _, h := range bistEvreni {
		codes = append(codes, h.Code)
	}

	// Parça sayısı kadar yer ayır; her goroutine kendi indeksini doldurur.
	parcalar := make([]map[string]HisseRow, (len(codes)+hisseChunkSize-1)/hisseChunkSize)

	var wg sync.WaitGroup
	sem := make(chan struct{}, hisseConcurrent)
	for i := 0; i < len(codes); i += hisseChunkSize {
		end := i + hisseChunkSize
		if end > len(codes) {
			end = len(codes)
		}
		idx := i / hisseChunkSize
		wg.Add(1)
		go func(idx int, chunk []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			parcalar[idx] = fetchHisseChunk(chunk, 12*time.Second)
		}(idx, codes[i:end])
	}
	wg.Wait()

	taze := make(map[string]HisseRow, len(codes))
	for _, p := range parcalar {
		for code, row := range p {
			taze[code] = row
		}
	}
	if len(taze) == 0 {
		return fmt.Errorf("hisseler: hiçbir sembol alınamadı")
	}

	// Eksik sembolleri eski önbellekten taşı.
	eski := HisselerOf(ctx, pool)
	byCode := make(map[string]HisseRow, len(eski.Rows))
	for _, r := range eski.Rows {
		byCode[r.Code] = r
	}
	birlesik := make([]HisseRow, 0, len(codes))
	for _, c := range codes {
		if r, ok := taze[c]; ok {
			birlesik = append(birlesik, r)
		} else if old, ok := byCode[c]; ok {
			birlesik = append(birlesik, old)
		}
	}

	b, err := json.Marshal(HisseData{Rows: birlesik, Updated: time.Now()})
	if err != nil {
		return err
	}
	if err := CacheSet(ctx, pool, "hisseler", "ana", b); err != nil {
		return err
	}
	if len(taze) < len(codes) {
		return fmt.Errorf("hisseler: %d/%d sembol alındı", len(taze), len(codes))
	}
	return nil
}

// HisselerOf önbellekteki hisse şeridini döndürür.
func HisselerOf(ctx context.Context, pool DB) HisseData {
	var hd HisseData
	if b, err := CacheGet(ctx, pool, "hisseler", "ana"); err == nil {
		_ = json.Unmarshal(b, &hd)
	}
	return hd
}

// ---- Fikstür (P3-6): Süper Lig maç programı + sonuçlar (TheSportsDB) ----

// FiksturMac tek bir futbol maç satırı.
type FiksturMac struct {
	Tarih     string `json:"tarih"`
	Ev        string `json:"ev"`
	Deplasman string `json:"deplasman"`
	EvSkor    *int   `json:"ev_skor"`
	DepSkor   *int   `json:"dep_skor"`
	Lig       string `json:"lig"`
	Tur       string `json:"tur"` // gelecek | sonuc
	Hafta     string `json:"hafta"`
}

// FiksturData fikstür özeti (önbellekte saklanır).
type FiksturData struct {
	Upcoming []FiksturMac `json:"upcoming"`
	Results  []FiksturMac `json:"results"`
	Demo     bool         `json:"demo"`
	Updated  time.Time    `json:"updated"`
}

var tsAPI = "https://www.thesportsdb.com/api/v1/json/%s/%s"

type tsEvent struct {
	Date      string `json:"dateEvent"`
	Time      string `json:"strTime"`
	Home      string `json:"strHomeTeam"`
	Away      string `json:"strAwayTeam"`
	HomeScore string `json:"intHomeScore"`
	AwayScore string `json:"intAwayScore"`
	League    string `json:"strLeague"`
	Season    string `json:"strSeason"`
	Round     string `json:"intRound"`
}

func tsEvents(key, path string, timeout time.Duration) []tsEvent {
	raw, err := httpGetJSONUA(fmt.Sprintf(tsAPI, key, path), timeout)
	if err != nil {
		return nil
	}
	var resp struct {
		Events []tsEvent `json:"events"`
	}
	if json.Unmarshal(raw, &resp) != nil {
		return nil
	}
	return resp.Events
}

// RefreshFikstur Süper Lig fikstürü ve sonuçlarını çeker.
// Önce yaklaşan/geçmiş maçtan hafta numarasını okur, o hafta(lar)ın tam listesini alır.
func RefreshFikstur(ctx context.Context, pool DB) error {
	key, league := "123", "4339"
	if b, err := GetSetting(ctx, pool, "servis", "fikstur_key"); err == nil {
		if s := strings.Trim(string(b), `"`); s != "" {
			key = s
		}
	}
	if b, err := GetSetting(ctx, pool, "servis", "fikstur_lig"); err == nil {
		if s := strings.Trim(string(b), `"`); s != "" {
			league = s
		}
	}

	var season string
	var upcoming, results []FiksturMac

	next := tsEvents(key, "eventsnextleague.php?id="+league, 10*time.Second)
	if len(next) > 0 {
		r := next[0].Round
		season = next[0].Season
		if season != "" {
			for _, rr := range []int{atoiSafe(r), atoiSafe(r) + 1} {
				if rr <= 0 {
					continue
				}
				for _, e := range tsEvents(key, fmt.Sprintf("eventsround.php?id=%s&r=%d&s=%s", league, rr, season), 10*time.Second) {
					upcoming = append(upcoming, toMac(e, "gelecek"))
				}
			}
		}
		if len(upcoming) == 0 {
			upcoming = append(upcoming, toMac(next[0], "gelecek"))
		}
	}

	past := tsEvents(key, "eventspastleague.php?id="+league, 10*time.Second)
	if len(past) > 0 {
		if season == "" {
			season = past[0].Season
		}
		if rr := atoiSafe(past[0].Round); rr > 0 && season != "" {
			for _, e := range tsEvents(key, fmt.Sprintf("eventsround.php?id=%s&r=%d&s=%s", league, rr, season), 10*time.Second) {
				results = append(results, toMac(e, "sonuc"))
			}
		}
		if len(results) == 0 {
			results = append(results, toMac(past[0], "sonuc"))
		}
	}

	demo := len(upcoming) == 0 && len(results) == 0
	if demo {
		upcoming, results = demoFikstur()
	}
	b, _ := json.Marshal(FiksturData{Upcoming: upcoming, Results: results, Demo: demo, Updated: time.Now()})
	return CacheSet(ctx, pool, "fikstur", "ana", b)
}

// FiksturOf önbellekteki fikstürü döndürür.
func FiksturOf(ctx context.Context, pool DB) FiksturData {
	var fd FiksturData
	if b, err := CacheGet(ctx, pool, "fikstur", "ana"); err == nil {
		_ = json.Unmarshal(b, &fd)
	}
	return fd
}

func toMac(e tsEvent, tur string) FiksturMac {
	t := e.Time
	if len(t) >= 5 {
		t = t[:5]
	}
	m := FiksturMac{
		Tarih:     strings.TrimSpace(e.Date + " " + t),
		Ev:        e.Home,
		Deplasman: e.Away,
		Lig:       e.League,
		Tur:       tur,
		Hafta:     e.Round,
	}
	if e.HomeScore != "" {
		if v, err := strconv.Atoi(e.HomeScore); err == nil {
			m.EvSkor = &v
		}
	}
	if e.AwayScore != "" {
		if v, err := strconv.Atoi(e.AwayScore); err == nil {
			m.DepSkor = &v
		}
	}
	return m
}

// atoiSafe metni sayıya çevirir; bozuk, boş veya taşan girdide 0 döner.
// (Eski hâli hata yok sayıp int taşmasında math.MaxInt64 döndürüyordu.)
func atoiSafe(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func demoFikstur() ([]FiksturMac, []FiksturMac) {
	e1, e2 := 2, 1
	up := []FiksturMac{{Tarih: "2026-10-16 17:00", Ev: "Çorum", Deplasman: "Rizespor", Lig: "Turkish Super Lig", Tur: "gelecek", Hafta: "8"}}
	res := []FiksturMac{{Tarih: "2026-10-12 17:00", Ev: "Eyüpspor", Deplasman: "Göztepe", EvSkor: &e1, DepSkor: &e2, Lig: "Turkish Super Lig", Tur: "sonuc", Hafta: "7"}}
	return up, res
}

// ---- Puan Durumu (Süper Lig, takım logolu) — TheSportsDB lookuptable ----

// StandingRow puan durumu tek satırı.
type StandingRow struct {
	Rank   int    `json:"rank"`
	Team   string `json:"team"`
	Badge  string `json:"badge"` // takım logosu URL'i
	Played int    `json:"played"`
	Win    int    `json:"win"`
	Draw   int    `json:"draw"`
	Loss   int    `json:"loss"`
	GF     int    `json:"gf"`
	GA     int    `json:"ga"`
	GD     int    `json:"gd"`
	Points int    `json:"points"`
	Form   string `json:"form"`
}

// StandingsData puan durumu tablosu (önbellekte saklanır).
type StandingsData struct {
	Rows    []StandingRow `json:"rows"`
	League  string        `json:"league"`
	Season  string        `json:"season"`
	Demo    bool          `json:"demo"`
	Updated time.Time     `json:"updated"`
}

// tsStanding TheSportsDB lookuptable.php satırı.
type tsStanding struct {
	Rank   string `json:"intRank"`
	Team   string `json:"strTeam"`
	Badge  string `json:"strBadge"`
	Played string `json:"intPlayed"`
	Win    string `json:"intWin"`
	Draw   string `json:"intDraw"`
	Loss   string `json:"intLoss"`
	GF     string `json:"intGoalsFor"`
	GA     string `json:"intGoalsAgainst"`
	GD     string `json:"intGoalDifference"`
	Points string `json:"intPoints"`
	Form   string `json:"strForm"`
	Season string `json:"strSeason"`
	League string `json:"strLeague"`
}

// seasonOf verilen tarihe göre futbol sezonunu döndürür (Tem–Haz).
// Örn. Ekim 2026 → "2026-2027".
func seasonOf(t time.Time) string {
	y := t.Year()
	if t.Month() >= time.July {
		return fmt.Sprintf("%d-%d", y, y+1)
	}
	return fmt.Sprintf("%d-%d", y-1, y)
}

// prevSeason bir önceki sezonu döndürür.
func prevSeason(t time.Time) string {
	y := t.Year()
	if t.Month() >= time.July {
		return fmt.Sprintf("%d-%d", y-1, y)
	}
	return fmt.Sprintf("%d-%d", y-2, y-1)
}

// badgeURL TheSportsDB küçük rozet sonekini ("/tiny") kaldırır (daha net logo).
func badgeURL(u string) string {
	return strings.TrimSuffix(strings.TrimSpace(u), "/tiny")
}

func toStandingRow(s tsStanding) StandingRow {
	return StandingRow{
		Rank:   atoiSafe(s.Rank),
		Team:   strings.TrimSpace(s.Team),
		Badge:  badgeURL(s.Badge),
		Played: atoiSafe(s.Played),
		Win:    atoiSafe(s.Win),
		Draw:   atoiSafe(s.Draw),
		Loss:   atoiSafe(s.Loss),
		GF:     atoiSafe(s.GF),
		GA:     atoiSafe(s.GA),
		GD:     atoiSafe(s.GD),
		Points: atoiSafe(s.Points),
		Form:   strings.TrimSpace(s.Form),
	}
}

var espnStandingsBase = "https://site.api.espn.com/apis/v2/sports/soccer/%s/standings"

type espnEntry struct {
	Team struct {
		DisplayName      string `json:"displayName"`
		ShortDisplayName string `json:"shortDisplayName"`
		Logos            []struct {
			Href string `json:"href"`
		} `json:"logos"`
	} `json:"team"`
	Stats []struct {
		Name  string   `json:"name"`
		Value *float64 `json:"value"`
	} `json:"stats"`
}

type espnStandingsResp struct {
	Children []struct {
		Standings struct {
			Entries []espnEntry `json:"entries"`
		} `json:"standings"`
	} `json:"children"`
	Standings struct {
		Entries []espnEntry `json:"entries"`
	} `json:"standings"`
}

// espnTeamTR ESPN'in ASCII takım adlarını Türkçe karşılıklarına çevirir.
var espnTeamTR = map[string]string{
	"Besiktas":            "Beşiktaş",
	"Amed SFK":            "Amed Sportif",
	"Fenerbahce":          "Fenerbahçe",
	"Kasimpasa":           "Kasımpaşa",
	"Caykur Rizespor":     "Çaykur Rizespor",
	"Istanbul Basaksehir": "İstanbul Başakşehir",
	"Genclerbirligi":      "Gençlerbirliği",
	"Erzurum BB":          "Erzurumspor",
	"Goztepe":             "Göztepe",
	"Eyupspor":            "Eyüpspor",
}

func trTeamName(name string) string {
	if tr, ok := espnTeamTR[name]; ok {
		return tr
	}
	return name
}

func fetchESPNStandings(code string, timeout time.Duration) []StandingRow {
	raw, err := httpGetJSONUA(fmt.Sprintf(espnStandingsBase, code), timeout)
	if err != nil {
		return nil
	}
	var resp espnStandingsResp
	if json.Unmarshal(raw, &resp) != nil {
		return nil
	}
	entries := resp.Standings.Entries
	if len(entries) == 0 && len(resp.Children) > 0 {
		entries = resp.Children[0].Standings.Entries
	}
	rows := make([]StandingRow, 0, len(entries))
	for _, e := range entries {
		get := func(name string) int {
			for _, s := range e.Stats {
				if s.Name == name && s.Value != nil {
					return int(*s.Value)
				}
			}
			return 0
		}
		badge := ""
		if len(e.Team.Logos) > 0 {
			badge = e.Team.Logos[0].Href
		}
		name := e.Team.DisplayName
		if name == "" {
			name = e.Team.ShortDisplayName
		}
		rows = append(rows, StandingRow{
			Rank:   get("rank"),
			Team:   trTeamName(name),
			Badge:  badge,
			Played: get("gamesPlayed"),
			Win:    get("wins"),
			Draw:   get("ties"),
			Loss:   get("losses"),
			GF:     get("pointsFor"),
			GA:     get("pointsAgainst"),
			GD:     get("pointDifferential"),
			Points: get("points"),
		})
	}
	// Diskta logosu olanları hemen yerel URL'e çevir (ağa çıkmadan).
	localizeStandingBadges(rows)
	return rows
}

func localizeStandingBadges(rows []StandingRow) {
	for i := range rows {
		if rows[i].Badge == "" {
			continue
		}
		key := TeamLogoKey(rows[i].Badge, rows[i].Team)
		if key != "" && platform.HasTeamLogo(key) {
			rows[i].Badge = TeamLogoURL(key)
		}
	}
}

func RefreshStandings(ctx context.Context, pool DB) error {
	for _, lg := range Leagues {
		rows := fetchESPNStandings(lg.ESPN, 12*time.Second)
		demo := len(rows) == 0
		if demo {
			// Demo veri yalnız varsayılan ligde anlamlı; diğer liglerde boş
			// tablo göstermek yerine önbelleği hiç yazmıyoruz.
			if lg.Key != "superlig" {
				continue
			}
			rows = demoStandings()
		}
		// Logoları yerelleştir: ilk çalıştırmada indirilir, sonra diskten okunur.
		// Yalnız önbelleğe yazmadan önce yapılır; istek yolu ağa çıkmaz.
		mirrorStandingBadges(ctx, rows)
		b, err := json.Marshal(StandingsData{Rows: rows, League: lg.Name, Demo: demo, Updated: time.Now()})
		if err != nil {
			continue
		}
		if err := CacheSet(ctx, pool, "puan", standingsKey(lg), b); err != nil {
			return err
		}
	}
	return nil
}

// StandingsOf varsayılan lig (Süper Lig) puan durumunu döndürür.
func StandingsOf(ctx context.Context, pool DB) StandingsData {
	return StandingsOfLeague(ctx, pool, LeagueByKey("superlig"))
}

func ManualStandings(ctx context.Context, pool DB) ([]StandingRow, bool) {
	b, err := GetSetting(ctx, pool, "servis", "puan_manuel")
	if err != nil || len(b) == 0 {
		return nil, false
	}
	var text string
	if json.Unmarshal(b, &text) != nil {
		text = strings.Trim(string(b), `"`)
	}
	rows := parseManualStandings(text)
	// Elle girilen logo adresi de dış CDN olabilir; diskte kopyası varsa
	// yerel URL'e çevrilir (ağa çıkılmaz).
	localizeStandingBadges(rows)
	return rows, len(rows) > 0
}

func parseManualStandings(text string) []StandingRow {
	var rows []StandingRow
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		num := func(i int) int {
			if i < len(parts) {
				n, _ := strconv.Atoi(parts[i])
				return n
			}
			return 0
		}
		r := StandingRow{
			Rank:   len(rows) + 1,
			Team:   parts[0],
			Played: num(2),
			Win:    num(3),
			Loss:   num(4),
			GD:     num(5),
			Points: num(6),
		}
		if len(parts) > 1 {
			r.Badge = parts[1]
		}
		rows = append(rows, r)
	}
	return rows
}

func demoStandings() []StandingRow {
	return []StandingRow{
		{Rank: 1, Team: "Galatasaray", Played: 7, Points: 16},
		{Rank: 2, Team: "Beşiktaş", Played: 7, Points: 15},
		{Rank: 3, Team: "Fenerbahçe", Played: 7, Points: 13},
	}
}

// ---- Astroloji (P3-6): günlük burç yorumları (deterministik, dış bağımlılık yok) ----

// Burc tek bir burç yorumu.
type Burc struct {
	Ad      string `json:"ad"`
	Sembol  string `json:"sembol"`
	Tarih   string `json:"tarih"`
	Element string `json:"element"`
	Yorum   string `json:"yorum"`
	Sayi    int    `json:"sayi"`
	Renk    string `json:"renk"`
}

// AstrolojiData günün burç yorumları.
type AstrolojiData struct {
	Tarih   string `json:"tarih"`
	Burclar []Burc `json:"burclar"`
}

var burcMeta = []struct{ Ad, Sembol, Tarih, Element string }{
	{"Koç", "♈", "21 Mart - 19 Nisan", "Ateş"},
	{"Boğa", "♉", "20 Nisan - 20 Mayıs", "Toprak"},
	{"İkizler", "♊", "21 Mayıs - 20 Haziran", "Hava"},
	{"Yengeç", "♋", "21 Haziran - 22 Temmuz", "Su"},
	{"Aslan", "♌", "23 Temmuz - 22 Ağustos", "Ateş"},
	{"Başak", "♍", "23 Ağustos - 22 Eylül", "Toprak"},
	{"Terazi", "♎", "23 Eylül - 22 Ekim", "Hava"},
	{"Akrep", "♏", "23 Ekim - 21 Kasım", "Su"},
	{"Yay", "♐", "22 Kasım - 21 Aralık", "Ateş"},
	{"Oğlak", "♑", "22 Aralık - 19 Ocak", "Toprak"},
	{"Kova", "♒", "20 Ocak - 18 Şubat", "Hava"},
	{"Balık", "♓", "19 Şubat - 20 Mart", "Su"},
}

var yorumHavuzu = []string{
	"Bugün iletişimin güçlü; ertelediğiniz önemli bir görüşmeyi bugün tamamlayın.",
	"Maddi konularda tedbirli olun; plansız harcamalardan kaçının.",
	"Yakın çevrenizden gelen bir teklif gündeminizi değiştirebilir.",
	"Enerjiniz yüksek; spor ve açık hava size iyi gelecek.",
	"Sabırlı olun, sabahın telaşı öğleden sonra yerini huzura bırakacak.",
	"Sezgilerinize güvenin; iç sesiniz doğru yolu gösterecek.",
	"İş hayatında takdir göreceğiniz bir gelişme kapıda.",
	"Duygusal bir konuşma ilişkinizi güçlendirebilir.",
	"Bugün attığınız küçük bir adım, uzun vadede büyük kazanç getirecek.",
	"Kendinize zaman ayırın; dinlenmek üretkenliğinizi artıracak.",
	"Yeni bir öğrenme fırsatı karşınıza çıkabilir, değerlendirin.",
	"Ölçülü ve sakin kalın; gerginlik kısa sürede dağılacak.",
	"Yolculuk veya kısa bir ara iyi gelebilir; planlarınızı gözden geçirin.",
	"Sosyal çevrenizde tanışacağınız yeni biri ufkunuzu açabilir.",
}

var renkHavuzu = []string{"Kırmızı", "Mavi", "Yeşil", "Sarı", "Turuncu", "Mor", "Beyaz", "Lacivert", "Pembe", "Turkuaz"}

// RefreshAstroloji günün burç yorumlarını üretip önbelleğe yazar.
func RefreshAstroloji(ctx context.Context, pool DB) error {
	bd := BuildAstroloji(time.Now())
	b, _ := json.Marshal(bd)
	return CacheSet(ctx, pool, "astroloji", "ana", b)
}

// BuildAstroloji verilen tarih için deterministik burç yorumları üretir.
func BuildAstroloji(t time.Time) AstrolojiData {
	day := t.Format("2006-01-02")
	bd := AstrolojiData{Tarih: day}
	for _, m := range burcMeta {
		seed := fnvHash(day + "|" + m.Ad)
		// #nosec G115 -- havuz uzunlukları derleme zamanında sabit ve
		// küçük (16 ve 8); int → uint32 dönüşümü taşamaz, mod da pozitif.
		yorum := yorumHavuzu[seed%uint32(len(yorumHavuzu))]
		// #nosec G115 -- aynı gerekçe: havuz uzunluğu sabit ve küçük.
		renk := renkHavuzu[(seed/7)%uint32(len(renkHavuzu))]
		// #nosec G115 -- seed%99 her zaman 0..98 aralığında; uint32 → int
		// dönüşümü taşamaz.
		sayi := int(seed%99) + 1
		bd.Burclar = append(bd.Burclar, Burc{Ad: m.Ad, Sembol: m.Sembol, Tarih: m.Tarih, Element: m.Element, Yorum: yorum, Sayi: sayi, Renk: renk})
	}
	return bd
}

// AstrolojiOf önbellekteki burç yorumlarını; yoksa bugün için üretir.
func AstrolojiOf(ctx context.Context, pool DB) AstrolojiData {
	var ad AstrolojiData
	if b, err := CacheGet(ctx, pool, "astroloji", "ana"); err == nil {
		_ = json.Unmarshal(b, &ad)
	}
	if ad.Tarih != time.Now().Format("2006-01-02") {
		return BuildAstroloji(time.Now())
	}
	return ad
}

func fnvHash(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// saatIkiSaatIcinde "HH:MM" damgası son 2 saat içinde mi?
func saatIkiSaatIcinde(ham string) bool {
	t, err := time.Parse("15:04", strings.TrimSpace(ham))
	if err != nil {
		return false
	}
	simdi := time.Now()
	o := time.Date(simdi.Year(), simdi.Month(), simdi.Day(),
		t.Hour(), t.Minute(), 0, 0, simdi.Location())
	if o.After(simdi) {
		o = o.Add(-24 * time.Hour) // gece yarısını geçmiş kayıt
	}
	return simdi.Sub(o) < 2*time.Hour
}
