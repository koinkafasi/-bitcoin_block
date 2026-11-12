package searchsvc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// --- NewClient ---

func TestNewClientBosURLNil(t *testing.T) {
	for _, in := range []string{"", "   ", "\t\n"} {
		if c := NewClient(in); c != nil {
			t.Fatalf("NewClient(%q) = %+v; nil bekleniyordu", in, c)
		}
	}
}

func TestNewClientSondakiEgilCizgiKesilir(t *testing.T) {
	c := NewClient("  http://127.0.0.1:8171///  ")
	if c == nil {
		t.Fatal("istemci üretilmeliydi")
	}
	if c.BaseURL != "http://127.0.0.1:8171" {
		t.Fatalf("BaseURL = %q", c.BaseURL)
	}
}

// nil istemcide bütün uçlar sessiz dönmeli (çağıran yedek yola düşer).
func TestNilIstemciGuvenli(t *testing.T) {
	var c *Client
	ctx := context.Background()
	if c.Healthy(ctx) {
		t.Error("nil istemci sağlıklı olmamalı")
	}
	if n, err := c.Index(ctx, []Doc{{ID: "1"}}); n != 0 || err != nil {
		t.Errorf("Index = (%d, %v)", n, err)
	}
	if hits, err := c.Search(ctx, "x", 5); hits != nil || err != nil {
		t.Errorf("Search = (%v, %v)", hits, err)
	}
	if st, err := c.Stats(ctx); st != nil || err != nil {
		t.Errorf("Stats = (%v, %v)", st, err)
	}
}

// --- Healthy ---

func TestHealthyDurumKodlari(t *testing.T) {
	code := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf("beklenmeyen yol: %s", r.URL.Path)
		}
		w.WriteHeader(code)
	}))
	defer srv.Close()
	c := NewClient(srv.URL)

	if !c.Healthy(context.Background()) {
		t.Error("200 sağlıklı sayılmalı")
	}
	code = http.StatusServiceUnavailable
	if c.Healthy(context.Background()) {
		t.Error("503 sağlıksız sayılmalı")
	}
}

func TestHealthySunucuKapali(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // kapalı adres → bağlantı hatası
	c := NewClient(url)
	if c.Healthy(context.Background()) {
		t.Error("bağlantı yokken Healthy false dönmeli")
	}
}

// Bozuk adres (denetim karakteri) istek üretiminde hata verir.
func TestHealthyBozukAdres(t *testing.T) {
	c := &Client{BaseURL: "http://\x00bad", http: &http.Client{}}
	if c.Healthy(context.Background()) {
		t.Error("bozuk adreste Healthy false dönmeli")
	}
}

// --- Index ---

func TestIndexBosListeIstekYapmaz(t *testing.T) {
	cagri := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { cagri++ }))
	defer srv.Close()
	c := NewClient(srv.URL)
	n, err := c.Index(context.Background(), nil)
	if n != 0 || err != nil {
		t.Fatalf("Index = (%d, %v)", n, err)
	}
	if cagri != 0 {
		t.Error("boş listede HTTP isteği yapılmamalı")
	}
}

func TestIndexBasarili(t *testing.T) {
	var got struct {
		Docs []Doc `json:"docs"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/index" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"indexed":2}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	docs := []Doc{
		{ID: "1", Title: "Bir", URL: "/bir-1", Tags: []string{"gündem"}},
		{ID: "2", Title: "İki", URL: "/iki-2"},
	}
	n, err := c.Index(context.Background(), docs)
	if err != nil {
		t.Fatalf("Index hatası: %v", err)
	}
	if n != 2 {
		t.Errorf("indexed = %d, 2 bekleniyordu", n)
	}
	if len(got.Docs) != 2 || got.Docs[0].ID != "1" || got.Docs[1].Title != "İki" {
		t.Errorf("gövde yanlış: %+v", got.Docs)
	}
}

func TestIndexHataYollari(t *testing.T) {
	t.Run("500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		if _, err := NewClient(srv.URL).Index(context.Background(), []Doc{{ID: "1"}}); err == nil {
			t.Fatal("500 hatasız dönmemeli")
		}
	})
	t.Run("bozuk json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{`))
		}))
		defer srv.Close()
		if _, err := NewClient(srv.URL).Index(context.Background(), []Doc{{ID: "1"}}); err == nil {
			t.Fatal("bozuk gövde hatasız dönmemeli")
		}
	})
	t.Run("sunucu kapalı", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		u := srv.URL
		srv.Close()
		if _, err := NewClient(u).Index(context.Background(), []Doc{{ID: "1"}}); err == nil {
			t.Fatal("bağlantı hatası dönmeli")
		}
	})
}

// --- Search ---

func TestSearchSorguVeLimit(t *testing.T) {
	var rawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/search" {
			t.Errorf("yol = %s", r.URL.Path)
		}
		rawQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"hits":[{"id":"7","title":"Başlık","url":"/x-7","score":1.5}],"total":1,"took_ms":3}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL)

	hits, err := c.Search(context.Background(), "deprem şiddet", 0) // limit<=0 → 20
	if err != nil {
		t.Fatalf("Search hatası: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != "7" || hits[0].Score != 1.5 {
		t.Fatalf("hits = %+v", hits)
	}
	q, err := url.ParseQuery(rawQuery)
	if err != nil {
		t.Fatalf("sorgu ayrıştırılamadı: %v", err)
	}
	if got := q.Get("q"); got != "deprem şiddet" {
		t.Errorf("q = %q; Türkçe sorgu sunucuya aynı gelmeli", got)
	}
	if got := q.Get("limit"); got != "20" {
		t.Errorf("limit = %q; varsayılan 20 olmalı", got)
	}
}

func TestSearchBosSorguIstekYapmaz(t *testing.T) {
	cagri := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { cagri++ }))
	defer srv.Close()
	hits, err := NewClient(srv.URL).Search(context.Background(), "   ", 5)
	if hits != nil || err != nil {
		t.Fatalf("Search = (%v, %v)", hits, err)
	}
	if cagri != 0 {
		t.Error("boş sorguda istek yapılmamalı")
	}
}

func TestSearchHataYollari(t *testing.T) {
	t.Run("500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer srv.Close()
		if _, err := NewClient(srv.URL).Search(context.Background(), "x", 1); err == nil {
			t.Fatal("500 hatasız dönmemeli")
		}
	})
	t.Run("bozuk json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<html>`))
		}))
		defer srv.Close()
		if _, err := NewClient(srv.URL).Search(context.Background(), "x", 1); err == nil {
			t.Fatal("bozuk gövde hatasız dönmemeli")
		}
	})
	t.Run("sunucu kapalı", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		u := srv.URL
		srv.Close()
		if _, err := NewClient(u).Search(context.Background(), "x", 1); err == nil {
			t.Fatal("bağlantı hatası dönmeli")
		}
	})
}

// --- Stats ---

func TestStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/stats" {
			t.Errorf("yol = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"docs":42,"size_mb":7}`))
	}))
	defer srv.Close()
	st, err := NewClient(srv.URL).Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats hatası: %v", err)
	}
	if st["docs"] != float64(42) {
		t.Errorf("docs = %v", st["docs"])
	}
}

func TestStatsBozukGovde(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`)) // dizi: map'e çözülemez
	}))
	defer srv.Close()
	if _, err := NewClient(srv.URL).Stats(context.Background()); err == nil {
		t.Fatal("dizi gövde hatasız dönmemeli")
	}
}

// --- urlQueryEscape (gerçek hata regresyonu) ---

func TestURLQueryEscapeCokBaytli(t *testing.T) {
	kasalar := map[string]string{
		"şeker":         "%C5%9Feker",
		"ÇĞİÖŞÜ":        "%C3%87%C4%9E%C4%B0%C3%96%C5%9E%C3%9C",
		"deprem şiddet": "deprem+%C5%9Fiddet",
		"a-b_c.d~e":     "a-b_c.d~e",
		"100% & =?":     "100%25+%26+%3D%3F",
		"emoji 🎯":       "emoji+%F0%9F%8E%AF",
	}
	for in, want := range kasalar {
		if got := urlQueryEscape(in); got != want {
			t.Errorf("urlQueryEscape(%q) = %q; %q bekleniyordu", in, got, want)
		}
	}
	// Standart kitaplıkla birebir aynı olmalı (tek doğruluk kaynağı).
	for _, in := range []string{"şeker", "ÇĞİÖŞÜ", "a b", "100%", "x/y?z"} {
		if got, want := urlQueryEscape(in), url.QueryEscape(in); got != want {
			t.Errorf("urlQueryEscape(%q) = %q; std %q", in, got, want)
		}
	}
}

// Üretilen yol gerçekten UTF-8 baytları taşımalı (çift kodlama olmamalı).
func TestSearchYolundaTurkceBozulmaz(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"hits":[]}`))
	}))
	defer srv.Close()
	if _, err := NewClient(srv.URL).Search(context.Background(), "Çanakkale", 3); err != nil {
		t.Fatalf("Search hatası: %v", err)
	}
	if !strings.Contains(gotQuery, "%C3%87anakkale") {
		t.Fatalf("sorgu UTF-8 kodlanmamış: %s", gotQuery)
	}
	q, _ := url.ParseQuery(gotQuery)
	if q.Get("q") != "Çanakkale" {
		t.Fatalf("geri çözülen sorgu = %q", q.Get("q"))
	}
}
