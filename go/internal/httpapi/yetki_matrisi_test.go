package httpapi

import (
	"strings"
	"testing"
	"time"

	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

// --- hasPathPrefix ---

func TestHasPathPrefixSinirDuyarli(t *testing.T) {
	prefixes := []string{"/admin/ayarlar", "/admin/siteler"}
	kasalar := map[string]bool{
		"/admin/ayarlar":     true,
		"/admin/ayarlar/":    true,
		"/admin/ayarlar/gsc": true,
		"/admin/ayarlar-gsc": false, // önek ama ayrı rota
		"/admin/ayarlarx":    false,
		"/admin/siteler/2":   true,
		"/admin":             false,
		"/admin/ayarlar?x=1": false, // sorgu ayrı ele alınır
	}
	for path, want := range kasalar {
		if got := hasPathPrefix(path, prefixes); got != want {
			t.Errorf("hasPathPrefix(%q) = %v; %v bekleniyordu", path, got, want)
		}
	}
	if hasPathPrefix("/admin", nil) {
		t.Error("boş önek listesinde hiçbir yol eşleşmemeli")
	}
}

// --- adminMinRole: bilinen rotalar ---

func TestAdminMinRoleSahipRotalari(t *testing.T) {
	rotalar := []string{
		"/admin/ayarlar",
		"/admin/ayarlar/gsc",
		"/admin/siteler",
		"/admin/members",
		"/admin/aboneler",
		"/admin/paylasim",
		"/admin/servisler",
		"/admin/site-sec",
		// Sır/yapılandırma sınıfı:
		"/admin/ses-ayarlari",   // sağlayıcı API anahtarı + base_url
		"/admin/trend/gsc/ayar", // Search Console mülkü
		"/admin/trend/ga4/ayar", // GA4 mülkü
		"/admin/trend/gsc/senkron",
		"/admin/trend/kaynak-ekle", // sunucuya çektirilen URL (SSRF)
	}
	for _, p := range rotalar {
		if got := adminMinRole(p); got != "Yönetici" {
			t.Errorf("adminMinRole(%q) = %q; Yönetici bekleniyordu", p, got)
		}
	}
}

func TestAdminMinRoleEditorRotalari(t *testing.T) {
	rotalar := []string{
		"/admin/tasarim",
		"/admin/marka",
		"/admin/menu",
		"/admin/bilesenler",
		"/admin/bulten",
		"/admin/raporlar",
		"/admin/olcu",
		"/admin/seo",
		// Site genelini yeniden yazan işlem:
		"/admin/iletisim",          // alan adı değişimi tüm sayfaları günceller
		"/admin/iletisim/degistir", // toplu metin değişimi
		"/admin/kategori-panelleri",
		"/admin/manset-editor",
	}
	for _, p := range rotalar {
		if got := adminMinRole(p); got != "Editör" {
			t.Errorf("adminMinRole(%q) = %q; Editör bekleniyordu", p, got)
		}
	}
}

// Bilinmeyen rota en düşük personel rolünü ister: yeni eklenen bir rota
// yanlışlıkla geniş yetki vermemeli (varsayılan güvenli taraf).
func TestAdminMinRoleBilinmeyenRotaYazar(t *testing.T) {
	for _, p := range []string{"/admin/yeni-ozellik", "/admin/x/y/z", "/admin"} {
		if got := adminMinRole(p); got != "Yazar" {
			t.Errorf("adminMinRole(%q) = %q; Yazar bekleniyordu", p, got)
		}
	}
}

// --- adminAllowed: rol × rota matrisi ---

func TestAdminAllowedMatrisi(t *testing.T) {
	kasalar := []struct {
		rol  string
		path string
		want bool
	}{
		// Yönetici her yere.
		{"Yönetici", "/admin/ayarlar", true},
		{"Yönetici", "/admin/icerikler", true},
		{"Yönetici", "/admin/bilinmeyen", true},
		// Editör: sahip rotalarına hayır, editör rotalarına evet.
		{"Editör", "/admin/ayarlar", false},
		{"Editör", "/admin/siteler", false},
		{"Editör", "/admin/icerikler", true},
		{"Editör", "/admin/medya", true},
		// Yazar: içerik üretimi serbest, yapılandırma kapalı.
		{"Yazar", "/admin/ayarlar", false},
		{"Yazar", "/admin/icerikler", true},
		{"Yazar", "/admin/medya", true},
		{"Yazar", "/admin/tasarim", false},
		{"Yazar", "/admin/iletisim", false},
		{"Yazar", "/admin/ses-ayarlari", false},
		{"Yazar", "/admin/trend/kaynak-ekle", false},
		{"Yazar", "/admin/bilinmeyen", true},
		// Muhabir: Yazar seviyesi (auth.User.Can sıralaması).
		{"Muhabir", "/admin/bilinmeyen", true},
		{"Muhabir", "/admin/icerikler", true},
		{"Muhabir", "/admin/ses-ayarlari", false},
		// Üye ve boş rol: panelin tamamına hayır.
		{"Üye", "/admin/bilinmeyen", false},
		{"Üye", "/admin/icerikler", false},
		{"", "/admin/bilinmeyen", false},
		{"Sahte", "/admin/ayarlar", false},
	}
	for _, k := range kasalar {
		if got := adminAllowed(k.rol, k.path); got != k.want {
			t.Errorf("adminAllowed(%q, %q) = %v; %v bekleniyordu", k.rol, k.path, got, k.want)
		}
	}
}

// --- canEditContent: sahiplik ---

func TestCanEditContentMatrisi(t *testing.T) {
	icerik := repo.Content{AuthorID: 7, ID: 1}
	kasalar := []struct {
		ad   string
		user auth.User
		want bool
	}{
		{"oturumsuz", auth.User{}, false},
		{"yönetici (başkasının)", auth.User{ID: 1, Role: "Yönetici"}, true},
		{"editör (başkasının)", auth.User{ID: 2, Role: "Editör"}, true},
		{"yazar (kendi)", auth.User{ID: 7, Role: "Yazar"}, true},
		{"yazar (başkasının)", auth.User{ID: 8, Role: "Yazar"}, false},
		{"muhabir (kendi)", auth.User{ID: 7, Role: "Muhabir"}, true},
		{"muhabir (başkasının)", auth.User{ID: 9, Role: "Muhabir"}, false},
		{"üye", auth.User{ID: 7, Role: "Üye"}, false},
	}
	for _, k := range kasalar {
		if got := canEditContent(k.user, icerik); got != k.want {
			t.Errorf("%s: canEditContent = %v; %v bekleniyordu", k.ad, got, k.want)
		}
	}
}

// --- isHTTPS ---

func TestIsHTTPS(t *testing.T) {
	kasalar := map[string]bool{
		"https://theblock.com.tr": true,
		"HTTPS://theblock.com.tr": true,
		"http://127.0.0.1:8070":   false,
		"http://localhost":        false,
		"":                        false,
		"ftp://theblock.com.tr":   false,
	}
	for in, want := range kasalar {
		if got := isHTTPS(in); got != want {
			t.Errorf("isHTTPS(%q) = %v; %v bekleniyordu", in, got, want)
		}
	}
}

// --- parseID ---

func TestParseID(t *testing.T) {
	kasalar := map[string]int64{
		"12":                    12,
		" 12 ":                  12,
		"0":                     0,
		"-3":                    -3, // ParseInt negatifi kabul eder; sorgular boş döner
		"abc":                   0,
		"":                      0,
		"12abc":                 0,
		"999999999999999999999": 0, // taşma → 0
	}
	for in, want := range kasalar {
		if got := parseID(in); got != want {
			t.Errorf("parseID(%q) = %d; %d bekleniyordu", in, got, want)
		}
	}
}

// --- sparkline ---

func TestSparkline(t *testing.T) {
	// Boş seri: çizgi üretilmemeli (boş SVG yolu).
	if got := sparkline(nil); got != "" {
		t.Errorf("sparkline(nil) = %q; boş bekleniyordu", got)
	}
	// Tek değer: bölme sıfıra düşmemeli.
	tek := sparkline([]int{4, 4, 4})
	if strings.Contains(tek, "NaN") || strings.Contains(tek, "Inf") {
		t.Errorf("tekil seri sayısal bozulma üretti: %q", tek)
	}
	if tek == "" {
		t.Error("tekil seri boş olmamalı")
	}
	// Artan seri: son nokta ilkinden daha yüksek olmalı (y aşağı doğru artar).
	if got := sparkline([]int{0, 10}); got == "" {
		t.Error("seri boş olmamalı")
	}
}

// --- pickTone ---

func TestPickTone(t *testing.T) {
	// Eşikler: 0 ve altı → iyi, yüksek değerler → kötü. Değer kümesi sabit.
	gorulen := map[string]bool{}
	for _, n := range []int{-5, 0, 1, 5, 10, 20, 50, 100} {
		ton := pickTone(n)
		if ton == "" {
			t.Errorf("pickTone(%d) boş", n)
		}
		gorulen[ton] = true
	}
	if len(gorulen) < 2 {
		t.Errorf("pickTone tek ton döndürüyor: %v", gorulen)
	}
}

// --- lenAll ---

func TestLenAll(t *testing.T) {
	if got := lenAll(nil); got != 0 {
		t.Errorf("nil = %d", got)
	}
	if got := lenAll(map[string]int{"a": 1, "b": 2}); got != 3 {
		t.Errorf("toplam = %d; 3 bekleniyordu", got)
	}
	if got := lenAll(map[string]int{"a": -5, "b": 2}); got != -3 {
		t.Errorf("negatif toplam = %d; -3 bekleniyordu", got)
	}
}

// --- ptrTime ---

func TestPtrTime(t *testing.T) {
	now := time.Now()
	p := ptrTime(now)
	if p == nil || !p.Equal(now) {
		t.Fatalf("ptrTime = %v", p)
	}
	now = now.Add(time.Hour) // özgün değişkeni değiştirmek kopyayı etkilememeli
	if !p.Equal(now.Add(-time.Hour)) {
		t.Error("ptrTime değerin kopyasını almalı")
	}
}
