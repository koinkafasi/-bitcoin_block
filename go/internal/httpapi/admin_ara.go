package httpapi

import (
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// handleAdminAra GET /admin/ara?q= — panel komut paleti varlık araması (Faz 162).
//
// Palet önceden yalnız 38 sabit rota listesiydi; kullanıcı bir içerik başlığı
// yazınca sonuç gelmiyordu. Bu uç nokta içerik, kategori, etiket, medya ve
// kişiyi gruplu döndürür.
//
// Sözleşme:
//   - 2 karakterden kısa sorgu → boş grup listesi (200). Hata değil: palet
//     açılır açılmaz boş sonuç göstermemesi için çağıran taraf da bunu bekler.
//   - Sonuçlar site kapsamlı (tb_site çerezi).
//   - HTML üretilmez; yalnız veri. Kaçış istemcide textContent ile yapılır.
func (s *Server) handleAdminAra(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "gruplar": []any{}, "q": q})
		return
	}
	gruplar, err := repo.AdminArama(r.Context(), s.pool, s.curSiteID(r), q, 5)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError,
			map[string]any{"ok": false, "error": "Arama başarısız", "gruplar": []any{}})
		return
	}
	if gruplar == nil {
		gruplar = []repo.AdminAramaGrup{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "gruplar": gruplar, "q": q})
}
