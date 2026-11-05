package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"theblock/site/internal/webui/templkit"
)

// optDateFmt *time.Time biçimlendirir (nil → "—").
func optDateFmt(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format("2006-01-02")
}

// reklamlarViewFrom adsData → templkit.ReklamlarView eşlemesi.
func reklamlarViewFrom(a *adsData, fl *flashMsg) templkit.ReklamlarView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.ReklamlarView{FlashKind: kind, FlashText: text}
	if a == nil {
		return v
	}
	v.Stats = templkit.ReklamStat{
		Views:  fmt.Sprint(a.Stats.Views),
		Clicks: fmt.Sprint(a.Stats.Clicks),
		CTR:    fmt.Sprintf("%.2f", a.Stats.CTR),
		EstTL:  fmt.Sprintf("%.2f", a.Stats.EstTL),
	}
	for _, ad := range a.Ads {
		v.Ads = append(v.Ads, templkit.ReklamOge{
			ID:        strconv.FormatInt(ad.ID, 10),
			Kind:      ad.Kind,
			Name:      ad.Name,
			SlotKey:   ad.SlotKey,
			Status:    ad.Status,
			Created:   ad.CreatedAt.Format("2006-01-02"),
			ImageURL:  ad.ImageURL,
			TargetURL: ad.TargetURL,
			StartsAt:  optDateFmt(ad.StartsAt),
			EndsAt:    optDateFmt(ad.EndsAt),
			Weight:    fmt.Sprint(ad.Weight),
			HTML:      ad.HTML,
			HTMLLen:   strconv.Itoa(len(ad.HTML)),
			ImageID:   fmt.Sprint(ad.ImageMediaID),
			Active:    ad.Status == "aktif",
		})
	}
	for _, s := range a.Slots {
		v.Slots = append(v.Slots, templkit.ReklamSlot{
			ID:     strconv.FormatInt(s.ID, 10),
			Key:    s.Key,
			Label:  s.Label,
			Width:  fmt.Sprint(s.Width),
			Height: fmt.Sprint(s.Height),
			Sticky: s.Sticky,
			Status: s.Status,
		})
	}
	return v
}

// renderReklamlarPanelV2 reklamlar panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderReklamlarPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.ReklamlarPanel(reklamlarViewFrom(d.Ads, d.Flash)))
}

// renderReklamlarPageV2 reklamlar panelini tam sayfa (admin kabuğu) yazar.
func (s *Server) renderReklamlarPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.ReklamlarPanel(reklamlarViewFrom(d.Ads, d.Flash)))
}
