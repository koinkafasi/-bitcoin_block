package httpapi

import (
	"strings"
	"testing"
	"time"
)

// İLETİŞİM FORMU TESTLERİ — 14 Kasım 2026'da ölçülen iki kusurun
// geri dönmemesini garanti eder.

// TestContactHizSiniriDogrulamadanSonra: onay kutusunu unutan kullanıcı
// hatasını düzeltip hemen tekrar gönderdiğinde ENGELLENMEMELİ.
//
// ÖLÇÜLEN KUSUR: hız sınırı doğrulamadan önce çalışıyor ve damgayı her
// denemede vuruyordu. Ölçüm: 1. deneme (onay yok) → "onaylayın",
// 2. deneme (1 sn sonra, onay var) → "Çok sık gönderim". Kullanıcı için
// form tamamen kilitliydi.
func TestContactHizSiniriDogrulamadanSonra(t *testing.T) {
	ip := "198.51.100.201"
	// Başarılı gönderim yok → damga vurulmamış olmalı.
	if limited, _ := contactRateLimited(ip); limited {
		t.Fatal("hiç gönderim yokken hız sınırı uygulanmamalı")
	}
	// Başarısız denemeler damga vurmaz: art arda 3 kontrol de serbest.
	for i := 0; i < 3; i++ {
		if limited, _ := contactRateLimited(ip); limited {
			t.Fatalf("doğrulama hatası sonrası %d. kontrol kilitledi", i+1)
		}
	}
	// Başarılı gönderim damgayı vurur.
	contactHizDamgala(ip)
	limited, kalan := contactRateLimited(ip)
	if !limited {
		t.Fatal("başarılı gönderimden sonra hız sınırı uygulanmalı")
	}
	if kalan <= 0 || kalan > contactHizSiniri {
		t.Fatalf("kalan süre saçma: %v", kalan)
	}
}

// TestContactHizSiniriBosaDoner damganın süresi dolunca serbest kalır.
func TestContactHizSiniriBosaDoner(t *testing.T) {
	ip := "198.51.100.202"
	contactHizDamgala(ip)
	// Süresi geçmiş damga: eski sürüm kilitli kalırdı, yeni okur.
	contactLimiter.mu.Lock()
	contactLimiter.last[ip] = time.Now().Add(-contactHizSiniri - time.Second)
	contactLimiter.mu.Unlock()
	if limited, _ := contactRateLimited(ip); limited {
		t.Fatal("süresi dolmuş damga kilit üretmemeli")
	}
}

func TestTemizMetin(t *testing.T) {
	// Kontrol karakterleri atılır; satır sonu/sekme korunur.
	girdi := "Satir1\x07\x00SELAM\x1b[31m\nIkinci\tsatir"
	got := temizMetin(girdi, 200)
	if strings.ContainsAny(got, "\x07\x00\x1b") {
		t.Fatalf("kontrol karakteri kaldı: %q", got)
	}
	if !strings.Contains(got, "\n") || !strings.Contains(got, "\t") {
		t.Fatalf("satır sonu/sekme korunmalı: %q", got)
	}
	// Uzunluk rune bazında kırpılır (çok baytlı karakter ortadan kesilmez).
	uzun := strings.Repeat("ğ", 50)
	if n := len([]rune(temizMetin(uzun, 10))); n != 10 {
		t.Fatalf("kırpma rune bazında olmalı, %d rune", n)
	}
	// Baş/son boşluk atılır.
	if got := temizMetin("  merhaba  ", 50); got != "merhaba" {
		t.Fatalf("trim = %q", got)
	}
}

func TestBaglantiSayisi(t *testing.T) {
	ornekler := map[string]int{
		"merhaba":                       0,
		"http://a.com":                  1,
		"http://a.com ve https://b.com": 2,
		"www.a.com www.b.com www.c.com": 3,
		"HTTP://A.COM http://b.com http://c.com http://d.com": 4,
	}
	for metin, bekle := range ornekler {
		if got := baglantiSayisi(metin); got != bekle {
			t.Errorf("baglantiSayisi(%q) = %d, beklenen %d", metin, got, bekle)
		}
	}
}

// TestContactKonulariFormdaVar: formdaki her konu sunucuda tanınmalı.
//
// ÖLÇÜLEN KUSUR: konu serbest metin kutusuydu; kullanıcı "Reklam" yazınca
// sunucu tanımıyor ve mesaj SESSİZCE "diger"e düşüyordu.
func TestContactKonulariFormdaVar(t *testing.T) {
	konular := contactTopics()
	if len(konular) < 5 {
		t.Fatalf("konu sayısı az: %d", len(konular))
	}
	for _, k := range konular {
		if strings.TrimSpace(k.Key) == "" || strings.TrimSpace(k.Name) == "" {
			t.Errorf("boş konu anahtarı/adı: %+v", k)
		}
		if k.Key == "spam" {
			t.Error("'spam' formda SEÇENEK OLMAMALI (yalnız panel etiketi)")
		}
	}
	// "diger" her zaman bulunmalı: bilinmeyen değerlerin düştüğü yer.
	bulundu := false
	for _, k := range konular {
		if k.Key == "diger" {
			bulundu = true
		}
	}
	if !bulundu {
		t.Error("'diger' konusu yok")
	}
}
