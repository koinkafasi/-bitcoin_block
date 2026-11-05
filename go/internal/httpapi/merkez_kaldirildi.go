package httpapi

import "net/http"

// handleMerkezKaldirildi — Site Yönetim Merkezi (/admin/merkez) kaldırıldı.
//
// Neden tamamen silinmedi, yönlendirme bırakıldı: sayfa kullanılmıyordu
// (iframe kabuğuydu, her bölüm kendi gerçek sayfasına gidiyordu), ama
// kullanıcının yer imlerinde ve tarayıcı geçmişinde duruyor olabilir.
// Kalıcı yönlendirme 404 göstermez; ayrıca arama motorları eski adresi
// panoya taşır.
//
// Kalıcı (301) seçildi: bu adres bir daha geri gelmeyecek.
func (s *Server) handleMerkezKaldirildi(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin", http.StatusMovedPermanently)
}
