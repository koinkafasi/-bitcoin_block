package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Kullanıcı arayüz tercihleri (Faz 173, A-3).
//
// NEDEN: içerik formundaki katlanabilir grupların açık/kapalı durumu ve
// daraltılmış yan menü yalnız localStorage'da tutuluyordu. Tercih başka
// cihaza taşınmıyordu.
//
// Tasarım: istemci localStorage'ı ANLIK önbellek olarak tutmaya devam eder
// (ilk boyamada FOUC olmasın, ağ yoksa da çalışsın); sunucu ise cihazlar
// arası doğruluk kaynağıdır. Yazma yolu tek yönlü ve son-yazan-kazanır:
// aynı anahtar iki cihazda farklıysa en son kaydeden geçerlidir. Çakışma
// çözümü (varlık birleştirme) bu ölçekte gereksiz karmaşıklık olurdu.

// KullanıcıTercihleri kullanıcının tüm tercihlerini döndürür.
// Kayıt yoksa boş (nil değil) harita döner: istemci tarafında null kontrolü
// gerekmesin.
func KullaniciTercihleri(ctx context.Context, pool *pgxpool.Pool, userID int64) (map[string]any, error) {
	out := map[string]any{}
	if userID <= 0 {
		return out, nil
	}
	var raw []byte
	err := pool.QueryRow(ctx,
		`SELECT prefs FROM user_prefs WHERE user_id = $1`, userID).Scan(&raw)
	if err != nil {
		// Kayıt yok: hata değil, boş tercih.
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return out, err
	}
	if len(raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}, err
	}
	return out, nil
}

// KullaniciTercihiKaydet tek bir anahtarı yazar. Değer jsonb olarak
// saklanır; diğer anahtarlar korunur (jsonb || birleştirme).
func KullaniciTercihiKaydet(ctx context.Context, pool *pgxpool.Pool, userID int64, anahtar string, deger any) error {
	if userID <= 0 {
		return fmt.Errorf("geçersiz kullanıcı")
	}
	if anahtar == "" {
		return fmt.Errorf("boş anahtar")
	}
	ham, err := json.Marshal(deger)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO user_prefs (user_id, prefs)
		VALUES ($1, jsonb_build_object($2::text, $3::jsonb))
		ON CONFLICT (user_id) DO UPDATE
		SET prefs = user_prefs.prefs || jsonb_build_object($2::text, $3::jsonb),
		    updated_at = now()`,
		userID, anahtar, string(ham))
	return err
}
