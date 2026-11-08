package youtube

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// Ayar panelden yönetilen YouTube entegrasyon ayarları.
type Ayar struct {
	KanalID     string
	Handle      string
	Dakika      int
	Aktif       bool
	VideoGoster bool
	ShortGoster bool
	CanliGoster bool
	// CanliVideoID doluysa otomatik canlı algılama yerine BU video canlı
	// sayılır (yayın başladığında algılama gecikirse panelden elle işaret).
	CanliVideoID string
	SonCekme     string
	SonHata      string
}

// VarsayilanAyar ayar satırları yoksa kullanılan değerler.
func VarsayilanAyar() Ayar {
	return Ayar{
		KanalID: "UCPtM806ijpKatyIHH5w4kmg", Handle: "bitcosar",
		Dakika: 15, Aktif: true,
		VideoGoster: true, ShortGoster: true, CanliGoster: true,
	}
}

// AyarlariOku settings/youtube bölümünü okur; eksik alanlar varsayılanda kalır.
//
// TEK SORGU: alanlar tek tek okunuyordu ve settings tablosuna anahtar başına
// bir gidiş-dönüş yapılıyordu (10 sorgu). Ana sayfada üç ayrı video bloğu
// ayarları okuduğu için istek başına 30 sorgu ediyordu — ölçüm (8 Kasım 2026,
// log_statement=all, tek ana sayfa isteği): 54 sorgunun 30'u buydu.
// Bölümün tamamı tek sorguda alınıp bellekte eşlenir.
func AyarlariOku(ctx context.Context, pool repo.DB) Ayar {
	a := VarsayilanAyar()
	tum, err := repo.GetSettingsText(ctx, pool, "youtube")
	if err != nil {
		return a
	}
	al := func(k string) string {
		return strings.TrimSpace(tum[k])
	}
	if v := al("kanal_id"); v != "" {
		a.KanalID = v
	}
	if v := al("kanal_handle"); v != "" {
		a.Handle = strings.TrimPrefix(v, "@")
	}
	if v := al("dakika"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 5 && n <= 1440 {
			a.Dakika = n
		}
	}
	if v := al("aktif"); v != "" {
		a.Aktif = v == "true"
	}
	if v := al("video_goster"); v != "" {
		a.VideoGoster = v == "true"
	}
	if v := al("short_goster"); v != "" {
		a.ShortGoster = v == "true"
	}
	if v := al("canli_goster"); v != "" {
		a.CanliGoster = v == "true"
	}
	a.CanliVideoID = VideoIDCikar(al("canli_video_id"))
	a.SonCekme = al("son_cekme")
	a.SonHata = al("son_hata")
	return a
}

// Kaydet ayarları yazar (panel formu).
func (a Ayar) Kaydet(ctx context.Context, pool repo.DB) error {
	kaydet := func(k string, v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return repo.SetSetting(ctx, pool, "youtube", k, b)
	}
	if err := kaydet("kanal_id", a.KanalID); err != nil {
		return err
	}
	if err := kaydet("kanal_handle", a.Handle); err != nil {
		return err
	}
	if err := kaydet("dakika", a.Dakika); err != nil {
		return err
	}
	if err := kaydet("aktif", a.Aktif); err != nil {
		return err
	}
	if err := kaydet("video_goster", a.VideoGoster); err != nil {
		return err
	}
	if err := kaydet("short_goster", a.ShortGoster); err != nil {
		return err
	}
	if err := kaydet("canli_goster", a.CanliGoster); err != nil {
		return err
	}
	return kaydet("canli_video_id", a.CanliVideoID)
}

// VideoIDCikar panelden girilen değeri video kimliğine indirger.
// Kabul edilenler: çıplak kimlik, watch?v=…, youtu.be/…, /shorts/…, /live/…
func VideoIDCikar(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) == 11 && !strings.ContainsAny(s, "/:? ") {
		return s
	}
	kes := func(ayrac string) string {
		if i := strings.LastIndex(s, ayrac); i >= 0 {
			parca := s[i+len(ayrac):]
			if j := strings.IndexAny(parca, "&?#/ "); j >= 0 {
				parca = parca[:j]
			}
			if len(parca) == 11 {
				return parca
			}
		}
		return ""
	}
	for _, a := range []string{"v=", "youtu.be/", "/shorts/", "/live/", "/embed/"} {
		if id := kes(a); id != "" {
			return id
		}
	}
	return ""
}

// Senkron kanalı bir kez çeker ve arşive yazar.
//
// Dönen Sonuc panel "Şimdi çek" düğmesinde özet göstermek için kullanılır.
// Hata olsa bile yazılabilen kısım yazılır (RSS düşse canlı yayın güncellenir).
func Senkron(ctx context.Context, pool repo.DB, logger *slog.Logger) (Sonuc, error) {
	a := AyarlariOku(ctx, pool)
	if !a.Aktif {
		return Sonuc{}, nil
	}
	c := Client{KanalID: a.KanalID, Handle: a.Handle}
	s, err := c.Cek(ctx)
	if err != nil {
		_ = repo.SetSetting(ctx, pool, "youtube", "son_hata", []byte(strconv.Quote(err.Error())))
		if logger != nil {
			logger.Warn("youtube çekme hatası", "error", err)
		}
		return s, err
	}

	kaydet := func(tur string, oge []Oge) {
		if len(oge) == 0 {
			return
		}
		kayitlar := make([]repo.YoutubeKayit, 0, len(oge))
		for _, o := range oge {
			k := repo.YoutubeKayit{
				VideoID: o.VideoID, Tur: tur, Baslik: o.Baslik, Aciklama: o.Aciklama,
				Kanal: o.Kanal, Gorsel: o.Gorsel, URL: o.URL,
				SureSn: o.SureSn, Izlenme: o.Izlenme,
			}
			if !o.YayinAt.IsZero() {
				t := o.YayinAt
				k.YayinAt = &t
			}
			kayitlar = append(kayitlar, k)
		}
		if _, err := repo.YoutubeKaydet(ctx, pool, kayitlar); err != nil && logger != nil {
			logger.Warn("youtube kayıt", "tur", tur, "error", err)
		}
	}
	kaydet("video", s.Videolar)
	kaydet("short", s.Shorts)
	kaydet("stream", s.Yayinlar)

	canli := s.CanliVideoID
	if a.CanliVideoID != "" {
		canli = a.CanliVideoID // elle işaretleme otomatik algılamayı ezer
	}
	if err := repo.YoutubeCanliAyarla(ctx, pool, canli); err != nil && logger != nil {
		logger.Warn("youtube canlı işareti", "error", err)
	}

	_ = repo.SetSetting(ctx, pool, "youtube", "son_cekme", []byte(strconv.Quote(time.Now().Format(time.RFC3339))))
	_ = repo.SetSetting(ctx, pool, "youtube", "son_hata", []byte(`""`))
	return s, nil
}
