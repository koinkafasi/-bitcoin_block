package repo

import (
	"encoding/json"
	"testing"
)

func blokYap(metinler ...string) []byte {
	var b []map[string]any
	for _, m := range metinler {
		b = append(b, map[string]any{"type": "p", "text": m})
	}
	d, _ := json.Marshal(b)
	return d
}

func TestIskeletGovdeKaynaklarBasligi(t *testing.T) {
	d := blokYap("Küme özeti burada.", "Kaynaklar:", "• 10Haber: Bir başlık")
	if !IskeletGovde(d) {
		t.Error("'Kaynaklar:' iskeleti yakalanmadı")
	}
}

func TestIskeletGovdeMaddeListesi(t *testing.T) {
	d := blokYap("Özet.", "• BBC Türkçe: Bakırköy operasyonu", "• DHA: Merkezefendi operasyonu")
	if !IskeletGovde(d) {
		t.Error("madde listesi iskeleti yakalanmadı")
	}
}

func TestIskeletGovdeGercekHaberGecer(t *testing.T) {
	d := blokYap(
		"Bakırköy Belediyesi'ne yönelik operasyonda 68 kişi hakkında gözaltı kararı verildi.",
		"Soruşturma kapsamında belediye başkan yardımcıları da gözaltına alındı.",
		"İçişleri Bakanlığı konuya ilişkin yazılı açıklama yaptı.",
	)
	if IskeletGovde(d) {
		t.Error("gerçek haber iskelet sayıldı (yanlış pozitif)")
	}
}

func TestIskeletGovdeTekMaddeGecer(t *testing.T) {
	d := blokYap("Özet metni.", "• Tek bir madde: detay burada.")
	if IskeletGovde(d) {
		t.Error("tek madde iskelet sayıldı (yanlış pozitif)")
	}
}

func TestIskeletGovdeBosVeNil(t *testing.T) {
	if IskeletGovde(nil) {
		t.Error("nil gövde iskelet sayıldı")
	}
	if IskeletGovde([]byte("[]")) {
		t.Error("boş dizi iskelet sayıldı")
	}
}
