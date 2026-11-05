package cryptoicon

import (
	"bytes"
	"strings"
	"testing"
)

func TestURLLocalVeSurumlu(t *testing.T) {
	want := URLBase + "btc.svg?v=" + SourceVersion
	for _, sym := range []string{"BTC", "btc", " btc "} {
		if got := URL(sym); got != want {
			t.Errorf("URL(%q) = %q, beklenen %q", sym, got, want)
		}
	}
	if !strings.HasPrefix(want, "/media/crypto/") {
		t.Fatalf("ikon yolu yerel /media/crypto/ altında olmalı: %q", want)
	}
}

func TestURLAliasVeBilinmeyen(t *testing.T) {
	cases := map[string]string{
		"IOTA": URLBase + "miota.svg?v=" + SourceVersion,
		"iota": URLBase + "miota.svg?v=" + SourceVersion,
		"POL":  URLBase + "matic.svg?v=" + SourceVersion,
	}
	for sym, want := range cases {
		if got := URL(sym); got != want {
			t.Errorf("URL(%q) = %q, beklenen %q", sym, got, want)
		}
	}
	for _, sym := range []string{"", "  ", "YOKBOYLE", "OKB", "LUNA", "LUNC"} {
		if got := URL(sym); got != "" {
			t.Errorf("URL(%q) ikon olmadığı için boş dönmeli, %q döndü", sym, got)
		}
	}
}

func TestBytesGecerliSVG(t *testing.T) {
	for _, sym := range []string{"BTC", "ETH", "MIOTA"} {
		b, ok := Bytes(sym)
		if !ok {
			t.Fatalf("Bytes(%q) ikon bulmalı", sym)
		}
		if !bytes.Contains(b, []byte("<svg")) {
			t.Errorf("Bytes(%q) SVG kökü içermeli", sym)
		}
		low := strings.ToLower(string(b))
		for _, tehlikeli := range []string{"<script", "<foreignobject", "javascript:", " onload="} {
			if strings.Contains(low, tehlikeli) {
				t.Errorf("Bytes(%q) %q içermemeli", sym, tehlikeli)
			}
		}
	}
	if _, ok := Bytes("YOKBOYLE"); ok {
		t.Error("bilinmeyen sembol için Bytes ok=false dönmeli")
	}
}

func TestKutuphaneButunlugu(t *testing.T) {
	if got := Count(); got < 400 {
		t.Fatalf("ikon kütüphanesi eksik görünüyor: %d", got)
	}
	syms := Symbols()
	if len(syms) != Count() {
		t.Fatalf("Symbols sayısı Count ile eşleşmeli: %d != %d", len(syms), Count())
	}
	for i := 1; i < len(syms); i++ {
		if syms[i-1] >= syms[i] {
			t.Fatalf("Symbols sıralı olmalı: %q >= %q", syms[i-1], syms[i])
		}
	}
	for _, sym := range []string{"btc", "eth", "miota", "matic"} {
		if _, ok := Bytes(sym); !ok {
			t.Errorf("temel ikon eksik: %s", sym)
		}
	}
}
