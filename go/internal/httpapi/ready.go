package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// ---- Hazırlık (readiness) sondası ----
//
// `/healthz` süreç AYAKTA mı sorusunu yanıtlar (canlılık). Yük dengeleyici
// ve systemd için yeterli değildir: süreç ayakta olup veritabanı erişilemez
// durumda olabilir. O hâlde trafik göndermek kullanıcıya 500 göstermek olur.
//
// `/readyz` trafiğe HAZIR mı sorusunu yanıtlar:
//   - kapanış başladıysa 503 (uçuştaki istekler bitsin, yenisi gelmesin),
//   - veritabanı ping'i başarısızsa 503,
//   - aksi hâlde 200.
//
// Yanıt gövdesi JSON'dur; sır içermez (yalnız durum ve gecikme).

const readyzPingTimeout = 2 * time.Second

// readyzDurum süreç seviyesindeki hazırlık durumunu tutar.
// 0 = kapanıyor, 1 = hazır.
var readyzDurum atomic.Int32

func init() { readyzDurum.Store(1) }

// SetReady kapanış başlarken false yapılır: yeni trafik kabul edilmesin.
func SetReady(v bool) {
	if v {
		readyzDurum.Store(1)
		return
	}
	readyzDurum.Store(0)
}

// Ready sunucunun trafiğe hazır olup olmadığını bildirir (testler için).
func Ready() bool { return readyzDurum.Load() == 1 }

// handleReadyz GET /readyz — trafiğe hazırlık sondası.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	type yanit struct {
		Status string `json:"status"`          // "ready" | "draining" | "db-unavailable"
		DB     string `json:"db"`              // "ok" | "fail"
		DBms   int64  `json:"db_ms,omitempty"` // ping gecikmesi
	}

	if !Ready() {
		// Kapanış sürüyor: yeni istek almıyoruz ama süreç yaşıyor.
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(yanit{Status: "draining", DB: "ok"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), readyzPingTimeout)
	defer cancel()

	bas := time.Now()
	var one int
	err := s.pool.QueryRow(ctx, "SELECT 1").Scan(&one)
	gecen := time.Since(bas).Milliseconds()

	if err != nil {
		s.logger.Warn("readyz: veritabanı erişilemiyor", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(yanit{Status: "db-unavailable", DB: "fail"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(yanit{Status: "ready", DB: "ok", DBms: gecen})
}
