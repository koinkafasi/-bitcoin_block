package templkit

import "strconv"

// ---- Kaynak bankası sağlık göstergeleri ----
//
// 12 Kasım 2026 denetiminde ortaya çıktı: kapalı bir kaynağın NEDEN
// kapalı olduğu listede görünmüyordu. 22 kapalı kaynağın hangisinin
// sağlam, hangisinin bozuk olduğu ancak tek tek çekilerek anlaşıldı.
// Bu yardımcılar o bilgiyi tabloya taşıyor.

// feedDilEtiketi dil rozeti. Boşsa İngilizce varsayılır (eski kayıtlar).
func feedDilEtiketi(dil string) string {
	if dil == "" {
		return "EN"
	}
	return dil
}

// feedPasifEtiketi kapalı kaynağın kısa gerekçesi.
//
// Sıra önemli: bot kapatmışsa bunu söylemek, "konu dışı" gibi insan
// kararından daha acil bir bilgi — kaynak düzeltilip açılabilir.
func feedPasifEtiketi(f AIKaynakFeed) string {
	if f.OtomatikKapandi {
		return "bot kapattı"
	}
	if f.KapanmaNedeni != "" {
		return f.KapanmaNedeni
	}
	return "pasif"
}

// feedGorselSinifi görsel oranına göre hücre sınıfı.
//
// Eşik 30: altındaki kaynaklar kart düzenini bozuyor (sitede görselsiz
// içerik için tipografik kapak üretiliyor ama her kart için üretmek
// listeyi tekdüze yapıyor).
func feedGorselSinifi(oran string) string {
	if oran == "" {
		return "small muted"
	}
	n, err := strconv.Atoi(oran)
	if err != nil {
		return "small muted"
	}
	if n < 30 {
		return "small feed-gorsel-dusuk"
	}
	return "small"
}

// kirpEtiket uzun hata metnini tabloda taşacak kadar uzamadan keser.
// Tam metin `title` özniteliğinde duruyor.
func kirpEtiket(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
