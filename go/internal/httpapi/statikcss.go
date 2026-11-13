package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
)

// cssYorumlariSil CSS'ten yorumları (/* ... */) çıkarır.
//
// NEDEN SUNUMDA SOYUYORUZ (kaynaktan silmiyoruz):
// depodaki CSS yorumları ölçüm ve gerekçe taşıyor ("şu kural şu yüzden
// böyle", "ölçüldü: X px"). Bunlar bakım için değerli; silinirse bilgi
// kaybı olur ve aynı hatalar geri gelir. Dışarıya ise yalnız TEMİZ CSS
// gitmeli: yorumlar iç yapıyı, planlanan işleri ve ölçüm yöntemini
// açıklıyor — kaynağı inceleyen biri için gereksiz bilgi.
//
// Sözdizimi güvenliği: yorum benzeri diziler STRING ve url() içinde
// geçebilir (ör. content: "/*"; background: url("a/*b.png")). Naif
// `strings.Index("/*")` bunları bozar. Bu yüzden tarayıcı tırnak
// durumunu izler: tırnak içindeyken yorum AÇILMAZ.
//
// Yorumun yerine tek boşluk bırakılır: `a/*x*/b` → `a b` (iki token
// birleşip anlam değiştirmesin).
func cssYorumlariSil(girdi []byte) []byte {
	// Tipik durumda yorum yok ya da az: kapasiteyi girdi kadar ayır.
	cikti := make([]byte, 0, len(girdi))
	n := len(girdi)
	i := 0
	for i < n {
		c := girdi[i]

		// Yorum başlangıcı (tırnak dışındayız).
		if c == '/' && i+1 < n && girdi[i+1] == '*' {
			j := bytes.Index(girdi[i+2:], []byte("*/"))
			if j < 0 {
				// Kapanmamış yorum: kalanı at. (Geçersiz CSS; yarısı
				// yorum, yarısı kural olarak sunulmasın.)
				break
			}
			son := i + 2 + j + 2

			// Yalnız GEREKLİYSE boşluk: iki taraf da "sözcük" karakteriyse
			// birleşip anlam değiştirebilir (`a/*x*/b` → `a b`). Noktalama
			// zaten ayırıyorsa (`.g/*x*/{`) boşluk eklenmez — çıktı daha
			// küçük kalır ve girdinin birebir alt dizisi olur.
			var onceki byte
			if len(cikti) > 0 {
				onceki = cikti[len(cikti)-1]
			}
			var sonraki byte
			if son < n {
				sonraki = girdi[son]
			}
			if sozcukKarakteri(onceki) && sozcukKarakteri(sonraki) {
				cikti = append(cikti, ' ')
			}
			i = son
			continue
		}

		// Tırnaklı string: kapanana kadar AYNEN kopyala.
		if c == '"' || c == '\'' {
			q := c
			cikti = append(cikti, c)
			i++
			for i < n {
				if girdi[i] == '\\' && i+1 < n {
					cikti = append(cikti, girdi[i], girdi[i+1])
					i += 2
					continue
				}
				cikti = append(cikti, girdi[i])
				if girdi[i] == q {
					i++
					break
				}
				i++
			}
			continue
		}

		// url(...) içinde tırnaksız da olabilir: parantez kapanana kadar
		// ilerle. (data: URI'lerde /* görünebilir.)
		if (c == 'u' || c == 'U') && i+4 <= n && strings.EqualFold(string(girdi[i:i+4]), "url(") {
			cikti = append(cikti, girdi[i:i+4]...)
			i += 4
			derinlik := 1
			for i < n && derinlik > 0 {
				if girdi[i] == '\\' && i+1 < n {
					cikti = append(cikti, girdi[i], girdi[i+1])
					i += 2
					continue
				}
				if girdi[i] == '(' {
					derinlik++
				} else if girdi[i] == ')' {
					derinlik--
				}
				cikti = append(cikti, girdi[i])
				i++
			}
			continue
		}

		cikti = append(cikti, c)
		i++
	}

	return bosSatirlariSik(cikti)
}

// sozcukKarakteri iki token'ın yorum silinince birleşip birleşmeyeceğini
// belirler. Harf, rakam, tire, alt çizgi, yüzde ve kaçış "sözcük" sayılır;
// noktalama (`{`, `}`, `;`, `,`, `:`, boşluk) zaten ayırıcıdır.
func sozcukKarakteri(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_', c == '%', c == '\\', c == '#', c == '.':
		return true
	}
	return false
}

// bosSatirlariSik art arda gelen boş satırları teke indirir.
//
// Yorumlar silinince geride yalnız boşluk kalan satırlar kalıyor; bunlar
// çıktıyı şişiriyor. Boşluk dışındaki hiçbir şeye dokunulmaz (CSS'te
// seçici/değer arası boşluk anlam taşır).
func bosSatirlariSik(g []byte) []byte {
	cikti := make([]byte, 0, len(g))
	satir := make([]byte, 0, 256)
	bosMu := func(b []byte) bool {
		for _, c := range b {
			if c != ' ' && c != '\t' && c != '\r' {
				return false
			}
		}
		return true
	}
	flus := func() {
		cikti = append(cikti, satir...)
		cikti = append(cikti, '\n')
		satir = satir[:0]
	}
	for _, c := range g {
		if c == '\n' {
			if bosMu(satir) {
				// Zaten boş satır mı geliyor? Ardışık boşları at.
				if len(cikti) == 0 || (len(cikti) >= 1 && cikti[len(cikti)-1] == '\n') {
					satir = satir[:0]
					continue
				}
				cikti = append(cikti, '\n')
				satir = satir[:0]
				continue
			}
			flus()
			continue
		}
		satir = append(satir, c)
	}
	if len(satir) > 0 {
		cikti = append(cikti, satir...)
	}
	return cikti
}

// cssSunucu CSS dosyalarını yorumsuz sunar.
//
// Dönüşüm süreç başına BİR kez yapılır ve bellekte tutulur: istek yolunda
// tarama maliyeti olmaz. 32 CSS dosyası toplam ~1,2 MB; şişme riski yok.
type cssSunucu struct {
	fsys fs.FS

	mu  sync.RWMutex
	onb map[string]*cssGirdi
}

type cssGirdi struct {
	govde []byte
	etag  string
}

func newCSSSunucu(fsys fs.FS) *cssSunucu {
	return &cssSunucu{fsys: fsys, onb: map[string]*cssGirdi{}}
}

// al dosyayı (gerekirse dönüştürüp) önbellekten döndürür.
func (c *cssSunucu) al(ad string) (*cssGirdi, error) {
	c.mu.RLock()
	g, ok := c.onb[ad]
	c.mu.RUnlock()
	if ok {
		return g, nil
	}

	ham, err := fs.ReadFile(c.fsys, ad)
	if err != nil {
		return nil, err
	}
	temiz := cssYorumlariSil(ham)
	ozet := sha256.Sum256(temiz)
	g = &cssGirdi{govde: temiz, etag: `"` + hex.EncodeToString(ozet[:8]) + `"`}

	c.mu.Lock()
	c.onb[ad] = g
	c.mu.Unlock()
	return g, nil
}

// ServeHTTP /static/... isteğini karşılar. CSS ise yorumsuz sürümü,
// değilse dosyayı olduğu gibi sunar.
func (c *cssSunucu) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ad := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if !strings.HasSuffix(strings.ToLower(ad), ".css") {
		http.FileServer(http.FS(c.fsys)).ServeHTTP(w, r)
		return
	}

	g, err := c.al(ad)
	if err != nil {
		http.FileServer(http.FS(c.fsys)).ServeHTTP(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("ETag", g.etag)
	if eslesmeVarMi(r, g.etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(g.govde)))
	_, _ = w.Write(g.govde)
}

// eslesmeVarMi If-None-Match başlığı verilen ETag ile eşleşiyor mu.
func eslesmeVarMi(r *http.Request, etag string) bool {
	im := r.Header.Get("If-None-Match")
	if im == "" {
		return false
	}
	for _, parca := range strings.Split(im, ",") {
		p := strings.TrimSpace(parca)
		if p == etag || p == "*" || strings.TrimPrefix(p, "W/") == etag {
			return true
		}
	}
	return false
}
