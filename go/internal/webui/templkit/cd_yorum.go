package templkit

import (
	"strconv"
	"strings"
)

// ---- Makale yorum bölümü görünüm modelleri ----

// CDYorumBolumu makale altındaki yorum alanı.
type CDYorumBolumu struct {
	// ContentID form hedefi (/yorum/{id}).
	ContentID string
	// Acik içerik yorumlara kapatılmadıysa true. Kapalıysa bölüm hiç
	// basılmaz — "yorumlar kapalı" yazısı da gürültü.
	Acik bool
	// UyeGirisli oturum açıksa form, değilse giriş daveti basılır.
	UyeGirisli bool
	UyeAdi     string
	// Sayi onaylı yorum sayısı (yanıtlar dahil). Sıfırsa başlıkta
	// gösterilmez.
	Sayi     int
	Yorumlar []CDYorum
}

// CDYorum tek yorum (ve altındaki yanıtlar).
type CDYorum struct {
	ID       string
	Ad       string
	BasHarf  string
	Metin    string
	Zaman    string
	Begeni   int
	Yanitlar []CDYorum
}

// cdYorumBaslik "12 yorum" gibi başlık üretir.
func cdYorumBaslik(n int) string {
	return strconv.Itoa(n) + " yorum"
}

// cdYorumFormBasligi yazana adıyla sesleniyor; "Yorum yaz" demekten
// daha davetkâr ve kimin adına yazıldığını da gösteriyor.
func cdYorumFormBasligi(ad string) string {
	if strings.TrimSpace(ad) == "" {
		return "Yorumun"
	}
	return ad + " olarak yazıyorsun"
}

func cdSayi(n int) string { return strconv.Itoa(n) }

func cdBegeniMetni(n int) string {
	return strconv.Itoa(n) + " kişi katılıyor"
}

// cdDilKodu <html lang> için güvenli dil kodu. Boş gelirse Türkçe
// varsayılır (sitenin ana dili).
func cdDilKodu(l string) string {
	l = strings.TrimSpace(l)
	if l == "" {
		return "tr"
	}
	return l
}

// cdXDefault hreflang="x-default" hedefi: varsa Türkçe sürüm, yoksa
// sayfanın kendi kanonik adresi.
func cdXDefault(h CDHead) string {
	if u, ok := h.Hreflang["tr"]; ok && strings.TrimSpace(u) != "" {
		return u
	}
	return h.Canonical
}

// cdGriSinif görsel sarmalayıcısına eklenecek sınıfı döndürür.
//
// Siyah-beyaz gösterim CSS filtresiyle yapılıyor: görsel dosyası tek
// kopya, yazı sayfasında aynı dosya renkli basılıyor. Sunucuda ikinci
// bir görsel üretmek depolama ve işlem maliyeti getirirdi.
func cdGriSinif(temel string, gri bool) string {
	if !gri {
		return temel
	}
	if temel == "" {
		return "cd-gri"
	}
	return temel + " cd-gri"
}
