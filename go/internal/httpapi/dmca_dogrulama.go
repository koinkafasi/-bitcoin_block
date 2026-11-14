package httpapi

import "net/http"

// dmcaDogrulamaJeton — DMCA.com'un site sahipliği doğrulaması için istediği
// değer (14 Kasım 2026). Panelden BİREBİR kopyalandı; tek karakter değişirse
// doğrulama düşer.
//
// DMCA iki yöntem sunuyor: `<meta name="dmca-site-verification" …>` etiketi
// ya da kökte bu jetonu taşıyan /dmca-validation.html dosyası. İkisi de
// yayında: etiket webui/templkit/cd_head.templ içinde, dosya burada.
// Hangisini denetlediğini baştan bilmiyoruz, ikisini de karşılamak bedava.
const dmcaDogrulamaJeton = "UlpPYStLTW94b1g3aW96c25tRWc0Zz090"

// handleDMCADogrulama /dmca-validation.html sunar.
//
// Yol KÖKTEN isteniyor; /static/ altına konan dosya işe yaramaz. Jeton 33
// bayt olduğu için gömülü dosya yerine sabit tutuluyor — ayrı bir dosya
// yolu, embed listesi ve disk bağımlılığı getirirdi.
//
// Gövde düz jeton, HTML etiketi YOK: DMCA'nın denetleyicisi içeriği metin
// olarak okuyor. `text/html` seçildi çünkü dosya uzantısı .html; tarayıcıda
// açıldığında jeton görünür, başka bir şey olmaz. `nosniff` ile tarayıcının
// içeriği farklı bir tür sanması engelleniyor.
func (s *Server) handleDMCADogrulama(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(dmcaDogrulamaJeton))
}
