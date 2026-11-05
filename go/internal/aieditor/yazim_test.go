package aieditor

import (
	"os"
	"strings"
	"testing"
)

// Faz 194 — otomatik AI ile yazım.
//
// ÖLÇÜM (üretim): bot günde ~3.600 taslak ekliyor ve bunlar RSS özeti
// olduğu gibi yazılıyor (yayındaki 982/986 içerik TEK paragraf). AI ile
// yazım yolu vardı ama elle tetikleniyordu (`AI Merkezi → küme → taslak`).

func TestYazimVarsayilanKapali(t *testing.T) {
	// Bu hat AI maliyeti doğurur ve metni doğrudan taslağa yazar. Varsayılan
	// AÇIK olursa kurulum yapan herkes farkında olmadan ücret öder.
	//
	// DİKKAT: yalnız `YazimAyarlar{}` sıfır değerini denetlemek yetmez —
	// varsayılan `YazimAyarlariOku` içinde kurulur. İlk sürümde bu boşluk
	// vardı ve mutasyon testi kaçırdı (varsayılan true yapılınca test yine
	// geçiyordu). Kaynağı da doğruluyoruz: DB gerekmeden sözleşmeyi korur.
	src := okuKaynak(t, "yazim_worker.go")
	if !strings.Contains(src, "a := YazimAyarlar{Aktif: false, Dakika: YazimVarsayilanDakika}") {
		t.Error("varsayılan KAPALI değil: ayar yazılmamışken hat kendiliğinden çalışır")
	}

	a := YazimAyarlar{}
	if a.Aktif {
		t.Error("sıfır değer açık olmamalı")
	}
	// sinirYazimDakika: 0 ve negatif varsayılana döner.
	if got := sinirYazimDakika(0); got != YazimVarsayilanDakika {
		t.Errorf("0 → %d, beklenen %d", got, YazimVarsayilanDakika)
	}
	if got := sinirYazimDakika(-5); got != YazimVarsayilanDakika {
		t.Errorf("negatif → %d", got)
	}
	if got := sinirYazimDakika(999); got != YazimMaksDakika {
		t.Errorf("üst sınır uygulanmadı: %d", got)
	}
	if got := sinirYazimDakika(7); got != 7 {
		t.Errorf("geçerli değer değişti: %d", got)
	}
}

// TestYazimSonucuEsleme: AI Result → içeriğe yazılacak alanlar.
func TestYazimSonucuEsleme(t *testing.T) {
	res := Result{
		Action: "news",
		Title:  "Yedek başlık",
		Fields: map[string]string{
			"title":            "AI başlığı",
			"spot":             "Kısa spot metni.",
			"body":             "Birinci paragraf.\nİkinci paragraf.\n\nÜçüncü paragraf.",
			"seo_title":        "SEO başlığı burada",
			"meta_description": "Meta açıklama burada.",
		},
	}
	y, ok := yazimSonucu(res)
	if !ok {
		t.Fatal("sonuç eşlenemedi")
	}
	if y.Title != "AI başlığı" {
		t.Errorf("başlık: %q", y.Title)
	}
	if len(y.Paragraf) != 3 {
		t.Errorf("paragraf sayısı %d, beklenen 3: %v", len(y.Paragraf), y.Paragraf)
	}
	if y.SeoTitle == "" || y.SeoDesc == "" {
		t.Errorf("SEO alanları taşınmadı: %+v", y)
	}
}

func TestYazimSonucuSectionsYedegi(t *testing.T) {
	// Alanlar boşsa Sections'tan okunur (eski işlerin sonucu böyle).
	res := Result{
		Action:   "news",
		Title:    "Başlık",
		Sections: []Section{{Title: "Gövde", Body: "Paragraf bir.\nParagraf iki."}},
	}
	y, ok := yazimSonucu(res)
	if !ok {
		t.Fatal("Sections yedeği çalışmadı")
	}
	if len(y.Paragraf) != 2 {
		t.Errorf("paragraf %d", len(y.Paragraf))
	}
}

func TestYazimSonucuBosGovdeReddedilir(t *testing.T) {
	// Boş gövde yazmak içeriği bozardı; sonuç uygulanmamalı.
	res := Result{Action: "news", Title: "Başlık", Fields: map[string]string{"title": "B"}}
	if _, ok := yazimSonucu(res); ok {
		t.Error("gövdesiz sonuç kabul edildi")
	}
	res2 := Result{Action: "news", Fields: map[string]string{"body": "gövde"}}
	if _, ok := yazimSonucu(res2); ok {
		t.Error("başlıksız sonuç kabul edildi")
	}
}

func TestYazimSonucuParagraflariTekler(t *testing.T) {
	res := Result{Action: "news", Fields: map[string]string{
		"title": "B",
		"body":  "  Bir.  \n\n\n  İki.  \n   \n",
	}}
	y, ok := yazimSonucu(res)
	if !ok {
		t.Fatal("eşlenemedi")
	}
	if len(y.Paragraf) != 2 {
		t.Fatalf("boş paragraflar atılmadı: %v", y.Paragraf)
	}
	if strings.TrimSpace(y.Paragraf[0]) != "Bir." {
		t.Errorf("kırpılmadı: %q", y.Paragraf[0])
	}
}

// TestYazimKuyruguKaynakGerektirir: seçim sorgusu kaynaklı olmayı şart koşar.
// Elle yazılan taslaklara AI çağrısı harcanmamalı.
func TestYazimKuyruguKaynakGerektirir(t *testing.T) {
	src := okuKaynak(t, "../repo/ai_yazim.go")
	if !strings.Contains(src, "c.source_url IS NOT NULL") {
		t.Error("kaynak koşulu yok: elle yazılan taslaklar da işlenirdi")
	}
	if !strings.Contains(src, "j.status = 'completed'") {
		t.Error("tamamlanmış iş denetimi yok: aynı içerik sonsuz yeniden yazılır")
	}
	if !strings.Contains(src, "j.status IN ('queued','running')") {
		t.Error("kuyrukta/çalışan iş denetimi yok: çift iş yazılır")
	}
	if !strings.Contains(src, "LIMIT $4") {
		t.Error("tur sınırı yok")
	}
}

// TestYazimSurumCakismasiEzmez: editör arada kaydettiyse AI sonucu yazılmaz.
func TestYazimSurumCakismasiEzmez(t *testing.T) {
	src := okuKaynak(t, "../repo/ai_yazim.go")
	if !strings.Contains(src, "($7 = 0 OR version = $7)") {
		t.Error("sürüm denetimi yok: AI, editörün düzeltmesini ezer")
	}
	if !strings.Contains(src, "ErrConflict") {
		t.Error("çakışma hatası dönmüyor")
	}
	// Yalnız dar alan kümesi güncellenir: UPDATE ... SET ile WHERE arası.
	// (Sorgunun WHERE bölümünde `c.status` geçmesi normal — orası seçim.)
	i := strings.Index(src, "UPDATE contents SET")
	if i < 0 {
		t.Fatal("güncelleme sorgusu bulunamadı")
	}
	son := strings.Index(src[i:], "WHERE id = $1")
	if son < 0 {
		t.Fatal("güncelleme WHERE'i bulunamadı")
	}
	setBlogu := src[i : i+son]
	for _, yasak := range []string{"category_id", "cover_media_id", "manset_media_id", "status", "locale", "source_url"} {
		if strings.Contains(setBlogu, yasak) {
			t.Errorf("kapsam dışı alan güncelleniyor: %q", yasak)
		}
	}
	// Yazılması GEREKEN alanlar.
	for _, gerekli := range []string{"title =", "spot =", "detail =", "seo_title =", "seo_description ="} {
		if !strings.Contains(setBlogu, gerekli) {
			t.Errorf("beklenen alan güncellenmiyor: %q", gerekli)
		}
	}
}

func TestYazimUygulamaIsaretlenir(t *testing.T) {
	src := okuKaynak(t, "../repo/ai_yazim.go")
	if !strings.Contains(src, "applied_at = now()") {
		t.Error("applied_at işaretlenmiyor: aynı sonuç her turda yeniden yazılır")
	}
}

func TestYazimHiziKota(t *testing.T) {
	src := okuKaynak(t, "yazim_worker.go")
	if !strings.Contains(src, "repo.YazimSayaciSonDakika") {
		t.Error("dakikalık hız kotası uygulanmıyor")
	}
	if !strings.Contains(src, "if !ayar.Aktif {") {
		t.Error("kapalıyken erken çıkış yok: AI çağrısı yapılırdı")
	}
	// Kuyruğa yazma AI ÇAĞIRMAZ; iş mevcut işçiye bırakılır.
	if strings.Contains(src, "ProcessJob(") {
		t.Error("yazım turu AI çağrısını kendisi yapıyor (eşzamanlılık kaybolur)")
	}
}

func TestYazimMigration(t *testing.T) {
	up := okuKaynak(t, "../../migrations/0103_ai_yazim.up.sql")
	if !strings.Contains(up, "ADD COLUMN IF NOT EXISTS applied_at") {
		t.Error("applied_at kolonu yok")
	}
	down := okuKaynak(t, "../../migrations/0103_ai_yazim.down.sql")
	if strings.Contains(down, "DELETE FROM") {
		t.Error("geri alma veri siliyor")
	}
}

// okuKaynak test için kaynak dosya okur (yol paket dizinine göre).
func okuKaynak(t *testing.T, yol string) string {
	t.Helper()
	b, err := os.ReadFile(yol)
	if err != nil {
		t.Fatalf("kaynak okunamadı %s: %v", yol, err)
	}
	return string(b)
}
