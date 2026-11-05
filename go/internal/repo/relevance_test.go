package repo

import (
	"testing"
	"time"
)

func pt[T any](v T) *T { return &v }

func TestJaccard(t *testing.T) {
	a := map[string]bool{"fenerbahce": true, "transfer": true}
	b := map[string]bool{"fenerbahce": true, "lig": true}
	// kesişim 1, birleşim 3
	if got := jaccard(a, b); got < 0.33 || got > 0.34 {
		t.Fatalf("jaccard = %v, beklenen ~0.333", got)
	}
	if jaccard(nil, a) != 0 {
		t.Fatal("boş küme 0 dönmeli")
	}
}

func TestContentWordsFiltreleri(t *testing.T) {
	// "Fenerbahçe" -> "fenerbahçe" (ç korunur)
	// "LİGİ"      -> "ligi" (İ doğru katlanır: i + birleşen nokta atılır)
	w := contentWords("Fenerbahçe LİGİ'nde yeni bir transfer için 2025")
	if !w["fenerbahçe"] || !w["transfer"] || !w["ligi"] {
		t.Fatalf("anlamli kelimeler eksik: %v", w)
	}
	// "için" ve "bir" gibi stop-word'ler, 3 harften kısa "lig" elenmeli
	if w["için"] || w["bir"] || w["lig"] {
		t.Fatalf("stop-word/kisa kelime sızdı: %v", w)
	}
}

func TestSharedTags(t *testing.T) {
	a := map[int64]bool{1: true, 2: true}
	b := map[int64]bool{2: true, 3: true}
	if got := sharedTags(a, b); got != 1 {
		t.Fatalf("sharedTags = %d, beklenen 1", got)
	}
	if sharedTags(nil, b) != 0 {
		t.Fatal("nil küme 0 dönmeli")
	}
}

func TestRelevanceScoreBilesenler(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	newish := now.Add(-2 * time.Hour)
	old := now.Add(-20 * 24 * time.Hour)

	seed := Content{
		ID: 1, Title: "Fenerbahçe transferinde yeni isim", CategoryID: pt(int64(4)),
		AuthorID: 7, PublishedAt: &newish,
	}

	t.Run("ayni kategori ve baslik benzerligi yuksek skor verir", func(t *testing.T) {
		cand := Content{
			ID: 2, Title: "Fenerbahçe için transfer görüşmeleri", CategoryID: pt(int64(4)),
			AuthorID: 9, PublishedAt: &newish,
		}
		same, _ := relevanceScore(seed, cand, nil, nil, 5, now)
		far := Content{
			ID: 3, Title: "Yapay zeka çalışmalarında yeni dönem", CategoryID: pt(int64(9)),
			AuthorID: 11, PublishedAt: &newish,
		}
		diff, _ := relevanceScore(seed, far, nil, nil, 0, now)
		if same <= diff {
			t.Fatalf("alakalı aday daha yüksek puan almalı: same=%.3f diff=%.3f", same, diff)
		}
		if same <= wCategory {
			t.Fatalf("kategori + başlık bileşenleri toplanmalı: %.3f", same)
		}
	})

	t.Run("populerlik arttikca skor artar", func(t *testing.T) {
		lo, _ := relevanceScore(seed, Content{ID: 4, Title: "x", CategoryID: pt(int64(4)), PublishedAt: &newish}, nil, nil, 0, now)
		hi, _ := relevanceScore(seed, Content{ID: 5, Title: "x", CategoryID: pt(int64(4)), PublishedAt: &newish}, nil, nil, 1000, now)
		if hi <= lo {
			t.Fatalf("popülerlik puanı artırmalı: lo=%.3f hi=%.3f", lo, hi)
		}
	})

	t.Run("eski icerik tazelik bileseni dusuk olur", func(t *testing.T) {
		fresh, partsF := relevanceScore(seed, Content{ID: 6, Title: "x", PublishedAt: &newish}, nil, nil, 0, now)
		stale, partsS := relevanceScore(seed, Content{ID: 7, Title: "x", PublishedAt: &old}, nil, nil, 0, now)
		if partsF["tazelik"] <= partsS["tazelik"] {
			t.Fatalf("tazelik bileşeni azalmalı: fresh=%v stale=%v", partsF, partsS)
		}
		if fresh <= stale {
			t.Fatalf("yeni içerik daha yüksek puan almalı: fresh=%.3f stale=%.3f", fresh, stale)
		}
		if _, ok := partsS["sicak"]; ok {
			t.Fatal("20 günlük içerik 'sıcak' bileşeni almamalı")
		}
	})

	t.Run("etiket ortusmesi ek puan verir", func(t *testing.T) {
		seedTags := map[int64]bool{1: true, 2: true, 3: true}
		candTags := map[int64]bool{2: true}
		s, parts := relevanceScore(seed, Content{ID: 8, Title: "x", PublishedAt: &newish},
			seedTags, candTags, 0, now)
		if parts["etiket"] != wTag {
			t.Fatalf("etiket bileşeni = %v, beklenen %v", parts["etiket"], wTag)
		}
		if s <= wTag {
			t.Fatalf("toplam etiket puanını içermeli: %.3f", s)
		}
	})

	t.Run("yayin tarihi gelecekte olsa da negatif yazmaz", func(t *testing.T) {
		future := now.Add(48 * time.Hour)
		_, parts := relevanceScore(seed, Content{ID: 9, Title: "x", PublishedAt: &future}, nil, nil, 0, now)
		if parts["tazelik"] <= 0 {
			t.Fatalf("gelecek tarih negatif puan üretmemeli: %v", parts)
		}
	})
}
