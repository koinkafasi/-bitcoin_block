package httpapi

import (
	"testing"
	"time"

	"theblock/site/internal/repo"
)

// cdKapakGorselTestSunucu mediaPath'i taklit eden sunucu üretir.
//
// NEDEN: gerçek çözüm DB'ye gidiyor (repo.GetMedia) ve birim testte havuz
// yok. Öncelik KURALINI sınamak için mediaPath'i enjekte ediyoruz.
func cdKapakGorselTestSunucu(harita map[int64]string) *Server {
	s := &Server{}
	s.mediaMu.Lock()
	s.mediaCache = map[int64]string{}
	for k, v := range harita {
		s.mediaCache[k] = v
	}
	s.mediaMu.Unlock()
	return s
}

// TestCdKapakGorselEditorKazanir — KAPAK ÖNCELİĞİ sözleşmesi.
//
// ÖLÇÜLEN HATA (kullanıcı bildirimi, 10 Kasım 2026): "yazıya medya
// yöneticisi ile resim ekledim ... yeni tasarımda sitede eski resim imaj
// yolu gösteriyor". Önce `image_url` (kaynak CDN) okunuyordu; editörün
// yüklediği `cover_media_id` yok sayılıyordu. Aynı hata liste kartlarını
// (cdStoryCevir) da vuruyordu.
func TestCdKapakGorselEditorKazanir(t *testing.T) {
	kapak := int64(7)
	s := cdKapakGorselTestSunucu(map[int64]string{
		7: "media/202611/kapak.webp",
	})

	c := repo.Content{
		CoverMediaID: &kapak,
		ImageURL:     "https://cdn.example.com/eski-gorsel.jpg",
	}
	got := s.cdKapakGorsel(c)
	if got != "/media/202611/kapak.webp" {
		t.Fatalf("kapak kullanılmadı: %q (image_url kazandı mı?)", got)
	}

	// Kapak YOKSA kaynak görseline düşer (eski davranış korunur).
	c2 := repo.Content{ImageURL: "https://cdn.example.com/kaynak.jpg"}
	if got := s.cdKapakGorsel(c2); got != c2.ImageURL {
		t.Errorf("kapaksız içerikte kaynak görsel beklenirdi, gelen: %q", got)
	}

	// İkisi de yoksa boş — uydurma görsel basılmaz.
	if got := s.cdKapakGorsel(repo.Content{}); got != "" {
		t.Errorf("görselsiz içerikte boş beklenirdi, gelen: %q", got)
	}
}

// TestCdKapakGorselSilinmisMedya medya kaydı silinmişse kaynak görsele
// düşmeli; "/media/none.svg" basılmamalı.
func TestCdKapakGorselSilinmisMedya(t *testing.T) {
	eksik := int64(999)
	// Önbellekte BOŞ yol: medya kaydı silinmiş/yolu boş demek. Aynı etkiyi
	// DB'ye gitmeden üretir (mediaPath "" döner → mediaURL "/media/none.svg").
	s := cdKapakGorselTestSunucu(map[int64]string{999: ""})
	c := repo.Content{CoverMediaID: &eksik, ImageURL: "https://cdn.example.com/kaynak.jpg"}

	got := s.cdKapakGorsel(c)
	if got == "/media/none.svg" {
		t.Errorf("silinmiş medya için none.svg basıldı")
	}
	if got != c.ImageURL {
		t.Errorf("kaynak görsele düşmedi: %q", got)
	}
}

// TestCdStoryCevirKapakKullanir liste kartlarının da kapak önceliğini
// kullandığını kilitler (makale dışındaki tüm yüzeyler).
func TestCdStoryCevirKapakKullanir(t *testing.T) {
	kapak := int64(7)
	s := cdKapakGorselTestSunucu(map[int64]string{7: "media/202611/kapak.webp"})

	st := s.cdStoryCevir(repo.Content{
		Title:        "Başlık",
		CoverMediaID: &kapak,
		ImageURL:     "https://cdn.example.com/eski.jpg",
	}, time.Now())

	if st.Image != "/media/202611/kapak.webp" {
		t.Errorf("kart eski görseli kullanıyor: %q", st.Image)
	}
}
