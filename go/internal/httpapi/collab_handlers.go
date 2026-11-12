package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"theblock/site/internal/live"
	"theblock/site/internal/repo"
)

func collabSecret() string { return strings.TrimSpace(os.Getenv("COLLAB_SECRET")) }

func collabURL() string { return strings.TrimSpace(os.Getenv("COLLAB_URL")) }

func collabServiceUp(raw string) bool {
	u, err := neturl.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	conn, err := net.DialTimeout("tcp", host, 400*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (s *Server) handleCollabToken(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false})
		return
	}
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	if !canEditContent(userFromContext(r.Context()), c) {
		s.writeJSON(w, http.StatusForbidden, map[string]any{"ok": false})
		return
	}
	secret, base := collabSecret(), collabURL()
	if secret == "" || base == "" || !collabServiceUp(base) {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "collab kapalı"})
		return
	}
	u := userFromContext(r.Context())
	exp := time.Now().Add(2 * time.Hour).Unix()
	payload := fmt.Sprintf("%d:%d:%d", id, u.ID, exp)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	token := payload + ":" + hex.EncodeToString(mac.Sum(nil))
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "url": base, "token": token, "name": fmt.Sprintf("content-%d", id),
	})
}

func collabYerelAğ(r *http.Request) bool {
	return isPrivateIP(clientIP(r))
}

func (s *Server) handleCollabGet(w http.ResponseWriter, r *http.Request) {
	if !collabYerelAğ(r) || !collabSecretOK(r) {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// Sürüm, istemcinin iyimser eşzamanlılık başlığı için gerekir.
	w.Header().Set("X-Collab-Version", strconv.Itoa(c.Version))
	_, _ = io.WriteString(w, s.detailToEditorJSON(c.Detail))
}

func (s *Server) handleCollabSave(w http.ResponseWriter, r *http.Request) {
	if !collabYerelAğ(r) || !collabSecretOK(r) {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil || !isTipTapRaw(string(body)) {
		http.Error(w, "geçersiz doküman", http.StatusBadRequest)
		return
	}
	expected := 0
	if v := strings.TrimSpace(r.Header.Get("X-Collab-Version")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			expected = n
		}
	}
	version, siteID, status, slug, err := repo.SaveCollabDetail(r.Context(), s.pool, id, body, expected)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			http.NotFound(w, r)
			return
		case errors.Is(err, repo.ErrSurumCakismasi):
			current := repo.ContentVersion(r.Context(), s.pool, id)
			w.Header().Set("X-Collab-Current-Version", strconv.Itoa(current))
			s.writeJSON(w, http.StatusConflict, map[string]any{
				"ok": false, "error": "sürüm çakışması", "version": current,
			})
			return
		}
		http.Error(w, "kaydedilemedi", http.StatusInternalServerError)
		return
	}
	kind := "content"
	if status != "yayinda" {
		kind = "removed"
	}
	live.Default.Publish(live.Event{
		SiteID: siteID, ContentID: id, Version: version, Status: status,
		Slug: slug, Kind: kind, At: time.Now().Unix(),
	})
	live.EditorDefault.Publish(live.Event{
		SiteID: siteID, ContentID: id, Version: version, Kind: "saved", Action: "saved",
		UserName: "Ortak düzenleme", At: time.Now().Unix(),
	})
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "version": version})
}

func collabSecretOK(r *http.Request) bool {
	secret := collabSecret()
	if secret == "" {
		return false
	}
	got := strings.TrimSpace(r.Header.Get("X-Collab-Secret"))
	if len(got) != len(secret) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(secret)) == 1
}
