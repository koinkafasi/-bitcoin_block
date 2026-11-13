package ai

import (
	"context"
	"errors"
	"net"
)

// ZamanAsimiHatasi hatanın zaman aşımı/iptal kaynaklı olup olmadığını söyler.
//
// NEDEN VAR: aieditor, JSON modunu desteklemeyen uçlar için CompleteStrict'i
// bir kez de JSON modu olmadan deniyor. O yedek YERİNDE — ama her hatada
// değil. 14 Kasım 2026 ölçümü:
//
//	BAŞARISIZ işler : medyan 240.0 sn, en uzun 240.1  (136'nın 128'i tam burada)
//	BAŞARILI işler  : medyan  73.7 sn, p90 194.7
//
// 240 = 2 × 120 (HTTP istemci zaman aşımı). Yani sağlayıcı yanıt vermeyince
// 120 saniye bekleniyor, sonra AYNI yavaş sağlayıcıya aynı şey tekrar
// soruluyor, 120 saniye daha bekleniyor. İkinci denemenin kazandırdığı bir
// şey yok: zaman aşımı JSON modundan kaynaklanmıyor. Maliyeti ise gerçek —
// işçi yuvası 4 dakika tutuluyor ve sağlayıcıya ikinci bir çağrı gidiyor.
//
// Bu yüzden yedek YALNIZ biçim reddi olabilecek hatalarda denenir; zaman
// aşımında iş hemen başarısız sayılır ve yuva boşalır.
func ZamanAsimiHatasi(err error) bool {
	if err == nil {
		return false
	}
	// Client.Timeout ve bağlam süresi ikisi de buraya düşer: Go'nun http
	// istemcisi her ikisini de context.DeadlineExceeded sarmalıyla döndürür.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	// Ağ katmanı zaman aşımları (bağlantı kurulamadı, okuma süresi doldu).
	var nerr net.Error
	if errors.As(err, &nerr) {
		return nerr.Timeout()
	}
	return false
}
