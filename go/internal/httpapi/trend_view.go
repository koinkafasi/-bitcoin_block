package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"theblock/site/internal/webui/templkit"
)

// trendViewFrom bot/trend verisi → templkit.TrendView eşlemesi.
func trendViewFrom(b *botData, fl *flashMsg) templkit.TrendView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.TrendView{FlashKind: kind, FlashText: text}
	if b == nil {
		return v
	}
	if b.TCBrief != nil {
		v.Brief = &templkit.TrendBrief{Term: b.TCBrief.Term.Term, Text: b.TCBrief.Text}
	}
	v.Stats = []templkit.TrendBasit{
		{A: "Toplam", B: fmt.Sprint(b.TCStats.Total)},
		{A: "Kapsandı", B: fmt.Sprint(b.TCStats.Covered)},
		{A: "Zayıf", B: fmt.Sprint(b.TCStats.Weak)},
		{A: "Boşluk", B: fmt.Sprint(b.TCStats.Gap)},
		{A: "Yükselen", B: fmt.Sprint(b.TCStats.Rising)},
	}
	v.AlertCount = fmt.Sprint(b.TCStats.Alerts)
	v.TCQ = b.TCQ
	v.TCFilter = b.TCFilter
	if b.TCMinScore > 0 {
		v.TCMinScore = strconv.Itoa(b.TCMinScore)
	}
	for _, it := range b.TCItems {
		color := "#b91c1c"
		if it.Score >= 70 {
			color = "var(--ok)"
		} else if it.Score >= 40 {
			color = "var(--warn)"
		}
		traffic := "—"
		if it.ApproxTraffic > 0 {
			traffic = strconv.Itoa(it.ApproxTraffic) + "+"
		}
		src := it.SourceName
		if it.PictureSource != "" {
			src += " · " + it.PictureSource
		}
		v.Items = append(v.Items, templkit.TrendOge{
			ID:         strconv.FormatInt(it.ID, 10),
			Term:       it.Term,
			Source:     src,
			Traffic:    traffic,
			Score:      it.Score,
			ScoreColor: color,
			SparkPath:  it.SparkPath,
			SparkTitle: it.SparkTitle,
			SparkX:     fmt.Sprintf("%.1f", it.SparkX),
			SparkY:     fmt.Sprintf("%.1f", it.SparkY),
			HasSpark:   it.SparkPath != "",
			Coverage:   it.CoverageCount,
			FirstSeen:  fmtTimeVal(it.FirstSeenAt),
			LastSeen:   fmtTimeVal(it.LastSeenAt),
		})
	}
	for _, a := range b.TCAlerts {
		v.Alerts = append(v.Alerts, templkit.TrendUyari{Term: a.Term, Score: fmt.Sprint(a.Score)})
	}
	for _, s := range b.TCSources {
		status := "diger"
		switch s.StatusLabel() {
		case "sağlıklı":
			status = "saglikli"
		case "hata":
			status = "hata"
		case "pasif":
			status = "pasif"
		}
		v.Sources = append(v.Sources, templkit.TrendKaynak{
			ID:        strconv.FormatInt(s.ID, 10),
			Name:      s.Name,
			URL:       s.URL,
			Kind:      s.KindLabel(),
			LastFetch: s.LastFetchLabel(),
			Status:    status,
			LastError: s.LastError,
			Aktif:     s.Aktif,
		})
	}
	v.Weights = []templkit.TrendBasit{
		{B: fmt.Sprintf("%.0f", b.TCWeights.Traffic)},
		{B: fmt.Sprintf("%.0f", b.TCWeights.Freshness)},
		{B: fmt.Sprintf("%.0f", b.TCWeights.News)},
		{B: fmt.Sprintf("%.0f", b.TCWeights.Relevance)},
		{B: fmt.Sprintf("%.0f", b.TCWeights.Demand)},
		{B: fmt.Sprint(b.TCThreshold)},
	}
	v.Auto = b.TCAuto
	v.Notify = b.TCNotify
	v.Paid = b.TCPaid
	v.GSCProperty = b.GSCProperty
	v.GSCReady = b.GSCConfigured
	for _, g := range b.GSCTop {
		v.GSCTop = append(v.GSCTop, templkit.TrendBasit{
			A: g.Query,
			B: fmt.Sprintf("%d tık · %d gösterim", g.Clicks, g.Impressions),
		})
	}
	for _, g := range b.SearchTop {
		v.SearchTop = append(v.SearchTop, templkit.TrendBasit{A: g.Query, B: fmt.Sprint(g.Count)})
	}
	v.GA4Property = b.GA4Property
	v.GA4Olcum = b.GA4Olcum
	v.GA4Ready = b.GA4Configured
	if b.GA4 != nil {
		v.GA4Aktif = fmt.Sprint(b.GA4.Aktif)
		v.GA4Time = b.GA4.Guncel
		for _, p := range b.GA4.Sayfalar {
			v.GA4Pages = append(v.GA4Pages, templkit.TrendBasit{A: p.Sayfa, B: fmt.Sprint(p.Aktif)})
		}
	}
	for _, f := range b.Feeds {
		v.Feeds = append(v.Feeds, templkit.TrendFeed{
			ID: strconv.FormatInt(f.ID, 10), Name: f.Name, URL: f.URL, Aktif: f.Aktif,
		})
	}
	for _, c := range b.Cats {
		v.Cats = append(v.Cats, templkit.TrendBasit{A: strconv.FormatInt(c.ID, 10), B: c.Name})
	}
	v.RulesTxt = b.RulesTxt
	v.BotAuto = b.Auto
	v.BotDakika = fmt.Sprint(b.Dakika)
	// Faz 188: otomatik SEO ayarları bot panelinden yönetilir.
	v.SeoOtomatik = b.SeoOtomatik
	v.SeoDakika = fmt.Sprint(b.SeoDakika)

	// Faz 194 — otomatik AI ile yazım.
	v.YazimOtomatik = b.YazimOtomatik
	v.YazimDakika = fmt.Sprint(b.YazimDakika)
	v.YazimDurum = b.YazimDurum
	v.CeviriOtomatik = b.CeviriOtomatik
	v.CeviriDiller = b.CeviriDiller
	v.CeviriDurum = b.CeviriDurum
	v.CeviriDakika = fmt.Sprint(b.CeviriDakika)
	v.CeviriBekleyen = b.CeviriBekleyen
	for _, r := range b.Recent {
		v.Recent = append(v.Recent, templkit.TrendBasit{
			A: strconv.FormatInt(r.ID, 10),
			B: r.Title,
			C: fmtTimePtr(&r.UpdatedAt),
		})
	}
	v.TrendTxt = b.TrendTxt
	v.TrendDakika = fmt.Sprint(b.TrendDakika)
	for _, t := range b.Trends {
		v.Trends = append(v.Trends, templkit.TrendGundem{
			ID: strconv.FormatInt(t.ID, 10), Source: t.Source, Title: t.Title, URL: t.URL,
		})
	}
	return v
}

// fmtTimePtr *time.Time biçimlendirir (02.01.2006 15:04).
func fmtTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "—"
	}
	return t.Local().Format("02.01.2006 15:04")
}

// fmtTimeVal time.Time biçimlendirir (02.01.2006 15:04).
func fmtTimeVal(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("02.01.2006 15:04")
}

// renderTrendPanelV2 bot/trend panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderTrendPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.TrendPanel(trendViewFrom(d.Bot, d.Flash)))
}

// renderTrendPageV2 bot/trend panelini tam sayfa (admin kabuğu) yazar.
func (s *Server) renderTrendPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.TrendPanel(trendViewFrom(d.Bot, d.Flash)))
}
