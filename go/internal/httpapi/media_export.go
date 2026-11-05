package httpapi

import (
	"encoding/csv"
	"net/http"
	"strconv"

	"theblock/site/internal/repo"
)

// handleMediaExport medya metadata'sını CSV olarak indirir.
func (s *Server) handleMediaExport(w http.ResponseWriter, r *http.Request) {
	items, err := repo.ListMediaLibrary(r.Context(), s.pool, s.curSiteID(r), repo.MediaFilter{Limit: 5000})
	if err != nil {
		http.Error(w, "Medya listesi alınamadı.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="medya-kutuphanesi.csv"`)
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "baslik", "dosya", "kategori", "alt", "caption", "aciklama", "seo_baslik", "seo_aciklama", "genislik", "yukseklik", "boyut", "url", "olusturma"})
	for _, m := range items {
		wValue, hValue, sizeValue := "", "", ""
		if m.Width != nil {
			wValue = strconv.Itoa(*m.Width)
		}
		if m.Height != nil {
			hValue = strconv.Itoa(*m.Height)
		}
		if m.SizeBytes != nil {
			sizeValue = strconv.FormatInt(*m.SizeBytes, 10)
		}
		_ = cw.Write([]string{
			strconv.FormatInt(m.ID, 10), m.Title, m.Filename, m.Category,
			anyStrPtr(m.Alt), anyStrPtr(m.Caption), m.Description, m.SEOTitle, m.SEODescription,
			wValue, hValue, sizeValue, s.mediaURL(&m.ID), m.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	cw.Flush()
}
