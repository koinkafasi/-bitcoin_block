package httpapi

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Gazete yönetim paneli (Faz 1/2) ----

// gazeteSiteLicense lisans tablosu satırı (site bazlı özellik anahtarı).
type gazeteSiteLicense struct {
	ID       int64
	Name     string
	Domain   string
	Enabled  bool
	IssueCnt int64
}

// gazeteViewData gazete paneli görünüm modeli.
type gazeteViewData struct {
	Active        string // genel | lisans | depolama | ekip | sablonlar | sayilar | aboneler
	SiteID        int64
	Role          string
	Licensed      bool
	Sites         []gazeteSiteLicense
	Issues        []repo.GazeteIssue
	Issue         *repo.GazeteIssue
	Pages         []repo.GazetePage
	Templates     []repo.GazeteTemplate
	Team          []repo.GazeteTeamMember
	Users         []repo.MemberRow
	Storage       repo.SiteStorage
	StorageSecret bool
	Subs          []repo.GazeteSubscription
	SubCount      int64
	SentCount     int64
	PageJSON      template.JS
	Perm          map[string]bool
	Ayar          map[string]string // settings('gazete'): saat, otomatik_yayin, ...
	WA            map[string]string // settings('whatsapp'): token, phone_id, ...
	SMS           map[string]string // settings('sms'): provider, from, netgsm_*, twilio_*
}

// gazeteView panel görünümünü kurar.
func (s *Server) gazeteView(r *http.Request, flash *flashMsg) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	d := s.baseView("tr", "/admin", "Gazete")
	d.User = ptr(userFromContext(ctx))
	d.Flash = flash
	v := &gazeteViewData{
		Active:   "genel",
		SiteID:   sid,
		Role:     s.gazeteRole(ctx, sid),
		Licensed: s.gazeteLicensed(ctx, sid),
		Perm: map[string]bool{
			"goruntule": s.gazeteCan(ctx, sid, gazeteActionView),
			"duzenle":   s.gazeteCan(ctx, sid, gazeteActionEdit),
			"yayinla":   s.gazeteCan(ctx, sid, gazeteActionPublish),
			"abone":     s.gazeteCan(ctx, sid, gazeteActionSubs),
			"ekip":      s.gazeteCan(ctx, sid, gazeteActionTeam),
			"lisans":    s.gazeteCan(ctx, sid, gazeteActionLicense),
		},
	}
	// Lisans tablosu (tüm siteler) — yalnızca yönetici görür.
	if v.Perm["lisans"] {
		if sites, err := repo.ListSites(ctx, s.pool); err == nil {
			for _, site := range sites {
				on, _ := repo.SiteFeatureEnabled(ctx, s.pool, site.ID, "gazete")
				issues, _ := repo.ListGazeteIssues(ctx, s.pool, site.ID, "", "", "", 1)
				cnt := int64(len(issues))
				v.Sites = append(v.Sites, gazeteSiteLicense{
					ID: site.ID, Name: site.Name, Domain: site.Domain, Enabled: on, IssueCnt: cnt,
				})
			}
		}
	}
	issues, _ := repo.ListGazeteIssues(ctx, s.pool, sid, "", "", "", 60)
	v.Issues = issues
	tpls, _ := repo.ListGazeteTemplates(ctx, s.pool, sid)
	v.Templates = tpls
	team, _ := repo.ListGazeteTeam(ctx, s.pool, sid)
	v.Team = team
	if v.Perm["ekip"] {
		if users, err := repo.ListUsers(ctx, s.pool); err == nil {
			v.Users = users
		}
	}
	st, _ := repo.GetSiteStorage(ctx, s.pool, sid)
	v.Storage = st
	v.StorageSecret = st.SecretEnc != ""
	v.Ayar = map[string]string{}
	for _, k := range []string{"saat", "otomatik_yayin", "haftalik_gun", "aylik_gun"} {
		if raw, err := repo.GetSetting(ctx, s.pool, "gazete", k); err == nil {
			v.Ayar[k] = strings.TrimSpace(string(raw))
		}
	}
	v.WA = map[string]string{}
	for _, k := range []string{"token", "phone_id", "api_version", "template"} {
		if raw, err := repo.GetSetting(ctx, s.pool, "whatsapp", k); err == nil {
			v.WA[k] = strings.TrimSpace(string(raw))
		}
	}
	v.SMS = map[string]string{}
	for _, k := range []string{"provider", "from", "netgsm_user", "netgsm_password", "twilio_sid", "twilio_token"} {
		if raw, err := repo.GetSetting(ctx, s.pool, "sms", k); err == nil {
			v.SMS[k] = strings.TrimSpace(string(raw))
		}
	}
	subs, _ := repo.ListGazeteSubscriptions(ctx, s.pool, sid, "", 200)
	v.Subs = subs
	v.SubCount, _ = repo.CountGazeteSubscriptions(ctx, s.pool, sid)
	d.Gazete = v
	return d
}

// ---- Genel panel ----

// handleGazeteAdmin GET /admin/gazete — ana panel.
func (s *Server) handleGazeteAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.gazeteCan(r.Context(), s.curSiteID(r), gazeteActionView) {
		s.gazeteDenied(w, r)
		return
	}
	d := s.gazeteView(r, s.consumeFlash(r))
	if IsHTMX(r) {
		s.renderGazetePanelV2(w, r, d)
		return
	}
	s.renderGazetePageV2(w, r, d)
}

// gazeteResult HTMX/normal istek sonuç yazıcısı.
func (s *Server) gazeteResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		d := s.gazeteView(r, &flashMsg{Kind: kind, Text: text})
		s.renderGazetePanelV2(w, r, d)
		return
	}
	s.flashRedirect(w, r, "/admin/gazete", kind, text)
}

// ---- Site lisansı (ticari paket) ----

// handleGazeteLisans POST /admin/gazete/lisans — site bazlı aç/kapat (Yönetici).
func (s *Server) handleGazeteLisans(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionLicense) {
		return
	}
	siteID := parseID(r.FormValue("site_id"))
	if siteID == 0 {
		siteID = s.curSiteID(r)
	}
	on := r.FormValue("acik") == "1"
	if err := repo.SetSiteFeature(r.Context(), s.pool, siteID, "gazete", on, nil); err != nil {
		s.logger.Error("gazete lisans", "error", err)
		s.gazeteResult(w, r, "error", "Lisans güncellenemedi.")
		return
	}
	s.audit(r, "gazete_lisans", "site:"+strconv.FormatInt(siteID, 10)+"="+strconv.FormatBool(on))
	s.gazeteResult(w, r, "info", "Gazete lisansı güncellendi.")
}

// ---- Depolama yapılandırması (R2/S3) ----

// handleGazeteDepolama POST /admin/gazete/depolama — site depolama ayarı (Yönetici).
func (s *Server) handleGazeteDepolama(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionLicense) {
		return
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	st := repo.SiteStorage{
		SiteID:        sid,
		Provider:      strings.TrimSpace(r.FormValue("provider")),
		Endpoint:      strings.TrimSpace(r.FormValue("endpoint")),
		Bucket:        strings.TrimSpace(r.FormValue("bucket")),
		Region:        strings.TrimSpace(r.FormValue("region")),
		AccessKeyID:   strings.TrimSpace(r.FormValue("access_key")),
		PublicBaseURL: strings.TrimSpace(r.FormValue("public_url")),
		PathPrefix:    strings.TrimSpace(r.FormValue("prefix")),
		Enabled:       r.FormValue("acik") == "1",
	}
	secret := strings.TrimSpace(r.FormValue("secret"))
	box := s.secretBox()
	if secret != "" && box != nil {
		enc, err := box.Seal(secret)
		if err == nil {
			st.SecretEnc = enc
		}
	} else {
		// Boş bırakıldıysa mevcut gizli anahtar korunur.
		old, _ := repo.GetSiteStorage(ctx, s.pool, sid)
		st.SecretEnc = old.SecretEnc
	}
	if err := repo.UpsertSiteStorage(ctx, s.pool, &st); err != nil {
		s.logger.Error("gazete depolama", "error", err)
		s.gazeteResult(w, r, "error", "Depolama ayarı kaydedilemedi.")
		return
	}
	s.audit(r, "gazete_depolama", st.Provider+"/"+st.Bucket)
	s.gazeteResult(w, r, "info", "Depolama ayarı kaydedildi.")
}

// handleGazeteDepolamaTest POST /admin/gazete/depolama/test — bağlantı testi.
func (s *Server) handleGazeteDepolamaTest(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionLicense) {
		return
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	st, err := repo.GetSiteStorage(ctx, s.pool, sid)
	if err != nil {
		s.gazeteResult(w, r, "error", "Yapılandırma okunamadı.")
		return
	}
	if st.Provider == "" || st.Provider == "local" {
		s.gazeteResult(w, r, "info", "Yerel disk depolaması aktif; test gerekmiyor.")
		return
	}
	secret := ""
	if box := s.secretBox(); box != nil && st.SecretEnc != "" {
		secret, _ = box.Open(st.SecretEnc)
	}
	store, err := platform.NewObjectStore(platform.StoreConfig{
		Provider: st.Provider, Endpoint: st.Endpoint, Bucket: st.Bucket, Region: st.Region,
		AccessKeyID: st.AccessKeyID, Secret: secret,
		PublicBaseURL: st.PublicBaseURL, PathPrefix: st.PathPrefix,
	}, platform.StorageDir())
	if err != nil {
		s.gazeteResult(w, r, "error", "Depolama kurulamadı: "+err.Error())
		return
	}
	key := platform.GazeteKey(sid, "test", "baglanti-testi.txt")
	if _, err := store.Put(key, []byte("theblock gazete depolama testi"), "text/plain"); err != nil {
		s.gazeteResult(w, r, "error", "Yazma testi başarısız: "+err.Error())
		return
	}
	data, err := store.Get(key)
	_ = store.Delete(key)
	if err != nil || string(data) != "theblock gazete depolama testi" {
		s.gazeteResult(w, r, "error", "Okuma testi başarısız.")
		return
	}
	s.gazeteResult(w, r, "info", "Bağlantı testi başarılı (yaz/oku/sil).")
}

// storeForSite sitenin nesne deposunu çözer (hata/eksikte yerel disk).
func (s *Server) storeForSite(ctx context.Context, siteID int64) platform.ObjectStore {
	st, err := repo.GetSiteStorage(ctx, s.pool, siteID)
	if err != nil || !st.Enabled {
		store, _ := platform.NewObjectStore(platform.StoreConfig{Provider: "local"}, platform.StorageDir())
		return store
	}
	secret := ""
	if box := s.secretBox(); box != nil && st.SecretEnc != "" {
		secret, _ = box.Open(st.SecretEnc)
	}
	store, err := platform.NewObjectStore(platform.StoreConfig{
		Provider: st.Provider, Endpoint: st.Endpoint, Bucket: st.Bucket, Region: st.Region,
		AccessKeyID: st.AccessKeyID, Secret: secret,
		PublicBaseURL: st.PublicBaseURL, PathPrefix: st.PathPrefix,
	}, platform.StorageDir())
	if err != nil {
		store, _ = platform.NewObjectStore(platform.StoreConfig{Provider: "local"}, platform.StorageDir())
	}
	return store
}

// ---- Ekip & yetkiler ----

// handleGazeteEkip POST /admin/gazete/ekip — kullanıcıya gazete rolü ata.
func (s *Server) handleGazeteEkip(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionTeam) {
		return
	}
	sid := s.curSiteID(r)
	userID := parseID(r.FormValue("user_id"))
	rol := strings.TrimSpace(r.FormValue("rol"))
	if userID == 0 || (rol != "sorumlu" && rol != "editor" && rol != "izleyici") {
		s.gazeteResult(w, r, "error", "Kullanıcı ve geçerli bir rol seçin.")
		return
	}
	if err := repo.SetGazeteTeamRole(r.Context(), s.pool, sid, userID, rol); err != nil {
		s.gazeteResult(w, r, "error", "Atama kaydedilemedi.")
		return
	}
	s.audit(r, "gazete_ekip", "user:"+strconv.FormatInt(userID, 10)+"="+rol)
	s.gazeteResult(w, r, "info", "Ekip ataması kaydedildi.")
}

// handleGazeteEkipSil POST /admin/gazete/ekip/{user_id}/sil — atamayı kaldır.
func (s *Server) handleGazeteEkipSil(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionTeam) {
		return
	}
	userID := parseID(r.PathValue("user_id"))
	if err := repo.RemoveGazeteTeamRole(r.Context(), s.pool, s.curSiteID(r), userID); err != nil {
		s.gazeteResult(w, r, "error", "Kaldırılamadı.")
		return
	}
	s.audit(r, "gazete_ekip_sil", strconv.FormatInt(userID, 10))
	s.gazeteResult(w, r, "info", "Atama kaldırıldı.")
}

// ---- Şablonlar ----

// handleGazeteSablonKaydet POST /admin/gazete/sablonlar — şablon oluştur/güncelle.
func (s *Server) handleGazeteSablonKaydet(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionTemplate) {
		return
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	id := parseID(r.FormValue("id"))
	pages := []byte(strings.TrimSpace(r.FormValue("sayfalar")))
	if !json.Valid(pages) {
		pages = []byte("[]")
	}
	t := &repo.GazeteTemplate{
		SiteID:      sid,
		Name:        strings.TrimSpace(r.FormValue("ad")),
		Description: strings.TrimSpace(r.FormValue("aciklama")),
		Pages:       pages,
		ThumbURL:    strings.TrimSpace(r.FormValue("kucuk")),
		IsDefault:   r.FormValue("varsayilan") == "1",
	}
	if t.Name == "" {
		s.gazeteResult(w, r, "error", "Şablon adı gerekli.")
		return
	}
	if id > 0 {
		t.ID = id
		if err := repo.UpdateGazeteTemplate(ctx, s.pool, t); err != nil {
			s.gazeteResult(w, r, "error", "Güncellenemedi.")
			return
		}
	} else {
		u := userFromContext(ctx)
		t.CreatedBy = &u.ID
		newID, err := repo.CreateGazeteTemplate(ctx, s.pool, t)
		if err != nil {
			s.gazeteResult(w, r, "error", "Oluşturulamadı.")
			return
		}
		t.ID = newID
	}
	s.audit(r, "gazete_sablon", t.Name)
	s.gazeteResult(w, r, "info", "Şablon kaydedildi.")
}

// handleGazeteSablonSil POST /admin/gazete/sablonlar/{id}/sil.
func (s *Server) handleGazeteSablonSil(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionTemplate) {
		return
	}
	if err := repo.DeleteGazeteTemplate(r.Context(), s.pool, parseID(r.PathValue("id"))); err != nil {
		s.gazeteResult(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "gazete_sablon_sil", r.PathValue("id"))
	s.gazeteResult(w, r, "info", "Şablon silindi.")
}

// ---- Sayılar (baskılar) ----

// handleGazeteSayiYeni POST /admin/gazete/sayilar — sayı oluştur (otomatik doldurma ile).
func (s *Server) handleGazeteSayiYeni(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	date := strings.TrimSpace(r.FormValue("tarih"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	period := strings.TrimSpace(r.FormValue("periyot"))
	if period == "" {
		period = "gunluk"
	}
	iss := &repo.GazeteIssue{
		SiteID:    sid,
		Period:    period,
		IssueDate: date,
		Title:     strings.TrimSpace(r.FormValue("baslik")),
		Status:    "taslak",
	}
	// Sayi her zaman bir plana bagli olmali (SPEC Faz 3): secim yapilmadiysa
	// sitenin varsayilan sablonu kullanilir, galeri bossa once tohumlanir.
	if tid := parseID(r.FormValue("sablon_id")); tid > 0 {
		iss.TemplateID = &tid
	} else {
		iss.TemplateID = s.gazeteVarsayilanSablonID(ctx, sid)
	}
	if iss.TemplateID == nil {
		s.gazeteResult(w, r, "error", "Sayi olusturulamadi: hic sablon yok. Once sablon galerisinden hazir peyzajlari yukleyin.")
		return
	}
	id, err := repo.UpsertGazeteIssue(ctx, s.pool, iss)
	if err != nil {
		s.logger.Error("gazete sayı", "error", err)
		s.gazeteResult(w, r, "error", "Sayı oluşturulamadı.")
		return
	}
	if r.FormValue("doldur") == "1" {
		if err := s.gazeteAutoFill(ctx, sid, id); err != nil {
			s.logger.Error("gazete doldurma", "error", err)
		}
	}
	s.audit(r, "gazete_sayi", date)
	if IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/admin/gazete/sayilar/"+strconv.FormatInt(id, 10)+"/duzenle")
		return
	}
	s.flashRedirect(w, r, "/admin/gazete/sayilar/"+strconv.FormatInt(id, 10)+"/duzenle", "info", "Sayı oluşturuldu.")
}

// handleGazeteSayiDurum POST /admin/gazete/sayilar/{id}/durum — taslak→hazir→yayinda→arsiv.
func (s *Server) handleGazeteSayiDurum(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionPublish) {
		return
	}
	id := parseID(r.PathValue("id"))
	status := strings.TrimSpace(r.FormValue("durum"))
	switch status {
	case "taslak", "hazir", "yayinda", "arsiv":
	default:
		s.gazeteResult(w, r, "error", "Geçersiz durum.")
		return
	}
	if err := repo.SetGazeteIssueStatus(r.Context(), s.pool, id, status); err != nil {
		s.gazeteResult(w, r, "error", "Durum güncellenemedi.")
		return
	}
	s.audit(r, "gazete_durum", strconv.FormatInt(id, 10)+"="+status)
	s.gazeteResult(w, r, "info", "Durum güncellendi: "+status)
}

// handleGazeteSayiAtama POST /admin/gazete/sayilar/{id}/atama — sorumlu editör ata.
func (s *Server) handleGazeteSayiAtama(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	id := parseID(r.PathValue("id"))
	editorID := parseID(r.FormValue("user_id"))
	if err := repo.SetGazeteIssueEditor(r.Context(), s.pool, id, editorID); err != nil {
		s.gazeteResult(w, r, "error", "Atama kaydedilemedi.")
		return
	}
	s.audit(r, "gazete_atama", strconv.FormatInt(id, 10))
	s.gazeteResult(w, r, "info", "Sorumlu editör güncellendi.")
}

// handleGazeteSayiSil POST /admin/gazete/sayilar/{id}/sil.
func (s *Server) handleGazeteSayiSil(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	if err := repo.DeleteGazeteIssue(r.Context(), s.pool, parseID(r.PathValue("id"))); err != nil {
		s.gazeteResult(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "gazete_sayi_sil", r.PathValue("id"))
	s.gazeteResult(w, r, "info", "Sayı silindi.")
}

// handleGazeteSayiDoldur POST /admin/gazete/sayilar/{id}/doldur — otomatik doldurma (yenile).
func (s *Server) handleGazeteSayiDoldur(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	id := parseID(r.PathValue("id"))
	if err := s.gazeteAutoFill(r.Context(), s.curSiteID(r), id); err != nil {
		s.logger.Error("gazete doldurma", "error", err)
		s.gazeteResult(w, r, "error", "Doldurma hatası: "+err.Error())
		return
	}
	s.audit(r, "gazete_doldur", strconv.FormatInt(id, 10))
	if IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/admin/gazete/sayilar/"+strconv.FormatInt(id, 10)+"/duzenle")
		return
	}
	s.flashRedirect(w, r, "/admin/gazete/sayilar/"+strconv.FormatInt(id, 10)+"/duzenle", "info", "Otomatik doldurma tamamlandı.")
}

// handleGazeteSayfaKaydet POST /admin/gazete/sayilar/{id}/sayfa — editörden sayfa bloklarını kaydet.
func (s *Server) handleGazeteSayfaKaydet(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	id := parseID(r.PathValue("id"))
	pageNo, _ := strconv.Atoi(r.FormValue("sayfa"))
	if pageNo < 1 || pageNo > 50 {
		http.Error(w, "geçersiz sayfa", http.StatusBadRequest)
		return
	}
	blocks := strings.TrimSpace(r.FormValue("bloklar"))
	if len(blocks) > 512*1024 {
		http.Error(w, "blok verisi çok büyük", http.StatusRequestEntityTooLarge)
		return
	}
	if !json.Valid([]byte(blocks)) {
		http.Error(w, "geçersiz JSON", http.StatusBadRequest)
		return
	}
	if err := repo.SaveGazetePage(r.Context(), s.pool, id, pageNo, "", []byte(blocks)); err != nil {
		http.Error(w, "kayıt hatası", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// ---- Aboneler ----

// handleGazeteAboneEkle POST /admin/gazete/aboneler — abone ekle (KVKK opt-in kaydı).
func (s *Server) handleGazeteAboneEkle(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionSubs) {
		return
	}
	sub := &repo.GazeteSubscription{
		SiteID:    s.curSiteID(r),
		Email:     strings.TrimSpace(r.FormValue("email")),
		Phone:     strings.TrimSpace(r.FormValue("telefon")),
		Channel:   strings.TrimSpace(r.FormValue("kanal")),
		Frequency: strings.TrimSpace(r.FormValue("siklik")),
		Status:    "aktif",
		OptInIP:   r.RemoteAddr,
	}
	if sub.Channel == "whatsapp" && sub.Phone == "" {
		s.gazeteResult(w, r, "error", "WhatsApp abonesi için telefon gerekli.")
		return
	}
	if sub.Channel == "sms" && sub.Phone == "" {
		s.gazeteResult(w, r, "error", "SMS abonesi için telefon gerekli.")
		return
	}
	if sub.Channel != "whatsapp" && sub.Channel != "sms" && sub.Email == "" {
		s.gazeteResult(w, r, "error", "E-posta gerekli.")
		return
	}
	if _, err := repo.CreateGazeteSubscription(r.Context(), s.pool, sub); err != nil {
		msg := "Eklenemedi."
		if strings.Contains(err.Error(), "uq_gazete_subs") {
			msg = "Bu e-posta/telefon zaten kayıtlı."
		}
		s.gazeteResult(w, r, "error", msg)
		return
	}
	s.audit(r, "gazete_abone", sub.Email+sub.Phone)
	s.gazeteResult(w, r, "info", "Abone eklendi.")
}

// handleGazeteAboneSil POST /admin/gazete/aboneler/{id}/sil — KVKK silme.
func (s *Server) handleGazeteAboneSil(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionSubs) {
		return
	}
	if err := repo.DeleteGazeteSubscription(r.Context(), s.pool, parseID(r.PathValue("id"))); err != nil {
		s.gazeteResult(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "gazete_abone_sil", r.PathValue("id"))
	s.gazeteResult(w, r, "info", "Abone silindi.")
}

// handleGazeteAboneDurum POST /admin/gazete/aboneler/{id}/durum — aktif/iptal.
func (s *Server) handleGazeteAboneDurum(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionSubs) {
		return
	}
	status := strings.TrimSpace(r.FormValue("durum"))
	if status != "aktif" && status != "iptal" {
		s.gazeteResult(w, r, "error", "Geçersiz durum.")
		return
	}
	if err := repo.SetGazeteSubscriptionStatus(r.Context(), s.pool, parseID(r.PathValue("id")), status); err != nil {
		s.gazeteResult(w, r, "error", "Güncellenemedi.")
		return
	}
	s.gazeteResult(w, r, "info", "Abone durumu güncellendi.")
}

// handleGazeteAyar POST /admin/gazete/ayar — otomasyon & WhatsApp ayarları.
func (s *Server) handleGazeteAyar(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionLicense) {
		return
	}
	ctx := r.Context()
	set := func(section, key, val string) {
		_ = repo.SetSetting(ctx, s.pool, section, key, []byte(strings.TrimSpace(val)))
	}
	set("gazete", "saat", r.FormValue("saat"))
	set("gazete", "otomatik_yayin", r.FormValue("otomatik_yayin"))
	set("gazete", "haftalik_gun", r.FormValue("haftalik_gun"))
	set("gazete", "aylik_gun", r.FormValue("aylik_gun"))
	set("whatsapp", "token", r.FormValue("wa_token"))
	set("whatsapp", "phone_id", r.FormValue("wa_phone"))
	set("whatsapp", "api_version", r.FormValue("wa_version"))
	set("whatsapp", "template", r.FormValue("wa_template"))
	// SMS (Netgsm / Twilio): parolalar boş bırakılırsa korunur; sağlayıcı
	// KAPATILIRSA saklanan sırlar da silinir (kapatma = kimlik bırakma).
	smsProvider := strings.TrimSpace(r.FormValue("sms_provider"))
	set("sms", "provider", smsProvider)
	set("sms", "from", r.FormValue("sms_from"))
	set("sms", "netgsm_user", r.FormValue("sms_netgsm_user"))
	if smsProvider == "" {
		set("sms", "netgsm_password", "")
		set("sms", "twilio_token", "")
	} else if p := strings.TrimSpace(r.FormValue("sms_netgsm_password")); p != "" {
		set("sms", "netgsm_password", p)
	}
	set("sms", "twilio_sid", r.FormValue("sms_twilio_sid"))
	if smsProvider != "" {
		if p := strings.TrimSpace(r.FormValue("sms_twilio_token")); p != "" {
			set("sms", "twilio_token", p)
		}
	}
	s.audit(r, "gazete_ayar", "otomasyon+whatsapp+sms")
	s.gazeteResult(w, r, "info", "Ayarlar kaydedildi.")
}

// secretBox uygulama gizli anahtarından AES-GCM kutusu üretir (settings 'guvenlik'.'anahtar').
func (s *Server) secretBox() *platform.SecretBox {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := repo.GetSetting(ctx, s.pool, "guvenlik", "anahtar")
	key := strings.TrimSpace(string(raw))
	if err != nil || key == "" {
		key = platform.RandomHex(32)
		_ = repo.SetSetting(ctx, s.pool, "guvenlik", "anahtar", []byte(key))
	}
	box, err := platform.NewSecretBox([]byte(key))
	if err != nil {
		return nil
	}
	return box
}
