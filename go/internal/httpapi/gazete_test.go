package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"theblock/site/internal/repo"
)

// ---- Yetki matrisi (plan §8.1) ----

func TestGazeteRoleMatrix(t *testing.T) {
	cases := []struct {
		role, action string
		want         bool
	}{
		// Yönetici: her şey
		{"yonetici", gazeteActionView, true},
		{"yonetici", gazeteActionEdit, true},
		{"yonetici", gazeteActionPublish, true},
		{"yonetici", gazeteActionSubs, true},
		{"yonetici", gazeteActionTeam, true},
		{"yonetici", gazeteActionLicense, true},
		// Sorumlu: lisans hariç
		{"sorumlu", gazeteActionEdit, true},
		{"sorumlu", gazeteActionPublish, true},
		{"sorumlu", gazeteActionSubs, true},
		{"sorumlu", gazeteActionTeam, true},
		{"sorumlu", gazeteActionLicense, false},
		// Editor: düzenleme + yayın; abone/ekip/lisans yok
		{"editor", gazeteActionView, true},
		{"editor", gazeteActionEdit, true},
		{"editor", gazeteActionPublish, true},
		{"editor", gazeteActionSubs, false},
		{"editor", gazeteActionTeam, false},
		{"editor", gazeteActionLicense, false},
		// İzleyici: yalnız görüntüleme
		{"izleyici", gazeteActionView, true},
		{"izleyici", gazeteActionEdit, false},
		{"izleyici", gazeteActionPublish, false},
		// Yetkisiz
		{"", gazeteActionView, false},
		{"", gazeteActionEdit, false},
	}
	for _, c := range cases {
		if got := gazeteRoleAllows(c.role, c.action); got != c.want {
			t.Errorf("gazeteRoleAllows(%q,%q)=%v istenen %v", c.role, c.action, got, c.want)
		}
	}
}

// ---- Varsayılan 10 sayfalık düzen ----

func TestDefaultGazetePages(t *testing.T) {
	pages := defaultGazetePages()
	if len(pages) != 10 {
		t.Fatalf("10 sayfa beklenirdi, %d", len(pages))
	}
	seenID := map[string]bool{}
	for _, p := range pages {
		if p.No < 1 || p.No > 10 {
			t.Errorf("sayfa no: %d", p.No)
		}
		if len(p.Blocks) == 0 {
			t.Errorf("sayfa %d boş", p.No)
		}
		for _, b := range p.Blocks {
			if b.ID == "" {
				t.Errorf("sayfa %d blok id'siz", p.No)
			}
			if seenID[b.ID] {
				t.Errorf("tekrar blok id: %s", b.ID)
			}
			seenID[b.ID] = true
			if b.X < 0 || b.Y < 0 || b.W <= 0 || b.H <= 0 || b.X+b.W > 100.1 || b.Y+b.H > 100.1 {
				t.Errorf("blok %s sınır dışı: x=%.1f y=%.1f w=%.1f h=%.1f", b.ID, b.X, b.Y, b.W, b.H)
			}
		}
	}
}

// ---- Abone sıklığı eşleşmesi ----

func TestGazeteFrequencyMatch(t *testing.T) {
	monday := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC) // Pazartesi
	sunday := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) // Pazar
	first := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	if !gazeteFrequencyMatch("gunluk", sunday, 1, 1) {
		t.Error("günlük her gün eşleşmeli")
	}
	if !gazeteFrequencyMatch("haftalik", monday, 1, 1) {
		t.Error("haftalık → pazartesi eşleşmeli")
	}
	if gazeteFrequencyMatch("haftalik", sunday, 1, 1) {
		t.Error("haftalık → pazar eşleşmemeli")
	}
	if !gazeteFrequencyMatch("aylik", first, 1, 1) {
		t.Error("aylık → ayın 1'i eşleşmeli")
	}
	if gazeteFrequencyMatch("aylik", monday, 1, 1) {
		t.Error("aylık → 26'sı eşleşmemeli")
	}
	if gazeteFrequencyMatch("bilinmeyen", monday, 1, 1) {
		t.Error("bilinmeyen sıklık eşleşmemeli")
	}
}

// ---- Blok render & kaçış ----

func TestGazeteBlockHTML(t *testing.T) {
	b := gazeteBlock{
		ID: "b1", Tip: "manset", X: 2, Y: 3, W: 96, H: 52,
		Icerik: &gazeteIcerik{Haberler: []gazeteArticle{{
			ID: 1, Baslik: `<script>alert("xss")</script>Manşet`, Spot: "spot",
			Kategori: "Gündem", Kaynak: "AA", Tarih: "25.10.2026",
		}}},
	}
	html := gazeteBlockHTML(b)
	if strings.Contains(html, "<script>") {
		t.Fatalf("başlık kaçırılmadı: %s", html)
	}
	if !strings.Contains(html, "left:2.00%") || !strings.Contains(html, "width:96.00%") {
		t.Fatalf("konum stili eksik: %s", html)
	}
	if !strings.Contains(html, "gb-manset") {
		t.Fatalf("blok sınıfı eksik: %s", html)
	}

	// Liste bloğu başlığı ve maddeleri
	l := gazeteBlock{ID: "b2", Tip: "liste", Baslik: "GÜNDEM",
		Icerik: &gazeteIcerik{Haberler: []gazeteArticle{{Baslik: "H1"}, {Baslik: "H2"}}}}
	lh := gazeteBlockHTML(l)
	if !strings.Contains(lh, "GÜNDEM") || !strings.Contains(lh, "<li>") {
		t.Fatalf("liste render: %s", lh)
	}
}

// ---- Sayfa render ----

func TestGazetePageHTML(t *testing.T) {
	blocks := []gazeteBlock{
		{ID: "a", Tip: "baslik", Baslik: "SPOR", X: 2, Y: 2, W: 96, H: 6},
		{ID: "b", Tip: "metin", Metin: "Merhaba\nDünya", X: 2, Y: 9, W: 47, H: 20},
	}
	raw, _ := json.Marshal(blocks)
	p := repo.GazetePage{PageNo: 2, Blocks: raw}
	out := gazetePageHTML(p, "Sayfa 2")
	if !strings.Contains(out, "Sayfa 2") {
		t.Fatalf("sayfa etiketi yok: %s", out)
	}
	if !strings.Contains(out, "gb-baslik") || !strings.Contains(out, "Merhaba") {
		t.Fatalf("bloklar render edilmedi: %s", out)
	}
}

// ---- Yardımcılar ----

func TestFirstLine(t *testing.T) {
	if got := firstLine("a\nb", 10); got != "a b" {
		t.Fatalf("firstLine: %q", got)
	}
	if got := firstLine("abcdefghij", 5); got != "abcde…" {
		t.Fatalf("firstLine kırpma: %q", got)
	}
}

func TestMediaAbs(t *testing.T) {
	if mediaAbs("media/202610/x.jpg") != "/media/202610/x.jpg" {
		t.Fatal("media yolu")
	}
	if mediaAbs("") != "" {
		t.Fatal("boş yol")
	}
}
