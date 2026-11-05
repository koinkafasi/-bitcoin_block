package httpapi

import (
	"net/http"
	"strconv"

	"theblock/site/internal/webui/templkit"
)

// mansetViewFrom manşet editörü verisi.
func mansetViewFrom(m *mansetData, fl *flashMsg) templkit.MansetView {
	kind, text := flashTo(fl)
	v := templkit.MansetView{FlashKind: kind, FlashText: text}
	if m == nil {
		return v
	}
	v.Design = string(m.Design)
	if v.Design == "" {
		v.Design = "{}"
	}
	v.Presets = string(m.Presets)
	if v.Presets == "" {
		v.Presets = "[]"
	}
	v.BgURL = m.BgURL
	v.BgQuoted = strconv.Quote(m.BgURL)
	for _, md := range m.Media {
		mid := md.ID
		v.Media = append(v.Media, templkit.MansetMedya{
			ID:       strconv.FormatInt(mid, 10),
			Filename: md.Filename,
			Thumb:    "/media/thumb/" + strconv.FormatInt(mid, 10) + "?v=" + strconv.FormatInt(md.UpdatedAt.Unix(), 10),
		})
	}
	for _, d := range m.Designs {
		v.Designs = append(v.Designs, templkit.MansetSablon{
			ID:         strconv.FormatInt(d.ID, 10),
			Name:       d.Name,
			IsDefault:  d.IsDefault,
			DeleteVals: `{"id":` + strconv.FormatInt(d.ID, 10) + `}`,
		})
	}
	return v
}

func (s *Server) renderMansetPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.MansetPanel(mansetViewFrom(d.Manset, d.Flash)))
}

func (s *Server) renderMansetPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.MansetPanel(mansetViewFrom(d.Manset, d.Flash)))
}
