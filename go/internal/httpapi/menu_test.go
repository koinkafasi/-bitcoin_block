package httpapi

import (
	"encoding/json"
	"testing"

	"theblock/site/internal/repo"
)

func TestMenuNamedAttachesAltKategoriler(t *testing.T) {
	pid := int64(1)
	ekonomiSlug := "ekonomi"
	cats := []repo.Category{
		{ID: 1, Name: "Güncel", Slug: "guncel", InHeader: true, Status: "yayinda", SortOrder: 1},
		{ID: 2, Name: "Ekonomi", Slug: "ekonomi", InHeader: true, Status: "yayinda", SortOrder: 2},
		{ID: 3, Name: "Borsa", Slug: "borsa", ParentID: &pid, ParentSlug: &ekonomiSlug, InMenu: true, Status: "yayinda", SortOrder: 1},
		{ID: 4, Name: "Döviz", Slug: "doviz", ParentID: &pid, ParentSlug: &ekonomiSlug, InMenu: true, Status: "yayinda", SortOrder: 2},
		{ID: 5, Name: "Gizli Alt", Slug: "gizli-alt", ParentID: &pid, ParentSlug: &ekonomiSlug, InMenu: false, Status: "yayinda", SortOrder: 3},
	}

	s := &Server{}
	rows := s.menuNamed(nil, cats)

	if len(rows) != 2 {
		t.Fatalf("üst menüde 2 öğe bekleniyordu, %d geldi: %+v", len(rows), rows)
	}
	for _, r := range rows {
		if r.Slug == "borsa" || r.Slug == "doviz" {
			t.Fatalf("alt kategori üst menüye sızmamalı: %+v", r)
		}
	}
	var ekonomi *MenuRow
	for i := range rows {
		if rows[i].Slug == "ekonomi" {
			ekonomi = &rows[i]
		}
	}
	if ekonomi == nil {
		t.Fatal("Ekonomi menüde bulunmalı")
	}
	if len(ekonomi.Children) != 2 {
		t.Fatalf("Ekonomi'nin 2 görünür alt kategorisi olmalı, %d: %+v", len(ekonomi.Children), ekonomi.Children)
	}
	if ekonomi.Children[0].Slug != "borsa" || ekonomi.Children[1].Slug != "doviz" {
		t.Fatalf("alt kategori sırası yanlış: %+v", ekonomi.Children)
	}
}

func TestMenuNamedOzelSatirEtiketiKorunur(t *testing.T) {
	cats := []repo.Category{
		{ID: 1, Name: "Güncel", Slug: "guncel", InHeader: true, Status: "yayinda"},
	}
	s := &Server{}
	rows := s.menuNamed([]MenuRow{{Slug: "guncel", Label: "Manşet"}}, cats)
	if len(rows) != 1 || rows[0].Label != "Manşet" || rows[0].Name != "Güncel" {
		t.Fatalf("özel etiket ve kategori adı korunmalı: %+v", rows)
	}
}

func TestBuilderKategoriSayfaAnahtari(t *testing.T) {
	if !validCatSlug("borsa") || !validCatSlug("is-dunyasi") {
		t.Fatal("geçerli slug reddedilmemeli")
	}
	if validCatSlug("") || validCatSlug("Borsa") || validCatSlug("../x") {
		t.Fatal("geçersiz slug kabul edilmemeli")
	}
	if !isKategoriPage("kategori") || !isKategoriPage("kategori:borsa") || isKategoriPage("home") {
		t.Fatal("kategori sayfa anahtarı tanınmadı")
	}
}

func TestKategoriTasarimSablonlari(t *testing.T) {
	for _, key := range []string{"genel", "piyasa", "manset", "dergi", "klasik", "sade-liste", "izgara", "mozaik"} {
		p, ok := categoryPresetByKey(key)
		if !ok {
			t.Fatalf("%s şablonu bulunamadı", key)
		}
		if p.Name == "" {
			t.Fatalf("%s şablonunun adı boş", key)
		}
		if p.Blocks != "" && !json.Valid([]byte(p.Blocks)) {
			t.Fatalf("%s şablonunun blok JSON'u geçersiz: %s", key, p.Blocks)
		}
		if !validCategoryView(p.View) {
			t.Fatalf("%s şablonunun görünümü geçersiz: %s", key, p.View)
		}
	}
	if _, ok := categoryPresetByKey("yok"); ok {
		t.Fatal("bilinmeyen şablon kabul edilmemeli")
	}
	for _, v := range []string{"", "klasik", "liste", "izgara", "mozaik"} {
		if !validCategoryView(v) {
			t.Fatalf("%q görünümü geçerli olmalı", v)
		}
	}
	if validCategoryView("slider") {
		t.Fatal("bilinmeyen görünüm reddedilmeli")
	}
}
