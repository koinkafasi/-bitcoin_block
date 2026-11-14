package repo

import (
	"strings"
	"testing"
)

// Canlı `ai_jobs`tan 14 Kasım 2026'da OKUNAN gerçek hata metinleri.
// Elle sadeleştirilmiş örnekler bu hatayı yakalamazdı: mesele metnin
// gerçekte ne kadar uzun olduğu.
const (
	gercekZamanAsimi = `AI sağlayıcısına ulaşılamıyor: AI sağlayıcısına ulaşılamadı: ` +
		`Post "http://omniroute.omniroute.svc.cluster.local:20128/v1/chat/completions": ` +
		`context deadline exceeded (Client.Timeout exceeded while awaiting headers)`
	gercekSurecKapandi = `AI sağlayıcısına ulaşılamıyor: AI sağlayıcısına ulaşılamadı: ` +
		`Post "http://omniroute.omniroute.svc.cluster.local:20128/v1/chat/completions": ` +
		`terminated signal received`
	gercekYetki = `AI sağlayıcısına ulaşılamıyor: AI sağlayıcısına ulaşılamadı: ` +
		`AI uç noktası 502: {"error":{"message":"unc/turboderp/Qwen3.8-27B-exl3: auth — [401]: The`
)

// ASIL GERİLEME TESTİ. Panelin ilk sürümü left(error, 80) ile
// gruplıyordu; bu üç hata ilk 80 karakterde AYNI olduğu için tek satıra
// çöküyordu ve tablo hiçbir şey söylemiyordu.
func TestImzaAyniOnEkliHatalariAyirir(t *testing.T) {
	const onEk = 80
	a := []rune(gercekZamanAsimi)[:onEk]
	b := []rune(gercekSurecKapandi)[:onEk]
	if string(a) != string(b) {
		t.Fatalf("test varsayımı düştü: ilk %d karakter artık farklı.\n a=%q\n b=%q",
			onEk, string(a), string(b))
	}
	if got := UretimHataImzasi(gercekZamanAsimi); got != ImzaZamanAsimi {
		t.Errorf("zaman aşımı yanlış sınıflandı: %q", got)
	}
	if got := UretimHataImzasi(gercekSurecKapandi); got != ImzaSurecKapandi {
		t.Errorf("süreç kapanması yanlış sınıflandı: %q", got)
	}
	if UretimHataImzasi(gercekZamanAsimi) == UretimHataImzasi(gercekSurecKapandi) {
		t.Error("iki ayrı sebep hâlâ aynı kovada")
	}
}

// Zaman aşımı metni hem "Post http://..." hem "context deadline
// exceeded" içeriyor. Sıra yanlış olursa bağlam kovasına düşer ve
// "sağlayıcı yavaş" ile "biz iptal ettik" karışır.
func TestImzaZamanAsimiBaglamdanOnceGelir(t *testing.T) {
	if !strings.Contains(gercekZamanAsimi, "context deadline exceeded") {
		t.Fatal("test varsayımı: metin bağlam ifadesini de taşıyor")
	}
	if got := UretimHataImzasi(gercekZamanAsimi); got != ImzaZamanAsimi {
		t.Fatalf("Client.Timeout önce gelmeli, gelen: %q", got)
	}
}

// 502 gövdesinin içinde 401 var: dıştaki durum koduna değil, asıl
// sebebe bakılmalı — yoksa "sağlayıcı çökmüş" diye bakılır, oysa
// anahtar reddedilmiş.
func TestImzaIcIcEDurumKodundaYetkiKazanir(t *testing.T) {
	if got := UretimHataImzasi(gercekYetki); got != ImzaYetki {
		t.Fatalf("yetki beklendi, gelen: %q", got)
	}
}

func TestImzaYeniIcerikHatalari(t *testing.T) {
	bos := `omniroute/evren/deepseek-v4.1-flash: AI boş içerik döndürdü: yanıt yalnız ` +
		`düşünme adımında kaldı (reasoning 412 bayt, content boş, bitiş "length")`
	if got := UretimHataImzasi(bos); got != ImzaBosIcerik {
		t.Errorf("boş içerik beklendi, gelen: %q", got)
	}
	kesik := `omniroute/m: AI çıktısı token sınırında kesildi (4000 token üretildi)`
	if got := UretimHataImzasi(kesik); got != ImzaKesikCikti {
		t.Errorf("kesik çıktı beklendi, gelen: %q", got)
	}
	// Kesik çıktı metni "kesildi" diyor ama JSON da ayrıştırılamaz;
	// yanlış kovaya düşerse çare de yanlış olur.
	if UretimHataImzasi(kesik) == ImzaJSONDegil {
		t.Error("kesik çıktı 'JSON değil' sayılmamalı: çaresi bütçe, biçim değil")
	}
}

func TestImzaBilinmeyenDigerOlur(t *testing.T) {
	if got := UretimHataImzasi("tanınmayan bir arıza"); got != ImzaDiger {
		t.Fatalf("diğer beklendi, gelen: %q", got)
	}
	if got := UretimHataImzasi(""); got != ImzaDiger {
		t.Fatalf("boş metin diğer olmalı, gelen: %q", got)
	}
}

// Örnek metin iki ucu da korumalı: yalnız baş kısım hangi sebep
// olduğunu, yalnız son kısım hangi yol olduğunu gizler.
func TestHataOrnekIkiUcuKorur(t *testing.T) {
	o := uretimHataOrnek(gercekZamanAsimi)
	if !strings.Contains(o, "AI sağlayıcısına") {
		t.Errorf("baş taraf kayıp: %q", o)
	}
	if !strings.Contains(o, "awaiting headers") {
		t.Errorf("son taraf kayıp (sebep burada): %q", o)
	}
	if len([]rune(o)) >= len([]rune(gercekZamanAsimi)) {
		t.Errorf("kısaltma olmamış: %d >= %d", len([]rune(o)), len([]rune(gercekZamanAsimi)))
	}
	kisa := "kısa hata"
	if uretimHataOrnek(kisa) != kisa {
		t.Error("kısa metne dokunulmamalı")
	}
}
