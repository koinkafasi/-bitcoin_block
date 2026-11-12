package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/auth"
	"theblock/site/internal/webui/templkit"
)

func TestCDShellOturum(t *testing.T) {
	sh := templkit.CDShell{
		SiteName:   "bitcosar",
		LoginHref:  "/giris",
		SignupHref: "/kayit",
	}
	cdShellOturum(&sh, auth.User{ID: 7, Name: "  ayşe yılmaz  "})

	assert.True(t, sh.LoggedIn, "LoggedIn işaretlenmedi")
	assert.Equal(t, "ayşe yılmaz", sh.UserName)
	assert.Equal(t, "A", sh.UserInitial, "baş harf büyük olmalı")
	assert.Equal(t, "/hesabim", sh.UserHref, "simge herkese hesabim'e gitmeli")
	assert.Empty(t, sh.LoginHref, "ölü 'Oturum Aç' bağlantısı kalmamalı")
	assert.Empty(t, sh.SignupHref, "ölü 'Kaydol' bağlantısı kalmamalı")
}

// TestCDShellOturumBosAd: adı olmayan oturumda avatar "?" ile dolmalı ve
// sayfa çökmemeli.
func TestCDShellOturumBosAd(t *testing.T) {
	sh := templkit.CDShell{LoginHref: "/giris", SignupHref: "/kayit"}
	cdShellOturum(&sh, auth.User{ID: 9})
	assert.True(t, sh.LoggedIn)
	assert.Equal(t, "?", sh.UserInitial)
	assert.Empty(t, sh.LoginHref)
}
