package httpapi

import (
	"context"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// ads.txt ÜRETİMİ — GELİR BÜTÜNLÜĞÜ.
//
// NEDEN ÖNEMLİ: ads.txt, "bu alan adındaki reklam envanterini KİMLER
// satabilir" beyanıdır. Dosya boş ya da eksikse:
//   · domain spoofing'e açıksın — başkası bizim envanterimizi kendi adına
//     satabilir (gelir ona gider, ceza bize kalır),
//   · SSP'ler (AdSense dahil) doğrulanmamış envantere daha düşük teklif verir
//     ya da hiç teklif vermez.
// Ölçüm (13 Kasım 2026): dosya yalnız `# theblock ads.txt — reklam ağları
// eklenecek` yorumundan ibaretti; hiçbir ağ beyan edilmemişti.
//
// TASARIM — UYDURMA YOK: burada yalnız DOĞRULANABİLİR satır üretilir.
//   · AdSense satırı, paneldeki yayıncı kimliği GEÇERLİYSE yazılır
//     (`ca-pub-…`). Kimlik yoksa satır hiç basılmaz — sahte bir pub kimliği
//     yazmak geliri başkasına yönlendirmek olurdu.
//   · Diğer ağların satırlarını panelden operatör yapıştırır; her satır
//     biçim denetiminden geçer (aşağıda). Coinzilla'nın satıcı kimliğini
//     BİLMİYORUZ, bu yüzden onun satırını uydurmuyoruz.
//
// Çıktı asla HTML içermez: dosya text/plain ve satırlar katı doğrulanır
// (panel ele geçse bile sayfaya kod enjekte edilemez).

// adsTxtMaxBayt üst sınır: ads.txt küçük bir dosyadır; 32 KB fazlasıyla
// yeterli ve kötüye kullanımı (dev dosya) engeller.
const adsTxtMaxBayt = 32 << 10

// adsTxtSatirGecerli mi: `domain, publisher-id, RELATIONSHIP[, cert-id]`.
//
// KRİTİK: satır içinde `<`, `>`, tırnak, ters bölü, noktalı virgül veya
// kontrol karakteri GEÇEMEZ. Bu karakterler dosyayı bozar ya da (bir gün
// farklı bir içerik türünde sunulursa) enjeksiyon yüzeyi olur.
func adsTxtSatirGecerli(satir string) bool {
	s := strings.TrimSpace(satir)
	if s == "" {
		return false
	}
	// Yorum satırına izin var (# ile başlar) ama yalnız kısa ve zararsız.
	if strings.HasPrefix(s, "#") {
		return len(s) <= 200 && !strings.ContainsAny(s, "<>\\;\x00\n\r")
	}
	if strings.ContainsAny(s, "<>\"\\;\x00\r\n\t") {
		return false
	}
	parca := strings.Split(s, ",")
	if len(parca) < 3 || len(parca) > 4 {
		return false
	}
	alan := make([]string, len(parca))
	for i, p := range parca {
		alan[i] = strings.TrimSpace(p)
		if alan[i] == "" {
			return false
		}
	}
	// domain: şema yok, yol yok, en az bir nokta var.
	domain := strings.ToLower(alan[0])
	if strings.Contains(domain, "/") || strings.Contains(domain, ":") || !strings.Contains(domain, ".") {
		return false
	}
	for _, r := range domain {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '-') {
			return false
		}
	}
	// publisher id: harf/rakam/tire/alt çizgi.
	for _, r := range alan[1] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	// ilişki türü: DIRECT veya RESELLER (ads.txt standardı).
	iliski := strings.ToUpper(alan[2])
	if iliski != "DIRECT" && iliski != "RESELLER" {
		return false
	}
	// sertifika yetkilisi kimliği (varsa) hex.
	if len(alan) == 4 {
		for _, r := range alan[3] {
			if !(r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F' || r >= '0' && r <= '9') {
				return false
			}
		}
	}
	return true
}

// adsTxtGovde ads.txt içeriğini ayarlardan üretir.
func (s *Server) adsTxtGovde(ctx context.Context) string {
	var b strings.Builder

	// Havuz yoksa ayar okunamaz; "hiç beyan yok" gibi davran. Paniklemek
	// yerine biçimsel olarak geçerli bir dosya döndürmek doğru: /ads.txt
	// 500 vermemeli (reklam ağları dosyayı yok sayar, gelir kaybı olur).
	if s.pool != nil {
		// 1) AdSense — yalnız GEÇERLİ yayıncı kimliği varsa.
		//
		// `f08c47fec0942fa0`, Google'ın ads.txt belgelerinde sabit olarak
		// yayımladığı sertifika yetkilisi kimliğidir; yayıncıya özel değildir.
		if pub := s.adsPublisher(ctx); pub != "" {
			b.WriteString("google.com, ")
			b.WriteString(strings.TrimPrefix(pub, "ca-")) // ca-pub-X → pub-X
			b.WriteString(", DIRECT, f08c47fec0942fa0\n")
		}

		// 2) Operatörün doğrulanmış ek satırları.
		if ek := string(s.cachedSetting(ctx, "baglantilar", "ads_txt_ek")); ek != "" {
			toplam := 0
			for _, satir := range strings.Split(ek, "\n") {
				satir = strings.TrimSpace(satir)
				if satir == "" {
					continue
				}
				if !adsTxtSatirGecerli(satir) {
					s.logger.Warn("ads.txt: geçersiz satır atlandı", "satir", kisalt(satir, 80))
					continue
				}
				if toplam+len(satir)+1 > adsTxtMaxBayt {
					s.logger.Warn("ads.txt: boyut sınırı aşıldı, kalan satırlar atlandı")
					break
				}
				toplam += len(satir) + 1
				b.WriteString(satir)
				b.WriteString("\n")
			}
		}
	}

	// 3) Hiç beyan yoksa dosya BOŞ değil, temiz bir yorum olsun: boş dosya
	// "ads.txt yok" ile aynı sayılır ama en azından biçim geçerlidir.
	if b.Len() == 0 {
		b.WriteString("# ads.txt: panelden yayıncı kimliği veya ağ satırı ekleyin\n")
	}
	return b.String()
}

// handleAdsTxt /ads.txt sunar.
func (s *Server) handleAdsTxt(w http.ResponseWriter, r *http.Request) {
	govde := s.adsTxtGovde(r.Context())
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// Reklam ağları bu dosyayı periyodik okur; 10 dk tazelik yeter ve
	// panel değişikliği makul sürede yansır.
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(govde))
}

// adsTxtRepoYardimcisi derleme hatasını önlemek için repo paketini kullanır
// (cachedSetting repo'ya bağlı); ileride doğrudan sorgu gerekirse burada.
var _ = repo.GetSetting
