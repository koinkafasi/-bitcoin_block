package httpapi

import (
	"context"
	"encoding/json"
	"time"
)

// İçeriklerdeki DIŞ görselleri medya kütüphanesine çeken arka plan işi.
//
// NEDEN: gövdedeki görsel başkasının sunucusunda durdukça o sunucu
// kapandığında yazı sakatlanır. 13 Kasım 2026 ölçümü: Ghost'tan
// aktarılan 50 yazının 47 gövde görseli ghost.bitcosar.com'a, 50 kapak
// görseli cdn.coinhit.net'e bakıyordu — ve ghost.bitcosar.com
// kapatılıyor. Aynı sorun RSS'ten gelen görseller için de geçerli.
//
// NEDEN İŞÇİ, KOMUT DEĞİL: medya dizini kümedeki kalıcı diskte (PVC) ve
// oraya yalnız site süreci yazabiliyor. Dışarıdan çalışan bir CLI
// dosyayı kaydedemezdi.
//
// DAVRANIŞ: her turda küçük bir parti, yavaş ve geri alınabilir.
// Orijinal adres `image_url` alanında KALIYOR — medya kaydı bir şekilde
// kaybolursa görsel yine görünür ve kaynağı izlenebilir kalır.

const (
	disGorselAraligi      = 3 * time.Minute
	disGorselTurLimiti    = 8
	disGorselBaslangicGec = 2 * time.Minute
)

// DisGorselAktariciBaslat arka plan döngüsünü başlatır.
func (s *Server) DisGorselAktariciBaslat(ctx context.Context) {
	go func() {
		// Açılışta beklemek: göç ve ısınma bitsin, ilk istekler
		// indirme trafiğiyle yarışmasın.
		select {
		case <-ctx.Done():
			return
		case <-time.After(disGorselBaslangicGec):
		}
		t := time.NewTicker(disGorselAraligi)
		defer t.Stop()
		for {
			if n := s.disGorselTuru(ctx); n > 0 {
				s.logger.Info("dış görsel: medyaya alındı", "adet", n)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

// disGorselAdayi indirilecek tek görsel.
type disGorselAdayi struct {
	ContentID int64
	SiteID    int64
	Baslik    string
	Adres     string
	// Kapak true ise contents.cover_media_id doldurulur; değilse
	// gövdedeki image düğümünün src'si mediaId ile değiştirilir.
	Kapak bool
}

// disGorselTuru tek turda en çok disGorselTurLimiti görsel indirir.
func (s *Server) disGorselTuru(ctx context.Context) int {
	// ÖNCE GÖVDE: gövde görselleri ghost.bitcosar.com gibi KAPANACAK
	// adreslerde duruyor; kapaklar ayrı bir CDN'de ve daha dayanıklı.
	liste, err := s.disGorselGovdeAdaylari(ctx, disGorselTurLimiti)
	if err != nil {
		s.logger.Warn("dış görsel: gövde adayları okunamadı", "error", err)
	}
	if kalan := disGorselTurLimiti - len(liste); kalan > 0 {
		kapaklar, err := s.disGorselKapakAdaylari(ctx, kalan)
		if err != nil {
			s.logger.Warn("dış görsel: kapak adayları okunamadı", "error", err)
		}
		liste = append(liste, kapaklar...)
	}

	n := 0
	for _, a := range liste {
		if ctx.Err() != nil {
			return n
		}
		id, err := s.disGorselIndir(ctx, a.Adres, a.Baslik, a.Baslik, a.SiteID, 0)
		if err != nil {
			s.disGorselBasarisiz(ctx, a)
			s.logger.Info("dış görsel: indirilemedi", "içerik", a.ContentID,
				"adres", kisaAdres(a.Adres), "error", err)
			continue
		}
		if err := s.disGorselBagla(ctx, a, id); err != nil {
			s.logger.Warn("dış görsel: içeriğe bağlanamadı",
				"içerik", a.ContentID, "medya", id, "error", err)
			continue
		}
		n++
	}
	return n
}

// disGorselGovdeAdaylari gövdesinde dış adresli görsel olan içerikler.
//
// `$.**` ile iç içe düğümler de taranıyor: görsel alıntı veya liste
// içinde de olabilir, yalnız üst düzeye bakmak onları kaçırırdı.
func (s *Server) disGorselGovdeAdaylari(ctx context.Context, limit int) ([]disGorselAdayi, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (c.id, n->'attrs'->>'src')
		       c.id, c.site_id, c.title, n->'attrs'->>'src'
		  FROM contents c,
		       LATERAL jsonb_path_query(c.detail, 'strict $.**') n
		 WHERE n->>'type' = 'image'
		   AND n->'attrs'->>'src' LIKE 'http%'
		   -- DENENİP OLMAYANI ATLA. Bu şart olmadan kuyruk tıkanıyor:
		   -- sıralama id DESC olduğu için en yeni içerik hep başa
		   -- geliyor ve ölü adresler her turda yeniden denenip tur
		   -- limitini doldurarak geri kalan binlerce görsele SIRA
		   -- BIRAKMIYOR. 13 Kasım 2026'da üretimde görüldü: 49 ölü
		   -- Ghost adresi, 2012 görsellik kuyruğu tümüyle durdurdu.
		   AND NOT COALESCE((n->'attrs'->>'indirilemedi')::boolean, false)
		 ORDER BY c.id DESC, n->'attrs'->>'src'
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []disGorselAdayi
	for rows.Next() {
		var a disGorselAdayi
		if err := rows.Scan(&a.ContentID, &a.SiteID, &a.Baslik, &a.Adres); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// disGorselKapakAdaylari kapak görseli dışarıda olan içerikler.
func (s *Server) disGorselKapakAdaylari(ctx context.Context, limit int) ([]disGorselAdayi, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, site_id, title, image_url
		  FROM contents
		 WHERE cover_media_id IS NULL
		   AND COALESCE(image_url,'') LIKE 'http%'
		   AND COALESCE(image_url_yedek,'') = ''
		 ORDER BY updated_at DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []disGorselAdayi
	for rows.Next() {
		var a disGorselAdayi
		if err := rows.Scan(&a.ContentID, &a.SiteID, &a.Baslik, &a.Adres); err != nil {
			return nil, err
		}
		a.Kapak = true
		out = append(out, a)
	}
	return out, rows.Err()
}

// disGorselBagla indirilen medyayı içeriğe bağlar.
func (s *Server) disGorselBagla(ctx context.Context, a disGorselAdayi, mediaID int64) error {
	if a.Kapak {
		_, err := s.pool.Exec(ctx, `
			UPDATE contents SET cover_media_id = $2, updated_at = now()
			 WHERE id = $1 AND cover_media_id IS NULL`, a.ContentID, mediaID)
		return err
	}
	return s.disGorselGovdeBagla(ctx, a, mediaID)
}

// disGorselBasarisiz ölü adresi işaretler ki bir daha denenmesin.
//
// KUYRUĞU TIKAMAMAK İÇİN ŞART. İlk sürümde yalnız kapaklar
// işaretleniyordu ve gövde için "tekrar deneme zararsız" diye
// düşünmüştüm. Değilmiş: sıralama id DESC olduğu için en yeni içerik
// hep başa geliyor, ölü adresler her turda tur limitini dolduruyor ve
// geri kalan görsellere SIRA GELMİYOR. 13 Kasım 2026'da üretimde
// görüldü — 49 ölü Ghost adresi 2012 görsellik kuyruğu durdurdu.
//
// Kapakta işaret `image_url_yedek` alanı, gövdede düğüm özniteliği
// (`indirilemedi: true`). İkisi de aday sorgusunda süzülüyor.
func (s *Server) disGorselBasarisiz(ctx context.Context, a disGorselAdayi) {
	if a.Kapak {
		if _, err := s.pool.Exec(ctx, `
			UPDATE contents SET image_url_yedek = $2
			 WHERE id = $1 AND COALESCE(image_url_yedek,'') = ''`,
			a.ContentID, a.Adres); err != nil {
			s.logger.Warn("dış görsel: başarısız işaretlenemedi", "içerik", a.ContentID, "error", err)
		}
		return
	}

	var ham []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT detail FROM contents WHERE id=$1`, a.ContentID).Scan(&ham); err != nil {
		return
	}
	yeni, degisti := tipTapGorselIsaretle(ham, a.Adres)
	if !degisti {
		return
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE contents SET detail=$2::jsonb WHERE id=$1`, a.ContentID, yeni); err != nil {
		s.logger.Warn("dış görsel: gövde işareti yazılamadı", "içerik", a.ContentID, "error", err)
	}
}

// tipTapGorselIsaretle eşleşen görsel düğümüne "indirilemedi" bayrağı
// koyar.
//
// `updated_at` BİLEREK DOKUNULMUYOR: bu bir içerik değişikliği değil,
// işçinin kendi defteri. Güncelleme zamanını kaydırmak içeriği
// listelerde öne taşır ve "bu yazı güncellendi" yanılgısı üretir.
func tipTapGorselIsaretle(ham []byte, adres string) ([]byte, bool) {
	var belge any
	if json.Unmarshal(ham, &belge) != nil {
		return ham, false
	}
	degisti := false
	var gez func(n any)
	gez = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			if v["type"] == "image" {
				if attrs, ok := v["attrs"].(map[string]any); ok {
					if s, _ := attrs["src"].(string); s == adres {
						attrs["indirilemedi"] = true
						degisti = true
					}
				}
			}
			for _, alt := range v {
				gez(alt)
			}
		case []any:
			for _, alt := range v {
				gez(alt)
			}
		}
	}
	gez(belge)
	if !degisti {
		return ham, false
	}
	out, err := json.Marshal(belge)
	if err != nil {
		return ham, false
	}
	return out, true
}

// disGorselGovdeBagla gövdedeki image düğümünün src'sini mediaId ile
// değiştirir.
//
// Düğüm `src` yerine `mediaId` taşıdığında çizici görseli KENDİ
// sunucumuzdan basıyor (bkz. renderTipTapNode, "image" dalı).
func (s *Server) disGorselGovdeBagla(ctx context.Context, a disGorselAdayi, mediaID int64) error {
	var ham []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT detail FROM contents WHERE id=$1`, a.ContentID).Scan(&ham); err != nil {
		return err
	}
	yeni, degisti := tipTapGorselDegistir(ham, a.Adres, mediaID)
	if !degisti {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE contents SET detail=$2::jsonb, updated_at=now() WHERE id=$1`,
		a.ContentID, yeni)
	return err
}

// tipTapGorselDegistir belgedeki eşleşen src'yi mediaId ile değiştirir.
//
// AYNI ADRES BİRDEN ÇOK KEZ geçebilir (aynı grafik iki bölümde);
// hepsi değiştiriliyor, yoksa kalanlar ölü adrese bakmaya devam eder.
func tipTapGorselDegistir(ham []byte, adres string, mediaID int64) ([]byte, bool) {
	var belge any
	if json.Unmarshal(ham, &belge) != nil {
		return ham, false
	}
	degisti := false
	var gez func(n any)
	gez = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			if v["type"] == "image" {
				if attrs, ok := v["attrs"].(map[string]any); ok {
					if s, _ := attrs["src"].(string); s == adres {
						delete(attrs, "src")
						attrs["mediaId"] = mediaID
						degisti = true
					}
				}
			}
			for _, alt := range v {
				gez(alt)
			}
		case []any:
			for _, alt := range v {
				gez(alt)
			}
		}
	}
	gez(belge)
	if !degisti {
		return ham, false
	}
	out, err := json.Marshal(belge)
	if err != nil {
		return ham, false
	}
	return out, true
}

// kisaAdres günlük için adresi kısaltır.
func kisaAdres(u string) string {
	if len(u) <= 70 {
		return u
	}
	return u[:67] + "..."
}
