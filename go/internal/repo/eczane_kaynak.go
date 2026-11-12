package repo

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// EczaneKaynakURL bir ilin nöbetçi eczane sayfasının adresidir.
func EczaneKaynakURL(slug string) string {
	return "https://eczaneler.org/" + slug + "-nobetci-eczaneleri"
}

// eczaneHTTPTimeout: sayfalar 150–540 KB; 6 sn yetmiyordu.
const eczaneHTTPTimeout = 20 * time.Second

var (
	// Kayıt çapası: her eczanede telefon bağlantısı var.
	eczaneTelRe = regexp.MustCompile(`href="tel:\+?90([0-9]{10})"`)
	// Ad, kayıt başlığındaki title alanında.
	eczaneAdRe = regexp.MustCompile(`title="([^"]{2,120})"\s+href="/[a-z0-9-]+-eczanesi"`)
	// Adres: kart içindeki <p class="… line-clamp-2 …">.
	eczaneAdresRe = regexp.MustCompile(`line-clamp-2 leading-relaxed">([^<]{5,200})</p>`)
	// İl / İlçe: gerçek biçim  >Ankara<!-- --> / <!-- -->Çankaya</span>
	// (React araya yorum düğümü koyuyor; ikinci grup ilçe).
	eczaneIlceRe = regexp.MustCompile(`>([^<]{2,40})<!-- -->\s*/\s*<!-- -->([^<]{2,60})<`)
	// Mahalle: ilçeden sonraki ilk <span>.
	eczaneMahalleRe = regexp.MustCompile(`</svg><span>([^<]{2,60})</span>`)
	// "Tarif:" bloğu (adres tarifi) — adres alanına eklenir.
	eczaneTarifRe = regexp.MustCompile(`mr-1">Tarif:</span><span class="line-clamp-2">([^<]{2,160})</span>`)
)

func eczaneCozHTML(il string, ham []byte) []Eczane {
	s := string(ham)
	telYerleri := eczaneTelRe.FindAllStringSubmatchIndex(s, -1)
	if len(telYerleri) == 0 {
		return nil
	}
	var out []Eczane
	for _, t := range telYerleri {
		bas := t[0] - 6000
		if bas < 0 {
			bas = 0
		}
		pencere := s[bas:t[0]]

		var e Eczane
		// Ad: penceredeki SON eşleşme bu kaydın adıdır.
		if m := eczaneAdRe.FindAllStringSubmatch(pencere, -1); len(m) > 0 {
			e.Ad = temizMetin(m[len(m)-1][1])
		}
		if e.Ad == "" {
			continue
		}
		if m := eczaneAdresRe.FindAllStringSubmatch(pencere, -1); len(m) > 0 {
			e.Adres = temizMetin(m[len(m)-1][1])
		}
		if m := eczaneIlceRe.FindAllStringSubmatch(pencere, -1); len(m) > 0 {
			e.Ilce = temizMetin(m[len(m)-1][2])
		}
		if m := eczaneMahalleRe.FindAllStringSubmatch(pencere, -1); len(m) > 0 {
			// Mahalle adresi çoğu zaman zaten içeriyor; tekrar etme.
			if mah := temizMetin(m[len(m)-1][1]); mah != "" && e.Adres != "" &&
				!strings.Contains(strings.ToLower(e.Adres), strings.ToLower(mah)) {
				e.Adres = mah + " — " + e.Adres
			}
		}
		if m := eczaneTarifRe.FindAllStringSubmatch(pencere, -1); len(m) > 0 {
			if tarif := temizMetin(m[len(m)-1][1]); tarif != "" {
				e.Adres = strings.TrimSpace(e.Adres + " (Tarif: " + tarif + ")")
			}
		}
		// Telefon: +90XXXXXXXXXX → 0XXX XXX XX XX
		if len(t) >= 4 {
			num := s[t[2]:t[3]]
			e.Tel = turkceTelefon(num)
		}
		if e.Ilce == "" {
			e.Ilce = il
		}
		out = append(out, e)
	}
	return eczaneTekrarsiz(out)
}

func eczaneTekrarsiz(list []Eczane) []Eczane {
	if len(list) < 2 {
		return list
	}
	seen := make(map[string]bool, len(list))
	out := make([]Eczane, 0, len(list))
	for _, e := range list {
		ad := strings.ToLower(temizMetin(e.Ad))
		tel := strings.Join(strings.Fields(e.Tel), "")
		anahtar := ad + "|" + tel
		if tel == "" {
			// Telefon yoksa ad + adres ile ayırt et; boş anahtarla
			// birbirinden farklı kayıtları eleme.
			anahtar = ad + "|" + strings.ToLower(temizMetin(e.Adres))
		}
		if ad == "" || seen[anahtar] {
			continue
		}
		seen[anahtar] = true
		out = append(out, e)
	}
	return out
}

// temizMetin HTML varlıklarını çözer ve boşlukları teker.
func temizMetin(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return strings.Join(strings.Fields(s), " ")
}

// turkceTelefon "5010086699" → "0 501 008 66 99".
func turkceTelefon(onHaneli string) string {
	if len(onHaneli) != 10 {
		return onHaneli
	}
	return "0 " + onHaneli[0:3] + " " + onHaneli[3:6] + " " +
		onHaneli[6:8] + " " + onHaneli[8:10]
}

func eczaneEczanelerOrg(ctx context.Context, slug string) ([]Eczane, error) {
	raw, err := httpGetTarayici(ctx, EczaneKaynakURL(slug), eczaneHTTPTimeout)
	if err != nil {
		return nil, err
	}
	return eczaneCozHTML(slug, raw), nil
}

const tarayiciUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// httpGetTarayici tarayıcı başlıklarıyla GET yapar ve gövdeyi döndürür.
func httpGetTarayici(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", tarayiciUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "tr-TR,tr;q=0.9")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	// Sayfalar 540 KB'a kadar çıkıyor; 2 MB sınırı yeterli.
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}
