package repo

import (
	"testing"

	"theblock/site/internal/quality"
)

func TestDedupeDecisionOrijinalAgregatoruYener(t *testing.T) {
	if got := DedupeDecision(false, true); got != DedupeReplace {
		t.Fatalf("orijinal agregatörü yenmeliydi, alınan: %s", got)
	}
}

func TestDedupeDecisionAgregatorOrijinaliYenmez(t *testing.T) {
	if got := DedupeDecision(true, false); got != DedupeSkip {
		t.Fatalf("agregatör orijinali yenmemeliydi, alınan: %s", got)
	}
}

func TestDedupeDecisionIkiAgregatorAtlar(t *testing.T) {
	if got := DedupeDecision(true, true); got != DedupeSkip {
		t.Fatalf("iki agregatör arasında atlama beklenirdi, alınan: %s", got)
	}
}

func TestDedupeDecisionIkiOrijinalAtlar(t *testing.T) {
	if got := DedupeDecision(false, false); got != DedupeSkip {
		t.Fatalf("iki orijinal arasında atlama beklenirdi, alınan: %s", got)
	}
}

func TestDuplicateOfCandidatesAtEsikDusukDahaKolayYakalar(t *testing.T) {
	cands := []quality.Candidate{{
		Ref:   "9",
		Title: "Bakanlık yeni destek paketini duyurdu",
		Text:  "Bakanlık yeni destek paketini duyurdu çiftçilere dönüm başına ek destek ödemesi yapılacak",
	}}
	title := "Bakanlık destek paketini açıkladı"
	spot := "Çiftçilere dönüm başına ek destek ödemesi yapılacak"

	// Eşik davranışı metnin kendi benzerliğine göre ölçülür: çok düşük eşik
	// her zaman yakalar, benzerliğin biraz üstündeki eşik yakalamaz.
	low, dup := DuplicateOfCandidatesAt(title, spot, cands, 0.01)
	if !dup {
		t.Fatal("0.01 eşiğinde eşleşme beklenirdi")
	}
	if _, dup := DuplicateOfCandidatesAt(title, spot, cands, low.Score+0.01); dup {
		t.Fatalf("benzerliğin (%.2f) üstündeki eşikte eşleşmemeliydi", low.Score)
	}
}
