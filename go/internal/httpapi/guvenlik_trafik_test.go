package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TRAFİK GÜVENLİĞİ TESTLERİ — skor, sınıflandırma ve sayım kapısı.
//
// Bu testlerin hepsi SAF fonksiyonları sınar (DNS/DB yok): sınıflandırma
// anahtar kelimeleri, puan ağırlıkları, eşikler, sayım kapısı ve
// saat dilimi/kıta tutarlılığı.

func TestASNTurSiniflandirma(t *testing.T) {
	// ÖLÇÜLEN gerçek AS adları (Team Cymru, 14 Kasım 2026):
	//   8.8.8.8     → "GOOGLE - Google LLC, US"
	//   5.9.1.1     → "HETZNER-AS - Hetzner Online GmbH, DE"
	//   45.83.89.1  → "M247 - M247 Europe SRL, RO"
	//   146.70.1.1  → "SECFIREWALLAS - F.N.S. HOLDINGS LIMITED, CY" (NordVPN)
	//   51.38.1.1   → "OVH - OVH SAS, FR"
	//   104.131.1.1 → "DIGITALOCEAN-ASN - DigitalOcean, LLC, US"
	ornekler := []struct {
		ad    string
		bekle string
	}{
		{"HETZNER-AS - Hetzner Online GmbH, DE", "hosting"},
		{"DIGITALOCEAN-ASN - DigitalOcean, LLC, US", "hosting"},
		{"OVH - OVH SAS, FR", "hosting"},
		{"M247 - M247 Europe SRL, RO", "vpn"}, // VPN anahtarı önce denenir
		{"SECFIREWALLAS - F.N.S. HOLDINGS LIMITED, CY", "vpn"},
		{"AMAZON-AES - Amazon.com, Inc., US", "hosting"},
		{"UFO-TECHNOLOGIES-LIMITED - UFO TECHNOLOGIES LIMITED, GB", "hosting"},
		{"TURKCELL - Turkcell Iletisim Hizmetleri A.S.", "mobil"},
		{"SUPERONLINE - Superonline Iletisim Hizmetleri A.S.", "mobil"},
		{"TTNET - Turk Telekomunikasyon A.S.", "mobil"},
		{"GOOGLE - Google LLC, US", "konut"},
		{"", "bilinmiyor"},
	}
	for _, o := range ornekler {
		if got := asnTur(o.ad); got != o.bekle {
			t.Errorf("asnTur(%q) = %q, beklenen %q", o.ad, got, o.bekle)
		}
	}
}

func TestASNCoz(t *testing.T) {
	// Team Cymru'nun döndürdüğü GERÇEK biçim.
	asn, ag, ulke, ok := asnCoz(`15169 | 8.8.8.0/24 | US | arin | 2023-12-01`)
	if !ok || asn != 15169 || ag != "8.8.8.0/24" || ulke != "US" {
		t.Fatalf("asnCoz = %d %q %q %v", asn, ag, ulke, ok)
	}
	if _, _, _, ok := asnCoz("çöp"); ok {
		t.Fatal("çöp cevap geçerli sayıldı")
	}
	if _, _, _, ok := asnCoz(""); ok {
		t.Fatal("boş cevap geçerli sayıldı")
	}
	// AS adı: son parça marka adıdır.
	if ad := asnAdCoz(`15169 | US | arin | 2000-03-30 | GOOGLE - Google LLC, US`); ad != "GOOGLE - Google LLC, US" {
		t.Errorf("asnAdCoz = %q", ad)
	}
}

func TestASNAnahtarTersIP(t *testing.T) {
	if got := asnAnahtar("8.8.8.8"); got != "8.8.8.0/24" {
		t.Errorf("asnAnahtar = %q", got)
	}
	if got := tersIP("8.8.8.8"); got != "8.8.8.8" {
		t.Errorf("tersIP tekrar simetrik olmalı, %q", got)
	}
	if got := tersIP("104.131.1.1"); got != "1.1.131.104" {
		t.Errorf("tersIP = %q, beklenen 1.1.131.104", got)
	}
	if got := asnAnahtar("5.9.1.1"); got != "5.9.1.0/24" {
		t.Errorf("asnAnahtar = %q", got)
	}
	if got := asnAnahtar("çöp"); got != "" {
		t.Errorf("geçersiz IP anahtar üretmemeli, %q", got)
	}
}

func TestSkorEsikleriVeSayim(t *testing.T) {
	pol := guvenlikPolitikasi{Aktif: true, BotEngel: true, HostingEngel: true}
	// 1) Temiz: hiç sinyal yok.
	k := guvenlikHesapla(guvenlikGirdi{Anahtar: "a", EsikYuksek: 60, EsikBot: 85, Politika: pol})
	if k.Puan != 0 || k.Sinif != "temiz" || !k.Sayilir || !k.Reklam || k.Engelle {
		t.Fatalf("temiz karar yanlış: %+v", k)
	}

	// 2) Tek zayıf sinyal (dil yok = 10) kimseyi dışlamaz.
	k = guvenlikHesapla(guvenlikGirdi{Anahtar: "a", DilYok: true, EsikYuksek: 60, EsikBot: 85, Politika: pol})
	if !k.Sayilir || k.Puan != pDilYok {
		t.Fatalf("dil yok tek başına dışlamamalı: %+v", k)
	}

	// 3) Proxy başlığı tek başına (35) → şüpheli, sayılmaz.
	k = guvenlikHesapla(guvenlikGirdi{Anahtar: "a", ProxyBaslik: true, EsikYuksek: 60, EsikBot: 85, Politika: pol})
	if k.Sinif != "supheli" || k.Sayilir || k.Reklam || k.Engelle {
		t.Fatalf("proxy başlığı şüpheli olmalı: %+v", k)
	}

	// 4) Otomasyon UA (80) → bot, engellenir.
	k = guvenlikHesapla(guvenlikGirdi{Anahtar: "a", UAOtomasyon: true, EsikYuksek: 60, EsikBot: 85, Politika: pol})
	if k.Sinif != "bot" || k.Sayilir || !k.Engelle {
		t.Fatalf("otomasyon UA bot olmalı: %+v", k)
	}

	// 5) Hosting ASN (45) → şüpheli; politika hosting engeli AÇIK olduğu
	//    için engellenir. Puan eşiği yetmese de politika bağlayıcı.
	b := asnBilgi{ASN: 24940, Ad: "HETZNER-AS - Hetzner Online GmbH, DE", Tur: "hosting", Ag: "5.9.0.0/16"}
	k = guvenlikHesapla(guvenlikGirdi{Anahtar: "a", ASN: &b, EsikYuksek: 60, EsikBot: 85, Politika: pol})
	if k.Sinif != "supheli" || !k.Engelle || k.Sayilir {
		t.Fatalf("hosting politikası uygulanmadı: %+v", k)
	}

	// 6) Hosting engeli KAPALI → engellenmez, yine sayılmaz.
	pol2 := pol
	pol2.HostingEngel = false
	k = guvenlikHesapla(guvenlikGirdi{Anahtar: "a", ASN: &b, EsikYuksek: 60, EsikBot: 85, Politika: pol2})
	if k.Engelle || k.Sayilir {
		t.Fatalf("hosting engeli kapalıyken engellenmemeli: %+v", k)
	}

	// 7) VPN ASN (60) → yüksek; varsayılan politikada ENGEL YOK (okuyucu
	//    kaybetmemek için), ama sayılmaz.
	v := asnBilgi{ASN: 206092, Ad: "SECFIREWALLAS - F.N.S. HOLDINGS LIMITED, CY", Tur: "vpn"}
	k = guvenlikHesapla(guvenlikGirdi{Anahtar: "a", ASN: &v, EsikYuksek: 60, EsikBot: 85, Politika: pol})
	if k.Sinif != "yuksek" || k.Engelle || k.Sayilir {
		t.Fatalf("VPN varsayılanı: yüksek + sayılmaz, engellenmez: %+v", k)
	}
	pol3 := pol
	pol3.VPNEngel = true
	k = guvenlikHesapla(guvenlikGirdi{Anahtar: "a", ASN: &v, EsikYuksek: 60, EsikBot: 85, Politika: pol3})
	if !k.Engelle {
		t.Fatalf("VPN engeli açıkken engellenmeli: %+v", k)
	}
}

func TestAramaBotuSinifi(t *testing.T) {
	pol := guvenlikPolitikasi{Aktif: true, BotEngel: true, HostingEngel: true}
	// Doğrulanmış arama motoru: ENGEL YOK, SAYIM YOK.
	b := asnBilgi{ASN: 15169, Ad: "GOOGLE - Google LLC, US", Tur: "hosting"}
	k := guvenlikHesapla(guvenlikGirdi{Anahtar: "a", ASN: &b, AramaMotoru: true,
		EsikYuksek: 60, EsikBot: 85, Politika: pol})
	if k.Sinif != "arama" || k.Sayilir || k.Engelle {
		t.Fatalf("arama botu engellenmemeli, sayılmamalı: %+v", k)
	}
	// Sahte crawler: iddia var, doğrulanamadı → bot (85) + UA yok.
	k = guvenlikHesapla(guvenlikGirdi{Anahtar: "a", SahteCrawler: true,
		EsikYuksek: 60, EsikBot: 85, Politika: pol})
	if k.Sinif != "bot" || !k.Engelle {
		t.Fatalf("sahte crawler bot olmalı: %+v", k)
	}
}

func TestAramaMotoruUA(t *testing.T) {
	ornekler := map[string]bool{
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)": true,
		"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)":  true,
		"Mozilla/5.0 (compatible; YandexBot/3.0)":                                  true,
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120":                     false,
		"": false,
	}
	for ua, bekle := range ornekler {
		if got, _ := aramaMotoruMu(ua); got != bekle {
			t.Errorf("aramaMotoruMu(%q) = %v", ua, got)
		}
	}
}

func TestSayimKapisi(t *testing.T) {
	// Karar yoksa (güvenlik kapalı) eski davranış: sayılır.
	r := httptest.NewRequest("GET", "/haber/x", nil)
	if !sayilirMi(r) || !reklamGosterilir(r) {
		t.Fatal("karar yokken sayılmalı/reklam gösterilmeli")
	}

	// Bot kararı bağlama konunca sayım ve reklam kapanır.
	karar := &guvenlikKarari{Sinif: "bot", Sayilir: false, Reklam: false}
	r = r.WithContext(contextWithKarar(r, karar))
	if sayilirMi(r) {
		t.Fatal("bot kararında sayılmamalı")
	}
	if reklamGosterilir(r) {
		t.Fatal("bot kararında reklam gizlenmeli")
	}
	if okunmaSayilirCtx(r.Context()) {
		t.Fatal("okunmaSayilirCtx bot kararında false dönmeli")
	}
}

func TestKitaUyusmazlik(t *testing.T) {
	// Türkiye IP + Türkiye saati → uyuşuyor.
	if kitaUyusmazlik("TR", "Europe/Istanbul") {
		t.Fatal("TR + Europe/Istanbul çelişmemeli")
	}
	// Almanya IP + İstanbul saati → İKİSİ DE AVRUPA: çelişki sayılmaz.
	// (Aynı kıta içi fark olağan; yakalanması istenmez.)
	if kitaUyusmazlik("DE", "Europe/Istanbul") {
		t.Fatal("DE + Europe/Istanbul çelişmemeli (aynı kıta)")
	}
	// Almanya IP + New York saati → çelişki.
	if !kitaUyusmazlik("DE", "America/New_York") {
		t.Fatal("DE + America/New_York çelişmeli")
	}
	// Bilinmeyen ülke / belirsiz bölge → fail-open.
	if kitaUyusmazlik("ZZ", "America/New_York") {
		t.Fatal("bilinmeyen ülke çelişki üretmemeli")
	}
	if kitaUyusmazlik("TR", "UTC") {
		t.Fatal("UTC belirsiz sayılmalı")
	}
}

func TestDilUyusmazlik(t *testing.T) {
	if dilUyusmazlik("TR", "tr-TR") {
		t.Fatal("TR + tr çelişmemeli")
	}
	if dilUyusmazlik("TR", "en-US") {
		t.Fatal("en dünya dili: çelişki sayılmamalı")
	}
	if !dilUyusmazlik("TR", "ja-JP") {
		t.Fatal("TR + ja çelişmeli")
	}
	if dilUyusmazlik("TR", "") {
		t.Fatal("boş dil çelişki üretmemeli")
	}
}

func TestYazilimGPU(t *testing.T) {
	for _, s := range []string{
		"Google SwiftShader", "llvmpipe (LLVM 15.0.6, 256 bits)",
		"ANGLE (Google, Vulkan 1.3.0 (SwiftShader Device (Subzero)))",
		"Microsoft Basic Render Driver",
	} {
		if !yazilimGPU(s) {
			t.Errorf("yazılım GPU tanınmadı: %q", s)
		}
	}
	for _, s := range []string{
		"ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 Direct3D11)",
		"Apple M2", "", "Intel(R) UHD Graphics 620",
	} {
		if yazilimGPU(s) {
			t.Errorf("gerçek GPU yanlış işaretlendi: %q", s)
		}
	}
}

func TestXFFCeliski(t *testing.T) {
	// Cloudflare normali: XFF ilk atlama == CF-Connecting-IP.
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.9")
	if xffCeliski(r, "203.0.113.9") {
		t.Fatal("aynı IP çelişki sayılmamalı")
	}
	r.Header.Set("X-Forwarded-For", "198.51.100.7, 203.0.113.9")
	if !xffCeliski(r, "203.0.113.9") {
		t.Fatal("farklı ilk atlama çelişki sayılmalı")
	}
	// Başlık yoksa çelişki yok.
	r2 := httptest.NewRequest("GET", "/", nil)
	if xffCeliski(r2, "203.0.113.9") {
		t.Fatal("XFF yokken çelişki üretilmemeli")
	}
}

func TestSinyalJSONVeEtiket(t *testing.T) {
	pol := guvenlikPolitikasi{Aktif: true}
	k := guvenlikHesapla(guvenlikGirdi{Anahtar: "a", ProxyBaslik: true, ProxyNeden: "proxy başlığı: Via",
		EsikYuksek: 60, EsikBot: 85, Politika: pol})
	j := k.sinyallerJSON()
	if !strings.Contains(j, `"baslik_proxy"`) {
		t.Fatalf("sinyal JSON'unda kod yok: %s", j)
	}
	if !k.sinifBayragi("baslik_proxy") {
		t.Fatal("sinifBayragi proxy sinyalini bulamadı")
	}
	if k.sinifBayragi("asn_hosting") {
		t.Fatal("olmayan sinyal true döndü")
	}
	// Ayrıştırma: panel listesi kısa adlara çevirir.
	adlar := sinyalListesi(j)
	if len(adlar) != 1 || adlar[0] != "Proxy başlığı" {
		t.Fatalf("sinyalListesi = %v", adlar)
	}
	if sinyalListesi("bozuk") != nil {
		t.Fatal("bozuk JSON sessizce boş liste dönmeli")
	}
}

func TestSaglayiciEtiket(t *testing.T) {
	ornekler := map[string]string{
		"HETZNER-AS - Hetzner Online GmbH, DE":     "HETZNER-AS",
		"DIGITALOCEAN-ASN - DigitalOcean, LLC, US": "DIGITALOCEAN-ASN",
		"": "",
	}
	for ham, bekle := range ornekler {
		if got := asnSaglayiciEtiket(ham); got != bekle {
			t.Errorf("asnSaglayiciEtiket(%q) = %q, beklenen %q", ham, got, bekle)
		}
	}
}

// contextWithKarar testte kararı istek bağlamına koyar.
func contextWithKarar(r *http.Request, k *guvenlikKarari) context.Context {
	return context.WithValue(r.Context(), guvenlikKarariAnahtari{}, k)
}
