package repo

import "testing"

func TestGorselAdresiCoz(t *testing.T) {
	proxy := "https://www.coindesk.com/_next/image?url=https%3A%2F%2Fcdn.sanity.io%2Fimages%2Fs3y3vcno%2Fproduction%2Fa.jpg%3Frect%3D0%2C0%2C10%2C10&w=1200&q=75"
	beklenen := "https://cdn.sanity.io/images/s3y3vcno/production/a.jpg?rect=0,0,10,10"
	if got := GorselAdresiCoz(proxy); got != beklenen {
		t.Errorf("proxy çözülmedi:\n  gelen: %q\n  beklenen: %q", got, beklenen)
	}

	// Proxy olmayan adres DEĞİŞMEZ.
	for _, a := range []string{
		"https://cdn.sanity.io/images/x/y.jpg",
		"https://static.cryptobriefing.com/a/b.webp",
		"",
	} {
		if got := GorselAdresiCoz(a); got != a {
			t.Errorf("adres bozuldu: %q → %q", a, got)
		}
	}

	// Proxy ama iç adres geçersiz şema: dış adres korunur (uydurma yok).
	kotu := "https://www.coindesk.com/_next/image?url=javascript%3Aalert(1)&w=10"
	if got := GorselAdresiCoz(kotu); got != kotu {
		t.Errorf("geçersiz iç adres kabul edildi: %q", got)
	}
}
