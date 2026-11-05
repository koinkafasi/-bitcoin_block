package webui

// app.css hem public hem admin tarafından yüklenir; admin-only.css yalnızca
// panelde. Bu testler iki yönlü sızmayı engeller:
//
//   - app.css içinde, bütün virgüllü seçicileri yalnızca admin sınıfından
//     oluşan bir kural kalmamalı (ziyaretçiye boşuna inen CSS).
//   - admin-only.css içinde, public tarafta da kullanılan bir sınıfı tek
//     başına hedefleyen bir kural olmamalı (public'in stilini götürür).
//
// Bir sınıf "admin-only" sayılır: public şablonlarda, public JS/CSS'te ve
// Go'nun ürettiği HTML'de hiç geçmiyor ama admin şablonlarında geçiyorsa.
// Kararsız kalan her durum public sayılır — yani test yalnızca kesin
// durumlarda taşımaya izin verir.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	reCSSClass = regexp.MustCompile(`\.(-?[A-Za-z_][\w-]*)`)
	reCSSNote  = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reAdmMark  = regexp.MustCompile(`template "admin_head"|template "admin_side"` +
		`|template "admin_top"|template "admin_autosave"|class="admin-body`)
	rePubMark = regexp.MustCompile(`template "public_head"|template "site_header"` +
		`|template "site_footer"`)
	reGoClass   = regexp.MustCompile(`class=\\?"([^"\\]{1,160})`)
	reClassTok  = regexp.MustCompile(`^-?[A-Za-z_][\w-]*$`)
	reInnerSel  = regexp.MustCompile(`([^{}]+)\{[^{}]*\}`)
	reAssetRef  = regexp.MustCompile(`\{\{\s*asset\s+"([^"]+)"\s*\}\}`)
	reAssetLit  = regexp.MustCompile(`"([\w.\-/]+\.js)"`)
	adminAssets = filepath.Join("..", "httpapi", "adminassets.go")
)

// splitAssets, bir varlığın hangi tarafta yüklendiğini partials.html'den ve
// adminassets.go'daki sayfa başına betik listelerinden türetir. Elle liste
// tutmak çürüyor; burada tek doğruluk kaynağı kodun kendisi.
//
// Yön önemli: bir dosyayı yanlışlıkla PUBLIC saymak yalnızca iyileştirmeyi
// kaçırtır, ADMIN saymak ise public'in stilini götürebilir. Bu yüzden
// bilinmeyen her dosya ve iki tarafta da yüklenen app.js PUBLIC sayılır.
func splitAssets(t *testing.T, pubHead, admHead string) (pub, adm map[string]bool) {
	t.Helper()
	pub, adm = map[string]bool{}, map[string]bool{}
	for _, m := range reAssetRef.FindAllStringSubmatch(pubHead, -1) {
		pub[m[1]] = true
	}
	refs := reAssetRef.FindAllStringSubmatch(admHead, -1)
	for _, m := range refs {
		if !pub[m[1]] {
			adm[m[1]] = true
		}
	}
	// admin_head'deki {{range .Scripts}} sayfa başına betikleri buradan alır.
	b, err := os.ReadFile(adminAssets)
	require.NoError(t, err, "sayfa başına admin betik listesi okunamadı")
	for _, m := range reAssetLit.FindAllStringSubmatch(string(b), -1) {
		if !pub[m[1]] {
			adm[m[1]] = true
		}
	}
	require.NotEmpty(t, pub, "public_head hiç varlık yüklemiyor olamaz")
	require.NotEmpty(t, adm, "admin betik listesi boş çıktı")
	return pub, adm
}

type cssBlock struct {
	at   bool   // @media / @supports / @keyframes
	sel  string // yorumları ayıklanmış seçici
	body string // tam blok metni
}

// splitCSS üst düzey blokları ayırır. Parantez eşlemesi yapar; iç içe
// @media bloklarını tek blok olarak döndürür.
// splitCSS CSS'i kural bloklarına ayırır.
//
// DİKKAT: yorumlar ÖNCE temizlenir. Aksi hâlde bir yorumun İÇİNDEKİ `{`
// karakteri (ör. "kapsamsız `.media-item img{height:100%}` kuralı") blok
// sınırlarını kaydırıyor ve takip eden tüm kurallar yanlış ayrıştırılıyordu.
// Belirti: yorum metni "seçici" sanılıp `TestAdminOnlyCSSHasNoPublicRules`
// yanlış kırmızı veriyordu. Yorumlar analizde gerekmediği için baştan
// silmek güvenli.
func splitCSS(css string) []cssBlock {
	css = reCSSNote.ReplaceAllString(css, " ")

	var out []cssBlock
	for pos := 0; pos < len(css); {
		br := strings.IndexByte(css[pos:], '{')
		if br < 0 {
			break
		}
		br += pos
		depth, j := 1, br+1
		for ; j < len(css) && depth > 0; j++ {
			switch css[j] {
			case '{':
				depth++
			case '}':
				depth--
			}
		}
		sel := strings.TrimSpace(css[pos:br])
		out = append(out, cssBlock{
			at:   strings.HasPrefix(sel, "@"),
			sel:  sel,
			body: css[pos:j],
		})
		pos = j
	}
	return out
}

func isNameByte(b byte) bool {
	return b == '-' || b == '_' ||
		b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// usedIn, name'i sınıf-adı sınırlarıyla arar: "card" ararken "card-pad"
// eşleşmez.
func usedIn(name, blob string) bool {
	for i := 0; i+len(name) <= len(blob); {
		j := strings.Index(blob[i:], name)
		if j < 0 {
			return false
		}
		p := i + j
		before := p == 0 || !isNameByte(blob[p-1])
		after := p+len(name) == len(blob) || !isNameByte(blob[p+len(name)])
		if before && after {
			return true
		}
		i = p + 1
	}
	return false
}

// classPools public ve admin işaretleme havuzlarını kurar.
func classPools(t *testing.T) (pub, adm string) {
	t.Helper()
	var pubParts, admParts []string

	partials, err := os.ReadFile(filepath.Join("templates", "partials.html"))
	require.NoError(t, err)
	cut := strings.Index(string(partials), `{{define "admin_head"}}`)
	require.Greater(t, cut, 0, `partials.html içinde {{define "admin_head"}} bulunamadı`)
	pubHead, admHead := string(partials)[:cut], string(partials)[cut:]
	pubParts = append(pubParts, pubHead)
	admParts = append(admParts, admHead)

	tpls, err := filepath.Glob(filepath.Join("templates", "*.html"))
	require.NoError(t, err)
	for _, p := range tpls {
		if filepath.Base(p) == "partials.html" {
			continue
		}
		b, err := os.ReadFile(p)
		require.NoError(t, err)
		s := string(b)
		isAdm, isPub := reAdmMark.MatchString(s), rePubMark.MatchString(s)
		switch {
		case isAdm && !isPub:
			admParts = append(admParts, s)
		case !isAdm && !isPub && strings.HasSuffix(p, "_partial.html"):
			// Tek bir admin sayfası tarafından dahil edilen parça.
			admParts = append(admParts, s)
		default:
			// public, ya da ikisi de / belirsiz -> public (muhafazakâr taraf).
			pubParts = append(pubParts, s)
		}
	}

	kits, err := filepath.Glob(filepath.Join("templkit", "*.templ"))
	require.NoError(t, err)
	for _, p := range kits {
		b, err := os.ReadFile(p)
		require.NoError(t, err)
		admParts = append(admParts, string(b))
	}

	// JS dosyaları da sınıf ekliyor (ör. ses_player.js dalga çubuklarını,
	// ai_editor.js çekmece durumlarını JS'te üretiyor). public.css de public
	// kanıtı sayılır. Admin CSS dosyaları (admin-theme/admin-shadcn) havuza
	// GIRMEZ: bir sınıfın orada stillenmiş olması işaretlemede kullanıldığının
	// kanıtı değil — ölü CSS de olabilir.
	pubAsset, admAsset := splitAssets(t, pubHead, admHead)
	assets, err := filepath.Glob(filepath.Join("static", "*.js"))
	require.NoError(t, err)
	if pc, err := os.ReadFile(filepath.Join("static", "public.css")); err == nil {
		pubParts = append(pubParts, string(pc))
	}
	for _, p := range assets {
		b, err := os.ReadFile(p)
		require.NoError(t, err)
		if admAsset[filepath.Base(p)] && !pubAsset[filepath.Base(p)] {
			admParts = append(admParts, string(b))
		} else {
			pubParts = append(pubParts, string(b))
		}
	}

	// Go kaynağı da HTML üretiyor: tiptap.go makale gövdesini, gazete_fill.go
	// gazete sayfasını, puan_serit.go puan şeridini renderlıyor ve bunlar
	// PUBLIC sayfalarda görünüyor. Bu yüzden .go dosyalarındaki class="..."
	// jetonları public sayılır. *_templ.go hariç: onlar admin templ
	// dosyalarından üretilmiş kopyalar, kaynakları zaten admin havuzunda.
	seen := map[string]bool{}
	err = filepath.Walk(filepath.Join("..", ".."), func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".go") {
			return nil //nolint:nilerr // okunamayan dosya testi düşürmesin
		}
		if strings.HasSuffix(p, "_test.go") || strings.HasSuffix(p, "_templ.go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		for _, m := range reGoClass.FindAllStringSubmatch(string(b), -1) {
			for _, tok := range strings.Fields(strings.ReplaceAll(m[1], "+", " ")) {
				if reClassTok.MatchString(tok) {
					seen[tok] = true
				}
			}
		}
		return nil
	})
	require.NoError(t, err)
	toks := make([]string, 0, len(seen))
	for k := range seen {
		toks = append(toks, k)
	}
	pubParts = append(pubParts, strings.Join(toks, " "))

	return strings.Join(pubParts, "\n"), strings.Join(admParts, "\n")
}

// adminOnlyClasses verilen CSS'teki sınıflardan yalnızca admin'de
// kullanılanları döndürür.
func adminOnlyClasses(css, pub, adm string) map[string]bool {
	out := map[string]bool{}
	for _, m := range reCSSClass.FindAllStringSubmatch(reCSSNote.ReplaceAllString(css, " "), -1) {
		c := m[1]
		if _, ok := out[c]; ok {
			continue
		}
		out[c] = !usedIn(c, pub) && usedIn(c, adm)
	}
	return out
}

// selectorIsAdminOnly: seçici grubunun HER virgüllü parçası en az bir
// admin-only sınıf içeriyor mu. ".ai-drawer .btn" evet; ".ai-drawer, .btn"
// hayır — ikincisi .btn'in public stilini taşır.
func selectorIsAdminOnly(sel string, adminOnly map[string]bool) bool {
	parts := strings.Split(sel, ",")
	n := 0
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		n++
		hit := false
		for _, m := range reCSSClass.FindAllStringSubmatch(p, -1) {
			if adminOnly[m[1]] {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return n > 0
}

// adKapsamliAtKurali: @keyframes gibi SEÇİCİ değil İSİM tanımlayan
// at-kuralları. İç blokları seçici değil adımdır (0%, 100%), bu yüzden
// seçici kuralına göre değerlendirilemezler.
func adKapsamliAtKurali(sel string) bool {
	l := strings.ToLower(strings.TrimSpace(sel))
	for _, p := range []string{"@keyframes", "@-webkit-keyframes", "@font-face", "@property", "@counter-style"} {
		if strings.HasPrefix(l, p) {
			return true
		}
	}
	return false
}

func blockIsAdminOnly(b cssBlock, adminOnly map[string]bool) bool {
	if !b.at {
		return selectorIsAdminOnly(b.sel, adminOnly)
	}
	if adKapsamliAtKurali(b.sel) {
		// Taşıma kararına girmez: ne app.css'ten çıkarılması önerilir
		// ne de admin-only.css'te sorun sayılır. Adının public'ten
		// referans edilmediği ayrı testte doğrulanır.
		return false
	}
	inner := reInnerSel.FindAllStringSubmatch(
		reCSSNote.ReplaceAllString(b.body[strings.IndexByte(b.body, '{')+1:], " "), -1)
	if len(inner) == 0 {
		return false
	}
	for _, m := range inner {
		if !selectorIsAdminOnly(strings.TrimSpace(m[1]), adminOnly) {
			return false
		}
	}
	return true
}

func TestAppCSSHasNoAdminOnlyRules(t *testing.T) {
	pub, adm := classPools(t)
	raw, err := os.ReadFile(filepath.Join("static", "app.css"))
	require.NoError(t, err)
	css := string(raw)
	adminOnly := adminOnlyClasses(css, pub, adm)

	var leaked []string
	for _, b := range splitCSS(css) {
		if blockIsAdminOnly(b, adminOnly) {
			leaked = append(leaked, b.sel)
		}
	}
	require.Emptyf(t, leaked,
		"app.css her public sayfada iniyor; şu %d kural yalnızca panelde "+
			"kullanılıyor, static/admin-only.css'e taşınmalı:\n%s",
		len(leaked), strings.Join(leaked, "\n"))
}

func TestAdminOnlyCSSHasNoPublicRules(t *testing.T) {
	pub, adm := classPools(t)
	appCSS, err := os.ReadFile(filepath.Join("static", "app.css"))
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join("static", "admin-only.css"))
	require.NoError(t, err)

	// admin-only sınıf kümesi iki dosyanın birleşiminden çıkarılır; yoksa
	// yalnızca admin-only.css'te görünen sınıflar değerlendirilemez.
	adminOnly := adminOnlyClasses(string(appCSS)+"\n"+string(raw), pub, adm)

	var leaked []string
	for _, b := range splitCSS(string(raw)) {
		if b.at && adKapsamliAtKurali(b.sel) {
			continue // ad tanımlar, seçici değil; aşağıda ayrıca denetleniyor
		}
		if !blockIsAdminOnly(b, adminOnly) {
			leaked = append(leaked, b.sel)
		}
	}
	require.Emptyf(t, leaked,
		"admin-only.css yalnızca panelde yükleniyor; şu %d kural public "+
			"tarafta da kullanılan bir sınıfı hedefliyor ve public'in "+
			"stilini götürür:\n%s", len(leaked), strings.Join(leaked, "\n"))
}

// Yükleme sırası davranışı belirler: taşınan kurallar app.css'ten sonra,
// admin-theme.css'ten önce gelmeli. Aksi hâlde aynı özgüllükteki
// çakışmalarda kazanan değişir.
func TestAdminOnlyCSSLoadOrder(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("templates", "partials.html"))
	require.NoError(t, err)
	s := string(raw)

	cut := strings.Index(s, `{{define "admin_head"}}`)
	require.Greater(t, cut, 0)
	pubHead, admHead := s[:cut], s[cut:]

	require.NotContains(t, pubHead, "admin-only.css",
		"admin-only.css public_head'de yüklenmemeli")

	iApp := strings.Index(admHead, `{{asset "app.css"}}`)
	iOnly := strings.Index(admHead, `{{asset "admin-only.css"}}`)
	iTheme := strings.Index(admHead, `{{asset "admin-theme.css"}}`)
	require.Greater(t, iApp, -1, "admin_head app.css yüklemiyor")
	require.Greater(t, iOnly, -1, "admin_head admin-only.css yüklemiyor")
	require.Greater(t, iTheme, -1, "admin_head admin-theme.css yüklemiyor")
	require.Less(t, iApp, iOnly, "admin-only.css app.css'ten SONRA gelmeli")
	require.Less(t, iOnly, iTheme, "admin-only.css admin-theme.css'ten ÖNCE gelmeli")
}

// admin-pano.css panelin TAMAMINI giydiriyor (Faz 151): kabuk, kart, KPI,
// tablo, form, sekme. Üç şeyi kilitliyoruz:
//  1. Public head'e sızmamalı (yoksa ön yüz panel görünümüne döner).
//  2. admin-shadcn.css'ten SONRA yüklenmeli (token'ları devralır).
//  3. Marka rengine (--accent) dokunmamalı; marka kiti her zaman kazanmalı.
func TestAdminPanoSkin(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("templates", "partials.html"))
	require.NoError(t, err)
	s := string(raw)

	cut := strings.Index(s, `{{define "admin_head"}}`)
	require.Greater(t, cut, 0)
	pubHead, admHead := s[:cut], s[cut:]

	require.NotContains(t, pubHead, "admin-pano.css",
		"admin-pano.css public_head'de yüklenmemeli — ön yüzü panelleştirir")
	require.Contains(t, admHead, `{{asset "admin-pano.css"}}`,
		"admin_head admin-pano.css yüklemiyor")

	iShadcn := strings.Index(admHead, `{{asset "admin-shadcn.css"}}`)
	iPano := strings.Index(admHead, `{{asset "admin-pano.css"}}`)
	require.Less(t, iShadcn, iPano,
		"admin-pano.css admin-shadcn.css'ten SONRA gelmeli (token'ları devralır)")

	// admin-theme (marka kiti) sonrasında olması gerekmez: pano katmanı
	// marka rengini değiştirmiyor, yalnız nötr paleti ve geometriyi.
	skin, err := os.ReadFile(filepath.Join("static", "admin-pano.css"))
	require.NoError(t, err)
	body := string(skin)
	require.NotContains(t, body, "--accent:",
		"admin-pano.css marka rengini (--accent) tanımlamamalı")
	require.NotContains(t, body, "--sh-sidebar: hsl",
		"shadcn paletini hsl() ile değil, sıcak nötr tonlarla eşlemeli")
}

// Ray katlama her genişlikte İKİ YÖNDE de çalışmalı.
//
// Regresyon: medya sorgusunda `.admin-side { width: 84px }` vardı. Aynı
// özgüllükte olduğu için `html.admin-rail .admin-side` kuralıyla çakışıyor
// ve menü 1280 px altında ASLA açılamıyordu — düğme "bozuk" görünüyordu.
// Ölçüm: 1500 px'te 252→84, 1280/1200/1024'te 84→84 (hiç değişmiyordu).
//
// Faz 159: Pacely'nin ray genişliği ölçüldü — 88px (84 değil). Dar mod
// değeri güncellendi; katlama mantığı aynı.
func TestAdminRailHerGenislikteCalisir(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("static", "admin-pano.css"))
	require.NoError(t, err)
	css := string(raw)

	// Genişlik kararı tek yerde olmalı: sınıfa bağlı kural.
	require.Contains(t, css, "html.admin-rail .admin-side { width: 88px; }",
		"dar mod kuralı html.admin-rail üzerinden tanımlanmalı")

	// Hiçbir medya sorgusu menü genişliğini koşulsuz sabitlememeli.
	// (`.admin-side` seçicisini içeren ama .admin-rail geçmeyen width kuralı
	// katlamayı tek yönlü kılar.)
	for _, blok := range medyaBloklari(css) {
		for _, satir := range strings.Split(blok, "\n") {
			s := strings.TrimSpace(satir)
			if !strings.Contains(s, "width:") {
				continue
			}
			if !strings.Contains(s, "admin-side") {
				continue
			}
			require.Containsf(t, s, "admin-rail",
				"medya sorgusunda menü genişliği koşulsuz sabitlenmiş; "+
					"katlama tek yönlü kalır: %q", s)
		}
	}

	// Ön boyama betiği olmalı: yoksa her sayfa geçişinde menü önce geniş
	// görünüp daralıyor (FOUC) ve kullanıcı zıplama görüyor.
	head, err := os.ReadFile(filepath.Join("templates", "partials.html"))
	require.NoError(t, err)
	s := string(head)
	cut := strings.Index(s, `{{define "admin_head"}}`)
	require.Greater(t, cut, 0)
	require.Contains(t, s[cut:], "pn:rail",
		"admin_head ray tercihini ilk boyamadan önce uygulamalı")
}

// medyaBloklari @media gövdelerini kaba ama yeterli biçimde ayıklar.
// Test yalnız "width" satırlarını aradığı için iç içe blok çözümlemesi
// gerekmiyor: ilk { ile son } arası yeterli.
func medyaBloklari(css string) []string {
	var out []string
	for {
		i := strings.Index(css, "@media")
		if i < 0 {
			return out
		}
		ac := strings.Index(css[i:], "{")
		if ac < 0 {
			return out
		}
		ac += i
		derinlik, j := 0, ac
		for ; j < len(css); j++ {
			switch css[j] {
			case '{':
				derinlik++
			case '}':
				derinlik--
				if derinlik == 0 {
					goto bulundu
				}
			}
		}
	bulundu:
		out = append(out, css[ac+1:j])
		css = css[j:]
	}
}

// admin-only.css'teki @keyframes gibi ad tanımları yalnız panelde yüklenir.
// Public bir kural o ada atıf yaparsa animasyon/font public'te bulunamaz ve
// sessizce çalışmaz. Bu test o sessiz kırılmayı yakalar.
func TestAdminOnlyAtKuralAdlariPublictenKullanilmiyor(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("static", "admin-only.css"))
	require.NoError(t, err)

	var adlar []string
	for _, b := range splitCSS(string(raw)) {
		if !b.at || !adKapsamliAtKurali(b.sel) {
			continue
		}
		f := strings.Fields(b.sel)
		if len(f) >= 2 {
			adlar = append(adlar, f[1])
		}
	}
	if len(adlar) == 0 {
		return
	}

	var public []string
	for _, n := range []string{"app.css", "public.css"} {
		b, err := os.ReadFile(filepath.Join("static", n))
		require.NoError(t, err)
		public = append(public, string(b))
	}
	blob := strings.Join(public, "\n")

	for _, ad := range adlar {
		require.Falsef(t, usedIn(ad, blob),
			"admin-only.css %q tanımlıyor ama public CSS ona atıf yapıyor; "+
				"public tarafta bulunamaz ve sessizce çalışmaz", ad)
	}
}
