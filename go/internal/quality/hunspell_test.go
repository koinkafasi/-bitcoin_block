package quality

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHunspellNilSafe(t *testing.T) {
	var d *ImlaDenetleyici
	got, err := d.Denetle(context.Background(), "metin")
	assert.NoError(t, err)
	assert.Nil(t, got)
	assert.False(t, d.Kullanilabilir(context.Background()))
}

func TestHunspellUnavailableIsSilent(t *testing.T) {
	d := YeniImlaDenetleyici("kesinlikle-olmayan-hunspell-binary", "")
	assert.False(t, d.Kullanilabilir(context.Background()))
	got, err := d.Denetle(context.Background(), "herşey yolunda")
	assert.NoError(t, err, "araç yoksa hata değil, katman kapalı")
	assert.Nil(t, got)
}

func TestHunspellEmptyText(t *testing.T) {
	d := YeniImlaDenetleyici("hunspell", "")
	got, err := d.Denetle(context.Background(), "   ")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestLooksLikeHunspellError(t *testing.T) {
	assert.True(t, looksLikeHunspellError("Can't open affix or dictionary files"))
	assert.True(t, looksLikeHunspellError("Ispell dictionary not found"))
	assert.False(t, looksLikeHunspellError("merhaba"))
	assert.False(t, looksLikeHunspellError("dünya"))
}

func TestNonEmptyLines(t *testing.T) {
	got := nonEmptyLines("a\n\n  b  \n c\n")
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestHecelemeBulguHazirla(t *testing.T) {
	assert.Nil(t, HecelemeBulguHazirla(nil))
	got := HecelemeBulguHazirla([]HunspellBulgu{
		{Word: "yanlş", Count: 2},
		{Word: "klavuz", Count: 1},
	})
	require.Len(t, got, 1)
	assert.Equal(t, "imla_sozluk", got[0].Code)
	assert.Equal(t, 2, got[0].Count)
	assert.Contains(t, got[0].Message, "yanlş")
	assert.Contains(t, got[0].Message, "klavuz")
}

func TestHecelemeBulguHazirlaTruncates(t *testing.T) {
	in := make([]HunspellBulgu, 12)
	for i := range in {
		in[i] = HunspellBulgu{Word: "x"}
	}
	got := HecelemeBulguHazirla(in)
	require.Len(t, got, 1)
	assert.Equal(t, 12, got[0].Count)
	assert.Contains(t, got[0].Message, "…")
}

// Hunspell gerçekten kuruluysa uçtan uca dene; değilse atla.
func TestHunspellRealDictionaryIntegration(t *testing.T) {
	if _, err := exec.LookPath("hunspell"); err != nil {
		t.Skip("hunspell PATH'te yok")
	}
	d := YeniImlaDenetleyici("", "tr_TR")
	if !d.Kullanilabilir(context.Background()) {
		t.Skip("tr_TR sözlüğü kurulu değil")
	}
	got, err := d.Denetle(context.Background(), "Bu cümle doğru yazıldı ve kntrol edildi.")
	require.NoError(t, err)
	assert.NotEmpty(t, got, "yanlış yazılmış sözcük bulunmalı")
	t.Logf("bulgular: %+v", got)
}
