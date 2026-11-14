package repo

import (
	"context"
	"sort"
	"strings"
	"time"
)

// ---- Başarısızlık nedenlerini imzaya göre gruplama ----
//
// NEDEN (ölçüm, 14 Kasım 2026 — canlı `ai_jobs`, son 48 saat):
//
//	252 başarısız işin 236'sı HTTP istemci zaman aşımı, 16'sı başka şey
//	(süreç kapanması, 502, yetki hatası).
//
// Panelin ilk sürümü `left(error, 80)` ile gruplıyordu ve bu AYIRMIYOR:
//
//	AI sağlayıcısına ulaşılamıyor: AI sağlayıcısına ulaşılamadı: Post
//	"http://omniroute.omniroute.svc.cluster.local:20128/v1/chat/completions":
//	  ^--- buraya kadar ~139 karakter ve hepsinde AYNI ---^  sonra sebep
//
// Yani ayırt edici kısım 80. karakterden çok sonra başlıyor; dört ayrı
// sebep tek satıra çöküyordu. Koddaki "80 karakter sağlayıcı/hata türünü
// ayırmaya yetiyor" yorumu ölçümle yanlışlandı: adres sabit, sebep sonda.
//
// Sınıflandırma SQL yerine Go'da: 24 saatte birkaç yüz satır var, taşıma
// maliyeti yok ve böylece imza kuralları DB olmadan test edilebiliyor.
// Bu dosyanın asıl değeri de o testte.

// UretimHataImza bir başarısızlık sınıfıdır.
type UretimHataImza struct {
	Imza  string
	Adet  int
	Son   time.Time
	Ornek string
}

// Hata imzaları. Sabit olarak duruyorlar ki panel ve test aynı adı görsün.
const (
	ImzaZamanAsimi   = "zaman aşımı (HTTP istemci sınırı)"
	ImzaBaglamBitti  = "bağlam süresi doldu / iptal"
	ImzaSurecKapandi = "süreç kapandı (sevk veya OOM)"
	ImzaKota         = "kota doldu (429/402)"
	ImzaSaglayici5xx = "sağlayıcı hatası (5xx)"
	ImzaYetki        = "yetki reddi (401/403)"
	ImzaJSONDegil    = "çıktı geçerli JSON değil"
	ImzaBosIcerik    = "boş içerik (düşünme adımında kaldı)"
	ImzaKesikCikti   = "çıktı token sınırında kesildi"
	ImzaKaynakYok    = "araştırma kaynağı bulunamadı"
	ImzaDiger        = "diğer"
)

// UretimHataImzasi bir hata metnini sınıfına çevirir.
//
// Sıra önemli: en belirgin imza önce gelir. Örneğin zaman aşımı metni
// hem "Post http://..." hem "Client.Timeout" içeriyor; adrese bakan bir
// kural onu "sağlayıcı hatası" sayardı.
func UretimHataImzasi(hata string) string {
	h := strings.ToLower(hata)
	switch {
	case h == "":
		return ImzaDiger
	case strings.Contains(h, "client.timeout"):
		return ImzaZamanAsimi
	case strings.Contains(h, "boş içerik") || strings.Contains(h, "düşünme adımında"):
		return ImzaBosIcerik
	case strings.Contains(h, "token sınırında kesildi"):
		return ImzaKesikCikti
	case strings.Contains(h, "kotası doldu") ||
		strings.Contains(h, "uç noktası 429") || strings.Contains(h, "uç noktası 402"):
		return ImzaKota
	case strings.Contains(h, "terminated signal"):
		return ImzaSurecKapandi
	case strings.Contains(h, "context deadline exceeded") ||
		strings.Contains(h, "context canceled") || strings.Contains(h, "context ca"):
		return ImzaBaglamBitti
	case strings.Contains(h, "geçerli json değil"):
		return ImzaJSONDegil
	case strings.Contains(h, "uç noktası 401") || strings.Contains(h, "uç noktası 403") ||
		strings.Contains(h, "auth — [401]"):
		return ImzaYetki
	case strings.Contains(h, "uç noktası 5"):
		return ImzaSaglayici5xx
	case strings.Contains(h, "kaynak bulunamadı"):
		return ImzaKaynakYok
	}
	return ImzaDiger
}

// uretimHataOrnek uzun hata metnini iki ucunu koruyarak kısaltır.
//
// Yalnız baş kısmı göstermek işe yaramıyor (hepsi aynı adres), yalnız son
// kısmı göstermek hangi sağlayıcı olduğunu gizliyor. İki uç birlikte
// hem yolu hem sebebi veriyor.
func uretimHataOrnek(hata string) string {
	h := strings.TrimSpace(hata)
	if h == "" {
		return "(boş)"
	}
	r := []rune(h)
	const uc = 55
	if len(r) <= 2*uc+3 {
		return h
	}
	return string(r[:uc]) + " … " + string(r[len(r)-uc:])
}

// UretimHataImzalariOku son 24 saatin başarısızlıklarını imzaya göre
// gruplar; her sınıf için adet, en son an ve bir örnek verir.
func UretimHataImzalariOku(ctx context.Context, pool DB, limit int) ([]UretimHataImza, error) {
	if limit <= 0 {
		limit = 8
	}
	// Üst sınır: panel sayfası patolojik bir günde de sınırlı iş yapsın.
	rows, err := pool.Query(ctx, `
		SELECT COALESCE(error, ''), created_at
		  FROM ai_jobs
		 WHERE status = 'failed'
		   AND created_at > now() - interval '24 hours'
		 ORDER BY created_at DESC
		 LIMIT 5000`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kova := map[string]*UretimHataImza{}
	for rows.Next() {
		var hata string
		var an time.Time
		if err := rows.Scan(&hata, &an); err != nil {
			return nil, err
		}
		imza := UretimHataImzasi(hata)
		k, ok := kova[imza]
		if !ok {
			k = &UretimHataImza{Imza: imza}
			kova[imza] = k
		}
		k.Adet++
		// Satırlar en yeniden eskiye geliyor: ilk görülen en yenisidir.
		if k.Son.IsZero() {
			k.Son = an
			k.Ornek = uretimHataOrnek(hata)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]UretimHataImza, 0, len(kova))
	for _, k := range kova {
		out = append(out, *k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Adet != out[j].Adet {
			return out[i].Adet > out[j].Adet
		}
		return out[i].Imza < out[j].Imza
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
