package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/aieditor"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

type aiEditorView struct {
	ContentID   int64
	ActionID    string
	ActionLabel string
	JobID       int64
	Status      string
	PollURL     string
	CancelURL   string
	Result      aieditor.Result
	Error       string
	Jobs        []aiJobView
	Revisions   []repo.ContentRevision
}

type aiJobView struct {
	Job    repo.AIJob
	Label  string
	When   string
	Status string
}

// handleAIEditorAction seçili metin işlemini başlatır ve sonuç panelini döndürür.
func (s *Server) handleAIEditorAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	act, ok := aieditor.ActionByID(r.FormValue("action"))
	if !ok {
		s.renderAIEditorError(w, r, "Bilinmeyen AI işlemi.")
		return
	}
	// Gizli eylemler (ör. audio_draft) yalnızca kendi özel akışından
	// çağrılır; seçili metin formundan doğrudan tetiklenemez.
	if act.Hidden {
		s.renderAIEditorError(w, r, "Bu AI işlemi yalnızca kendi panelinden başlatılabilir.")
		return
	}
	if s.aiEditor == nil {
		s.renderAIEditorError(w, r, "AI editör servisi hazır değil.")
		return
	}
	_ = r.ParseForm()
	req := aieditor.Request{
		ContentID:   id,
		User:        userFromContext(ctx),
		Action:      act.ID,
		Selection:   r.FormValue("selection"),
		Instruction: r.FormValue("instruction"),
		Angle:       r.FormValue("angle"),
	}
	if act.Research {
		jobID, err := s.aiEditor.Enqueue(ctx, req)
		if err != nil {
			s.logger.Warn("ai editor enqueue", "content", id, "action", act.ID, "error", err)
			s.renderAIEditorError(w, r, aiErrorText(err))
			return
		}
		s.renderAIEditorPending(w, r, id, act, jobID, "Kuyrukta")
		return
	}
	res, jobID, err := s.aiEditor.Run(ctx, req)
	if err != nil {
		s.logger.Warn("ai editor action", "content", id, "action", act.ID, "error", err)
		s.renderAIEditorError(w, r, aiErrorText(err))
		return
	}
	s.renderAIEditorResult(w, r, &aiEditorView{
		ContentID: id, ActionID: act.ID, ActionLabel: act.Label, JobID: jobID, Result: res,
	})
}

func aiErrorText(err error) string {
	switch {
	case errors.Is(err, aieditor.ErrForbidden):
		return "Bu içerikte AI işlemi yapma yetkiniz yok."
	case errors.Is(err, aieditor.ErrAIUnavailable):
		return "AI sağlayıcısına ulaşılamadı. Anahtar/adres yapılandırmasını ve sağlayıcı durumunu kontrol edin."
	case errors.Is(err, aieditor.ErrResearchUnavailable):
		return "Araştırma sağlayıcısına ulaşılamadı. Brave Search anahtarını ve bağlantıyı kontrol edin."
	case errors.Is(err, aieditor.ErrRateLimited):
		return "AI işlem kotanız doldu. Bir dakika sonra tekrar deneyin."
	case errors.Is(err, aieditor.ErrJobCancelled):
		return "AI işi iptal edildi."
	}
	return err.Error()
}

func (s *Server) renderAIEditorPending(w http.ResponseWriter, r *http.Request, contentID int64, act aieditor.Action, jobID int64, status string) {
	base := "/admin/icerikler/" + itoa64(contentID) + "/ai/is/" + itoa64(jobID)
	s.renderPartial(w, r, "ai_editor_pending", &tmplData{AI: &aiEditorView{
		ContentID: contentID, ActionID: act.ID, ActionLabel: act.Label,
		JobID: jobID, Status: status, PollURL: base + "/durum", CancelURL: base + "/iptal",
	}})
}

func (s *Server) renderAIEditorResult(w http.ResponseWriter, r *http.Request, v *aiEditorView) {
	s.renderPartial(w, r, "ai_editor_result", &tmplData{AI: v})
}

func (s *Server) renderAIEditorError(w http.ResponseWriter, r *http.Request, msg string) {
	s.renderPartial(w, r, "ai_editor_result", &tmplData{AI: &aiEditorView{Error: msg}})
}

// handleAIEditorJob geçmişteki bir işin sonucunu yeniden açar.
func (s *Server) handleAIEditorJob(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	if s.aiEditor == nil {
		s.renderAIEditorError(w, r, "AI editör servisi hazır değil.")
		return
	}
	jobID := parseID(r.PathValue("job"))
	job, err := s.aiEditor.GetJob(r.Context(), jobID)
	if err != nil || job.ContentID == nil || *job.ContentID != id {
		s.renderAIEditorError(w, r, "AI işi bulunamadı.")
		return
	}
	act, _ := aieditor.ActionByID(job.Action)
	if job.Status == "failed" {
		msg := job.Error
		if strings.TrimSpace(msg) == "" {
			msg = "AI işi başarısız oldu."
		}
		s.renderAIEditorError(w, r, aiErrorText(errors.New(msg)))
		return
	}
	if job.Status == "cancelled" {
		s.renderAIEditorError(w, r, "AI işi iptal edildi.")
		return
	}
	if job.Status != "completed" {
		s.renderAIEditorPending(w, r, id, act, job.ID, aiStatusTR(job.Status))
		return
	}
	res, err := aieditor.DecodeResult(job.Result)
	if err != nil {
		s.renderAIEditorError(w, r, "Bu işin sonucu okunamadı.")
		return
	}
	s.renderAIEditorResult(w, r, &aiEditorView{
		ContentID: id, ActionID: job.Action, ActionLabel: act.Label, JobID: job.ID, Result: res,
	})
}

// handleAIEditorStatus kuyruktaki/çalışan işin durumunu döndürür. UI bu ucu
// polling ile çağırır; iş bitince sonuç veya açık hata paneli gelir.
func (s *Server) handleAIEditorStatus(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	if s.aiEditor == nil {
		s.renderAIEditorError(w, r, "AI editör servisi hazır değil.")
		return
	}
	jobID := parseID(r.PathValue("job"))
	job, err := s.aiEditor.GetJob(r.Context(), jobID)
	if err != nil || job.ContentID == nil || *job.ContentID != id {
		s.renderAIEditorError(w, r, "AI işi bulunamadı.")
		return
	}
	act, _ := aieditor.ActionByID(job.Action)
	switch job.Status {
	case "completed":
		res, derr := aieditor.DecodeResult(job.Result)
		if derr != nil {
			s.renderAIEditorError(w, r, "AI sonucu okunamadı.")
			return
		}
		s.renderAIEditorResult(w, r, &aiEditorView{
			ContentID: id, ActionID: job.Action, ActionLabel: act.Label, JobID: job.ID, Result: res,
		})
	case "failed":
		msg := job.Error
		if strings.TrimSpace(msg) == "" {
			msg = "AI işi başarısız oldu."
		}
		s.renderAIEditorError(w, r, aiErrorText(errors.New(msg)))
	case "cancelled":
		s.renderAIEditorError(w, r, "AI işi iptal edildi.")
	default:
		s.renderAIEditorPending(w, r, id, act, job.ID, aiStatusTR(job.Status))
	}
}

// handleAIEditorCancel kuyruktaki veya çalışan işi iptal eder.
func (s *Server) handleAIEditorCancel(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	if s.aiEditor == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false})
		return
	}
	jobID := parseID(r.PathValue("job"))
	job, err := s.aiEditor.GetJob(r.Context(), jobID)
	if err != nil || job.ContentID == nil || *job.ContentID != id {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "hata": "AI işi bulunamadı"})
		return
	}
	ok, err := repo.CancelAIJob(r.Context(), s.pool, jobID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "hata": "iptal edilemedi"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": ok, "durum": "cancelled"})
}

// handleContentRevisionDiff, seçili sürüm ile mevcut içeriğin satır bazlı
// karşılaştırmasını döndürür.
func (s *Server) handleContentRevisionDiff(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	rid := parseID(r.PathValue("rid"))
	if id <= 0 || rid <= 0 {
		http.Error(w, "geçersiz sürüm", http.StatusBadRequest)
		return
	}
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		http.Error(w, "içerik bulunamadı", http.StatusNotFound)
		return
	}
	if !canEditContent(userFromContext(r.Context()), c) {
		http.Error(w, "yetkisiz", http.StatusForbidden)
		return
	}
	rev, err := repo.GetContentRevision(r.Context(), s.pool, id, rid)
	if err != nil {
		http.Error(w, "sürüm bulunamadı", http.StatusNotFound)
		return
	}
	d := &tmplData{RevisionDiff: diffTextLines(blocksToText(rev.Detail), blocksToText(c.Detail))}
	s.renderPartial(w, r, "content_revision_diff", d)
}

func diffTextLines(oldText, newText string) []revisionDiffLine {
	a := splitDiffLines(oldText)
	b := splitDiffLines(newText)
	if len(a)*len(b) > 300000 {
		out := make([]revisionDiffLine, 0, len(a)+len(b))
		for _, s := range a {
			out = append(out, revisionDiffLine{Kind: "del", Text: s})
		}
		for _, s := range b {
			out = append(out, revisionDiffLine{Kind: "add", Text: s})
		}
		return out
	}
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out []revisionDiffLine
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			out = append(out, revisionDiffLine{Kind: "same", Text: a[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			out = append(out, revisionDiffLine{Kind: "del", Text: a[i]})
			i++
		} else {
			out = append(out, revisionDiffLine{Kind: "add", Text: b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, revisionDiffLine{Kind: "del", Text: a[i]})
	}
	for ; j < m; j++ {
		out = append(out, revisionDiffLine{Kind: "add", Text: b[j]})
	}
	return out
}

func splitDiffLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// handleAIEditorHistory içerik için AI işleri + sürüm geçmişini döndürür.
func (s *Server) handleAIEditorHistory(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	if s.aiEditor == nil {
		s.renderAIEditorError(w, r, "AI editör servisi hazır değil.")
		return
	}
	jobs, _ := s.aiEditor.ListJobs(r.Context(), id)
	revs, _ := repo.ListContentRevisions(r.Context(), s.pool, id, 20)
	view := &aiEditorView{ContentID: id}
	for _, j := range jobs {
		act, _ := aieditor.ActionByID(j.Action)
		view.Jobs = append(view.Jobs, aiJobView{
			Job: j, Label: act.Label,
			When: j.CreatedAt.Format("02.01.2006 15:04"), Status: aiStatusTR(j.Status),
		})
	}
	view.Revisions = revs
	s.renderPartial(w, r, "ai_editor_history", &tmplData{AI: view})
}

// handleAIEditorNewDraft onaylanan sonucu yeni taslak içerik olarak kaydeder.
// Mevcut habere otomatik yazmaz; yalnızca kullanıcının açık "yeni taslak"
// isteğiyle çağrılır.
func (s *Server) handleAIEditorNewDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	orig, ok := s.editableContent(w, r, id)
	if !ok {
		return
	}
	failDraft := func(msg string) {
		if IsHTMX(r) {
			s.renderAIEditorError(w, r, msg)
			return
		}
		s.flashRedirect(w, r, "/admin/icerikler/"+itoa64(id), "error", msg)
	}
	if s.aiEditor == nil {
		failDraft("AI editör servisi hazır değil.")
		return
	}
	jobID := parseID(r.FormValue("job_id"))
	job, err := s.aiEditor.GetJob(ctx, jobID)
	if err != nil || job.ContentID == nil || *job.ContentID != id {
		failDraft("AI işi bulunamadı.")
		return
	}
	res, err := aieditor.DecodeResult(job.Result)
	if err != nil {
		failDraft("AI sonucu okunamadı.")
		return
	}
	title := strings.TrimSpace(res.Fields["title"])
	if title == "" {
		title = strings.TrimSpace(res.Title)
	}
	if title == "" && len(res.Alternatives) > 0 {
		title = res.Alternatives[0].Text
	}
	body := strings.TrimSpace(res.Fields["body"])
	if body == "" {
		for _, sec := range res.Sections {
			switch sec.Title {
			case "Gövde", "Yeni metin", "Özet":
				body = sec.Body
			}
			if body != "" {
				break
			}
		}
	}
	if title == "" || body == "" {
		failDraft("Yeni taslak için başlık ve gövde gerekli.")
		return
	}
	u := userFromContext(ctx)
	slug := optStr(platform.Slugify(title))
	c := &repo.Content{
		TypeKey: orig.TypeKey, Status: "taslak", Title: title, Spot: strings.TrimSpace(res.Fields["spot"]),
		Detail: textToBlocks(body), CategoryID: orig.CategoryID, AuthorID: u.ID,
		SiteID: orig.SiteID, Locale: orig.Locale, CommentStatus: "acik",
		Slug: slug,
	}
	nid, err := repo.CreateContent(ctx, s.pool, c)
	if err != nil {
		s.logger.Error("ai yeni taslak", "error", err)
		failDraft("Yeni taslak oluşturulamadı.")
		return
	}
	s.audit(r, "ai_yeni_taslak", "kaynak="+itoa64(id)+" yeni="+itoa64(nid)+" job="+itoa64(jobID))
	target := "/admin/icerikler/" + itoa64(nid)
	if s.htmxRedirect(w, r, target) {
		return
	}
	s.flashRedirect(w, r, target, "info", "AI sonucu yeni taslak olarak kaydedildi.")
}

// handleAIEditorRestore içerik sürümünü geri yükler (yeni sürüm olarak).
func (s *Server) handleAIEditorRestore(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	rid := parseID(r.PathValue("rid"))
	u := userFromContext(r.Context())
	version, err := repo.RestoreContentRevision(r.Context(), s.pool, id, rid, &u.ID)
	if err != nil {
		if IsHTMX(r) {
			s.renderAIEditorError(w, r, "Sürüm geri yüklenemedi.")
			return
		}
		s.flashRedirect(w, r, "/admin/icerikler/"+itoa64(id), "error", "Sürüm geri yüklenemedi.")
		return
	}
	s.audit(r, "icerik_surum_geri_al", "içerik="+itoa64(id)+" revizyon="+itoa64(rid)+" yeni_surum="+itoa64(int64(version)))
	target := "/admin/icerikler/" + itoa64(id)
	if s.htmxRedirect(w, r, target) {
		return
	}
	s.flashRedirect(w, r, target, "info", "Sürüm geri yüklendi.")
}

func aiStatusTR(s string) string {
	switch s {
	case "completed":
		return "Tamamlandı"
	case "failed":
		return "Başarısız"
	case "running":
		return "Çalışıyor"
	case "cancelled":
		return "İptal"
	default:
		return "Kuyrukta"
	}
}

func itoa64(n int64) string {
	return strconv.FormatInt(n, 10)
}
