package httpapi

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

const (
	// mmPt bir milimetrenin punto karşılığı.
	mmPt = 72.0 / 25.4
	// gazeteMatbaaDosya depolamadaki matbaa dosyası adı.
	gazeteMatbaaDosya = "gazete-matbaa.pdf"
)

type gazeteMatbaaAyar struct {
	PDFX    string // X-1a | X-3 | X-4
	ICCYolu string // çıktı amacı profili; boş ise Ghostscript'in CMYK profili
	Kosul   string // OutputConditionIdentifier, ör. "CGATS TR 001"
}

func (s *Server) gazeteMatbaaAyari(ctx context.Context) gazeteMatbaaAyar {
	a := gazeteMatbaaAyar{PDFX: "X-3", Kosul: "CGATS TR 001"}
	oku := func(anahtar string) string {
		v, err := repo.GetSetting(ctx, s.pool, "gazete", anahtar)
		if err != nil {
			return ""
		}
		return strings.Trim(strings.TrimSpace(string(v)), `"`)
	}
	if v := oku("matbaa_pdfx"); v != "" {
		a.PDFX = v
	}
	if v := oku("matbaa_icc"); v != "" {
		a.ICCYolu = v
	}
	if v := oku("matbaa_kosul"); v != "" {
		a.Kosul = v
	}
	if a.ICCYolu == "" {
		a.ICCYolu = gazeteVarsayilanCMYKProfili()
	}
	return a
}

// gazeteVarsayilanCMYKProfili Ghostscript ile gelen genel CMYK profilini bulur.
// Matbaanın kendi profili ayarlanana kadar geçici çözümdür.
func gazeteVarsayilanCMYKProfili() string {
	kaliplar := []string{
		"/usr/share/ghostscript/*/iccprofiles/default_cmyk.icc",
		"/usr/share/color/icc/*cmyk*.icc",
	}
	for _, k := range kaliplar {
		if m, err := filepath.Glob(k); err == nil && len(m) > 0 {
			return m[len(m)-1]
		}
	}
	return ""
}

// gazetePDFXAnahtari Ghostscript'in PDFX sürüm anahtarı.
func gazetePDFXAnahtari(surum string) string {
	switch strings.ToUpper(strings.TrimSpace(surum)) {
	case "X-1A", "X1A", "PDF/X-1A":
		return "/PDF/X-1a:2001"
	case "X-4", "X4", "PDF/X-4":
		return "/PDF/X-4"
	default:
		return "/PDF/X-3:2002"
	}
}

type gazeteMatbaaKutu struct {
	Isaret           float64 // isaret alani (pt)
	Tasma            float64 // tasma payi (pt)
	TrimW, TrimH     float64 // kesim olcusu (pt)
	MediaW, MediaH   float64 // sayfa kutusu (pt)
	TrimX0, TrimY0   float64 // kesim kutusunun sol-alt kosesi (pt)
	BleedX0, BleedY0 float64
	BleedW, BleedH   float64
}

// gazeteIsaretAlaniMm kesim isaretleri icin ayrilan kenar payi.
const gazeteIsaretAlaniMm = 8.0

func gazeteMatbaaGeometrisi(trimWmm, trimHmm, tasmaMm float64) gazeteMatbaaKutu {
	m := gazeteIsaretAlaniMm * mmPt
	b := tasmaMm * mmPt
	tw, th := trimWmm*mmPt, trimHmm*mmPt
	return gazeteMatbaaKutu{
		Isaret: m, Tasma: b, TrimW: tw, TrimH: th,
		MediaW: tw + 2*b + 2*m, MediaH: th + 2*b + 2*m,
		TrimX0: m + b, TrimY0: m + b,
		BleedX0: m, BleedY0: m,
		BleedW: tw + 2*b, BleedH: th + 2*b,
	}
}

func gazetePDFXTanimi(a gazeteMatbaaAyar, trimW, trimH, tasmaMm float64) string {
	g := gazeteMatbaaGeometrisi(trimW, trimH, tasmaMm)

	var sb strings.Builder
	sb.WriteString("%!\n")
	// Surum dosyanin kendisine yazilmali; yoksa ayar gorunur ama ciktiya
	// yansimaz ve matbaa istediginden baska bir standart alir.
	fmt.Fprintf(&sb, "[ /GTS_PDFXVersion (%s) /DOCINFO pdfmark\n",
		gazetePSKacir(strings.TrimPrefix(gazetePDFXAnahtari(a.PDFX), "/")))
	if a.ICCYolu != "" {
		fmt.Fprintf(&sb, "/ICCProfile (%s) def\n", gazetePSKacir(a.ICCYolu))
		sb.WriteString(`[/_objdef {icc_PDFX} /type /stream /OBJ pdfmark
[{icc_PDFX} <</N 4>> /PUT pdfmark
[{icc_PDFX} ICCProfile (r) file /PUT pdfmark
[/_objdef {OutputIntent_PDFX} /type /dict /OBJ pdfmark
`)
		fmt.Fprintf(&sb, `[{OutputIntent_PDFX} <<
  /Type /OutputIntent /S /GTS_PDFX
  /OutputCondition (%s)
  /OutputConditionIdentifier (%s)
  /RegistryName (http://www.color.org)
  /DestOutputProfile {icc_PDFX}
>> /PUT pdfmark
`, gazetePSKacir(a.Kosul), gazetePSKacir(a.Kosul))
		sb.WriteString("[{Catalog} <</OutputIntents [ {OutputIntent_PDFX} ]>> /PUT pdfmark\n")
	}
	// Her sayfada kesim ve tasma kutularini yaz.
	fmt.Fprintf(&sb, `<< /BeginPage {
  [ {ThisPage} <<
     /TrimBox [%.4f %.4f %.4f %.4f]
     /BleedBox [%.4f %.4f %.4f %.4f]
  >> /PUT pdfmark
} bind >> setpagedevice
`,
		g.TrimX0, g.TrimY0, g.TrimX0+g.TrimW, g.TrimY0+g.TrimH,
		g.BleedX0, g.BleedY0, g.BleedX0+g.BleedW, g.BleedY0+g.BleedH)
	return sb.String()
}

// gazetePSKacir PostScript dizgisi icin parantez ve ters bolu kacirir.
// Yol ve kosul adi disaridan gelebiliyor; kacrilmazsa tanim bozulur.
func gazetePSKacir(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return r.Replace(s)
}

// gazeteMatbaaRapor preflight sonucu.
type gazeteMatbaaRapor struct {
	Anahtar     string   `json:"anahtar"`
	SayfaBoyu   string   `json:"sayfaBoyu"`
	Sayfa       int      `json:"sayfa"`
	CiktiAmaci  bool     `json:"ciktiAmaci"`
	TrimBox     bool     `json:"trimBox"`
	BleedBox    bool     `json:"bleedBox"`
	GomulmeyenF []string `json:"gomulmeyenFontlar"`
	Uyarilar    []string `json:"uyarilar"`
	PDFX        string   `json:"pdfx"`
	ICC         string   `json:"icc"`
}

// Hazir rapordaki uyarilar bos ise dosya matbaaya gonderilebilir.
func (r gazeteMatbaaRapor) Hazir() bool { return len(r.Uyarilar) == 0 }

// gazeteMatbaaUret matbaa dosyasini uretir, depolar ve preflight raporu doner.
func (s *Server) gazeteMatbaaUret(ctx context.Context, siteID, issueID int64) (gazeteMatbaaRapor, error) {
	var rap gazeteMatbaaRapor

	iss, err := repo.GetGazeteIssue(ctx, s.pool, issueID)
	if err != nil {
		return rap, err
	}
	belge := s.gazeteSayiBelgesi(ctx, iss)
	boyut := gazeteBoyut(belge.Sayfa.Boyut)
	trimW := gazeteMmSayi(boyut.W)
	trimH := gazeteMmSayi(boyut.H)
	tasma := belge.Sayfa.TasmaPayi

	htmlDoc, err := s.gazeteMatbaaHTML(ctx, siteID, iss, belge, boyut, tasma)
	if err != nil {
		return rap, err
	}
	chrome := findChrome()
	if chrome == "" {
		return rap, fmt.Errorf("Chromium bulunamadı (GAZETE_CHROME ayarlayın)")
	}
	gsYol, err := exec.LookPath("gs")
	if err != nil {
		return rap, fmt.Errorf("Ghostscript bulunamadı; matbaa dosyası üretilemez")
	}

	dir, err := os.MkdirTemp("", "gazete-matbaa-*")
	if err != nil {
		return rap, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	htmlPath := filepath.Join(dir, "matbaa.html")
	if err := os.WriteFile(htmlPath, []byte(gazeteBaskiMedyaMutlak(htmlDoc)), 0o644); err != nil {
		return rap, err
	}
	hamPDF := filepath.Join(dir, "ham.pdf")
	cmd := exec.CommandContext(ctx, chrome, // #nosec G702 -- sabit bayraklar + MkdirTemp yolu (G703 notuyla aynı gerekçe)
		"--headless=new", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage",
		"--no-pdf-header-footer", "--virtual-time-budget=20000",
		"--print-to-pdf="+hamPDF, "file://"+htmlPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return rap, fmt.Errorf("baskı PDF'i üretilemedi: %w (%s)", err, firstLine(string(out), 200))
	}

	ayar := s.gazeteMatbaaAyari(ctx)
	rap.PDFX = ayar.PDFX
	rap.ICC = ayar.ICCYolu

	defPath := filepath.Join(dir, "pdfx_def.ps")
	if err := os.WriteFile(defPath,
		[]byte(gazetePDFXTanimi(ayar, trimW, trimH, tasma)), 0o644); err != nil {
		return rap, err
	}

	matbaaPDF := filepath.Join(dir, gazeteMatbaaDosya)
	args := []string{
		"-dBATCH", "-dNOPAUSE", "-dSAFER", "-dQUIET",
		"-sDEVICE=pdfwrite",
		"-dPDFX", "-dPDFSETTINGS=/prepress",
		"-dCompatibilityLevel=1.6",
		"-sColorConversionStrategy=CMYK",
		"-dProcessColorModel=/DeviceCMYK",
		"-dAutoRotatePages=/None",
		"-sOutputFile=" + matbaaPDF,
	}
	if ayar.ICCYolu != "" {
		args = append(args, "-sOutputICCProfile="+ayar.ICCYolu)
	}
	args = append(args, defPath, hamPDF)

	// #nosec G702 -- komut adı sabit (gsYol), argv sabit bayraklar +
	// MkdirTemp kaynaklı yollar; dış girdi komut adına girmiyor.
	gsKomut := exec.CommandContext(ctx, gsYol, args...)
	if out, err := gsKomut.CombinedOutput(); err != nil {
		return rap, fmt.Errorf("PDF/X dönüşümü başarısız: %w (%s)", err, firstLine(string(out), 300))
	}

	rap = s.gazeteMatbaaPreflight(ctx, matbaaPDF, rap)

	pdf, err := os.ReadFile(matbaaPDF)
	if err != nil {
		return rap, err
	}
	key := platform.GazeteKey(siteID, iss.IssueDate, gazeteMatbaaDosya)
	if _, err := s.storeForSite(ctx, siteID).Put(key, pdf, "application/pdf"); err != nil {
		return rap, err
	}
	rap.Anahtar = key
	return rap, nil
}

// gazeteMmSayi "210mm" gibi bir olcuyu sayiya cevirir.
func gazeteMmSayi(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "mm"), 64)
	if err != nil {
		return 0
	}
	return f
}

func (s *Server) gazeteMatbaaPreflight(ctx context.Context, yol string, rap gazeteMatbaaRapor) gazeteMatbaaRapor {
	ekle := func(f string, a ...any) { rap.Uyarilar = append(rap.Uyarilar, fmt.Sprintf(f, a...)) }

	// #nosec G702 -- komut adı ve bayraklar sabit; yol çağırandan gelir
	pdfinfoKomut := exec.CommandContext(ctx, "pdfinfo", "-box", yol)
	if b, err := pdfinfoKomut.Output(); err == nil {
		metin := string(b)
		for _, satir := range strings.Split(metin, "\n") {
			alan := strings.SplitN(satir, ":", 2)
			if len(alan) != 2 {
				continue
			}
			ad, deger := strings.TrimSpace(alan[0]), strings.TrimSpace(alan[1])
			switch ad {
			case "Page size":
				rap.SayfaBoyu = deger
			case "Pages":
				rap.Sayfa, _ = strconv.Atoi(deger)
			case "TrimBox":
				rap.TrimBox = true
			case "BleedBox":
				rap.BleedBox = true
			}
		}
	} else {
		ekle("pdfinfo çalıştırılamadı; sayfa kutuları denetlenemedi")
	}
	if !rap.TrimBox {
		ekle("TrimBox yok: matbaa kesimin nereden geçeceğini bilemez")
	}
	if !rap.BleedBox {
		ekle("BleedBox yok: taşma payı tanımsız")
	}

	// Cikti amaci (OutputIntent) PDF/X'in sarti.
	if b, err := os.ReadFile(yol); err == nil {
		rap.CiktiAmaci = strings.Contains(string(b), "OutputIntent")
	}
	if !rap.CiktiAmaci {
		ekle("çıktı amacı (OutputIntent) yok: dosya PDF/X sayılmaz")
	}

	// #nosec G702 -- komut adı ve bayraklar sabit; yol çağırandan gelir
	pdffontsKomut := exec.CommandContext(ctx, "pdffonts", yol)
	if b, err := pdffontsKomut.Output(); err == nil {
		satirlar := strings.Split(string(b), "\n")
		for i, satir := range satirlar {
			if i < 2 || strings.TrimSpace(satir) == "" {
				continue
			}
			alan := strings.Fields(satir)
			if len(alan) < 5 {
				continue
			}
			// pdffonts sutunlari: name type encoding emb sub uni object ID
			if alan[len(alan)-5] == "no" {
				rap.GomulmeyenF = append(rap.GomulmeyenF, alan[0])
			}
		}
	} else {
		ekle("pdffonts çalıştırılamadı; font gömülmesi denetlenemedi")
	}
	if len(rap.GomulmeyenF) > 0 {
		ekle("gömülü olmayan font: %s", strings.Join(rap.GomulmeyenF, ", "))
	}
	if rap.ICC == "" {
		ekle("ICC profili ayarlı değil; genel CMYK kullanıldı — matbaanın profilini isteyin")
	}
	return rap
}

func (s *Server) gazeteMatbaaHTML(ctx context.Context, siteID int64,
	iss repo.GazeteIssue, belge gazeteSablonDoc, boyut gazeteSayfaBoyut,
	tasmaMm float64) (string, error) {
	pages, err := repo.ListGazetePages(ctx, s.pool, iss.ID)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	for _, p := range pages {
		body.WriteString(`<div class="gz-matbaa-sayfa">` +
			gazeteKesimIsaretleri() +
			gazetePageHTML(p, fmt.Sprintf("Sayfa %d", p.PageNo)) +
			`</div>`)
	}
	title := iss.Title
	if title == "" {
		title = "Gazete — " + iss.IssueDate
	}

	kenarMm := gazeteIsaretAlaniMm + tasmaMm
	medyaW := gazeteMmSayi(boyut.W) + 2*kenarMm
	medyaH := gazeteMmSayi(boyut.H) + 2*kenarMm

	// Sayfa kutusu buyudu; icerik kutusu kesim olcusunde kaliyor.
	degiskenler := fmt.Sprintf(
		"@page { size: %.3fmm %.3fmm; margin: 0; }\nbody{margin:0}\n"+
			":root{--gz-sayfa-w:%s;--gz-sayfa-h:%s;--gz-kenar:%s;--gz-tip-tab:%s;"+
			"--gz-tip-olcek:%g;--gz-yazi:%s;--gz-matbaa-kenar:%.3fmm;}",
		medyaW, medyaH, boyut.W, boyut.H, boyut.Kenar, boyut.TipTab,
		belge.Sayfa.TipOlcek, gazeteYaziTipiCSS(belge.Sayfa.YaziTipi), kenarMm)

	return "<!doctype html><html lang=\"tr\"><head><meta charset=\"utf-8\"><title>" +
		html.EscapeString(title) + "</title><style>" +
		degiskenler + "\n" + gazeteBaskiCSS() + "\n" + gazeteMatbaaCSS +
		"</style></head><body>" + body.String() + "</body></html>", nil
}

// gazeteMatbaaCSS matbaa sayfasinin disinda kalan isaret alani ve kesim
// isaretleri. Dijital nushaya girmez.
const gazeteMatbaaCSS = `
.gz-matbaa-sayfa { position: relative; break-after: page; page-break-after: always;
  width: calc(var(--gz-sayfa-w) + 2 * var(--gz-matbaa-kenar));
  height: calc(var(--gz-sayfa-h) + 2 * var(--gz-matbaa-kenar));
  background: #fff; overflow: hidden; }
.gz-matbaa-sayfa:last-child { break-after: auto; page-break-after: auto; }
.gz-matbaa-sayfa > .gazete-sayfa { position: absolute;
  left: var(--gz-matbaa-kenar); top: var(--gz-matbaa-kenar); }
.gz-kesim-im { position: absolute; background: #000; }
.gz-kesim-im.yatay { height: 0.25pt; width: 5mm; }
.gz-kesim-im.dikey { width: 0.25pt; height: 5mm; }
`

func gazeteKesimIsaretleri() string {
	const bosluk = "2mm"
	kenar := "var(--gz-matbaa-kenar)"
	// Kesim kenarlarinin sayfa kutusuna gore konumu.
	sol := kenar
	ust := kenar
	sag := "calc(100% - " + kenar + ")"
	alt := "calc(100% - " + kenar + ")"

	im := func(sinif, stil string) string {
		return `<div class="gz-kesim-im ` + sinif + `" style="` + stil + `"></div>`
	}
	var sb strings.Builder
	// Sol ust
	sb.WriteString(im("yatay", "left:calc("+sol+" - "+bosluk+" - 5mm);top:"+ust+";"))
	sb.WriteString(im("dikey", "left:"+sol+";top:calc("+ust+" - "+bosluk+" - 5mm);"))
	// Sag ust
	sb.WriteString(im("yatay", "left:calc("+sag+" + "+bosluk+");top:"+ust+";"))
	sb.WriteString(im("dikey", "left:"+sag+";top:calc("+ust+" - "+bosluk+" - 5mm);"))
	// Sol alt
	sb.WriteString(im("yatay", "left:calc("+sol+" - "+bosluk+" - 5mm);top:"+alt+";"))
	sb.WriteString(im("dikey", "left:"+sol+";top:calc("+alt+" + "+bosluk+");"))
	// Sag alt
	sb.WriteString(im("yatay", "left:calc("+sag+" + "+bosluk+");top:"+alt+";"))
	sb.WriteString(im("dikey", "left:"+sag+";top:calc("+alt+" + "+bosluk+");"))
	return sb.String()
}

func (s *Server) handleGazeteMatbaa(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	id := parseID(r.PathValue("id"))
	rap, err := s.gazeteMatbaaUret(r.Context(), s.curSiteID(r), id)
	if err != nil {
		s.logger.Error("gazete matbaa", "sayi", id, "error", err)
		s.writeJSON(w, http.StatusOK, map[string]any{
			"ok":   false,
			"hata": err.Error(),
		})
		return
	}
	s.audit(r, "gazete_matbaa", strconv.FormatInt(id, 10))
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"hazir": rap.Hazir(),
		"rapor": rap,
	})
}
