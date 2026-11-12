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
