package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// WebSub (PubSubHubbub) yayın bildirimi.
//
// NE YAPAR: RSS/Atom beslemesi güncellendiğinde hub'a "kaynağı yeniden çek"
// der; hub da abone arama motorlarına anında dağıtır. Google'ın beslemeyi
// taramasını beklemek yerine saniyeler içinde görülür — haber siteleri için
// indeksleme gecikmesini kısaltan standart yöntem.
//
// GÜVENLİK: hub adresleri BEYAZ LİSTEDEN gelir (validWebSubHubs). Serbest
// bırakılsaydı panelden iç ağ adresine istek attırılabilirdi (SSRF).
//
// HATA SİYASETİ: bildirim başarısız olursa yayın ASLA düşmez. Bu bir
// hızlandırıcı, kritik yol değil.

// websubBildir yayınlanan içerik için hub'lara ping atar.
//
// feedAdresi: sitenin RSS adresi (hub bunu kaynak olarak çeker).
func (s *Server) websubBildir(ctx context.Context, feedAdresi string) {
	e := s.cdEntegrasyonCoz(ctx)
	hubs := e.WebSubHubs
	if len(hubs) == 0 {
		// Ayar boşsa Google hub'ı: `rel="hub"` etiketiyle aynı varsayılan.
		hubs = []string{"https://pubsubhubbub.appspot.com/"}
	}
	if !strings.HasPrefix(feedAdresi, "https://") {
		return
	}

	for _, hub := range hubs {
		// Ayrı kısa zaman aşımı: hub yavaşsa istek yolu beklemesin.
		istekCtx, iptal := context.WithTimeout(ctx, 8*time.Second)
		form := url.Values{
			"hub.mode":   {"publish"},
			"hub.topic":  {feedAdresi},
			"hub.verify": {"async"},
			"hub.url":    {feedAdresi},
		}
		req, err := http.NewRequestWithContext(istekCtx, http.MethodPost,
			strings.TrimRight(hub, "/")+"/", strings.NewReader(form.Encode()))
		if err != nil {
			iptal()
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "bitcosar/1.0 (+https://bitcosar.com)")

		// SSRF korumali istemci: hub adresi beyaz listeden gelse de
		// yonlendirme ile ic aga dusme ihtimaline karsi ayni kapidan gecer.
		resp, err := platform.DogrulaVeCekUA(ctx, req.URL.String(), 8*time.Second,
			"bitcosar/1.0 (+https://bitcosar.com)")
		if err != nil {
			iptal()
			s.logger.Warn("websub bildirim hatası", "hub", hub, "error", err)
			continue
		}
		resp.Body.Close()
		iptal()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			s.logger.Warn("websub hub yanıtı", "hub", hub, "status", resp.StatusCode)
		}
	}
}

// websubYayinBildir yayınlanan içerikler için besleme adresiyle ping atar.
// İçerik kimlikleri yalnız "gerçekten yayında mı" kontrolü için kullanılır.
func (s *Server) websubYayinBildir(ctx context.Context, ids []int64) {
	if len(ids) == 0 {
		return
	}
	yayinda := false
	for _, id := range ids {
		c, err := repo.GetContent(ctx, s.pool, id)
		if err != nil {
			continue
		}
		if c.Status == "yayinda" {
			yayinda = true
			break
		}
	}
	if !yayinda {
		return
	}
	// Kanonik kök istek nesnesi olmadan da çözülebilir (r == nil):
	// varsayılan taban adres APP_BASE_URL'den gelir.
	kok := strings.TrimRight(s.kanonikKok(nil), "/")
	if kok == "" {
		return
	}
	s.websubBildir(ctx, kok+"/rss.xml")
}
