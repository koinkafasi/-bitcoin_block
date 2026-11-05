package httpapi

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Admin görünüm / marka kiti (Faz A) ----
//
// WiseCP "Appearance" tarzı: logo'dan renk çıkarma (istemci canvas),
// ana/ikincil/metin/navbar renkleri, font seçimi, dikey/yatay navbar.
// Saklama: settings('admin_gorunum','secim') JSON.

// gorunumView panel görünüm ayarları.
type gorunumView struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	TextColor string `json:"text"`
	Navbar    string `json:"navbar"` // dikey | yatay
	Font      string `json:"font"`
	Logo      string `json:"logo"`
	Icon      string `json:"icon"`
	LogoMedia int64  `json:"logo_media"`
	IconMedia int64  `json:"icon_media"`
}

var (
	gorunumHexRe  = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)
	gorunumFontRe = regexp.MustCompile(`^[A-Za-z0-9 ',-]{1,60}$`)
)

// sanitize renk/font alanlarını güvenli hale getirir.
func (g *gorunumView) sanitize() {
	if !gorunumHexRe.MatchString(g.Primary) {
		g.Primary = ""
	}
	if !gorunumHexRe.MatchString(g.Secondary) {
		g.Secondary = ""
	}
	if !gorunumHexRe.MatchString(g.TextColor) {
		g.TextColor = ""
	}
	if !gorunumFontRe.MatchString(g.Font) {
		g.Font = ""
	}
	if g.Navbar != "yatay" {
		g.Navbar = "dikey"
	}
	g.Logo = strings.TrimSpace(g.Logo)
	g.Icon = strings.TrimSpace(g.Icon)
}

// gorunumLoad ayarları okur (yoksa boş/sıfırlanmış görünüm).
func gorunumLoad(ctx context.Context, pool repo.DB) gorunumView {
	raw, err := repo.GetSetting(ctx, pool, "admin_gorunum", "secim")
	var g gorunumView
	if err == nil {
		_ = json.Unmarshal(raw, &g)
	}
	g.sanitize()
	return g
}

// CSS panel için :root override üretir.
func (g gorunumView) CSS() template.CSS {
	var b strings.Builder
	b.WriteString(":root{")
	if g.Primary != "" {
		b.WriteString("--accent:" + g.Primary + ";--accent-2:" + g.Primary + ";")
		// Aynı değer ikinci bir adla: public.css yükleyen sayfalar (Sayfa
		// Tasarımcısı) `--accent`'i kendi temasına çeviriyor; orada bu
		// değerden geri yüklenir (Faz 182).
		b.WriteString("--tb-accent:" + g.Primary + ";--tb-accent-2:" + g.Primary + ";")
		// Dolgu tonu: üstünde beyaz yazı 4.5'i geçsin diye türetilir.
		// Marka rengi (--accent) DEĞİŞMEZ (Faz 180, A-7).
		if f := accentFill(g.Primary); f != "" {
			b.WriteString("--accent-fill:" + f + ";")
		}
		// Metin tonu: açık zeminde 4.5 (Faz 180).
		// Açık tema metin tonu. Adı `-acik`: düz `--accent-text` yazmak
		// koyu temada da geçerli oluyordu (ölçüldü: 2.9 kontrast).
		if t := accentTextFill(g.Primary); t != "" {
			b.WriteString("--accent-text-acik:" + t + ";")
		}
		// Koyu tema metin tonu (Faz 182).
		if t := accentTextKoyu(g.Primary); t != "" {
			b.WriteString("--accent-text-dark:" + t + ";")
		}
	}
	if g.Secondary != "" {
		b.WriteString("--brand:" + g.Secondary + ";")
	}
	if g.TextColor != "" {
		// METIN RENGI TEK BIR DEGER: iki temaya da uygulanirsa koyu temada
		// siyah metin okunmaz (olculdu: kontrast 1.2). Acik tema secimi
		// oldugu gibi, koyu tema icin acilmis tonu yazilir; hangisinin
		// kullanilacagina admin-shadcn karar verir (Faz 182).
		b.WriteString("--text-kullanici:" + g.TextColor + ";")
		if t := koyuTemaMetin(g.TextColor); t != "" {
			b.WriteString("--text-dark:" + t + ";")
		}
	}
	if g.Font != "" {
		b.WriteString("--font-ui:'" + strings.ReplaceAll(g.Font, "'", "") + "',system-ui,sans-serif;")
	}
	b.WriteString("}")
	if g.Font != "" {
		b.WriteString("body{font-family:var(--font-ui)}")
	}
	if g.Navbar == "yatay" {
		b.WriteString("body.admin-hnav .admin-wrap{flex-direction:column}body.admin-hnav .admin-side{position:static;width:auto;height:auto;border-right:0;border-bottom:1px solid var(--border);display:flex;gap:4px;align-items:center;overflow-x:auto;padding:8px 12px}body.admin-hnav .admin-side .brand{margin-right:12px}body.admin-hnav .admin-side .menu{display:flex;gap:2px;margin:0}body.admin-hnav .admin-side .menu-h{display:none}body.admin-hnav .admin-side .menu a{white-space:nowrap;padding:7px 10px}")
	}
	return template.CSS(b.String())
}

// FontList font seçimi için hazır liste (WiseCP tarzı).
var gorunumFonts = []string{
	"Inter", "Poppins", "Roboto", "Open Sans", "Montserrat", "Lato", "Nunito",
	"Nunito Sans", "Work Sans", "DM Sans", "Plus Jakarta Sans", "Manrope",
	"Figtree", "Outfit", "Sora", "Urbanist", "Rubik", "Source Sans 3",
	"IBM Plex Sans", "Mulish", "Karla", "Titillium Web",
}

// handleGorunumAdmin GET /admin/gorunum — görünüm ayarları sayfası.
func (s *Server) handleGorunumAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.baseView("tr", "/admin", "Görünüm")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	g := gorunumLoad(r.Context(), s.pool)
	d.Gorunum = &g
	d.GorunumFonts = gorunumFonts
	s.render(w, r, "admin_gorunum", d)
}

// handleGorunumSave POST /admin/gorunum — ayarları kaydeder (logo/icon yükleme dahil).
func (s *Server) handleGorunumSave(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		s.flashRedirect(w, r, "/admin/gorunum", "error", "Form okunamadı.")
		return
	}
	ctx := r.Context()
	g := gorunumLoad(ctx, s.pool)
	g.Primary = strings.TrimSpace(r.FormValue("primary"))
	g.Secondary = strings.TrimSpace(r.FormValue("secondary"))
	g.TextColor = strings.TrimSpace(r.FormValue("text"))
	g.Navbar = strings.TrimSpace(r.FormValue("navbar"))
	g.Font = strings.TrimSpace(r.FormValue("font"))
	if v := strings.TrimSpace(r.FormValue("logo_url")); v != "" {
		g.Logo = v
	}
	if v := strings.TrimSpace(r.FormValue("icon_url")); v != "" {
		g.Icon = v
	}
	// Logo/icon dosya yüklemesi → medya deposu.
	if fh, _, err := r.FormFile("logo"); err == nil {
		defer fh.Close()
		if data, err := io.ReadAll(fh); err == nil {
			if path, err := platform.SaveMedia(data, "logo.png"); err == nil {
				g.Logo = "/" + path
			}
		}
	}
	if fh, _, err := r.FormFile("icon"); err == nil {
		defer fh.Close()
		if data, err := io.ReadAll(fh); err == nil {
			if path, err := platform.SaveMedia(data, "icon.png"); err == nil {
				g.Icon = "/" + path
			}
		}
	}
	g.sanitize()
	raw, _ := json.Marshal(g)
	if err := repo.SetSetting(ctx, s.pool, "admin_gorunum", "secim", raw); err != nil {
		s.flashRedirect(w, r, "/admin/gorunum", "error", "Kaydedilemedi.")
		return
	}
	s.audit(r, "admin_gorunum", g.Navbar+"/"+g.Font)
	if r.FormValue("sifirla") == "1" {
		_ = repo.SetSetting(ctx, s.pool, "admin_gorunum", "secim", []byte("{}"))
		s.flashRedirect(w, r, "/admin/gorunum", "info", "Görünüm varsayılana döndürüldü.")
		return
	}
	s.flashRedirect(w, r, "/admin/gorunum", "info", "Görünüm kaydedildi.")
}
