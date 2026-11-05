package httpapi

import (
	"testing"
	"time"

	"theblock/site/internal/repo"
)

func yorum(id, parent int64) repo.Comment {
	return repo.Comment{ID: id, ParentID: parent, CreatedAt: time.Unix(id, 0)}
}

// TestBuildCommentTreeKokYanit: kökler ve yanıtlar doğru gruplanır (Faz 36).
func TestBuildCommentTreeKokYanit(t *testing.T) {
	in := []repo.Comment{
		yorum(1, 0), // kök
		yorum(2, 1), // 1'in yanıtı
		yorum(3, 0), // kök
		yorum(4, 2), // 2'nin yanıtı (iç içe → 1'in zincirinde)
		yorum(5, 3), // 3'ün yanıtı
	}
	tree := buildCommentTree(in)
	if len(tree) != 2 {
		t.Fatalf("2 kök beklenir, %d bulundu", len(tree))
	}
	if tree[0].C.ID != 1 || tree[1].C.ID != 3 {
		t.Fatalf("kök sırası yanlış: %d %d", tree[0].C.ID, tree[1].C.ID)
	}
	if len(tree[0].Replies) != 2 { // 2 (doğrudan) + 4 (2'nin yanıtı, kök altında toplanır)
		t.Fatalf("kök 1'in 2 yanıtı olmalı: %+v", tree[0].Replies)
	}
	if len(tree[1].Replies) != 1 {
		t.Fatalf("kök 3'ün 1 yanıtı olmalı")
	}
	if tree[0].Replies[0].ID != 2 || tree[0].Replies[1].ID != 4 {
		t.Fatalf("yanıt sırası: %d %d", tree[0].Replies[0].ID, tree[0].Replies[1].ID)
	}
}

// TestBuildCommentTreeEbeveynsizYanit: kökü silinmiş yanıt kök gibi gösterilmeli.
func TestBuildCommentTreeEbeveynsizYanit(t *testing.T) {
	in := []repo.Comment{
		yorum(10, 999), // ebeveyn yok (silinmiş/onaysız)
		yorum(11, 0),
	}
	tree := buildCommentTree(in)
	if len(tree) != 2 {
		t.Fatalf("2 kök beklenir (11 + ebeveynsiz 10), %d bulundu", len(tree))
	}
	// kök olarak sıralı; 10'un ebeveyni olmadığı için kök görünmeli
	found := false
	for _, n := range tree {
		if n.C.ID == 10 {
			found = true
		}
	}
	if !found {
		t.Fatalf("ebeveynsiz yanıt ağaçta kök olarak bulunmalı: %+v", tree)
	}
}

// TestBuildCommentTreeBos: boş liste → boş ağaç.
func TestBuildCommentTreeBos(t *testing.T) {
	if n := len(buildCommentTree(nil)); n != 0 {
		t.Fatalf("boş girince %d düğüm olmamalı", n)
	}
}
