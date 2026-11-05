package httpapi

import (
	"net/http"
	"strconv"

	"theblock/site/internal/webui/templkit"
)

// gazetePanelViewFrom gazeteViewData → templkit.GazeteView eşlemesi.
func gazetePanelViewFrom(g *gazeteViewData, fl *flashMsg) templkit.GazeteView {
	kind, text := flashTo(fl)
	v := templkit.GazeteView{FlashKind: kind, FlashText: text}
	if g == nil {
		return v
	}
	v.Licensed = g.Licensed
	v.Role = g.Role
	v.IssueCount = strconv.Itoa(len(g.Issues))
	v.TplCount = strconv.Itoa(len(g.Templates))
	v.SubCount = strconv.FormatInt(g.SubCount, 10)
	v.PermLisans = g.Perm["lisans"]
	v.PermEkip = g.Perm["ekip"]
	v.PermDuzenle = g.Perm["duzenle"]
	v.PermAbone = g.Perm["abone"]

	for _, s := range g.Sites {
		acik := "0"
		btn := "Aç"
		if s.Enabled {
			acik, btn = "1", "Kapat"
		}
		v.Sites = append(v.Sites, templkit.GazeteSiteLisans{
			ID: strconv.FormatInt(s.ID, 10), Name: s.Name, Domain: s.Domain,
			Enabled: s.Enabled, AcikVal: acik, BtnText: btn,
		})
	}

	st := g.Storage
	v.StorageSecret = g.StorageSecret
	v.Storage = templkit.GazeteStorageView{
		ProviderLocal: st.Provider == "local",
		ProviderR2:    st.Provider == "r2",
		ProviderS3:    st.Provider == "s3",
		ProviderMinio: st.Provider == "minio",
		Endpoint:      st.Endpoint,
		Bucket:        st.Bucket,
		Region:        st.Region,
		AccessKeyID:   st.AccessKeyID,
		PublicBaseURL: st.PublicBaseURL,
		PathPrefix:    st.PathPrefix,
		Enabled:       st.Enabled,
	}

	v.AyarSaat = g.Ayar["saat"]
	v.AyarHaftalik = g.Ayar["haftalik_gun"]
	if v.AyarHaftalik == "" {
		v.AyarHaftalik = "1"
	}
	v.AyarAylik = g.Ayar["aylik_gun"]
	if v.AyarAylik == "" {
		v.AyarAylik = "1"
	}
	v.AyarOtoYayin = g.Ayar["otomatik_yayin"] == "1"
	v.WAPhone = g.WA["phone_id"]
	v.WATokenSet = g.WA["token"] != ""
	v.WAVersion = g.WA["api_version"]
	if v.WAVersion == "" {
		v.WAVersion = "v21.0"
	}
	v.WATemplate = g.WA["template"]
	v.SMSProvider = g.SMS["provider"]
	v.SMSFrom = g.SMS["from"]
	v.SMSNetgsmUser = g.SMS["netgsm_user"]
	v.SMSNetgsmPassSet = g.SMS["netgsm_password"] != ""
	v.SMSTwilioSID = g.SMS["twilio_sid"]
	v.SMSTwilioTokenSet = g.SMS["twilio_token"] != ""

	for _, u := range g.Users {
		v.Users = append(v.Users, templkit.GazeteKullanici{
			ID: strconv.FormatInt(u.ID, 10), Name: u.Name, Role: u.Role,
		})
	}
	for _, m := range g.Team {
		v.Team = append(v.Team, templkit.GazeteEkipSatir{
			UserID:    strconv.FormatInt(m.UserID, 10),
			UserName:  m.UserName,
			UserEmail: m.UserEmail,
			UserRole:  m.UserRole,
			Rol:       m.Rol,
			SilURL:    "/admin/gazete/ekip/" + strconv.FormatInt(m.UserID, 10) + "/sil",
		})
	}
	for _, t := range g.Templates {
		// Galeriyle AYNI onizleme: sablonu secmeden once neye benzedigini
		// gormek icin (Faz 184). En fazla 10 sayfa, galeriyle tutarli.
		pages := sablonPagePreview(t.Pages)
		if len(pages) > 10 {
			pages = pages[:10]
		}
		id := strconv.FormatInt(t.ID, 10)
		v.Templates = append(v.Templates, templkit.GazeteSablonSatir{
			ID:          id,
			Name:        t.Name,
			Description: t.Description,
			IsDefault:   t.IsDefault,
			SilURL:      "/admin/gazete/sablonlar/" + id + "/sil",
			Pages:       pages,
			SayfaSayisi: len(pages),
			YeniSayiURL: "/admin/gazete/sayilar?sablon_id=" + id,
		})
	}
	for _, i := range g.Issues {
		id := strconv.FormatInt(i.ID, 10)
		durumVal := "yayinda"
		durumBtn := "Yayınla"
		if i.Status == "yayinda" {
			durumVal, durumBtn = "arsiv", "Geri Çek"
		}
		v.Issues = append(v.Issues, templkit.GazeteSayiSatir{
			ID:         id,
			IssueDate:  i.IssueDate,
			Period:     i.Period,
			Title:      i.Title,
			Status:     i.Status,
			PageCount:  strconv.Itoa(i.PageCount),
			DuzenleURL: "/admin/gazete/sayilar/" + id + "/duzenle",
			PDFURL:     "/admin/gazete/sayilar/" + id + "/pdf",
			DurumURL:   "/admin/gazete/sayilar/" + id + "/durum",
			DurumVal:   durumVal,
			DurumBtn:   durumBtn,
			GonderURL:  "/admin/gazete/sayilar/" + id + "/gonder",
			SilURL:     "/admin/gazete/sayilar/" + id + "/sil",
		})
	}
	for _, s := range g.Subs {
		id := strconv.FormatInt(s.ID, 10)
		durumVal, durumBtn := "aktif", "Aktif Et"
		if s.Status == "aktif" {
			durumVal, durumBtn = "iptal", "İptal Et"
		}
		v.Subs = append(v.Subs, templkit.GazeteAboneSatir{
			ID:        id,
			Email:     s.Email,
			Phone:     s.Phone,
			Channel:   s.Channel,
			Frequency: s.Frequency,
			Status:    s.Status,
			DurumURL:  "/admin/gazete/aboneler/" + id + "/durum",
			DurumVal:  durumVal,
			DurumBtn:  durumBtn,
			SilURL:    "/admin/gazete/aboneler/" + id + "/sil",
		})
	}
	return v
}

// renderGazetePanelV2 gazete panelini HTMX fragment'i olarak yazar.
func (s *Server) renderGazetePanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.GazetePanel(gazetePanelViewFrom(d.Gazete, d.Flash)))
}

// renderGazetePageV2 gazete panelini tam sayfa yazar.
func (s *Server) renderGazetePageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.GazetePanel(gazetePanelViewFrom(d.Gazete, d.Flash)))
}
