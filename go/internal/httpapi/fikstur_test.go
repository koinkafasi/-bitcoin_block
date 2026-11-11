package httpapi

import (
	"testing"

	"theblock/site/internal/repo"

	"github.com/stretchr/testify/assert"
)

// ---- fikstür sayfası: "hafta" parametresi doğrulama ----------------
//
// Sözleşme: yalnızca 1..Rounds aralığında tam sayı kabul edilir. Diğer
// her durumda güncel hafta (meta.Round) kullanılır — bozuk sorgu
// parametresi sayfayı bozmamalı, boş hafta göstermemeli.

func TestParseFiksturRound(t *testing.T) {
	meta := repo.FiksturMeta{Season: "2026-2027", Round: 8, Rounds: 34}

	cases := []struct {
		name string
		in   string
		want int
	}{
		{"gecerli hafta", "5", 5},
		{"ilk hafta", "1", 1},
		{"son hafta", "34", 34},
		{"bos -> guncel hafta", "", meta.Round},
		{"yalnizca bosluk -> guncel", "   ", meta.Round},
		{"kirpili bosluk duzeltilir", "  5  ", 5},
		{"sayi degil -> guncel", "abc", meta.Round},
		{"ondalik -> guncel", "5.5", meta.Round},
		{"sifir -> guncel", "0", meta.Round},
		{"negatif -> guncel", "-3", meta.Round},
		{"sezon disi -> guncel", "35", meta.Round},
		{"cok buyuk -> guncel", "999999", meta.Round},
		{"sql benzeri girdi -> guncel", "1 OR 1=1", meta.Round},
		{"turkce rakam -> guncel", "５", meta.Round},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, parseFiksturRound(c.in, meta))
		})
	}
}

func TestParseFiksturRoundEdgeMeta(t *testing.T) {
	cases := []struct {
		name string
		meta repo.FiksturMeta
		in   string
		want int
	}{
		{"Rounds 0 ise hicbir hafta kabul edilmez", repo.FiksturMeta{Round: 3, Rounds: 0}, "1", 3},
		{"meta bos, parametre de bos", repo.FiksturMeta{}, "", 0},
		{"meta bos, parametre gecerli ama sinir yok", repo.FiksturMeta{}, "5", 0},
		{"guncel hafta 0", repo.FiksturMeta{Round: 0, Rounds: 34}, "", 0},
		{"Rounds negatif", repo.FiksturMeta{Round: 4, Rounds: -1}, "2", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseFiksturRound(c.in, c.meta)
			assert.Equal(t, c.want, got)
			assert.GreaterOrEqual(t, got, 0, "hafta asla negatif olmamalı")
		})
	}
}

func TestParseFiksturRoundAlwaysInRange(t *testing.T) {
	meta := repo.FiksturMeta{Round: 8, Rounds: 34}
	for _, in := range []string{"", " ", "0", "-1", "abc", "1e3", "999", "34", "1", "34.0", "+5"} {
		got := parseFiksturRound(in, meta)
		assert.GreaterOrEqual(t, got, 1, "%q -> 1'den küçük hafta üretmemeli", in)
		assert.LessOrEqual(t, got, 34, "%q -> sezon dışı hafta üretmemeli", in)
	}
}
