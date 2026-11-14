package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
)

// contactAdminData yönetim ekranının şablon verisi.
type contactAdminData struct {
	Config   ContactConfig
	Previews []contactPreview
	Messages []repo.ContactMessage
	Yeni     int
	Okundu   int
	Arsiv    int
	Filtre   string
	Topics   map[string]string
}

type contactPreview struct {
	Label string
	Value string
	Token string
}

// handleContactAdminGet /admin/iletisim ekranını basar.
func (s *Server) handleContactAdminGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	d := s.baseView("tr", "/admin", "İletişim & Form")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	cfg := s.contactConfig(ctx)
	filter := strings.TrimSpace(r.URL.Query().Get("durum"))
	data := &contactAdminData{Config: cfg, Filtre: filter}
	for _, f := range cfg.emailFields() {
		data.Previews = append(data.Previews, contactPreview{Label: f.Label, Value: *f.Ptr, Token: f.Hint})
	}
	sid := s.curSiteID(r)
	msgs, err := repo.ListContactMessages(ctx, s.pool, sid, filter, 200)
	if err != nil {
		s.logger.Warn("iletişim mesajları okunamadı", "err", err)
	}
	data.Messages = msgs
	data.Yeni, data.Okundu, data.Arsiv, _ = repo.CountContactMessages(ctx, s.pool, sid)
	data.Topics = map[string]string{}
	for _, t := range contactTopics() {
		data.Topics[t.Key] = t.Name
	}
	// "spam" yalnız panel etiketidir: formda SEÇENEK OLARAK GÖRÜNMEZ
	// (kullanıcı kendini spam diye işaretlemesin), ama bağlantı sayısı
	// yüksek mesajlar bu konuya düşer ve panelde okunur bir adı olmalı —
	// aksi hâlde rozet BOŞ çıkıyordu.
	data.Topics["spam"] = "Spam şüphesi"
	d.Iletisim = data
	s.render(w, r, "iletisimadmin", d)
}

// handleContactAdminSave iletişim bilgilerini kaydeder.
func (s *Server) handleContactAdminSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	old := s.contactConfig(ctx)
	c := ContactConfig{
		Domain:    r.FormValue("domain"),
		Info:      r.FormValue("eposta_info"),
		Haber:     r.FormValue("eposta_haber"),
		Reklam:    r.FormValue("eposta_reklam"),
		Destek:    r.FormValue("eposta_destek"),
		Telif:     r.FormValue("eposta_telif"),
		Duzeltme:  r.FormValue("eposta_duzeltme"),
		KVKK:      r.FormValue("eposta_kvkk"),
		Kariyer:   r.FormValue("eposta_kariyer"),
		Telefon:   r.FormValue("telefon"),
		Adres:     r.FormValue("adres"),
		Calisma:   r.FormValue("calisma"),
		FormAktif: r.FormValue("form_aktif") != "",
	}
	c = normalizeContactConfig(c)
	// Alan adı değiştiyse eski alan adını taşıyan adresler yeni alan adına
	// geçer; özel (başka alan adlı) adreslere dokunulmaz.
	c = shiftEmailsToDomain(c, old.Domain)
	// Geçersiz adresler kaydedilmez: mevcut değer korunur.
	for _, f := range c.emailFields() {
		if !validEmail(*f.Ptr) {
			s.flashRedirect(w, r, "/admin/iletisim", "error",
				"Geçersiz e-posta adresi: "+f.Label)
			return
		}
	}
	raw, _ := json.Marshal(c)
	if err := repo.SetSetting(ctx, s.pool, "iletisim", "ana", raw); err != nil {
		s.flashRedirect(w, r, "/admin/iletisim", "error", "Kaydedilemedi: "+err.Error())
		return
	}
	s.invalidateContentCache()
	// Alan adı değiştiyse gömülü kalmış eski alan adı metinleri de düzeltilir:
	// sayfa gövdeleri + Sayfa Tasarımcısı metinleri tek seferde güncellenir.
	if old.Domain != "" && !strings.EqualFold(old.Domain, c.Domain) {
		if p, t, err := repo.ReplaceTextInPages(ctx, s.pool, old.Domain, c.Domain); err == nil {
			s.audit(r, "iletisim_domain_degisim",
				old.Domain+" → "+c.Domain+" ("+strconv.FormatInt(p, 10)+" sayfa, "+strconv.FormatInt(t, 10)+" metin)")
		} else {
			s.logger.Warn("eski alan adı metinleri güncellenemedi", "err", err)
		}
	}
	s.audit(r, "iletisim_ayar", c.Domain)
	s.flashRedirect(w, r, "/admin/iletisim", "ok", "İletişim bilgileri kaydedildi. Tüm sayfalar güncellendi.")
}

// handleContactReplace seçili metni sayfa gövdelerinde ve tasarımcı
// metinlerinde toplu olarak değiştirir (ör. eski alan adı → yeni alan adı).
func (s *Server) handleContactReplace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	from := strings.TrimSpace(r.FormValue("eski"))
	to := strings.TrimSpace(r.FormValue("yeni"))
	if from == "" {
		s.flashRedirect(w, r, "/admin/iletisim", "error", "Aranacak metin boş olamaz.")
		return
	}
	pages, texts, err := repo.ReplaceTextInPages(ctx, s.pool, from, to)
	if err != nil {
		s.flashRedirect(w, r, "/admin/iletisim", "error", "Değiştirilemedi: "+err.Error())
		return
	}
	s.invalidateContentCache()
	s.audit(r, "iletisim_toplu_degisim", from+" → "+to)
	s.flashRedirect(w, r, "/admin/iletisim", "ok",
		"Değiştirildi: "+strconv.FormatInt(pages, 10)+" sayfa, "+
			strconv.FormatInt(texts, 10)+" metin kaydı.")
}

// handleContactMessageStatus mesajı okundu/arşiv olarak işaretler.
func (s *Server) handleContactMessageStatus(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id := parseID(r.PathValue("id"))
	status := strings.TrimSpace(r.FormValue("durum"))
	switch status {
	case "yeni", "okundu", "arsiv", "sil":
	default:
		status = "okundu"
	}
	if id <= 0 {
		s.flashRedirect(w, r, "/admin/iletisim", "error", "Geçersiz mesaj.")
		return
	}
	if status == "sil" {
		if err := repo.DeleteContactMessage(r.Context(), s.pool, id); err != nil {
			s.flashRedirect(w, r, "/admin/iletisim", "error", "Silinemedi.")
			return
		}
		s.audit(r, "iletisim_mesaj_sil", strconv.FormatInt(id, 10))
	} else {
		if err := repo.SetContactMessageStatus(r.Context(), s.pool, id, status); err != nil {
			s.flashRedirect(w, r, "/admin/iletisim", "error", "Güncellenemedi.")
			return
		}
	}
	back := "/admin/iletisim"
	if f := r.FormValue("filtre"); f != "" {
		back += "?durum=" + url.QueryEscape(f)
	}
	s.flashRedirect(w, r, back, "ok", "Mesaj güncellendi.")
}
