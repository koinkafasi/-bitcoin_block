package httpapi

import (
	"context"
	"encoding/json"
	"fmt"

	"theblock/site/internal/repo"
)

// Hazır gazete şablonları — specs/13-gazete-sayfa-duzenleyici.md Faz 3
//
// `gazete_templates` boştu: "var olan plana göre düzenleme" denen şeyin planı
// yoktu. Burada peyzajlar tanımlanır; galeri bunları gösterir, sayı
// oluşturulurken biri seçilir.
//
// `A4 — Klasik Manşet`, kullanıcının referans tasarımının (Newspaper #6003.svg)
// anatomisini izler: künye şeridi → lacivert başlık bandı → kategori şeridi →
// 2 kolon kahraman alan → 3 kolon orta şerit → 2 kolon alt blok → alt bilgi.

// gzBlok şablon tanımlarını okunur kılan kısa kurucu.
func gzBlok(id, tip string, x, y, w, h float64) gazeteBlock {
	return gazeteBlock{ID: id, Tip: tip, X: x, Y: y, W: w, H: h}
}

func (b gazeteBlock) ile(baslik string) gazeteBlock { b.Baslik = baslik; return b }

func (b gazeteBlock) kaynak(tip string, limit int) gazeteBlock {
	b.Kaynak = &gazeteKaynak{Tip: tip, Limit: limit}
	return b
}

func (b gazeteBlock) stil(ciftler ...any) gazeteBlock {
	m := map[string]any{}
	for i := 0; i+1 < len(ciftler); i += 2 {
		k, _ := ciftler[i].(string)
		m[k] = ciftler[i+1]
	}
	b.Stil = m
	return b
}

// gazeteBolumSayfasi tek bir bölüm (kategori) sayfasının standart düzeni:
// başlık bandı + manşet + iki liste. Çoğu iç sayfa bu kalıbı izler.
func gazeteBolumSayfasi(no int, ad, baslik string) gazetePageDoc {
	p := fmt.Sprintf("p%d", no)
	return gazetePageDoc{No: no, Ad: ad, Blocks: []gazeteBlock{
		gzBlok(p+"b", "baslik", 2, 2, 96, 6).ile(baslik),
		gzBlok(p+"m", "manset", 2, 10, 96, 40).kaynak("son", 1).
			stil("bindirme", "alt", "metinRenk", "acik"),
		gzBlok(p+"l1", "liste", 2, 52, 47, 44).ile("ÖNE ÇIKANLAR").kaynak("son", 6),
		gzBlok(p+"l2", "liste", 51, 52, 47, 44).ile("KISA KISA").kaynak("son", 6),
		gzBlok(p+"k", "kunye", 2, 97, 96, 2),
	}}
}

// gazeteKlasikMansetKapak referans peyzajının kapak sayfası (§3).
func gazeteKlasikMansetKapak() gazetePageDoc {
	return gazetePageDoc{No: 1, Ad: "Kapak", Blocks: []gazeteBlock{
		// 1) Sayı no / tarih şeridi
		gzBlok("k-ust", "kunye", 2, 1, 96, 3).stil("hiza", "orta"),
		// 2) Lacivert künye bandı — gazetenin adı
		gzBlok("k-bant", "baslik", 2, 5, 96, 8).ile("THEBLOCK").
			stil("zemin", "lacivert", "metinRenk", "acik", "hiza", "orta"),
		// 3) Kategori şeridi
		gzBlok("k-kat", "baslik", 2, 14, 96, 4).
			ile("EKONOMİ · DÜNYA · GEZİ · KÜLTÜR · TEKNOLOJİ · SPOR").
			stil("hiza", "orta", "kenarlik", "ince"),
		// 4) İki kolon kahraman alan: görsel + bindirmeli metin / renkli kutu
		gzBlok("k-hero", "manset", 2, 20, 60, 32).kaynak("manset", 1).
			stil("bindirme", "alt", "metinRenk", "acik"),
		gzBlok("k-yan", "kart", 64, 20, 34, 32).ile("GÜNÜN ÖZETİ").kaynak("son", 2).
			stil("zemin", "turuncu", "metinRenk", "acik"),
		// 5) Üç kolon orta şerit: koyu kutu · görsel · açık kutu
		gzBlok("k-o1", "kart", 2, 54, 31, 16).ile("GÜNDEM").kaynak("son", 1).
			stil("zemin", "koyu", "metinRenk", "acik"),
		gzBlok("k-o2", "gorsel", 35, 54, 28, 16),
		gzBlok("k-o3", "kart", 65, 54, 33, 16).ile("EKONOMİ").kaynak("son", 1).
			stil("zemin", "acik"),
		// 6) İki kolon alt blok: büyük manşet / bindirmeli görsel
		gzBlok("k-a1", "manset", 2, 72, 56, 22).kaynak("son", 1).stil("olcek", 1.15),
		gzBlok("k-a2", "manset", 60, 72, 38, 22).kaynak("son", 1).
			stil("bindirme", "alt", "metinRenk", "acik", "zemin", "koyu"),
		// 7) Alt bilgi
		gzBlok("k-alt", "kunye", 2, 96, 96, 3).stil("hiza", "orta"),
	}}
}

// gazeteTohumSablonlari projeyle gelen hazır şablonlar.
func gazeteTohumSablonlari() []struct {
	Ad       string
	Aciklama string
	Doc      gazeteSablonDoc
} {
	a4 := func(tipOlcek float64, sayfalar []gazetePageDoc) gazeteSablonDoc {
		s := gazeteSablonVarsayilanSayfa()
		s.TipOlcek = tipOlcek
		return gazeteSablonDoc{
			Surum: 1, Sayfa: s,
			Izgara:   gazeteSablonVarsayilanIzgara(),
			Sayfalar: sayfalar,
		}
	}

	// --- 1) Klasik Manşet: referans peyzajı + bölüm sayfaları ---
	klasik := []gazetePageDoc{gazeteKlasikMansetKapak()}
	for i, ad := range []string{"Gündem", "Ekonomi", "Siyaset", "Dünya",
		"Spor", "Kültür-Sanat", "Teknoloji", "Yaşam"} {
		klasik = append(klasik, gazeteBolumSayfasi(i+2, ad, bolumBasligi(ad)))
	}
	klasik = append(klasik, gazeteSonSayfa(10))

	// --- 2) İki Kolon Ağırlıklı: metin yoğun, az görsel ---
	ikiKolon := []gazetePageDoc{{No: 1, Ad: "Kapak", Blocks: []gazeteBlock{
		gzBlok("i-bant", "baslik", 2, 2, 96, 7).ile("THEBLOCK").
			stil("zemin", "lacivert", "metinRenk", "acik", "hiza", "orta"),
		gzBlok("i-m", "manset", 2, 11, 96, 30).kaynak("manset", 1),
		gzBlok("i-l1", "liste", 2, 43, 47, 36).ile("GÜNDEM").kaynak("son", 7),
		gzBlok("i-l2", "liste", 51, 43, 47, 36).ile("EKONOMİ").kaynak("son", 7),
		gzBlok("i-m2", "metin", 2, 81, 96, 15).ile("BAŞYAZI").stil("sutun", 3),
		gzBlok("i-k", "kunye", 2, 97, 96, 2),
	}}}
	for i, ad := range []string{"Gündem", "Ekonomi", "Siyaset", "Dünya", "Spor"} {
		ikiKolon = append(ikiKolon, gazeteIkiKolonSayfasi(i+2, ad, bolumBasligi(ad)))
	}

	// --- 3) Görsel Ağırlıklı: büyük fotoğraflar ---
	gorselAgir := []gazetePageDoc{{No: 1, Ad: "Kapak", Blocks: []gazeteBlock{
		gzBlok("g-m", "manset", 2, 2, 96, 56).kaynak("manset", 1).
			stil("bindirme", "alt", "metinRenk", "acik", "olcek", 1.3),
		gzBlok("g-k1", "kart", 2, 60, 31, 24).kaynak("son", 1).stil("bindirme", "alt", "metinRenk", "acik"),
		gzBlok("g-k2", "kart", 35, 60, 31, 24).kaynak("son", 1).stil("bindirme", "alt", "metinRenk", "acik"),
		gzBlok("g-k3", "kart", 68, 60, 30, 24).kaynak("son", 1).stil("bindirme", "alt", "metinRenk", "acik"),
		gzBlok("g-l", "liste", 2, 86, 96, 10).ile("KISA KISA").kaynak("son", 4),
		gzBlok("g-k", "kunye", 2, 97, 96, 2),
	}}}
	for i, ad := range []string{"Foto Galeri", "Dünya", "Yaşam"} {
		gorselAgir = append(gorselAgir, gazeteGorselSayfasi(i+2, ad, bolumBasligi(ad)))
	}

	// --- 4) Kategori Sayfası: tek bölümlük kısa ek ---
	kategori := []gazetePageDoc{
		gazeteBolumSayfasi(1, "Bölüm", "BÖLÜM"),
		gazeteBolumSayfasi(2, "Devam", "DEVAMI"),
	}

	// --- 5) A3 Mevcut Düzen: bugünkü çıktının birebir karşılığı ---
	a3Sayfa := gazeteSablonVarsayilanSayfa()
	a3Sayfa.Boyut = "A3"
	a3Sayfa.TipOlcek = 1

	return []struct {
		Ad       string
		Aciklama string
		Doc      gazeteSablonDoc
	}{
		{"A4 — Klasik Manşet",
			"Referans peyzaj: lacivert künye bandı, kategori şeridi, bindirmeli kahraman görsel, üç kolon orta şerit. 10 sayfa.",
			a4(1, klasik)},
		{"A4 — İki Kolon Ağırlıklı",
			"Metin yoğun klasik gazete düzeni; az görsel, çok liste ve başyazı. 6 sayfa.",
			a4(1, ikiKolon)},
		{"A4 — Görsel Ağırlıklı",
			"Büyük fotoğraf ve bindirmeli başlıklar; magazin hissi. 4 sayfa.",
			a4(1, gorselAgir)},
		{"A4 — Kategori Eki",
			"Tek bölümlük kısa ek; kategori bazlı yayınlar için. 2 sayfa.",
			a4(1, kategori)},
		{"A3 — Mevcut Düzen",
			"Bugüne kadar üretilen A3 çıktısının birebir karşılığı; eski görünümü korumak için.",
			gazeteSablonDoc{Surum: 1, Sayfa: a3Sayfa,
				Izgara: gazeteSablonVarsayilanIzgara(), Sayfalar: defaultGazetePages()}},
	}
}

func bolumBasligi(ad string) string {
	switch ad {
	case "Kültür-Sanat":
		return "KÜLTÜR & SANAT"
	case "Foto Galeri":
		return "FOTO GALERİ"
	}
	return map[string]string{
		"Gündem": "GÜNDEM", "Ekonomi": "EKONOMİ", "Siyaset": "SİYASET",
		"Dünya": "DÜNYA", "Spor": "SPOR", "Teknoloji": "TEKNOLOJİ",
		"Yaşam": "YAŞAM",
	}[ad]
}

func gazeteIkiKolonSayfasi(no int, ad, baslik string) gazetePageDoc {
	p := fmt.Sprintf("ik%d", no)
	return gazetePageDoc{No: no, Ad: ad, Blocks: []gazeteBlock{
		gzBlok(p+"b", "baslik", 2, 2, 96, 6).ile(baslik),
		gzBlok(p+"l1", "liste", 2, 10, 47, 60).ile("").kaynak("son", 9),
		gzBlok(p+"l2", "liste", 51, 10, 47, 60).ile("").kaynak("son", 9),
		gzBlok(p+"m", "metin", 2, 72, 96, 24).ile("DEĞERLENDİRME").stil("sutun", 3),
		gzBlok(p+"k", "kunye", 2, 97, 96, 2),
	}}
}

func gazeteGorselSayfasi(no int, ad, baslik string) gazetePageDoc {
	p := fmt.Sprintf("gs%d", no)
	return gazetePageDoc{No: no, Ad: ad, Blocks: []gazeteBlock{
		gzBlok(p+"b", "baslik", 2, 2, 96, 6).ile(baslik),
		gzBlok(p+"g1", "manset", 2, 10, 96, 42).kaynak("son", 1).
			stil("bindirme", "alt", "metinRenk", "acik"),
		gzBlok(p+"g2", "kart", 2, 54, 47, 42).kaynak("son", 1).
			stil("bindirme", "alt", "metinRenk", "acik"),
		gzBlok(p+"g3", "kart", 51, 54, 47, 42).kaynak("son", 1).
			stil("bindirme", "alt", "metinRenk", "acik"),
		gzBlok(p+"k", "kunye", 2, 97, 96, 2),
	}}
}

func gazeteSonSayfa(no int) gazetePageDoc {
	return gazetePageDoc{No: no, Ad: "Son Sayfa", Blocks: []gazeteBlock{
		gzBlok("s-b", "baslik", 2, 2, 96, 6).ile("SON SAYFA"),
		gzBlok("s-m", "manset", 2, 10, 60, 44).kaynak("son", 1),
		gzBlok("s-l", "liste", 64, 10, 34, 44).ile("GÜNÜN NOTLARI").kaynak("son", 6),
		gzBlok("s-r", "reklam", 2, 57, 96, 24),
		gzBlok("s-m2", "metin", 2, 83, 96, 13).ile("YARIN").stil("sutun", 2),
		gzBlok("s-k", "kunye", 2, 97, 96, 2),
	}}
}

// gazeteSablonTohumla eksik hazır şablonları siteye yükler.
// İsme göre eşler; var olanı değiştirmez, yani tekrar çalıştırmak güvenlidir.
func (s *Server) gazeteSablonTohumla(ctx context.Context, siteID int64) (int, error) {
	mevcut, err := repo.ListGazeteTemplates(ctx, s.pool, siteID)
	if err != nil {
		return 0, err
	}
	var kayitli int
	for _, t := range gazeteTohumSablonlari() {
		if gazeteSablonVar(mevcut, t.Ad) {
			continue
		}
		if sorun := gazeteSablonDogrula(t.Doc); len(sorun) > 0 {
			// Bozuk şablonu yüklemek, bozuk gazete üretmek demek.
			return kayitli, fmt.Errorf("şablon %q geçersiz: %v", t.Ad, sorun)
		}
		raw, err := json.Marshal(t.Doc)
		if err != nil {
			return kayitli, err
		}
		tpl := &repo.GazeteTemplate{
			SiteID:      siteID,
			Name:        t.Ad,
			Description: t.Aciklama,
			Pages:       raw,
			IsDefault:   len(mevcut) == 0 && kayitli == 0,
		}
		if _, err := repo.CreateGazeteTemplate(ctx, s.pool, tpl); err != nil {
			return kayitli, err
		}
		kayitli++
	}
	return kayitli, nil
}

func gazeteSablonVar(liste []repo.GazeteTemplate, ad string) bool {
	for _, t := range liste {
		if t.Name == ad {
			return true
		}
	}
	return false
}

// gazeteVarsayilanSablonID sitenin varsayilan sablonunu dondurur.
// Galeri bossa once hazir peyzajlari yukler: sayi her zaman bir plana
// bagli olsun istiyoruz (SPEC Faz 3), "sablonsuz sayi" durumu kalmasin.
func (s *Server) gazeteVarsayilanSablonID(ctx context.Context, siteID int64) *int64 {
	sec := func(liste []repo.GazeteTemplate) *int64 {
		for _, t := range liste {
			if t.IsDefault {
				id := t.ID
				return &id
			}
		}
		if len(liste) > 0 {
			id := liste[0].ID
			return &id
		}
		return nil
	}
	liste, err := repo.ListGazeteTemplates(ctx, s.pool, siteID)
	if err != nil {
		return nil
	}
	if id := sec(liste); id != nil {
		return id
	}
	if _, err := s.gazeteSablonTohumla(ctx, siteID); err != nil {
		s.logger.Error("gazete sablon tohumlama", "error", err)
		return nil
	}
	liste, err = repo.ListGazeteTemplates(ctx, s.pool, siteID)
	if err != nil {
		return nil
	}
	return sec(liste)
}

// gazeteSablonAdaGore ada gore sablon kimligi arar.
func (s *Server) gazeteSablonAdaGore(ctx context.Context, siteID int64, ad string) *int64 {
	liste, err := repo.ListGazeteTemplates(ctx, s.pool, siteID)
	if err != nil {
		return nil
	}
	for _, t := range liste {
		if t.Name == ad {
			id := t.ID
			return &id
		}
	}
	return nil
}

// gazeteEskiSayilariBagla sablonu olmayan sayilari verilen sablona baglar.
//
// Mevcut sayilar sablon kavrami yokken, eski A3 varsayilan duzeniyle
// uretilmisti; bu yuzden onlari "A3 - Mevcut Duzen" sablonuna baglamak
// gorunumlerini korur. Yalniz NULL olanlara dokunur.
func (s *Server) gazeteEskiSayilariBagla(ctx context.Context, siteID, sablonID int64) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE gazete_issues SET template_id=$1, updated_at=now()
		   WHERE site_id=$2 AND template_id IS NULL`, sablonID, siteID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
