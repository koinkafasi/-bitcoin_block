package scoring

// Yayın önceliği: tek sayı yerine ayrı eksenler.
//
// # Neden
//
// Eski yapıda tek bir `ai_score` vardı ve o da yalnız üstveri eksiksizliğini
// ölçüyordu (bkz. Explain). 29 Ekim 2026 ölçümü sonucu: 34.156 taslağın
// %87,9'u tek bir 60–79 bandında. Tek sayı hem "yayına hazır mı" hem
// "yayınlamaya değer mi" sorusunu aynı anda yanıtlamaya çalışıyordu ve
// ikisini de yanıtlayamıyordu.
//
// Bu dosya dört ekseni ayırır. Her biri farklı bir soruya yanıt verir ve
// panelde ayrı ayrı gösterilebilir:
//
//	Tamlık   — Yayına hazır mı?        (başlık/spot/etiket/kapak/kaynak)
//	Trend    — Şu an aranıyor mu?      (trend_matches + trend_terms)
//	Güven    — Kaynağa güveniliyor mu? (feeds.trust)
//	Tazelik  — Hâlâ haber mi?          (created_at)
//
// Öncelik bunların ağırlıklı birleşimidir ve triyaj sıralaması için kullanılır.
//
// # Ağırlıklar neden böyle
//
// Ağırlıklar ölçülen DAĞILIMA göre seçildi, sezgiyle değil. Bir eksenin
// sıralamaya katkısı, ağırlığı × o eksenin gerçek yayılımı kadardır:
//
//	Trend   45 — taslakların çok azı eşleşir, ama eşleşen gerçekten öne
//	             çıkmalı. Yayılım: 0 ya da 60–100. Sıralamada en belirleyici.
//	Tazelik 35 — 0–182 saate düzgün yayılmış (ölçüldü). Eşleşmeyen
//	             taslaklar arasındaki asıl ayırıcı bu.
//	Güven   10 — feeds.trust 60–90; gerçek yayılımı dar, o yüzden ağırlığı da.
//	Tamlık  10 — en dar yayılım (%88'i 60–79). Yalnız eşitlik bozucu.
//
// İlk denemede Tamlık 15 + Güven 20 idi; ölçünce görüldü ki bu iki eksen
// korpusta neredeyse sabit olduğu için ağırlıklarının %35'i sıralamaya hiç
// katkı vermiyor, yalnız puan aralığını daraltıyordu. Ağırlık, eksenin
// GERÇEK yayılımına göre verilir — yoksa eski ai_score'un hatası tekrarlanır.
//
// Tamlık asıl işini yayın KAPISINDA yapar (categories.min_score), sıralamada
// değil. İki iş ayrıldığı için ikisi de düzgün çalışır.

import (
	"math"
	"time"
)

// OncelikGirdi bir içeriğin dört eksenini hesaplamak için gereken her şeydir.
// Paket veritabanı bilmez; bu alanları çağıran taraf doldurur.
type OncelikGirdi struct {
	Tamlik Input // başlık/spot/etiket/kapak/kaynak sinyalleri

	// Trend uyumu. TrendEslesme, trend_matches.score değeridir (0–100);
	// eşleşme yoksa 0. TrendSonGorulme, eşleşen terimin last_seen_at'idir:
	// üç gün önce popüler olmuş bir terim bugün için sinyal sayılmaz.
	TrendEslesme    int
	TrendSonGorulme time.Time
	TrendTrafik     int // trend_terms.approx_traffic; 0 = bilinmiyor

	// Kaynak güveni. feeds.trust (0–100). Agregatör akışları (Google News
	// gibi) özgün kaynak değildir, bu yüzden ayrıca cezalandırılır.
	KaynakGuven int
	Agregator   bool

	// İçeriğin yaşı için referans noktaları.
	OlusturmaZamani time.Time
	Simdi           time.Time // sıfırsa time.Now() kullanılır
}

// Eksenler dört ekseni ve birleşik önceliği taşır. Hepsi 0–100.
type Eksenler struct {
	Tamlik  int `json:"tamlik"`
	Trend   int `json:"trend"`
	Guven   int `json:"guven"`
	Tazelik int `json:"tazelik"`
	Oncelik int `json:"oncelik"`
}

// Ağırlıklar tek yerde; değiştirilirse kayıtlı öncelikler yeniden
// hesaplanmalıdır (cmd/backfill -is=oncelik).
const (
	AgirlikTrend   = 45
	AgirlikTazelik = 35
	AgirlikGuven   = 10
	AgirlikTamlik  = 10
)

// Tazelik iki parçalı sönüm kullanır: ilk gün hızlı, sonrası doğrusal.
//
// Önce üstel sönüm (24 saat yarı ömür) denendi ve ÖLÇÜLDÜ: taslak yaşları
// 0–182 saate yayıldığı için üstel sönüm 72 saatten sonra herkesi ~0'a
// indiriyor ve eksen korpusun çoğunda ayırt etmeyi bırakıyordu — öncelik
// yine tek bantta (%61) toplanıyordu, yani eski ai_score'un hatası tekrar
// ediyordu. İki parçalı eğri ikisini birden verir:
//
//	 0–24 saat : 100 → 70   (haber değeri ilk gün hızlı düşer)
//	24–168 saat:  70 →  0   (kuyrukta sıralama yapmayı sürdürür)
const (
	tazelikIlkGun   = 24 * time.Hour
	tazelikSinir    = 7 * 24 * time.Hour
	tazelikIlkGunDe = 70.0 // ilk günün sonundaki değer
)

// Tazelik yaşa göre 100'den 0'a iner.
func Tazelik(yas time.Duration) int {
	if yas <= 0 {
		return 100
	}
	if yas >= tazelikSinir {
		return 0
	}
	var v float64
	if yas <= tazelikIlkGun {
		v = 100 - (100-tazelikIlkGunDe)*(yas.Hours()/tazelikIlkGun.Hours())
	} else {
		kalan := (yas - tazelikIlkGun).Hours() / (tazelikSinir - tazelikIlkGun).Hours()
		v = tazelikIlkGunDe * (1 - kalan)
	}
	return clamp(int(math.Round(v)))
}

// trendTazelikSiniri: eşleşilen terim bu süreden eskiyse trend sinyali
// sayılmaz. Google Trends "şu an" demektir; dünkü trend bugünün trendi değil.
const trendTazelikSiniri = 48 * time.Hour

// TrendUyumu eşleşme gücünü terimin güncelliğiyle ve trafiğiyle ağırlıklar.
//
// Eşleşme yoksa 0 döner — ve bu doğrudur: taslakların çoğu hiçbir trend
// terimiyle eşleşmez. Eksenin bimodal olması kusur değil, amaçtır; eşleşen
// içerik sıralamada gerçekten yukarı çıkmalıdır.
func TrendUyumu(eslesme int, sonGorulme, simdi time.Time, trafik int) int {
	if eslesme <= 0 {
		return 0
	}
	taze := 1.0
	if !sonGorulme.IsZero() {
		yas := simdi.Sub(sonGorulme)
		switch {
		case yas <= 0:
			taze = 1.0
		case yas >= trendTazelikSiniri:
			return 0
		default:
			taze = 1 - yas.Hours()/trendTazelikSiniri.Hours()
		}
	}
	// Trafik bonusu küçük tutuldu: approx_traffic kaynakta sık sık 0
	// geliyor (2.053 terimin 217'sinde yok) ve 0 olması terimin önemsiz
	// olduğu anlamına gelmiyor. Bu yüzden çarpan değil, ek puan.
	bonus := 0.0
	if trafik >= 50000 {
		bonus = 10
	} else if trafik >= 10000 {
		bonus = 5
	}
	return clamp(int(math.Round(float64(clamp(eslesme))*taze + bonus)))
}

// KaynakGuveni feeds.trust değerini kullanır. Bilinmeyen kaynak 50 sayılır:
// ne ödüllendirilir ne cezalandırılır. Agregatör akışları özgün kaynak
// olmadığı için 15 puan düşer.
func KaynakGuveni(trust int, agregator bool) int {
	v := trust
	if v <= 0 {
		v = 50
	}
	if agregator {
		v -= 15
	}
	return clamp(v)
}

// Hesapla dört ekseni ve birleşik önceliği üretir.
func Hesapla(g OncelikGirdi) Eksenler {
	simdi := g.Simdi
	if simdi.IsZero() {
		simdi = time.Now()
	}
	var yas time.Duration
	if !g.OlusturmaZamani.IsZero() {
		yas = simdi.Sub(g.OlusturmaZamani)
	}

	tamlik, _ := Explain(g.Tamlik)
	e := Eksenler{
		Tamlik:  tamlik,
		Trend:   TrendUyumu(g.TrendEslesme, g.TrendSonGorulme, simdi, g.TrendTrafik),
		Guven:   KaynakGuveni(g.KaynakGuven, g.Agregator),
		Tazelik: Tazelik(yas),
	}
	toplam := e.Trend*AgirlikTrend + e.Tazelik*AgirlikTazelik +
		e.Guven*AgirlikGuven + e.Tamlik*AgirlikTamlik
	e.Oncelik = clamp(int(math.Round(float64(toplam) /
		float64(AgirlikTrend+AgirlikTazelik+AgirlikGuven+AgirlikTamlik))))
	return e
}

// TazelikSinirSaat tazeliğin sıfırlandığı yaşı saat olarak verir.
// Çağıranlar bu değeri kendi sorgularını sınırlamak için kullanır;
// sabiti kopyalamak yerine buradan almaları iki tarafın ayrışmasını önler.
func TazelikSinirSaat() int {
	return int(tazelikSinir.Hours())
}
