package httpapi

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// renderTemplPanel templ panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderTemplPanel(w http.ResponseWriter, r *http.Request, comp templ.Component) {
	var sb strings.Builder
	if err := comp.Render(r.Context(), &sb); err != nil {
		s.logger.Error("templ panel", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request")
	_, _ = w.Write([]byte(sb.String()))
}

// renderTemplPage templ panelini tam sayfa (admin kabuğu) içinde yazar.
func (s *Server) renderTemplPage(w http.ResponseWriter, r *http.Request, d *tmplData, comp templ.Component) {
	var sb strings.Builder
	if err := comp.Render(r.Context(), &sb); err != nil {
		s.logger.Error("templ sayfa", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	d.SlotHTML = template.HTML(sb.String())
	s.render(w, r, "admin_bilesenler", d)
}

func flashTo(f *flashMsg) (kind, text string) {
	if f != nil {
		return f.Kind, f.Text
	}
	return "", ""
}

// ---- Yorumlar ----

func yorumlarViewFrom(c *commentsData, f *flashMsg) templkit.YorumlarView {
	kind, text := flashTo(f)
	v := templkit.YorumlarView{FlashKind: kind, FlashText: text}
	if c == nil {
		return v
	}
	v.Tab = c.Tab
	v.Pending = c.Pending
	v.Ara = c.Ara
	for _, it := range c.Items {
		v.Items = append(v.Items, templkit.YorumRow{
			ID:           it.ID,
			ContentID:    it.ContentID,
			AuthorName:   it.AuthorName,
			ContentTitle: it.ContentTitle,
			Time:         it.CreatedAt.Local().Format("02.01.2006 15:04"),
			Onayli:       it.Status == "onayli",
			Body:         it.Body,
		})
	}
	return v
}

// ---- Geri Bildirim & Asistan ----

func geriBildirimViewFrom(fb *feedbackData, f *flashMsg) templkit.GeriBildirimView {
	kind, text := flashTo(f)
	v := templkit.GeriBildirimView{
		FlashKind: kind,
		FlashText: text,
		Bekliyor:  strconv.Itoa(fb.Counts["bekliyor"]),
		Cozuldu:   strconv.Itoa(fb.Counts["cozuldu"]),
		AsstCnt:   strconv.Itoa(fb.AsstCnt),
	}
	for _, it := range fb.Items {
		v.Items = append(v.Items, templkit.GBR{
			ID:         it.ID,
			ContentID:  it.ContentID,
			Kind:       it.Kind,
			Title:      it.Title,
			Status:     it.Status,
			Time:       it.CreatedAt.Format("2006-01-02 15:04"),
			Detail:     it.Detail,
			Reporter:   it.Reporter,
			Resolution: it.Resolution,
		})
	}
	for _, l := range fb.Logs {
		v.Logs = append(v.Logs, templkit.AsistLog{
			Time:     l.CreatedAt.Format("2006-01-02 15:04"),
			Provider: l.Provider,
			Question: l.Question,
			Answer:   l.Answer,
		})
	}
	return v
}

// ---- Üyelikler ----

func (s *Server) uyelerViewFrom(r *http.Request, m []repo.MemberRow, f *flashMsg, stripe map[string]string) templkit.UyelerView {
	kind, text := flashTo(f)
	v := templkit.UyelerView{FlashKind: kind, FlashText: text}
	v.Ara = strings.TrimSpace(r.URL.Query().Get("q"))
	if stripe != nil {
		v.StripeSecret = stripe["secret_key"]
		v.StripeSecretSet = v.StripeSecret != ""
		v.StripeWebhook = stripe["webhook_secret"]
		v.StripeWebhookSet = v.StripeWebhook != ""
		v.StripePrice = stripe["price_id"]
		v.StripeAktif = stripe["aktif"] == "1"
	}
	if google := s.googleSettings(r); google != nil {
		v.GoogleClientID = google["client_id"]
		v.GoogleClientSecret = google["client_secret"]
		v.GoogleSecretSet = v.GoogleClientSecret != ""
	}
	v.KayitAcik = s.uyeKayitAcik(r.Context())
	// iyzico ayarları.
	if iz := s.iyzicoSettings(r); iz != nil {
		v.IyzicoKey = iz["api_key"]
		v.IyzicoSecretSet = iz["secret_key"] != ""
		v.IyzicoMode = iz["mode"]
		if v.IyzicoMode == "" {
			v.IyzicoMode = "sandbox"
		}
		v.IyzicoAktif = iz["aktif"] == "1" || iz["aktif"] == "true"
	}
	// Katmanlar + ödeme panosu.
	tiers, _ := repo.ListTiers(r.Context(), s.pool, false)
	for _, t := range tiers {
		v.Tiers = append(v.Tiers, templkit.UyeTier{
			ID: t.ID, Slug: t.Slug, Name: t.Name, Description: t.Description,
			ColorHex: t.ColorHex, Weight: t.Weight,
			Monthly: trimMoney(t.MonthlyPrice), Annual: trimMoney(t.AnnualPrice),
			Perks: strings.Join(t.Perks, "\n"), Active: t.IsActive,
		})
	}
	stats, _ := repo.SubscriptionSummary(r.Context(), s.pool)
	v.StatActive = stats.ActiveSubs
	v.StatRev30 = trimMoney(stats.Revenue30)
	v.StatRevY = trimMoney(stats.RevenueYear)
	v.StatFailed = stats.Failed30
	for _, t := range tiers {
		if n, ok := stats.TierCounts[t.Name]; ok {
			v.TierCounts = append(v.TierCounts, templkit.UyeTierCount{Name: t.Name, Count: n})
		}
	}
	payments, _ := repo.ListPayments(r.Context(), s.pool, 25)
	for _, p := range payments {
		v.Payments = append(v.Payments, templkit.UyePayment{
			ID: p.ID, UserName: p.UserName, TierName: p.TierName,
			Provider: p.Provider, Status: p.Status, Amount: trimMoney(p.Amount),
			Cycle: cycleTR(p.BillingCycle), Created: p.CreatedAt.Format("02.01.2006 15:04"),
		})
	}
	// Ekonomi metrikleri (Faz 5).
	if ec, err := repo.SubscriptionEconomy(r.Context(), s.pool); err == nil {
		v.EconMRR = trimMoney(ec.MRR)
		v.EconARR = trimMoney(ec.ARR)
		v.EconARPU = trimMoney(ec.ARPU)
		v.EconChurn = trimMoney(ec.Churn30)
		v.EconLTV = trimMoney(ec.LTV)
		v.EconRenewal = trimMoney(ec.Renewal30)
		v.EconRefund = trimMoney(ec.Refunded30)
		v.EconNew = ec.NewSubs30
		v.EconCancel = ec.Canceled30
	}
	promos, _ := repo.ListPromos(r.Context(), s.pool)
	for _, p := range promos {
		uses := itoa(p.UsedCount)
		if p.MaxUses > 0 {
			uses += " / " + itoa(p.MaxUses)
		}
		expires := "—"
		if p.ExpiresAt != nil {
			expires = p.ExpiresAt.Format("02.01.2006")
		}
		v.Promos = append(v.Promos, templkit.UyePromo{
			ID: p.ID, Code: p.Code, Percent: p.Percent, TierSlug: p.TierSlug,
			Uses: uses, Expires: expires, Active: p.IsActive,
		})
	}
	// Aylık gelir trendi (son 12 ay) — çubuk yüksekliği en yüksek aya göre.
	if trend, err := repo.RevenueTrend(r.Context(), s.pool, 12); err == nil {
		var max float64
		for _, m := range trend {
			if m.Amount > max {
				max = m.Amount
			}
		}
		v.TrendHasData = max > 0
		for _, m := range trend {
			pct := 0
			if max > 0 && m.Amount > 0 {
				pct = int(m.Amount/max*100 + 0.5)
				if pct < 4 {
					pct = 4 // sıfır olmayan ay görünür kalsın
				}
			}
			v.Trend = append(v.Trend, templkit.UyeTrend{
				Short: m.Short, Label: m.Label, Amount: trimMoney(m.Amount),
				Count: m.Count, Pct: pct,
			})
		}
	}
	for _, u := range m {
		v.Items = append(v.Items, templkit.UyeRow{
			ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role,
			Membership: u.Membership, Premium: u.Membership == "premium",
			TierName: u.TierName, TierColor: u.TierColor, Until: u.Until, Cycle: u.Cycle,
		})
	}
	return v
}

// trimMoney 150.00 → "150", 150.50 → "150,50".
func trimMoney(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimSuffix(s, ".00")
	return strings.ReplaceAll(s, ".", ",")
}

// ---- Mahreçler ----

func mahreclerViewFrom(m *mahretsData, f *flashMsg) templkit.MahreclerView {
	kind, text := flashTo(f)
	v := templkit.MahreclerView{FlashKind: kind, FlashText: text}
	if m != nil {
		for _, it := range m.Items {
			v.Items = append(v.Items, templkit.MahrecRow{ID: it.ID, Name: it.Name})
		}
	}
	return v
}
