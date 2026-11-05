package trends

import (
	"testing"
	"time"
)

// Bu testler B1 hatasını kalıcı olarak korur: kapsama analizi tek bir 72
// saatlik pencereye bakıyordu. Site 72 saatte yayın yapmayınca eşleşme hiç
// bulunamıyor, coverage_count tüm terimlerde 0 kalıyor ve skorlamada her
// terime "içerik boşluğu bonusu" veriliyordu — editöre arşivde 7 içerik
// varken "boşluk" gösteriliyordu. Düzeltme: arşiv sayısı (Total) ile taze
// pencere sayısı (Recent) ayrı üretilir.

func docs() []CoverageDoc {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	return []CoverageDoc{
		// Arşiv: 10 gün önce yayınlanmış, terimi kapsıyor.
		{ID: 1, Text: "Roma'da protestolar sürüyor", PublishedAt: now.Add(-10 * 24 * time.Hour), Hits: 120},
		// Arşiv: 40 gün önce.
		{ID: 2, Text: "Roma deplasmanında kritik maç", PublishedAt: now.Add(-40 * 24 * time.Hour), Hits: 30},
		// Taze: 6 saat önce.
		{ID: 3, Text: "Roma kentinde güvenlik önlemleri artırıldı", PublishedAt: now.Add(-6 * time.Hour), Hits: 10},
		// Alakasız.
		{ID: 4, Text: "Kocaeli'de hava durumu", PublishedAt: now.Add(-1 * time.Hour), Hits: 5},
	}
}

func TestCoverageArsivVeTazeyiAyirir(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	got := Coverage("roma", docs(), now, 72*time.Hour)

	if got.Total != 3 {
		t.Fatalf("arşiv dahil kapsama = %d, beklenen 3 (1,2,3 eşleşmeli)", got.Total)
	}
	if got.Recent != 1 {
		t.Fatalf("taze kapsama = %d, beklenen 1 (yalnız 6 saat önceki)", got.Recent)
	}
	if len(got.Matches) != 3 {
		t.Fatalf("eşleşme haritası = %d kayıt, beklenen 3", len(got.Matches))
	}
	if _, ok := got.Matches[4]; ok {
		t.Error("alakasız içerik (id=4) eşleşmemeliydi")
	}
	if got.HitSum != 160 {
		t.Errorf("okunma toplamı = %d, beklenen 160", got.HitSum)
	}
}

// Asıl regresyon: yalnız arşivde içerik varsa Total>0 (panel "kapsandı"
// demeli) ama Recent=0 (skorlama boşluk bonusu verebilir). Eski kod bu iki
// durumu ayırmadığı için panel yanlış bilgi veriyordu.
func TestCoverageArsivVarTazeYokIseBoslukSayilmaz(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	only := []CoverageDoc{
		{ID: 7, Text: "Faiz kararı piyasaları hareketlendirdi", PublishedAt: now.Add(-20 * 24 * time.Hour)},
	}
	got := Coverage("faiz", only, now, 72*time.Hour)
	if got.Total != 1 {
		t.Fatalf("arşiv kapsaması = %d, beklenen 1 — panel 'boşluk' dememeli", got.Total)
	}
	if got.Recent != 0 {
		t.Fatalf("taze kapsama = %d, beklenen 0 — haber döngüsünde işlenmemiş", got.Recent)
	}
}

func TestCoverageTazePencereSifirsaRecentUretilmez(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	got := Coverage("roma", docs(), now, 0)
	if got.Total != 3 {
		t.Fatalf("Total = %d, beklenen 3", got.Total)
	}
	if got.Recent != 0 {
		t.Fatalf("recentWindow=0 iken Recent = %d, beklenen 0", got.Recent)
	}
}

// PublishedAt boş olan kayıt taze sayılmamalı (eski içe aktarımlarda tarih
// boş olabiliyor; boş tarihi "şimdi" varsaymak yanlış pozitif üretir).
func TestCoverageTarihsizKayitTazeSayilmaz(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	only := []CoverageDoc{{ID: 9, Text: "Roma turu rehberi"}}
	got := Coverage("roma", only, now, 72*time.Hour)
	if got.Total != 1 {
		t.Fatalf("Total = %d, beklenen 1", got.Total)
	}
	if got.Recent != 0 {
		t.Fatalf("tarihsiz kayıt için Recent = %d, beklenen 0", got.Recent)
	}
}

func TestCoverageBosGirdi(t *testing.T) {
	got := Coverage("", docs(), time.Now(), 72*time.Hour)
	if got.Total != 0 || len(got.Matches) != 0 {
		t.Fatalf("boş terim için kapsama üretilmemeli: %+v", got)
	}
	got2 := Coverage("roma", nil, time.Now(), 72*time.Hour)
	if got2.Total != 0 {
		t.Fatalf("belge yoksa kapsama 0 olmalı: %+v", got2)
	}
	if got2.Matches == nil {
		t.Error("Matches nil dönmemeli (çağıran doğrudan yazabilmeli)")
	}
}
