package httpapi

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"theblock/site/internal/webui/templkit"
)

// medyaViewFrom mediaData → templkit.MedyaView eşlemesi (media_panel karşılığı).
func (s *Server) medyaViewFrom(d *tmplData) templkit.MedyaView {
	m := d.Media
	v := templkit.MedyaView{}
	if d.Flash != nil {
		v.FlashKind, v.FlashText = d.Flash.Kind, d.Flash.Text
	}
	if m == nil {
		return v
	}
	v.Query = m.Query
	v.Category = m.Category
	v.Year = m.Year
	v.Month = m.Month
	v.Total = m.Total
	v.Storage = m.Storage
	v.Page = m.Page
	v.Pages = m.Pages
	v.HasPrev = m.HasPrev
	v.HasNext = m.HasNext
	v.Years = append(v.Years, m.Years...)
	aylar := []string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran",
		"Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}
	for i, ad := range aylar {
		val := strconv.Itoa(i + 1)
		v.Months = append(v.Months, templkit.MedyaAy{
			Value: val, Label: ad, Selected: m.Month == val,
		})
	}

	// Adres üretici: q/yıl/ay korunur, kategori isteğe bağlı.
	adres := func(kategori string) string {
		q := url.Values{}
		if v.Query != "" {
			q.Set("q", v.Query)
		}
		if kategori != "" {
			q.Set("kategori", kategori)
		}
		if v.Year != "" {
			q.Set("yil", v.Year)
		}
		if v.Month != "" {
			q.Set("ay", v.Month)
		}
		if len(q) == 0 {
			return "/admin/dosyalar"
		}
		return "/admin/dosyalar?" + q.Encode()
	}

	// aria-current: seçili chip "page", diğerleri "false".
	aria := func(on bool) string {
		if on {
			return "page"
		}
		return "false"
	}
	for _, c := range m.Categories {
		v.Cats = append(v.Cats, templkit.MedyaFacet{
			Value:    c.Value,
			Label:    c.Value,
			Count:    c.Count,
			URL:      adres(c.Value),
			Selected: m.Category == c.Value,
			Aria:     aria(m.Category == c.Value),
		})
		v.FacetTotal += c.Count
	}
	if v.FacetTotal == 0 {
		v.FacetTotal = m.Total
	}
	v.TumuURL = adres("")
	v.TumuAria = aria(m.Category == "")

	for _, g := range m.Groups {
		gv := templkit.MedyaGrup{Label: g.Label}
		for _, it := range g.Items {
			title := it.Title
			if title == "" {
				title = it.Filename
			}
			alt := strVal(it.Alt)
			if alt == "" {
				alt = title
			}
			dims := ""
			if it.Width != nil && it.Height != nil && *it.Width > 0 && *it.Height > 0 {
				dims = fmt.Sprintf("%d×%d", *it.Width, *it.Height)
			}
			size := ""
			if it.SizeBytes != nil && *it.SizeBytes > 0 {
				size = humanBytes(*it.SizeBytes)
			}
			gv.Items = append(gv.Items, templkit.MedyaOge{
				ID:           it.ID,
				Filename:     it.Filename,
				Title:        title,
				Alt:          alt,
				Caption:      strVal(it.Caption),
				Description:  it.Description,
				SEOTitle:     it.SEOTitle,
				SEODesc:      it.SEODescription,
				Category:     it.Category,
				URL:          s.mediaURL(&it.ID),
				Thumb:        s.mediaThumbURL(&it.ID) + "?v=" + strconv.FormatInt(it.UpdatedAt.Unix(), 10),
				Created:      it.CreatedAt.Format("02.01.2006 15:04"),
				CreatedShort: it.CreatedAt.Format("02.01.2006"),
				Size:         size,
				Dims:         dims,
				KindLabel:    mediaKindLabel(it.Kind),
			})
		}
		v.Groups = append(v.Groups, gv)
	}

	// Sonraki dilim adresi. `parca=1` yalnız grupların döndüğünü bildirir
	// (sonsuz kaydırma gözcüsü bunu çağırır).
	dilim := func(page int) string {
		q := url.Values{}
		if v.Query != "" {
			q.Set("q", v.Query)
		}
		if v.Category != "" {
			q.Set("kategori", v.Category)
		}
		if v.Year != "" {
			q.Set("yil", v.Year)
		}
		if v.Month != "" {
			q.Set("ay", v.Month)
		}
		q.Set("sayfa", strconv.Itoa(page))
		q.Set("parca", "1")
		return "/admin/dosyalar?" + q.Encode()
	}
	// Sayfalama bağlantıları (kaldırıldı ama eski yer imleri/otomasyon için).
	mk := func(page int) string {
		q := url.Values{}
		if v.Query != "" {
			q.Set("q", v.Query)
		}
		if v.Category != "" {
			q.Set("kategori", v.Category)
		}
		if v.Year != "" {
			q.Set("yil", v.Year)
		}
		if v.Month != "" {
			q.Set("ay", v.Month)
		}
		q.Set("sayfa", strconv.Itoa(page))
		return "/admin/dosyalar?" + q.Encode()
	}
	v.PrevURL = mk(m.Page - 1)
	v.NextURL = mk(m.Page + 1)
	v.ParcaURL = dilim(m.Page + 1)
	return v
}

// renderMedyaPanel medya panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderMedyaPanel(w http.ResponseWriter, r *http.Request, d *tmplData) {
	var sb strings.Builder
	if err := templkit.MedyaPanel(s.medyaViewFrom(d)).Render(r.Context(), &sb); err != nil {
		s.logger.Error("medya render", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request")
	_, _ = w.Write([]byte(sb.String()))
}

// renderMedyaParca sonsuz kaydırmanın sonraki dilimini yazar: yalnız grup
// bölümleri + yeni gözcü. Panel kabuğu (araç çubuğu, modal) tekrar
// gönderilmez; gelen HTML gözcünün yerine geçer.
func (s *Server) renderMedyaParca(w http.ResponseWriter, r *http.Request, d *tmplData) {
	var sb strings.Builder
	if err := templkit.MedyaGruplar(s.medyaViewFrom(d), false).Render(r.Context(), &sb); err != nil {
		s.logger.Error("medya parça render", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(sb.String()))
}

// renderMedyaPage medya panelini tam sayfa (admin kabuğu) yazar.
func (s *Server) renderMedyaPage(w http.ResponseWriter, r *http.Request, d *tmplData) {
	var sb strings.Builder
	if err := templkit.MedyaPanel(s.medyaViewFrom(d)).Render(r.Context(), &sb); err != nil {
		s.logger.Error("medya render", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	d.SlotHTML = template.HTML(sb.String())
	s.render(w, r, "admin_bilesenler", d)
}

// mediaKindLabel medya türü rozet metni.
func mediaKindLabel(kind string) string {
	switch kind {
	case "image":
		return "Görsel"
	case "audio":
		return "Ses"
	case "video":
		return "Video"
	case "document", "file":
		return "Dosya"
	}
	return kind
}
