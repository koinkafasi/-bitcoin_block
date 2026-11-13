package templkit

import (
	"strings"
	"testing"
)

// Kaynak satırı İngilizce sayfalarda da Türkçe basılıyordu (canlıda
// ölçüldü: "Kaynak: CryptoBriefing — bu içerik özet ve atıf ile
// sunulur"). Okuyucu için kırık, arama motoru için karışık dil sinyali.
func TestCDMakaleKaynakBloguDiliIzler(t *testing.T) {
	testler := []struct {
		ad      string
		dil     CDMakaleDil
		bekle   []string
		beklemE []string
	}{
		{
			ad:      "türkçe",
			dil:     CDMakaleDilTR(),
			bekle:   []string{"Kaynak", "tam metin kaynağındadır"},
			beklemE: []string{"Source", "the full text is at the source"},
		},
		{
			ad:      "ingilizce",
			dil:     CDMakaleDilEN(),
			bekle:   []string{"Source", "the full text is at the source"},
			beklemE: []string{"tam metin kaynağındadır"},
		},
	}
	for _, tt := range testler {
		t.Run(tt.ad, func(t *testing.T) {
			v := ornekMakale()
			v.Dil = tt.dil
			v.SourceName = "CoinDesk"
			v.SourceURL = "https://www.coindesk.com/x"

			html := makaleCiz(t, v)
			if !strings.Contains(html, `class="cd-source"`) {
				t.Fatal("kaynak bloğu basılmadı")
			}
			for _, s := range tt.bekle {
				if !strings.Contains(html, s) {
					t.Errorf("%q bekleniyordu", s)
				}
			}
			for _, s := range tt.beklemE {
				if strings.Contains(html, s) {
					t.Errorf("%q basılmamalıydı (yanlış dil)", s)
				}
			}
		})
	}
}

// nofollow, satırın kendi iddiasını yalanlıyordu: "Kaynak budur" deyip
// bağlantıya "bu bağlantıya kefil değilim" demek. Google nofollow'u
// güvenilmeyen ve sponsorlu bağlantılar için tanımlıyor; editoryal
// kaynak atfı ikisi de değil.
func TestCDMakaleKaynakBaglantisiNofollowDegil(t *testing.T) {
	v := ornekMakale()
	v.SourceName = "CoinDesk"
	v.SourceURL = "https://www.coindesk.com/x"

	html := makaleCiz(t, v)
	i := strings.Index(html, `class="cd-source"`)
	if i < 0 {
		t.Fatal("kaynak bloğu basılmadı")
	}
	j := strings.Index(html[i:], "</div>")
	blok := html[i : i+j]

	if strings.Contains(blok, "nofollow") {
		t.Error("kaynak bağlantısında nofollow var")
	}
	if !strings.Contains(blok, "noopener") {
		t.Error("target=_blank bağlantısında noopener olmalı")
	}
	if !strings.Contains(blok, "https://www.coindesk.com/x") {
		t.Error("kaynak adresi basılmadı")
	}
}

// Adı olmayan kaynakta blok hiç basılmamalı: boş "Kaynak:" satırı
// okuyucuya bir şey söylemez.
func TestCDMakaleKaynakYokBlokYok(t *testing.T) {
	html := makaleCiz(t, ornekMakale())
	if strings.Contains(html, `class="cd-source"`) {
		t.Error("kaynaksız içerikte blok basılmamalı")
	}
}
