package httpapi

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// Hisseler modülü (BIST hisse şeridi).
//
// Referans: Habertürk ekonomi sayfasındaki "HİSSELER" bölümü — solda kod,
// altında fiyat, sağda yön oku; tek satırda kesintisiz kayan şerit.
//
// Görünüm seçenekleri Kur & Borsa modülüyle AYNI sözlüğü kullanır:
//
//	hs_layout : serit | izgara              (yerleşim)
//	hs_style  : referans | sade | kart | dar (görünüm şablonu)
//	hs_arrow  : noktali | ucgen | ok | uzun | yok (yön göstergesi)
//	hs_bg     : kart | yok | zemin | ozel   (arka plan)
//
// Aynı anahtarlar bilinçli olarak seçildi: iki modül yan yana durduğunda
// kullanıcı aynı şablon adlarını görür ve aynı görsel dili elde eder.

const (
	hisseYerlesimVarsayilan = "serit"
	hisseStilVarsayilan     = "referans"
	hisseOkVarsayilan       = "noktali"
	hisseBgVarsayilan       = "kart"
	hisseSiralaVarsayilan   = "kod"
	hisseGecisVarsayilan    = "surekli"
	hisseHizVarsayilan      = 60 // px/sn (sürekli model)
	hisseHizEnAz            = 10
	hisseHizEnCok           = 300
	hisseBekleVarsayilan    = 4   // sn (aralıklı model)
	hisseKaymaVarsayilan    = 700 // ms (aralıklı model geçiş süresi)
	hisseItemPx             = 120 // kalem başına yaklaşık genişlik (CSS yedeği için)
	hisseSayiVarsayilan     = 20
	hisseSayiAzami          = 40
)

// gecerliHisseGecis geçiş modelleri.
//
//	surekli — Habertürk modeli: kesintisiz, sabit hızda akan şerit
//	adim    — aralıklı: her N saniyede bir yumuşak kayma + bekleme
//
// İkisi de aynı işaretlemeyi kullanır; modeli hisse.js uygular. JS yoksa
// CSS yedeği (kesintisiz) devreye girer, içerik yine okunabilir kalır.
var gecerliHisseGecis = map[string]bool{
	"surekli": true,
	"adim":    true,
}

// hisseGecisOf geçiş modelini okur; bilinmeyen değer varsayılana döner.
func hisseGecisOf(s string) string {
	if gecerliHisseGecis[s] {
		return s
	}
	return hisseGecisVarsayilan
}

// hisseHizOf şerit hızını px/sn olarak döndürür.
//
// Yeni ayar `hs_speed` (px/sn). Eski düzenlerde `hs_hiz` "bir turun süresi
// (sn)" idi; iki anlamı karıştırmamak için eski değer, kalem sayısına göre
// px/sn'ye çevrilir (count*itemPx / eskiSure). Böylece eski düzenler
// görsel olarak aynı hızda kalır.
func hisseHizOf(speed, eskiSure, count int) int {
	v := speed
	if v <= 0 && eskiSure > 0 {
		n := count
		if n < 1 {
			n = hisseSayiVarsayilan
		}
		v = n * hisseItemPx / eskiSure
	}
	if v <= 0 {
		v = hisseHizVarsayilan
	}
	if v < hisseHizEnAz {
		v = hisseHizEnAz
	}
	if v > hisseHizEnCok {
		v = hisseHizEnCok
	}
	return v
}

// hisseSureTur CSS yedeği için tur süresini saniye olarak hesaplar.
//
// Şerit -50% kaydığında bir grup tamamlanır; grup genişliği ≈ count*itemPx.
// Süre = genişlik / hız → hız kalem sayısından bağımsız sabit kalır.
func hisseSureTur(count, speed int) int {
	if count < 1 {
		count = hisseSayiVarsayilan
	}
	if speed < hisseHizEnAz {
		speed = hisseHizVarsayilan
	}
	s := count * hisseItemPx / speed
	if s < 6 {
		s = 6
	}
	if s > 600 {
		s = 600
	}
	return s
}

// gecerliHisseYerlesim şerit yerleşimleri.
var gecerliHisseYerlesim = map[string]bool{
	"serit":  true, // kayan şerit (referans)
	"izgara": true, // sabit ızgara (sayfa içinde kutu)
}

// gecerliHisseStil görünüm şablonları — Kur & Borsa ile birebir aynı.
var gecerliHisseStil = map[string]bool{
	"referans": true, // fiyat yön renginde, çerçevesiz
	"sade":     true, // nötr fiyat, yalnız değişim renkli
	"kart":     true, // her hisse kendi çerçevesinde
	"dar":      true, // tek satır kompakt (yan sütun)
}

// gecerliHisseSirala bilinen sıralamalar.
var gecerliHisseSirala = map[string]bool{
	"kod":      true, // alfabetik (referans)
	"yukselen": true, // en çok yükselen başta
	"dusen":    true, // en çok düşen başta
}

// hisseHizSecenekleri akış hızı ön ayarları (px/sn).
//
// Kullanıcı "birebir değiştirilebilir" hız istediği için hem hazır
// adımlar hem serbest px/sn alanı sunulur. Ön ayar seçiliyse sayısal alan
// yok sayılır; "ozel" seçilirse serbest alan geçerli olur.
var hisseHizSecenekleri = []struct{ Key, Label string }{
	{"20", "Çok yavaş — 20 px/sn"},
	{"30", "Yavaş — 30 px/sn"},
	{"45", "Orta — 45 px/sn"},
	{"60", "Normal — 60 px/sn (Habertürk)"},
	{"90", "Hızlı — 90 px/sn"},
	{"120", "Çok hızlı — 120 px/sn"},
	{"180", "Ultra — 180 px/sn"},
	{"ozel", "Özel (aşağıdaki px/sn)"},
}

// hisseHizSecimPx ön ayar anahtarını px/sn'ye çevirir.
func hisseHizSecimPx(key string) (int, bool) {
	if key == "" || key == "ozel" {
		return 0, false
	}
	n, err := strconv.Atoi(key)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// hisseHizCoz ön ayar + serbest alan + eski (sn/tur) değeri tek yerde çözer.
//
// Öncelik: ön ayar → serbest px/sn → eski sn/tur → varsayılan.
func hisseHizCoz(secim string, ozel, eskiSure, count int) int {
	if px, ok := hisseHizSecimPx(secim); ok {
		return hisseHizOf(px, 0, count)
	}
	return hisseHizOf(ozel, eskiSure, count)
}

// hisseYerlesimOf blok ayarından yerleşimi okur.
//
// Geriye dönük uyum: modül ilk sürümde yerleşimi `hs_style` içinde
// tutuyordu (serit | izgara). Eski düzenler bozulmasın diye o değerler
// yerleşim sayılır ve görünüm "referans"a düşer.
func hisseYerlesimOf(layout, eskiStil string) string {
	if gecerliHisseYerlesim[layout] {
		return layout
	}
	if gecerliHisseYerlesim[eskiStil] {
		return eskiStil
	}
	return hisseYerlesimVarsayilan
}

// hisseStilOf blok ayarından görünüm şablonunu okur; bilinmeyen değer
// varsayılana döner (eski "serit"/"izgara" değerleri buraya düşer).
//
// "otomatik" özel bir değerdir: bölgeye göre seçim yapılır (bkz. hisseEtkinStil).
func hisseStilOf(s string) string {
	if s == "otomatik" {
		return "otomatik"
	}
	if gecerliHisseStil[s] {
		return s
	}
	return hisseStilVarsayilan
}

// darBolge modülün dar bir bölgeye (sağ ray / yan sütun) konduğunu söyler.
//
// Sağ ray ~300-380 px'tir; orada 2-4 sütunlu ızgara ve geniş kartlar
// sıkışır. "otomatik" şablon bu bilgiyi kullanarak kompakt görünüme geçer.
func darBolge(zone string) bool {
	return zone == "yan"
}

// hisseEtkinStil "otomatik" seçimini bölgeye göre çözer.
//
//	geniş bölge (ust/ana/genis/govde) → referans (tam kart)
//	dar bölge  (yan)                  → dar (tek satır kompakt)
func hisseEtkinStil(secili string, dar bool) string {
	if secili == "otomatik" {
		if dar {
			return "dar"
		}
		return hisseStilVarsayilan
	}
	return secili
}

// hisseSiralaOf blok ayarından sıralamayı okur.
func hisseSiralaOf(s string) string {
	if gecerliHisseSirala[s] {
		return s
	}
	return hisseSiralaVarsayilan
}

// hisseBgOf blok ayarından arka plan seçimini okur.
// Sözlük: kart (modül yüzeyi) | yok (saydam) | zemin | ozel.
func hisseBgOf(s string) string {
	if gecerliKbBg[s] {
		return s
	}
	return hisseBgVarsayilan
}

// hisseOkOf yön göstergesi tasarımını okur (kb_arrow ile aynı sözlük).
//
// Geriye dönük uyum: ilk sürümde tek bir `hs_ok` bool'u vardı. Anahtar
// varsa ve false ise gösterge kapalı sayılır; true ise varsayılan (noktalı).
func hisseOkOf(stil string, eskiOk *bool) string {
	if gecerliKbOk[stil] {
		if stil == hisseOkVarsayilan && eskiOk != nil && !*eskiOk {
			return "yok"
		}
		return stil
	}
	if eskiOk != nil && !*eskiOk {
		return "yok"
	}
	return hisseOkVarsayilan
}

// hisseRows önbellekteki satırları sıralayıp istenen sayıya kırpar.
//
// Girdi dilimi DEĞİŞTİRİLMEZ: sıralama kopya üzerinde yapılır, aynı istekte
// iki farklı sıralamalı hisse modülü birbirini etkilemez.
func hisseRows(rows []repo.HisseRow, sirala string, count int) []repo.HisseRow {
	if len(rows) == 0 {
		return nil
	}
	out := make([]repo.HisseRow, len(rows))
	copy(out, rows)
	switch hisseSiralaOf(sirala) {
	case "yukselen":
		sort.SliceStable(out, func(i, j int) bool { return out[i].Delta > out[j].Delta })
	case "dusen":
		sort.SliceStable(out, func(i, j int) bool { return out[i].Delta < out[j].Delta })
	default:
		sort.SliceStable(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	}
	if count > 0 && len(out) > count {
		out = out[:count]
	}
	return out
}

// hisseYon fiyat yönünü sınıf adına çevirir (up | down | flat).
func hisseYon(delta float64) string {
	switch {
	case delta > 0:
		return "up"
	case delta < 0:
		return "down"
	}
	return "flat"
}

// hisseYonInt şablonun ok çizimi için yönü -1/0/1'e çevirir (kb_ok sözleşmesi).
func hisseYonInt(delta float64) int {
	switch {
	case delta > 0:
		return 1
	case delta < 0:
		return -1
	}
	return 0
}

// ---- "Tüm Hisseler" sayfası -------------------------------------------
//
// Referans: haberturk.com/ekonomi/borsa/tum-hisseler
//   TÜM HİSSELER başlığı + "Hisse Ara" kutusu + A-Z dizini + tablo
//   (HİSSE ADI · FİYAT · FARK).
//
// Sayfa TÜM evreni (633 sembol) listeler: fiyatı olmayan sembol de görünür
// (fiyatı "—" olarak basılır). Böylece arama yapan kullanıcı, o an işlem
// görmeyen/verisi gelmeyen hisseyi de bulabilir. Uydurma fiyat basılmaz.

// hisseSatir sayfadaki tek tablo satırı.
type hisseSatir struct {
	Code  string
	Name  string
	Ilk   string // alfabetik avatar harfi (A-Z)
	Arama string // arama için sadeleştirilmiş "kod + ad" (istemci filtresi)
	Value float64
	Delta float64
	Var   bool // fiyat verisi var mı
}

// Yon sınıfı (CSS): up | down | flat.
func (h hisseSatir) Yon() string { return hisseYon(h.Delta) }

// YonInt ok çizimi sözleşmesi (-1/0/1).
func (h hisseSatir) YonInt() int { return hisseYonInt(h.Delta) }

// hisseSayfa "Tüm Hisseler" sayfasının görünüm verisi.
type hisseSayfa struct {
	Satir   []hisseSatir
	Harfler []string // A-Z dizini
	Q       string   // arama metni
	Harf    string   // seçili harf ("" = tümü)
	Toplam  int      // filtresiz toplam
	Adet    int      // filtreden sonra
	// Guncel fiyat verisinin zamanı (boşsa "veri bekleniyor" notu).
	Guncel time.Time
}

// harfler alfabetik dizin (referansta Q, W, X olmadığı için yok).
var hisseHarfleri = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "R", "S", "T", "U", "V", "Y", "Z",
}

// hisseAramaNorm arama için metni sadeleştirir: Türkçe harfler ASCII'ye,
// küçük harfe, fazla boşluklar tek boşluğa iner.
//
// Neden: kullanıcı "şişe", "SISE" veya "sise" yazabilir; üçü de aynı hisseyi
// bulmalı. Ayrıca kod ortasındaki boşluk/orta nokta aramayı bozmasın.
func hisseAramaNorm(s string) string {
	repl := strings.NewReplacer(
		"ı", "i", "İ", "i", "I", "i",
		"ş", "s", "Ş", "s",
		"ğ", "g", "Ğ", "g",
		"ü", "u", "Ü", "u",
		"ö", "o", "Ö", "o",
		"ç", "c", "Ç", "c",
		"â", "a", "Â", "a",
		"î", "i", "Î", "i",
		"û", "u", "Û", "u",
	)
	s = repl.Replace(s)
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), " ")
}

// hisseSayfaOlustur evreni fiyat önbelleğiyle birleştirip filtreler.
//
// Filtre sırası: önce harf, sonra arama. Arama kodu ve adı birlikte tarar;
// kod eşleşmesi öne alınır (kullanıcı çoğunlukla kod yazar).
func hisseSayfaOlustur(hd repo.HisseData, q, harf string) *hisseSayfa {
	byCode := make(map[string]repo.HisseRow, len(hd.Rows))
	for _, r := range hd.Rows {
		byCode[r.Code] = r
	}
	qn := hisseAramaNorm(q)
	harf = strings.ToUpper(strings.TrimSpace(harf))

	out := &hisseSayfa{Q: strings.TrimSpace(q), Harf: harf, Guncel: hd.Updated, Harfler: hisseHarfleri}
	for _, meta := range repo.HisseEvreni() {
		out.Toplam++
		ilk := string([]rune(meta.Code)[0])
		if harf != "" && ilk != harf {
			continue
		}
		if qn != "" {
			codeN := hisseAramaNorm(meta.Code)
			nameN := hisseAramaNorm(meta.Name)
			if !strings.Contains(codeN, qn) && !strings.Contains(nameN, qn) {
				continue
			}
		}
		row := hisseSatir{
			Code: meta.Code, Name: meta.Name, Ilk: ilk,
			Arama: hisseAramaNorm(meta.Code + " " + meta.Name),
		}
		if r, ok := byCode[meta.Code]; ok && r.Value > 0 {
			row.Value, row.Delta, row.Var = r.Value, r.Delta, true
		}
		out.Satir = append(out.Satir, row)
	}
	out.Adet = len(out.Satir)
	return out
}

// paraKisa büyük tutarları okunur kısaltmayla yazar (TRY).
//
//	845        → "845,00"
//	12.345     → "12,3 b"   (bin)
//	45.600.000 → "45,6 mn"  (milyon)
//	1.234.000.000 → "1,23 mr" (milyar)
//
// Türkçe kısaltmalar: b=bin, mn=milyon, mr=milyar, tn=trilyon. Tablo
// sütununda tam sayı (25.520.814.082) okunmuyor; Habertürk tam yazıyor ama
// TRY hacimleri 12 haneye çıkıyor. Kısaltma + başlık ipucu (title) ile
// tam değer korunur.
func paraKisa(v float64) string {
	if v <= 0 {
		return "—"
	}
	bicim := func(x float64, hane int) string {
		return piyasaNum(x, hane)
	}
	switch {
	case v >= 1e12:
		return bicim(v/1e12, 2) + " tn"
	case v >= 1e9:
		return bicim(v/1e9, 2) + " mr"
	case v >= 1e6:
		return bicim(v/1e6, 1) + " mn"
	case v >= 1e3:
		return bicim(v/1e3, 1) + " b"
	}
	return bicim(v, 2)
}
func ilkHarf(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) == 0 {
		return "?"
	}
	return strings.ToUpper(string(r[0]))
}
