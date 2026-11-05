package db

import (
	"strings"
	"testing"
	"testing/fstest"
)

// --- Gerçek (embedded) migration seti ---

func TestLoadMigrationsGercek(t *testing.T) {
	ls, err := LoadMigrations()
	if err != nil {
		t.Fatalf("LoadMigrations hatası: %v", err)
	}
	if len(ls) == 0 {
		t.Fatal("hiç migration yüklenmedi")
	}

	onceki := 0
	for i, m := range ls {
		if m.Version <= 0 {
			t.Errorf("[%d] geçersiz sürüm: %d", i, m.Version)
		}
		if m.Version <= onceki {
			t.Errorf("[%d] sıra bozuk: %d <= %d", i, m.Version, onceki)
		}
		onceki = m.Version
		if strings.TrimSpace(m.Up) == "" {
			t.Errorf("%d_%s: up gövdesi boş", m.Version, m.Name)
		}
		if m.Name == "" {
			t.Errorf("%d: ad boş", m.Version)
		}
		if m.HasDown && strings.TrimSpace(m.Down) == "" {
			t.Errorf("%d_%s: HasDown true ama down boş", m.Version, m.Name)
		}
		if !m.HasDown && m.Down != "" {
			t.Errorf("%d_%s: HasDown false ama down dolu", m.Version, m.Name)
		}
	}

	// İlk migration çekirdek şemadır: sürüm 1 ve `users` tablosu.
	if ls[0].Version != 1 {
		t.Errorf("ilk sürüm = %d; 1 bekleniyordu", ls[0].Version)
	}
	if !strings.Contains(ls[0].Up, "CREATE TABLE users") {
		t.Error("0001 up içinde users tablosu yok")
	}
}

// Aynı sürümün up ve down dosyaları TEK Migration'da birleşmeli.
func TestLoadMigrationsUpDownBirlestirir(t *testing.T) {
	fsys := fstest.MapFS{
		"0007_ornek.up.sql":   {Data: []byte("UP7")},
		"0007_ornek.down.sql": {Data: []byte("DOWN7")},
	}
	ls, err := loadFrom(fsys)
	if err != nil {
		t.Fatalf("hata: %v", err)
	}
	if len(ls) != 1 {
		t.Fatalf("%d migration; 1 bekleniyordu", len(ls))
	}
	m := ls[0]
	if m.Version != 7 || m.Name != "ornek" || m.Up != "UP7" || m.Down != "DOWN7" || !m.HasDown {
		t.Fatalf("yanlış birleştirme: %+v", m)
	}
}

// Sıralama sayısal olmalı (0002 < 0010); dizge sıralaması 0010'u öne alırdı.
func TestLoadMigrationsSayisalSiralama(t *testing.T) {
	fsys := fstest.MapFS{
		"0010_on.up.sql":  {Data: []byte("10")},
		"0002_iki.up.sql": {Data: []byte("2")},
		"0100_yuz.up.sql": {Data: []byte("100")},
	}
	ls, err := loadFrom(fsys)
	if err != nil {
		t.Fatalf("hata: %v", err)
	}
	want := []int{2, 10, 100}
	if len(ls) != len(want) {
		t.Fatalf("%d migration; %d bekleniyordu", len(ls), len(want))
	}
	for i, v := range want {
		if ls[i].Version != v {
			t.Errorf("[%d] sürüm = %d; %d bekleniyordu", i, ls[i].Version, v)
		}
	}
}

func TestLoadMigrationsSqlOlmayanYoksayilir(t *testing.T) {
	fsys := fstest.MapFS{
		"0001_a.up.sql": {Data: []byte("x")},
		"README.md":     {Data: []byte("belge")},
		".gitkeep":      {Data: []byte("")},
	}
	ls, err := loadFrom(fsys)
	if err != nil {
		t.Fatalf("hata: %v", err)
	}
	if len(ls) != 1 {
		t.Fatalf("%d migration; yalnız .sql sayılmalı", len(ls))
	}
}

func TestLoadMigrationsBosDizin(t *testing.T) {
	ls, err := loadFrom(fstest.MapFS{})
	if err != nil {
		t.Fatalf("boş dizin hata vermemeli: %v", err)
	}
	if len(ls) != 0 {
		t.Fatalf("%d migration; 0 bekleniyordu", len(ls))
	}
}

// --- Hata yolları ---

func TestLoadMigrationsHatalari(t *testing.T) {
	kasalar := []struct {
		ad    string
		dosya map[string]string
		ipucu string
	}{
		{
			ad:    "up/down uzantısı yok",
			dosya: map[string]string{"0001_eksik.sql": "x"},
			ipucu: "up/down yok",
		},
		{
			ad:    "sürüm ayırıcısı yok",
			dosya: map[string]string{"bozuk.up.sql": "x"},
			ipucu: "sürüm yok",
		},
		{
			ad:    "sayısal olmayan sürüm",
			dosya: map[string]string{"x1_abc.up.sql": "x"},
			ipucu: "geçersiz sürüm",
		},
		{
			ad:    "bilinmeyen yön",
			dosya: map[string]string{"0001_abc.side.sql": "x"},
			ipucu: "bilinmeyen yön",
		},
	}
	for _, k := range kasalar {
		t.Run(k.ad, func(t *testing.T) {
			fsys := fstest.MapFS{}
			for name, body := range k.dosya {
				fsys[name] = &fstest.MapFile{Data: []byte(body)}
			}
			ls, err := loadFrom(fsys)
			if err == nil {
				t.Fatalf("hata bekleniyordu, alındı: %+v", ls)
			}
			if !strings.Contains(err.Error(), k.ipucu) {
				t.Fatalf("hata metni %q, %q içermeli", err.Error(), k.ipucu)
			}
		})
	}
}
