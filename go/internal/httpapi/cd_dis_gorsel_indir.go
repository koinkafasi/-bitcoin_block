package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// Dış kaynaklı görseli medya kütüphanesine indirme.
//
// NEDEN AYRI BİRİM: aynı akış `handleMediaURLUpload` içinde vardı ama
// HTTP isteğine gömülüydü (form okuma, flash mesaj, HTMX yanıtı). Arka
// plan işçisi oradan çağıramıyordu. Çekirdek buraya alındı; iki çağıran
// da aynı SSRF korumasını, aynı boyut sınırını ve aynı `source` değerini
// kullanıyor.
//
// NEDEN ÖNEMLİ: içerik gövdesindeki görsel BAŞKASININ sunucusunda
// durdukça o sunucu kapandığında yazı sakatlanır. 13 Kasım 2026'da
// ölçüldü: Ghost'tan aktarılan 50 yazının 47 gövde görseli
// ghost.bitcosar.com'a, 50 kapak görseli cdn.coinhit.net'e bakıyordu ve
// ghost.bitcosar.com kapatılıyor.

const (
	// disGorselAzamiBoyut tek görsel için okunacak en fazla bayt.
	disGorselAzamiBoyut = 30 << 20
	// disGorselZamanAsimi tek indirme için üst süre.
	disGorselZamanAsimi = 25 * time.Second
)

// ErrDisGorselYok indirilecek geçerli bir görsel bulunamadı.
var ErrDisGorselYok = errors.New("dış görsel indirilemedi")

// disGorselIndir adresteki görseli medya kütüphanesine alır ve medya
// kimliğini döndürür.
//
// `kaynak` alanı HER ZAMAN "internet": AI görsel üretici geldiğinde
// hangi görsellerin dışarıdan geldiği buradan bulunacak
// (repo.DisKaynakliMedyalar).
func (s *Server) disGorselIndir(ctx context.Context, adres, baslik, alt string, siteID, userID int64) (int64, error) {
	hedef, err := validateBookmarkURL(strings.TrimSpace(adres))
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrDisGorselYok, err)
	}
	ictx, iptal := context.WithTimeout(ctx, disGorselZamanAsimi)
	defer iptal()

	// #nosec G704 -- adres validateBookmarkURL'den geçti (şema, kullanıcı
	// bilgisi, localhost ve özel ağ reddi) ve istek safeRemoteHTTPClient
	// ile atılıyor: DialContext her bağlantıda DNS'i yeniden çözüp özel
	// IP'yi reddediyor. Bkz. handleMediaURLUpload'daki aynı gerekçe.
	istek, err := http.NewRequestWithContext(ictx, http.MethodGet, hedef.String(), nil)
	if err != nil {
		return 0, ErrDisGorselYok
	}
	istek.Header.Set("User-Agent", "TheBlockBot/1.0 (+https://bitcosar.com)")
	istek.Header.Set("Accept", "image/*")

	// #nosec G704 -- safeRemoteHTTPClient her istekte ve her yönlendirmede
	// hedefi yeniden doğruluyor (genel IP, 80/443).
	yanit, err := safeRemoteHTTPClient(disGorselZamanAsimi).Do(istek)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrDisGorselYok, err)
	}
	defer yanit.Body.Close()
	if yanit.StatusCode < 200 || yanit.StatusCode >= 400 {
		return 0, fmt.Errorf("%w: HTTP %d", ErrDisGorselYok, yanit.StatusCode)
	}
	if ct := strings.ToLower(yanit.Header.Get("Content-Type")); ct != "" && !strings.HasPrefix(ct, "image/") {
		return 0, fmt.Errorf("%w: içerik türü %q", ErrDisGorselYok, ct)
	}

	veri, err := io.ReadAll(io.LimitReader(yanit.Body, disGorselAzamiBoyut))
	if err != nil || len(veri) == 0 {
		return 0, ErrDisGorselYok
	}
	cozulmus, err := platform.DecodeImage(veri)
	if err != nil {
		return 0, fmt.Errorf("%w: desteklenmeyen biçim", ErrDisGorselYok)
	}
	sinir := cozulmus.Bounds()
	genislik, yukseklik := sinir.Dx(), sinir.Dy()

	ad := disGorselDosyaAdi(hedef.Path, veri)
	yol, err := platform.SaveMedia(veri, ad)
	if err != nil {
		return 0, fmt.Errorf("%w: depolama: %w", ErrDisGorselYok, err)
	}

	var kucukYol *string
	if kucuk, err := platform.CreateThumb(veri, 480); err == nil {
		if p, err := platform.SaveMedia(kucuk, "thumb.jpg"); err == nil {
			kucukYol = &p
		}
	}

	if strings.TrimSpace(baslik) == "" {
		baslik = ad
	}
	boyut := int64(len(veri))
	mime, _ := platformFileExt(ad)
	altMetin := kirpAltMetin(alt, 125)
	bosAciklama := ""

	id, err := repo.CreateMedia(ctx, s.pool, &repo.Media{
		SiteID: siteID, UserID: userID, Kind: "image", Filename: ad,
		Path: yol, Mime: mime, SizeBytes: &boyut,
		Width: &genislik, Height: &yukseklik,
		Title: kirpAltMetin(baslik, 120), Category: "haber",
		Alt: &altMetin, Caption: &bosAciklama,
		ThumbPath: kucukYol,
		// Dış kaynak işareti: AI görsel üretici geldiğinde bu kayıtlar
		// yeniden üretilecek.
		Source: "internet",
	})
	if err != nil {
		// Kayıt başarısızsa diskte yetim dosya bırakma.
		_ = platform.DeleteMedia(yol)
		if kucukYol != nil {
			_ = platform.DeleteMedia(*kucukYol)
		}
		return 0, fmt.Errorf("%w: kayıt: %w", ErrDisGorselYok, err)
	}
	return id, nil
}

// disGorselDosyaAdi adresten okunur bir dosya adı üretir.
//
// Uzantı İÇERİKTEN belirleniyor, adresten değil: Ghost ve CDN'ler
// uzantısız ya da yanlış uzantılı adres verebiliyor.
func disGorselDosyaAdi(yol string, veri []byte) string {
	ad := filepath.Base(strings.TrimSpace(yol))
	taban := strings.TrimSuffix(ad, filepath.Ext(ad))
	taban = temizDosyaAdi(taban)
	if taban == "" || taban == "." || taban == "/" {
		taban = "dis-gorsel"
	}
	if len(taban) > 60 {
		taban = taban[:60]
	}
	uzanti := ".jpg"
	switch ct := http.DetectContentType(veri); {
	case strings.Contains(ct, "png"):
		uzanti = ".png"
	case strings.Contains(ct, "webp"):
		uzanti = ".webp"
	case strings.Contains(ct, "gif"):
		uzanti = ".gif"
	}
	return taban + uzanti
}
