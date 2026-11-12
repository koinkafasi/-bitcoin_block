package trends

import (
	"fmt"
	"testing"
	"time"
)

func TestHazirEslestirmeCoveredIleAyni(t *testing.T) {
	terimler := []string{
		"Galatasaray",
		"Fenerbahçe Aykut Kocaman",
		"dolar kuru",
		"İstanbul Boğazı gemi",
		"Gemini 4 Carbon",
		"deprem",
		"a", // tokenlanamaz
		"",
		"günde 8 bardak su efsanesi",
	}
	metinler := []string{
		"Galatasaray'ın yıldızından flaş açıklama geldi",
		"Fenerbahçe'de ibre Aykut Kocaman'a döndü! Yönetimden flaş karar",
		"Dolar kuru güne yükselişle başladı",
		"İstanbul Boğazı'nda gemi arızası: Trafik askıya alındı",
		"Google yeni yapay zeka modeli Gemini 4 Carbon'u tanıttı",
		"Harşit Çayı kenarında mahsur kalan yavru kediyi itfaiye kurtardı",
		"",
		"Günde 8 bardak su efsanesi bitti! Çay ve çorba da su yerine geçiyor",
	}

	for _, term := range terimler {
		ht, ok := Hazirla(1, term, time.Now(), 0)
		for _, metin := range metinler {
			wantKapsiyor := Covered(term, metin)
			wantOran := int(MatchRatio(term, metin) * 100)

			if !ok {
				// Ayrıştırılamayan terim hiçbir şeyi kapsamamalı.
				if wantKapsiyor {
					t.Errorf("Covered(%q,%q)=true ama Hazirla başarısız", term, metin)
				}
				continue
			}
			hb := HazirlaBelge(metin)
			gotOran, gotKapsiyor := hb.kapsiyor(ht)
			if gotKapsiyor != wantKapsiyor {
				t.Errorf("kapsama ayrıştı: terim=%q metin=%q — hazır %v, Covered %v",
					term, metin, gotKapsiyor, wantKapsiyor)
			}
			if gotKapsiyor && gotOran != wantOran {
				t.Errorf("oran ayrıştı: terim=%q metin=%q — hazır %d, MatchRatio %d",
					term, metin, gotOran, wantOran)
			}
		}
	}
}

func TestEnIyiEslesmeEnGucluyuSecer(t *testing.T) {
	simdi := time.Now()
	var hazir []HazirTerim
	for i, s := range []string{"Galatasaray", "Galatasaray Icardi gol", "deprem"} {
		h, ok := Hazirla(int64(i+1), s, simdi, 0)
		if !ok {
			t.Fatalf("hazırlanamadı: %q", s)
		}
		hazir = append(hazir, h)
	}
	b := HazirlaBelge("Galatasaray'da Icardi gol attı, taraftar coştu")
	en, oran, ok := EnIyiEslesme(b, hazir)
	if !ok {
		t.Fatal("eşleşme bulunamadı")
	}
	if oran != 100 {
		t.Errorf("oran %d, 100 bekleniyordu", oran)
	}
	if en.ID != 1 && en.ID != 2 {
		t.Errorf("beklenmeyen terim id %d", en.ID)
	}

	// Hiç eşleşmeyen belge.
	if _, _, ok := EnIyiEslesme(HazirlaBelge("Hava durumu yarın yağmurlu"), hazir); ok {
		t.Error("alakasız belge eşleşti")
	}
	// Boş belge.
	if _, _, ok := EnIyiEslesme(HazirlaBelge(""), hazir); ok {
		t.Error("boş belge eşleşti")
	}
}

// Eşit oranda eşleşen iki terimden daha YENİ görüleni seçilmeli.
func TestEnIyiEslesmeEsitlikteGunceliSecer(t *testing.T) {
	simdi := time.Now()
	eski, _ := Hazirla(1, "deprem", simdi.Add(-40*time.Hour), 0)
	yeni, _ := Hazirla(2, "deprem", simdi.Add(-1*time.Hour), 0)
	b := HazirlaBelge("Ege'de deprem meydana geldi")

	for _, sira := range [][]HazirTerim{{eski, yeni}, {yeni, eski}} {
		en, _, ok := EnIyiEslesme(b, sira)
		if !ok {
			t.Fatal("eşleşme yok")
		}
		if en.ID != 2 {
			t.Errorf("eski terim seçildi (id %d); sıra bağımlılığı var", en.ID)
		}
	}
}

// Hazır yol, saf yoldan belirgin ölçüde hızlı olmalı — zaten varlık sebebi bu.
func BenchmarkEslestirmeHazir(b *testing.B) {
	simdi := time.Now()
	var hazir []HazirTerim
	for i := 0; i < 600; i++ {
		h, _ := Hazirla(int64(i), fmt.Sprintf("terim %d gündem", i), simdi, 0)
		hazir = append(hazir, h)
	}
	metin := "Fenerbahçe'de ibre Aykut Kocaman'a döndü! Yönetimden flaş karar"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EnIyiEslesme(HazirlaBelge(metin), hazir)
	}
}
