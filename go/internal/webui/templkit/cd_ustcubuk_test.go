package templkit

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ustCubukCiz üst çubuğu (arama paneli dahil) çizer.
func ustCubukCiz(t *testing.T, s CDShell) string {
	t.Helper()
	var sb strings.Builder
	require.NoError(t, cdTopbar(s).Render(context.Background(), &sb))
	return sb.String()
}

// TestCDUstCubukOturumYokken: anonim ziyaretçide iki bağlantı da yaşar.
func TestCDUstCubukOturumYokken(t *testing.T) {
	h := ustCubukCiz(t, CDShell{
		SiteName:   "bitcosar",
		LoginHref:  "/giris",
		SignupHref: "/kayit",
		Diller:     []CDDil{{Kod: "tr", Ad: "Türkçe", Aktif: true}},
	})
	assert.Contains(t, h, `href="/giris">Oturum Aç</a>`, "Oturum Aç bağlantısı yok")
	assert.Contains(t, h, `href="/kayit">Kaydol</a>`, "Kaydol bağlantısı yok")
	assert.NotContains(t, h, "data-hesap", "oturum yokken hesap menüsü basılmamalı")
	assert.NotContains(t, h, "cd-hesap-ac__avatar", "oturum yokken avatar basılmamalı")
	// Dil ile oturum arasındaki ayraç anonimde de durur.
	assert.Contains(t, h, "cd-topbar__ayirici")
}

// TestCDUstCubukOturumVarken: giriş yapılmışken iki bağlantı KALKAR,
// yerine kullanıcı simgesi + hesap menüsü gelir.
func TestCDUstCubukOturumVarken(t *testing.T) {
	h := ustCubukCiz(t, CDShell{
		SiteName:    "bitcosar",
		LoginHref:   "/giris", // cdShellOturum bunları siler; dal LoggedIn'e bakar
		SignupHref:  "/kayit",
		Diller:      []CDDil{{Kod: "tr", Ad: "Türkçe", Aktif: true}},
		LoggedIn:    true,
		UserName:    "Ayşe Yılmaz",
		UserInitial: "A",
		UserHref:    "/hesabim",
		UserUnread:  3,
	})

	assert.NotContains(t, h, "Oturum Aç", "giriş yapılmışken Oturum Aç basılmamalı")
	assert.NotContains(t, h, "Kaydol", "giriş yapılmışken Kaydol basılmamalı")

	// Kullanıcı simgesi: daire + baş harf.
	assert.Contains(t, h, `class="cd-hesap-ac__avatar" aria-hidden="true">A</span>`,
		"avatar baş harfi yok")
	assert.Contains(t, h, `data-hesap`, "hesap menüsü (details) yok")
	assert.Contains(t, h, "Ayşe Yılmaz", "menüde kullanıcı adı görünmeli")

	// Menü hedefleri gerçek hesap rotaları.
	for _, yol := range []string{
		`href="/hesabim"`,
		`href="/hesabim/kaydedilenler"`,
		`href="/hesabim/bildirimler"`,
	} {
		assert.Contains(t, h, yol, "menü hedefi eksik: %s", yol)
	}

	// Çıkış GET DEĞİL (forced-logout CSRF — cikis_test.go).
	assert.Contains(t, h, `method="post" action="/cikis"`, "çıkış POST olmalı")
	assert.Contains(t, h, ">Çıkış</button>")

	// Okunmamış bildirim rozeti.
	assert.Contains(t, h, `class="cd-hesap-ac__rozet" aria-hidden="true">3</span>`,
		"bildirim rozeti yok")

	// Ayraç dil ile hesap arasında kalmalı (LoginHref silinince kayboluyordu).
	assert.Contains(t, h, "cd-topbar__ayirici")
}

// TestCDMobilNavOturum: mobil hamburger paneli üst çubukla AYNI kararı verir.
func TestCDMobilNavOturum(t *testing.T) {
	ciz := func(s CDShell) string {
		var sb strings.Builder
		require.NoError(t, cdMobilNav(s).Render(context.Background(), &sb))
		return sb.String()
	}

	anonim := ciz(CDShell{SiteName: "bitcosar", LoginHref: "/giris", SignupHref: "/kayit"})
	// Metinler CoinDesk ölçümüyle hizalandı (9 Kasım 2026): "Oturum Aç" ve
	// "Kaydol" — mobil panel ile masaüstü üst çubuk AYNI etiketleri kullanır.
	assert.Contains(t, anonim, `href="/giris">Oturum Aç</a>`)
	assert.Contains(t, anonim, `href="/kayit">Kaydol</a>`)
	assert.NotContains(t, anonim, "data-hesap")

	girisli := ciz(CDShell{
		SiteName: "bitcosar", LoginHref: "/giris", SignupHref: "/kayit",
		LoggedIn: true, UserName: "Ayşe", UserInitial: "A", UserHref: "/hesabim",
	})
	// DİKKAT: şablon yorumu ("…Giriş + Ücretsiz üye ol…") çıktıya yazılıyor —
	// kapı BAĞLANTI etiketinde, çıplak metinde değil.
	assert.NotContains(t, girisli, `href="/giris"`, "giriş yapılmışken Giriş basılmamalı")
	assert.NotContains(t, girisli, `href="/kayit"`, "giriş yapılmışken kayıt düğmesi kalkmalı")
	assert.Contains(t, girisli, `href="/hesabim">Hesabım</a>`)
	assert.Contains(t, girisli, `method="post" action="/cikis"`)
}

// TestCDBildirimSayi: rozet metni 9 üstü "9+" (kutu 3 haneli sayıya
// şişmesin; eski header ile aynı sınır).
func TestCDBildirimSayi(t *testing.T) {
	assert.Equal(t, "0", cdBildirimSayi(0))
	assert.Equal(t, "9", cdBildirimSayi(9))
	assert.Equal(t, "9+", cdBildirimSayi(10))
	assert.Equal(t, "9+", cdBildirimSayi(1234))
}
