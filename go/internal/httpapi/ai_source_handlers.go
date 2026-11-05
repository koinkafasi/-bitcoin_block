package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/aieditor"
	"theblock/site/internal/repo"
)

type aiSourceFeedView struct {
	repo.Feed
	CategoryName string
}

type aiCategoryPlan struct {
	Name       string
	Feeds      int
	Clusters   int
	Sources    int
	TrendHits  int
	TopCluster string
}

type aiSourceData struct {
	Feeds         []aiSourceFeedView
	Clusters      []repo.AIStoryCluster
	Plan          []aiCategoryPlan
	Plans         []repo.AICategoryPlan
	Trends        []repo.TrendTermRow
	TotalItems    int
	EnrichPending int
	Cats          []repo.Category
}

func (s *Server) buildAISourceData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	d := s.baseView("tr", "/admin", "AI Kaynak Merkezi")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)

	data := &aiSourceData{}
	feeds, _ := repo.ListFeeds(ctx, s.pool)
	cats, _ := repo.ListCategories(ctx, s.pool, sid)
	catName := map[int64]string{}
	for _, c := range cats {
		catName[c.ID] = c.Name
	}
	for _, f := range feeds {
		v := aiSourceFeedView{Feed: f}
		if f.CategoryID != nil {
			v.CategoryName = catName[*f.CategoryID]
		}
		data.Feeds = append(data.Feeds, v)
	}
	data.Cats = cats
	data.Clusters, _ = repo.ListAIStoryClusters(ctx, s.pool, sid, 60)
	data.Plans, _ = repo.ListAICategoryPlans(ctx, s.pool, sid)
	data.EnrichPending, _ = repo.CountPendingAIEnrichment(ctx, s.pool, sid)
	data.Trends, _ = repo.TrendTermsForScoring(ctx, s.pool, sid, 30)
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM ai_source_items WHERE site_id=$1`, sid).Scan(&data.TotalItems)

	byCat := map[string]*aiCategoryPlan{}
	for _, f := range data.Feeds {
		name := f.CategoryName
		if name == "" {
			name = "Kategorisiz"
		}
		p := byCat[name]
		if p == nil {
			p = &aiCategoryPlan{Name: name}
			byCat[name] = p
		}
		p.Feeds++
	}
	for _, c := range data.Clusters {
		name := c.CategoryName
		if name == "" {
			name = "Kategorisiz"
		}
		p := byCat[name]
		if p == nil {
			p = &aiCategoryPlan{Name: name}
			byCat[name] = p
		}
		p.Clusters++
		p.Sources += c.SourceCount
		if c.TrendTerm != "" {
			p.TrendHits++
		}
		if p.TopCluster == "" {
			p.TopCluster = c.Title
		}
	}
	for _, p := range byCat {
		data.Plan = append(data.Plan, *p)
	}
	// Planı küme sayısına, eşitlikte kaynak sayısına göre sırala.
	for i := 0; i < len(data.Plan); i++ {
		for j := i + 1; j < len(data.Plan); j++ {
			if data.Plan[j].Clusters > data.Plan[i].Clusters ||
				(data.Plan[j].Clusters == data.Plan[i].Clusters && data.Plan[j].Feeds > data.Plan[i].Feeds) {
				data.Plan[i], data.Plan[j] = data.Plan[j], data.Plan[i]
			}
		}
	}
	d.AISource = data
	return d
}

func (s *Server) handleAISourceGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildAISourceData(r)
	if IsHTMX(r) {
		s.renderAIKaynakPanelV2(w, r, d)
		return
	}
	s.renderAIKaynakPageV2(w, r, d)
}

func (s *Server) renderAISourcePanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildAISourceData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderAIKaynakPanelV2(w, r, d)
}

func (s *Server) handleAISourceScan(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	sid := s.curSiteID(r)
	n, c, merged, err := repo.ScanAndClusterAISources(ctx, s.pool, sid)
	if err != nil {
		s.renderAISourcePanel(w, r, "error", "Kaynak taraması tamamlanamadı: "+err.Error())
		return
	}
	msg := itoa(n) + " yeni kaynak öğesi alındı, " + itoa(c) + " benzer haber kümesi oluşturuldu."
	if merged > 0 {
		msg += " " + itoa(merged) + " çapraz kategori kümesi birleştirildi."
	}
	s.renderAISourcePanel(w, r, "info", msg)
}

func (s *Server) handleAISourceEnrich(w http.ResponseWriter, r *http.Request) {
	res, err := repo.EnrichAISourceItems(r.Context(), s.pool, s.curSiteID(r), 60)
	if err != nil {
		s.renderAISourcePanel(w, r, "error", "Zenginleştirme tamamlanamadı: "+err.Error())
		return
	}
	msg := ""
	if res.Enriched > 0 {
		msg = itoa(res.Enriched) + " kaynak öğesi zenginleştirildi."
	}
	if res.Recategorized > 0 {
		if msg != "" {
			msg += " "
		}
		msg += itoa(res.Recategorized) + " öğenin kategorisi düzeltildi."
	}
	if res.Failed > 0 {
		if msg != "" {
			msg += " "
		}
		msg += itoa(res.Failed) + " öğe okunamadı."
	}
	if res.Pending > 0 {
		if msg != "" {
			msg += " "
		}
		msg += itoa(res.Pending) + " öğe bekliyor."
	}
	if msg == "" {
		msg = "İşlenecek öğe kalmadı."
	}
	s.renderAISourcePanel(w, r, "info", msg)
}

func (s *Server) handleAISourceDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	id := parseID(r.PathValue("id"))
	if id == 0 {
		s.renderAISourcePanel(w, r, "error", "Küme bulunamadı.")
		return
	}
	cid, err := repo.CreateDraftFromCluster(ctx, s.pool, sid, id, userFromContext(ctx).ID)
	if err != nil {
		s.renderAISourcePanel(w, r, "error", "Taslak oluşturulamadı: "+err.Error())
		return
	}
	// AI editör kuyruğuna haber taslağı işi ekle (senkron değil).
	if _, err := s.aiEditor.Enqueue(ctx, aieditor.Request{
		ContentID: cid, User: userFromContext(ctx), Action: "news",
		Instruction: "Kaynak özetlerini kullanarak özgün, doğrulanabilir bir Türkçe haber taslağı oluştur.",
	}); err != nil {
		s.flashRedirect(w, r, "/admin/icerikler/"+itoa64(cid), "info", "Taslak oluşturuldu; AI işi kuyruğa alınamadı: "+err.Error())
		return
	}
	s.flashRedirect(w, r, "/admin/icerikler/"+itoa64(cid), "info", "Kaynak kümesinden taslak oluşturuldu; AI işi kuyruğa alındı.")
}

func (s *Server) handleAISourcePlanSave(w http.ResponseWriter, r *http.Request) {
	sid := s.curSiteID(r)
	p, err := s.saveAIPlanForm(r.Context(), r, sid, parseID(r.PathValue("id")))
	if err != nil {
		s.renderAISourcePanel(w, r, "error", "Plan güncellenemedi: "+err.Error())
		return
	}
	s.renderAISourcePanel(w, r, "info", p.CategoryName+" üretim planı güncellendi.")
}

func (s *Server) handleAISourcePlanRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	p, err := s.saveAIPlanForm(ctx, r, sid, parseID(r.PathValue("id")))
	if err != nil {
		s.renderAISourcePanel(w, r, "error", "Plan çalıştırılamadı: "+err.Error())
		return
	}
	if !p.Aktif {
		s.renderAISourcePanel(w, r, "info", p.CategoryName+" planı pasif; önce aktifleştirin.")
		return
	}
	remaining := p.DailyTarget - p.DraftedToday
	if remaining <= 0 {
		s.renderAISourcePanel(w, r, "info", p.CategoryName+" için günlük hedef doldu ("+itoa(p.DraftedToday)+"/"+itoa(p.DailyTarget)+").")
		return
	}
	clusters, err := repo.ListAIClustersForPlan(ctx, s.pool, sid, p.CategoryID, p.MinSources, p.TrendWeight, p.MaxAgeHours, remaining)
	if err != nil {
		s.renderAISourcePanel(w, r, "error", "Uygun kümeler okunamadı: "+err.Error())
		return
	}
	drafted, enqueueErrors := 0, 0
	for _, c := range clusters {
		cid, err := repo.CreateDraftFromCluster(ctx, s.pool, sid, c.ID, userFromContext(ctx).ID)
		if err != nil {
			continue
		}
		drafted++
		if _, err := s.aiEditor.Enqueue(ctx, aieditor.Request{
			ContentID: cid, User: userFromContext(ctx), Action: "news",
			Instruction: aiPlanInstruction(*p),
		}); err != nil {
			enqueueErrors++
		}
	}
	repo.TouchAICategoryPlan(ctx, s.pool, sid, p.ID)
	if drafted == 0 {
		s.renderAISourcePanel(w, r, "info", p.CategoryName+" için uygun yeni küme bulunamadı.")
		return
	}
	msg := fmt.Sprintf("%s: %d taslak oluşturuldu.", p.CategoryName, drafted)
	if enqueueErrors > 0 {
		msg += fmt.Sprintf(" %d AI işi kuyruğa alınamadı.", enqueueErrors)
	}
	s.renderAISourcePanel(w, r, "info", msg)
}

func (s *Server) saveAIPlanForm(ctx context.Context, r *http.Request, siteID, id int64) (*repo.AICategoryPlan, error) {
	if id == 0 {
		return nil, fmt.Errorf("plan bulunamadı")
	}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	p, err := repo.GetAICategoryPlan(ctx, s.pool, siteID, id)
	if err != nil {
		return nil, err
	}
	target := clampInt(aiPlanFormInt(r, "hedef", p.DailyTarget), 0, 50)
	minSources := clampInt(aiPlanFormInt(r, "minsrc", p.MinSources), 1, 20)
	trendWeight := clampInt(aiPlanFormInt(r, "trend", p.TrendWeight), 0, 100)
	maxAge := clampInt(aiPlanFormInt(r, "yas", p.MaxAgeHours), 1, 720)
	aktif := aiPlanFormInt(r, "aktif", boolToInt(p.Aktif)) == 1
	if err := repo.UpdateAICategoryPlan(ctx, s.pool, siteID, id, target, minSources, trendWeight, maxAge, aktif); err != nil {
		return nil, err
	}
	return repo.GetAICategoryPlan(ctx, s.pool, siteID, id)
}

func aiPlanFormInt(r *http.Request, key string, def int) int {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func aiPlanInstruction(p repo.AICategoryPlan) string {
	base := "Kaynak kümesindeki doğrulanabilir bilgileri kullanarak özgün, akıcı bir Türkçe haber taslağı yaz. Kaynakta olmayan olgu, kişi, tarih, rakam veya alıntı ekleme; eksik bilgiyi açıkça işaretle."
	switch p.CategorySlug {
	case "ekonomi":
		return base + " Rakamları, kurum adlarını ve dönemleri koru; yorum ile veriyi ayır."
	case "spor":
		return base + " Maç/skor/transfer bilgisini kaynaktaki haliyle koru; taraftar yorumu katma."
	case "dunya":
		return base + " Ülke, kurum ve aktör adlarını netleştir; jeopolitik yorumu ayrı tut."
	case "teknoloji":
		return base + " Ürün adı, sürüm ve teknik iddiaları kaynaktaki haliyle aktar; abartılı pazarlama dili kullanma."
	case "gundem":
		return base + " İlk paragrafta en önemli gelişmeyi ver; resmi açıklama ile tanık/iddia ayrımını koru."
	default:
		return base + " Kategoriye uygun sade bir editoryal ton kullan."
	}
}
