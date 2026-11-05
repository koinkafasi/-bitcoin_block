package httpapi

import (
	"net/http"
	"path/filepath"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// handleMediaTransform JPEG/PNG görsellerde 90° döndürme ve yatay/dikey çevirme yapar.
func (s *Server) handleMediaTransform(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	ctx := r.Context()
	media, err := repo.GetMedia(ctx, s.pool, id)
	if err != nil || media.SiteID != s.curSiteID(r) {
		s.flashRedirect(w, r, "/admin/dosyalar", "error", "Medya bulunamadı.")
		return
	}
	if media.Kind != "image" {
		s.flashRedirect(w, r, "/admin/dosyalar", "error", "Yalnızca görseller düzenlenebilir.")
		return
	}
	_ = r.ParseForm()
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	_ = r.ParseMultipartForm(1 << 20)
	op := strings.TrimSpace(r.FormValue("islem"))
	ext := strings.ToLower(filepath.Ext(media.Path))
	if ext == ".webp" {
		s.flashRedirect(w, r, "/admin/dosyalar", "error", "WebP görseller şu an döndürülemiyor; JPEG/PNG kullanın.")
		return
	}
	data, err := platform.ReadMedia(media.Path)
	if err != nil {
		s.flashRedirect(w, r, "/admin/dosyalar", "error", "Görsel dosyası okunamadı.")
		return
	}
	out, img, err := platform.TransformImage(data, ext, op)
	if err != nil {
		s.flashRedirect(w, r, "/admin/dosyalar", "error", "Görsel düzenlenemedi: "+err.Error())
		return
	}
	if err := platform.WriteMedia(media.Path, out); err != nil {
		s.flashRedirect(w, r, "/admin/dosyalar", "error", "Görsel kaydedilemedi.")
		return
	}
	if media.ThumbPath != nil {
		_ = platform.DeleteMedia(*media.ThumbPath)
	}
	platform.DeleteMediaVariants(id)
	var thumbPath *string
	if thumb, err := platform.CreateThumb(out, 480); err == nil {
		if p, err := platform.SaveMedia(thumb, "thumb.jpg"); err == nil {
			thumbPath = &p
		}
	}
	b := img.Bounds()
	if err := repo.UpdateMediaImageMeta(ctx, s.pool, id, b.Dx(), b.Dy(), thumbPath); err != nil {
		s.logger.Error("medya düzenleme meta", "error", err)
		s.flashRedirect(w, r, "/admin/dosyalar", "error", "Görsel kaydedildi ama kayıt güncellenemedi.")
		return
	}
	s.audit(r, "medya_duzenle", media.Filename+" "+op)
	s.flashRedirect(w, r, "/admin/dosyalar", "info", media.Filename+" düzenlendi.")
}
