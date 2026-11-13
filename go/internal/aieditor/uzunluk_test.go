package aieditor

import (
	"strings"
	"testing"
)

// Kategori tür anahtarını EZMELİ. Yazım hattı her taslağı
// type_key="haber" olarak açıyor (ölçüm: son 30 günün 325 AI içeriğinin
// 325'i "haber"); bir taslak "Kripto 101"e düşmüşse o bir rehberdir ve
// bunu yalnız kategori söyler.
func TestUzunlukHedefiKategoriTuruEzer(t *testing.T) {
	testler := []struct {
		ad          string
		typeKey     string
		kategori    string
		istekMin    int
		istekAd     string
		altBaslikMi bool
	}{
		{"haber/politika → haber bandı", "haber", "policy", 450, "haber", false},
		{"haber/learn → REHBER bandı", "haber", "learn", 1500, "rehber", true},
		{"haber/kriptopedi → ansiklopedi", "haber", "kriptopedi", 1200, "ansiklopedi maddesi", true},
		{"haber/analiz → analiz", "haber", "analiz", 800, "analiz", true},
		{"haber/opinion → görüş", "haber", "opinion", 600, "görüş yazısı", false},
		{"analiz/bilinmeyen → tür bandı", "analiz", "madencilik", 800, "analiz", true},
		{"arastirma/yok → tür bandı", "arastirma", "", 1200, "araştırma", true},
		{"basin-bulteni/press-release", "basin-bulteni", "press-release", 350, "basın bülteni", false},
		{"ikisi de bilinmiyor → haber", "", "", 450, "haber", false},
		{"büyük harf ve boşluk", " HABER ", " LEARN ", 1500, "rehber", true},
	}
	for _, tt := range testler {
		t.Run(tt.ad, func(t *testing.T) {
			h := UzunlukHedefiIcin(tt.typeKey, tt.kategori)
			if h.MinKelime != tt.istekMin {
				t.Errorf("MinKelime = %d, beklenen %d", h.MinKelime, tt.istekMin)
			}
			if h.Ad != tt.istekAd {
				t.Errorf("Ad = %q, beklenen %q", h.Ad, tt.istekAd)
			}
			if h.AltBaslikGerekli() != tt.altBaslikMi {
				t.Errorf("AltBaslikGerekli() = %v, beklenen %v", h.AltBaslikGerekli(), tt.altBaslikMi)
			}
		})
	}
}

// Her bandın kendi içinde tutarlı olması gerekir: min < max, paragraf
// bandı kelime bandıyla uyumlu. Elle yazılmış tabloda kolay kaçan hata.
func TestUzunlukBantlariTutarli(t *testing.T) {
	tablolar := map[string]map[string]UzunlukHedefi{
		"tür":      uzunlukBantlari,
		"kategori": kategoriBantlari,
	}
	for tabloAd, tablo := range tablolar {
		for anahtar, h := range tablo {
			if h.Ad == "" {
				t.Errorf("%s/%s: Ad boş", tabloAd, anahtar)
			}
			if h.MinKelime <= 0 || h.MinKelime >= h.MaxKelime {
				t.Errorf("%s/%s: kelime bandı bozuk (%d-%d)", tabloAd, anahtar, h.MinKelime, h.MaxKelime)
			}
			if h.MinParagraf <= 0 || h.MinParagraf >= h.MaxParagraf {
				t.Errorf("%s/%s: paragraf bandı bozuk (%d-%d)", tabloAd, anahtar, h.MinParagraf, h.MaxParagraf)
			}
			// Paragraf bandı kelime bandıyla UYUMLU olmalı, yoksa modele
			// çelişkili iki hedef verilir. Prompt 2-4 cümle istiyor;
			// Türkçe'de bu ~35-70 kelime. Köşe durumlarının bile
			// 20-120 aralığında kalması gerekiyor.
			if kpp := h.MinKelime / h.MaxParagraf; kpp < 20 {
				t.Errorf("%s/%s: paragraf başına %d kelime — paragraflar duyuruya döner",
					tabloAd, anahtar, kpp)
			}
			if kpp := h.MaxKelime / h.MinParagraf; kpp > 120 {
				t.Errorf("%s/%s: paragraf başına %d kelime — metin duvarı", tabloAd, anahtar, kpp)
			}
		}
	}
}

// 13 Kasım 2026 ölçümündeki gerçek durum: hat 156 kelime üretiyordu ve
// eski eşik (700 karakter ≈ 100 kelime) bunu "yeterli" buluyordu, yani
// hiç tetiklenmiyordu.
func TestUzunlukUyarisiOlculenKisaMetniYakalar(t *testing.T) {
	govde := strings.TrimSpace(strings.Repeat("kelime ", 156))
	hedef := UzunlukHedefiIcin("haber", "policy")

	u := uzunlukUyarilari(govde, hedef, false)
	if len(u) == 0 {
		t.Fatal("156 kelimelik haber uyarı üretmedi")
	}
	if !strings.Contains(u[0], "156 kelime") {
		t.Errorf("uyarı ölçülen sayıyı vermiyor: %q", u[0])
	}
	if !strings.Contains(u[0], "450-700") {
		t.Errorf("uyarı hedefi vermiyor: %q", u[0])
	}
}

// Hedef bir bant, çit değil: sınıra yakın metin için uyarı gürültüdür.
func TestUzunlukUyarisiSapmaPayi(t *testing.T) {
	hedef := UzunlukHedefiIcin("haber", "")
	// 450 * 0.85 = 382; 400 kelime payın içinde.
	if u := uzunlukUyarilari(strings.Repeat("x ", 400), hedef, false); len(u) != 0 {
		t.Errorf("payın içindeki metin uyarı üretti: %v", u)
	}
	// 700 * 1.15 = 805; 750 kelime payın içinde.
	if u := uzunlukUyarilari(strings.Repeat("x ", 750), hedef, false); len(u) != 0 {
		t.Errorf("üst payın içindeki metin uyarı üretti: %v", u)
	}
	// 1200 kelime haberde dolgu şüphesi.
	u := uzunlukUyarilari(strings.Repeat("x ", 1200), hedef, false)
	if len(u) == 0 || !strings.Contains(u[0], "Dolgu") {
		t.Errorf("aşırı uzun haber için dolgu uyarısı yok: %v", u)
	}
}

// Alt başlık uyarısı YANLIŞ SIRAYLA verilmemeli: hedefi tutturamamış
// 200 kelimelik metne "alt başlık ekle" demek anlamsız.
func TestUzunlukAltBaslikUyarisiSirasi(t *testing.T) {
	rehber := UzunlukHedefiIcin("haber", "learn")

	kisa := uzunlukUyarilari(strings.Repeat("x ", 200), rehber, false)
	for _, s := range kisa {
		if strings.Contains(s, "alt başlık yok") {
			t.Errorf("kısa metne alt başlık uyarısı verildi: %q", s)
		}
	}

	uzun := uzunlukUyarilari(strings.Repeat("x ", 1600), rehber, false)
	var bulundu bool
	for _, s := range uzun {
		if strings.Contains(s, "alt başlık yok") {
			bulundu = true
		}
	}
	if !bulundu {
		t.Errorf("1600 kelimelik rehberde alt başlık uyarısı yok: %v", uzun)
	}

	// Alt başlık varsa uyarı olmamalı.
	for _, s := range uzunlukUyarilari(strings.Repeat("x ", 1600), rehber, true) {
		if strings.Contains(s, "alt başlık yok") {
			t.Errorf("alt başlık varken uyarı verildi: %q", s)
		}
	}
}

// Haber bandında alt başlık HİÇ istenmemeli, metin ne kadar uzasa da:
// 800 kelimeyi aşan haber zaten "dolgu" uyarısı alıyor.
func TestUzunlukHaberdeAltBaslikIstenmez(t *testing.T) {
	haber := UzunlukHedefiIcin("haber", "markets")
	if haber.AltBaslikGerekli() {
		t.Fatal("haber bandı alt başlık istiyor")
	}
	for _, s := range uzunlukUyarilari(strings.Repeat("x ", 900), haber, false) {
		if strings.Contains(s, "alt başlık") {
			t.Errorf("haberde alt başlık uyarısı: %q", s)
		}
	}
}

func TestUzunlukBosGovdeSessiz(t *testing.T) {
	if u := uzunlukUyarilari("   ", UzunlukHedefiIcin("haber", ""), false); u != nil {
		t.Errorf("boş gövde uyarı üretti: %v", u)
	}
}

// Prompt satırı buyurgan ve ölçülebilir olmalı: ölçümde model bağlam
// JSON'una gömülü sayıyı yok sayıp alt sınıra yapışıyordu.
func TestUzunlukPromptSatiri(t *testing.T) {
	p := UzunlukHedefiIcin("haber", "learn").PromptSatiri()
	for _, s := range []string{"HEDEF UZUNLUK", "rehber", "1500-2500 KELİME", "ALT BAŞLIK zorunlu", "DOLGU"} {
		if !strings.Contains(p, s) {
			t.Errorf("prompt satırında %q yok:\n%s", s, p)
		}
	}
	// Haberde alt başlık cümlesi GEÇMEMELİ.
	if strings.Contains(UzunlukHedefiIcin("haber", "policy").PromptSatiri(), "ALT BAŞLIK zorunlu") {
		t.Error("haber promptunda alt başlık zorunluluğu var")
	}
}

// Hedef yalnız gövde yazan eylemlere basılmalı; başlık önerisi veya SEO
// alanı dolduran eyleme uzunluk hedefi vermek prompt'u şişirirdi.
func TestUzunlukHedefiUygulananEylemler(t *testing.T) {
	for _, id := range []string{"news", "research_news", "rewrite_angle", "audio_draft"} {
		if !uzunlukHedefiUygulanir(id) {
			t.Errorf("%q gövde yazıyor ama hedef uygulanmıyor", id)
		}
	}
	for _, id := range []string{"headlines", "seo", "seo_fill", "summary", "social", "verify", "questions", "background", "intro"} {
		if uzunlukHedefiUygulanir(id) {
			t.Errorf("%q gövde yazmıyor ama hedef uygulanıyor", id)
		}
	}
}

// Hedef istemde GERÇEKTEN görünmeli ve EDİTÖR TALİMATINDAN ÖNCE
// gelmeli: editör "daha kısa yaz" derse onun sözü geçerli olsun.
func TestUserPromptUzunlukHedefiniIcerir(t *testing.T) {
	act, ok := ActionByID("news")
	if !ok {
		t.Fatal("news eylemi yok")
	}
	hedef := UzunlukHedefiIcin("haber", "learn")
	ctx := articleContext{Title: "T", Uzunluk: &hedef}

	p := userPrompt(act, ctx, "seçim", "gövde", "Daha kısa yaz", "", nil)

	iHedef := strings.Index(p, "HEDEF UZUNLUK")
	if iHedef < 0 {
		t.Fatalf("istemde hedef bloğu yok:\n%s", p)
	}
	iTalimat := strings.Index(p, "EDİTÖR TALİMATI")
	if iTalimat < 0 {
		t.Fatal("istemde editör talimatı yok")
	}
	if iHedef > iTalimat {
		t.Error("hedef bloğu editör talimatından SONRA basılmış")
	}
	if !strings.Contains(p, "1500-2500 KELİME") {
		t.Error("rehber bandı isteme girmemiş")
	}
}

// Hedefi olmayan bağlam istemi bozmamalı (editörün elle çağırdığı
// eylemlerde Uzunluk nil kalabilir).
func TestUserPromptHedefsizCalisir(t *testing.T) {
	act, _ := ActionByID("news")
	p := userPrompt(act, articleContext{Title: "T"}, "seçim", "gövde", "", "", nil)
	if strings.Contains(p, "HEDEF UZUNLUK") {
		t.Error("hedefsiz bağlamda hedef bloğu basıldı")
	}
	if !strings.HasSuffix(strings.TrimSpace(p), "Yalnızca geçerli JSON döndür.") {
		t.Error("istem beklenen biçimde bitmiyor")
	}
}

// Gövde yazmayan eyleme hedef basılmamalı.
func TestUserPromptBaslikEylemineHedefBasilmaz(t *testing.T) {
	act, ok := ActionByID("headlines")
	if !ok {
		t.Fatal("headlines eylemi yok")
	}
	hedef := UzunlukHedefiIcin("haber", "")
	p := userPrompt(act, articleContext{Title: "T", Uzunluk: &hedef}, "s", "g", "", "", nil)
	if strings.Contains(p, "HEDEF UZUNLUK") {
		t.Error("başlık eylemine uzunluk hedefi basıldı")
	}
}

func TestAltBaslikVar(t *testing.T) {
	bos := &Result{Sections: []Section{{Title: "Gövde", Body: "metin"}}}
	if altBaslikVar(bos) {
		t.Error("alt başlık yokken true döndü")
	}
	bosBaslik := &Result{Sections: []Section{{Title: "Alt başlıklar", Body: "  "}}}
	if altBaslikVar(bosBaslik) {
		t.Error("boş alt başlık bölümü true döndü")
	}
	dolu := &Result{Sections: []Section{{Title: "Alt başlıklar", Body: "- Birinci\n- İkinci"}}}
	if !altBaslikVar(dolu) {
		t.Error("alt başlık varken false döndü")
	}
}

func TestKelimeSay(t *testing.T) {
	testler := []struct {
		metin string
		istek int
	}{
		{"", 0},
		{"   ", 0},
		{"tek", 1},
		{"iki kelime", 2},
		{"  araya   fazla   boşluk  ", 3},
		{"satır\nsonu\tsekme", 3},
	}
	for _, tt := range testler {
		if got := KelimeSay(tt.metin); got != tt.istek {
			t.Errorf("KelimeSay(%q) = %d, beklenen %d", tt.metin, got, tt.istek)
		}
	}
}
