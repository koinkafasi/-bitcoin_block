package httpapi

import (
	"net/http"
	"strings"

	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

// ---- Panel yetkilendirmesi ----
//
// Yetki hiyerarşisi auth.roleRank'te tanımlıdır:
// Yönetici > Editör > Muhabir > Yazar > Üye.
//
// requireAdmin her panel isteğinde önce "personel mi" (auth.StaffRoles)
// sonra rota bazlı asgari rolü denetler. Üye rolü, oturumu olsa bile panelin
// hiçbir yerine giremez.

// adminOwnerPrefixes yalnızca Yönetici'nin erişebildiği alanlar: site/servis
// yapılandırması, ödeme ve abone verileri, sosyal medya anahtarları, ekip.
//
// `/admin/ses-ayarlari` burada: sayfa sağlayıcı API anahtarını ve dış servis
// adresini (base_url) kaydediyor. Editör seviyesinde bırakılsaydı, kendi
// anahtarını yazabilen biri transkripsiyonu kendi sunucusuna yönlendirebilir,
// mevcut anahtarı silebilir ya da ücretli sağlayıcıyı değiştirebilirdi.
//
// `/admin/trend/gsc` ve `/admin/trend/ga4`: aynı ayarların `/admin/ayarlar/gsc`
// altındaki ikinci kapısı. O kapı Yönetici'nde, bu kapı Yazar'daydı — yani
// aynı veriye iki farklı yetki seviyesinden erişilebiliyordu.
//
// `/admin/trend/kaynak`: kaynak adresi sunucuya çektirilir (SSRF yüzeyi) ve
// tüm sitenin gündem akışını besler.
var adminOwnerPrefixes = []string{
	"/admin/ayarlar",
	"/admin/servisler",
	"/admin/adsense",
	"/admin/siteler",
	"/admin/site-sec",
	"/admin/members",
	"/admin/aboneler",
	"/admin/paylasim",
	"/admin/gazete/ekip",
	"/admin/ses-ayarlari",
	"/admin/trend/gsc",
	"/admin/trend/ga4",
	"/admin/trend/kaynak",
	"/admin/trend/kaynak-ekle",
}

// adminEditorPrefixes düzen ve yayın yapılandırması: Editör ve üstü.
// (Muhabir/Yazar içerik üretir; bunlar site genelini değiştiren alanlar.)
var adminEditorPrefixes = []string{
	"/admin/reklamlar",
	"/admin/reklam-yerleri",
	"/admin/tasarim",
	"/admin/marka",
	"/admin/gorunum",
	"/admin/header",
	"/admin/footer",
	"/admin/arka-plan",
	"/admin/menu",
	"/admin/bilesenler",
	"/admin/kategori-panelleri",
	"/admin/tab-kategorileri",
	"/admin/manset-editor",
	"/admin/seo",
	"/admin/indexleme",
	"/admin/backlink",
	"/admin/ceviriler",
	"/admin/bildirim",
	"/admin/bulten",
	"/admin/raporlar",
	"/admin/olcu",
	// İletişim: yalnız adres/telefon değil, SİTE ALAN ADI ve tüm sayfa
	// gövdelerindeki alan adı metinleri tek istekte değişiyor
	// (repo.ReplaceTextInPages). Site genelini yeniden yazan bir işlem
	// içerik üreticisinin değil, yayın yapılandırmasının işi.
	"/admin/iletisim",
}

// adminMinRole bir panel rotası için asgari rolü döndürür.
// Bilinmeyen rotalar en düşük personel rolünü (Yazar) ister; böylece yeni
// eklenen bir rota yanlışlıkla daha geniş yetki vermez.
func adminMinRole(path string) string {
	if hasPathPrefix(path, adminOwnerPrefixes) {
		return "Yönetici"
	}
	if hasPathPrefix(path, adminEditorPrefixes) {
		return "Editör"
	}
	return "Yazar"
}

func hasPathPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

// adminAllowed rolün verilen panel rotasına erişip erişemeyeceğini söyler.
func adminAllowed(role, path string) bool {
	if !auth.StaffRoles[role] {
		return false
	}
	return auth.User{Role: role}.Can(adminMinRole(path))
}

// forbidden yetkisiz panel isteğini reddeder ve denetim için günlüğe yazar.
func (s *Server) forbidden(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	s.logger.Warn("yetkisiz panel erişimi",
		"path", r.URL.Path,
		"method", r.Method,
		"user_id", u.ID,
		"role", u.Role,
		"remote", r.RemoteAddr,
	)
	w.Header().Set("Cache-Control", "no-store")
	if IsHTMX(r) {
		// htmx yanıtı yer değiştirmesin; kullanıcı mevcut sayfada kalsın.
		w.Header().Set("HX-Reswap", "none")
	}
	http.Error(w, "Bu işlem için yetkiniz yok.", http.StatusForbidden)
}

// canEditContent içerik düzenleme yetkisini belirler.
// Yönetici ve Editör tüm içerikleri, Muhabir/Yazar yalnızca kendi içeriklerini
// düzenleyebilir. Üye ve bilinmeyen roller düzenleyemez.
func canEditContent(u auth.User, c repo.Content) bool {
	if u.ID == 0 {
		return false
	}
	if u.Can("Editör") {
		return true
	}
	if !u.Can("Yazar") {
		return false
	}
	return c.AuthorID == u.ID
}

// editableContent içeriği yükler ve oturum sahibinin düzenleme yetkisini
// kontrol eder. Yetki yoksa istek sahibine uygun yanıtı yazar.
func (s *Server) editableContent(w http.ResponseWriter, r *http.Request, id int64) (repo.Content, bool) {
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		if IsHTMX(r) {
			s.renderAIEditorError(w, r, "İçerik bulunamadı.")
		} else {
			s.flashRedirect(w, r, "/admin/icerikler", "error", "İçerik bulunamadı.")
		}
		return repo.Content{}, false
	}
	if !canEditContent(userFromContext(r.Context()), c) {
		if IsHTMX(r) {
			s.renderAIEditorError(w, r, "Bu içeriği düzenleme yetkiniz yok.")
		} else {
			s.flashRedirect(w, r, "/admin/icerikler", "error", "Bu içeriği düzenleme yetkiniz yok.")
		}
		return repo.Content{}, false
	}
	return c, true
}
