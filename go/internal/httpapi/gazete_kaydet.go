package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// gazeteKayitSayfa istemciden gelen tek sayfa.
type gazeteKayitSayfa struct {
	No      int             `json:"no"`
	Ad      string          `json:"ad"`
	Bloklar json.RawMessage `json:"bloklar"`
}

const gazeteKayitAzami = 4 << 20 // 4 MiB; 50 sayfalık bir sayı için fazlasıyla yeterli

// handleGazeteSayfalarKaydet POST /admin/gazete/sayilar/{id}/sayfalar
func (s *Server) handleGazeteSayfalarKaydet(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	ctx := r.Context()
	id := parseID(r.PathValue("id"))

	ham := strings.TrimSpace(r.FormValue("sayfalar"))
	if len(ham) > gazeteKayitAzami {
		http.Error(w, "sayfa verisi çok büyük", http.StatusRequestEntityTooLarge)
		return
	}
	var sayfalar []gazeteKayitSayfa
	if err := json.Unmarshal([]byte(ham), &sayfalar); err != nil {
		http.Error(w, "geçersiz sayfa JSON'u", http.StatusBadRequest)
		return
	}
	if err := gazeteKayitDogrula(sayfalar); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cakisma, err := s.gazeteSayfalariYaz(ctx, id, sayfalar, r.FormValue("surum"))
	if cakisma != "" {
		// 409: istemci kullanıcıya sorsun, sessizce ezmesin.
		s.writeJSON(w, http.StatusConflict, map[string]any{
			"ok":      false,
			"cakisma": true,
			"mesaj":   cakisma,
		})
		return
	}
	if err != nil {
		s.logger.Error("gazete sayfa kaydı", "sayi", id, "error", err)
		http.Error(w, "kayıt hatası", http.StatusInternalServerError)
		return
	}
	s.audit(r, "gazete_sayfalar", fmt.Sprint(id))

	iss, _ := repo.GetGazeteIssue(ctx, s.pool, id)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"surum": gazeteSurum(iss.UpdatedAt),
		"sayfa": len(sayfalar),
	})
}

// gazeteSurum iyimser kilit için sayının sürüm damgası.
func gazeteSurum(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// gazeteSayfalariYaz tüm sayfaları tek transaction'da yazar.
// Dönen ilk değer boş değilse çakışma var demektir (hata değil, kullanıcı kararı).
func (s *Server) gazeteSayfalariYaz(ctx context.Context, issueID int64,
	sayfalar []gazeteKayitSayfa, beklenenSurum string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// İyimser kilit: sayıyı kilitleyip sürümü karşılaştır.
	var guncel time.Time
	if err := tx.QueryRow(ctx,
		`SELECT updated_at FROM gazete_issues WHERE id=$1 FOR UPDATE`, issueID).
		Scan(&guncel); err != nil {
		return "", err
	}
	if beklenenSurum != "" && gazeteSurum(guncel) != beklenenSurum {
		return "Bu sayı siz düzenlerken başka bir yerde değiştirildi. " +
			"Sayfayı yenileyip değişikliklerinizi tekrar uygulayın.", nil
	}

	var numaralar []int
	for _, p := range sayfalar {
		bloklar := p.Bloklar
		if len(bloklar) == 0 {
			bloklar = []byte("[]")
		}
		if err := repo.SaveGazetePage(ctx, tx, issueID, p.No, p.Ad, bloklar); err != nil {
			return "", fmt.Errorf("sayfa %d: %w", p.No, err)
		}
		numaralar = append(numaralar, p.No)
	}

	// Gönderilmeyen sayfalar silinmiş demektir (sayfa yönetimi, Faz 5).
	if _, err := tx.Exec(ctx,
		`DELETE FROM gazete_pages WHERE issue_id=$1 AND page_no <> ALL($2)`,
		issueID, numaralar); err != nil {
		return "", err
	}

	// Sürüm damgasını ilerlet: sonraki kayıt bunu bekleyecek.
	if _, err := tx.Exec(ctx,
		`UPDATE gazete_issues SET updated_at=now() WHERE id=$1`, issueID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return "", nil
}

func gazeteKayitDogrula(sayfalar []gazeteKayitSayfa) error {
	if len(sayfalar) == 0 {
		return fmt.Errorf("hiç sayfa gönderilmedi")
	}
	if len(sayfalar) > 50 {
		return fmt.Errorf("en fazla 50 sayfa")
	}
	gorulen := map[int]bool{}
	for _, p := range sayfalar {
		if p.No < 1 || p.No > 50 {
			return fmt.Errorf("geçersiz sayfa numarası: %d", p.No)
		}
		if gorulen[p.No] {
			return fmt.Errorf("sayfa %d iki kez gönderildi", p.No)
		}
		gorulen[p.No] = true
		if len(p.Bloklar) > 0 && !json.Valid(p.Bloklar) {
			return fmt.Errorf("sayfa %d blok JSON'u geçersiz", p.No)
		}
	}
	return nil
}
