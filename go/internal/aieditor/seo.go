package aieditor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/repo"
)

// SEO alanlarının hedef uzunlukları. Google sonuç sayfasında başlık ~60,
// açıklama ~155 karakterde kesiliyor; kesilen metin kullanıcıya yarım
// cümle gösteriyor. Üretimi bu bandın içine zorluyoruz.
const (
	SeoBaslikMin = 30
	SeoBaslikMax = 60
	// Kesin üst sınır: band aşılsa bile bundan uzun yazılmaz.
	SeoBaslikSertMax = 70

	SeoAciklamaMin     = 110
	SeoAciklamaMax     = 155
	SeoAciklamaSertMax = 165

	// Modele verilen gövde metni bu kadar karakterle sınırlanır: SEO için
	// ilk paragraflar yeterli, uzun haberde maliyet gereksiz artıyordu.
	seoGovdeLimiti = 2500
)

// SeoOneri üretilen SEO alanlarıdır.
//
// Alanlar DB'ye OTOMATİK yazılmaz (editör akışında); bot akışında yazılır.
// İkisi de aynı üretim + doğrulama yolunu kullanır.
type SeoOneri struct {
	Baslik   string
	Aciklama string
	// Anahtar modelin seçtiği anahtar kelimelerdir; yalnız bilgi amaçlı
	// gösterilir, DB'de ayrı bir kolon yok.
	Anahtar []string
	// Uyarilar bandın dışına çıkan alanları söyler (yazma engellenmez).
	Uyarilar []string
	Provider string
	Model    string
}

// SeoInstruksiyon SEO üretiminin modele verilen kuralıdır. PromptVersion ile
// birlikte sürümlenir; ai_jobs kaydında iz kalır.
const SeoInstruksiyon = `Verilen haber metnine göre SEO alanlarını doldur.
Kurallar:
- seo_title: 30-60 karakter. Haberin özünü veren, tıklama tuzağı olmayan başlık.
  Haber başlığını birebir kopyalama; arama sorgusuna uygun doğal ifade kur.
- meta_description: 110-155 karakter. Tek cümle veya iki kısa cümle; haberin
  ne olduğunu ve neden önemli olduğunu söyle. "Tıklayın", "okumaya devam edin",
  "şok edici" gibi ifadeler kullanma.
- keywords: 3-6 kısa anahtar kelime. Arama hacmi/rekabet metriği UYDURMA.
- Metinde olmayan kişi, kurum, tarih veya rakam EKLEME. Metinde yoksa genel
  ifade kullan.
- HABERİN KAYNAĞI BAŞKA BİR YAYIN OLSA BİLE o yayının adını veya alan adını
  YAZMA. "…com.tr'da", "… gazetesine göre" gibi ifade kullanma. Site adı
  verme; metni kendi haberimiz gibi yaz.
- Türkçe yaz. Tırnak, satır sonu ve emoji kullanma.
Şu JSON şemasını döndür:
{"seo_title":"","meta_description":"","keywords":[""]}`

// seoJSON modelin döndürdüğü JSON'u çözer.
//
// Modeller zaman zaman JSON'u ```json bloğuna sarıyor veya başına bir cümle
// ekliyor. Katı çözüm bu yüzden tek başına yetmiyor; ilk { ... } aralığı
// ayıklanır.
func seoJSON(ham string) (map[string]any, error) {
	ham = strings.TrimSpace(ham)
	if i := strings.Index(ham, "{"); i > 0 {
		ham = ham[i:]
	}
	if j := strings.LastIndex(ham, "}"); j >= 0 && j < len(ham)-1 {
		ham = ham[:j+1]
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(ham), &m); err == nil {
		return m, nil
	} else if kurtarilmis, ok := seoJSONKurtar(ham); ok {
		// Kesik JSON: alanlar tek tek çıkarılır.
		return kurtarilmis, nil
	} else {
		return nil, fmt.Errorf("SEO yanıtı JSON değil: %w", err)
	}
}

// seoJSONKurtar KESİK JSON'dan alanları kurtarır.
//
// Sağlayıcı bazen yanıtı token sınırında ortasından kesiyor; canlı ölçümde
// yanıtların bir kısmı "Unterminated string" / "unexpected end of JSON
// input" ile geliyor ve içerik boş kalıyordu. Başlık ve açıklama genellikle
// kesik metnin BAŞINDA olduğu için bunlar kurtarılabilir; uzunluk
// doğrulaması zaten kırpar.
//
// Yalnız iki alan aranır ve ikisinden en az biri bulunmalıdır; aksi hâlde
// "kurtarma" uydurma sayılır.
func seoJSONKurtar(ham string) (map[string]any, bool) {
	alan := func(ad string) string {
		re := regexp.MustCompile(`"` + ad + `"\s*:\s*"((?:[^"\\]|\\.)*)`)
		m := re.FindStringSubmatch(ham)
		if len(m) < 2 {
			return ""
		}
		// Kaçış dizilerini çöz; bozuk kaçışta ham değeri kullan.
		if coz, err := strconv.Unquote(`"` + m[1] + `"`); err == nil {
			return coz
		}
		return m[1]
	}
	baslik := alan("seo_title")
	aciklama := alan("meta_description")
	if baslik == "" && aciklama == "" {
		return nil, false
	}
	m := map[string]any{}
	if baslik != "" {
		m["seo_title"] = baslik
	}
	if aciklama != "" {
		m["meta_description"] = aciklama
	}
	// Anahtar kelimeler kesik olabilir; kurtarmada istenmez.
	return m, true
}

// seoTemizle metni tek satıra indirir: satır sonları, fazla boşluk, sarma
// tırnakları ve kontrol karakterleri atılır.
//
// Neden gerekli: bu değerler <title> ve <meta content> içine giriyor; satır
// sonu veya tırnak SEO çıktısını bozuyor (ve HTML kaçışı gereksiz yer
// kaplıyor). Modelin tırnak içine alması sık görülüyor.
func seoTemizle(s string) string {
	// Kontrol karakterleri: boşluk olanlar (tab, satır sonu) boşluğa
	// ÇEVRİLİR, diğerleri atılır.
	//
	// Doğrudan silmek kelimeleri yapıştırıyordu: "konutlar\tteslim" →
	// "konutlarteslim" (testte yakalandı). Boşluğa çevrilince
	// strings.Fields aşağıda tek boşluğa indiriyor.
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f':
			return ' '
		case r < 32 || r == 127:
			return -1
		}
		return r
	}, s)
	// strings.Fields tüm Unicode boşluklarını (satır sonu, sekme, çoklu
	// boşluk) tek boşluğa indirir; ayrı ReplaceAll çağrıları gereksizdi.
	s = strings.Join(strings.Fields(s), " ")
	// Sarma tırnakları: düz ve kıvrık. Raw string yerine yorumlanan
	// string kullanılmalı: `\u201c` backtick içinde kaçış DEĞİL, dört
	// karakterlik metindir (canlı testte kıvrık tırnak temizlenmiyordu).
	s = strings.Trim(s, "\"'"+string(rune(0x201c))+string(rune(0x201d))+
		string(rune(0x2018))+string(rune(0x2019)))
	return strings.TrimSpace(s)
}

// seoKirp metni rune sınırında, kelimeyi bölmeden kısaltır.
//
// Byte ile kesmek Türkçe karakterleri bozuyor (ı, ş, ğ iki bayt). Cümle
// ortasında kesmek yerine son tam kelimeye kadar geri gidilir.
func seoKirp(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	r = r[:max]
	// Son boşluğa kadar geri git (en az yarısı kalsın).
	for i := len(r) - 1; i > max/2; i-- {
		if r[i] == ' ' {
			r = r[:i]
			break
		}
	}
	return strings.TrimRight(strings.TrimRightFunc(string(r), unicode.IsSpace), ".,;:")
}

// SeoDogrula uzunluk bandını denetler, sert sınırı uygular ve uyarı üretir.
func SeoDogrula(o SeoOneri) SeoOneri {
	o.Baslik = seoKirp(seoTemizle(o.Baslik), SeoBaslikSertMax)
	o.Aciklama = seoKirp(seoTemizle(o.Aciklama), SeoAciklamaSertMax)

	nb := utf8.RuneCountInString(o.Baslik)
	na := utf8.RuneCountInString(o.Aciklama)

	o.Uyarilar = nil
	if nb < SeoBaslikMin {
		o.Uyarilar = append(o.Uyarilar, fmt.Sprintf(
			"SEO başlığı kısa (%d karakter); %d-%d arası önerilir.", nb, SeoBaslikMin, SeoBaslikMax))
	} else if nb > SeoBaslikMax {
		o.Uyarilar = append(o.Uyarilar, fmt.Sprintf(
			"SEO başlığı uzun (%d karakter); Google %d civarında kesiyor.", nb, SeoBaslikMax))
	}
	if na < SeoAciklamaMin {
		o.Uyarilar = append(o.Uyarilar, fmt.Sprintf(
			"Meta açıklama kısa (%d karakter); %d-%d arası önerilir.", na, SeoAciklamaMin, SeoAciklamaMax))
	} else if na > SeoAciklamaMax {
		o.Uyarilar = append(o.Uyarilar, fmt.Sprintf(
			"Meta açıklama uzun (%d karakter); %d civarında kesiliyor.", na, SeoAciklamaMax))
	}
	if o.Baslik == "" {
		o.Uyarilar = append(o.Uyarilar, "SEO başlığı üretilemedi.")
	}
	if o.Aciklama == "" {
		o.Uyarilar = append(o.Uyarilar, "Meta açıklama üretilemedi.")
	}
	return o
}

// alanAdiDeseni alan adı benzeri tokenleri yakalar: "cumhuriyet.com.tr",
// "https://x.com", "www.foo.net".
//
// Neden gerekli: model, kaynak yayının adını metne ekliyordu. Canlı ölçümde
// gövdesinde "Cumhuriyet" GEÇMEYEN bir spor haberinin meta açıklaması
// "… Cumhuriyet.com.tr'da" diye bitiyordu. Bu, olgusal olarak yanlış ve
// başka bir yayına atıf; yayına girmemeli.
var alanAdiDeseni = regexp.MustCompile(
	`(?i)(?:https?://\S+|www\.[\w.-]+|\b[\w-]+(?:\.[\w-]+)*\.(?:com|net|org|gov|edu|info|io)(?:\.[a-z]{2})?\b)`)

// seoAlanAdiTemizle metinde GEÇMEYEN alan adlarını içeren cümleleri atar.
//
// Kaynak metinde geçen bir alan adı korunur (haberin konusu olabilir).
// Temizlik sonrası alan boş kalırsa alan boşaltılır: uydurma bir cümleyi
// tutmaktansa alanı boş bırakıp uyarı vermek doğru.
func seoAlanAdiTemizle(metin, kaynak string) (string, bool) {
	if metin == "" {
		return "", false
	}
	kaynakKucuk := strings.ToLower(kaynak)
	degisti := false

	cumleler := seoCumlelere(metin)
	kalan := make([]string, 0, len(cumleler))
	for _, c := range cumleler {
		uygun := true
		for _, eslesme := range alanAdiDeseni.FindAllString(c, -1) {
			temiz := strings.ToLower(strings.Trim(eslesme, `.,;:()"'`))
			if temiz == "" {
				continue
			}
			if !strings.Contains(kaynakKucuk, temiz) {
				uygun = false
				break
			}
		}
		if uygun {
			kalan = append(kalan, c)
		} else {
			degisti = true
		}
	}
	return strings.Join(kalan, " "), degisti
}

// SeoAlanAdiKorumasi üretilen alanlardaki uydurma site adlarını temizler.
//
// Sahadan gelen gerçek vaka: gövdesinde "Cumhuriyet" geçmeyen bir spor
// haberinin meta açıklaması "… Cumhuriyet.com.tr'da" diye bitiyordu.
// Model kaynak yayının adını uydurmuştu; bu yanlış atıf yayına girmemeli.
//
// Temizlik SeoUret içinde satır içi yazıldığında testler yalnız
// seoAlanAdiTemizle'yi doğruluyordu; uygulama silinse bile testler yeşil
// kalıyordu (mutasyon denemesinde yakalandı). Bu yüzden tek saf fonksiyon:
// hem üretim hem test aynı yolu kullanır.
func SeoAlanAdiKorumasi(o SeoOneri, kaynak string) SeoOneri {
	if temiz, degisti := seoAlanAdiTemizle(o.Baslik, kaynak); degisti {
		o.Uyarilar = append(o.Uyarilar,
			"SEO başlığında metinde geçmeyen site adı vardı; çıkarıldı.")
		o.Baslik = temiz
	}
	if temiz, degisti := seoAlanAdiTemizle(o.Aciklama, kaynak); degisti {
		o.Uyarilar = append(o.Uyarilar,
			"Meta açıklamada metinde geçmeyen site adı vardı; o cümle çıkarıldı.")
		o.Aciklama = temiz
	}
	return o
}

// seoCumlelere metni cümlelere böler (sonlandırıcı cümlede kalır).
//
// Sonlandırıcı YALNIZ boşluk/bitiş ile takip ediliyorsa cümle bitirir.
// Aksi hâlde alan adının içindeki nokta da bölme yapıyordu:
// "Cumhuriyet.com.tr'da" → "Cumhuriyet." + " com." + " tr'da" (testte
// yakalandı). Bu hem metni bozuyor hem alan adı korumasını etkisiz
// kılıyordu.
func seoCumlelere(s string) []string {
	var out []string
	var sb strings.Builder
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		sb.WriteRune(r[i])
		if r[i] != '.' && r[i] != '!' && r[i] != '?' {
			continue
		}
		// Sonlandırıcıdan sonra boşluk veya metin sonu gelmeli.
		if i+1 < len(r) && !unicode.IsSpace(r[i+1]) {
			continue
		}
		if c := strings.TrimSpace(sb.String()); c != "" {
			out = append(out, c)
		}
		sb.Reset()
	}
	if kalan := strings.TrimSpace(sb.String()); kalan != "" {
		out = append(out, kalan)
	}
	return out
}

// SeoUret içerikten SEO alanlarını üretir. Sağlayıcı yoksa hata döner
// (mock'a düşülmez: uydurma SEO metni yayına girmemeli).
func SeoUret(ctx context.Context, aiclient *ai.Client, c repo.Content) (SeoOneri, error) {
	if aiclient == nil || !aiclient.Enabled() {
		return SeoOneri{}, errors.New("AI sağlayıcısı yapılandırılmadı")
	}
	baslik := strings.TrimSpace(c.Title)
	govde := seoGovde(c.Detail)
	spot := strings.TrimSpace(c.Spot)
	if baslik == "" && govde == "" {
		return SeoOneri{}, errors.New("SEO için başlık veya gövde gerekli")
	}
	if len([]rune(govde)) > seoGovdeLimiti {
		govde = seoKirp(govde, seoGovdeLimiti)
	}

	kullanici := strings.Builder{}
	kullanici.WriteString("Haber başlığı: ")
	kullanici.WriteString(baslik)
	if spot != "" {
		kullanici.WriteString("\nSpot: ")
		kullanici.WriteString(spot)
	}
	if govde != "" {
		kullanici.WriteString("\nGövde:\n")
		kullanici.WriteString(govde)
	}

	resp, err := aiclient.CompleteStrict(ctx, ai.Request{
		Messages: []ai.Msg{
			{Role: "system", Content: CommonSystemPrompt},
			{Role: "user", Content: SeoInstruksiyon + "\n\n---\n" + kullanici.String()},
		},
		JSONMode: true,
		// 400 token YETMİYOR: başlık + 155 karakterlik açıklama + 6 anahtar
		// kelime JSON'u 465-500 token tutuyor ve yanıt ORTASINDAN kesiliyordu
		// ("Unterminated string", canlı ölçümde 3 yanıttan 1'i). Kesilen
		// yanıt ayrıştırılamıyor, içerik boş kalıyordu.
		MaxTokens:   700,
		Temperature: 0.2,
	})
	if err != nil {
		return SeoOneri{}, err
	}

	m, err := seoJSON(resp.Text)
	if err != nil {
		return SeoOneri{}, err
	}
	o := SeoOneri{
		Baslik:   seoMetin(m["seo_title"]),
		Aciklama: seoMetin(m["meta_description"]),
		Anahtar:  seoAnahtarlar(m["keywords"]),
		Provider: resp.Provider,
		Model:    resp.Model,
	}
	// Alan adı koruması: gövdede/spotta/başlıkta geçmeyen site adı atılır.
	kaynak := strings.Join([]string{baslik, spot, govde}, " ")
	o = SeoAlanAdiKorumasi(o, kaynak)

	o = SeoDogrula(o)
	if o.Baslik == "" || o.Aciklama == "" {
		return o, errors.New("AI SEO alanlarını üretemedi")
	}
	return o, nil
}

// seoMetin JSON değerini metne çevirir (null/eksik güvenli).
func seoMetin(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

// seoAnahtarlar keywords alanını temizler; en çok 6 öğe.
func seoAnahtarlar(v any) []string {
	var ham []string
	switch t := v.(type) {
	case []any:
		for _, it := range t {
			if s := seoTemizle(seoMetin(it)); s != "" {
				ham = append(ham, s)
			}
		}
	case []string:
		for _, it := range t {
			if s := seoTemizle(it); s != "" {
				ham = append(ham, s)
			}
		}
	case string:
		for _, it := range strings.Split(t, ",") {
			if s := seoTemizle(it); s != "" {
				ham = append(ham, s)
			}
		}
	}
	if len(ham) > 6 {
		ham = ham[:6]
	}
	return ham
}

// seoGovde içerik gövdesini düz metne çevirir (bloklar veya TipTap).
func seoGovde(detail []byte) string {
	if len(detail) == 0 {
		return ""
	}
	return strings.TrimSpace(blocksToText(detail))
}

// seoYazSQL SEO alanlarını yazan tek sorgudur.
//
// $1 id · $2 seo_title · $3 force · $4 seo_description
//
// force=false iken CASE yalnız BOŞ alanı doldurur; dolu alan kendi değerini
// korur. Bu, arka plandaki işin editörün elle yazdığı metni sessizce
// ezmesini engeller. RETURNING dolu alan sayısını verir (0 = değişiklik yok).
const seoYazSQL = `
	UPDATE contents SET
		seo_title = CASE
			WHEN $3 OR COALESCE(NULLIF(TRIM(seo_title),''),'') = ''
				THEN COALESCE(NULLIF($2,''), seo_title)
			ELSE seo_title END,
		seo_description = CASE
			WHEN $3 OR COALESCE(NULLIF(TRIM(seo_description),''),'') = ''
				THEN COALESCE(NULLIF($4,''), seo_description)
			ELSE seo_description END
	WHERE id = $1
	RETURNING
		(CASE WHEN COALESCE(NULLIF(TRIM(seo_title),''),'') <> '' THEN 1 ELSE 0 END) +
		(CASE WHEN COALESCE(NULLIF(TRIM(seo_description),''),'') <> '' THEN 1 ELSE 0 END)`

// SeoYaz SEO alanlarını içeriğe yazar.
//
// force=false iken DOLU ALANLAR KORUNUR: editörün elle yazdığı SEO metni,
// arka plandaki bir iş tarafından sessizce ezilmemeli. force=true yalnız
// editörün açıkça "üzerine yaz" dediği yerde kullanılır.
//
// Dönen değer gerçekten yazılan alan sayısıdır (0 ise hiçbir şey değişmedi).
func SeoYaz(ctx context.Context, pool *pgxpool.Pool, contentID int64, o SeoOneri, force bool) (int, error) {
	if contentID <= 0 {
		return 0, errors.New("geçersiz içerik")
	}
	if o.Baslik == "" && o.Aciklama == "" {
		return 0, errors.New("yazılacak SEO alanı yok")
	}
	var n int
	err := pool.QueryRow(ctx, seoYazSQL,
		contentID, o.Baslik, force, o.Aciklama).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// SeoBosTaslaklar SEO alanları boş, kaynağı olan (otomatik çekilmiş)
// taslakları döndürür. Bot akışı bu listeyi işler.
//
// Neden source_url: yalnız elle açılmış taslaklara dokunmuyoruz; editörün
// kendi başlattığı içerikte SEO'yu kendisi yazmak isteyebilir.
func SeoBosTaslaklar(ctx context.Context, pool *pgxpool.Pool, limit int) ([]repo.Content, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := pool.Query(ctx, `
		SELECT id FROM contents
		WHERE status = 'taslak'
		  AND COALESCE(NULLIF(TRIM(seo_title),''),'') = ''
		  AND COALESCE(NULLIF(TRIM(seo_description),''),'') = ''
		  AND COALESCE(source_url,'') <> ''
		  AND COALESCE(title,'') <> ''
		ORDER BY created_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]repo.Content, 0, len(ids))
	for _, id := range ids {
		c, err := repo.GetContent(ctx, pool, id)
		if err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}
