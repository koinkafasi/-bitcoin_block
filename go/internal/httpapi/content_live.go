package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"theblock/site/internal/live"
	"theblock/site/internal/repo"
)

// handleContentLiveSSE, yayındaki bir içerik için Server-Sent Events bağlantısı
// açar. İçerik metni gönderilmez; yalnızca sürüm/durum sinyali gönderilir.
func (s *Server) handleContentLiveSSE(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil || c.SiteID != siteIDFromCtx(r.Context()) {
		http.NotFound(w, r)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming desteklenmiyor", http.StatusInternalServerError)
		return
	}
	// http.Server.WriteTimeout SSE bağlantısını kesmesin.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	kind := "content"
	if c.Status != "yayinda" {
		kind = "removed"
	}
	_ = writeSSE(w, "snapshot", live.Event{
		SiteID: c.SiteID, ContentID: c.ID, Version: c.Version, Status: c.Status,
		Slug: c.DisplaySlug(), Kind: kind, At: time.Now().Unix(),
	})
	flusher.Flush()
	if kind == "removed" {
		return
	}

	ch, cancel := live.Default.Subscribe(id)
	defer cancel()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if err := writeSSE(w, "content", ev); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSE(w io.Writer, event string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	return err
}

// handleContentLiveJSON, SSE sinyali sonrası istemcinin çektiği güncel içerik
// parçasını döner. Yalnızca yayındaki içerikler erişilebilir.
func (s *Server) handleContentLiveJSON(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil || c.SiteID != siteIDFromCtx(r.Context()) {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	if c.Status != "yayinda" {
		s.writeJSON(w, http.StatusGone, map[string]any{
			"ok": false, "content_id": c.ID, "version": c.Version, "status": c.Status,
		})
		return
	}
	locked := s.contentLiveLocked(r, c)
	html := ""
	if !locked {
		html = string(s.blocksToHTML(c.Detail, nil))
	}
	payload := map[string]any{
		"ok":            true,
		"content_id":    c.ID,
		"version":       c.Version,
		"status":        c.Status,
		"title":         c.Title,
		"spot":          c.Spot,
		"html":          html,
		"locked":        locked,
		"url":           publicArticleURL(c.Locale, c.DisplaySlug(), c.ID),
		"updated_at":    c.UpdatedAt.UTC().Format(time.RFC3339),
		"category_name": stringValue(c.CategoryName),
		"category_slug": stringValue(c.CategorySlug),
	}
	if c.CoverMediaID != nil {
		payload["cover_url"] = s.mediaURL(c.CoverMediaID)
	}
	s.writeJSON(w, http.StatusOK, payload)
}

func (s *Server) contentLiveLocked(r *http.Request, c repo.Content) bool {
	required := contentRequiredWeight(c.IsExclusive, c.MinTierWeight)
	if required == 0 {
		return false
	}
	if tierAllows(s.memberWeight(r), required) {
		return false
	}
	return s.paywallAktif(r.Context())
}

func stringValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// publishContentLive, DB commit sonrası yayın sinyali gönderir. Taslak/arşiv
// durumundaki içerikler için removed olayı gider; içerik metni gönderilmez.
// publishEditorSaved, aynı içeriği açmış diğer editörlere yeni sürüm sinyali
// gönderir. Public SSE'den ayrı bir hub kullanır.
func (s *Server) publishEditorSaved(ctx context.Context, id, siteID int64, version int) {
	if id <= 0 {
		return
	}
	if version <= 0 {
		if c, err := repo.GetContent(ctx, s.pool, id); err == nil {
			version = c.Version
		}
	}
	u := userFromContext(ctx)
	live.EditorDefault.Publish(live.Event{
		SiteID: siteID, ContentID: id, Version: version, Kind: "saved", Action: "saved",
		UserID: u.ID, UserName: u.Name, At: time.Now().Unix(),
	})
}

func (s *Server) publishContentLive(ctx context.Context, id, siteID int64, forceRemoved bool) {
	if id <= 0 {
		return
	}
	if forceRemoved {
		live.Default.Publish(live.Event{
			SiteID: siteID, ContentID: id, Kind: "removed", At: time.Now().Unix(),
		})
		return
	}
	c, err := repo.GetContent(ctx, s.pool, id)
	if err != nil {
		live.Default.Publish(live.Event{
			SiteID: siteID, ContentID: id, Kind: "removed", At: time.Now().Unix(),
		})
		return
	}
	kind := "content"
	if c.Status != "yayinda" {
		kind = "removed"
	}
	live.Default.Publish(live.Event{
		SiteID: c.SiteID, ContentID: c.ID, Version: c.Version, Status: c.Status,
		Slug: c.DisplaySlug(), Kind: kind, At: time.Now().Unix(),
	})
}
