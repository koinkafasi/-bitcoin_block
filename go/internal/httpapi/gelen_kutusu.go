package httpapi

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// handleGelenKutusu GET /admin/gelen-kutusu — birleşik gelen kutusu (Faz 167).
//
// Üç kaynağı (iletişim, geri bildirim, yorum) tek listede gösterir.
// SALT OKUMA: satırlar kendi ekranına yönlendirir; işlem orada yapılır.
func (s *Server) handleGelenKutusu(w http.ResponseWriter, r *http.Request) {
	d := s.baseView("tr", "/admin", "Gelen Kutusu")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)

	ctx := r.Context()
	sid := s.curSiteID(r)
	suzgec := strings.TrimSpace(r.URL.Query().Get("tur"))

	oge, err := repo.GelenKutusu(ctx, s.pool, sid, 40)
	if err != nil {
		s.logger.Error("gelen kutusu", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	sayilar, err := repo.GelenSayilar(ctx, s.pool, sid)
	if err != nil {
		// Sayaç hatası listeyi engellememeli: kartlar 0 gösterir.
		s.logger.Warn("gelen kutusu sayaçları", "error", err)
		sayilar = map[string]int{}
	}

	// Süzer: "hepsi" ya da boş ise hepsi.
	if suzgec != "" && suzgec != "hepsi" {
		f := oge[:0]
		for _, o := range oge {
			if o.Tur == suzgec {
				f = append(f, o)
			}
		}
		oge = f
	}

	v := templkit.GelenKutuView{
		YeniMesaj:  sayilar["iletisim"],
		YeniBildir: sayilar["bildirim"],
		YeniYorum:  sayilar["yorum"],
		Suzgec:     suzgec,
	}
	for _, o := range oge {
		v.Satirlar = append(v.Satirlar, templkit.GelenSatir{
			Tur:    o.Tur,
			TurAd:  o.TurAd,
			Ikon:   o.Ikon,
			Baslik: o.Baslik,
			Ozet:   o.Ozet,
			Kimden: o.Kimden,
			Durum:  o.Durum,
			Href:   o.Href,
			Zaman:  gelenZaman(o.Zaman),
			Yeni:   o.Durum == "yeni" || o.Durum == "bekliyor",
		})
	}

	var sb strings.Builder
	if err := templkit.GelenKutuPanel(v).Render(ctx, &sb); err != nil {
		s.logger.Error("gelen kutusu paneli", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	d.SlotHTML = template.HTML(sb.String())
	s.render(w, r, "admin_gelen_kutusu", d)
}

// gelenZaman göreli zaman: bugünse saat, dünse "Dün", else tarih.
//
// NEDEN göreli: gelen kutusunda amaç "ne zaman geldi" değil "ne kadar taze".
// Mutlak tarih 40 satırda taramayı zorlaştırıyor.
func gelenZaman(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	simdi := time.Now()
	bugun := time.Date(simdi.Year(), simdi.Month(), simdi.Day(), 0, 0, 0, 0, simdi.Location())
	gun := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, simdi.Location())
	switch {
	case gun.Equal(bugun):
		return t.Format("15:04")
	case gun.Equal(bugun.AddDate(0, 0, -1)):
		return "Dün " + t.Format("15:04")
	default:
		return t.Format("02.01.2006")
	}
}
