package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ---- Aboneler merkezi: bülten + gazete abone listeleri tek sayfada ----
// (denetim açık maddesi: iki abone listesinin çoğalmasını toparlar)

// buildAbonelerView birleşik abone görünümünü kurar.
func (s *Server) buildAbonelerView(r *http.Request) (templkit.AbonelerView, *tmplData) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	durum := strings.TrimSpace(r.URL.Query().Get("durum"))
	kind, text := "", ""
	if f := s.consumeFlash(r); f != nil {
		kind, text = f.Kind, f.Text
	}
	v := templkit.AbonelerView{FlashKind: kind, FlashText: text, Durum: durum}
	d := s.baseView("tr", "/admin", "Aboneler")
	d.User = ptr(userFromContext(ctx))

	bCounts := repo.CountSubscribers(ctx, s.pool, sid)
	gAktif, _ := repo.CountGazeteSubscriptions(ctx, s.pool, sid)
	gTumu, _ := repo.ListGazeteSubscriptions(ctx, s.pool, sid, "", 500)
	wa := 0
	for _, g := range gTumu {
		if g.Channel == "whatsapp" && g.Status == "aktif" {
			wa++
		}
	}
	v.Stats = []templkit.AboneStat{
		{Label: "Bülten Aktif", Value: strconv.Itoa(bCounts["aktif"]), Tone: "ok"},
		{Label: "Bülten Onay Bekleyen", Value: strconv.Itoa(bCounts["onay bekliyor"]), Tone: "warn"},
		{Label: "Gazete Aktif", Value: strconv.FormatInt(gAktif, 10), Tone: "ok"},
		{Label: "WhatsApp Abonesi", Value: strconv.Itoa(wa)},
		{Label: "Toplam Kayıt", Value: strconv.Itoa(len(gTumu) + bCounts["aktif"] + bCounts["onay bekliyor"] + bCounts["iptal"])},
	}

	if subs, err := repo.ListSubscribers(ctx, s.pool, sid, durum, 300); err == nil {
		for _, b := range subs {
			onay := "—"
			if b.ConfirmedAt != nil {
				onay = b.ConfirmedAt.Format("02.01.2006")
			}
			v.Bulten = append(v.Bulten, templkit.BultenAbone{
				ID: strconv.FormatInt(b.ID, 10), Email: b.Email, Locale: b.Locale,
				Status: b.Status, Time: b.CreatedAt.Format("02.01.2006 15:04"), Onay: onay,
			})
		}
	}
	// Gazete listesi: "onay bekliyor" filtresi gazetede yok → tümü gibi ele alınmaz.
	if durum != "onay bekliyor" {
		if subs, err := repo.ListGazeteSubscriptions(ctx, s.pool, sid, durum, 300); err == nil {
			for _, g := range subs {
				v.Gazete = append(v.Gazete, templkit.GazeteAbone{
					ID:        strconv.FormatInt(g.ID, 10),
					Email:     g.Email,
					Phone:     g.Phone,
					Channel:   g.Channel,
					Frequency: g.Frequency,
					Status:    g.Status,
					Time:      g.CreatedAt.Format("02.01.2006 15:04"),
				})
			}
		}
	}
	return v, d
}

// handleAbonelerGet GET /admin/aboneler — birleşik abone merkezi.
func (s *Server) handleAbonelerGet(w http.ResponseWriter, r *http.Request) {
	v, d := s.buildAbonelerView(r)
	s.renderTemplPage(w, r, d, templkit.AbonelerPanel(v))
}

// abonelerRefresh aksiyon sonrası paneli taze basar.
func (s *Server) abonelerRefresh(w http.ResponseWriter, r *http.Request, kind, text string) {
	r2 := withHTMXCurrentQuery(r)
	v, _ := s.buildAbonelerView(r2)
	v.FlashKind, v.FlashText = kind, text
	s.renderTemplPanel(w, r, templkit.AbonelerPanel(v))
}

// ---- Bülten aboneleri ----

func (s *Server) handleAbonelerBultenEkle(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	if email == "" || !strings.Contains(email, "@") {
		s.abonelerRefresh(w, r, "error", "Geçerli bir e-posta girin.")
		return
	}
	if _, err := repo.CreateSubscriber(r.Context(), s.pool, s.curSiteID(r), email, "tr"); err != nil {
		s.abonelerRefresh(w, r, "error", "Eklenemedi: "+err.Error())
		return
	}
	s.audit(r, "abone_bulten_ekle", email)
	s.abonelerRefresh(w, r, "info", "Bülten abonesi eklendi: "+email)
}

func (s *Server) handleAbonelerBultenDurum(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	status := strings.TrimSpace(r.FormValue("durum"))
	if status != "aktif" && status != "iptal" {
		s.abonelerRefresh(w, r, "error", "Geçersiz durum.")
		return
	}
	if err := repo.SetSubscriberStatus(r.Context(), s.pool, id, status); err != nil {
		s.abonelerRefresh(w, r, "error", "Güncellenemedi.")
		return
	}
	s.audit(r, "abone_bulten_durum", strconv.FormatInt(id, 10)+"="+status)
	s.abonelerRefresh(w, r, "info", "Bülten abonesi durumu güncellendi.")
}

func (s *Server) handleAbonelerBultenSil(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteSubscriber(r.Context(), s.pool, id); err != nil {
		s.abonelerRefresh(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "abone_bulten_sil", strconv.FormatInt(id, 10))
	s.abonelerRefresh(w, r, "info", "Bülten abonesi silindi (KVKK).")
}

// ---- Gazete aboneleri ----

func (s *Server) handleAbonelerGazeteEkle(w http.ResponseWriter, r *http.Request) {
	channel := strings.TrimSpace(r.FormValue("kanal"))
	sub := &repo.GazeteSubscription{
		SiteID:    s.curSiteID(r),
		Email:     strings.TrimSpace(r.FormValue("email")),
		Phone:     strings.TrimSpace(r.FormValue("telefon")),
		Channel:   channel,
		Frequency: strings.TrimSpace(r.FormValue("siklik")),
		Status:    "aktif",
		OptInIP:   r.RemoteAddr,
		OptInAt:   ptrTime(time.Now()),
	}
	if sub.Channel == "whatsapp" && sub.Phone == "" {
		s.abonelerRefresh(w, r, "error", "WhatsApp abonesi için telefon gerekli.")
		return
	}
	if sub.Channel != "whatsapp" && sub.Email == "" {
		s.abonelerRefresh(w, r, "error", "E-posta gerekli.")
		return
	}
	if _, err := repo.CreateGazeteSubscription(r.Context(), s.pool, sub); err != nil {
		msg := "Eklenemedi."
		if strings.Contains(err.Error(), "uq_gazete_subs") {
			msg = "Bu e-posta/telefon zaten kayıtlı."
		}
		s.abonelerRefresh(w, r, "error", msg)
		return
	}
	s.audit(r, "abone_gazete_ekle", sub.Email+sub.Phone)
	s.abonelerRefresh(w, r, "info", "Gazete abonesi eklendi.")
}

func (s *Server) handleAbonelerGazeteDurum(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	status := strings.TrimSpace(r.FormValue("durum"))
	if status != "aktif" && status != "iptal" {
		s.abonelerRefresh(w, r, "error", "Geçersiz durum.")
		return
	}
	if err := repo.SetGazeteSubscriptionStatus(r.Context(), s.pool, id, status); err != nil {
		s.abonelerRefresh(w, r, "error", "Güncellenemedi.")
		return
	}
	s.audit(r, "abone_gazete_durum", strconv.FormatInt(id, 10)+"="+status)
	s.abonelerRefresh(w, r, "info", "Gazete abonesi durumu güncellendi.")
}

func (s *Server) handleAbonelerGazeteSil(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteGazeteSubscription(r.Context(), s.pool, id); err != nil {
		s.abonelerRefresh(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "abone_gazete_sil", strconv.FormatInt(id, 10))
	s.abonelerRefresh(w, r, "info", "Gazete abonesi silindi (KVKK).")
}

// ptrTime *time.Time yardımcısı.
func ptrTime(t time.Time) *time.Time { return &t }
