//go:build integration

package httpapi_test

import (
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

// adminGETRotalari server.go'dan GET /admin yollarını çıkarır.
func adminGETRotalari(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("server.go okunamadı: %v", err)
	}
	re := regexp.MustCompile(`\{"GET",\s*"(/admin[^"]*)"`)
	var out []string
	gorulen := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		yol := m[1]
		if gorulen[yol] {
			continue
		}
		gorulen[yol] = true
		out = append(out, yol)
	}
	if len(out) < 50 {
		t.Fatalf("yalnız %d GET /admin rotası bulundu; ayrıştırma bozulmuş olabilir", len(out))
	}
	return out
}

// TestPanelGETRotalariAciliyor: Yönetici oturumuyla her panel sayfası açılır.
func TestPanelGETRotalariAciliyor(t *testing.T) {
	skipIfNoDB(t)

	var uid int64
	if err := itPool.QueryRow(t.Context(), `
		INSERT INTO users (name, email, password_hash, role, status)
		VALUES ('Duman Testi','duman-yonetici@example.com','x','Yönetici','aktif')
		ON CONFLICT (email) DO UPDATE SET role='Yönetici', status='aktif'
		RETURNING id`).Scan(&uid); err != nil {
		t.Fatalf("kullanıcı: %v", err)
	}
	t.Cleanup(func() {
		_, _ = itPool.Exec(t.Context(), `DELETE FROM users WHERE id=$1`, uid)
	})

	client := panelIstemcisi(t, itHTTP.URL, uid, "")

	for _, yol := range adminGETRotalari(t) {
		// Yol parametreleri: var olmayan bir kimlikle de sayfa açılmalı
		// (kayıt yoksa 404/boş durum, panik değil).
		hedef := strings.NewReplacer(
			"{id}", "1", "{media}", "1", "{bicim}", "srt",
			"{lang}", "en", "{code}", "am", "{slug}", "test",
		).Replace(yol)

		t.Run(yol, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, itHTTP.URL+hedef, nil)
			if err != nil {
				t.Fatalf("istek: %v", err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("istek hatası: %v", err)
			}
			defer resp.Body.Close()
			govde, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

			if resp.StatusCode == http.StatusServiceUnavailable {
				if !strings.Contains(string(govde), `"ok":false`) {
					t.Fatalf("503 açıklamasız döndü — gövde: %s", ilkBayt(govde, 240))
				}
				return // dış bağımlılık kapalı: yapılandırma eksikliği, arıza değil
			}
			if resp.StatusCode >= 500 {
				t.Fatalf("HTTP %d — gövde: %s", resp.StatusCode, ilkBayt(govde, 240))
			}
			// Gövdede panik izi olmamalı (kurtarma katmanı 500'e çevirse de
			// metin sızabilir).
			if strings.Contains(strings.ToLower(string(govde)), "panic:") {
				t.Fatalf("gövdede panic izi: %s", ilkBayt(govde, 240))
			}
		})
	}
}

func ilkBayt(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
