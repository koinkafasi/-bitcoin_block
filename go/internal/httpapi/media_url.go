package httpapi

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// handleMediaURLUpload, güvenli SSRF kontrolleriyle uzak görsel URL'sini medyaya alır.
func (s *Server) handleMediaURLUpload(w http.ResponseWriter, r *http.Request) {
	jsonOut := r.URL.Query().Get("json") == "1"
	mediaErr := func(msg string) {
		if jsonOut {
			s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": msg})
			return
		}
		if IsHTMX(r) {
			s.renderMediaPanel(w, r, "error", msg)
			return
		}
		s.flashRedirect(w, r, "/admin/dosyalar", "error", msg)
	}
	_ = r.ParseForm()
	target, err := validateBookmarkURL(strings.TrimSpace(r.FormValue("url")))
	if err != nil {
		mediaErr(err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		mediaErr("Adres geçersiz.")
		return
	}
	req.Header.Set("User-Agent", "TheBlockBot/1.0 (+https://theblock.com.tr)")
	req.Header.Set("Accept", "image/*")
	// #nosec G704 -- safeRemoteHTTPClient: her istekte ve her yönlendirmede
	// hedefi yeniden doğrular (genel IP, 80/443). Bkz. platform.SafeFetchClient.
	resp, err := safeRemoteHTTPClient(20 * time.Second).Do(req)
	if err != nil {
		mediaErr("Uzak görsel getirilemedi.")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		mediaErr("Uzak sunucu görseli vermedi.")
		return
	}
	if ct := strings.ToLower(resp.Header.Get("Content-Type")); ct != "" && !strings.HasPrefix(ct, "image/") {
		mediaErr("Adres bir görsel döndürmüyor.")
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 30<<20))
	if err != nil || len(data) == 0 {
		mediaErr("Görsel okunamadı.")
		return
	}
	decoded, err := platform.DecodeImage(data)
	if err != nil {
		mediaErr("Uzak dosya desteklenen bir görsel değil (jpg/png/webp).")
		return
	}
	b := decoded.Bounds()
	width, height := b.Dx(), b.Dy()
	name := filepath.Base(strings.TrimSpace(target.Path))
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" || base == "." || base == "/" {
		base = "url-gorsel-" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	ext := ".jpg"
	switch ct := http.DetectContentType(data); {
	case strings.Contains(ct, "png"):
		ext = ".png"
	case strings.Contains(ct, "webp"):
		ext = ".webp"
	}
	name = base + ext
	storedPath, err := platform.SaveMedia(data, name)
	if err != nil {
		mediaErr("Depolama hatası.")
		return
	}
	var thumbPath *string
	if thumb, err := platform.CreateThumb(data, 480); err == nil {
		if p, err := platform.SaveMedia(thumb, "thumb.jpg"); err == nil {
			thumbPath = &p
		}
	}
	size := int64(len(data))
	title := strings.TrimSpace(r.FormValue("baslik"))
	if title == "" {
		title = name
	}
	category := strings.TrimSpace(r.FormValue("kategori"))
	if category == "" {
		category = "genel"
	}
	alt := strings.TrimSpace(r.FormValue("alt"))
	caption := strings.TrimSpace(r.FormValue("caption"))
	mimeType, _ := platformFileExt(name)
	id, err := repo.CreateMedia(ctx, s.pool, &repo.Media{
		SiteID: s.curSiteID(r), UserID: userFromContext(ctx).ID, Kind: "image", Filename: name,
		Path: storedPath, Mime: mimeType, SizeBytes: &size, Width: &width, Height: &height,
		Title: title, Category: category, Alt: &alt, Caption: &caption,
		Description: strings.TrimSpace(r.FormValue("aciklama")), ThumbPath: thumbPath, Source: "internet",
	})
	if err != nil {
		_ = platform.DeleteMedia(storedPath)
		if thumbPath != nil {
			_ = platform.DeleteMedia(*thumbPath)
		}
		s.logger.Error("url medya kaydı", "error", err)
		mediaErr("Kayıt hatası.")
		return
	}
	s.audit(r, "medya_url_yukle", name)
	if jsonOut {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "id": id, "url": s.mediaURL(&id), "thumb": s.mediaThumbURL(&id), "filename": name,
		})
		return
	}
	if IsHTMX(r) {
		s.renderMediaPanel(w, r, "info", name+" URL'den yüklendi.")
		return
	}
	s.flashRedirect(w, r, "/admin/dosyalar", "info", name+" URL'den yüklendi.")
}
