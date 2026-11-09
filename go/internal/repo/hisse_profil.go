package repo

// Hisse profili (şirket bilgisi) — CoinDesk §10.6 "About …" bölümü.
//
// KAYNAK: Yahoo Finance `quoteSummary` ucu. Bu uç CRUMB ister:
// `/v1/test/getcrumb` (çerezle) → crumb, sonra `?crumb=…`. Crumb iki
// ek HTTP isteği demek; bu yüzden SÜREÇ İÇİ ÖNBELLEKTE tutulur ve
// yalnız 401 alındığında yenilenir.
//
// NEDEN AYRI DOSYA: `hisse_detay.go` çizim (chart) ucunu kullanır ve
// çerez gerektirmez. Profil akışı kendi durumunu (çerez kavanozu +
// crumb) taşıdığı için ayrı tutuldu; karıştırılırsa her fiyat isteği
// crumb yenileme yoluna girebilirdi.
//
// UYDURMA YOK: alan gelmezse boş/0 kalır, şablon o satırı basmaz.
// Ölçüldü (9 Kasım 2026): META ve THYAO.IS için de longBusinessSummary,
// sector/industry, fullTimeEmployees, website ve marketCap geliyor.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"
)

// yahooQuoteSummary aralık parametresiz profil ucu (crumb ile).
const yahooQuoteSummary = "https://query1.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=assetProfile,summaryDetail,price&crumb=%s"

// HisseProfil şirket künyesi.
type HisseProfil struct {
	Ozet        string  `json:"ozet"`         // iş tanımı (kaynak dili İngilizce)
	Sektor      string  `json:"sektor"`       // "Communication Services"
	Endustri    string  `json:"endustri"`     // "Internet Content & Information"
	Calisan     int     `json:"calisan"`      // tam zamanlı çalışan
	Web         string  `json:"web"`          // şirket sitesi
	PiyasaDeger float64 `json:"piyasa_deger"` // market cap (yerel para)
	Beta        float64 `json:"beta"`
	OrtHacim    float64 `json:"ort_hacim"` // ortalama günlük hacim
}

// sozluk dolu mu (şablon "hiçbir alan yok" durumunu bununla anlar).
func (p HisseProfil) Bos() bool {
	return p.Ozet == "" && p.PiyasaDeger == 0 && p.Sektor == "" && p.Calisan == 0
}

// yahooCrumb süreç içi crumb + çerez durumu.
//
// NEDEN ÖNBELLEK: crumb akışı 2 istek (404 sayfa + getcrumb) demek.
// Her hisse sayfasında tekrarlanırsa sayfa başına 3 istek yerine 1
// kalır. Yahoo crumb'ı uzun ömürlü; 6 saat güvenli.
type yahooCrumb struct {
	mu      sync.Mutex
	jar     http.CookieJar
	crumb   string
	zaman   time.Time
	gecerli time.Duration
}

var yahooCrumbDurum = &yahooCrumb{gecerli: 6 * time.Hour}

// al taze crumb döndürür (gerekirse yeniler). Çağıran kilit tutmaz.
func (y *yahooCrumb) al(ctx context.Context) (string, http.CookieJar, error) {
	y.mu.Lock()
	defer y.mu.Unlock()

	if y.crumb != "" && time.Since(y.zaman) < y.gecerli {
		return y.crumb, y.jar, nil
	}

	// NOT: çerez kavanozu her yenilemede sıfırlanır; eski crumb eski
	// kavanoza bağlı, karıştırmak 401 üretir.
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", nil, err
	}
	client := &http.Client{Timeout: 8 * time.Second, Jar: jar}
	ua := "Mozilla/5.0 (compatible; theblock/1.0)"

	// 1) Çerez tohumu. Sayfa 404 döner ama Set-Cookie bırakır; bu yüzden
	// durum kodu ÖNEMSİZ (ölçüldü: fc.yahoo.com → 404 + A3 çerezi).
	if req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://fc.yahoo.com", nil); err == nil {
		req.Header.Set("User-Agent", ua)
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
		}
	}

	// 2) Crumb.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://query1.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("crumb HTTP %d", resp.StatusCode)
	}
	buf := make([]byte, 0, 64)
	tmp := make([]byte, 64)
	for i := 0; i < 2; i++ {
		n, err := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	c := strings.TrimSpace(string(buf))
	if c == "" || len(c) > 32 || strings.ContainsAny(c, " <>{}\n") {
		return "", nil, fmt.Errorf("crumb biçimsiz")
	}

	y.crumb, y.jar, y.zaman = c, jar, time.Now()
	return y.crumb, y.jar, nil
}

// sifirla crumb'ı geçersiz kılar (401 sonrası yenileme için).
func (y *yahooCrumb) sifirla() {
	y.mu.Lock()
	defer y.mu.Unlock()
	y.crumb = ""
	y.zaman = time.Time{}
}

// HisseProfilCek sembolün şirket künyesini çeker.
//
// Hata durumunda sıfır değer döner; çağıran "About" bölümünü BASMAZ.
func HisseProfilCek(ctx context.Context, sembol string) (HisseProfil, error) {
	sym := strings.ToUpper(strings.TrimSpace(sembol))
	if !HisseSembolGecerli(sym) {
		return HisseProfil{}, fmt.Errorf("geçersiz sembol")
	}

	for deneme := 0; deneme < 2; deneme++ {
		crumb, jar, err := yahooCrumbDurum.al(ctx)
		if err != nil {
			return HisseProfil{}, err
		}
		u := fmt.Sprintf(yahooQuoteSummary, sym, crumb)
		client := &http.Client{Timeout: 10 * time.Second, Jar: jar}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return HisseProfil{}, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; theblock/1.0)")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return HisseProfil{}, err
		}
		body, rerr := readSinirli(resp, 2<<20)
		if rerr != nil {
			return HisseProfil{}, rerr
		}
		// 401 = crumb bayatladı: bir kez yenile, tekrar dene.
		if resp.StatusCode == http.StatusUnauthorized && deneme == 0 {
			yahooCrumbDurum.sifirla()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return HisseProfil{}, fmt.Errorf("quoteSummary HTTP %d", resp.StatusCode)
		}
		return hisseProfilCoz(body)
	}
	return HisseProfil{}, fmt.Errorf("profil alınamadı")
}

// readSinirli gövdeyi sınırla okur (bellek emniyeti).
func readSinirli(resp *http.Response, sinir int64) ([]byte, error) {
	defer resp.Body.Close()
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 8192)
	var toplam int64
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			toplam += int64(n)
			if toplam > sinir {
				return nil, fmt.Errorf("yanıt çok büyük")
			}
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}

// yahooSayi Yahoo'nun {"raw":…,"fmt":…} sarmalını çözer.
// Bazı alanlar düz sayı gelir; ikisi de desteklenir.
type yahooSayi float64

func (y *yahooSayi) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '{' {
		var s struct {
			Raw float64 `json:"raw"`
		}
		if err := json.Unmarshal(b, &s); err != nil {
			return nil // biçim beklenmedik: alanı boş bırak, hata yükseltme
		}
		*y = yahooSayi(s.Raw)
		return nil
	}
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return nil
	}
	*y = yahooSayi(f)
	return nil
}

// hisseProfilCoz quoteSummary yanıtını HisseProfil'e indirir.
func hisseProfilCoz(raw []byte) (HisseProfil, error) {
	var y struct {
		QuoteSummary struct {
			Result []struct {
				AssetProfile struct {
					LongBusinessSummary string    `json:"longBusinessSummary"`
					Sector              string    `json:"sector"`
					Industry            string    `json:"industry"`
					FullTimeEmployees   yahooSayi `json:"fullTimeEmployees"`
					Website             string    `json:"website"`
				} `json:"assetProfile"`
				SummaryDetail struct {
					MarketCap     yahooSayi `json:"marketCap"`
					Beta          yahooSayi `json:"beta"`
					AverageVolume yahooSayi `json:"averageVolume"`
				} `json:"summaryDetail"`
				Price struct {
					MarketCap yahooSayi `json:"marketCap"`
				} `json:"price"`
			} `json:"result"`
		} `json:"quoteSummary"`
	}
	if err := json.Unmarshal(raw, &y); err != nil {
		return HisseProfil{}, err
	}
	if len(y.QuoteSummary.Result) == 0 {
		return HisseProfil{}, fmt.Errorf("profil sonucu yok")
	}
	r := y.QuoteSummary.Result[0]

	p := HisseProfil{
		Ozet:        strings.TrimSpace(r.AssetProfile.LongBusinessSummary),
		Sektor:      strings.TrimSpace(r.AssetProfile.Sector),
		Endustri:    strings.TrimSpace(r.AssetProfile.Industry),
		Calisan:     int(r.AssetProfile.FullTimeEmployees),
		Web:         strings.TrimSpace(r.AssetProfile.Website),
		Beta:        float64(r.SummaryDetail.Beta),
		OrtHacim:    float64(r.SummaryDetail.AverageVolume),
		PiyasaDeger: float64(r.Price.MarketCap),
	}
	if p.PiyasaDeger == 0 {
		p.PiyasaDeger = float64(r.SummaryDetail.MarketCap)
	}
	if p.Bos() {
		return HisseProfil{}, fmt.Errorf("profil boş")
	}
	return p, nil
}

// OzetKirp iş tanımını cümle sınırında kısaltır.
//
// NEDEN: CoinDesk "About" bölümü 326×407 px (≈13 satır). Tam metin
// (META'da 1988 karakter) taşırıyor; cümle ortasında kesmek de okunmuyor.
func OzetKirp(ozet string, sinir int) string {
	o := strings.Join(strings.Fields(ozet), " ")
	if sinir <= 0 || len(o) <= sinir {
		return o
	}
	// "…" 3 bayt: sınır aşılmasın diye kesim payı düşülür.
	kes := o[:sinir-3]
	// Son cümle sonunu ara (. ! ?) — yoksa son boşluktan kes.
	if i := strings.LastIndexAny(kes, ".!?"); i > sinir/2 {
		return strings.TrimSpace(kes[:i+1])
	}
	if i := strings.LastIndex(kes, " "); i > 0 {
		return strings.TrimSpace(kes[:i]) + "…"
	}
	return strings.TrimSpace(kes) + "…"
}
