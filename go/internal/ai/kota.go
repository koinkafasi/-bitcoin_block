package ai

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ---- Kota penceresi farkındalığı ----
//
// NEDEN (ölçüm, 13 Kasım 2026 — canlı `ai_jobs` son 24 saat):
//
//	saat   başarılı  hatalı
//	01:00     0        41     ← kota tükenmiş, hat körlemesine deniyor
//	02:00     0        19     ← aynı
//	03:00     2         0     ← KOTA SIFIRLANDI
//	04-18     bol       ~0
//	21-23    50→13     13↑    ← kota tükenmeye başlıyor
//
// İki ayrı kayıp vardı:
//
//  1. Kota bittikten sonra hat istek atmaya devam ediyordu. Sağlayıcı 429
//     dönüyor, iş `failed` işaretleniyor ve KAYBEDİLİYORDU — 24 saatte 118
//     iş (toplamın ~%22'si). Oysa o içerik bir saat sonra üretilebilirdi.
//
//  2. Kotanın en bol olduğu pencere (03:00 sonrası) boş geçiyordu: hat sabit
//     aralıkla çalıştığı için 04:00'te saatte 8 iş üretiyordu, 18:00'de 81.
//     Yani bedava kapasite kullanılmıyordu.
//
// Bu dosya ikisini de çözer: 429 ayrı bir hata tipi olarak tanınır (iş
// kaybedilmez, kuyrukta bekler) ve sıfırlama saati bilindiği için hat
// pencereye göre hızlanır.

// ErrKotaDoldu sağlayıcı 429 döndüğünde sarmalanan hatadır.
//
// Çağıran taraf bunu `errors.Is` ile ayırt edip işi BAŞARISIZ saymak
// yerine kuyrukta bırakmalıdır: kota bir sonraki sıfırlamada döner.
var ErrKotaDoldu = errors.New("AI kotası doldu")

// KotaSifirlamaSaati sağlayıcının günlük kotayı sıfırladığı saattir
// (yerel saat, KotaBolgesi'ne göre).
//
// 3 değeri ölçümle doğrulandı: 01:00 ve 02:00'de sıfır başarı varken
// 03:00'te hatalar tamamen kesildi.
const KotaSifirlamaSaati = 3

// KotaBolgesi sıfırlama saatinin yorumlandığı saat dilimidir.
const KotaBolgesi = "Europe/Istanbul"

// kotaKonum saat dilimini bir kez çözer. Yüklenemezse UTC+3 sabitine
// düşer: bölge veritabanı olmayan ince konteynerlerde (scratch/distroless)
// time.LoadLocation başarısız olur ve sıfırlama saati kayardı.
var kotaKonum = func() *time.Location {
	if l, err := time.LoadLocation(KotaBolgesi); err == nil {
		return l
	}
	return time.FixedZone("TRT", 3*60*60)
}()

// SonrakiSifirlama verilen andan SONRAKİ ilk sıfırlama anını döndürür.
//
// Tam sıfırlama saatinde çağrılırsa ERTESİ günü verir: "şu an sıfırlandı"
// ile "bir sonraki sıfırlamaya kadar bekle" karışmasın.
func SonrakiSifirlama(an time.Time) time.Time {
	yerel := an.In(kotaKonum)
	hedef := time.Date(yerel.Year(), yerel.Month(), yerel.Day(),
		KotaSifirlamaSaati, 0, 0, 0, kotaKonum)
	if !hedef.After(yerel) {
		hedef = hedef.AddDate(0, 0, 1)
	}
	return hedef
}

// KotaBolPencere kotanın taze olduğu ve hattın hızlanması gereken
// pencerede olup olmadığımızı söyler.
//
// Pencere sıfırlamadan başlar ve BolPencereSuresi kadar sürer. Ölçümde
// 03:00-09:00 arası hiç 429 görülmedi; o aralık güvenle doludur.
func KotaBolPencere(an time.Time) bool {
	yerel := an.In(kotaKonum)
	baslangic := time.Date(yerel.Year(), yerel.Month(), yerel.Day(),
		KotaSifirlamaSaati, 0, 0, 0, kotaKonum)
	if yerel.Before(baslangic) {
		return false
	}
	return yerel.Before(baslangic.Add(BolPencereSuresi))
}

// BolPencereSuresi sıfırlamadan sonra kotanın bol sayıldığı süredir.
//
// 6 saat: ölçümde 03:00-09:00 arası sıfır hata görüldü. Daha uzun tutmak
// (ör. 12 saat) öğleden sonraki yoğun saatlerde yanlış güven verirdi.
const BolPencereSuresi = 6 * time.Hour

// ---- Tükenme durumu ----

// KotaIzleyici 429'ları hatırlar; böylece kota bittiğinde hat boşa istek
// atmayı keser.
//
// Neden bellek içi: bu bilgi kısa ömürlü (en çok bir güne kadar) ve süreç
// yeniden başlarsa ilk 429 onu tekrar öğretir. Veritabanına yazmak her
// çağrıda fazladan gidiş-dönüş demekti.
type KotaIzleyici struct {
	mu        sync.RWMutex
	tukendiAt time.Time // son 429 anı (sıfır = hiç görülmedi)
	sayac     int       // bu pencerede kaç 429 görüldü (tanı için)
}

// NewKotaIzleyici yeni bir izleyici verir.
func NewKotaIzleyici() *KotaIzleyici { return &KotaIzleyici{} }

// Isaretle 429 görüldüğünü kaydeder.
func (k *KotaIzleyici) Isaretle(an time.Time) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tukendiAt = an
	k.sayac++
}

// Tukendi şu anda kotanın tükenmiş sayılıp sayılmayacağını söyler.
//
// Mantık: son 429'dan SONRA bir sıfırlama geçtiyse kota dönmüştür.
// Geçmediyse hâlâ tükenmiş sayılır ve çağrı yapılmamalıdır.
func (k *KotaIzleyici) Tukendi(an time.Time) bool {
	k.mu.RLock()
	son := k.tukendiAt
	k.mu.RUnlock()
	if son.IsZero() {
		return false
	}
	// Son 429'dan sonraki ilk sıfırlama henüz gelmediyse tükenmiş.
	return an.Before(SonrakiSifirlama(son))
}

// Temizle sıfırlama geçtiyse durumu sıfırlar ve temizlenip temizlenmediğini
// söyler. İşçiler bunu günlüğe "kota döndü" diye yazmak için kullanır.
func (k *KotaIzleyici) Temizle(an time.Time) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.tukendiAt.IsZero() || an.Before(SonrakiSifirlama(k.tukendiAt)) {
		return false
	}
	k.tukendiAt = time.Time{}
	k.sayac = 0
	return true
}

// Sayac bu pencerede görülen 429 sayısını verir (tanı/panel için).
func (k *KotaIzleyici) Sayac() int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.sayac
}

// BeklemeSuresi kota tükendiyse sıfırlamaya kalan süreyi verir; tükenmemişse
// sıfır döner.
func (k *KotaIzleyici) BeklemeSuresi(an time.Time) time.Duration {
	k.mu.RLock()
	son := k.tukendiAt
	k.mu.RUnlock()
	if son.IsZero() {
		return 0
	}
	kalan := SonrakiSifirlama(son).Sub(an)
	if kalan < 0 {
		return 0
	}
	return kalan
}

// kotaHatasi bir HTTP durum kodunu kota hatasına çevirir.
//
// 429 dışında 402 (ödeme gerekli) ve 403'ün kota metni taşıyan hâli de
// kota sayılır: OmniRoute arkasındaki sağlayıcılar kota bitince bu üçünden
// birini dönebiliyor.
func kotaHatasi(durum int, govde string) error {
	switch durum {
	case http.StatusTooManyRequests, http.StatusPaymentRequired:
		return fmt.Errorf("%w (HTTP %d): %s", ErrKotaDoldu, durum, govde)
	}
	return nil
}

// ---- İstemci üzerinden kota sorgusu ----
//
// İşçiler kota durumunu bu iki metotla okur: döngülerinde boşa çağrı
// yapmak yerine sıfırlamaya kadar beklemek için.

// KotaTukendi istemcinin gördüğü son 429'a göre kotanın tükenmiş sayılıp
// sayılmayacağını söyler. İstemci yoksa false (kota bilinmiyorsa dene).
func (c *Client) KotaTukendi() bool {
	if c == nil || c.kota == nil {
		return false
	}
	simdi := time.Now()
	// Sıfırlama geçtiyse durumu temizle: çağıran tekrar denemeli.
	c.kota.Temizle(simdi)
	return c.kota.Tukendi(simdi)
}

// KotaBeklemesi kota tükendiyse sıfırlamaya kalan süreyi verir.
func (c *Client) KotaBeklemesi() time.Duration {
	if c == nil || c.kota == nil {
		return 0
	}
	return c.kota.BeklemeSuresi(time.Now())
}

// KotaIhlalSayisi bu pencerede görülen 429 sayısını verir (panel/tanı).
func (c *Client) KotaIhlalSayisi() int {
	if c == nil || c.kota == nil {
		return 0
	}
	return c.kota.Sayac()
}
