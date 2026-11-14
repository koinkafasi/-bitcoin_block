//go:build integration

package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/config"
)

// OLAY KAYDI UÇTAN UCA — gerçek Postgres, gerçek middleware.
//
// NEDEN AYRI DOSYA + BUILD ETİKETİ: normal kapı veritabanı istemez.
// Bu test yalnız OLAY_DB_URL verildiğinde koşar (göçler uygulanmış bir
// veritabanı; bkz. deploy/scripts/e2e-izole.sh).
//
// SINANAN ZİNCİR: istek → withGuvenlik → karar → olayKaydet → kuyruk →
// işçi → toplu INSERT → okuma. Zincirin her halkası ayrı bir yerde
// kırılabilir; birim testleri yalnız ilk ve son halkayı görüyor.
func TestOlayKaydiUctanUca(t *testing.T) {
	url := os.Getenv("OLAY_DB_URL")
	if url == "" {
		t.Skip("OLAY_DB_URL yok")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("havuz: %v", err)
	}
	defer pool.Close()

	// Güvenlik katmanı açık ve engelleme AÇIK olmalı ki 403 yolu da sınansın.
	for _, k := range [][2]string{
		{"aktif", `"1"`}, {"bot_engel", `"1"`}, {"hosting_engel", `"1"`},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO settings (section,key,value) VALUES ('guvenlik',$1,$2::jsonb)
			ON CONFLICT (section,key) DO UPDATE SET value=EXCLUDED.value`, k[0], k[1]); err != nil {
			t.Fatalf("ayar: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM settings WHERE section='guvenlik'`)
	})

	s := &Server{
		cfg:    config.Config{Env: "test", BaseURL: "https://bitcosar.com"},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		pool:   pool,
	}

	// Kuyruğu boşalt ki önceki testlerin artığı karışmasın.
	k := olaylariBaslat()
	for len(k) > 0 {
		<-k
	}

	// İşçiyi başlat: 2 saniyelik turda yazacak.
	ictx, iptal := context.WithCancel(ctx)
	defer iptal()
	s.OlayIsci(ictx)

	okundu := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h := s.withGuvenlik(okundu)

	const anahtar = "test-uctan-uca"
	istekler := []struct {
		yol, ua  string
		beklenen int
	}{
		// Zafiyet yolu, tarayıcı UA'sı: sınıflandırma zafiyet olmalı.
		{"/wp-login.php?token=GIZLI123", "Mozilla/5.0 Chrome/120", http.StatusOK},
		// Otomasyon UA'sı: 90 puan → bot → 403.
		{"/haber/bitcoin", "python-requests/2.31.0", http.StatusForbidden},
		// Önbelleklenebilir içerik sayfası.
		{"/haber/ethereum", "Mozilla/5.0 Chrome/120", http.StatusOK},
	}
	for _, is := range istekler {
		r := httptest.NewRequest(http.MethodGet, is.yol, nil)
		r.RemoteAddr = "203.0.113.77:1234"
		r.Header.Set("User-Agent", is.ua)
		r.Header.Set("CF-Connecting-IP", "203.0.113.77")
		// Anahtarı sabitle: çerez olmadan "ip:<adres>" kullanılırdı ve
		// test IP'si başka testlerle çakışabilirdi.
		r.AddCookie(&http.Cookie{Name: vidCookie, Value: uuidSabit})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != is.beklenen {
			t.Errorf("%s → %d, beklenen %d", is.yol, w.Code, is.beklenen)
		}
	}
	_ = anahtar

	// İşçinin turunu bekle (2 sn) + yazma payı.
	var satirlar []struct {
		Yol  string
		Tur  string
		Dur  int
		Eng  bool
		Sorg string
	}
	bitis := time.Now().Add(8 * time.Second)
	for time.Now().Before(bitis) {
		satirlar = satirlar[:0]
		rows, err := pool.Query(ctx, `
			SELECT yol, tur, durum, engellendi, COALESCE(sorgu,'')
			  FROM visitor_events WHERE anahtar = $1 ORDER BY zaman`, uuidSabit)
		if err != nil {
			t.Fatalf("okuma: %v", err)
		}
		for rows.Next() {
			var s struct {
				Yol  string
				Tur  string
				Dur  int
				Eng  bool
				Sorg string
			}
			if err := rows.Scan(&s.Yol, &s.Tur, &s.Dur, &s.Eng, &s.Sorg); err == nil {
				satirlar = append(satirlar, s)
			}
		}
		rows.Close()
		if len(satirlar) >= 3 {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if len(satirlar) < 3 {
		t.Fatalf("3 olay beklenirdi, %d yazıldı (kuyruk: %d)", len(satirlar), len(k))
	}
	// 1) Zafiyet sınıflandırması ve JETON MASKELEMESİ.
	if satirlar[0].Yol != "/wp-login.php" || satirlar[0].Tur != turZafiyet {
		t.Errorf("zafiyet sınıflandırması yanlış: %+v", satirlar[0])
	}
	if satirlar[0].Sorg != "token=***" {
		t.Errorf("jeton maskelenmedi: %q", satirlar[0].Sorg)
	}
	// 2) Bot engellendi ve 403 kaydedildi.
	if satirlar[1].Dur != 403 || !satirlar[1].Eng {
		t.Errorf("403/engel kaydı yanlış: %+v", satirlar[1])
	}
	// 3) Temiz içerik isteği 200 ve içerik türünde.
	if satirlar[2].Tur != turIcerik || satirlar[2].Dur != 200 {
		t.Errorf("içerik kaydı yanlış: %+v", satirlar[2])
	}

	// Temizlik.
	if _, err := pool.Exec(ctx,
		`DELETE FROM visitor_events WHERE anahtar = $1`, uuidSabit); err != nil {
		t.Logf("temizlik: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`DELETE FROM visitor_risk WHERE anahtar = $1`, uuidSabit); err != nil {
		t.Logf("temizlik 2: %v", err)
	}
}

// uuidSabit testin kullandığı sabit ziyaretçi kimliği.
const uuidSabit = "11111111-2222-4333-8444-555555555555"
