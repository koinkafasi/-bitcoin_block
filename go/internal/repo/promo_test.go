package repo

import (
	"testing"
	"time"
)

// Promo bandı (§3) sözleşmesi.
//
// NEDEN TEST: bant settings'ten gelir ve yanlış yapılandırma sessizdir —
// geçmiş tarihli kampanya sayfada kalır, boş metin boş çerçeve basar,
// ya da geri sayım "0 dakika" yazar. Bu sınıf testle kapatılır.

// TestKalanSure: geri sayım metni kaba ve okunur olmalı.
func TestKalanSure(t *testing.T) {
	vaka := []struct {
		d     time.Duration
		bekle string
	}{
		{0, ""},
		{-time.Hour, ""},
		{30 * time.Second, "1 dakika"},
		{5 * time.Minute, "5 dakika"},
		{3 * time.Hour, "3 saat"},
		{26 * time.Hour, "1 gün 2 saat"},
		{72 * time.Hour, "3 gün"},
	}
	for _, v := range vaka {
		if got := KalanSure(v.d); got != v.bekle {
			t.Errorf("KalanSure(%v) = %q, beklenen %q", v.d, got, v.bekle)
		}
	}
}

// TestPromoBannerBos: kapalı veya metinsiz bant GÖRÜNMEZ olmalı.
//
// Boş çerçeve basmak düzeni bozar; ana sayfada sıfır yükseklikte bir
// çizgi kalır. Bu yüzden iki koşul birlikte aranır.
func TestPromoBannerBos(t *testing.T) {
	vaka := []PromoBanner{
		{},
		{Aktif: false, Metin: "Duyuru"},
		{Aktif: true, Metin: ""},
		{Aktif: true, Metin: "   "},
	}
	for i, p := range vaka {
		if !p.Bos() {
			t.Errorf("vaka %d: boş bant görünür sayıldı (%+v)", i, p)
		}
	}
	dolu := PromoBanner{Aktif: true, Metin: "Duyuru"}
	if dolu.Bos() {
		t.Error("aktif ve metinli bant boş sayıldı")
	}
}

// TestPromoAyarlardan: ayar haritası → bant dönüşümü.
//
// En kritik kural: SÜRESİ GEÇMİŞ kampanya sayfada kalmaz. Editör tarihi
// güncellemeyi unutsa bile bant kendiliğinden düşer.
func TestPromoAyarlardan(t *testing.T) {
	simdi := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	ayar := func(kv map[string]string) map[string][]byte {
		out := map[string][]byte{}
		for k, v := range kv {
			out[k] = []byte(v)
		}
		return out
	}

	// Kapalı.
	if p := promoAyarlardan(ayar(map[string]string{
		"aktif": "false", "metin": `"Duyuru"`,
	}), simdi); !p.Bos() {
		t.Error("kapalı bant görünür oldu")
	}

	// Açık, geri sayımsız.
	p := promoAyarlardan(ayar(map[string]string{
		"aktif": `"true"`, "metin": `"Duyuru"`,
		"baglanti": `"/newsletters"`, "baglanti_metin": `"Abone ol"`,
	}), simdi)
	if p.Bos() || p.Kalan != "" || p.CTA != "Abone ol" {
		t.Errorf("geri sayımsız bant yanlış: %+v", p)
	}

	// Gelecekte bitiş → geri sayım var.
	p = promoAyarlardan(ayar(map[string]string{
		"aktif": "true", "metin": "Duyuru", "bitis": `"2026-11-09T12:00"`,
	}), simdi)
	if p.Kalan != "3 gün" {
		t.Errorf("geri sayım = %q, beklenen \"3 gün\"", p.Kalan)
	}

	// Geçmiş bitiş → bant DÜŞER.
	p = promoAyarlardan(ayar(map[string]string{
		"aktif": "true", "metin": "Duyuru", "bitis": `"2026-11-05T12:00"`,
	}), simdi)
	if !p.Bos() {
		t.Errorf("süresi geçmiş bant hâlâ görünür: %+v", p)
	}

	// Bozuk tarih → bant kalır, geri sayım yok (editör hatası duyuruyu düşürmez).
	p = promoAyarlardan(ayar(map[string]string{
		"aktif": "true", "metin": "Duyuru", "bitis": `"yarın"`,
	}), simdi)
	if p.Bos() || p.Kalan != "" {
		t.Errorf("bozuk tarih davranışı yanlış: %+v", p)
	}
}

// TestAyarYardimcilari: settings değerleri hem jsonb hem düz metin okunmalı.
//
// Panel checkbox'ı "1"/"true" gönderebiliyor; eski kayıtlar düz metin
// olabilir. Tek biçim varsaymak bandı sessizce kapatıyordu.
func TestAyarYardimcilari(t *testing.T) {
	if !ayarBool([]byte("true")) || !ayarBool([]byte(`"true"`)) || !ayarBool([]byte("1")) {
		t.Error("true biçimleri okunamadı")
	}
	for _, ham := range []string{"false", `"false"`, "0", "", "null", `""`} {
		if ayarBool([]byte(ham)) {
			t.Errorf("%q yanlışlıkla true okundu", ham)
		}
	}
	if got := ayarStr([]byte(`"merhaba"`)); got != "merhaba" {
		t.Errorf("ayarStr json string = %q", got)
	}
	if got := ayarStr([]byte("duz metin")); got != "duz metin" {
		t.Errorf("ayarStr duz metin = %q", got)
	}
}
