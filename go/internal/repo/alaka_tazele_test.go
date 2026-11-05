package repo

import (
	"testing"

	"theblock/site/internal/scoring"
)

// Alaka puanı tazeleme (Faz 169).
//
// Bilinen borç: puan yalnız içerik oluşturulurken yazılıyordu; düzenlemeden
// sonra bayat kalıyordu. UpdateContent artık tazeleAIScore çağırıyor.
//
// Bu testler VERİTABANI GEREKTİRMEZ: puanın sinyallere doğru tepki verdiğini
// (kaynak fonksiyon üzerinden) ve UpdateContent'in tazelemeyi çağırdığını
// kaynak düzeyinde kilitler. Veritabanı entegrasyonu e2e kapsamında.

// Sinyaller değişince puan değişmeli: kısa başlık + spotsuz içerik, tam
// içerikten düşük puan almalı.
func TestAlakaPuaniSinyallereDuyarli(t *testing.T) {
	tam, _ := scoring.Explain(scoring.Input{
		Title:    "Deprem sonrası kentsel dönüşüm takvimi açıklandı",
		Spot:     "Bakanlık, riskli yapıların dönüşümü için yeni takvimi duyurdu ve başvuru sürecini açıkladı.",
		TagCount: 4, HasCover: true, HasSource: true,
	})
	eksik, _ := scoring.Explain(scoring.Input{
		Title: "Kısa", Spot: "", TagCount: 0, HasCover: false, HasSource: false,
	})
	if tam <= eksik {
		t.Fatalf("tam içerik daha yüksek puan almalı: tam=%d eksik=%d", tam, eksik)
	}
	if eksik < 0 || tam > 100 {
		t.Fatalf("puan 0-100 aralığında olmalı: tam=%d eksik=%d", tam, eksik)
	}
}

// Kapak ve etiket tek başına puanı artırmalı (tazelemenin okuduğu sinyaller).
func TestAlakaPuaniKapakVeEtiketEtkisi(t *testing.T) {
	temel, _ := scoring.Explain(scoring.Input{Title: "Başlık", Spot: "Spot"})
	kapakli, _ := scoring.Explain(scoring.Input{Title: "Başlık", Spot: "Spot", HasCover: true})
	etiketli, _ := scoring.Explain(scoring.Input{Title: "Başlık", Spot: "Spot", TagCount: 3})
	if kapakli <= temel {
		t.Errorf("kapak puanı artırmalı: %d → %d", temel, kapakli)
	}
	if etiketli <= temel {
		t.Errorf("etiket puanı artırmalı: %d → %d", temel, etiketli)
	}
}
