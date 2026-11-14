package templkit

import (
	"os"
	"testing"
)

// TestYazOrnekHTML gerçek tarayıcı ölçümü için örnek sayfa üretir.
// Yalnız TZ_ORNEK_HTML ortam değişkeni verildiğinde yazar; normal test
// koşusunda hiçbir şey yapmaz (kapıyı kirletmesin).
func TestYazOrnekHTML(t *testing.T) {
	yol := os.Getenv("TZ_ORNEK_HTML")
	if yol == "" {
		t.Skip("TZ_ORNEK_HTML yok")
	}
	v := TrafikZiyaretciView{
		Anahtar:     "ip:5.9.1.17",
		Var:         true,
		IP:          "5.9.1.17",
		IPAgg:       "5.9.0.0/16",
		Ulke:        "Almanya",
		UlkeKod:     "DE",
		ASN:         24940,
		Saglayici:   "HETZNER-AS",
		ASNAd:       "HETZNER-AS - Hetzner Online GmbH, DE",
		Tur:         "hosting",
		TurAd:       "veri merkezi",
		Sinif:       "bot",
		SinifAd:     "Bot",
		Puan:        92,
		IstekSayisi: 1841,
		IPDegisim:   4,
		UlkeDegisim: 2,
		IlkGorulme:  "13.11.2026 21:04",
		SonGorulme:  "14.11.2026 03:12",
		UV:          "python-requests",
		UserAgent:   "python-requests/2.31.0",
		WebRTCIP:    "198.51.100.***",
		Hosting:     true,
		Bot:         true,

		Sinyaller: []string{"Veri merkezi ağı", "Otomasyon aracı (UA)",
			"WebRTC IP sızıntısı", "İstek hızı aşımı"},

		NiyetBaslik: "Zafiyet taraması",
		NiyetAciklama: "39 istek bilinen açık tarama yoluna gitti (wp-login, .env, " +
			"phpmyadmin gibi). Meşru okur bu adresleri istemez.",
		NiyetSeviye: "tehdit",
		OlaySayisi:  212,
		Engellenen:  198,

		Turler: []ZiyaretciTurSatir{
			{Kod: "zafiyet", Ad: "zafiyet taraması", Adet: 39, Pay: 18},
			{Kod: "icerik", Ad: "içerik okuma", Adet: 170, Pay: 80},
			{Kod: "api", Ad: "API", Adet: 3, Pay: 2},
		},
		Yollar: []ZiyaretciYolSatir{
			{Yol: "/wp-login.php", Adet: 12, Durum: "403"},
			{Yol: "/.env", Adet: 9, Durum: "403"},
			{Yol: "/haber/bitcoin-100-bin-dolar", Adet: 7, Durum: "200"},
			{Yol: "/phpmyadmin/index.php", Adet: 6, Durum: "403"},
		},
		Akrabalar: []ZiyaretciAkrabaSatir{
			{Anahtar: "ip:5.9.1.18", Adet: 44, Son: "14.11.2026 03:10"},
			{Anahtar: "9f2c1d44-1a2b-4c3d-8e9f-0a1b2c3d4e5f", Adet: 12, Son: "14.11.2026 02:51"},
		},
	}
	for i := 0; i < 12; i++ {
		o := ZiyaretciOlaySatir{
			Zaman: "14.11 03:12:04", Yontem: "GET", Yol: "/wp-login.php",
			Durum: 403, Tur: "zafiyet", TurAd: "zafiyet taraması",
			Kaynak: "yok", KaynakAd: "doğrudan", Engellendi: true,
		}
		if i%3 == 0 {
			o.Yol, o.Tur, o.TurAd, o.Durum, o.Engellendi =
				"/haber/bitcoin-100-bin-dolar", "icerik", "içerik okuma", 200, false
			o.Kaynak, o.KaynakAd = "arama", "arama motoru"
			o.Referer = "https://www.google.com/search?q=bitcoin+100+bin"
		}
		if i%5 == 0 {
			o.Sorgu = "q=bitcoin&sayfa=2&token=***"
		}
		v.Olaylar = append(v.Olaylar, o)
	}

	var sb []byte
	buf := &yazici{}
	if err := TrafikZiyaretciPanel(v).Render(t.Context(), buf); err != nil {
		t.Fatalf("çizilemedi: %v", err)
	}
	sb = buf.b
	if err := os.WriteFile(yol, sb, 0o644); err != nil {
		t.Fatalf("yazılamadı: %v", err)
	}
	t.Logf("%d bayt yazıldı: %s", len(sb), yol)
}

type yazici struct{ b []byte }

func (y *yazici) Write(p []byte) (int, error) {
	y.b = append(y.b, p...)
	return len(p), nil
}
