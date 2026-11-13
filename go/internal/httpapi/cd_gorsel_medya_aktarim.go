package httpapi

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Kaynak görsellerini medya kütüphanesine aktarma ----
//
// NEDEN (ölçüm, 13 Kasım 2026): `source_images` 712 indirilmiş görsel
// tutuyor (152 MB, `{depo}/kaynak/` ağacında) ama `media_files` yalnız 39
// kayıt: brand 30, upload 9. Görseller bizim diskimizde ama kütüphanemizde
// değil; alt metni yazılamıyor, panelden düzenlenemiyor ve AI görsel
// üretici geldiğinde "bunları yeniden üret" denemiyor.
//
// Bu işçi her görseli kütüphaneye alır ve `source='internet'` işaretler.
// İşaret şemada zaten tanımlı (media_files source CHECK kısıtı), yani yeni
// bir alan gerekmedi. Görsel üretici gelince `DisKaynakliMedyalar` kuyruğu
// hazır bekliyor.
//
// DOSYA KOPYALANIR, TAŞINMAZ: `/kg/<hash>` yolu yayındaki içeriklerde
// referanslı. Taşımak onları anında kırardı. Kopya sonrası içeriğin
// `cover_media_id`'si bağlanır; `/kg/` ağacı ayrı bir temizlik işine kalır.

const (
	// gorselAktarimAraligi iki tur arası bekleme.
	//
	// 5 dakika: aktarım disk okuma + yazma demek, AI çağrısı yok — yani
	// kotadan bağımsız ve ucuz. Ama diski gereksiz meşgul etmemek için
	// ön indiriciden (3 dk) daha seyrek.
	gorselAktarimAraligi = 5 * time.Minute

	// gorselAktarimTurLimiti tek turda aktarılacak görsel sayısı.
	//
	// 10: ortalama görsel ~215 KB (152 MB / 712), yani tur başına ~2 MB
	// okuma+yazma. 712 görsel ~6 saatte kütüphaneye girer.
	gorselAktarimTurLimiti = 10

	// gorselAktarimBaslangicGecikmesi süreç açılışında beklenen süre.
	//
	// Açılışta göç, şablon derleme ve ön indirici zaten disk ve CPU
	// kullanıyor; aktarım onların arasına girmesin.
	gorselAktarimBaslangicGecikmesi = 90 * time.Second
)

// GorselMedyaAktariciBaslat aktarım işçisini başlatır.
func (s *Server) GorselMedyaAktariciBaslat(ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(gorselAktarimBaslangicGecikmesi):
		}

		if ozet, err := repo.GorselAktarimDurumu(ctx, s.pool); err == nil {
			s.logger.Info("görsel medya aktarıcısı başladı",
				"toplam", ozet.Toplam, "aktarilan", ozet.Aktarilan,
				"bekleyen", ozet.Bekleyen, "her", gorselAktarimAraligi.String())
		}

		ticker := time.NewTicker(gorselAktarimAraligi)
		defer ticker.Stop()
		for {
			s.gorselAktarimTuru(ctx)
			select {
			case <-ctx.Done():
				s.logger.Info("görsel medya aktarıcısı durdu")
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *Server) gorselAktarimTuru(ctx context.Context) {
	liste, err := repo.AktarilacakGorseller(ctx, s.pool, gorselAktarimTurLimiti)
	if err != nil {
		s.logger.Warn("görsel aktarım: kuyruk okunamadı", "error", err)
		return
	}
	if len(liste) == 0 {
		return
	}

	aktarilan := 0
	for _, g := range liste {
		if ctx.Err() != nil {
			return
		}
		if s.birGorselAktar(ctx, g) {
			aktarilan++
		}
	}
	if aktarilan > 0 {
		s.logger.Info("görsel aktarım turu", "aktarilan", aktarilan, "kuyruk", len(liste))
	}
}

// birGorselAktar tek görseli kütüphaneye alır; başarılıysa true döner.
func (s *Server) birGorselAktar(ctx context.Context, g repo.AktarilacakGorsel) bool {
	b, algilananTip, ok := kaynakGorselOku(g.Hash, "")
	if !ok || len(b) == 0 {
		// Kayıt indirilmiş diyor ama dosya yok: tekrar indirilsin diye
		// bayt sayacını sıfırlamak yerine kaydı atlıyoruz — ön indirici
		// zaten bytes=0 olanları alıyor, burada veri yazmak iki işçiyi
		// çakıştırırdı.
		s.logger.Warn("görsel aktarım: dosya diskte yok", "hash", g.Hash)
		return false
	}

	mimeTipi := strings.TrimSpace(g.ContentType)
	if mimeTipi == "" {
		mimeTipi = algilananTip
	}
	if mimeTipi == "" {
		mimeTipi = http.DetectContentType(b)
	}

	ad := gorselDosyaAdi(g.URL, g.Hash, mimeTipi)
	yol, err := platform.SaveMedia(b, ad)
	if err != nil {
		s.logger.Warn("görsel aktarım: medyaya yazılamadı", "hash", g.Hash, "error", err)
		return false
	}

	// Alt metin İÇERİKTEN gelir, kaynak URL'sinden değil.
	//
	// SEO gerekçesi: aynı görsel onlarca sitede olabilir, bizim
	// benzersizliğimiz alt metinden gelir. Kaynak dosya adı
	// ("shutterstock_12345.jpg") hiçbir arama sorgusuna karşılık gelmez.
	alt, baslik := s.gorselAltMetni(ctx, g.Hash)

	m := &repo.Media{
		SiteID:    1,
		Kind:      "image",
		Filename:  ad,
		Path:      yol,
		Mime:      mimeTipi,
		SizeBytes: &g.Bytes,
		Source:    "internet", // şemada tanımlı işaret; AI üretimi 'ai' olacak
		Category:  "haber",
		Title:     baslik,
		// Dış kaynaktan gelen görsel editör kütüphanesinde öne çıkmasın:
		// yıldızlı olanlar panelde ilk sırada listeleniyor.
		IsStarred:  false,
		Visibility: "public",
	}
	if alt != "" {
		m.Alt = &alt
	}

	mediaID, err := repo.CreateMedia(ctx, s.pool, m)
	if err != nil {
		s.logger.Warn("görsel aktarım: medya kaydı açılamadı", "hash", g.Hash, "error", err)
		return false
	}
	if err := repo.GorselMedyayaBagla(ctx, s.pool, g.Hash, mediaID); err != nil {
		s.logger.Warn("görsel aktarım: bağlantı yazılamadı", "hash", g.Hash, "error", err)
		// Medya kaydı açıldı ama bağlantı yazılamadı: bir sonraki tur
		// aynı görseli tekrar aktarırdı. Kaydı silmek yerine uyarı
		// bırakıyoruz — çift kayıt, kayıp kayıttan iyidir.
		return false
	}

	// Görseli kapak olarak kullanan içerikleri kütüphane kaydına bağla.
	if ids, err := repo.GorselKullananIcerikler(ctx, s.pool, g.Hash); err == nil {
		for _, cid := range ids {
			if err := repo.IcerikKapakMedyasiAta(ctx, s.pool, cid, mediaID); err != nil {
				s.logger.Warn("görsel aktarım: kapak bağlanamadı",
					"icerik", cid, "medya", mediaID, "error", err)
			}
		}
	}
	return true
}

// gorselAltMetni görseli kullanan ilk içerikten alt metin ve başlık üretir.
//
// Alt metin görsel aramanın tek metin girdisi; boş bırakmak hem erişim
// hem SEO kaybı. İçerik başlığı, görselin ne gösterdiğine dair elimizdeki
// en iyi bilgi.
func (s *Server) gorselAltMetni(ctx context.Context, hash string) (alt, baslik string) {
	ids, err := repo.GorselKullananIcerikler(ctx, s.pool, hash)
	if err != nil || len(ids) == 0 {
		return "", ""
	}
	c, err := repo.GetContent(ctx, s.pool, ids[0])
	if err != nil {
		return "", ""
	}
	t := strings.TrimSpace(c.Title)
	if t == "" {
		return "", ""
	}
	// Alt metin 125 karakterde kesilir: ekran okuyucular daha uzununu
	// okumuyor, arama motorları da fazlasını değerlendirmiyor.
	return kirpAltMetin(t, 125), t
}

func kirpAltMetin(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	// Kelime ortasından kesmemek için son boşluğa kadar geri gel.
	kirpik := string(r[:max])
	if i := strings.LastIndex(kirpik, " "); i > max/2 {
		kirpik = kirpik[:i]
	}
	return strings.TrimRight(kirpik, " ,;:-") + "…"
}

// gorselDosyaAdi kaynak URL'sinden okunabilir bir dosya adı üretir.
//
// Kaynak adı kullanılamazsa hash'e düşer. Uzantı MIME'den belirlenir:
// kaynak URL'sinde uzantı olmayabiliyor (CDN'ler sorgu parametresiyle
// servis ediyor) ve SaveMedia uzantıyı dosya adından okuyor.
func gorselDosyaAdi(kaynakURL, hash, mimeTipi string) string {
	ext := gorselUzanti(mimeTipi)

	taban := ""
	if i := strings.IndexByte(kaynakURL, '?'); i >= 0 {
		kaynakURL = kaynakURL[:i]
	}
	if p := path.Base(kaynakURL); p != "." && p != "/" {
		taban = strings.TrimSuffix(p, path.Ext(p))
	}
	taban = temizDosyaAdi(taban)
	if taban == "" {
		taban = hash
	}
	if len(taban) > 60 {
		taban = taban[:60]
	}
	return taban + ext
}

func gorselUzanti(mimeTipi string) string {
	if i := strings.IndexByte(mimeTipi, ';'); i >= 0 {
		mimeTipi = strings.TrimSpace(mimeTipi[:i])
	}
	switch mimeTipi {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/avif":
		return ".avif"
	}
	if exts, err := mime.ExtensionsByType(mimeTipi); err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ".jpg"
}

// temizDosyaAdi yalnız güvenli karakterleri bırakır.
//
// Kaynak dosya adları yol ayracı, boşluk ve Unicode taşıyabiliyor; bunlar
// hem depolama hem URL tarafında sorun çıkarır.
func temizDosyaAdi(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-_")
}

// gorselAktarimRaporu tanı için tek satır (günlük/panel).
func gorselAktarimRaporu(o repo.GorselAktarimOzeti) string {
	if o.Toplam == 0 {
		return "kaynak görsel yok"
	}
	yuzde := float64(o.Aktarilan) / float64(o.Toplam) * 100
	return fmt.Sprintf("%d/%d görsel kütüphanede (%.1f%%), %d bekliyor",
		o.Aktarilan, o.Toplam, yuzde, o.Bekleyen)
}

// gorselAktarimRaporu panel ucu eklenene kadar yalnız testlerde çağrılıyor.
