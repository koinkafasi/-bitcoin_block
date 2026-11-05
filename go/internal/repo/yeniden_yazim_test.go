package repo

import "testing"

// Faz 204 — gövde çözümleme iki biçimde de çalışmalı.
//
// ÖLÇÜM: yayındaki 986 içeriğin 982'si blok dizisi, 4'ü TipTap belgesi.
// Yeniden yazım her ikisini de okuyabilmeli; aksi hâlde o 4 içerik
// "gövdesi boş" sayılıp hiç önerilmezdi.

func TestGovdeParagraflariBlokDizisi(t *testing.T) {
	detail := []byte(`[{"type":"p","text":"Birinci paragraf."},{"type":"p","text":"İkinci paragraf."}]`)
	p := GovdeParagraflari(detail)
	if len(p) != 2 {
		t.Fatalf("2 paragraf bekleniyordu: %v", p)
	}
	if p[1] != "İkinci paragraf." {
		t.Errorf("sıra bozuk: %v", p)
	}
}

func TestGovdeParagraflariTipTapBelgesi(t *testing.T) {
	detail := []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"TipTap paragrafı."}]},{"type":"paragraph","content":[{"type":"text","text":"İkinci blok."}]}]}`)
	p := GovdeParagraflari(detail)
	if len(p) != 2 {
		t.Fatalf("2 paragraf bekleniyordu: %v", p)
	}
	if p[0] != "TipTap paragrafı." {
		t.Errorf("metin okunamadı: %v", p)
	}
}

func TestGovdeParagraflariBos(t *testing.T) {
	if p := GovdeParagraflari(nil); len(p) != 0 {
		t.Errorf("boş girdi paragraf üretmemeli: %v", p)
	}
	if p := GovdeParagraflari([]byte(`[]`)); len(p) != 0 {
		t.Errorf("boş dizi paragraf üretmemeli: %v", p)
	}
}

func TestGovdeParagraflariBosBloklariAtlar(t *testing.T) {
	detail := []byte(`[{"type":"p","text":"Dolu."},{"type":"p","text":"  "}]`)
	p := GovdeParagraflari(detail)
	if len(p) != 1 {
		t.Errorf("boş blok atılmalıydı: %v", p)
	}
	if GovdeMetni(detail) != "Dolu." {
		t.Errorf("GovdeMetni beklenen metni vermedi: %q", GovdeMetni(detail))
	}
}
