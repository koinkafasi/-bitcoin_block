package httpapi

import (
	"context"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// iceriklerViewFromContents contentsData → templkit.IceriklerView eşlemesi.
// Eski contents_panel şablonunun veri bağlamına birebir karşılık üretir.
func (s *Server) iceriklerViewFromContents(ctx context.Context, c *contentsData) templkit.IceriklerView {
	eksenler := c.Eksenler
	v := templkit.IceriklerView{
		Status:      c.Status,
		Tip:         c.Tip,
		Baslangic:   c.Baslangic,
		Bitis:       c.Bitis,
		SesFiltre:   c.SesFiltre,
		Sirala:      c.Sirala,
		MinSkor:     c.MinSkor,
		Q:           c.Q,
		CurrentQS:   c.CurrentQS,
		Sayfa:       c.Sayfa,
		ToplamSayfa: c.ToplamSayfa,
		Total:       c.Total,
		TumuHref:    string(c.TumuHref),
		TumuOn:      c.Status == "",
		PrevHref:    string(c.PrevHref),
		NextHref:    string(c.NextHref),
		PrevVar:     c.Sayfa > 1,
		NextVar:     c.Sayfa < c.ToplamSayfa,
	}
	for _, st := range c.Statuses {
		v.Statuses = append(v.Statuses, templkit.ISekme{
			Href: string(st.Href), Label: st.Label,
			Count: itoa(st.Count), Active: st.Active,
		})
	}
	for _, t := range c.Types {
		v.Types = append(v.Types, templkit.IFiltre{
			Value: t.Key, Label: t.Label, Selected: c.Tip == t.Key,
		})
	}
	for _, cat := range c.Categories {
		v.Cats = append(v.Cats, templkit.IFiltre{
			Value:    strconv.FormatInt(cat.ID, 10),
			Label:    cat.Name,
			Selected: c.Kategori == cat.ID,
			Nested:   strVal(cat.ParentSlug) != "",
		})
	}
	// Toplu erişim ayarı için abonelik kademeleri. Liste veritabanından
	// geliyor; kademe adları değiştirilebildiği için sabitlenemez.
	if kademeler, err := repo.ListTiers(ctx, s.pool, true); err == nil {
		for _, k := range kademeler {
			if k.Weight <= 0 {
				continue // ağırlıksız kademe kilit anlamına gelmiyor
			}
			v.Kademeler = append(v.Kademeler, templkit.IFiltre{
				Value: strconv.Itoa(k.Weight),
				Label: k.Name,
			})
		}
	}
	v.Gunler = []templkit.IFiltre{
		{Value: "", Label: "Tüm zamanlar", Selected: c.Gun == 0},
		{Value: "7", Label: "Son 7 gün", Selected: c.Gun == 7},
		{Value: "14", Label: "Son 14 gün", Selected: c.Gun == 14},
		{Value: "30", Label: "Son 30 gün", Selected: c.Gun == 30},
	}
	// Kategori filtresi seçimi (form için).
	for i := range v.Cats {
		if v.Cats[i].Value == strconv.FormatInt(c.Kategori, 10) && c.Kategori > 0 {
			v.Cats[i].Selected = true
		}
	}
	// Aktif filtre özeti
	if c.Tip != "" || c.KategoriAd != "" || c.Gun > 0 || c.Baslangic != "" || c.Bitis != "" || c.SesFiltre != "" {
		v.FiltreVar = true
		parts := []string{"Aktif filtre:"}
		if c.Tip != "" {
			for _, t := range c.Types {
				if t.Key == c.Tip {
					parts = append(parts, t.Label)
				}
			}
		}
		if c.KategoriAd != "" {
			parts = append(parts, c.KategoriAd)
		}
		if c.Gun > 0 {
			parts = append(parts, "son "+itoa(c.Gun)+" gün")
		}
		if c.Baslangic != "" {
			parts = append(parts, c.Baslangic+"'dan")
		}
		if c.Bitis != "" {
			parts = append(parts, c.Bitis+"'e kadar")
		}
		switch c.SesFiltre {
		case repo.AudioHas:
			parts = append(parts, "röportajı olan")
		case repo.AudioPending:
			parts = append(parts, "transkript sürüyor")
		case repo.AudioFailed:
			parts = append(parts, "transkript başarısız")
		}
		v.FiltreOzet = strings.Join(parts, " · ") + " · "
		href := "/admin/icerikler"
		if c.Status != "" {
			href += "?durum=" + c.Status
		}
		v.TemizHref = href
	}
	for _, g := range c.Views {
		v.Views = append(v.Views, templkit.IGorunum{Ad: g.Ad, QS: g.QS})
	}
	if c.Sayfa > 0 {
		v.SayfaBilgi = itoa(c.Sayfa) + "/" + itoa(c.ToplamSayfa)
	}
	for _, p := range c.Pages {
		v.Pages = append(v.Pages, templkit.ISayfa{
			Href: string(p.Href), Label: p.Label, Active: p.Active, Dots: p.Dots,
		})
	}
	for _, r := range c.Rows {
		ct := r.Content
		meta := ""
		if cat := strVal(ct.CategoryName); cat != "" {
			meta += cat + " · "
		}
		meta += ct.AuthorName + " · günc. " + ct.UpdatedAt.Local().Format("02.01.2006 15:04")
		if ct.TranslationOf != nil {
			meta += " · çeviri"
		}
		hit := "—"
		if r.Hit > 0 {
			hit = strconv.FormatInt(r.Hit, 10)
		}
		e, eksenVar := eksenler[ct.ID]
		v.Rows = append(v.Rows, templkit.IcerikRow{
			ID:          ct.ID,
			Title:       ct.Title,
			Meta:        meta,
			TypeKey:     ct.TypeKey,
			Status:      ct.Status,
			BadgeAM:     ct.IsAM,
			BadgeSD:     ct.IsSD,
			BadgeDM:     ct.IsDM,
			Yayin:       r.Yayin,
			Hit:         hit,
			OncelikVar:  eksenVar,
			Oncelik:     strconv.Itoa(e.Oncelik),
			Trend:       strconv.Itoa(e.Trend),
			Tazelik:     strconv.Itoa(e.Tazelik),
			Guven:       strconv.Itoa(e.Guven),
			Tamlik:      strconv.Itoa(e.Tamlik),
			ShowYayinla: ct.Status == "taslak",
			ShowTaslak:  ct.Status == "yayinda",
			ShowArsiv:   ct.Status == "yayinda",
			ShowSil:     ct.Status != "cop",
			Diller:      s.dilEtiketleri(ct, r.Diller),
		})
	}
	return v
}

// renderIceriklerPanel içerik listesi panelini (templ) HTMX fragment'ı olarak yazar.
func (s *Server) renderIceriklerPanel(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := s.iceriklerViewFromContents(r.Context(), d.Contents)
	if d.Flash != nil {
		v.FlashKind = d.Flash.Kind
		v.FlashText = d.Flash.Text
	}
	var sb strings.Builder
	if err := templkit.IceriklerPanel(v).Render(r.Context(), &sb); err != nil {
		s.logger.Error("içerik listesi render", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request")
	_, _ = w.Write([]byte(sb.String()))
}

// renderIceriklerPage içerik listesini tam sayfa (admin kabuğu + panel) yazar.
func (s *Server) renderIceriklerPage(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := s.iceriklerViewFromContents(r.Context(), d.Contents)
	if d.Flash != nil {
		v.FlashKind = d.Flash.Kind
		v.FlashText = d.Flash.Text
	}
	var sb strings.Builder
	if err := templkit.IceriklerPanel(v).Render(r.Context(), &sb); err != nil {
		s.logger.Error("içerik listesi render", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	d.SlotHTML = template.HTML(sb.String())
	s.render(w, r, "admin_bilesenler", d)
}

func (s *Server) dilEtiketleri(c repo.Content, surumler []repo.DilSurum) []templkit.IcerikDil {
	kok := strconv.FormatInt(c.TranslationRootID(), 10)
	out := []templkit.IcerikDil{{
		Kod: c.Locale, ID: c.ID, Durum: c.Status, Var: true, Aktif: true,
	}}
	for _, d := range surumler {
		if d.Locale == c.Locale {
			continue
		}
		out = append(out, templkit.IcerikDil{
			Kod: d.Locale, ID: d.ID, Durum: d.Status, Var: true,
		})
	}
	// Eksik diller: yalnız panelde ETKİN diller, en çok 2 tane.
	if s.i18n != nil {
		for _, l := range s.i18n.Enabled() {
			if len(out) >= 4 {
				break
			}
			zaten := false
			for _, o := range out {
				if o.Kod == l.Code {
					zaten = true
					break
				}
			}
			if zaten {
				continue
			}
			out = append(out, templkit.IcerikDil{Kod: l.Code, Kaynak: kok})
		}
	}
	return out
}
