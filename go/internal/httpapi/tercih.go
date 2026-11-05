package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode"

	"theblock/site/internal/repo"
)

// Kullanıcı arayüz tercihleri (Faz 173, A-3).
//
// NEDEN: içerik formundaki katlanabilir grupların açık/kapalı durumu ve
// daraltılmış yan menü yalnız localStorage'da tutuluyordu; aynı editör başka
// bir cihazda her şeyi baştan açmak zorunda kalıyordu.
//
// Sözleşme:
//   GET  — tercihleri sayfaya gömer (render sırasında meta etiketi).
//   POST — tek anahtar yazar; gövde {"k":"...","v":...}.
//
// İstemci localStorage'ı anlık önbellek olarak korumaya devam eder; sunucu
// cihazlar arası doğruluk kaynağıdır. Yazma son-yazan-kazanır.

// tercihAnahtarGecerli istemciden gelen anahtarı sınırlar.
//
// NEDEN: anahtarlar istemci tarafında üretiliyor. Sınırsız bırakılsaydı
// jsonb belgesi kontrolsüz büyüyebilir (tek kullanıcı için megabaytlar) ve
// uzun anahtarlarla sorgu maliyeti artardı.
func tercihAnahtarGecerli(k string) bool {
	if k == "" || len(k) > 80 {
		return false
	}
	// Yalnız harf (Türkçe dahil: "tb_form_grup:Künye"), rakam, iki nokta,
	// alt çizgi, nokta, tire ve boşluk. ASCII sınırı KOYULMAZ: grup
	// anahtarları ekrandaki Türkçe başlıktan türetiliyor.
	for _, r := range k {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
		case r == ':' || r == '_' || r == '-' || r == '.' || r == ' ':
		default:
			return false
		}
	}
	return true
}

func (s *Server) handleTercihKaydet(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	if u.ID <= 0 {
		http.Error(w, "oturum yok", http.StatusUnauthorized)
		return
	}
	var govde struct {
		K string          `json:"k"`
		V json.RawMessage `json:"v"`
	}
	// 4 KB yeter: bir boolean ya da kısa bir dize saklanıyor.
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&govde); err != nil {
		http.Error(w, "gövde okunamadı", http.StatusBadRequest)
		return
	}
	if !tercihAnahtarGecerli(govde.K) {
		http.Error(w, "geçersiz anahtar", http.StatusBadRequest)
		return
	}
	// Değer ham JSON olarak saklanır; istemci ne gönderdiyse o.
	if len(govde.V) == 0 || len(govde.V) > 512 {
		http.Error(w, "geçersiz değer", http.StatusBadRequest)
		return
	}
	var deger any
	if err := json.Unmarshal(govde.V, &deger); err != nil {
		http.Error(w, "geçersiz değer", http.StatusBadRequest)
		return
	}
	if err := repo.KullaniciTercihiKaydet(r.Context(), s.pool, u.ID, govde.K, deger); err != nil {
		s.logger.Error("tercih kaydetme", "error", err, "anahtar", govde.K)
		http.Error(w, "kaydedilemedi", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// tercihlerJSON sayfaya gömülecek tercih belgesini üretir.
//
// Hata durumunda boş nesne döner: tercih yüklenememesi sayfayı bozmamalı,
// istemci localStorage'daki değerle devam eder.
func (s *Server) tercihlerJSON(r *http.Request) string {
	u := userFromContext(r.Context())
	if u.ID <= 0 {
		return "{}"
	}
	m, err := repo.KullaniciTercihleri(r.Context(), s.pool, u.ID)
	if err != nil {
		s.logger.Warn("tercih okuma", "error", err)
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) == 0 {
		return "{}"
	}
	return string(b)
}

// tercihlerMetaJSON <meta content="..."> içinde güvenli olsun diye tek
// tırnak ve çift tırnak kaçırılır; html/template zaten kaçırıyor, bu ikinci
// kat savunma.
func tercihlerMetaJSON(k string) string {
	return strings.ReplaceAll(k, "'", "&#39;")
}
