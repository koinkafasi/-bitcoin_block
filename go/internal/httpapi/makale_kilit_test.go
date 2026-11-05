package httpapi

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func TestMakaleKilitKutusuCTA(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.MetaTitle = "Haber"
	d.Article = &articleData{
		Locked:       true,
		FreeHTML:     template.HTML("<p>Önizleme</p>"),
		RequiredTier: "Gümüş",
		Content:      repo.Content{ID: 5, Title: "Örnek", TypeKey: "haber"},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "article", d))
	out := buf.String()
	assert.Contains(t, out, "Abonelere Özel")
	assert.Contains(t, out, "Gümüş</b> ve üzeri")
	assert.Contains(t, out, "/abonelik")
	assert.Contains(t, out, "/giris?redirect=abonelik")
	assert.Contains(t, out, "/kayit?redirect=abonelik")
	assert.NotContains(t, out, "/admin/giris")
	assert.NotContains(t, out, "art-locked\" role=\"note\"></div>")
}

func TestMakaleKilitliGirisliUye(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.Article = &articleData{
		Locked:  true,
		Content: repo.Content{ID: 6, Title: "Örnek 2", TypeKey: "haber"},
	}
	d.MemberLoggedIn = true
	d.MemberName = "Ayşe"
	d.MemberInitial = "A"
	d.MemberURL = "/hesabim"
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "article", d))
	out := buf.String()
	assert.Contains(t, out, "/hesabim/abonelik")
	assert.NotContains(t, out, "/giris?redirect=abonelik")
}
