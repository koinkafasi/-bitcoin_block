package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ---- Gazete editörü (Faz 2) — Canva benzeri DOM tabanlı düzenleyici ----

// handleGazeteDuzen GET /admin/gazete/sayilar/{id}/duzenle — editör sayfası.
func (s *Server) handleGazeteDuzen(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionView) {
		return
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	id := parseID(r.PathValue("id"))
	iss, err := repo.GetGazeteIssue(ctx, s.pool, id)
	if err != nil {
		s.flashRedirect(w, r, "/admin/gazete", "error", "Sayı bulunamadı.")
		return
	}
	d := s.baseView("tr", "/admin", "Gazete Düzenle")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	v := &gazeteViewData{
		Active:   "sayilar",
		SiteID:   sid,
		Role:     s.gazeteRole(ctx, sid),
		Licensed: s.gazeteLicensed(ctx, sid),
		Issue:    &iss,
		Perm: map[string]bool{
			"goruntule": s.gazeteCan(ctx, sid, gazeteActionView),
			"duzenle":   s.gazeteCan(ctx, sid, gazeteActionEdit),
			"yayinla":   s.gazeteCan(ctx, sid, gazeteActionPublish),
		},
	}
	pages, _ := repo.ListGazetePages(ctx, s.pool, id)
	v.Pages = pages
	tpls, _ := repo.ListGazeteTemplates(ctx, s.pool, sid)
	v.Templates = tpls
	team, _ := repo.ListGazeteTeam(ctx, s.pool, sid)
	v.Team = team

	// Editör başlatma verisi: {issue, sayfalar:[{no,ad,bloklar}], sablonlar}
	type edPage struct {
		No       int                        `json:"no"`
		Ad       string                     `json:"ad"`
		Blocks   json.RawMessage            `json:"bloklar"`
		BlokHTML map[string]gazeteBlokGovde `json:"blokHtml"`
	}
	belge := s.gazeteSayiBelgesi(ctx, iss)
	boyut := gazeteBoyut(belge.Sayfa.Boyut)
	boot := struct {
		IssueID   int64           `json:"issueId"`
		IssueDate string          `json:"issueDate"`
		Title     string          `json:"title"`
		Status    string          `json:"status"`
		Pages     []edPage        `json:"sayfalar"`
		Templates []edTpl         `json:"sablonlar"`
		Sayfa     gazeteSayfaAyar `json:"sayfa"`
		Izgara    gazeteIzgara    `json:"izgara"`
		BoyutW    string          `json:"sayfaW"`
		BoyutH    string          `json:"sayfaH"`
		Kenar     string          `json:"kenar"`
		TipTab    string          `json:"tipTab"`
		// Surum: iyimser kilit damgasi. Kayit bunu geri gonderir; arada
		// baskasi kaydettiyse sunucu cakisma bildirir (SPEC K6).
		Surum string `json:"surum"`
	}{
		IssueID: iss.ID, IssueDate: iss.IssueDate, Title: iss.Title, Status: iss.Status,
		Sayfa: belge.Sayfa, Izgara: belge.Izgara,
		BoyutW: boyut.W, BoyutH: boyut.H, Kenar: boyut.Kenar, TipTab: boyut.TipTab,
		Surum: gazeteSurum(iss.UpdatedAt),
	}
	for _, p := range pages {
		b := p.Blocks
		if len(b) == 0 {
			b = []byte("[]")
		}
		boot.Pages = append(boot.Pages, edPage{
			No: p.PageNo, Ad: p.PageKey, Blocks: b, BlokHTML: gazeteBlokHTMLHaritasi(b),
		})
	}
	if len(boot.Pages) == 0 {
		for _, doc := range defaultGazetePages() {
			raw, _ := json.Marshal(doc.Blocks)
			boot.Pages = append(boot.Pages, edPage{
				No: doc.No, Ad: doc.Ad, Blocks: raw, BlokHTML: gazeteBlokHTMLHaritasi(raw),
			})
		}
	}
	for _, t := range tpls {
		boot.Templates = append(boot.Templates, edTpl{ID: t.ID, Name: t.Name})
	}
	raw, _ := json.Marshal(boot)
	// < karakterini kaçır (script etiketi kapanış sızıntısına karşı).
	raw = bytes.ReplaceAll(raw, []byte("<"), []byte("\\u003c"))
	v.PageJSON = template.JS(raw)
	d.Gazete = v
	kok := fmt.Sprintf(
		`<style>#gz-canvas{--gz-sayfa-w:%s;--gz-sayfa-h:%s;--gz-kenar:%s;`+
			`--gz-tip-tab:%s;--gz-tip-olcek:%g;--gz-yazi:%s;`+
			`--gz-tip:calc(var(--gz-tip-tab) * var(--gz-tip-olcek));`+
			// Kesim ve tasma payi kilavuzlari sayfa genisligine gore yuzde.
			`--gz-kesim:%.3f%%;--gz-tasma:%.3f%%;}</style>`,
		boyut.W, boyut.H, boyut.Kenar, boyut.TipTab, belge.Sayfa.TipOlcek,
		gazeteYaziTipiCSS(belge.Sayfa.YaziTipi),
		gazeteMmYuzde(belge.Sayfa.KesimPayi, boyut.W),
		gazeteMmYuzde(belge.Sayfa.TasmaPayi, boyut.W))
	d.PageHead = template.HTML(
		`<link rel="stylesheet" href="` + s.assetURL("gazete-baski.css") + `">` + kok +
			`<link rel="stylesheet" href="` + s.assetURL("gazete.css") + `">` +
			`<script src="` + s.assetURL("gazete-editor.js") + `" defer></script>`)
	s.renderTemplPage(w, r, d, templkit.GazeteEditorPanel(templkit.GazeteEditorView{
		Title: v.Issue.Title, IssueDate: v.Issue.IssueDate, Period: v.Issue.Period,
		Status: v.Issue.Status, PageJSON: string(v.PageJSON),
	}))
}

// edTpl şablon seçim listesi.
type edTpl struct {
	ID   int64  `json:"id"`
	Name string `json:"ad"`
}

type gazeteBlokGovde struct {
	HTML  string `json:"html"`
	Sinif string `json:"sinif"`
	Stil  string `json:"stil"`
}

// gazeteBlokGovdesi tek blogun editor gorunumunu uretir.
func gazeteBlokGovdesi(b gazeteBlock) gazeteBlokGovde {
	return gazeteBlokGovde{
		HTML:  gazeteBlockInnerHTML(b),
		Sinif: strings.TrimSpace(gazeteStilSinif(b.Stil)),
		Stil:  gazeteStilDeger(b.Stil),
	}
}

// gazeteBlokHTMLHaritasi blok JSON'undan {blok kimligi: govde} uretir.
func gazeteBlokHTMLHaritasi(raw json.RawMessage) map[string]gazeteBlokGovde {
	var bloklar []gazeteBlock
	if err := json.Unmarshal(raw, &bloklar); err != nil {
		return nil
	}
	out := make(map[string]gazeteBlokGovde, len(bloklar))
	for _, b := range bloklar {
		if b.ID == "" {
			continue
		}
		out[b.ID] = gazeteBlokGovdesi(b)
	}
	return out
}

func (s *Server) handleGazeteBlokHTML(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	var bloklar []gazeteBlock
	if err := json.Unmarshal([]byte(r.FormValue("bloklar")), &bloklar); err != nil {
		http.Error(w, "gecersiz blok JSON'u", http.StatusBadRequest)
		return
	}
	type blokCevap struct {
		gazeteBlokGovde
		Icerik *gazeteIcerik `json:"icerik,omitempty"`
	}
	out := make(map[string]blokCevap, len(bloklar))
	for i := range bloklar {
		b := &bloklar[i]
		if b.ID == "" {
			continue
		}
		if b.Kaynak != nil {
			b.Icerik = &gazeteIcerik{Haberler: s.gazeteArticlesFor(ctx, sid, b.Kaynak)}
		}
		out[b.ID] = blokCevap{gazeteBlokGovde: gazeteBlokGovdesi(*b), Icerik: b.Icerik}
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGazeteDogrula(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	id := parseID(r.PathValue("id"))
	tasmalar, err := s.gazeteDogrula(r.Context(), s.curSiteID(r), id)
	if err != nil {
		s.logger.Error("gazete dogrulama", "error", err)
		s.writeJSON(w, http.StatusOK, map[string]any{
			"olculdu": false,
			"hata":    err.Error(),
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"olculdu":  true,
		"tasmalar": tasmalar,
		"ozet":     gazeteTasmaOzeti(tasmalar),
	})
}
