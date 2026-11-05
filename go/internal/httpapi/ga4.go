package httpapi

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

// GA4 ölçüm kodu ve mülk kimliği.
//
// Tasarım kuralları:
//
//  1. **Varsayılan izleme yok.** Ölçüm kimliği boşsa public sayfa hiçbir
//     üçüncü taraf kodu basmaz. Ziyaretçi verisi toplamak bilinçli bir
//     karardır; yönetici panelden açar (KVKK/çerez politikası ile uyumlu).
//
//  2. **Doğrulanmadan basılmaz.** Değer panelden gelen serbest metindir;
//     doğrudan <script> etiketine yazmak enjeksiyon yüzeyi açar. Biçimi
//     tutmayan değer sessizce yok sayılır.
//
//  3. **Mülk kimliği tek kaynaktan okunur.** Aynı değer geçmişte iki ayrı
//     ayarda tutuluyordu (trend/ga4_property ve adsense/ga_property); biri
//     doldurulup diğeri unutulduğunda panelin bir yarısı "yapılandırılmadı"
//     diyordu. Artık tek okuma yolu var; eski adsense ayarı yalnızca geriye
//     dönük yedek olarak okunur.

// ga4MeasurementRe GA4 ölçüm kimliği: "G-" + en az 4 harf/rakam.
var ga4MeasurementRe = regexp.MustCompile(`^G-[A-Z0-9]{4,}$`)

// ga4PropertyRe GA4 mülk kimliği: yalnız rakam.
var ga4PropertyRe = regexp.MustCompile(`^[0-9]{4,}$`)

// validGA4Measurement ölçüm kimliğini doğrular ve normalleştirir.
// Geçersizse boş döner — çağıran boş değeri "izleme kapalı" olarak yorumlar.
func validGA4Measurement(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !ga4MeasurementRe.MatchString(s) {
		return ""
	}
	return s
}

// normalizeGA4PropertyID mülk kimliğini temizler. Panelde kullanıcı bazen
// API biçimini ("properties/561087489") yapıştırıyor; sayı kısmı alınır.
func normalizeGA4PropertyID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "properties/")
	if !ga4PropertyRe.MatchString(s) {
		return ""
	}
	return s
}

// ga4Measurement yapılandırılmış ölçüm kimliğini döndürür (yoksa boş).
//
// Her public sayfa render'ında okunduğu için TTL önbellekli cachedSetting
// kullanılır; ayar değişikliği önbellek süresi dolunca yayılır.
func (s *Server) ga4Measurement(ctx context.Context) string {
	return validGA4Measurement(string(s.cachedSetting(ctx, "trend", "ga4_olcum")))
}

// ga4PropertyID GA4 mülk kimliğini tek kaynaktan döndürür. Kanonik ayar
// trend/ga4_property'dir; boşsa eski adsense/ga_property okunur (geriye
// dönük yedek — iki ayrı ayar tutma hatasının kalıntısı).
func (s *Server) ga4PropertyID(ctx context.Context) string {
	if id := normalizeGA4PropertyID(string(s.cachedSetting(ctx, "trend", "ga4_property"))); id != "" {
		return id
	}
	return normalizeGA4PropertyID(string(s.cachedSetting(ctx, "adsense", "ga_property")))
}

// gVerification Search Console site doğrulama kodunu döndürür.
//
// Neden: panelde "Google Site Doğrulama" alanı vardı, değer kaydediliyordu
// ama hiçbir sayfada basılmıyordu — yönetici kodu giriyor, Search Console
// doğrulaması yine başarısız oluyordu. Kod artık public <head>'e meta
// etiketi olarak basılır (boşsa hiç basılmaz).
//
// Değer serbest metindir; doğrudan HTML'e yazmak enjeksiyon yüzeyi açar.
// Bu yüzden yalnız Google'ın ürettiği biçim kabul edilir: harf, rakam,
// tire ve alt çizgi, en çok 120 karakter.
func (s *Server) gVerification(ctx context.Context) string {
	ham := s.cachedSetting(ctx, "baglantilar", "meta")
	if len(ham) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(ham, &m); err != nil {
		return ""
	}
	s2, _ := m["g_verification"].(string)
	return validGVerification(s2)
}

// gVerificationRe Google doğrulama kodu biçimi.
var gVerificationRe = regexp.MustCompile(`^[A-Za-z0-9_-]{8,120}$`)

// validGVerification kodu doğrular; biçim tutmuyorsa boş döner.
func validGVerification(s string) string {
	s = strings.TrimSpace(s)
	// Kullanıcı bazen tam etiketi yapıştırır:
	// <meta name="google-site-verification" content="XXXX" />
	if i := strings.Index(s, "content="); i >= 0 {
		rest := s[i+len("content="):]
		rest = strings.TrimPrefix(rest, "\"")
		rest = strings.TrimPrefix(rest, "'")
		if j := strings.IndexAny(rest, "\"' >"); j >= 0 {
			rest = rest[:j]
		}
		s = strings.TrimSpace(rest)
	}
	if !gVerificationRe.MatchString(s) {
		return ""
	}
	return s
}

// adsPublisher AdSense yayıncı kimliğini (ca-pub-…) doğrulanmış olarak döndürür.
//
// Neden: Ayarlar → "AdSense (pub-id)" alanı vardı, değer kaydediliyordu ama
// hiçbir sayfada kullanılmıyordu. Google, AdSense site doğrulaması için
// <meta name="google-adsense-account" content="ca-pub-…"> etiketini istiyor;
// alan artık oraya basılır. Boş/geçersiz değer hiç basılmaz.
func (s *Server) adsPublisher(ctx context.Context) string {
	return validAdsensePublisher(string(s.cachedSetting(ctx, "baglantilar", "adsense")))
}

// adsPublisherRe AdSense yayıncı kimliği biçimi.
var adsPublisherRe = regexp.MustCompile(`^ca-pub-[0-9]{10,20}$`)

// validAdsensePublisher kimliği doğrular; biçim tutmuyorsa boş döner.
func validAdsensePublisher(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	// Tam etiket yapıştırıldıysa içindeki değeri al.
	if i := strings.Index(s, "ca-pub-"); i > 0 {
		s = s[i:]
	}
	if j := strings.IndexAny(s, `"' >`); j > 0 {
		s = s[:j]
	}
	if !adsPublisherRe.MatchString(s) {
		return ""
	}
	return s
}
