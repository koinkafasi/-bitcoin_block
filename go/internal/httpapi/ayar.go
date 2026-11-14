package httpapi

import (
	"context"
	"encoding/json"
	"strings"
)

// AYAR OKUMA — tek yer.
//
// NEDEN: `cachedSetting` ham `jsonb` baytı döndürür. Panelden yazılan değer
// JSON string ise `"0"` (tırnaklı) gelir. Doğrudan `string(b) != "0"`
// karşılaştırması bu yüzden HİÇBİR ZAMAN eşleşmez: tırnaklı `"0"` üç
// karakterdir. Ölçüldü (14 Kasım 2026): `konumZorunlu` bu yüzden ayar
// kapalıyken de açık dönüyordu — kapı kapatılamıyordu.
//
// Tek yardımcı: üç biçimi de (JSON string, JSON sayı/bool, ham metin) aynı
// sonuca indirger. Yeni güvenlik ayarları da bunu kullanır.
func (s *Server) ayarMetin(ctx context.Context, bolum, anahtar, varsayilan string) string {
	ham := s.cachedSetting(ctx, bolum, anahtar)
	if len(ham) == 0 {
		return varsayilan
	}
	var v string
	if json.Unmarshal(ham, &v) == nil {
		if v = strings.TrimSpace(v); v == "" {
			return varsayilan
		}
		return v
	}
	// Sayı/bool/nesne: ham JSON'u olduğu gibi metne çevir.
	if t := strings.TrimSpace(string(ham)); t != "" && t != "null" {
		return strings.Trim(t, `"`)
	}
	return varsayilan
}

// ayarAcik bool ayarı okur: "1", "true", "on", "evet" → true.
//
// Varsayılan çağıranın kararı: güvenlik ayarlarında "kapalıysa koruma yok"
// demek yerine çağıran açıkça söyler.
func (s *Server) ayarAcik(ctx context.Context, bolum, anahtar string, varsayilan bool) bool {
	v := strings.ToLower(strings.TrimSpace(s.ayarMetin(ctx, bolum, anahtar, "")))
	switch v {
	case "":
		return varsayilan
	case "1", "true", "on", "evet", "açık", "acik":
		return true
	}
	return false
}

// ayarSayi int ayarı okur; geçersizse varsayılana döner.
func (s *Server) ayarSayi(ctx context.Context, bolum, anahtar string, varsayilan, alt, ust int) int {
	v := s.ayarMetin(ctx, bolum, anahtar, "")
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return varsayilan
		}
		n = n*10 + int(c-'0')
		if n > 1000000 {
			return varsayilan
		}
	}
	if v == "" || n < alt || n > ust {
		return varsayilan
	}
	return n
}
