package repo

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

// Nöbetçi eczane — NosyAPI (RapidAPI) kaynağı.
//
// NEDEN: eczaneler.org küme çıkış IP'sinden Cloudflare 403 alıyor
// (bkz. eczane_aeo.go başı); resmi eczacı odası adaptörü yalnız Ankara'yı
// kapsıyor. NosyAPI 81 ili veriyor ve veri merkezi IP'sinden erişilebiliyor
// (5 Kasım 2026'da doğrulandı).
//
// UÇ: RapidAPI anahtarıyla `pharmacies-on-duty/all` AÇIK DEĞİL (404).
// Yalnız şehir bazlı uç var:
//
//	GET https://nobetci-eczane-api-turkiye.p.rapidapi.com/pharmacies-on-duty?city=ankara
//	X-RapidAPI-Key: …
//	X-RapidAPI-Host: nobetci-eczane-api-turkiye.p.rapidapi.com
//
//	{"status":"success","rowCount":49,"creditUsed":0,
//	 "data":[{"pharmacyID":1452,"pharmacyName":"…","address":"…",
//	          "city":"Ankara","cityId":6,"district":"Çankaya","town":"",
//	          "directions":"…","phone":"0 (312) 512-26-12", …}]}
//
// `cityId` il plakasıyla birebir aynıdır (Ankara 6, Bayburt 69 — ölçüldü).
//
// MALİYET: il başına günde bir istek (81 istek/gün). Yanıtlar il bazında
// GÜN BOYU önbelleklenir; aynı gün "Şimdi Yenile" veya boş il yeniden
// denemeleri ek istek harcamaz. Anahtar boşsa kaynak tamamen kapalıdır.

const (
	eczaneRapidAPIHost = "nobetci-eczane-api-turkiye.p.rapidapi.com"
	eczaneRapidAPIPath = "/pharmacies-on-duty"
	// Tek il yanıtı en fazla ~500 KB (İstanbul); 4 MB bol.
	eczaneRapidAPIMaxGovde = 4 << 20
	eczaneRapidAPITimeout  = 25 * time.Second
	// Sayfada "kaynak" olarak gösterilen ad.
	eczaneRapidAPIKaynak = "nosyapi.com"
)

// eczaneAPISehirCache, bir ilin günlük API yanıtıdır. Boş liste de
// önbelleğe yazılır: "bugün nöbetçi eczane yok" cevabı da bir cevaptır.
type eczaneAPISehirCache struct {
	Tarih string   `json:"tarih"` // YYYY-MM-DD
	Liste []Eczane `json:"liste"`
}

// eczaneRapidAPIYanit, API'nin JSON gövdesidir.
type eczaneRapidAPIYanit struct {
	Status string                `json:"status"`
	Data   []eczaneRapidAPISatir `json:"data"`
}

type eczaneRapidAPISatir struct {
	PharmacyID int64  `json:"pharmacyID"`
	Ad         string `json:"pharmacyName"`
	Adres      string `json:"address"`
	Sehir      string `json:"city"`
	Plaka      int    `json:"cityId"`
	Ilce       string `json:"district"`
	Mahalle    string `json:"town"`
	Tarif      string `json:"directions"`
	Telefon    string `json:"phone"`
	Telefon2   string `json:"phone2"`
}

// eczaneAPISehir, bir ilin listesini günlük önbellekten okur; yoksa
// API'den bir kez çekip yazar. İkinci dönüş: kaynak erişilebilir mi?
//
// Erişilemediyse çağıran eczaneler.org'a düşer. Erişildi ama liste boşsa
// "bugün nöbetçi eczane yok"tur; demo/uydurma liste basılmaz.
func eczaneAPISehir(ctx context.Context, pool DB, anahtar, slug string, plaka int) ([]Eczane, bool) {
	bugun := time.Now().Format("2006-01-02")
	cacheKey := strconv.Itoa(plaka)
	if b, err := CacheGet(ctx, pool, "eczane_api", cacheKey); err == nil && len(b) > 0 {
		var c eczaneAPISehirCache
		if json.Unmarshal(b, &c) == nil && c.Tarih == bugun {
			return c.Liste, true
		}
	}
	liste, err := eczaneRapidAPISehir(ctx, anahtar, slug)
	if err != nil {
		return nil, false
	}
	if b, err := json.Marshal(eczaneAPISehirCache{Tarih: bugun, Liste: liste}); err == nil {
		_ = CacheSet(ctx, pool, "eczane_api", cacheKey, b)
	}
	return liste, true
}

// eczaneRapidAPISehir tek bir ilin nöbetçi eczanelerini çeker.
func eczaneRapidAPISehir(ctx context.Context, anahtar, slug string) ([]Eczane, error) {
	anahtar = strings.TrimSpace(anahtar)
	slug = strings.TrimSpace(slug)
	if anahtar == "" {
		return nil, fmt.Errorf("nosyapi: anahtar boş")
	}
	if slug == "" {
		return nil, fmt.Errorf("nosyapi: il slug boş")
	}
	u := "https://" + eczaneRapidAPIHost + eczaneRapidAPIPath + "?city=" + url.QueryEscape(slug)
	istek, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	istek.Header.Set("X-RapidAPI-Key", anahtar)
	istek.Header.Set("X-RapidAPI-Host", eczaneRapidAPIHost)
	istek.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: eczaneRapidAPITimeout}
	yanit, err := client.Do(istek)
	if err != nil {
		return nil, err
	}
	defer yanit.Body.Close()
	if yanit.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nosyapi: HTTP %d", yanit.StatusCode)
	}
	govde, err := io.ReadAll(io.LimitReader(yanit.Body, eczaneRapidAPIMaxGovde))
	if err != nil {
		return nil, err
	}
	return eczaneRapidAPICoz(govde)
}

// eczaneRapidAPICoz, tek il yanıtını Eczane listesine çevirir.
//
// Bozuk/eksik kayıtlar atılır; adı olmayan eczane gösterilmez. Aynı kayıt
// birden çok kez gelirse eczaneTekrarsiz tekilleştirir.
func eczaneRapidAPICoz(govde []byte) ([]Eczane, error) {
	var y eczaneRapidAPIYanit
	if err := json.Unmarshal(govde, &y); err != nil {
		return nil, fmt.Errorf("nosyapi: JSON çözülemedi: %w", err)
	}
	if y.Status != "success" {
		return nil, fmt.Errorf("nosyapi: status=%q", y.Status)
	}
	out := make([]Eczane, 0, len(y.Data))
	for _, s := range y.Data {
		if strings.TrimSpace(s.Ad) == "" {
			continue
		}
		e := Eczane{
			Ilce:  strings.TrimSpace(s.Ilce),
			Ad:    temizMetin(s.Ad),
			Adres: eczaneRapidAPIAdres(s),
			Tel:   eczaneTelefonDuzelt(s.Telefon),
		}
		if e.Ilce == "" {
			e.Ilce = strings.TrimSpace(s.Mahalle)
		}
		if e.Ilce == "" {
			e.Ilce = strings.TrimSpace(s.Sehir)
		}
		out = append(out, e)
	}
	return eczaneTekrarsiz(out), nil
}

// eczaneRapidAPIAdres, adres ile "Tarif" alanını tek metinde birleştirir.
func eczaneRapidAPIAdres(s eczaneRapidAPISatir) string {
	adres := temizMetin(s.Adres)
	tarif := temizMetin(s.Tarif)
	if tarif == "" {
		return adres
	}
	if adres == "" {
		return tarif
	}
	// Tarif adresin içinde tekrarlanıyorsa ekleme.
	if strings.Contains(strings.ToLower(adres), strings.ToLower(tarif)) {
		return adres
	}
	return strings.TrimSpace(adres + " (Tarif: " + tarif + ")")
}

// eczaneTelefonDuzelt, API'nin "0 (312) 512-26-12" biçimini sitede
// kullanılan "0 312 512 26 12" biçimine çevirir. Çözülemeyen değer
// olduğu gibi bırakılır — uydurma numara üretilmez.
func eczaneTelefonDuzelt(ham string) string {
	ham = strings.TrimSpace(ham)
	rakam := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, ham)
	rakam = strings.TrimPrefix(rakam, "90")
	rakam = strings.TrimPrefix(rakam, "0")
	if len(rakam) == 10 {
		return turkceTelefon(rakam)
	}
	return ham
}
