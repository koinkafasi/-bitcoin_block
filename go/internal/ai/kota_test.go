package ai

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

// trt testlerde kullanılan sabit saat dilimi. Bölge veritabanı olmayan
// ortamlarda da aynı sonucu versin diye sabit ofset.
var trt = time.FixedZone("TRT", 3*60*60)

func an(gun, saat, dakika int) time.Time {
	return time.Date(2026, 11, gun, saat, dakika, 0, 0, trt)
}

func TestSonrakiSifirlamaAyniGun(t *testing.T) {
	// 01:30'dayız; sıfırlama AYNI günün 03:00'ü.
	got := SonrakiSifirlama(an(13, 1, 30))
	istenen := an(13, 3, 0)
	if !got.Equal(istenen) {
		t.Fatalf("01:30 için sıfırlama %s, beklenen %s", got, istenen)
	}
}

func TestSonrakiSifirlamaErtesiGun(t *testing.T) {
	// 21:00'dayız; 03:00 geçti, sıfırlama ERTESİ gün.
	got := SonrakiSifirlama(an(13, 21, 0))
	istenen := an(14, 3, 0)
	if !got.Equal(istenen) {
		t.Fatalf("21:00 için sıfırlama %s, beklenen %s", got, istenen)
	}
}

func TestSonrakiSifirlamaTamSaatteErtesiGun(t *testing.T) {
	// TAM 03:00'te: "şu an sıfırlandı" ile "bir sonrakine kadar bekle"
	// karışmasın — ertesi günü vermeli. Aksi hâlde bekleme süresi sıfır
	// çıkıp döngü aynı anda tekrar denerdi.
	got := SonrakiSifirlama(an(13, 3, 0))
	istenen := an(14, 3, 0)
	if !got.Equal(istenen) {
		t.Fatalf("tam 03:00 için %s, beklenen %s", got, istenen)
	}
}

func TestKotaBolPencere(t *testing.T) {
	testler := []struct {
		ad    string
		zaman time.Time
		istek bool
		neden string
	}{
		{"02:59 sıfırlama öncesi", an(13, 2, 59), false, "kota henüz dönmedi"},
		{"03:00 sıfırlama anı", an(13, 3, 0), true, "pencere başlıyor"},
		{"06:00 pencere ortası", an(13, 6, 0), true, "ölçümde 0 hata"},
		{"08:59 pencere sonu", an(13, 8, 59), true, "6 saat içinde"},
		{"09:00 pencere bitti", an(13, 9, 0), false, "6 saat doldu"},
		{"18:00 akşam", an(13, 18, 0), false, "kota tükeniyor"},
		{"00:30 gece yarısı", an(13, 0, 30), false, "sıfırlamadan önce"},
	}
	for _, tt := range testler {
		if got := KotaBolPencere(tt.zaman); got != tt.istek {
			t.Errorf("%s: KotaBolPencere=%v, beklenen %v (%s)",
				tt.ad, got, tt.istek, tt.neden)
		}
	}
}

func TestKotaIzleyiciSifirlamaIleTemizlenir(t *testing.T) {
	k := NewKotaIzleyici()

	// Başta tükenmiş değil.
	if k.Tukendi(an(13, 1, 0)) {
		t.Fatal("hiç 429 görülmeden tükenmiş sayıldı")
	}

	// 01:00'de 429 geldi → tükenmiş.
	k.Isaretle(an(13, 1, 0))
	if !k.Tukendi(an(13, 1, 30)) {
		t.Fatal("429'dan sonra 01:30'da tükenmiş sayılmadı")
	}
	if !k.Tukendi(an(13, 2, 59)) {
		t.Fatal("sıfırlamadan hemen önce tükenmiş sayılmadı")
	}

	// 03:00 sıfırlaması geçti → artık tükenmiş DEĞİL.
	if k.Tukendi(an(13, 3, 1)) {
		t.Fatal("sıfırlama geçtiği hâlde hâlâ tükenmiş sayıldı")
	}

	// Temizle sıfırlama sonrası true dönmeli (bir kez).
	if !k.Temizle(an(13, 3, 1)) {
		t.Fatal("Temizle sıfırlama sonrası false döndü")
	}
	if k.Temizle(an(13, 3, 2)) {
		t.Fatal("Temizle ikinci kez true döndü (durum zaten boş)")
	}
}

func TestKotaIzleyiciBeklemeSuresi(t *testing.T) {
	k := NewKotaIzleyici()
	if k.BeklemeSuresi(an(13, 1, 0)) != 0 {
		t.Fatal("429 yokken bekleme sıfır olmalı")
	}
	k.Isaretle(an(13, 1, 0))
	// 01:00'de 429; 01:00'de bakınca 03:00'e 2 saat kalmış.
	if got := k.BeklemeSuresi(an(13, 1, 0)); got != 2*time.Hour {
		t.Fatalf("bekleme %s, beklenen 2h", got)
	}
	// Sıfırlama geçtiyse negatif değil, sıfır.
	if got := k.BeklemeSuresi(an(13, 5, 0)); got != 0 {
		t.Fatalf("sıfırlama sonrası bekleme %s, beklenen 0", got)
	}
}

func TestKotaHatasiDurumKodlari(t *testing.T) {
	// 429 ve 402 kota sayılır; diğerleri sayılmaz.
	if err := kotaHatasi(http.StatusTooManyRequests, "rate limit"); !errors.Is(err, ErrKotaDoldu) {
		t.Error("429 kota hatası olarak tanınmadı")
	}
	if err := kotaHatasi(http.StatusPaymentRequired, "ödeme"); !errors.Is(err, ErrKotaDoldu) {
		t.Error("402 kota hatası olarak tanınmadı")
	}
	// 500 kota DEĞİL: sunucu hatası tekrar denemeyle düzelmez, iş
	// gerçekten başarısız olmalı.
	if err := kotaHatasi(http.StatusInternalServerError, "patladı"); err != nil {
		t.Errorf("500 kota sayıldı: %v", err)
	}
	if err := kotaHatasi(http.StatusOK, ""); err != nil {
		t.Errorf("200 kota sayıldı: %v", err)
	}
}

// Kota hatası sarmalandığında errors.Is zinciri korunmalı: işçiler bunu
// `failed` ile `ertelendi` ayrımı için kullanıyor.
func TestKotaHatasiSarmalamaZinciri(t *testing.T) {
	ham := kotaHatasi(http.StatusTooManyRequests, "quota exceeded")
	sarmal := errors.New("üst katman: " + ham.Error())
	if errors.Is(sarmal, ErrKotaDoldu) {
		t.Fatal("düz metin kopyası kota hatası sayılmamalı (%w gerekli)")
	}
	// Doğru sarmalama:
	dogru := errWrap(ham)
	if !errors.Is(dogru, ErrKotaDoldu) {
		t.Fatal("%w ile sarmalanan kota hatası zinciri kopmuş")
	}
}

func errWrap(err error) error { return errors.Join(errors.New("işçi"), err) }
