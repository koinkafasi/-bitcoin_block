package mailer

import (
	"encoding/json"
	"testing"

	"theblock/site/internal/platform"
)

// ayar, repo.GetSettings'in döndürdüğü ham baytları taklit eder.
func ayar(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func TestParseConfigTam(t *testing.T) {
	gs := map[string][]byte{
		"smtp_host":      ayar("smtp.example.com"),
		"smtp_port":      ayar("465"),
		"smtp_user":      ayar("kullanici"),
		"smtp_pass":      ayar("gizli"),
		"smtp_from":      ayar("haber@example.com"),
		"smtp_from_name": ayar("theblock"),
		"smtp_tls":       ayar("1"),
	}
	c := ParseConfig(gs)
	if c.Host != "smtp.example.com" || c.Port != 465 || c.User != "kullanici" ||
		c.Pass != "gizli" || c.From != "haber@example.com" || c.FromName != "theblock" {
		t.Fatalf("yanlış ayrıştırma: %+v", c)
	}
	if !c.ImplicitTLS {
		t.Error("smtp_tls=1 → ImplicitTLS true olmalı")
	}
	if !c.Configured() {
		t.Error("host+from dolu → Configured true olmalı")
	}
}

func TestParseConfigBos(t *testing.T) {
	c := ParseConfig(nil)
	if c.Port != 587 {
		t.Errorf("varsayılan port = %d; 587 bekleniyordu", c.Port)
	}
	if c.Configured() {
		t.Error("boş ayarda Configured false olmalı")
	}
	if c.ImplicitTLS {
		t.Error("ImplicitTLS varsayılan false olmalı")
	}
}

// Eski kurulumlar düz metin (JSON'suz) değer yazmış olabilir; ikisi de okunmalı.
func TestParseConfigDuzMetinDeger(t *testing.T) {
	gs := map[string][]byte{
		"smtp_host": []byte(`"smtp.duz"`), // tırnaklı düz metin
		"smtp_port": []byte(`"2525"`),
		"smtp_from": []byte(`haber@duz`), // tırnaksız düz metin
		"smtp_tls":  []byte(`"0"`),
	}
	c := ParseConfig(gs)
	if c.Host != "smtp.duz" {
		t.Errorf("Host = %q", c.Host)
	}
	if c.Port != 2525 {
		t.Errorf("Port = %d", c.Port)
	}
	if c.From != "haber@duz" {
		t.Errorf("From = %q", c.From)
	}
	if c.ImplicitTLS {
		t.Error("smtp_tls=0 → false olmalı")
	}
}

// Bozuk port varsayılana düşmeli; geçersiz aralık da kabul edilmemeli.
func TestParseConfigGecersizPort(t *testing.T) {
	kasalar := []string{"", "abc", "58a7", "-25", "0", "99999999", "70000", "  "}
	for _, p := range kasalar {
		c := ParseConfig(map[string][]byte{"smtp_port": ayar(p)})
		if c.Port != 587 {
			t.Errorf("smtp_port=%q → Port=%d; 587 bekleniyordu", p, c.Port)
		}
	}
	// Sınır değerleri geçerli.
	for _, p := range []string{"1", "25", "465", "587", "65535"} {
		c := ParseConfig(map[string][]byte{"smtp_port": ayar(p)})
		var want int
		for _, r := range p {
			want = want*10 + int(r-'0')
		}
		if c.Port != want {
			t.Errorf("smtp_port=%q → Port=%d; %d bekleniyordu", p, c.Port, want)
		}
	}
}

// SMTP yoksa Send sessizce nil döner (çağıran akışı sürer).
func TestSendConfigYapilandirilmamisSessiz(t *testing.T) {
	if err := SendConfig(platform.SMTPConfig{}, "uye@example.com", "Konu", "<p>x</p>"); err != nil {
		t.Fatalf("yapılandırılmamışta hata dönmemeli: %v", err)
	}
	// Host var, From yok → hâlâ yapılandırılmamış.
	if err := SendConfig(platform.SMTPConfig{Host: "smtp.example.com"}, "uye@example.com", "Konu", "x"); err != nil {
		t.Fatalf("from eksikken hata dönmemeli: %v", err)
	}
}

// Alıcı boşsa (boşluk dahil) gönderim denenmez.
func TestSendConfigBosAlici(t *testing.T) {
	cfg := platform.SMTPConfig{Host: "smtp.example.com", From: "a@b.c", Port: 1}
	for _, to := range []string{"", "   ", "\t"} {
		if err := SendConfig(cfg, to, "Konu", "x"); err != nil {
			t.Errorf("alıcı %q iken hata: %v", to, err)
		}
	}
}

// Gerçek gönderim hatası YUTULMAZ: port 1'de dinleyen yok.
func TestSendConfigGonderimHatasiDoner(t *testing.T) {
	cfg := platform.SMTPConfig{Host: "127.0.0.1", Port: 1, From: "a@b.c"}
	err := SendConfig(cfg, "uye@example.com", "Konu", "<p>x</p>")
	if err == nil {
		t.Fatal("ulaşılamayan SMTP hatasız dönmemeli (sessiz yutma regresyonu)")
	}
}
