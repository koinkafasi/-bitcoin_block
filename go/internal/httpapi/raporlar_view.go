package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// anketlerViewFrom anket paneli verisi → templkit.AnketlerView.
func anketlerViewFrom(items []repo.PollItem, fl *flashMsg) templkit.AnketlerView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.AnketlerView{FlashKind: kind, FlashText: text}
	for _, it := range items {
		og := templkit.AnketOge{
			ID:     strconv.FormatInt(it.Content.ID, 10),
			Title:  it.Content.Title,
			Status: it.Content.Status,
			Active: it.Content.Status == "yayinda",
			Total:  fmt.Sprint(it.Poll.Total()),
		}
		for _, o := range it.Poll.Options {
			og.Options = append(og.Options, templkit.AnketSecenek{
				ID: fmt.Sprint(o.ID), Text: o.Text, Votes: fmt.Sprint(o.Votes),
			})
		}
		v.Items = append(v.Items, og)
	}
	return v
}

// raporlarViewFrom reportsData → templkit.RaporlarView.
func raporlarViewFrom(rp *reportsData, fl *flashMsg) templkit.RaporlarView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.RaporlarView{FlashKind: kind, FlashText: text}
	if rp == nil {
		return v
	}
	for _, h := range rp.MostRead {
		v.MostRead = append(v.MostRead, templkit.RapIcerik{
			ID: strconv.FormatInt(h.ID, 10), Title: h.Title, Hits: fmt.Sprint(h.Hits),
		})
	}
	for _, h := range rp.MostCommented {
		v.MostCommented = append(v.MostCommented, templkit.RapIcerik{
			ID: strconv.FormatInt(h.ID, 10), Title: h.Title, Hits: fmt.Sprint(h.Hits),
		})
	}
	for _, u := range rp.AIUsage {
		v.AIUsage = append(v.AIUsage, templkit.RapSatir{Label: u.Label, Value: fmt.Sprint(u.Value)})
	}
	v.RPMDefault = fmt.Sprintf("%.2f", rp.RPMDefault)
	for _, r := range rp.Revenue {
		v.Revenue = append(v.Revenue, templkit.RapIcerik{
			ID:       strconv.FormatInt(r.ID, 10),
			Title:    r.Title,
			Hits:     fmt.Sprint(r.Hits),
			Revenue:  fmt.Sprintf("%.2f", r.Revenue),
			Cost:     fmt.Sprintf("%.2f", r.Cost),
			Profit:   fmt.Sprintf("%.2f", r.Profit),
			Positive: r.Profit >= 0,
		})
	}
	v.YorumPending = strconv.Itoa(rp.Yorum.Pending)
	v.YorumOnayli = strconv.Itoa(rp.Yorum.Approved)
	v.YorumToplam = strconv.Itoa(rp.Yorum.Total)
	v.BotTrapTotal = strconv.FormatInt(rp.BotTrapTotal, 10)
	v.BotTrap24 = strconv.FormatInt(rp.BotTrap24, 10)
	v.BlockedIPs = strconv.FormatInt(rp.BlockedIPs, 10)
	// Grafik JSON'u (orijinal: template.JS) — güvenli JSON dizisi olarak.
	chart := string(rp.MostJSON)
	if !json.Valid([]byte(chart)) {
		chart = "[]"
	}
	v.ChartJSON = chart
	return v
}

// renderAnketlerPanelV2 anketler panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderAnketlerPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.AnketlerPanel(anketlerViewFrom(pollItemsFrom(d), d.Flash)))
}

// renderAnketlerPageV2 anketler panelini tam sayfa (admin kabuğu) yazar.
func (s *Server) renderAnketlerPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.AnketlerPanel(anketlerViewFrom(pollItemsFrom(d), d.Flash)))
}

// pollItemsFrom tmplData içinden anket kayıtlarını alır.
func pollItemsFrom(d *tmplData) []repo.PollItem {
	if d.AnketPanel == nil {
		return nil
	}
	return d.AnketPanel.Items
}

// renderRaporlarPanelV2 raporlar panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderRaporlarPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.RaporlarPanel(raporlarViewFrom(d.Reports, d.Flash)))
}

// renderRaporlarPageV2 raporlar panelini tam sayfa (admin kabuğu) yazar.
func (s *Server) renderRaporlarPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.RaporlarPanel(raporlarViewFrom(d.Reports, d.Flash)))
}
