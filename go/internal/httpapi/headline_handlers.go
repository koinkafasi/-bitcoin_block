package httpapi

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// mansetData Görsel Manşet Editörü sayfa verisidir.
type mansetData struct {
	Designs []repo.HeadlineDesign
	Design  template.JS
	Presets template.JS
	Media   []repo.Media
	BgURL   string
}

const headlinePresets = `[
{"name":"1 · Klasik Sol","design":{"version":3,"canvas":{"w":1200,"h":675},"background":{"color":"#0b0e14","overlay":{"active":true,"mode":"gradient","color":"#000000","direction":"left","spread":70,"opacity":88}},"badge":{"active":true,"text":"SON DAKİKA","color":"#e11d48","textColor":"#ffffff","fontSize":24,"x":60,"y":60,"radius":8,"padding":12,"uppercase":true},"layers":[{"id":"ust","role":"ust_baslik","text":"GÜNDEM","x":80,"y":320,"fontSize":26,"weight":700,"color":"#FBBF24","align":"left","lineHeight":1.2,"maxWidth":0,"uppercase":true,"shadow":true,"shadowColor":"#000000","visible":true,"locked":false},{"id":"ana","role":"ana_baslik","text":"Haber başlığınız buraya","x":80,"y":372,"fontSize":58,"weight":800,"color":"#FFFFFF","align":"left","lineHeight":1.08,"maxWidth":760,"uppercase":false,"shadow":true,"shadowColor":"#000000","visible":true,"locked":false},{"id":"alt","role":"alt_baslik","text":"Haber özeti buraya…","x":80,"y":510,"fontSize":23,"weight":400,"color":"#E5E7EB","align":"left","lineHeight":1.3,"maxWidth":700,"uppercase":false,"shadow":false,"shadowColor":"#000000","visible":true,"locked":false}]}},
{"name":"2 · Alt Bant","design":{"version":3,"canvas":{"w":1200,"h":675},"background":{"color":"#151515","overlay":{"active":true,"mode":"gradient","color":"#000000","direction":"bottom","spread":80,"opacity":92}},"badge":{"active":true,"text":"ÖZEL","color":"#f59e0b","textColor":"#111111","fontSize":22,"x":70,"y":430,"radius":4,"padding":12,"uppercase":true},"layers":[{"id":"ust","role":"ust_baslik","text":"DOSYA","x":80,"y":470,"fontSize":24,"weight":700,"color":"#FCD34D","align":"left","lineHeight":1.2,"maxWidth":0,"uppercase":true,"shadow":true,"shadowColor":"#000000","visible":true,"locked":false},{"id":"ana","role":"ana_baslik","text":"Haber başlığınız buraya","x":80,"y":505,"fontSize":52,"weight":800,"color":"#FFFFFF","align":"left","lineHeight":1.1,"maxWidth":850,"uppercase":false,"shadow":true,"shadowColor":"#000000","visible":true,"locked":false},{"id":"alt","role":"alt_baslik","text":"","x":80,"y":600,"fontSize":20,"weight":400,"color":"#E5E7EB","align":"left","lineHeight":1.3,"maxWidth":820,"uppercase":false,"shadow":false,"shadowColor":"#000000","visible":false,"locked":false}]}},
{"name":"3 · Merkez","design":{"version":3,"canvas":{"w":1200,"h":675},"background":{"color":"#111827","overlay":{"active":true,"mode":"solid","color":"#000000","direction":"all","spread":60,"opacity":48}},"badge":{"active":true,"text":"MANŞET","color":"#2563eb","textColor":"#ffffff","fontSize":22,"x":540,"y":185,"radius":999,"padding":14,"uppercase":true},"layers":[{"id":"ust","role":"ust_baslik","text":"","x":600,"y":255,"fontSize":24,"weight":700,"color":"#93C5FD","align":"center","lineHeight":1.2,"maxWidth":0,"uppercase":true,"shadow":false,"shadowColor":"#000000","visible":false,"locked":false},{"id":"ana","role":"ana_baslik","text":"Haber başlığınız buraya","x":600,"y":290,"fontSize":56,"weight":800,"color":"#FFFFFF","align":"center","lineHeight":1.08,"maxWidth":900,"uppercase":false,"shadow":true,"shadowColor":"#000000","visible":true,"locked":false},{"id":"alt","role":"alt_baslik","text":"Haber özeti buraya…","x":600,"y":440,"fontSize":22,"weight":400,"color":"#E5E7EB","align":"center","lineHeight":1.3,"maxWidth":760,"uppercase":false,"shadow":false,"shadowColor":"#000000","visible":true,"locked":false}]}},
{"name":"4 · Minimal","design":{"version":3,"canvas":{"w":1200,"h":675},"background":{"color":"#0f172a","overlay":{"active":true,"mode":"gradient","color":"#020617","direction":"bottom","spread":72,"opacity":76}},"badge":{"active":false,"text":"","color":"#e11d48","textColor":"#ffffff","fontSize":22,"x":60,"y":60,"radius":8,"padding":12,"uppercase":true},"layers":[{"id":"ust","role":"ust_baslik","text":"","x":80,"y":430,"fontSize":24,"weight":600,"color":"#CBD5E1","align":"left","lineHeight":1.2,"maxWidth":0,"uppercase":true,"shadow":false,"shadowColor":"#000000","visible":false,"locked":false},{"id":"ana","role":"ana_baslik","text":"Haber başlığınız buraya","x":80,"y":500,"fontSize":50,"weight":700,"color":"#FFFFFF","align":"left","lineHeight":1.08,"maxWidth":850,"uppercase":false,"shadow":false,"shadowColor":"#000000","visible":true,"locked":false},{"id":"alt","role":"alt_baslik","text":"","x":80,"y":600,"fontSize":20,"weight":400,"color":"#CBD5E1","align":"left","lineHeight":1.3,"maxWidth":800,"uppercase":false,"shadow":false,"shadowColor":"#000000","visible":false,"locked":false}]}},
{"name":"5 · Lacivert","design":{"version":3,"canvas":{"w":1200,"h":675},"background":{"color":"#0f172a","overlay":{"active":true,"mode":"solid","color":"#0b1e3a","direction":"all","spread":60,"opacity":45}},"badge":{"active":true,"text":"ANALİZ","color":"#38bdf8","textColor":"#082f49","fontSize":22,"x":70,"y":70,"radius":6,"padding":12,"uppercase":true},"layers":[{"id":"ust","role":"ust_baslik","text":"EKONOMİ","x":80,"y":300,"fontSize":24,"weight":700,"color":"#7DD3FC","align":"left","lineHeight":1.2,"maxWidth":0,"uppercase":true,"shadow":false,"shadowColor":"#000000","visible":true,"locked":false},{"id":"ana","role":"ana_baslik","text":"Haber başlığınız buraya","x":80,"y":350,"fontSize":56,"weight":800,"color":"#F8FAFC","align":"left","lineHeight":1.08,"maxWidth":820,"uppercase":false,"shadow":true,"shadowColor":"#020617","visible":true,"locked":false},{"id":"alt","role":"alt_baslik","text":"Haber özeti buraya…","x":80,"y":500,"fontSize":22,"weight":400,"color":"#CBD5E1","align":"left","lineHeight":1.3,"maxWidth":720,"uppercase":false,"shadow":false,"shadowColor":"#000000","visible":true,"locked":false}]}},
{"name":"6 · Sıcak","design":{"version":3,"canvas":{"w":1200,"h":675},"background":{"color":"#2a1208","overlay":{"active":true,"mode":"gradient","color":"#3b1204","direction":"left","spread":75,"opacity":78}},"badge":{"active":true,"text":"SON DAKİKA","color":"#fbbf24","textColor":"#1c1917","fontSize":23,"x":60,"y":65,"radius":3,"padding":12,"uppercase":true},"layers":[{"id":"ust","role":"ust_baslik","text":"SICAK GELİŞME","x":80,"y":320,"fontSize":25,"weight":700,"color":"#FDE68A","align":"left","lineHeight":1.2,"maxWidth":0,"uppercase":true,"shadow":true,"shadowColor":"#1c1917","visible":true,"locked":false},{"id":"ana","role":"ana_baslik","text":"Haber başlığınız buraya","x":80,"y":372,"fontSize":56,"weight":800,"color":"#FFFBEB","align":"left","lineHeight":1.08,"maxWidth":800,"uppercase":false,"shadow":true,"shadowColor":"#1c1917","visible":true,"locked":false},{"id":"alt","role":"alt_baslik","text":"Haber özeti buraya…","x":80,"y":510,"fontSize":22,"weight":400,"color":"#FED7AA","align":"left","lineHeight":1.3,"maxWidth":720,"uppercase":false,"shadow":false,"shadowColor":"#000000","visible":true,"locked":false}]}}
]`

// skeleton varsayılan (boş) tasarım belgesi.
const skeleton = `{"version":2,"canvas":{"w":1200,"h":675},"background":{"color":"#0b0e14","overlay":{"active":true,"mode":"gradient","color":"#000000","direction":"left","spread":70,"opacity":85}},"badge":{"active":true,"text":"SON DAKİKA","color":"#e11d48","textColor":"#ffffff","fontSize":24,"x":60,"y":60,"radius":8,"padding":12,"uppercase":true},"layers":[{"id":"ust","role":"ust_baslik","text":"SON DAKİKA","x":80,"y":330,"fontSize":28,"weight":700,"color":"#FBBF24","align":"left","lineHeight":1.2,"maxWidth":0,"uppercase":true,"shadow":true,"shadowColor":"#000000","visible":true,"locked":false},{"id":"ana","role":"ana_baslik","text":"Haber başlığınız buraya","x":80,"y":385,"fontSize":56,"weight":800,"color":"#FFFFFF","align":"left","lineHeight":1.1,"maxWidth":780,"uppercase":false,"shadow":true,"shadowColor":"#000000","visible":true,"locked":false},{"id":"alt","role":"alt_baslik","text":"Haber özeti buraya…","x":80,"y":520,"fontSize":24,"weight":400,"color":"#E5E7EB","align":"left","lineHeight":1.3,"maxWidth":780,"uppercase":false,"shadow":false,"shadowColor":"#000000","visible":true,"locked":false}]}`

// handleMansetGet editör sayfası.
func (s *Server) handleMansetGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildMansetData(r)
	if IsHTMX(r) {
		s.renderMansetPanelV2(w, r, d)
		return
	}
	s.renderMansetPageV2(w, r, d)
}

func (s *Server) buildMansetData(r *http.Request) *tmplData {
	ctx := r.Context()
	u := userFromContext(ctx)

	d := s.baseView("tr", "/admin", "Manşet Editörü")
	d.User = ptr(u)
	d.Flash = s.consumeFlash(r)

	md := &mansetData{}
	md.Design = template.JS(skeleton)
	md.Presets = template.JS(headlinePresets)
	md.BgURL = "/media/none.svg"

	designs, err := repo.ListHeadlineDesigns(ctx, s.pool)
	if err == nil {
		md.Designs = designs
		for _, h := range designs {
			if h.IsDefault {
				md.Design = template.JS(platform.NormalizeDesign(h.Design))
				// bg url
				var req platform.DesignReq
				if json.Unmarshal(platform.NormalizeDesign(h.Design), &req) == nil && req.Background.MediaID != nil {
					md.BgURL = s.mediaURL(req.Background.MediaID)
				}
				break
			}
		}
	}
	md.Media, _ = repo.ListMedia(ctx, s.pool, s.curSiteID(r), u.ID, 200)

	d.Manset = md
	return d
}

func (s *Server) renderMansetPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildMansetData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderMansetPanelV2(w, r, d)
}

// handleMansetGetTemplate tasarımı JSON döndürür.
func (s *Server) handleMansetGetTemplate(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	h, err := repo.GetHeadlineDesign(r.Context(), s.pool, id)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": h.ID, "name": h.Name, "is_default": h.IsDefault, "design": json.RawMessage(platform.NormalizeDesign(h.Design))})
}

// handleMansetSave tasarım şablon kaydı (oluştur/güncelle).
func (s *Server) handleMansetSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID        int64           `json:"id"`
		Name      string          `json:"name"`
		IsDefault bool            `json:"is_default"`
		Design    json.RawMessage `json:"design"`
	}
	if IsHTMX(r) {
		_ = r.ParseForm()
		req.ID = parseID(r.FormValue("id"))
		req.Name = r.FormValue("name")
		req.IsDefault = r.FormValue("is_default") != ""
		req.Design = json.RawMessage(r.FormValue("design"))
		if len(req.Design) == 0 {
			s.renderMansetPanel(w, r, "error", "Tasarım verisi boş.")
			return
		}
	} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz istek: " + err.Error()})
		return
	}
	u := userFromContext(r.Context())
	if req.Name == "" {
		req.Name = "Manşet " + rndSuffix()
	}
	design := platform.NormalizeDesign(req.Design)
	var id int64
	var err error
	if req.ID != 0 {
		err = repo.UpdateHeadlineDesign(r.Context(), s.pool, req.ID, req.Name, design, req.IsDefault)
		id = req.ID
	} else {
		id, err = repo.CreateHeadlineDesign(r.Context(), s.pool, req.Name, design, req.IsDefault, nil, u.ID)
	}
	if err != nil {
		if IsHTMX(r) {
			s.renderMansetPanel(w, r, "error", err.Error())
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.audit(r, "manset_kaydet", req.Name)
	if IsHTMX(r) {
		s.renderMansetPanel(w, r, "info", "Manşet şablonu kaydedildi: "+req.Name)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "name": req.Name})
}

// handleMansetDelete tasarımı siler.
func (s *Server) handleMansetDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if IsHTMX(r) {
		_ = r.ParseForm()
		req.ID = parseID(r.FormValue("id"))
	} else {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if err := repo.DeleteHeadlineDesign(r.Context(), s.pool, req.ID); err != nil {
		if IsHTMX(r) {
			s.renderMansetPanel(w, r, "error", "Şablon silinemedi.")
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false})
		return
	}
	if IsHTMX(r) {
		s.renderMansetPanel(w, r, "info", "Şablon silindi.")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleMansetExport tasarımı sunucuda render edip medyaya kaydeder.
func (s *Server) handleMansetExport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Design json.RawMessage `json:"design"`
		Fmt    string          `json:"fmt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Design) == 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "tasarım yok"})
		return
	}
	raw, err := platform.RenderDesign(platform.NormalizeDesign(req.Design), s.mediaRaw)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ext, mimeType := "jpg", "image/jpeg"
	format := "jpeg"
	if req.Fmt == "webp" {
		if w, werr := platform.WebpFromJpeg(raw, 85); werr == nil {
			raw, ext, mimeType, format = w, "webp", "image/webp", "webp"
		}
	}
	stored, err := platform.SaveMedia(raw, "manset."+ext)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "depolama hatası"})
		return
	}
	u := userFromContext(r.Context())
	sz := int64(len(raw))
	mid, err := repo.CreateMedia(r.Context(), s.pool, &repo.Media{
		UserID: u.ID, Kind: "image", Filename: "manset-render." + ext, Path: stored,
		Mime: mimeType, SizeBytes: &sz, Source: "render",
	})
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "kayıt hatası"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "media_id": mid, "url": publicMediaURL(&stored), "format": format})
}

// handleMansetMediaURL medya id'den URL döner (bg seçici için).
func (s *Server) handleMansetMediaURL(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	path := s.mediaPath(r.Context(), id)
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": path != "", "url": "/media/" + path})
}

// mediaRaw medya id'den ham görsel baytları (render için).
func (s *Server) mediaRaw(id int64) ([]byte, error) {
	p := s.mediaPath(context.Background(), id)
	if p == "" {
		return nil, repo.ErrNotFound
	}
	return platform.ReadMedia(p)
}
