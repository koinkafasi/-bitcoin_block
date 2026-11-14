package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"theblock/site/internal/repo"
)

// OLAY KAYDI — istek başına davranışı diske yazan hat.
//
// NEDEN KUYRUK: middleware her istekte çalışır. İstek yolunda tek INSERT
// yapmak, saniyede yüzlerce istekte bağlantı havuzunu doldurup sayfa
// gecikmesine dönüşürdü — trafik güvenliğinin tüm tasarımı "sayfayı
// yavaşlatma" ilkesine dayanıyor (ASN sorgusunun 120 ms bütçesi gibi).
//
// NEDEN TOPLU YAZMA: 200 satır tek INSERT ile yazılıyor (UNNEST). Kuyruk
// dolduğunda tek tek yazmak yerine elde ne varsa yazılır.
//
// KAYIP KABUL EDİLİR: kanal dolarsa olay DÜŞÜRÜLÜR (sayılır ve loglanır).
// Bir davranış kaydını kaybetmek, sayfayı bekletmekten daha az maliyetli.
// Aynı gerekçe ASN arka plan kuyruğunda da yazılı.

const (
	olayKuyrukBoyu   = 8192
	olayTopluSinir   = 250
	olayYazmaAraligi = 2 * time.Second
	// Olay saklama süresi (gün). Aydınlatma metnindeki "trafik güvenliği
	// kaydı 12 ay" ifadesi visitor_risk için; ham istek kaydı 14 gün.
	olaySaklamaGun = 14
)

var (
	olayKuyruk     chan repo.ZiyaretOlay
	olayKuyrukBir  sync.Once
	olayDusenSayac int64 // atomik değil: yalnız log için, yarış zararsız
	olayDusenMu    sync.Mutex
)

// OlayKayitBaslat kanalı kurar. İşçi ayrı: (s *Server) OlayIsci.

// olaylariBaslat kanalı (bir kez) kurar.
func olaylariBaslat() chan repo.ZiyaretOlay {
	olayKuyrukBir.Do(func() {
		olayKuyruk = make(chan repo.ZiyaretOlay, olayKuyrukBoyu)
	})
	return olayKuyruk
}

// olayYaz olayı kuyruğa bırakır; ASLA bloklamaz.
func olayYaz(o repo.ZiyaretOlay) {
	if o.Anahtar == "" {
		return
	}
	k := olaylariBaslat()
	select {
	case k <- o:
	default:
		olayDusenMu.Lock()
		olayDusenSayac++
		olayDusenMu.Unlock()
	}
}

// OlayIsci kuyruğu boşaltan işçiyi başlatır ve günlük temizliği kurar.
//
// İki görev tek goroutine'de: ikisi de "arka plan, kimseyi bekletmez"
// işi. Ayrı goroutine açmak kapanışta ikisini ayrı izlemeyi gerektirirdi.
func (s *Server) OlayIsci(ctx context.Context) {
	k := olaylariBaslat()

	yaz := func() {
		// DİLİM HER DURUMDA BOŞALTILIR. Havuz yoksa yazma atlanır ama
		// kuyruktan çekmek zorunlu: erken dönüş, olayları kanalda
		// bırakıyor ve bir sonraki turda önlerinde tıkanıyordu (test
		// yakaladı: TestOlayIsciKapanistaBosaltir).
		toplu := make([]repo.ZiyaretOlay, 0, olayTopluSinir)
		for len(toplu) < olayTopluSinir {
			select {
			case o := <-k:
				toplu = append(toplu, o)
			default:
				goto yaz
			}
		}
	yaz:
		if len(toplu) == 0 || s.pool == nil {
			return
		}
		wctx, iptal := context.WithTimeout(ctx, 10*time.Second)
		defer iptal()
		if err := repo.OlayTopluKaydet(wctx, s.pool, toplu); err != nil {
			s.logger.Debug("ziyaret olayları yazılamadı", "adet", len(toplu), "error", err)
		}
	}

	sil := func() {
		if s.pool == nil {
			return
		}
		sctx, iptal := context.WithTimeout(ctx, 30*time.Second)
		defer iptal()
		n, err := repo.OlayTemizle(sctx, s.pool, olaySaklamaGun)
		if err != nil {
			s.logger.Warn("ziyaret olayı temizliği", "error", err)
			return
		}
		if n > 0 {
			s.logger.Info("ziyaret olayları: süresi dolanlar silindi", "adet", n)
		}
	}

	go func() {
		// ZAMANLAYICILAR GOROUTINE'IN İÇİNDE. Bir kez dışarıda
		// kurulmuşlardı ve `defer ticker.Stop()` OlayIsci dönerken
		// çalışıyordu: fonksiyon işçiyi başlatıp hemen döndüğü için
		// ticker DAHA İLK TURU BEKLEMEDEN duruyordu. Sonuç: olaylar
		// yalnız süreç kapanırken yazılıyordu — yani üretimde hiç.
		// Uçtan uca test yakaladı (TestOlayKaydiUctanUca): kuyrukta 3
		// olay 8 saniye bekledi, tabloda 0 satır vardı.
		ticker := time.NewTicker(olayYazmaAraligi)
		defer ticker.Stop()
		temizlik := time.NewTicker(24 * time.Hour)
		defer temizlik.Stop()

		sil()
		for {
			select {
			case <-ctx.Done():
				// Kapanışta elde kalanı yazmayı DENE: son 2 saniyenin
				// kaydı kaybolmasın (en fazla bir tur, sınırlı).
				yaz()
				return
			case <-ticker.C:
				yaz()
			case <-temizlik.C:
				sil()
			}
		}
	}()
}

// durumYazan yanıt kodunu yakalar (olay kaydı için).
//
// NEDEN GEREKLİ: middleware kararı verirken henüz handler çalışmamıştır;
// gerçek HTTP durumu (200/404/403/500) ancak yanıt yazılınca bilinir.
// Sarmalayıcı olmadan "bot 403 aldı" bilgisini kaydedemezdik.
type durumYazan struct {
	http.ResponseWriter
	durum int
}

func (d *durumYazan) WriteHeader(kod int) {
	if d.durum == 0 {
		d.durum = kod
	}
	d.ResponseWriter.WriteHeader(kod)
}

func (d *durumYazan) Write(b []byte) (int, error) {
	if d.durum == 0 {
		d.durum = http.StatusOK
	}
	return d.ResponseWriter.Write(b)
}

// Flush/Unwrap: SSE ve http.ResponseController için sarmalayıcı saydam olmalı.
func (d *durumYazan) Flush() {
	if f, ok := d.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (d *durumYazan) Unwrap() http.ResponseWriter { return d.ResponseWriter }

// olayKaydet istekten davranış satırı üretir ve kuyruğa bırakır.
func (s *Server) olayKaydet(r *http.Request, karar *guvenlikKarari, durum int, engellendi bool) {
	if s.pool == nil || karar == nil {
		return
	}
	yol := yolTemizle(r.URL.Path)
	o := repo.ZiyaretOlay{
		Anahtar:    karar.Anahtar,
		IP:         karar.IP,
		Zaman:      time.Now(),
		Yontem:     r.Method,
		Yol:        yol,
		Sorgu:      sorguTemizle(r.URL.RawQuery),
		Durum:      durum,
		Tur:        olayTuru(yol),
		Kaynak:     olayKaynagi(r.Referer(), s.siteHost()),
		Referer:    kirp(r.Referer(), 300),
		UA:         kirp(r.UserAgent(), 400),
		Sinif:      karar.Sinif,
		Engellendi: engellendi,
	}
	olayYaz(o)
}

// siteHost sitenin kendi alan adı (Referer karşılaştırması için).
func (s *Server) siteHost() string {
	if s.cfg.BaseURL == "" {
		return ""
	}
	u, err := url.Parse(s.cfg.BaseURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// kirp metni s sınırına indirger (rune sınırına saygılı değil; kayıt
// alanı, kullanıcıya basılmıyor — yine de geçersiz UTF-8 bırakmamak için
// rune sınırına inilir).
func kirp(v string, n int) string {
	if len(v) <= n {
		return v
	}
	kes := n
	for kes > 0 && !utf8RuneBaslangici(v[kes]) {
		kes--
	}
	return v[:kes] + "…"
}

// utf8RuneBaslangici b aykırı bir UTF-8 devam baytı mı.
func utf8RuneBaslangici(b byte) bool { return b&0xC0 != 0x80 }

// olayDusen toplam düşen olay sayısı (testler için).
func olayDusen() int64 {
	olayDusenMu.Lock()
	defer olayDusenMu.Unlock()
	return olayDusenSayac
}
