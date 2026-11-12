package httpapi

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
	"theblock/site/internal/scheduler"
	"theblock/site/internal/trends"
)

// trendBriefView panelde gösterilen AI brifingidir.
type trendBriefView struct {
	Term repo.TrendTermRow
	Text string
}

// trendTermView panel satırı + sunucu tarafı sparkline'dır.
// Sparkline JS gerektirmez; SVG doğrudan HTML'e gömülür.
type trendTermView struct {
	repo.TrendTermRow
	SparkPath  string
	SparkTitle string
	SparkX     float64
	SparkY     float64
}

const (
	sparkWidth  = 100.0
	sparkHeight = 28.0
	sparkPad    = 3.0
)

// trendTermViews görünen terimlerin sparkline serilerini tek sorguda yükler.
func (s *Server) trendTermViews(ctx context.Context, rows []repo.TrendTermRow) []trendTermView {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	sparks, _ := repo.LoadTrendSparklines(ctx, s.pool, ids, time.Now().Add(-7*24*time.Hour))
	out := make([]trendTermView, 0, len(rows))
	for _, r := range rows {
		v := trendTermView{TrendTermRow: r}
		v.SparkPath, v.SparkTitle, v.SparkX, v.SparkY = trendSparkline(sparks[r.ID])
		out = append(out, v)
	}
	return out
}

// trendSparkline sıralama serisini 100x28 viewBox'ında SVG path'e çevirir.
// Sıra 1 en üsttedir; en az iki nokta yoksa çizgi üretilmez.
func trendSparkline(points []repo.TrendSparkPoint) (path, title string, lastX, lastY float64) {
	if len(points) < 2 {
		return "", "", 0, 0
	}
	minRank, maxRank := points[0].Rank, points[0].Rank
	for _, p := range points[1:] {
		if p.Rank < minRank {
			minRank = p.Rank
		}
		if p.Rank > maxRank {
			maxRank = p.Rank
		}
	}
	span := maxRank - minRank
	var b strings.Builder
	for i, p := range points {
		x := sparkPad + float64(i)*(sparkWidth-2*sparkPad)/float64(len(points)-1)
		y := sparkHeight / 2
		if span > 0 {
			y = sparkPad + (p.Rank-minRank)/span*(sparkHeight-2*sparkPad)
		}
		if i == 0 {
			b.WriteString("M")
		} else {
			b.WriteString(" L")
		}
		b.WriteString(strconv.FormatFloat(x, 'f', 1, 64))
		b.WriteByte(',')
		b.WriteString(strconv.FormatFloat(y, 'f', 1, 64))
		lastX, lastY = x, y
	}
	title = "Son 7 gün · " + strconv.Itoa(len(points)) + " nokta · en iyi sıra " +
		strconv.Itoa(int(math.Round(minRank))) + " · son sıra " +
		strconv.Itoa(int(math.Round(points[len(points)-1].Rank)))
	return b.String(), title, lastX, lastY
}

// fillTrendCenterData Bot sayfasına Trend Merkezi verilerini yükler.
func (s *Server) fillTrendCenterData(r *http.Request, bd *botData) {
	ctx := r.Context()
	siteID := s.curSiteID(r)
	q := r.URL.Query()
	bd.TCQ = strings.TrimSpace(q.Get("tq"))
	bd.TCFilter = strings.TrimSpace(q.Get("tf"))
	if n, err := strconv.Atoi(q.Get("tskor")); err == nil && n > 0 {
		bd.TCMinScore = n
	}
	bd.TCStats, _ = repo.GetTrendStats(ctx, s.pool, siteID)
	rows, _ := repo.ListTrendTerms(ctx, s.pool, repo.TrendFilter{
		SiteID: siteID, Q: bd.TCQ, OnlyGap: bd.TCFilter == "bosluk",
		OnlyRising: bd.TCFilter == "yukselen", MinScore: bd.TCMinScore, Limit: 50,
	})
	bd.TCItems = s.trendTermViews(ctx, rows)
	bd.TCSources, _ = repo.ListTrendSources(ctx, s.pool, siteID)
	bd.TCAlerts, _ = repo.ListTrendAlerts(ctx, s.pool, siteID, 8)
	bd.TCWeights = repo.TrendWeights(ctx, s.pool)
	bd.TCThreshold = repo.TrendAlertThreshold(ctx, s.pool)
	bd.TCPaid = repo.GetTrendSetting(ctx, s.pool, "odeme_aktif") == "true"
	bd.TCNotify = repo.GetTrendSetting(ctx, s.pool, "bildirim_aktif") != "false"
	bd.TCAuto = repo.GetTrendSetting(ctx, s.pool, "otomatik_taslak") == "true"
	bd.SearchTop, _ = repo.TopSearchQueries(ctx, s.pool, siteID, 12)
	bd.GSCTop, _ = repo.TopGSCQueries(ctx, s.pool, siteID, 28, 12)
	bd.GSCProperty = repo.GetTrendSetting(ctx, s.pool, "gsc_property")
	bd.GA4Property = repo.GetTrendSetting(ctx, s.pool, "ga4_property")
	bd.GA4Olcum = repo.GetTrendSetting(ctx, s.pool, "ga4_olcum")
	if b, err := repo.GetSetting(ctx, s.pool, "seo", "gsc_service_account"); err == nil && len(strings.TrimSpace(string(b))) > 4 {
		bd.GSCConfigured = true
		bd.GA4Configured = true
	}
	if b, err := repo.GetSetting(ctx, s.pool, "trend", "ga4_ozet"); err == nil {
		var sum scheduler.GA4Summary
		if json.Unmarshal(b, &sum) == nil && sum.Guncel != "" {
			bd.GA4 = &sum
		}
	}
	if id := parseID(q.Get("brifing")); id > 0 {
		if term, err := repo.GetTrendTerm(ctx, s.pool, id); err == nil {
			if text, err := repo.LatestTrendBrief(ctx, s.pool, id); err == nil {
				bd.TCBrief = &trendBriefView{Term: term, Text: text}
			}
		}
	}
}

// handleTrendCenterCek tüm aktif dış trend kaynaklarını hemen çeker.
func (s *Server) handleTrendCenterCek(w http.ResponseWriter, r *http.Request) {
	added, updated, err := scheduler.FetchAllTrendSourcesNow(r.Context(), s.pool, s.logger)
	msg := "Trend kaynakları çekildi: " + itoa(added) + " yeni, " + itoa(updated) + " güncel terim."
	kind := "info"
	if err != nil {
		msg += " (" + err.Error() + ")"
		kind = "error"
	}
	s.audit(r, "trend_merkezi_cek", itoa(added)+"/"+itoa(updated))
	s.botResult(w, r, kind, msg)
}

// handleTrendSourceEkle yeni dış trend kaynağı ekler.
func (s *Server) handleTrendSourceEkle(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	rawURL := strings.TrimSpace(r.FormValue("url"))
	if name == "" || rawURL == "" {
		s.botResult(w, r, "error", "Kaynak adı ve URL gerekli.")
		return
	}
	if u, err := url.Parse(rawURL); err != nil {
		s.botResult(w, r, "error", "Kaynak adresi geçersiz.")
		return
	} else if err := platform.ValidateOutboundURL(u); err != nil {
		s.botResult(w, r, "error", "Kaynak adresi kabul edilmedi: "+err.Error())
		return
	}
	kind := strings.TrimSpace(r.FormValue("kind"))
	if kind == "" {
		kind = "google_trends"
	}
	interval, _ := strconv.Atoi(r.FormValue("interval"))
	if interval < 300 {
		interval = 900
	}
	row := repo.TrendSourceRow{
		SiteID: s.curSiteID(r), Name: name, Kind: kind, URL: rawURL,
		Geo: strings.TrimSpace(r.FormValue("geo")), HL: strings.TrimSpace(r.FormValue("hl")),
		IntervalSeconds: interval, Aktif: r.FormValue("pasif") == "",
	}
	if row.Geo == "" {
		row.Geo = "TR"
	}
	if row.HL == "" {
		row.HL = "tr-TR"
	}
	if _, err := repo.CreateTrendSource(r.Context(), s.pool, row); err != nil {
		s.botResult(w, r, "error", "Kaynak eklenemedi: "+err.Error())
		return
	}
	s.audit(r, "trend_kaynak_ekle", name)
	s.botResult(w, r, "info", name+" eklendi.")
}

// handleTrendSourceCek tek kaynağı hemen çeker.
func (s *Server) handleTrendSourceCek(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	added, updated, err := scheduler.FetchTrendSourceNow(r.Context(), s.pool, s.logger, id)
	if err != nil {
		s.botResult(w, r, "error", "Kaynak çekilemedi: "+err.Error())
		return
	}
	s.botResult(w, r, "info", "Kaynak çekildi: "+itoa(added)+" yeni, "+itoa(updated)+" güncel.")
}

// handleTrendSourceDurum kaynağı açar/kapatır.
func (s *Server) handleTrendSourceDurum(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	aktif := r.FormValue("durum") == "1"
	if err := repo.SetTrendSourceAktif(r.Context(), s.pool, id, aktif); err != nil {
		s.botResult(w, r, "error", "Durum değiştirilemedi.")
		return
	}
	s.botResult(w, r, "info", "Kaynak durumu güncellendi.")
}

// handleTrendSourceSil kaynağı siler.
func (s *Server) handleTrendSourceSil(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteTrendSource(r.Context(), s.pool, id); err != nil {
		s.botResult(w, r, "error", "Kaynak silinemedi.")
		return
	}
	s.audit(r, "trend_kaynak_sil", itoa(int(id)))
	s.botResult(w, r, "info", "Kaynak silindi.")
}

// handleTrendBulkDraft içerik boşluğu olan en yüksek skorlu terimlerden
// toplu taslak üretir. Editör daha sonra taslakları gözden geçirir.
func (s *Server) handleTrendBulkDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := userFromContext(ctx)
	terms, err := repo.ListTrendTerms(ctx, s.pool, repo.TrendFilter{
		SiteID: s.curSiteID(r), OnlyGap: true, MinScore: 30, Limit: 3,
	})
	if err != nil {
		s.botResult(w, r, "error", "Trendler okunamadı: "+err.Error())
		return
	}
	if len(terms) == 0 {
		s.botResult(w, r, "error", "Uygun içerik boşluğu bulunamadı (skor ≥ 30).")
		return
	}
	author := s.botAuthorDefault(ctx)
	created := 0
	for _, term := range terms {
		news, _ := repo.TrendNewsForTerm(ctx, s.pool, term.ID, 3)
		input := term.Term
		for _, n := range news {
			if n.Title != "" {
				input += "\n" + n.Title
				break
			}
		}
		title, terr := s.runAIAction(r, "title", input, 0, u.ID)
		spot, serr := s.runAIAction(r, "spot", input, 0, u.ID)
		if terr != nil {
			title = ""
		}
		if serr != nil {
			spot = ""
		}
		title = cleanTrendTitle(title)
		spot = cleanTrendSpot(spot)
		if _, err := repo.TrendTermToContent(ctx, s.pool, term.ID, author, title, spot); err == nil {
			created++
		}
	}
	s.audit(r, "trend_toplu_taslak", itoa(created))
	s.botResult(w, r, "info", itoa(created)+" trend taslağı oluşturuldu. Taslaklar İçerikler > Taslaklar'da.")
}

// handleTrendTermGizle terimi panelden kaldırır.
func (s *Server) handleTrendTermGizle(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.SetTrendTermHidden(r.Context(), s.pool, id, true); err != nil {
		s.botResult(w, r, "error", "Terim gizlenemedi.")
		return
	}
	s.botResult(w, r, "info", "Terim gizlendi.")
}

// handleTrendTermBrifing terim için AI editör brifingi üretir.
func (s *Server) handleTrendTermBrifing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	term, err := repo.GetTrendTerm(ctx, s.pool, id)
	if err != nil {
		s.botResult(w, r, "error", "Terim bulunamadı.")
		return
	}
	news, _ := repo.TrendNewsForTerm(ctx, s.pool, id, 8)
	var b strings.Builder
	b.WriteString("Terim: " + term.Term + "\n")
	b.WriteString("Yaklaşık arama: " + itoa(term.ApproxTraffic) + "+\n")
	b.WriteString("Haberler:\n")
	for _, n := range news {
		if n.Title != "" {
			b.WriteString("- " + n.Title)
			if n.Source != "" {
				b.WriteString(" (" + n.Source + ")")
			}
			b.WriteString("\n")
		}
	}
	if len(news) == 0 {
		b.WriteString("- (haber öğesi yok)\n")
	}
	u := userFromContext(ctx)
	text, err := s.runAIAction(r, "trend_brief", b.String(), 0, u.ID)
	if err != nil {
		s.botResult(w, r, "error", "Brifing üretilemedi: "+err.Error())
		return
	}
	if _, err := repo.SaveTrendBrief(ctx, s.pool, id, text, "ai", u.ID); err != nil {
		s.logger.Warn("trend brifing kaydı", "error", err)
	}
	s.audit(r, "trend_brifing", itoa(int(id)))
	if IsHTMX(r) {
		d := s.buildBotData(r)
		d.Flash = &flashMsg{Kind: "info", Text: "AI brifingi hazır."}
		d.Bot.TCBrief = &trendBriefView{Term: term, Text: text}
		s.renderTrendPanelV2(w, r, d)
		return
	}
	s.flashRedirect(w, r, "/admin/bot?brifing="+itoa(int(id)), "info", "AI brifingi hazır.")
}

// handleTrendTermTaslak terimden AI destekli taslak içerik üretir.
func (s *Server) handleTrendTermTaslak(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	term, err := repo.GetTrendTerm(ctx, s.pool, id)
	if err != nil {
		s.botResult(w, r, "error", "Terim bulunamadı.")
		return
	}
	news, _ := repo.TrendNewsForTerm(ctx, s.pool, id, 5)
	input := term.Term
	for _, n := range news {
		if n.Title != "" {
			input += "\n" + n.Title
			break
		}
	}
	u := userFromContext(ctx)
	title, terr := s.runAIAction(r, "title", input, 0, u.ID)
	spot, serr := s.runAIAction(r, "spot", input, 0, u.ID)
	if terr != nil {
		title = ""
	}
	if serr != nil {
		spot = ""
	}
	title = cleanTrendTitle(title)
	spot = cleanTrendSpot(spot)
	nid, err := repo.TrendTermToContent(ctx, s.pool, id, s.botAuthorDefault(ctx), title, spot)
	if err != nil {
		s.botResult(w, r, "error", "Taslak oluşturulamadı: "+err.Error())
		return
	}
	s.audit(r, "trend_taslak", itoa(int(id)))
	if s.htmxRedirect(w, r, "/admin/icerikler/"+itoa(int(nid))) {
		return
	}
	s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(nid)), "info", "Trendden taslak oluşturuldu.")
}

// handleTrendAlertsSeen uyarıları okundu işaretler.
func (s *Server) handleTrendAlertsSeen(w http.ResponseWriter, r *http.Request) {
	if err := repo.MarkTrendAlertsSeen(r.Context(), s.pool, s.curSiteID(r)); err != nil {
		s.botResult(w, r, "error", "Uyarılar güncellenemedi.")
		return
	}
	s.botResult(w, r, "info", "Trend uyarıları okundu.")
}

// handleTrendSettings skor ağırlıkları ve uyarı eşiğini kaydeder.
func (s *Server) handleTrendSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	parse := func(key string) float64 {
		v, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue(key)), 64)
		if v < 0 {
			v = 0
		}
		return v
	}
	wgt := trends.Weights{
		Traffic: parse("w_trafik"), Freshness: parse("w_tazelik"),
		News: parse("w_haber"), Relevance: parse("w_alaka"), Demand: parse("w_talep"),
	}
	if err := repo.SaveTrendWeights(ctx, s.pool, wgt); err != nil {
		s.botResult(w, r, "error", "Ağırlıklar kaydedilemedi.")
		return
	}
	n, _ := strconv.Atoi(r.FormValue("uyari_esik"))
	if n < 1 || n > 100 {
		n = 70
	}
	_ = repo.SetTrendSetting(ctx, s.pool, "uyari_esik", itoa(n))
	auto := "false"
	if r.FormValue("otomatik_taslak") != "" {
		auto = "true"
	}
	_ = repo.SetTrendSetting(ctx, s.pool, "otomatik_taslak", auto)
	bildirim := "false"
	if r.FormValue("bildirim_aktif") != "" {
		bildirim = "true"
	}
	_ = repo.SetTrendSetting(ctx, s.pool, "bildirim_aktif", bildirim)
	odeme := "false"
	if r.FormValue("odeme_aktif") != "" {
		odeme = "true"
	}
	_ = repo.SetTrendSetting(ctx, s.pool, "odeme_aktif", odeme)
	s.audit(r, "trend_ayar", "eşik="+itoa(n))
	s.botResult(w, r, "info", "Trend ayarları kaydedildi.")
}

// handleTrendGSCSetting Search Console özelliğini kaydeder.
func (s *Server) handleTrendGSCSetting(w http.ResponseWriter, r *http.Request) {
	property := strings.TrimSpace(r.FormValue("gsc_property"))
	if err := repo.SetTrendSetting(r.Context(), s.pool, "gsc_property", property); err != nil {
		s.botResult(w, r, "error", "GSC özelliği kaydedilemedi.")
		return
	}
	s.botResult(w, r, "info", "Search Console özelliği kaydedildi.")
}

// handleTrendGSCSync Search Console verilerini hemen çeker.
func (s *Server) handleTrendGSCSync(w http.ResponseWriter, r *http.Request) {
	n, err := scheduler.SyncGSC(r.Context(), s.pool, s.logger, s.curSiteID(r))
	if err != nil {
		s.botResult(w, r, "error", "GSC senkronu: "+err.Error())
		return
	}
	s.audit(r, "trend_gsc_sync", itoa(n))
	s.botResult(w, r, "info", "Search Console senkronu tamam: "+itoa(n)+" sorgu satırı.")
}

// handleTrendGA4Setting GA4 özellik kimliğini kaydeder.
func (s *Server) handleTrendGA4Setting(w http.ResponseWriter, r *http.Request) {
	// Mülk kimliği: Data API için. Kullanıcı bazen "properties/561087489"
	// biçimini yapıştırıyor; normalizeGA4PropertyID sayı kısmını alır.
	rawProperty := strings.TrimSpace(r.FormValue("ga4_property"))
	property := normalizeGA4PropertyID(rawProperty)
	if rawProperty != "" && property == "" {
		s.botResult(w, r, "error", "GA4 mülk kimliği yalnız rakamlardan oluşmalı (örn. 561087489).")
		return
	}
	if err := repo.SetTrendSetting(r.Context(), s.pool, "ga4_property", property); err != nil {
		s.botResult(w, r, "error", "GA4 özelliği kaydedilemedi.")
		return
	}

	// Ölçüm kimliği: public sayfadaki gtag.js için. Boş bırakılırsa site
	// hiçbir izleme kodu basmaz — varsayılan izleme yoktur.
	rawOlcum := strings.TrimSpace(r.FormValue("ga4_olcum"))
	olcum := validGA4Measurement(rawOlcum)
	if rawOlcum != "" && olcum == "" {
		s.botResult(w, r, "error", "GA4 ölçüm kimliği G- ile başlamalı (örn. G-XK4YN7T0PE).")
		return
	}
	if err := repo.SetTrendSetting(r.Context(), s.pool, "ga4_olcum", olcum); err != nil {
		s.botResult(w, r, "error", "GA4 ölçüm kimliği kaydedilemedi.")
		return
	}
	// Ayar önbelleği TTL'li; kaydettikten hemen sonra temizle ki yönetici
	// değişikliği sayfayı yenileyince görsün.
	s.invalidateContentCache()

	msg := "GA4 ayarları kaydedildi."
	if olcum == "" {
		msg = "GA4 ayarları kaydedildi. Ölçüm kimliği boş — sitede izleme kodu basılmıyor."
	}
	s.botResult(w, r, "info", msg)
}

// handleTrendGA4Sync GA4 canlı özetini hemen çeker.
func (s *Server) handleTrendGA4Sync(w http.ResponseWriter, r *http.Request) {
	n, err := scheduler.SyncGA4(r.Context(), s.pool, s.logger)
	if err != nil {
		s.botResult(w, r, "error", "GA4 senkronu: "+err.Error())
		return
	}
	s.audit(r, "trend_ga4_sync", itoa(n))
	s.botResult(w, r, "info", "GA4 canlı özet güncellendi: "+itoa(n)+" aktif kullanıcı.")
}

// handleTrendReportCSV trend terimlerini Excel uyumlu CSV olarak indirir.
func (s *Server) handleTrendReportCSV(w http.ResponseWriter, r *http.Request) {
	items, err := repo.ListTrendTerms(r.Context(), s.pool, repo.TrendFilter{
		SiteID: s.curSiteID(r), Limit: 500,
	})
	if err != nil {
		http.Error(w, "rapor üretilemedi", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=trend-raporu-"+time.Now().Format("20060102")+".csv")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("\xEF\xBB\xBF")) // Excel için UTF-8 BOM
	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	_ = cw.Write([]string{"Terim", "Kaynak", "Yaklaşık Trafik", "Skor", "Kapsama", "İlk Görülme", "Son Görülme", "Gizli"})
	for _, t := range items {
		gizli := "hayır"
		if t.Hidden {
			gizli = "evet"
		}
		_ = cw.Write([]string{
			csvSafe(t.Term), csvSafe(t.SourceName), itoa(t.ApproxTraffic), itoa(t.Score), itoa(t.CoverageCount),
			t.FirstSeenAt.Format("2006-01-02 15:04"), t.LastSeenAt.Format("2006-01-02 15:04"), gizli,
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		s.logger.Warn("trend CSV yazılamadı", "error", err)
	}
}

// csvSafe, Excel'in formül olarak yorumlayabileceği hücreleri etkisizleştirir.
func csvSafe(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	}
	return v
}

// cleanTrendTitle AI başlığını tek satıra indirir ve makul uzunlukta tutar.
func cleanTrendTitle(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 180 {
		s = strings.TrimSpace(string(r[:180]))
	}
	return s
}

// cleanTrendSpot spot metnindeki satır sonlarını ve fazla boşlukları temizler.
func cleanTrendSpot(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 240 {
		s = strings.TrimSpace(string(r[:240]))
	}
	return s
}
