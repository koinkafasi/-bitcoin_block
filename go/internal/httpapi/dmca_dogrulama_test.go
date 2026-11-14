package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"theblock/site/internal/webui/templkit"
)

// TestDMCADogrulamaDosyasi serves the DMCA validation file and checks the
// exact token. DMCA compares the body verbatim, so a stray newline or a
// wrapping HTML tag would silently break ownership verification — the test
// exists to catch exactly that.
func TestDMCADogrulamaDosyasi(t *testing.T) {
	var s Server
	rr := httptest.NewRecorder()
	s.handleDMCADogrulama(rr, httptest.NewRequest(http.MethodGet, "/dmca-validation.html", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("durum = %d, 200 bekleniyordu", rr.Code)
	}
	if got := rr.Body.String(); got != dmcaDogrulamaJeton {
		t.Fatalf("gövde = %q, beklenen %q", got, dmcaDogrulamaJeton)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, text/html bekleniyordu", ct)
	}
}

// TestDMCADogrulamaMetaEtiketi guards the second half of the same
// verification. DMCA accepts either the file or this meta tag; if the tag
// drops out of the head template, the alternate check fails and nobody
// notices until the badge silently loses its verified status.
func TestDMCADogrulamaMetaEtiketi(t *testing.T) {
	var b bytes.Buffer
	if err := templkit.CDHeadParca(templkit.CDHead{Title: "doğrulama"}).Render(context.Background(), &b); err != nil {
		t.Fatalf("head render: %v", err)
	}
	istenen := `name="dmca-site-verification" content="` + dmcaDogrulamaJeton + `"`
	if !strings.Contains(b.String(), istenen) {
		t.Fatalf("head çıktısında dmca-site-verification meta etiketi yok")
	}
}
