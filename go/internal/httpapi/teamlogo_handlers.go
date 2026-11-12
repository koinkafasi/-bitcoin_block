package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/platform"
)

func (s *Server) handleTeamLogo(w http.ResponseWriter, r *http.Request) {
	// Go yönlendiricisi joker karakterin ortasına sabit metin koymuyor
	// (`{key}.png` geçersiz); uzantı burada ayrılır.
	key := strings.TrimSuffix(r.PathValue("key"), ".png")
	if !platform.ValidTeamLogoKey(key) {
		http.NotFound(w, r)
		return
	}
	f, st, err := platform.OpenTeamLogo(key)
	if err != nil {
		// Logo yoksa çağıran şablon zaten takım adını gösteriyor; 404 doğru
		// cevap. Kırık resim simgesi çıkmaması için şablon `onerror` kullanır.
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", teamLogoCacheHeader)
	// ETag son değişiklik + boyuttan üretilir; logo içeriği değişmez.
	etag := `"` + strconv.FormatInt(st.ModTime().Unix(), 36) + `-` + strconv.FormatInt(st.Size(), 36) + `"`
	w.Header().Set("ETag", etag)
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	// ServeContent Range ve HEAD isteklerini doğru ele alır.
	http.ServeContent(w, r, key+".png", st.ModTime(), f)
}

// teamLogoCacheHeader logo yanıtının önbellek politikası (testler de okur).
const teamLogoCacheHeader = "public, max-age=31536000, immutable"
