package repo

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// Promo banner (§3) — CoinDesk ana sayfa sırasındaki duyuru bandı.
//
// TASARIM: metin/bağlantı settings "promo" bölümünden gelir; kod deploy'u
// gerekmeden değişir. Bant yalnız `aktif` ise ve `metin` doluysa basılır.
//
// NEDEN BURADA geri sayım: CoinDesk'te bant "etkinlik geri sayımı" taşır.
// Bizde etkinlik takvimi yok; uydurma tarih basmak yanlış bilgi olurdu.
// Bu yüzden geri sayım **isteğe bağlı** ve yalnız editörün girdiği gerçek
// `bitis` zamanına göre hesaplanır. `bitis` geçmişse bant kendiliğinden
// kapanır (süresi dolmuş kampanya sayfada kalmaz).
type PromoBanner struct {
	Aktif    bool
	Metin    string
	Baglanti string // iç yol ("/newsletters") veya mutlak https URL
	CTA      string // bağlantı metni; boşsa bağlantı yalnız metne sarılır
	Bitis    time.Time
	// Kalan, bitişe kalan sürenin insan okur metnidir ("3 gün 4 saat").
	// Boşsa bant geri sayım göstermez.
	Kalan string
}

// Bos mu (şablon basmamalı mı).
func (p PromoBanner) Bos() bool { return !p.Aktif || strings.TrimSpace(p.Metin) == "" }

// AktifPromo settings/promo bölümünü okuyup yayınlanabilir bandı döndürür.
//
// Hata durumunda boş bant döner: bir duyuru bandı yüzünden ana sayfanın
// 500 dönmesi kabul edilemez (bant süs, sayfa çekirdek).
func AktifPromo(ctx context.Context, pool DB, simdi time.Time) PromoBanner {
	ayar, err := GetSettings(ctx, pool, "promo")
	if err != nil {
		return PromoBanner{}
	}
	return promoAyarlardan(ayar, simdi)
}

// promoAyarlardan ayar haritasını banda çevirir (DB'siz, test edilebilir).
//
// Kurallar:
//   - aktif değilse veya metin boşsa GÖRÜNMEZ (boş çerçeve basılmaz),
//   - bitis doluysa geri sayım hesaplanır,
//   - bitis geçmişse kampanya kapanmıştır: bant basılmaz.
func promoAyarlardan(ayar map[string][]byte, simdi time.Time) PromoBanner {
	p := PromoBanner{
		Aktif:    ayarBool(ayar["aktif"]),
		Metin:    strings.TrimSpace(ayarStr(ayar["metin"])),
		Baglanti: strings.TrimSpace(ayarStr(ayar["baglanti"])),
		CTA:      strings.TrimSpace(ayarStr(ayar["baglanti_metin"])),
	}
	if p.Bos() {
		return PromoBanner{}
	}
	ham := strings.TrimSpace(ayarStr(ayar["bitis"]))
	if ham == "" {
		return p
	}
	// datetime-local "2026-11-20T18:00" biçiminde gelir; RFC3339 ve tarih
	// tek başına da kabul edilir (panel farklı tarayıcılarda farklı yazar).
	for _, d := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(d, ham, simdi.Location()); err == nil {
			p.Bitis = t
			break
		}
	}
	// Bitiş okunamadıysa geri sayım YOK ama bant kalır: editörün yazım
	// hatası yüzünden duyuruyu tamamen düşürmek veri kaybı olurdu.
	if p.Bitis.IsZero() {
		return p
	}
	if !p.Bitis.After(simdi) {
		return PromoBanner{}
	}
	p.Kalan = KalanSure(p.Bitis.Sub(simdi))
	return p
}

// KalanSure kalan süreyi kaba insan diline çevirir.
//
// Kaba olması bilinçli: sayfa 60 sn önbellekleniyor, saniye hassasiyeti
// yanıltıcı olurdu. "3 gün 4 saat" / "12 saat" / "45 dakika".
func KalanSure(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	g := int(d.Hours()) / 24
	s := int(d.Hours()) % 24
	dk := int(d.Minutes()) % 60
	switch {
	case g > 0 && s > 0:
		return itoa(g) + " gün " + itoa(s) + " saat"
	case g > 0:
		return itoa(g) + " gün"
	case s > 0:
		return itoa(s) + " saat"
	default:
		if dk < 1 {
			dk = 1
		}
		return itoa(dk) + " dakika"
	}
}

// ayarStr settings değerini metne çevirir (JSON string veya düz metin).
func ayarStr(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

// ayarBool settings boolean değerini okur; jsonb true/false, JSON string
// `"true"` ve düz `1`/`true` biçimleri kabul edilir. Panel checkbox'ı metin
// gönderebiliyor, SetSetting de geçersiz JSON'u string'e sarıyor — tek biçim
// varsaymak bandı sessizce kapatıyordu.
func ayarBool(raw []byte) bool {
	t := strings.TrimSpace(string(raw))
	t = strings.Trim(t, `"`)
	if t == "" || t == "null" {
		return false
	}
	var b bool
	if err := json.Unmarshal([]byte(t), &b); err == nil {
		return b
	}
	return t == "1" || strings.EqualFold(t, "true")
}
