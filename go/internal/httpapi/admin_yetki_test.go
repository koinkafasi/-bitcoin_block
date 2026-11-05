package httpapi

import "testing"

// TestAdminMinRole rota → asgari rol eşlemesini doğrular.
// Regresyon: yeni bir rota eklenirse varsayılan en düşük personel rolü olmalı.
func TestAdminMinRole(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/admin", "Yazar"},
		{"/admin/icerikler", "Yazar"},
		{"/admin/icerikler/yeni", "Yazar"},
		{"/admin/dosyalar", "Yazar"},
		{"/admin/2fa", "Yazar"},
		{"/admin/ayarlar", "Yönetici"},
		{"/admin/ayarlar/ai", "Yönetici"},
		{"/admin/servisler", "Yönetici"},
		{"/admin/siteler/{id}/sil", "Yönetici"},
		{"/admin/site-sec", "Yönetici"},
		{"/admin/members/{id}/{plan}", "Yönetici"},
		{"/admin/aboneler/bulten/ekle", "Yönetici"},
		{"/admin/paylasim", "Yönetici"},
		{"/admin/gazete/ekip/{user_id}/sil", "Yönetici"},
		{"/admin/reklamlar/{id}/sil", "Editör"},
		{"/admin/tasarim", "Editör"},
		{"/admin/bilesenler", "Editör"},
		{"/admin/menu", "Editör"},
		// Önek benzerliği yanlış eşleşmemeli: /admin/ayarlar-degil
		{"/admin/ayarlar-degil", "Yazar"},
	}
	for _, c := range cases {
		if got := adminMinRole(c.path); got != c.want {
			t.Errorf("adminMinRole(%q) = %q, beklenen %q", c.path, got, c.want)
		}
	}
}

// TestAdminAllowed rol matrisini doğrular. En kritik satır: Üye ve boş rol,
// oturumu olsa bile panele giremez.
func TestAdminAllowed(t *testing.T) {
	cases := []struct {
		role string
		path string
		want bool
	}{
		// Okuyucu ve tanımsız roller: hiçbir panel rotası
		{"Üye", "/admin", false},
		{"Üye", "/admin/icerikler", false},
		{"", "/admin", false},
		{"bilinmeyen", "/admin/pano", false},

		// Yazar: yalnız içerik alanı
		{"Yazar", "/admin/icerikler", true},
		{"Yazar", "/admin/ayarlar", false},
		{"Yazar", "/admin/menu", false},

		// Muhabir: içerik alanı
		{"Muhabir", "/admin/icerikler/yeni", true},
		{"Muhabir", "/admin/reklamlar", false},

		// Editör: düzen/yayın yapılandırması, ama ödeme/site ayarı değil
		{"Editör", "/admin/menu", true},
		{"Editör", "/admin/reklamlar/{id}/sil", true},
		{"Editör", "/admin/ayarlar", false},
		{"Editör", "/admin/members", false},

		// Yönetici: her şey
		{"Yönetici", "/admin", true},
		{"Yönetici", "/admin/ayarlar", true},
		{"Yönetici", "/admin/siteler/{id}/sil", true},
	}
	for _, c := range cases {
		if got := adminAllowed(c.role, c.path); got != c.want {
			t.Errorf("adminAllowed(%q, %q) = %v, beklenen %v", c.role, c.path, got, c.want)
		}
	}
}
