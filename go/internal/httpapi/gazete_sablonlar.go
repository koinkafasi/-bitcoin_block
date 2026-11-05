package httpapi

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ---- Gazete şablon galerisi + editörden şablon kaydetme (templ modülü) ----

// sablonPagePreview şablon sayfası JSON'undan mini önizleme verisi çıkarır.
func sablonPagePreview(raw []byte) []templkit.PageDoc {
	// Hem yeni belge bicimini hem eski duz sayfa dizisini kabul et.
	belge, err := gazeteSablonCoz(raw)
	if err != nil {
		return nil
	}
	docs := belge.Sayfalar
	var out []templkit.PageDoc
	for _, d := range docs {
		pd := templkit.PageDoc{No: d.No, Ad: d.Ad}
		for _, b := range d.Blocks {
			pd.Boxes = append(pd.Boxes, templkit.BlockBox{
				X: b.X, Y: b.Y, W: b.W, H: b.H, Tip: b.Tip,
			})
		}
		out = append(out, pd)
	}
	return out
}

// handleGazeteSablonlar GET /admin/gazete/sablonlar — şablon galerisi (templ).
func (s *Server) handleGazeteSablonlar(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionView) {
		return
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	tpls, _ := repo.ListGazeteTemplates(ctx, s.pool, sid)
	// Galeri bossa hazir peyzajlari yukle (Faz 3). Isme gore esledigi icin
	// tekrar calismasi guvenli; kullanicinin kendi sablonlarina dokunmaz.
	if len(tpls) == 0 {
		if n, err := s.gazeteSablonTohumla(ctx, sid); err != nil {
			s.logger.Error("gazete sablon tohumlama", "error", err)
		} else if n > 0 {
			s.logger.Info("gazete hazir sablonlari yuklendi", "adet", n, "site", sid)
			tpls, _ = repo.ListGazeteTemplates(ctx, s.pool, sid)
		}
	}
	var items []templkit.Sablon
	for _, t := range tpls {
		pages := sablonPagePreview(t.Pages)
		if len(pages) > 10 {
			pages = pages[:10]
		}
		items = append(items, templkit.Sablon{
			ID: t.ID, Ad: t.Name, Aciklama: t.Description,
			Varsayilan: t.IsDefault, SayfaSayisi: len(pages), Pages: pages,
		})
	}
	d := s.baseView("tr", "/admin", "Gazete Şablonları")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	var sb strings.Builder
	if err := templkit.SablonGaleri(items).Render(ctx, &sb); err != nil {
		s.logger.Error("şablon galeri", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	d.SlotHTML = template.HTML(sb.String())
	s.render(w, r, "admin_bilesenler", d)
}

// handleGazeteSablonJSON GET /admin/gazete/sablonlar/{id} — editör için şablon sayfaları.
func (s *Server) handleGazeteSablonJSON(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionView) {
		return
	}
	t, err := repo.GetGazeteTemplate(r.Context(), s.pool, parseID(r.PathValue("id")))
	if err != nil {
		http.Error(w, "şablon bulunamadı", http.StatusNotFound)
		return
	}
	// Sablon belgesi nesne bicimindedir; editor yalniz sayfa dizisini bekler.
	// Eski duz dizi bicimi de cozulur (gazeteSablonCoz).
	d, err := gazeteSablonCoz(t.Pages)
	if err != nil {
		s.logger.Warn("gazete sablonu cozulemedi", "sablon", t.ID, "error", err)
		http.Error(w, "sablon okunamadi", http.StatusUnprocessableEntity)
		return
	}
	sayfalar, err := json.Marshal(d.Sayfalar)
	if err != nil {
		http.Error(w, "sablon okunamadi", http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ad":       t.Name,
		"sayfalar": json.RawMessage(sayfalar),
		"sayfa":    d.Sayfa,
		"izgara":   d.Izgara,
	})
}

// handleGazeteSablonVarsayilan POST /admin/gazete/sablonlar/{id}/varsayilan.
func (s *Server) handleGazeteSablonVarsayilan(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionTemplate) {
		return
	}
	id := parseID(r.PathValue("id"))
	if err := repo.SetDefaultGazeteTemplate(r.Context(), s.pool, s.curSiteID(r), id); err != nil {
		s.flashRedirect(w, r, "/admin/gazete/sablonlar", "error", "Varsayılan ayarlanamadı.")
		return
	}
	s.audit(r, "gazete_sablon_varsayilan", strconv.FormatInt(id, 10))
	s.flashRedirect(w, r, "/admin/gazete/sablonlar", "info", "Varsayılan şablon güncellendi.")
}

// handleGazeteSayiSablonaKaydet POST /admin/gazete/sayilar/{id}/sablona-kaydet —
// sayının mevcut sayfa düzenlerini yeni şablona çevirir (editördeki "Şablon olarak Kaydet").
func (s *Server) handleGazeteSayiSablonaKaydet(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionTemplate) {
		return
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	issueID := parseID(r.PathValue("id"))
	name := strings.TrimSpace(r.FormValue("ad"))
	if name == "" {
		s.flashRedirect(w, r, "/admin/gazete/sablonlar", "error", "Şablon adı gerekli.")
		return
	}
	pages, err := repo.ListGazetePages(ctx, s.pool, issueID)
	if err != nil || len(pages) == 0 {
		s.flashRedirect(w, r, "/admin/gazete/sablonlar", "error", "Sayıda kayıtlı sayfa yok.")
		return
	}
	var docs []gazetePageDoc
	for _, p := range pages {
		var blocks []gazeteBlock
		_ = json.Unmarshal(p.Blocks, &blocks)
		if blocks == nil {
			blocks = []gazeteBlock{}
		}
		docs = append(docs, gazetePageDoc{No: p.PageNo, Ad: p.PageKey, Blocks: blocks})
	}
	raw, err := json.Marshal(docs)
	if err != nil {
		s.flashRedirect(w, r, "/admin/gazete/sablonlar", "error", "Şablon üretilemedi.")
		return
	}
	u := userFromContext(ctx)
	t := &repo.GazeteTemplate{
		SiteID:      sid,
		Name:        name,
		Description: strings.TrimSpace(r.FormValue("aciklama")),
		Pages:       raw,
		IsDefault:   r.FormValue("varsayilan") == "1",
		CreatedBy:   &u.ID,
	}
	newID, err := repo.CreateGazeteTemplate(ctx, s.pool, t)
	if err != nil {
		s.flashRedirect(w, r, "/admin/gazete/sablonlar", "error", "Şablon kaydedilemedi.")
		return
	}
	s.audit(r, "gazete_sablon_olustur", name)
	if IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/admin/gazete/sablonlar")
		return
	}
	s.flashRedirect(w, r, "/admin/gazete/sablonlar", "info", "Şablon kaydedildi: "+name+" (#"+strconv.FormatInt(newID, 10)+")")
}

// adminBilesenlerShell galeri sayfalarının ortak kabuğu (admin_bilesenler) için
// kullanılır; ayrı bir define'a gerek yoktur.

// handleGazeteSablonTohum POST /admin/gazete/sablonlar/tohum
// Projeyle gelen hazir peyzajlari yukler. Isme gore esler: var olani
// degistirmez, yalniz eksikleri ekler.
func (s *Server) handleGazeteSablonTohum(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	n, err := s.gazeteSablonTohumla(r.Context(), s.curSiteID(r))
	if err != nil {
		s.logger.Error("gazete sablon tohumlama", "error", err)
		s.flashRedirect(w, r, "/admin/gazete/sablonlar", "error", "Sablonlar yuklenemedi: "+err.Error())
		return
	}
	mesaj := "Eksik sablon yoktu; hepsi zaten yuklu."
	if n > 0 {
		mesaj = fmt.Sprintf("%d hazir sablon yuklendi.", n)
	}
	// Sablonsuz eski sayilari, gorunumlerini koruyan A3 duzenine bagla.
	sid := s.curSiteID(r)
	if id := s.gazeteSablonAdaGore(r.Context(), sid, "A3 — Mevcut Düzen"); id != nil {
		if adet, err := s.gazeteEskiSayilariBagla(r.Context(), sid, *id); err != nil {
			s.logger.Error("gazete sayi baglama", "error", err)
		} else if adet > 0 {
			mesaj += fmt.Sprintf(" Sablonsuz %d sayi A3 duzenine baglandi.", adet)
		}
	}
	s.audit(r, "gazete_sablon_tohum", "")
	s.flashRedirect(w, r, "/admin/gazete/sablonlar", "info", mesaj)
}
