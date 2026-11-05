package httpapi

import (
	"fmt"
	"net/http"
	"strconv"

	"theblock/site/internal/webui/templkit"
)

// indexlemeViewFrom indeksleme verisi → templkit.IndexlemeView.
func indexlemeViewFrom(i *indexingData, fl *flashMsg) templkit.IndexlemeView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.IndexlemeView{FlashKind: kind, FlashText: text}
	if i == nil {
		return v
	}
	v.IndexNowReady = i.IndexNowKey != ""
	v.SAConfigured = i.SAConfigured
	v.Auto = i.Auto
	v.BlockedCount = fmt.Sprint(i.BlockedCount)
	v.KeyURL = i.KeyURL
	v.Endpoint = i.Endpoint
	v.Allowlist = i.Allowlist
	v.NewsURL = i.NewsSitemapURL
	if i.NewsReport != nil {
		v.NewsReport = &templkit.IdnNewsRapor{
			Durum:     i.NewsReport.Durum,
			Mesaj:     i.NewsReport.Mesaj,
			ToplamURL: fmt.Sprint(i.NewsReport.ToplamURL),
			EnEski:    fmt.Sprint(i.NewsReport.EnEski),
			EnYeni:    fmt.Sprint(i.NewsReport.EnYeni),
			EksikAlan: i.NewsReport.EksikAlan,
		}
	}
	for _, c := range i.Contents {
		v.Contents = append(v.Contents, templkit.IdnIcerik{
			ID: strconv.FormatInt(c.ID, 10), Title: c.Title,
		})
	}
	for _, l := range i.Logs {
		v.Logs = append(v.Logs, templkit.IdnLog{
			Engine: l.Engine, URL: l.URL, OK: l.Status == "ok",
			Detail: l.Detail, Time: fmt.Sprint(l.CreatedAt),
		})
	}
	v.TrapCount = fmt.Sprint(i.TrapCount)
	for _, t := range i.TrapHits {
		v.TrapHits = append(v.TrapHits, templkit.IdnTuzak{
			IP: t.IP, UserAgent: t.UserAgent, Path: t.Path, Time: fmt.Sprint(t.CreatedAt),
		})
	}
	for _, b := range i.Blocked {
		v.Blocked = append(v.Blocked, templkit.IdnBlok{
			IP: b.IP, Reason: b.Reason, Hits: fmt.Sprint(b.Hits), Time: fmt.Sprint(b.CreatedAt),
		})
	}
	return v
}

// tasimaViewFrom taşıma verisi → templkit.TasimaView.
func tasimaViewFrom(t *tasimaData, fl *flashMsg) templkit.TasimaView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.TasimaView{FlashKind: kind, FlashText: text}
	if t == nil {
		return v
	}
	v.Count = fmt.Sprint(t.Count)
	v.Query = t.Query
	for _, r := range t.Redirects {
		v.Redirects = append(v.Redirects, templkit.TasimaYonlendirme{
			SourcePath: r.SourcePath, TargetURL: r.TargetURL,
			StatusCode: fmt.Sprint(r.StatusCode), Hits: fmt.Sprint(r.Hits), Note: r.Note,
		})
	}
	for _, p := range t.Preview {
		v.Preview = append(v.Preview, templkit.TasimaOnizleme{
			Line: fmt.Sprint(p.Line), Title: p.Title, Category: p.Category,
			Status: p.Status, Tags: p.Tags, OK: p.OK, Err: p.Err,
		})
	}
	v.Apply = t.Apply
	v.Imported = fmt.Sprint(t.Imported)
	v.ImportErrs = t.ImportErrs
	v.RawCSV = t.RawCSV
	v.Categories = t.Categories
	return v
}

// ---- Render yardımcıları ----

func (s *Server) renderIndexlemePanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.IndexlemePanel(indexlemeViewFrom(d.Indexleme, d.Flash)))
}

func (s *Server) renderIndexlemePageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.IndexlemePanel(indexlemeViewFrom(d.Indexleme, d.Flash)))
}

func (s *Server) renderTasimaPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.TasimaPanel(tasimaViewFrom(d.Tasima, d.Flash)))
}

func (s *Server) renderTasimaPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.TasimaPanel(tasimaViewFrom(d.Tasima, d.Flash)))
}
