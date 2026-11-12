package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var eczaneOdaKaynaklari = map[int]func(context.Context) ([]Eczane, string, error){
	6: eczaneAEOAnkara, // Ankara Eczacı Odası
}

// aeoHTTPTimeout: yanıt 196 KB ve origin ~0.8s'de dönüyor; 15 sn bol.
const aeoHTTPTimeout = 15 * time.Second

func aeoURL(tarih string) string {
	return "https://www.aeo.org.tr/getPharmacies/" + tarih
}

func eczaneAEOAnkara(ctx context.Context) ([]Eczane, string, error) {
	const ad = "aeo.org.tr"
	raw, err := httpGetTarayici(ctx, aeoURL(time.Now().Format("2006-01-02")), aeoHTTPTimeout)
	if err != nil {
		return nil, ad, err
	}
	var yanit struct {
		Status string `json:"status"`
		HTML   string `json:"html"`
		Date   string `json:"date"`
	}
	if err := json.Unmarshal(raw, &yanit); err != nil {
		return nil, ad, fmt.Errorf("aeo: JSON çözülemedi: %w", err)
	}
	if yanit.Status != "success" {
		return nil, ad, fmt.Errorf("aeo: status=%q", yanit.Status)
	}
	list := eczaneAEOCoz(yanit.HTML)
	if len(list) == 0 {
		// Uç 200 döndü ama hiç kayıt çıkmadı: ya o gün liste yok ya da
		// işaretler değişti. Çağıran bunu "veri yok" sayar, uydurmaz.
		return nil, ad, nil
	}
	return list, ad, nil
}

var (
	// İlçe doğrudan veriliyor: adresten çıkarmaya gerek yok.
	aeoIlceRe = regexp.MustCompile(`data-district="([^"]*)"`)
	// Ad kayıttaki tek <h4>.
	aeoAdRe = regexp.MustCompile(`<h4>([^<]{2,120})</h4>`)
	// Adres + tarif tek <p> içinde; <br> ile ayrılır, telefon <span>'de.
	aeoAdresRe  = regexp.MustCompile(`(?s)<p>(.*?)</p>`)
	aeoTelRe    = regexp.MustCompile(`<span>\s*(0?\d{10,11})\s*</span>`)
	aeoBrRe     = regexp.MustCompile(`(?i)<br\s*/?>`)
	aeoEtiketRe = regexp.MustCompile(`<[^>]*>`)
)

func eczaneAEOCoz(h string) []Eczane {
	parcalar := strings.Split(h, `<div class="inline-box"`)
	if len(parcalar) < 2 {
		return nil
	}
	out := make([]Eczane, 0, len(parcalar)-1)
	for _, p := range parcalar[1:] {
		var e Eczane

		if m := aeoAdRe.FindStringSubmatch(p); m != nil {
			e.Ad = temizMetin(m[1])
		}
		// Ad yoksa kayıt kullanılamaz: adsız eczane gösterilmez.
		if e.Ad == "" {
			continue
		}
		if m := aeoIlceRe.FindStringSubmatch(p); m != nil {
			e.Ilce = temizMetin(m[1])
		}
		if m := aeoAdresRe.FindStringSubmatch(p); m != nil {
			icerik := m[1]
			// Telefon <span>'i adresin içine karışmasın.
			if i := strings.Index(icerik, "<span"); i >= 0 {
				icerik = icerik[:i]
			}
			// <br> adres ile tarifi ayırıyor; ikisi tek satırda birleşir.
			icerik = aeoBrRe.ReplaceAllString(icerik, " ")
			e.Adres = temizMetin(aeoEtiketRe.ReplaceAllString(icerik, " "))
		}
		if m := aeoTelRe.FindStringSubmatch(p); m != nil {
			// Kaynak "03122131678" (başta 0, 11 hane) veriyor.
			if t := strings.TrimPrefix(m[1], "0"); len(t) == 10 {
				e.Tel = turkceTelefon(t)
			} else {
				e.Tel = temizMetin(m[1])
			}
		}
		out = append(out, e)
	}
	return eczaneTekrarsiz(out)
}
