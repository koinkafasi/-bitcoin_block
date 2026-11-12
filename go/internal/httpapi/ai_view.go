package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/webui/templkit"
)

// tfaViewFrom 2FA paneli verisi.
func tfaViewFrom(d *tmplData) templkit.TFAView {
	kind, text := "", ""
	if d.Flash != nil {
		kind, text = d.Flash.Kind, d.Flash.Text
	}
	return templkit.TFAView{FlashKind: kind, FlashText: text, Enabled: d.TOTPEnabled, Secret: d.TOTPSecret}
}

// aiKaynakViewFrom AI Kaynak Merkezi verisi.
func aiKaynakViewFrom(a *aiSourceData, fl *flashMsg) templkit.AIKaynakView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.AIKaynakView{FlashKind: kind, FlashText: text}
	if a == nil {
		return v
	}
	v.AISaglayici = a.AISaglayici
	if a.CeviriBekleyen > 0 {
		v.CeviriBekleyen = strconv.Itoa(a.CeviriBekleyen)
	}
	if a.CeviriHatali > 0 {
		v.CeviriHatali = strconv.Itoa(a.CeviriHatali)
	}
	v.TotalItems = strconv.Itoa(a.TotalItems)
	v.ClusterCount = strconv.Itoa(len(a.Clusters))
	v.FeedCount = strconv.Itoa(len(a.Feeds))
	v.TrendCount = strconv.Itoa(len(a.Trends))
	if a.EnrichPending > 0 {
		v.EnrichPending = strconv.Itoa(a.EnrichPending)
	}
	for _, p := range a.Plan {
		v.Plan = append(v.Plan, templkit.AIKaynakPlanSatir{
			Name:       p.Name,
			Feeds:      strconv.Itoa(p.Feeds),
			Clusters:   strconv.Itoa(p.Clusters),
			TrendHits:  strconv.Itoa(p.TrendHits),
			TopCluster: p.TopCluster,
		})
	}
	for _, p := range a.Plans {
		v.Plans = append(v.Plans, templkit.AIKaynakPlan{
			ID:           strconv.FormatInt(p.ID, 10),
			CategoryName: p.CategoryName,
			DraftedToday: strconv.Itoa(p.DraftedToday),
			DailyTarget:  strconv.Itoa(p.DailyTarget),
			Eligible:     strconv.Itoa(p.EligibleClusters),
			LastRun:      fmtClock(p.LastRunAt),
			MinSources:   strconv.Itoa(p.MinSources),
			TrendWeight:  strconv.Itoa(p.TrendWeight),
			MaxAgeHours:  strconv.Itoa(p.MaxAgeHours),
			Aktif:        p.Aktif,
		})
	}
	for _, c := range a.Clusters {
		k := templkit.AIKaynakKume{
			ID:            strconv.FormatInt(c.ID, 10),
			Title:         c.Title,
			CategoryName:  c.CategoryName,
			ItemCount:     strconv.Itoa(c.ItemCount),
			SourceCount:   strconv.Itoa(c.SourceCount),
			CrossCategory: c.CrossCategory,
			TrendTerm:     c.TrendTerm,
			Sources:       c.Sources,
		}
		if c.TrendScore > 0 {
			k.TrendScore = strconv.Itoa(c.TrendScore)
		}
		if c.ContentID != nil {
			k.ContentID = strconv.FormatInt(*c.ContentID, 10)
		}
		v.Clusters = append(v.Clusters, k)
	}
	for _, f := range a.Feeds {
		row := templkit.AIKaynakFeed{
			ID:              strconv.FormatInt(f.ID, 10),
			Name:            f.Name,
			URL:             f.URL,
			Homepage:        f.Homepage,
			CategoryName:    f.CategoryName,
			KindLabel:       feedKindLabel(f.Kind),
			Trust:           strconv.Itoa(f.Trust),
			Aktif:           f.Aktif,
			LastStatus:      f.LastStatus,
			LastError:       f.LastError,
			Dil:             strings.ToUpper(f.Dil),
			OtomatikKapandi: f.OtomatikKapandi,
			KapanmaNedeni:   feedKapanmaNedeni(f.Notes),
		}
		if f.GorselOran != nil {
			row.GorselOran = strconv.Itoa(*f.GorselOran)
		}
		if f.LastOkAt != nil {
			row.SonBasari = f.LastOkAt.Local().Format("02.01 15:04")
		}
		if f.ArdisikHata > 0 {
			row.ArdisikHata = strconv.Itoa(f.ArdisikHata)
		}
		if f.LastCheckedAt != nil {
			row.LastChecked = f.LastCheckedAt.Local().Format("02.01 15:04")
		}
		if f.LastItemCount > 0 {
			row.LastItems = strconv.Itoa(f.LastItemCount)
		}
		v.Feeds = append(v.Feeds, row)
	}
	for _, t := range a.Trends {
		b := ""
		if t.Score > 0 {
			b = strconv.Itoa(t.Score)
		}
		v.Trends = append(v.Trends, templkit.TrendBasit{A: t.Term, B: b})
	}
	return v
}

// feedKindLabel kaynak türü etiketi.
func feedKindLabel(kind string) string {
	switch kind {
	case "html":
		return "HTML"
	case "sitemap":
		return "Sitemap"
	case "bingnews":
		return "Bing RSS"
	case "passive":
		return "Pasif"
	}
	return "RSS"
}

// fmtClock *time.Time → "02.01 15:04" (boşsa "").
func fmtClock(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Local().Format("02.01 15:04")
}

// aiSohbetViewFrom AI sohbet verisi.
func aiSohbetViewFrom(sc *sohbetData, fl *flashMsg) templkit.AISohbetView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.AISohbetView{FlashKind: kind, FlashText: text}
	if sc == nil {
		return v
	}
	v.ActiveID = strconv.FormatInt(sc.ActiveID, 10)
	for _, c := range sc.Convs {
		v.Convs = append(v.Convs, templkit.AISohbetKonu{
			ID: strconv.FormatInt(c.ID, 10), Title: c.Title, Active: c.ID == sc.ActiveID,
		})
	}
	for _, m := range sc.Msgs {
		v.Msgs = append(v.Msgs, templkit.AISohbetMesaj{Role: m.Role, Content: m.Content})
	}
	for _, t := range sc.Templates {
		v.Templates = append(v.Templates, templkit.AISohbetSablon{Icon: t.Icon, Title: t.Title, Prompt: t.Prompt})
	}
	return v
}

// ---- Render yardımcıları ----

func (s *Server) renderTFAPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.TFAPanel(tfaViewFrom(d)))
}

func (s *Server) renderTFAV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.TFAPanel(tfaViewFrom(d)))
}

func (s *Server) renderAIKaynakPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.AIKaynakPanel(aiKaynakViewFrom(d.AISource, d.Flash)))
}

func (s *Server) renderAIKaynakPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.AIKaynakPanel(aiKaynakViewFrom(d.AISource, d.Flash)))
}

func (s *Server) renderAISohbetPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.AISohbetPanel(aiSohbetViewFrom(d.AISohbet, d.Flash)))
}

func (s *Server) renderAISohbetPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.AISohbetPanel(aiSohbetViewFrom(d.AISohbet, d.Flash)))
}

// feedKapanmaNedeni notes alanından kısa etiket çıkarır.
//
// Kapalı bir kaynağın NEDEN kapalı olduğu listede görünmüyordu:
// 12 Kasım 2026 denetiminde 22 kapalı kaynağın hangisinin sağlam,
// hangisinin bozuk olduğu ancak tek tek çekilerek anlaşılabildi.
func feedKapanmaNedeni(notes string) string {
	switch {
	case strings.HasPrefix(notes, "KONU DISI"):
		return "konu dışı"
	case strings.HasPrefix(notes, "BOZUK"):
		return "bozuk"
	case strings.Contains(notes, "bot korumasi"):
		return "bot koruması"
	case strings.Contains(notes, "ACILABILIR"):
		return "açılabilir"
	}
	return ""
}
