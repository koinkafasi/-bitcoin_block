package config

import (
	"os"
	"testing"
	"time"
)

// TestGetInt bozuk değerin varsayılana döndüğünü doğrular.
// Yapılandırma hatası yüzünden sürecin açılmaması gerekir.
func TestGetInt(t *testing.T) {
	cases := []struct {
		raw  string
		def  int
		want int
	}{
		{"", 7, 7},      // eksik → varsayılan
		{"12", 7, 12},   // geçerli
		{"abc", 7, 7},   // bozuk → varsayılan
		{"-5", 7, 7},    // negatif → varsayılan
		{" 42 ", 7, 42}, // boşluklu
		{"0", 9, 0},     // sıfır geçerli (ayarlanmadı demek)
		{"3.5", 7, 7},   // ondalık → varsayılan
	}
	for _, c := range cases {
		if c.raw == "" {
			_ = os.Unsetenv("TB_TEST_INT")
		} else {
			_ = os.Setenv("TB_TEST_INT", c.raw)
		}
		if got := getInt("TB_TEST_INT", c.def); got != c.want {
			t.Errorf("getInt(%q, %d) = %d, beklenen %d", c.raw, c.def, got, c.want)
		}
	}
	_ = os.Unsetenv("TB_TEST_INT")
}

// TestGetDuration süre ayrıştırmasının bozuk girdide varsayılana döndüğünü
// doğrular.
func TestGetDuration(t *testing.T) {
	cases := []struct {
		raw  string
		def  time.Duration
		want time.Duration
	}{
		{"", 25 * time.Second, 25 * time.Second},
		{"30s", 25 * time.Second, 30 * time.Second},
		{"2m", 25 * time.Second, 2 * time.Minute},
		{"bozuk", 25 * time.Second, 25 * time.Second},
		{"-5s", 25 * time.Second, 25 * time.Second},
		{"10", 25 * time.Second, 25 * time.Second}, // birimsiz → geçersiz
	}
	for _, c := range cases {
		if c.raw == "" {
			_ = os.Unsetenv("TB_TEST_DUR")
		} else {
			_ = os.Setenv("TB_TEST_DUR", c.raw)
		}
		if got := getDuration("TB_TEST_DUR", c.def); got != c.want {
			t.Errorf("getDuration(%q, %v) = %v, beklenen %v", c.raw, c.def, got, c.want)
		}
	}
	_ = os.Unsetenv("TB_TEST_DUR")
}

// TestPoolConfigAyarlanmamis varsayılanların KORUNDUĞUNU doğrular:
// bu değişiklik mevcut kurulumların davranışını değiştirmemeli.
func TestPoolConfigAyarlanmamis(t *testing.T) {
	c := Config{DatabaseURL: "postgres://u:p@127.0.0.1:5432/db?sslmode=disable"}
	pc, err := c.PoolConfig()
	if err != nil {
		t.Fatalf("PoolConfig hata verdi: %v", err)
	}
	// pgx varsayılanı: MaxConns = max(4, NumCPU). Sıfır bırakılırsa
	// varsayılan korunmalı — yani MaxConns 0 OLMAMALI.
	if pc.MaxConns <= 0 {
		t.Errorf("MaxConns varsayılanı korunmadı: %d", pc.MaxConns)
	}
	if pc.MinConns < 0 {
		t.Errorf("MinConns negatif: %d", pc.MinConns)
	}
}

// TestPoolConfigUygulanir açık ayarların geçtiğini doğrular.
func TestPoolConfigUygulanir(t *testing.T) {
	c := Config{
		DatabaseURL:       "postgres://u:p@127.0.0.1:5432/db?sslmode=disable",
		DBMaxConns:        40,
		DBMinConns:        5,
		DBMaxConnLifetime: 30 * time.Minute,
		DBMaxConnIdleTime: 5 * time.Minute,
	}
	pc, err := c.PoolConfig()
	if err != nil {
		t.Fatalf("PoolConfig hata verdi: %v", err)
	}
	if pc.MaxConns != 40 {
		t.Errorf("MaxConns = %d, beklenen 40", pc.MaxConns)
	}
	if pc.MinConns != 5 {
		t.Errorf("MinConns = %d, beklenen 5", pc.MinConns)
	}
	if pc.MaxConnLifetime != 30*time.Minute {
		t.Errorf("MaxConnLifetime = %v", pc.MaxConnLifetime)
	}
	if pc.MaxConnIdleTime != 5*time.Minute {
		t.Errorf("MaxConnIdleTime = %v", pc.MaxConnIdleTime)
	}
}

// TestPoolConfigMinMaxKirpilir MinConns > MaxConns olursa pgx hata verir;
// biz kırpıyoruz ki yapılandırma hatası süreci düşürmesin.
func TestPoolConfigMinMaxKirpilir(t *testing.T) {
	c := Config{
		DatabaseURL: "postgres://u:p@127.0.0.1:5432/db?sslmode=disable",
		DBMaxConns:  10,
		DBMinConns:  50,
	}
	pc, err := c.PoolConfig()
	if err != nil {
		t.Fatalf("PoolConfig hata verdi: %v", err)
	}
	if pc.MinConns != 10 {
		t.Errorf("MinConns kırpılmadı: %d (MaxConns 10)", pc.MinConns)
	}
}

func TestPoolConfigBozukDSNHataDoner(t *testing.T) {
	c := Config{DatabaseURL: "bu-bir-url-degil %%%"}
	pc, err := c.PoolConfig()
	if err == nil {
		t.Fatal("bozuk DSN hata döndürmeliydi")
	}
	if pc != nil {
		t.Error("hata varken yapı da dönmemeli (nil beklenirdi)")
	}
}

// TestShutdownWindow varsayılanın 25 sn olduğunu, ayarlanırsa kullanıldığını
// doğrular.
func TestShutdownWindow(t *testing.T) {
	if got := (Config{}).ShutdownWindow(); got != 25*time.Second {
		t.Errorf("varsayılan = %v, beklenen 25s", got)
	}
	if got := (Config{ShutdownTimeout: 10 * time.Second}).ShutdownWindow(); got != 10*time.Second {
		t.Errorf("ayarlı = %v, beklenen 10s", got)
	}
}
