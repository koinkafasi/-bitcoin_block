package piyasa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Kalem tek bir piyasa göstergesidir.
type Kalem struct {
	// Anahtar kararlı kimliktir (CSS sınıfı, sıralama, ayar anahtarı olarak
	// kullanılır). Görünen ada göre değil buna göre eşleşilir.
	Anahtar string  `json:"anahtar"`
	Ad      string  `json:"ad"`
	Kisa    string  `json:"kisa,omitempty"`
	Deger   float64 `json:"deger"`
	Degisim float64 `json:"degisim"` // yüzde
	Birim   string  `json:"birim,omitempty"`
	// Ondalik gösterimde kaç hane (USD 4, BIST 2, faiz 2).
	Ondalik int `json:"ondalik"`
	// Yon fiyat yönü değil, değişimin işaretidir: 1 artı, -1 eksi, 0 yatay.
	Yon int `json:"yon"`
	// Demo true ise değer gerçek kaynaktan gelmedi (sayfada rozet gösterilir).
	Demo bool `json:"demo"`
}

// Sonuc tüm göstergeleri ve verinin tazeliğini taşır.
type Sonuc struct {
	Kalemler []Kalem   `json:"kalemler"`
	Guncel   time.Time `json:"guncel"`
	// Kaynak hangi uçtan geldiği (günlük/teşhis için).
	Kaynak string `json:"kaynak"`
}

type Tanim struct {
	Anahtar string
	Ad      string
	Kisa    string
	Ondalik int
	Birim   string
	// truncgil anahtarı (varsa oradan okunur).
	Truncgil string
	// yahoo sembolü (varsa oradan okunur).
	Yahoo string
	// carpan: kaynak değeriyle çarpılır (parite/gram çevrimi için).
	Carpan float64
}

// Tanimlar gösterge tanımları — sıra ekrandaki sıradır.
var Tanimlar = []Tanim{
	{Anahtar: "bist100", Ad: "BIST 100", Kisa: "BIST 100", Ondalik: 2, Truncgil: "XU100", Yahoo: "XU100.IS"},
	{Anahtar: "usd", Ad: "Dolar", Kisa: "DOLAR", Ondalik: 4, Truncgil: "USD"},
	{Anahtar: "eur", Ad: "Euro", Kisa: "EURO", Ondalik: 4, Truncgil: "EUR"},
	{Anahtar: "gram-altin", Ad: "Gram Altın", Kisa: "GRAM ALTIN", Ondalik: 2, Truncgil: "GRA"},
	{Anahtar: "btc", Ad: "Bitcoin", Kisa: "BITCOIN", Ondalik: 2, Yahoo: "BTC-USD"},
	{Anahtar: "tahvil2y", Ad: "2 Yıllık Tahvil", Kisa: "TR 2 YILLIK", Ondalik: 2, Birim: "%"},
	{Anahtar: "gumus", Ad: "Gümüş (gram)", Kisa: "GÜMÜŞ GRAM", Ondalik: 2, Truncgil: "GUMUS"},
	{Anahtar: "gbp", Ad: "Sterlin", Kisa: "STERLİN", Ondalik: 4, Truncgil: "GBP"},
	{Anahtar: "eurusd", Ad: "Euro/Dolar", Kisa: "EURO / DOLAR", Ondalik: 4, Yahoo: "EURUSD=X"},
	{Anahtar: "brent", Ad: "Brent Petrol", Kisa: "BRENT PETROL", Ondalik: 2, Yahoo: "BZ=F", Birim: "$"},
	{Anahtar: "ceyrek-altin", Ad: "Çeyrek Altın", Kisa: "ÇEYREK ALTIN", Ondalik: 2, Truncgil: "CEYREKALTIN"},
}

// Anahtarlar tüm göstergelerin kimliklerini döndürür (ayar arayüzü için).
func Anahtarlar() []string {
	out := make([]string, 0, len(Tanimlar))
	for _, t := range Tanimlar {
		out = append(out, t.Anahtar)
	}
	return out
}

func KaynakliTanimSayisi() int {
	n := 0
	for _, t := range Tanimlar {
		if t.Truncgil != "" || t.Yahoo != "" {
			n++
		}
	}
	return n
}

const (
	truncgilURL = "https://finans.truncgil.com/v4/today.json"
	yahooURL    = "https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d"
	kullaniciUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"
)

// httpClient testlerde httptest sunucusuna yönlendirilebilsin diye değişken.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// truncgilBase / yahooBase testlerde değiştirilir.
var (
	truncgilBase = truncgilURL
	yahooBase    = yahooURL
)

// ---- truncgil (döviz + altın) ----

// tgKalem truncgil v4 satırıdır.
type tgKalem struct {
	Buying  float64 `json:"Buying"`
	Selling float64 `json:"Selling"`
	Change  float64 `json:"Change"`
	Type    string  `json:"Type"`
	Name    string  `json:"Name"`
}

func (k tgKalem) deger() (float64, bool) {
	if k.Selling > 0 {
		return k.Selling, true
	}
	if k.Buying > 0 {
		return k.Buying, true
	}
	return 0, false
}

func fetchTruncgil(ctx context.Context) (map[string]tgKalem, error) {
	var sonHata error
	for deneme := 0; deneme < 2; deneme++ {
		if deneme > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(700 * time.Millisecond):
			}
		}
		m, err := fetchTruncgilOnce(ctx)
		if err == nil && len(m) > 0 {
			return m, nil
		}
		if err != nil {
			sonHata = err
		}
	}
	if sonHata == nil {
		sonHata = errors.New("truncgil: boş yanıt")
	}
	return nil, sonHata
}

func fetchTruncgilOnce(ctx context.Context) (map[string]tgKalem, error) {
	raw, err := getJSON(ctx, truncgilBase)
	if err != nil {
		return nil, err
	}
	// Update_Date gibi skaler alanlar tgKalem'e çevrilemez; bu yüzden önce
	// ham map'e çözüp yalnız nesne olan satırları alıyoruz.
	var ham map[string]json.RawMessage
	if err := json.Unmarshal(raw, &ham); err != nil {
		return nil, err
	}
	out := make(map[string]tgKalem, len(ham))
	for k, v := range ham {
		if len(v) == 0 || v[0] != '{' {
			continue
		}
		var row tgKalem
		if json.Unmarshal(v, &row) != nil {
			continue
		}
		out[k] = row
	}
	return out, nil
}

// ---- Yahoo chart ----

type yahooResp struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice     *float64 `json:"regularMarketPrice"`
				ChartPreviousClose     *float64 `json:"chartPreviousClose"`
				PreviousClose          *float64 `json:"previousClose"`
				RegularMarketChangePct *float64 `json:"regularMarketChangePercent"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct {
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

type yahooDeger struct {
	fiyat   float64
	degisim float64
	ok      bool
}

func fetchYahoo(ctx context.Context, sembol string) yahooDeger {
	raw, err := getJSON(ctx, fmt.Sprintf(yahooBase, urlEscape(sembol)))
	if err != nil {
		return yahooDeger{}
	}
	var r yahooResp
	if json.Unmarshal(raw, &r) != nil || len(r.Chart.Result) == 0 {
		return yahooDeger{}
	}
	m := r.Chart.Result[0].Meta
	if m.RegularMarketPrice == nil || *m.RegularMarketPrice <= 0 {
		return yahooDeger{}
	}
	out := yahooDeger{fiyat: *m.RegularMarketPrice, ok: true}
	// Değişim: varsa hazır yüzde; yoksa önceki kapanıştan hesaplanır.
	if m.RegularMarketChangePct != nil {
		out.degisim = *m.RegularMarketChangePct
		return out
	}
	prev := 0.0
	if m.ChartPreviousClose != nil {
		prev = *m.ChartPreviousClose
	} else if m.PreviousClose != nil {
		prev = *m.PreviousClose
	}
	if prev > 0 {
		out.degisim = (out.fiyat - prev) / prev * 100
	}
	return out
}

// ---- toplama ----

// Get tüm göstergeleri toplar. Bazı kaynaklar erişilemezse elde olanlarla
// döner; hiçbiri yoksa hata verir (çağıran önbelleği ezmez).
func Get(ctx context.Context) (Sonuc, error) {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		tg       map[string]tgKalem
		tgErr    error
		yahooMap = map[string]yahooDeger{}
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		m, err := fetchTruncgil(ctx)
		mu.Lock()
		tg, tgErr = m, err
		mu.Unlock()
	}()

	// Yahoo sembolleri paralel çekilir: 5 istek sırayla ~10 sn sürerdi.
	var semboller []string
	for _, t := range Tanimlar {
		if t.Yahoo != "" {
			semboller = append(semboller, t.Yahoo)
		}
	}
	var ywg sync.WaitGroup
	for _, s := range semboller {
		s := s
		ywg.Add(1)
		go func() {
			defer ywg.Done()
			d := fetchYahoo(ctx, s)
			mu.Lock()
			yahooMap[s] = d
			mu.Unlock()
		}()
	}
	ywg.Wait()
	wg.Wait()

	simdi := time.Now()
	out := Sonuc{Guncel: simdi, Kalemler: make([]Kalem, 0, len(Tanimlar))}
	var kaynaklar []string
	if tgErr == nil && len(tg) > 0 {
		kaynaklar = append(kaynaklar, "truncgil")
	}
	yahooVar := false
	for _, d := range yahooMap {
		if d.ok {
			yahooVar = true
			break
		}
	}
	if yahooVar {
		kaynaklar = append(kaynaklar, "yahoo")
	}

	for _, t := range Tanimlar {
		k := Kalem{
			Anahtar: t.Anahtar, Ad: t.Ad, Kisa: t.Kisa,
			Ondalik: t.Ondalik, Birim: t.Birim,
		}
		ok := false
		if t.Truncgil != "" && tgErr == nil {
			if row, found := tg[t.Truncgil]; found {
				if v, has := row.deger(); has {
					k.Deger = v * orOne(t.Carpan)
					k.Degisim = row.Change
					ok = true
				}
			}
		}
		if !ok && t.Yahoo != "" {
			if d := yahooMap[t.Yahoo]; d.ok {
				k.Deger = d.fiyat * orOne(t.Carpan)
				k.Degisim = d.degisim
				ok = true
			}
		}
		if !ok {
			continue
		}
		k.Yon = yon(k.Degisim)
		out.Kalemler = append(out.Kalemler, k)
	}
	if len(out.Kalemler) == 0 {
		if tgErr != nil {
			return Sonuc{}, tgErr
		}
		return Sonuc{}, errors.New("piyasa verisi alınamadı")
	}
	out.Kaynak = strings.Join(kaynaklar, "+")
	return out, nil
}

func orOne(f float64) float64 {
	if f == 0 {
		return 1
	}
	return f
}

// yon değişimin işaretini döndürür; 0,005'in altı yatay sayılır (ekranda
// "0,00" yazıp ok göstermek tutarsız olurdu).
func yon(d float64) int {
	switch {
	case d > 0.005:
		return 1
	case d < -0.005:
		return -1
	}
	return 0
}

// getJSON ortak HTTP okuma (boyut ve hata denetimi).
func getJSON(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", kullaniciUA)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	// io.ReadAll + LimitReader: elle okuma dongusu EOF'u string
	// karsilastirmasiyla yakaliyordu (kirilgan). Sinir 4 MB.
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

// urlEscape sembolü sorgu için kaçırır (= ve ^ karakterleri).
func urlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '=':
			b.WriteString("%3D")
		case '^':
			b.WriteString("%5E")
		case ' ':
			b.WriteString("%20")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ---- biçimlendirme (Türkçe) ----

// FormatTR sayıyı Türkçe biçimde yazar: binlik nokta, ondalık virgül.
// Örnek: 11151.03 → "11.151,03"; 49.8335 → "49,8335".
func FormatTR(v float64, ondalik int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "-"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatFloat(v, 'f', ondalik, 64)
	tam, kes, _ := strings.Cut(s, ".")
	// binlik ayırıcı
	var b strings.Builder
	for i, c := range tam {
		if i > 0 && (len(tam)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	out := b.String()
	if kes != "" {
		out += "," + kes
	}
	if neg {
		out = "-" + out
	}
	return out
}

// FormatDegisim yüzde değişimi işaretli yazar: "+0,05", "-0,15", "0,00".
func FormatDegisim(d float64) string {
	s := FormatTR(math.Abs(d), 2)
	if yon(d) > 0 {
		return "+" + s
	}
	if yon(d) < 0 {
		return "-" + s
	}
	return s
}

// GuvenliHTML zararsız metin (şablon güvenliği için).
func GuvenliHTML(s string) string { return html.EscapeString(s) }
