package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Gazete bülten dağıtımı (Faz 6) + günlük otomasyon (Faz 9) ----

// gazeteWhatsAppConfig ayarlardan (section=whatsapp) Cloud API yapılandırmasını okur.
func (s *Server) gazeteWhatsAppConfig(ctx context.Context) platform.WhatsAppConfig {
	get := func(key string) string {
		raw, err := repo.GetSetting(ctx, s.pool, "whatsapp", key)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(raw))
	}
	return platform.WhatsAppConfig{
		Token:      get("token"),
		PhoneID:    get("phone_id"),
		APIVersion: get("api_version"),
		Template:   get("template"),
	}
}

// gazeteSMSConfig ayarlardan (section=sms) SMS yapılandırmasını okur.
func (s *Server) gazeteSMSConfig(ctx context.Context) platform.SMSConfig {
	get := func(key string) string {
		raw, err := repo.GetSetting(ctx, s.pool, "sms", key)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(raw))
	}
	return platform.SMSConfig{
		Provider:       get("provider"),
		From:           get("from"),
		NetgsmUser:     get("netgsm_user"),
		NetgsmPassword: get("netgsm_password"),
		TwilioSID:      get("twilio_sid"),
		TwilioToken:    get("twilio_token"),
	}
}

// gazetePDFURL baskının genel PDF adresini üretir (e-posta/WhatsApp bağlantısı).
func gazetePDFURL(domain string, iss repo.GazeteIssue) string {
	scheme := "https"
	return scheme + "://" + domain + "/gazete/" + iss.IssueDate + "/pdf?periyot=" + iss.Period
}

// gazeteFrequencyMatch abone sıklığı ile baskı tarihinin eşleşip eşleşmediğini belirler.
// gunluk → her gün; haftalik → haftanın belirli günü; aylik → ayın belirli günü.
func gazeteFrequencyMatch(freq string, issueDate time.Time, weeklyDay, monthlyDay int) bool {
	switch freq {
	case "gunluk":
		return true
	case "haftalik":
		return int(issueDate.Weekday()) == weeklyDay
	case "aylik":
		return issueDate.Day() == monthlyDay
	}
	return false
}

// gazeteDeliverIssue baskıyı uygun abonelere iletir (idempotent).
// Dönen: gönderilen adet, hatalı adet.
func (s *Server) gazeteDeliverIssue(ctx context.Context, siteID, issueID int64, allFrequencies bool) (int, int) {
	iss, err := repo.GetGazeteIssue(ctx, s.pool, issueID)
	if err != nil {
		return 0, 0
	}
	domain := "localhost"
	if site, e := repo.GetSite(ctx, s.pool, siteID); e == nil && site.Domain != "" {
		domain = site.Domain
	}
	pdfURL := gazetePDFURL(domain, iss)

	// PDF yoksa üret.
	var pdfData []byte
	if iss.PDFPath != "" {
		pdfData, _ = s.storeForSite(ctx, siteID).Get(iss.PDFPath)
	}
	if pdfData == nil {
		if _, e := s.gazeteGeneratePDF(ctx, siteID, issueID); e == nil {
			if iss2, e2 := repo.GetGazeteIssue(ctx, s.pool, issueID); e2 == nil {
				iss = iss2
				pdfData, _ = s.storeForSite(ctx, siteID).Get(iss.PDFPath)
			}
		}
	}

	subs, _ := repo.ListGazeteSubscriptions(ctx, s.pool, siteID, "aktif", 1000)
	mailCfg := s.smtpConfig(ctx)
	waCfg := s.gazeteWhatsAppConfig(ctx)
	smsCfg := s.gazeteSMSConfig(ctx)
	weeklyDay := s.gazeteSettingInt(ctx, "haftalik_gun", 1) // Pazartesi
	monthlyDay := s.gazeteSettingInt(ctx, "aylik_gun", 1)

	sent, failed := 0, 0
	for _, sub := range subs {
		if !allFrequencies {
			// Periyoda göre filtre: haftalık baskıya günlük+haftalık aboneler vb.
			match := gazeteFrequencyMatch(sub.Frequency, issueDateOf(iss), weeklyDay, monthlyDay)
			if iss.Period == "haftalik" {
				match = sub.Frequency == "gunluk" || sub.Frequency == "haftalik"
			}
			if iss.Period == "aylik" {
				match = true
			}
			if !match {
				continue
			}
		}
		// İdempotent: bu baskı bu aboneye gönderilmişse geç.
		if s.gazeteAlreadyDelivered(ctx, issueID, sub.ID) {
			continue
		}
		d := &repo.GazeteDelivery{IssueID: issueID, SubscriptionID: sub.ID, Channel: sub.Channel, Status: "kuyrukta"}
		var sendErr error
		switch sub.Channel {
		case "whatsapp":
			sendErr = waCfg.SendDocument(sub.Phone,
				"Bugünün gazetesi hazır: "+pdfURL, pdfURL, "gazete-"+iss.IssueDate+".pdf")
		case "sms":
			// SMS'e PDF sığmaz; okuma + çıkış bağlantıları kısa metinle gider.
			sendErr = smsCfg.Send(ctx, sub.Phone,
				"Bugünün gazetesi hazır: "+pdfURL+
					" · Abonelikten çık: https://"+domain+"/gazete/abonelik/iptal/"+sub.Token)
		default:
			d.Channel = "email"
			body := `<p>Merhaba,</p><p>Bugünün gazetesi hazır. <a href="` + pdfURL + `">Çevrimiçi okuyun</a> veya ekteki PDF'i indirin.</p>` +
				`<p><a href="https://` + domain + `/gazete">Gazete Arşivi</a> · <a href="https://` + domain + `/gazete/abonelik/iptal/` + sub.Token + `">Abonelikten çık</a></p>`
			if pdfData != nil {
				sendErr = mailCfg.SendMailAttach(sub.Email, "Gazeteniz — "+iss.IssueDate, body,
					"gazete-"+iss.IssueDate+".pdf", pdfData)
			} else {
				sendErr = mailCfg.SendMail(sub.Email, "Gazeteniz — "+iss.IssueDate, body)
			}
		}
		if sendErr != nil {
			d.Status = "hata"
			d.Error = firstLine(sendErr.Error(), 200)
			failed++
		} else {
			d.Status = "gonderildi"
			now := time.Now()
			d.SentAt = &now
			sent++
		}
		_, _ = repo.UpsertGazeteDelivery(ctx, s.pool, d)
	}
	return sent, failed
}

// gazeteAlreadyDelivered aboneye bu baskı gönderilmiş mi?
func (s *Server) gazeteAlreadyDelivered(ctx context.Context, issueID, subID int64) bool {
	var exists bool
	_ = s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM gazete_deliveries
		WHERE issue_id=$1 AND subscription_id=$2 AND status='gonderildi')`, issueID, subID).Scan(&exists)
	return exists
}

// gazeteSettingInt gazete ayarını int okur (yoksa varsayılan).
func (s *Server) gazeteSettingInt(ctx context.Context, key string, def int) int {
	raw, err := repo.GetSetting(ctx, s.pool, "gazete", key)
	if err != nil {
		return def
	}
	if n, e := strconv.Atoi(strings.TrimSpace(string(raw))); e == nil && n > 0 {
		return n
	}
	return def
}

// gazeteSettingString gazete ayarını string okur.
func (s *Server) gazeteSettingString(ctx context.Context, key, def string) string {
	raw, err := repo.GetSetting(ctx, s.pool, "gazete", key)
	if err != nil {
		return def
	}
	v := strings.TrimSpace(string(raw))
	if v == "" {
		return def
	}
	return v
}

func issueDateOf(iss repo.GazeteIssue) time.Time {
	t, _ := time.Parse("2006-01-02", iss.IssueDate)
	return t
}

// ---- Günlük otomasyon ----

// gazeteDailyJob lisanslı tüm siteler için günün baskısını üretir ve dağıtır.
// Ayarlar (settings 'gazete'): saat (HH:MM, varsayılan 07:00), otomatik_yayin (0/1).
func (s *Server) gazeteDailyJob(ctx context.Context) {
	sites, err := repo.ListSites(ctx, s.pool)
	if err != nil {
		return
	}
	for _, site := range sites {
		on, err := repo.SiteFeatureEnabled(ctx, s.pool, site.ID, "gazete")
		if err != nil || !on {
			continue
		}
		date := time.Now().Format("2006-01-02")
		iss, err := repo.GetGazeteIssueByDate(ctx, s.pool, site.ID, "gunluk", date)
		if err != nil {
			// Yeni sayı: şablon + otomatik doldurma.
			iss = repo.GazeteIssue{SiteID: site.ID, Period: "gunluk", IssueDate: date, Title: "Gazete — " + date, Status: "taslak"}
			id, err := repo.UpsertGazeteIssue(ctx, s.pool, &iss)
			if err != nil {
				s.logger.Error("gazete günlük sayı", "site", site.ID, "error", err)
				continue
			}
			iss.ID = id
			if err := s.gazeteAutoFill(ctx, site.ID, id); err != nil {
				s.logger.Error("gazete günlük doldurma", "site", site.ID, "error", err)
			}
		}
		// Otomatik yayın açık mı?
		if s.gazeteSettingString(ctx, "otomatik_yayin", "0") == "1" && iss.Status != "yayinda" {
			_ = repo.SetGazeteIssueStatus(ctx, s.pool, iss.ID, "yayinda")
		}
		// PDF üret + dağıt.
		sent, failed := s.gazeteDeliverIssue(ctx, site.ID, iss.ID, false)
		s.logger.Info("gazete günlük iş", "site", site.ID, "sayi", iss.ID, "gonderilen", sent, "hatali", failed)
	}
}

// RunGazeteJobs gazete otomasyonlarını belirli aralıkta değerlendirir.
// Ayarlanan saatte (settings 'gazete'.'saat') günün işi bir kez çalışır.
func (s *Server) RunGazeteJobs(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.maybeRunGazeteToday(ctx)
		}
	}
}

// maybeRunGazeteToday gün içinde planlanan saatte işi bir kez çalıştırır.
func (s *Server) maybeRunGazeteToday(ctx context.Context) {
	now := time.Now()
	want := s.gazeteSettingString(ctx, "saat", "07:00")
	if len(want) < 5 {
		want = "07:00"
	}
	nowHM := now.Format("15:04")
	if nowHM < want {
		return
	}
	today := now.Format("2006-01-02")
	last, _ := repo.GetSetting(ctx, s.pool, "gazete", "son_calisma")
	if strings.TrimSpace(string(last)) == today {
		return
	}
	_ = repo.SetSetting(ctx, s.pool, "gazete", "son_calisma", []byte(today))
	s.gazeteDailyJob(ctx)
}

// handleGazeteGonder POST /admin/gazete/sayilar/{id}/gonder — elle dağıtım tetikle.
func (s *Server) handleGazeteGonder(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionSubs) {
		return
	}
	id := parseID(r.PathValue("id"))
	all := r.FormValue("tumu") == "1"
	sent, failed := s.gazeteDeliverIssue(r.Context(), s.curSiteID(r), id, all)
	s.audit(r, "gazete_gonder", strconv.FormatInt(id, 10))
	s.gazeteResult(w, r, "info", "Dağıtım tamamlandı: "+strconv.Itoa(sent)+" gönderildi, "+strconv.Itoa(failed)+" hatalı.")
}
