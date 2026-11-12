package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"theblock/site/internal/webui/templkit"
)

// hxValsJSON hx-vals için güvenli JSON üretir.
func hxValsJSON(pairs ...string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// icerikFormViewFrom formData → templkit.IcerikFormView eşlemesi.
func (s *Server) icerikFormViewFrom(d *tmplData) templkit.IcerikFormView {
	f := d.Form
	kind, text := flashTo(d.Flash)
	v := templkit.IcerikFormView{FlashKind: kind, FlashText: text, Lang: d.Lang, SiteName: d.SiteName}
	if f == nil {
		return v
	}
	v.IsNew = f.IsNew
	v.NumID = f.Content.ID
	v.HasContent = f.Content.ID != 0
	v.ContentID = strconv.FormatInt(f.Content.ID, 10)
	v.Version = strconv.Itoa(f.Content.Version)
	if d.User != nil {
		v.UserID = strconv.FormatInt(d.User.ID, 10)
		v.UserName = d.User.Name
	} else {
		v.UserID = "0"
	}
	if f.IsNew {
		v.ActionURL = "/admin/icerikler"
		v.AutosaveURL = ""
	} else {
		v.ActionURL = "/admin/icerikler/" + v.ContentID
		v.AutosaveURL = "/admin/icerikler/" + v.ContentID + "/otomatik-kaydet"
	}
	v.PreviewURL = "/admin/icerikler/" + v.ContentID + "/onizleme"
	v.DeleteURL = "/admin/icerikler/" + v.ContentID + "/sil"
	v.AudioActionURL = "/admin/icerikler/" + v.ContentID + "/seslendir"
	v.AISkorURL = "/admin/icerikler/" + v.ContentID + "/ai-skor"
	v.KaynakKontrol = "/admin/icerikler/" + v.ContentID + "/kaynak-kontrol"
	v.KaynakEkle = "/admin/icerikler/" + v.ContentID + "/kaynak-ekle"

	v.Title = f.Content.Title
	v.Spot = f.Content.Spot
	v.DetailJSON = f.DetailJSON
	v.TitleLen = f.TitleLen
	v.TypeKey = f.TypeKey
	v.TagsText = f.TagsText
	v.VideoSure = f.VideoSure
	v.HesaplayiciSema = f.HesaplayiciSema
	v.PublishAtValue = f.PublishAtValue
	v.YMYL = f.IsYMYL
	v.YMYLDurumu = f.YMYLDurumu
	v.YMYLUzman = f.YMYLUzman
	v.OrigIndex = f.OrigIndex
	v.AiScore = strconv.Itoa(f.Content.AiScore)
	v.CommentAcik = f.Content.CommentStatus != "kapali"
	v.IsExclusive = f.Content.IsExclusive
	v.ImageGrayscale = f.Content.ImageGrayscale
	v.MinTierWeight = f.Content.MinTierWeight
	v.SourceURL = derefStr(f.Content.SourceURL)
	v.SourceName = derefStr(f.Content.SourceName)
	v.SeoTitle = derefStr(f.Content.SeoTitle)
	v.SeoDesc = derefStr(f.Content.SeoDesc)
	v.SeoScore = f.SEO.Score
	v.SeoText = f.SEOText
	for _, p := range f.SEO.Parts {
		v.SeoParts = append(v.SeoParts, templkit.IcerikSeoPart{Name: p.Name, Weight: p.Weight})
	}
	v.ImageCredit = f.Fields.ImageCredit
	v.SocialSummary = f.Fields.SocialSummary
	v.AISummary = f.Fields.AISummary
	v.FieldAudioURL = f.Fields.AudioURL
	v.FieldAudioStatus = f.Fields.AudioStatus
	if f.Fields.AudioSeconds > 0 {
		v.FieldAudioSeconds = fmt.Sprintf("%.0f", f.Fields.AudioSeconds)
	}
	if f.Fields.ScrollCount > 0 {
		v.FieldScrollCount = strconv.FormatInt(f.Fields.ScrollCount, 10)
	}
	v.AudioURL = f.AudioURL
	v.AudioStatus = f.AudioStatus
	if f.AudioSeconds > 0 {
		v.AudioSeconds = fmt.Sprintf("%.0f", f.AudioSeconds)
	}

	v.TitleActionVals = hxValsJSON("action", "title", "content_id", v.ContentID, "target", "#f_title")
	v.SpotActionVals = hxValsJSON("action", "spot", "content_id", v.ContentID, "target", "#f_spot")
	v.SeoTitleVals = hxValsJSON("action", "seo_title", "content_id", v.ContentID, "target", "[name=seo_baslik]")
	v.SeoDescVals = hxValsJSON("action", "seo_description", "content_id", v.ContentID, "target", "[name=seo_aciklama]")

	for _, t := range f.Types {
		v.Types = append(v.Types, templkit.IOpt{Value: t.Key, Label: t.Label, Selected: t.Key == f.TypeKey})
	}
	for _, c := range f.VideoCats {
		v.VideoCats = append(v.VideoCats, templkit.IOpt{Value: strconv.FormatInt(c.ID, 10), Label: c.Name})
	}
	for _, vid := range f.Videos {
		cat := ""
		if vid.CategoryID != nil {
			cat = strconv.FormatInt(*vid.CategoryID, 10)
		}
		v.Videos = append(v.Videos, templkit.IcerikVideo{
			ID:       strconv.FormatInt(vid.ID, 10),
			Title:    vid.Title,
			Cover:    s.mediaURL(vid.CoverMediaID),
			Slug:     vid.DisplaySlug() + "-" + strconv.FormatInt(vid.ID, 10),
			Spot:     vid.Spot,
			Sure:     f.VideoSureMap[vid.ID],
			Cat:      cat,
			Selected: vid.ID == f.RelatedVideoID,
		})
	}
	for _, c := range f.Categories {
		label := c.Name
		if c.ParentSlug != nil && *c.ParentSlug != "" {
			label = "↳ " + label
		}
		v.Categories = append(v.Categories, templkit.IOpt{
			Value: strconv.FormatInt(c.ID, 10), Label: label, Selected: c.ID == f.CategorySelected,
		})
	}
	for _, m := range f.Mahrets {
		v.Mahrets = append(v.Mahrets, templkit.IOpt{
			Value: strconv.FormatInt(m.ID, 10), Label: m.Name, Selected: m.ID == f.MahretSelected,
		})
	}
	for _, p := range f.People {
		label := p.Name + " (" + p.Role + ")"
		id := strconv.FormatInt(p.ID, 10)
		v.Authors = append(v.Authors, templkit.IOpt{Value: id, Label: label, Selected: p.ID == f.AuthorSelected})
		v.Editors = append(v.Editors, templkit.IOpt{Value: id, Label: label, Selected: p.ID == f.EditorSelected})
	}
	for _, c := range f.Clusters {
		v.Clusters = append(v.Clusters, templkit.IOpt{
			Value:    strconv.FormatInt(c.ID, 10),
			Label:    c.Name + " (" + c.Intent + ")",
			Selected: c.ID == f.ClusterSelected,
		})
	}
	for _, r := range f.Rozets {
		v.Rozets = append(v.Rozets, templkit.IcerikRozet{Code: r.Code, On: r.On})
	}
	v.Langs = f.LangRows
	v.CeviriSayisi = f.CeviriSayisi
	v.IcerikLocale = f.Content.Locale
	v.IcerikLocaleAd = s.dilAdiTR(f.Content.Locale)
	v.CeviriMi = f.Content.TranslationOf != nil
	if v.CeviriMi {
		v.KaynakID = strconv.FormatInt(f.Content.TranslationRootID(), 10)
	} else if f.Content.ID != 0 {
		v.KaynakID = strconv.FormatInt(f.Content.ID, 10)
	}

	v.GelismisVar, v.GelismisIpucu = gelismisAlanlar(f.TypeKey)

	v.KaynakBaslik = f.KaynakBaslik
	v.KaynakSpot = f.KaynakSpot
	v.KaynakGovdeHTML = f.KaynakGovdeHTML
	for _, slot := range f.MediaSlots {
		s := templkit.IcerikMediaSlot{Name: slot.Name, Help: slot.Help}
		for _, o := range slot.Options {
			s.Options = append(s.Options, templkit.IcerikMediaOption{
				ID: strconv.FormatInt(o.ID, 10), Filename: o.Filename, Thumb: o.ThumbURL, Selected: o.Selected,
			})
		}
		v.MediaSlots = append(v.MediaSlots, s)
	}
	for _, m := range f.Media {
		if m.Kind != "image" {
			continue
		}
		mid := strconv.FormatInt(m.ID, 10)
		title := m.Title
		if title == "" {
			title = m.Filename
		}
		v.Media = append(v.Media, templkit.IcerikMedia{
			ID:             mid,
			URL:            s.mediaURL(&m.ID),
			Thumb:          s.mediaThumbURL(&m.ID) + "?v=" + strconv.FormatInt(m.UpdatedAt.Unix(), 10),
			Alt:            derefStr(m.Alt),
			Caption:        derefStr(m.Caption),
			Title:          title,
			Description:    m.Description,
			SeoTitle:       m.SEOTitle,
			SeoDescription: m.SEODescription,
			Category:       m.Category,
		})
	}
	for _, t := range f.TrendSuggestions {
		v.Trends = append(v.Trends, templkit.IcerikTrend{
			Term: t.Term, Score: strconv.Itoa(t.Score), Coverage: strconv.Itoa(t.CoverageCount),
		})
	}
	for _, a := range f.AIActions {
		v.AIActions = append(v.AIActions, templkit.IcerikAIAction{
			ID: a.ID, Label: a.Label, Description: a.Description, Icon: a.Icon, Research: a.Research,
		})
	}
	for _, s := range f.Sources {
		v.Sources = append(v.Sources, templkit.IcerikSource{
			ID: strconv.FormatInt(s.ID, 10), URL: s.URL, Accessible: s.Accessible,
		})
	}
	v.SesUploadURL = "/admin/icerikler/" + v.ContentID + "/ses"
	v.Audios = s.icerikAudioViews(f.Content.ID, f.Audio)
	for _, a := range v.Audios {
		if a.IsActive {
			v.SesPollURL = "/admin/icerikler/" + v.ContentID + "/ses/durum"
			break
		}
	}
	if f.KaliteChecked {
		v.Kalite = qualityView(f.Content.ID, f.Kalite)
	}
	return v
}

// renderIcerikFormPanelV2 içerik formunu HTMX parçası olarak yazar.
func (s *Server) renderIcerikFormPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.IcerikFormPanel(s.icerikFormViewFrom(d)))
}

// renderIcerikFormPageV2 içerik formunu tam sayfa yazar.
func (s *Server) renderIcerikFormPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.IcerikFormPanel(s.icerikFormViewFrom(d)))
}

// derefStr *string alanını güvenli biçimde string'e çevirir.
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func gelismisAlanlar(typeKey string) (bool, string) {
	switch typeKey {
	case "video":
		return true, "video süresi"
	case "hesaplayici":
		return true, "hesaplama şeması"
	}
	return false, ""
}
