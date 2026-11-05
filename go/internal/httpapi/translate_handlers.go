package httpapi

import (
	"context"
	"net/http"
	"strings"

	"theblock/site/internal/i18n"
	"theblock/site/internal/repo"
)

// ---- Çeviri (i18n) yönetimi (Faz 45) ----

// cevirilerData çeviri editörü sayfa verisi.
type cevirilerData struct {
	Locale     string
	LocaleName string
	Locales    []i18n.Locale
	Keys       []string
	Values     map[string]string // seçili dil
	TrValues   map[string]string // tr referansı
}

// reloadI18n çeviri paketini DB'den yeniden yükler (kayıt sonrası canlıya yansır).
func (s *Server) reloadI18n(ctx context.Context) error {
	nb, err := i18n.Load(ctx, s.pool)
	if err != nil {
		return err
	}
	s.i18nMu.Lock()
	s.i18n = nb
	s.i18nMu.Unlock()
	return nil
}

func (s *Server) handleCevirilerGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildCevirilerData(r)
	if IsHTMX(r) {
		s.renderCevirilerPanelV2(w, r, d)
		return
	}
	s.renderCevirilerPageV2(w, r, d)
}

func (s *Server) buildCevirilerData(r *http.Request) *tmplData {
	ctx := r.Context()
	locale := strings.TrimSpace(r.URL.Query().Get("dil"))
	if locale == "" || !s.i18n.IsSupported(locale) {
		locale = s.defaultLang(ctx)
	}
	keys, _ := repo.ListTranslationKeys(ctx, s.pool)
	vals, _ := repo.ListTranslations(ctx, s.pool, locale)
	trVals, _ := repo.ListTranslations(ctx, s.pool, "tr")

	d := s.baseView("tr", "/admin", "Çeviriler")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.Ceviriler = &cevirilerData{
		Locale:     locale,
		LocaleName: nameOfLocale(s.i18n.Enabled(), locale),
		Locales:    s.i18n.Enabled(),
		Keys:       keys,
		Values:     vals,
		TrValues:   trVals,
	}
	return d
}

func (s *Server) ceviriResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	locale := strings.TrimSpace(r.FormValue("dil"))
	if locale == "" {
		locale = r.URL.Query().Get("dil")
	}
	if locale == "" {
		locale = s.defaultLang(r.Context())
	}
	if !IsHTMX(r) {
		s.flashRedirect(w, r, "/admin/ceviriler?dil="+locale, kind, text)
		return
	}
	r2 := withHTMXCurrentQuery(r)
	q := r2.URL.Query()
	q.Set("dil", locale)
	r2.URL.RawQuery = q.Encode()
	d := s.buildCevirilerData(r2)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderCevirilerPanelV2(w, r, d)
}

func nameOfLocale(list []i18n.Locale, code string) string {
	for _, l := range list {
		if l.Code == code {
			return l.NativeName
		}
	}
	return code
}

// handleCevirilerSave seçili dilin tüm değerlerini kaydeder (v<anahtar> alanları).
func (s *Server) handleCevirilerSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	locale := strings.TrimSpace(r.FormValue("dil"))
	if locale == "" || !s.i18n.IsSupported(locale) {
		s.ceviriResult(w, r, "error", "Geçersiz dil.")
		return
	}
	for _, key := range r.Form["anahtar"] {
		val := r.FormValue("v" + key)
		_ = repo.UpsertTranslation(ctx, s.pool, locale, key, val)
	}
	if err := s.reloadI18n(ctx); err != nil {
		s.logger.Error("i18n yeniden yükleme", "error", err)
	}
	s.audit(r, "ceviri_kaydet", locale)
	s.ceviriResult(w, r, "info", "Çeviriler kaydedildi ("+locale+").")
}

// handleCevirilerAddKey yeni anahtarı tüm dillere ekler.
func (s *Server) handleCevirilerAddKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := strings.TrimSpace(r.FormValue("anahtar"))
	if key == "" {
		s.ceviriResult(w, r, "error", "Anahtar boş olamaz.")
		return
	}
	if strings.ContainsAny(key, " \"'<>") {
		s.ceviriResult(w, r, "error", "Anahtarda boşluk/tırnak olmaz.")
		return
	}
	if err := repo.EnsureTranslationKey(ctx, s.pool, key); err != nil {
		s.ceviriResult(w, r, "error", "Kaydedilemedi: "+err.Error())
		return
	}
	_ = s.reloadI18n(ctx)
	s.audit(r, "ceviri_anahtar_ekle", key)
	s.ceviriResult(w, r, "info", "Anahtar eklendi: "+key)
}

// handleCevirilerDelKey anahtarı tüm dillerden siler.
func (s *Server) handleCevirilerDelKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := strings.TrimSpace(r.FormValue("sil_anahtar"))
	if key == "" {
		s.ceviriResult(w, r, "error", "Anahtar boş olamaz.")
		return
	}
	if err := repo.DeleteTranslationKey(ctx, s.pool, key); err != nil {
		s.ceviriResult(w, r, "error", "Silinemedi: "+err.Error())
		return
	}
	_ = s.reloadI18n(ctx)
	s.audit(r, "ceviri_anahtar_sil", key)
	s.ceviriResult(w, r, "info", "Anahtar silindi: "+key)
}
