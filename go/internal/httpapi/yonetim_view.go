package httpapi

import (
	"fmt"
	"net/http"
	"strconv"

	"theblock/site/internal/webui/templkit"
)

// markaViewFrom brandEditData → templkit.MarkaView eşlemesi.
func (s *Server) markaViewFrom(d *tmplData) templkit.MarkaView {
	v := templkit.MarkaView{}
	if d.Flash != nil {
		v.FlashKind, v.FlashText = d.Flash.Kind, d.Flash.Text
	}
	if be := d.BrandEdit; be != nil && be.Config != nil {
		c := be.Config
		v.Mode = c.Mode
		v.Text = c.Text
		v.Tagline = c.Tagline
		if c.ImageID != nil {
			v.LogoURL = s.mediaURL(c.ImageID)
		}
		if c.FaviconID != nil {
			v.FaviconURL = s.mediaURL(c.FaviconID)
		}
		if c.OGImageID != nil {
			v.OGURL = s.mediaURL(c.OGImageID)
		}
		v.SiteBaslik = be.SiteBaslik
		v.Slogan = be.Slogan
	}
	if d.Gorunum != nil {
		v.PanelLogo = d.Gorunum.Logo
	}
	return v
}

// kumelerViewFrom kumelerData → templkit.KumelerView eşlemesi.
func kumelerViewFrom(k *kumelerData, fl *flashMsg) templkit.KumelerView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.KumelerView{FlashKind: kind, FlashText: text}
	if k == nil {
		return v
	}
	for _, c := range k.Clusters {
		v.Clusters = append(v.Clusters, templkit.KumeRow{
			ID: strconv.FormatInt(c.ID, 10), Name: c.Name, Count: c.ContentCount,
		})
	}
	if k.Active != nil {
		v.HasActive = true
		v.ActiveID = strconv.FormatInt(k.Active.ID, 10)
		v.ActiveName = k.Active.Name
		v.ActiveIntent = k.Active.Intent
	}
	for _, w := range k.Warnings {
		v.Warnings = append(v.Warnings, templkit.KumeUyari{
			A: strconv.FormatInt(w.A, 10), B: strconv.FormatInt(w.B, 10),
			TitleA: w.TitleA, TitleB: w.TitleB, Overlap: fmt.Sprint(w.Overlap),
		})
	}
	for _, p := range k.Pool {
		v.Pool = append(v.Pool, templkit.KumeItem{
			ID: strconv.FormatInt(p.ID, 10), Title: p.Title,
		})
	}
	for _, it := range k.Items {
		v.Items = append(v.Items, templkit.KumeItem{
			ID: strconv.FormatInt(it.ID, 10), Title: it.Title, Status: it.Status,
		})
	}
	return v
}

// adsenseViewFrom adsenseData → templkit.AdsenseView eşlemesi.
func adsenseViewFrom(a *adsenseData, fl *flashMsg) templkit.AdsenseView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.AdsenseView{FlashKind: kind, FlashText: text}
	if a == nil {
		return v
	}
	v.Connected = a.Connected
	v.AccountName = a.AccountName
	v.TokenExpiry = a.TokenExpiry
	v.SonSenkron = a.SonSenkron
	v.ClientID = a.ClientID
	v.GAProperty = a.GAProperty
	v.RedirectURI = a.RedirectURI
	return v
}

// ---- Render yardımcıları ----

func (s *Server) renderMarkaPanel(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.MarkaPanel(s.markaViewFrom(d)))
}

func (s *Server) renderMarkaPage(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.MarkaPanel(s.markaViewFrom(d)))
}

func (s *Server) renderKonuKumeleriPanel(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.KumelerPanel(kumelerViewFrom(d.Kumeler, d.Flash)))
}

func (s *Server) renderKonuKumeleriPage(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.KumelerPanel(kumelerViewFrom(d.Kumeler, d.Flash)))
}

func (s *Server) renderAdsenseMerkezPanel(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.AdSensePanel(adsenseViewFrom(d.Adsense, d.Flash)))
}

func (s *Server) renderAdsenseMerkezPage(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.AdSensePanel(adsenseViewFrom(d.Adsense, d.Flash)))
}
