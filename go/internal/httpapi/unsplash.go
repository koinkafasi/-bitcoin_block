package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type unsplashPhoto struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Alt         string `json:"alt_description"`
	URLs        struct {
		Small   string `json:"small"`
		Regular string `json:"regular"`
	} `json:"urls"`
	User struct {
		Name  string `json:"name"`
		Links struct {
			HTML string `json:"html"`
		} `json:"links"`
	} `json:"user"`
	Links struct {
		HTML string `json:"html"`
	} `json:"links"`
}

type unsplashSearchPayload struct {
	Results []unsplashPhoto `json:"results"`
}

func (s *Server) handleUnsplashSearch(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(os.Getenv("UNSPLASH_ACCESS_KEY"))
	if key == "" {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"ok":    false,
			"error": "Unsplash kapalı: deploy/.env içine UNSPLASH_ACCESS_KEY ekleyin.",
		})
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Arama kelimesi gerekli."})
		return
	}
	runes := []rune(query)
	if len(runes) > 120 {
		query = string(runes[:120])
	}
	endpoint := "https://api.unsplash.com/search/photos?per_page=12&orientation=landscape&content_filter=high&query=" + url.QueryEscape(query)
	// #nosec G704 -- Hedef SABİT: şema+host kod içinde yazılı
	// (api.unsplash.com), kullanıcı girdisi yalnız query parametresi ve
	// url.QueryEscape'ten geçiyor. SSRF mümkün değil.
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "Unsplash isteği hazırlanamadı."})
		return
	}
	req.Header.Set("Authorization", "Client-ID "+key)
	req.Header.Set("Accept-Version", "v1")
	client := &http.Client{Timeout: 9 * time.Second}
	// #nosec G704 -- Hedef istek başında sabit api.unsplash.com adresine
	// kuruldu; buradaki istemci yalnız o isteği yürütür.
	resp, err := client.Do(req)
	if err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "Unsplash’a ulaşılamadı."})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "Unsplash araması başarısız (HTTP " + resp.Status + ")."})
		return
	}
	var payload unsplashSearchPayload
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "Unsplash yanıtı okunamadı."})
		return
	}
	results := make([]map[string]any, 0, len(payload.Results))
	for _, photo := range payload.Results {
		if photo.URLs.Regular == "" {
			continue
		}
		alt := strings.TrimSpace(photo.Alt)
		if alt == "" {
			alt = strings.TrimSpace(photo.Description)
		}
		credit := strings.TrimSpace(photo.User.Name)
		creditURL := strings.TrimSpace(photo.User.Links.HTML)
		if creditURL == "" {
			creditURL = strings.TrimSpace(photo.Links.HTML)
		}
		results = append(results, map[string]any{
			"id":        photo.ID,
			"url":       photo.URLs.Regular,
			"thumb":     photo.URLs.Small,
			"alt":       alt,
			"credit":    credit,
			"creditUrl": creditURL,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "results": results})
}
