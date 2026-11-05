package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/webui/templkit"
)

// sinemalarViewFrom filmsData → templkit.SinemalarView eşlemesi.
func sinemalarViewFrom(f *filmsData, fl *flashMsg) templkit.SinemalarView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.SinemalarView{FlashKind: kind, FlashText: text}
	if f == nil {
		return v
	}
	v.Week = f.Week
	v.Weeks = f.Weeks
	v.Total = strconv.FormatInt(f.Total, 10)
	for _, it := range f.Items {
		v.Items = append(v.Items, templkit.FilmOge{
			ID:          strconv.FormatInt(it.ID, 10),
			Title:       it.Title,
			Cinema:      it.Cinema,
			Genre:       it.Genre,
			DurationMin: it.DurationMin,
			PosterURL:   it.PosterURL,
			Synopsis:    it.Synopsis,
		})
	}
	return v
}

// ilanlarViewFrom noticesData → templkit.IlanlarView eşlemesi.
func ilanlarViewFrom(n *noticesData, fl *flashMsg) templkit.IlanlarView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.IlanlarView{FlashKind: kind, FlashText: text}
	if n == nil {
		return v
	}
	v.Date = n.Date
	v.Dates = n.Dates
	v.Total = strconv.FormatInt(n.Total, 10)
	for _, it := range n.Items {
		v.Items = append(v.Items, templkit.IlanOge{
			ID:          strconv.FormatInt(it.ID, 10),
			Title:       it.Title,
			Institution: it.Institution,
			City:        it.City,
			NoticeDate:  it.NoticeDate,
			URL:         it.URL,
		})
	}
	return v
}

// eventWhenStr etkinlik zaman etiketi (Bugün / Yarın / X gün sonra / tarih).
func eventWhenStr(t time.Time) string {
	now := time.Now()
	a := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	b := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	gun := int(b.Sub(a).Hours() / 24)
	switch {
	case gun < 0:
		return t.Format("02.01.2006")
	case gun == 0:
		return "Bugün"
	case gun == 1:
		return "Yarın"
	default:
		return strconv.Itoa(gun) + " gün sonra"
	}
}

// etkinliklerViewFrom eventsData → templkit.EtkinliklerView eşlemesi.
func etkinliklerViewFrom(e *eventsData, fl *flashMsg) templkit.EtkinliklerView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.EtkinliklerView{FlashKind: kind, FlashText: text}
	if e == nil {
		return v
	}
	for _, it := range e.Events {
		rangeStr := it.StartAt.Format("2006-01-02 15:04")
		if it.EndAt != nil {
			rangeStr += " → " + it.EndAt.Format("2006-01-02 15:04")
		}
		endLocal := ""
		if it.EndAt != nil {
			endLocal = it.EndAt.Format("2006-01-02T15:04")
		}
		remind := make([]string, 0, len(it.RemindDays))
		for i, d := range it.RemindDays {
			if i > 0 {
				remind = append(remind, ","+strconv.Itoa(d))
			} else {
				remind = append(remind, strconv.Itoa(d))
			}
		}
		v.Items = append(v.Items, templkit.EtkinlikOge{
			ID:          strconv.FormatInt(it.ID, 10),
			Title:       it.Title,
			When:        eventWhenStr(it.StartAt),
			Status:      it.Status,
			Active:      it.Status == "aktif",
			Range:       rangeStr,
			Description: it.Description,
			EditTitle:   it.Title,
			StartLocal:  it.StartAt.Format("2006-01-02T15:04"),
			EndLocal:    endLocal,
			Location:    it.Location,
			URL:         it.URL,
			Remind:      strings.Join(remind, ""),
			CategoryID:  strconv.FormatInt(it.CategoryID, 10),
		})
	}
	for _, c := range e.Cats {
		v.Cats = append(v.Cats, templkit.EtkinlikCat{
			ID:   strconv.FormatInt(c.ID, 10),
			Name: c.Name,
		})
	}
	return v
}

// ---- Render yardımcıları (HTMX fragment + tam sayfa) ----

func (s *Server) renderSinemalarPanel(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := sinemalarViewFrom(d.FilmlerPanel, d.Flash)
	s.renderTemplPanel(w, r, templkit.SinemalarPanel(v))
}

func (s *Server) renderSinemalarPage(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := sinemalarViewFrom(d.FilmlerPanel, d.Flash)
	s.renderTemplPage(w, r, d, templkit.SinemalarPanel(v))
}

func (s *Server) renderIlanlarPanel(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := ilanlarViewFrom(d.IlanlarPanel, d.Flash)
	s.renderTemplPanel(w, r, templkit.IlanlarPanel(v))
}

func (s *Server) renderIlanlarPage(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := ilanlarViewFrom(d.IlanlarPanel, d.Flash)
	s.renderTemplPage(w, r, d, templkit.IlanlarPanel(v))
}

func (s *Server) renderEtkinliklerPanel(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := etkinliklerViewFrom(d.EtkinlikPanel, d.Flash)
	s.renderTemplPanel(w, r, templkit.EtkinliklerPanel(v))
}

func (s *Server) renderEtkinliklerPage(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := etkinliklerViewFrom(d.EtkinlikPanel, d.Flash)
	s.renderTemplPage(w, r, d, templkit.EtkinliklerPanel(v))
}
