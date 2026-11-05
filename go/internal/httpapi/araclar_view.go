package httpapi

import (
	"fmt"
	"net/http"
	"strconv"

	"theblock/site/internal/webui/templkit"
)

// ---- Bülten ----

func bultenViewFrom(b *bultenData, fl *flashMsg) templkit.BultenView {
	kind, text := flashTo(fl)
	v := templkit.BultenView{FlashKind: kind, FlashText: text}
	if b == nil {
		return v
	}
	v.Aktif = strconv.Itoa(b.Counts["aktif"])
	v.OnayBek = strconv.Itoa(b.Counts["onay bekliyor"])
	v.Iptal = strconv.Itoa(b.Counts["iptal"])
	v.HasSMTP = b.SMTP.Host != ""
	v.Host = b.SMTP.Host
	if b.SMTP.Port > 0 {
		v.Port = strconv.Itoa(b.SMTP.Port)
	}
	v.User = b.SMTP.User
	v.From = b.SMTP.From
	v.FromName = b.SMTP.FromName
	v.Implicit = b.SMTP.ImplicitTLS
	return v
}

func (s *Server) renderBultenPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.BultenPanel(bultenViewFrom(d.Bulten, d.Flash)))
}

func (s *Server) renderBultenPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.BultenPanel(bultenViewFrom(d.Bulten, d.Flash)))
}

// ---- Etiketler ----

func etiketlerViewFrom(t *tagsData, fl *flashMsg) templkit.EtiketlerView {
	kind, text := flashTo(fl)
	v := templkit.EtiketlerView{FlashKind: kind, FlashText: text}
	if t == nil {
		return v
	}
	v.Total = strconv.Itoa(t.Total)
	v.Q = t.Q
	for _, it := range t.Items {
		v.Items = append(v.Items, templkit.EtiketRow{
			ID: strconv.FormatInt(it.ID, 10), Name: it.Name, Slug: it.Slug, Count: strconv.Itoa(it.Count),
		})
	}
	return v
}

func (s *Server) renderEtiketlerPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.EtiketlerPanel(etiketlerViewFrom(d.Tags, d.Flash)))
}

func (s *Server) renderEtiketlerPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.EtiketlerPanel(etiketlerViewFrom(d.Tags, d.Flash)))
}

// ---- Gazeteler (basılı ön sayfalar) ----

func gazetelerViewFrom(g *newspapersData, fl *flashMsg) templkit.GazetelerView {
	kind, text := flashTo(fl)
	v := templkit.GazetelerView{FlashKind: kind, FlashText: text}
	if g == nil {
		return v
	}
	v.Total = strconv.FormatInt(g.Total, 10)
	v.Date = g.Date
	v.HasDates = len(g.Dates) > 0
	for _, d := range g.Dates {
		v.Dates = append(v.Dates, templkit.IOpt{Value: d, Label: d, Selected: d == g.Date})
	}
	for _, it := range g.Items {
		v.Items = append(v.Items, templkit.GazeteManset{
			ID: strconv.FormatInt(it.ID, 10), Name: it.Name,
			ImageURL: it.ImageURL, ThumbURL: it.ThumbURL,
			PageDate: it.PageDate, Sort: strconv.Itoa(it.Sort),
		})
	}
	return v
}

func (s *Server) renderGazetelerPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.GazetelerPanel(gazetelerViewFrom(d.GazetelerPanel, d.Flash)))
}

func (s *Server) renderGazetelerPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.GazetelerPanel(gazetelerViewFrom(d.GazetelerPanel, d.Flash)))
}

// ---- Çeviriler ----

func cevirilerViewFrom(c *cevirilerData, fl *flashMsg) templkit.CevirilerView {
	kind, text := flashTo(fl)
	v := templkit.CevirilerView{FlashKind: kind, FlashText: text}
	if c == nil {
		return v
	}
	v.Locale = c.Locale
	v.LocaleName = c.LocaleName
	v.KeyCount = strconv.Itoa(len(c.Keys))
	for _, l := range c.Locales {
		v.Locales = append(v.Locales, templkit.IOpt{Value: l.Code, Label: l.NativeName, Selected: l.Code == c.Locale})
	}
	for _, k := range c.Keys {
		v.Rows = append(v.Rows, templkit.CeviriSatir{
			Key:     k,
			TrValue: c.TrValues[k],
			Value:   c.Values[k],
			SilVals: `{"dil":"` + c.Locale + `","sil_anahtar":"` + k + `"}`,
		})
	}
	return v
}

func (s *Server) renderCevirilerPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.CevirilerPanel(cevirilerViewFrom(d.Ceviriler, d.Flash)))
}

func (s *Server) renderCevirilerPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.CevirilerPanel(cevirilerViewFrom(d.Ceviriler, d.Flash)))
}

// ---- Sekme kategorileri ----

func tabCatsViewFrom(t *tabCatsEditData, fl *flashMsg) templkit.TabCatsView {
	kind, text := flashTo(fl)
	v := templkit.TabCatsView{FlashKind: kind, FlashText: text}
	if t == nil {
		return v
	}
	v.Configured = t.Configured
	v.Slugs = t.Slugs
	v.Count = strconv.Itoa(len(t.Slugs))
	for _, c := range t.Categories {
		v.Categories = append(v.Categories, templkit.TabCatsKategori{
			Slug: c.Slug, Name: c.Name, Yayinda: c.Status == "yayinda",
		})
	}
	return v
}

func (s *Server) renderTabCatsPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.TabCatsPanel(tabCatsViewFrom(d.TabCatsEdit, d.Flash)))
}

func (s *Server) renderTabCatsPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.TabCatsPanel(tabCatsViewFrom(d.TabCatsEdit, d.Flash)))
}

// ---- SEO ----

func seoMerkezViewFrom(se *seoData, fl *flashMsg) templkit.SeoView {
	kind, text := flashTo(fl)
	v := templkit.SeoView{FlashKind: kind, FlashText: text}
	if se == nil {
		return v
	}
	v.StaleDays = strconv.Itoa(se.StaleDays)
	for _, e := range se.Evergreen {
		cat := ""
		if e.CategoryName != nil {
			cat = *e.CategoryName
		}
		v.Evergreen = append(v.Evergreen, templkit.SeoBayat{
			ID: strconv.FormatInt(e.ID, 10), Title: e.Title, CategoryName: cat,
			YasGun: strconv.Itoa(e.YasGun), Updated: e.UpdatedAt.Format("2006-01-02"),
		})
	}
	tid := ""
	if se.Target != nil {
		v.HasTarget = true
		tid = strconv.FormatInt(se.Target.ID, 10)
		v.TargetID = tid
		v.Target = se.Target.Title
	}
	for _, c := range se.Contents {
		v.Contents = append(v.Contents, templkit.IOpt{
			Value: strconv.FormatInt(c.ID, 10), Label: c.Title, Selected: strconv.FormatInt(c.ID, 10) == tid,
		})
	}
	for _, sg := range se.Suggs {
		v.Suggs = append(v.Suggs, templkit.SeoOneri{
			Title: sg.Title, URL: sg.URL, Score: strconv.Itoa(sg.Score), Reasons: sg.Reasons, HTML: sg.HTML,
		})
	}
	return v
}

func (s *Server) renderSeoMerkezPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.SeoPanel(seoMerkezViewFrom(d.Seo, d.Flash)))
}

func (s *Server) renderSeoMerkezPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.SeoPanel(seoMerkezViewFrom(d.Seo, d.Flash)))
}

// ---- Servisler ----

func servislerViewFrom(sv *servicesData, fl *flashMsg) templkit.ServislerView {
	kind, text := flashTo(fl)
	v := templkit.ServislerView{FlashKind: kind, FlashText: text}
	if sv == nil {
		return v
	}
	v.KurVar = len(sv.Kurlar) > 0
	v.KurDemo = sv.KurDemo
	v.USD = fmt.Sprintf("%.2f", sv.Kurlar["dolar"])
	v.EUR = fmt.Sprintf("%.2f", sv.Kurlar["euro"])
	v.ALTIN = fmt.Sprintf("%.2f", sv.Kurlar["altin"])
	for _, w := range sv.Weather {
		v.Weather = append(v.Weather, templkit.ServisHava{
			City: w.City, Sicak: fmt.Sprintf("%.1f", w.Sicak), Aciklama: w.Aciklama, Demo: w.Demo,
		})
	}
	v.BorsaDemo = sv.Borsa.Demo
	for _, b := range sv.Borsa.Rows {
		v.Borsa = append(v.Borsa, templkit.ServisBorsa{
			Name: b.Name, Kind: b.Kind, Code: b.Code,
			Value: fmt.Sprintf("%.2f", b.Value), Delta: fmt.Sprintf("%.2f", b.Delta), Pozitif: b.Delta >= 0,
		})
	}
	v.FiksturDemo = sv.Fikstur.Demo
	for _, m := range sv.Fikstur.Upcoming {
		v.Fikstur = append(v.Fikstur, templkit.ServisMac{Tarih: m.Tarih, Ev: m.Ev, Deplasman: m.Deplasman})
	}
	v.FiksturKey = sv.FiksturKey
	v.EczaneKey = sv.EczaneKey
	v.AstroTarih = sv.Astro.Tarih
	for _, b := range sv.Astro.Burclar {
		v.Astro = append(v.Astro, templkit.ServisBurc{
			Ad: b.Ad, Sembol: b.Sembol, Sayi: strconv.Itoa(b.Sayi), Renk: b.Renk,
		})
	}
	return v
}

func (s *Server) renderServislerPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.ServislerPanel(servislerViewFrom(d.Servis, d.Flash)))
}

func (s *Server) renderServislerPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.ServislerPanel(servislerViewFrom(d.Servis, d.Flash)))
}

// ---- Yazarlar / Editörler ----

func peopleViewFrom(d *tmplData) templkit.PeopleView {
	kind, text := flashTo(d.Flash)
	v := templkit.PeopleView{FlashKind: kind, FlashText: text}
	for _, p := range d.AdminPeople {
		v.Items = append(v.Items, templkit.KisiSatir{
			ID: strconv.FormatInt(p.ID, 10), Name: p.Name, Slug: p.Slug,
			Role: p.Role, Title: p.Title, ArticleCount: strconv.Itoa(p.ArticleCount), Status: p.Status,
		})
	}
	if d.EditPerson != nil {
		p := d.EditPerson
		e := &templkit.KisiDuzenle{
			ID: strconv.FormatInt(p.ID, 10), Name: p.Name, Slug: p.Slug, Title: p.Title, Bio: p.Bio,
			SosyalX:        p.Socials["x"],
			SosyalLinkedIn: p.Socials["linkedin"],
			SosyalWebsite:  p.Socials["website"],
			SosyalEmail:    p.Socials["email"],
		}
		for _, r := range []string{"Yönetici", "Editör", "Muhabir", "Yazar"} {
			e.RoleOptions = append(e.RoleOptions, templkit.IOpt{Value: r, Label: r, Selected: p.Role == r})
		}
		for _, st := range []string{"aktif", "pasif", "askida"} {
			e.StatusOptions = append(e.StatusOptions, templkit.IOpt{Value: st, Label: st, Selected: p.Status == st})
		}
		cur := d.EditAvatarID
		if cur == 0 && p.AvatarMediaID != nil {
			cur = *p.AvatarMediaID
		}
		if cur != 0 {
			e.AvatarID = strconv.FormatInt(cur, 10)
		}
		for _, m := range d.PersonMedia {
			mid := m.ID
			thumb := "/media/thumb/" + strconv.FormatInt(mid, 10) + "?v=" + strconv.FormatInt(m.UpdatedAt.Unix(), 10)
			e.Media = append(e.Media, templkit.KisiMedya{
				ID: strconv.FormatInt(mid, 10), Filename: m.Filename, Thumb: thumb, Selected: cur == mid,
			})
		}
		v.Edit = e
	}
	return v
}

func (s *Server) renderPeoplePanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.PeoplePanel(peopleViewFrom(d)))
}

func (s *Server) renderPeoplePageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.PeoplePanel(peopleViewFrom(d)))
}
