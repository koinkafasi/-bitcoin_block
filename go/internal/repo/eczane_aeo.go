package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Nöbetçi eczane — RESMİ eczacı odası kaynakları.
//
// NEDEN BU DOSYA VAR: eczaneler.org (ücretsiz HTML kaynağı, eczane_kaynak.go)
// 4 Kasım 2026'daki Kubernetes taşımasından sonra kullanılamaz oldu.
// Cloudflare, küme çıkış IP'sine (173.249.30.38, Contabo veri merkezi)
// "managed challenge" veriyor ve 403 dönüyor. Birebir aynı istek ev
// IP'sinden 200 + 600 KB gerçek veri döndürüyor; yani kod, başlık ya da
// regex sorunu DEĞİL — veri merkezi IP itibarı. IPv4/IPv6 zorlamak da
// değiştirmiyor (kümede IPv6 çıkışı yok).
//
// Eczacı odaları ASIL kaynaktır: nöbet listesini onlar yayınlar, toplayıcı
// siteler onlardan alır. Ölçüm (5 Kasım 2026, KÜME İÇİNDEN):
//
//	www.aeo.org.tr   HTTP 200, 196 KB, 57 eczane, 0.78s — UA bile gerekmiyor
//	eczaneler.org    HTTP 403 (challenge)
//
// Her oda AYRI yazılım kullanıyor, ortak uç YOK (ölçüldü: İstanbul
// istanbuleczaciodasi.org.tr'ye taşınmış, İzmir ve Konya kümeden 403,
// Bursa/Trabzon JSON vermiyor). Bu yüzden tek bir "oda kaynağı" yazmak
// mümkün değil; il bazlı adaptör tablosu tutuluyor. Yeni il eklemek =
// bir fonksiyon yazıp tabloya bir satır eklemek.

// eczaneOdaKaynaklari il PLAKASINA göre resmi eczacı odası adaptörüdür.
// Dönüş: (liste, kaynak adı, hata). Liste boşsa çağıran sıradaki kaynağa
// geçer.
var eczaneOdaKaynaklari = map[int]func(context.Context) ([]Eczane, string, error){
	6: eczaneAEOAnkara, // Ankara Eczacı Odası
}

// aeoHTTPTimeout: yanıt 196 KB ve origin ~0.8s'de dönüyor; 15 sn bol.
const aeoHTTPTimeout = 15 * time.Second

// aeoURL bir günün nöbet listesi ucudur.
//
// `www.` ZORUNLU: çıplak aeo.org.tr 301 veriyor (ölçüldü).
func aeoURL(tarih string) string {
	return "https://www.aeo.org.tr/getPharmacies/" + tarih
}

// eczaneAEOAnkara, Ankara Eczacı Odası'nın o an GEÇERLİ nöbet listesini çeker.
//
// Yanıt JSON: {"status":"success","html":"…","date":"2026-11-04", …}
// Liste HTML olarak `html` alanında geliyor; ayrı bir veri ucu yok.
//
// Tarih olarak BUGÜN verilir. Nöbet 08:30'da devredildiği için sabahın
// erken saatlerinde yanıttaki `date` DÜNÜN tarihi olur — bu doğrudur,
// o liste hâlâ geçerlidir ("05.11.2026 saat 08:30'a kadar 04.11.2026
// tarihli nöbet listesi geçerlidir" notu yanıtın içinde geliyor).
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

// eczaneAEOCoz, getPharmacies yanıtının `html` alanından kayıtları çıkarır.
//
// Neden ayrıştırıcı (parser) bağımlılığı yok: eczane_kaynak.go'daki
// gerekçenin aynısı — `golang.org/x/net/html` eklemek bağımlılık ağacını
// büyütürdü ve burada yalnız 4 sabit işaret var.
//
// Kayıt sınırı `<div class="inline-box"` ile BÖLÜNEREK bulunur, regex ile
// EŞLEŞTİRİLMEZ: kayıt içinde iç içe <div>'ler var ve kapanış etiketini
// regex'le doğru yakalamak kırılgan olurdu.
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
