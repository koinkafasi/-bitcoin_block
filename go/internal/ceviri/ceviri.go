// Package ceviri yayımlanmış içerikleri başka dillere AI ile çevirir.
//
// Neden ayrı paket: aieditor kullanıcıya bağlı, tek içerik odaklı bir iş
// akışı (seçim → öneri → onay). Otomatik çeviri ise editörsüz, toplu ve
// süreklidir; kullanıcı/izin kavramı yoktur.
package ceviri

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// VarsayilanDiller otomatik çeviri yapılan dillerdir.
//
// Neden yalnız "en": her dil her içerik için AI maliyeti demek. İngilizce
// kullanıcının önceliği; diğerleri panelden eklenebilir.
var VarsayilanDiller = []string{"en"}

// VarsayilanDakika dakikada çevrilecek içerik sayısıdır.
const VarsayilanDakika = 8

// VarsayilanEszamanli aynı anda uçuşta olacak çeviri isteği sayısı.
const VarsayilanEszamanli = 3

// MaksDakika panelden girilebilecek üst sınır.
const MaksDakika = 120

// TurLimiti tek turda işlenecek en fazla içerik.
const TurLimiti = 10

// Ayarlar otomatik çevirinin açık/kapalı, hedef dil ve hız bilgisidir.
type Ayarlar struct {
	Aktif  bool
	Diller []string
	// Durum çevirinin hangi durumla kaydedileceği: "yayinda" veya "taslak".
	//
	// "yayinda" varsayılan: kullanıcı İngilizce içeriğin temada nasıl
	// göründüğünü görmek istedi; taslak olsaydı /en boş kalırdı. Kalite
	// kontrolü isteyen "taslak" seçebilir (panel).
	Durum  string
	Dakika int
}

// AyarlariOku settings tablosundan okur.
func AyarlariOku(ctx context.Context, pool *pgxpool.Pool) Ayarlar {
	a := Ayarlar{Aktif: true, Diller: VarsayilanDiller, Durum: "yayinda", Dakika: VarsayilanDakika}
	if b, err := repo.GetSetting(ctx, pool, "ceviri", "otomatik"); err == nil {
		a.Aktif = strings.Trim(string(b), `" `) == "true"
	}
	if b, err := repo.GetSetting(ctx, pool, "ceviri", "diller"); err == nil {
		var diller []string
		ham := strings.Trim(string(b), `" `)
		for _, d := range strings.Split(ham, ",") {
			d = strings.TrimSpace(d)
			if d != "" && d != "tr" {
				diller = append(diller, d)
			}
		}
		if len(diller) > 0 {
			a.Diller = diller
		}
	}
	if b, err := repo.GetSetting(ctx, pool, "ceviri", "durum"); err == nil {
		if d := strings.Trim(string(b), `" `); d == "taslak" || d == "yayinda" {
			a.Durum = d
		}
	}
	if b, err := repo.GetSetting(ctx, pool, "ceviri", "dakika"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `" `)); e == nil && n > 0 {
			a.Dakika = sinirDakika(n)
		}
	}
	return a
}

// AyarlariYaz panel formundan yazar.
func AyarlariYaz(ctx context.Context, pool *pgxpool.Pool, aktif bool, diller []string, durum string, dakika int) error {
	v := "false"
	if aktif {
		v = "true"
	}
	if err := repo.SetSetting(ctx, pool, "ceviri", "otomatik", []byte(v)); err != nil {
		return err
	}
	temiz := make([]string, 0, len(diller))
	for _, d := range diller {
		d = strings.TrimSpace(d)
		if d != "" && d != "tr" {
			temiz = append(temiz, d)
		}
	}
	if len(temiz) == 0 {
		temiz = VarsayilanDiller
	}
	if err := repo.SetSetting(ctx, pool, "ceviri", "diller", []byte(strings.Join(temiz, ","))); err != nil {
		return err
	}
	if durum != "taslak" {
		durum = "yayinda"
	}
	if err := repo.SetSetting(ctx, pool, "ceviri", "durum", []byte(durum)); err != nil {
		return err
	}
	return repo.SetSetting(ctx, pool, "ceviri", "dakika", []byte(strconv.Itoa(sinirDakika(dakika))))
}

// sinirDakika hızı geçerli aralığa sıkıştırır.
func sinirDakika(n int) int {
	if n <= 0 {
		return VarsayilanDakika
	}
	if n > MaksDakika {
		return MaksDakika
	}
	return n
}

// Ceviri AI'nın döndürdüğü çevrilmiş alanlardır.
type Ceviri struct {
	Baslik      string
	Spot        string
	Paragraflar []string
	SeoBaslik   string
	SeoAciklama string
	Provider    string
	Model       string
}

// eksikIcerik çevirisi olmayan yayımlanmış bir içeriktir.
type eksikIcerik struct {
	ID    int64
	Slug  string
	Title string
}

// Cevrilecekler verilen dilde çevirisi olmayan yayımlanmış içerikleri
// (en yeniden eskiye) döndürür.
//
// Neden yalnız yayımlanmış + kaynak dil tr: taslak bir haberin çevirisini
// yayına almak, editörün henüz bitirmediği metni yayımlamak olurdu.
func Cevrilecekler(ctx context.Context, pool *pgxpool.Pool, siteID int64, hedef string, limit int) ([]eksikIcerik, error) {
	if limit <= 0 {
		limit = TurLimiti
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, COALESCE(c.slug,''), c.title
		FROM contents c
		WHERE c.status = 'yayinda'
		  AND c.locale = 'tr'
		  AND c.site_id = $1
		  AND COALESCE(c.title,'') <> ''
		  AND NOT EXISTS (
			SELECT 1 FROM contents t
			WHERE t.locale = $2
			  AND (
				t.translation_of = COALESCE(c.translation_of, c.id)
				OR t.id = c.id
			  )
		  )
		  -- Editör aynı içeriği istediyse otomatik tur onu tekrar
		  -- çevirmesin: iki kez AI maliyeti ve yarış olurdu.
		  --
		  -- Kök üzerinden bakılır: editör çeviri kaydını (locale=en)
		  -- düzenleyip başka dil isterse istek satırı kökü değil o kaydı
		  -- gösterir; düz content_id karşılaştırması bunu kaçırırdı.
		  AND NOT EXISTS (
			SELECT 1
			FROM ceviri_istek i
			JOIN contents ic ON ic.id = i.content_id
			WHERE COALESCE(ic.translation_of, ic.id) = COALESCE(c.translation_of, c.id)
			  AND i.locale = $2
			  AND i.durum IN ('bekliyor','calisiyor')
		  )
		ORDER BY c.published_at DESC NULLS LAST, c.id DESC
		LIMIT $3`, siteID, hedef, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []eksikIcerik
	for rows.Next() {
		var e eksikIcerik
		if err := rows.Scan(&e.ID, &e.Slug, &e.Title); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// BekleyenSayisi kuyruk derinliğini verir (panel göstergesi).
func BekleyenSayisi(ctx context.Context, pool *pgxpool.Pool, siteID int64, hedef string) int {
	var n int
	_ = pool.QueryRow(ctx, `
		SELECT count(*) FROM contents c
		WHERE c.status='yayinda' AND c.locale='tr' AND c.site_id=$1
		  AND NOT EXISTS (
			SELECT 1 FROM contents t
			WHERE t.locale=$2 AND t.translation_of = COALESCE(c.translation_of, c.id)
		  )`, siteID, hedef).Scan(&n)
	return n
}

// ceviriJSON model yanıtını çözer (```json sarmasını da kabul eder).
func ceviriJSON(ham string) (map[string]any, error) {
	ham = strings.TrimSpace(ham)
	if i := strings.Index(ham, "{"); i > 0 {
		ham = ham[i:]
	}
	if j := strings.LastIndex(ham, "}"); j >= 0 && j < len(ham)-1 {
		ham = ham[:j+1]
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(ham), &m); err != nil {
		return nil, fmt.Errorf("çeviri yanıtı JSON değil: %w", err)
	}
	return m, nil
}

// Uret bir içeriği hedef dile çevirir.
// Gövde çevirisi parçalara bölünür.
//
// Neden: canlı ölçümde 28 paragraflık bir haberin çeviri yanıtı 1200 token
// sınırını aştı ve yanıt ORTASINDAN kesildi ("unexpected end of JSON
// input"); iki deneme de başarısız oldu, içerik çevirisiz kaldı.
//
// Sınır PARAGRAF SAYISI değil KARAKTER: paragraflar çok farklı uzunlukta.
// Yalnız sayıya bakmak yetmedi — 6 uzun paragraf (≈1000 token kaynak)
// İngilizceye genişleyince 1200 tokenlık yanıtı yine doldurdu (ölçüm:
// completion_tokens=1200, iki deneme). 2200 karakter ≈ 600 token kaynak,
// çeviri ~800 token: 1600'lük sınırın altında güvenli.
const (
	parcaKarakter = 2200
	parcaBoyu     = 6 // kısa paragraflarda istek sayısı artmasın
)

// parcalaraBol gövdeyi karakter ve paragraf sınırlarına göre böler.
//
// Her parça en az bir paragraf içerir (tek paragraf sınırı aşsa bile):
// aksi hâlde döngü ilerlemez.
func parcalaraBol(p []string) [][]string {
	var out [][]string
	var grup []string
	uzunluk := 0
	for _, par := range p {
		if len(grup) > 0 && (len(grup) >= parcaBoyu || uzunluk+len(par) > parcaKarakter) {
			out = append(out, grup)
			grup = nil
			uzunluk = 0
		}
		grup = append(grup, par)
		uzunluk += len(par)
	}
	if len(grup) > 0 {
		out = append(out, grup)
	}
	return out
}

// Uret bir içeriği hedef dile çevirir.
//
// Başlık + spot tek istekte; gövde PARÇALI çevrilir ve sıra korunarak
// birleştirilir. Her parça kendi isteği olduğu için token sınırı aşılmaz.
func Uret(ctx context.Context, aiclient *ai.Client, c repo.Content, hedef string) (Ceviri, error) {
	if aiclient == nil || !aiclient.Enabled() {
		return Ceviri{}, errors.New("AI sağlayıcısı yapılandırılmadı")
	}
	p := Paragraflar(c.Detail)
	if len(p) == 0 && strings.TrimSpace(c.Title) == "" {
		return Ceviri{}, errors.New("çevrilecek metin yok")
	}
	dilAdiMetin := dilAdi(hedef)

	// 1) Başlık + spot
	out, err := ceviriBaslik(ctx, aiclient, c, dilAdiMetin)
	if err != nil {
		return out, err
	}

	// 2) Gövde: karakter sınırına göre parçalar, SIRA korunarak birleştirilir.
	if len(p) > 0 {
		var tumu []string
		for _, grup := range parcalaraBol(p) {
			parca, err := ceviriParcaGuvenli(ctx, aiclient, grup, dilAdiMetin)
			if err != nil {
				return out, err
			}
			if len(parca) != len(grup) {
				return out, fmt.Errorf("parça paragraf sayısı uyuşmuyor: beklenen %d, gelen %d",
					len(grup), len(parca))
			}
			tumu = append(tumu, parca...)
		}
		out.Paragraflar = tumu
	}
	return out, nil
}

// ceviriBaslik başlık + spot + SEO alanlarını tek istekte çevirir.
func ceviriBaslik(ctx context.Context, aiclient *ai.Client, c repo.Content, dil string) (Ceviri, error) {
	istek, _ := json.Marshal(map[string]any{
		"title": strings.TrimSpace(c.Title),
		"spot":  strings.TrimSpace(c.Spot),
	})
	resp, err := aiclient.CompleteStrict(ctx, ai.Request{
		Messages: []ai.Msg{
			{Role: "system", Content: ceviriSystem(dil)},
			{Role: "user", Content: ceviriInstruksiyonBaslik(dil) + "\n\n---\n" + string(istek)},
		},
		JSONMode: true,
		// 500 token muhakeme tabanlı modellerde YETMİYOR: model bütçenin
		// tamamını düşünmeye harcayıp JSON'u yarıda kesiyordu (canlı ölçüm
		// 6 Kasım 2026: "çeviri yanıtı JSON değil: unexpected end of JSON
		// input"). Başlık + spot + SEO + özet rahat sığsın diye 1200.
		MaxTokens:   1200,
		Temperature: 0.2,
	})
	if err != nil {
		return Ceviri{}, err
	}
	m, err := ceviriJSON(resp.Text)
	if err != nil {
		return Ceviri{}, err
	}
	out := Ceviri{
		Baslik:      temiz(metin(m["title"])),
		Spot:        temiz(metin(m["spot"])),
		SeoBaslik:   metin(m["seo_title"]),
		SeoAciklama: metin(m["meta_description"]),
		Provider:    resp.Provider,
		Model:       resp.Model,
	}
	if out.Baslik == "" {
		return out, errors.New("çeviri başlığı boş")
	}
	return out, nil
}

// ceviriParcaGuvenli bir grubu çevirir; yanıt kesilirse grubu ikiye bölüp
// yeniden dener.
//
// Neden: canlı ölçümde 2200 karakterlik bir parça bile 1600 tokenlık yanıtı
// doldurdu (completion_tokens=1600 → "unexpected end of JSON input"). Bu
// durumda TÜM makaleyi baştan çevirmek hem pahalı hem yavaştı (ölçüm: 3,5 dk
// boşa gitti). Yerel bölme, hatayı o parçayla sınırlar.
//
// Tek paragraf da kesilirse yapılacak bir şey yok: hata yukarı çıkar.
func ceviriParcaGuvenli(ctx context.Context, aiclient *ai.Client, grup []string, dil string) ([]string, error) {
	out, err := ceviriParca(ctx, aiclient, grup, dil)
	if err == nil && len(out) == len(grup) {
		return out, nil
	}
	if len(grup) <= 1 {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("parça paragraf sayısı uyuşmuyor: beklenen %d, gelen %d",
			len(grup), len(out))
	}
	orta := len(grup) / 2
	sol, err := ceviriParcaGuvenli(ctx, aiclient, grup[:orta], dil)
	if err != nil {
		return nil, err
	}
	sağ, err := ceviriParcaGuvenli(ctx, aiclient, grup[orta:], dil)
	if err != nil {
		return nil, err
	}
	return append(sol, sağ...), nil
}

// ceviriParca bir paragraf grubunu çevirir.
func ceviriParca(ctx context.Context, aiclient *ai.Client, parcalar []string, dil string) ([]string, error) {
	istek, _ := json.Marshal(map[string]any{"paragraphs": parcalar})
	resp, err := aiclient.CompleteStrict(ctx, ai.Request{
		Messages: []ai.Msg{
			{Role: "system", Content: ceviriSystem(dil)},
			{Role: "user", Content: ceviriInstruksiyonGovde(dil) + "\n\n---\n" + string(istek)},
		},
		JSONMode: true,
		// 2200 karakterlik kaynak ≈ 800 token İngilizce üretim; 1600 sınırı
		// kesilmeye karşı pay bırakır (ölçüm: 1200 yetmedi). Muhakeme tabanlı
		// modeller bütçenin bir kısmını düşünmeye harcadığı için 2600.
		MaxTokens:   2600,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}
	m, err := ceviriJSON(resp.Text)
	if err != nil {
		return nil, err
	}
	return metinListesi(m["paragraphs"]), nil
}

// ceviriSystem ortak sistem istemidir.
func ceviriSystem(dil string) string {
	return "Sen deneyimli bir Türkçe-" + dil + " haber çevirmenisin. " +
		"Yanıtın SADECE geçerli JSON olsun."
}

// ceviriInstruksiyonBaslik başlık/spot kuralıdır.
func ceviriInstruksiyonBaslik(dil string) string {
	return `Verilen Türkçe haber başlığını ve spotunu ` + strings.ToUpper(dil) + ` diline çevir.
Kurallar:
- Haber dilini koru; akıcı, doğal çeviri yaz. Kelime kelime çeviri yapma.
- Kişi, kurum, yer, tarih ve sayıları DEĞİŞTİRME.
- Metne yeni bilgi EKLEME, çıkarma, yorum katma.
- Başlığı tıklama tuzağına kaçmadan haber başlığı olarak çevir.
- seo_title 30-60 karakter, meta_description 110-155 karakter olsun.
Şu JSON şemasını döndür:
{"title":"","spot":"","seo_title":"","meta_description":""}`
}

// ceviriInstruksiyonGovde gövde kuralıdır.
//
// Paragraf sayısının korunması ZORUNLU: sunucu sıra eşleşmesini doğrular ve
// uyuşmazsa kaydı reddeder (yanlış sıralı gövde yayına girmemeli).
func ceviriInstruksiyonGovde(dil string) string {
	return `Verilen Türkçe haber paragraflarını ` + strings.ToUpper(dil) + ` diline çevir.
Kurallar:
- Haber dilini koru; akıcı, doğal çeviri yaz. Kelime kelime çeviri yapma.
- Kişi, kurum, yer, tarih, sayı ve ölçü birimlerini DEĞİŞTİRME.
- Metne yeni bilgi EKLEME, çıkarma, yorum katma.
- paragraphs dizisi, verilen paragraf sayısıyla AYNI uzunlukta olmalı; her
  öğe karşılık gelen paragrafın çevirisidir. Paragraf BİRLEŞTİRME veya BÖLME.
- Tırnak, satır sonu ve emoji kullanma.
Şu JSON şemasını döndür:
{"paragraphs":[""]}`
}

// dilAdi model prompt'unda kullanılacak dil adı.
func dilAdi(kod string) string {
	switch kod {
	case "en":
		return "İngilizce"
	case "de":
		return "Almanca"
	case "fr":
		return "Fransızca"
	case "es":
		return "İspanyolca"
	case "ar":
		return "Arapça"
	}
	return kod
}

func metin(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func metinListesi(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, it := range arr {
		out = append(out, metin(it))
	}
	return out
}

// temiz tek satıra indirir (başlık/spot alanları için).
func temiz(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			return ' '
		case r < 32 || r == 127:
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// Paragraflar detail (blok dizisi veya TipTap) içinden düz paragrafları
// çıkarır. Sıra korunur: çeviri aynı sırayla geri gelir.
func Paragraflar(detail []byte) []string {
	if len(detail) == 0 {
		return nil
	}
	var bloklar []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(detail, &bloklar); err == nil && len(bloklar) > 0 {
		out := make([]string, 0, len(bloklar))
		for _, b := range bloklar {
			out = append(out, b.Text)
		}
		return out
	}
	// TipTap dokümanı: metin düğümlerini sırayla topla.
	var doc struct {
		Type    string `json:"type"`
		Content []any  `json:"content"`
	}
	if err := json.Unmarshal(detail, &doc); err == nil && doc.Type == "doc" {
		var out []string
		for _, blok := range doc.Content {
			if s := tiptapBlokMetni(blok); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// tiptapBlokMetni bir TipTap bloğunun düz metnini verir.
func tiptapBlokMetni(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	var sb strings.Builder
	var yuru func(any)
	yuru = func(node any) {
		switch t := node.(type) {
		case map[string]any:
			if s, ok := t["text"].(string); ok {
				sb.WriteString(s)
			}
			if ch, ok := t["content"].([]any); ok {
				for _, c := range ch {
					yuru(c)
				}
			}
		case []any:
			for _, c := range t {
				yuru(c)
			}
		}
	}
	if ch, ok := m["content"].([]any); ok {
		for _, c := range ch {
			yuru(c)
		}
	}
	return strings.TrimSpace(sb.String())
}

// Bloklar çevrilmiş paragrafları detail JSON'una çevirir.
//
// Girdi biçimi (blok dizisi) korunur: site şablonu bu biçimi bekliyor.
func Bloklar(paragraflar []string) []byte {
	bloklar := make([]map[string]any, 0, len(paragraflar))
	for _, p := range paragraflar {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		bloklar = append(bloklar, map[string]any{"type": "p", "text": p})
	}
	b, _ := json.Marshal(bloklar)
	return b
}

// Kaydet çeviriyi yeni bir içerik olarak yazar.
//
// translation_of kaynak içeriğin ÇEVİRİ KÖKÜNÜ gösterir; böylece bir
// haberin tüm dilleri tek zincirde toplanır (kanonik/hreflang bu zinciri
// kullanır).
func Kaydet(ctx context.Context, pool *pgxpool.Pool, kaynak repo.Content, hedef string, c Ceviri, durum string) (int64, error) {
	if kaynak.ID <= 0 {
		return 0, errors.New("geçersiz kaynak içerik")
	}
	if c.Baslik == "" {
		return 0, errors.New("çeviri başlığı boş")
	}
	kok := kaynak.TranslationRootID()

	// Paragraflar kaynakla aynı sayıda değilse (model eksik döndürdü)
	// kaynak paragrafların sayısını korumak için eksikleri boş bırakırız:
	// sıra bozulursa gövde anlamsız karışır.
	kaynakP := Paragraflar(kaynak.Detail)
	hedefP := c.Paragraflar
	if len(kaynakP) > 0 && len(hedefP) != len(kaynakP) {
		return 0, fmt.Errorf("paragraf sayısı uyuşmuyor: kaynak %d, çeviri %d", len(kaynakP), len(hedefP))
	}

	slug := platform.Slugify(c.Baslik)
	if slug == "" {
		slug = fmt.Sprintf("haber-%d", kaynak.ID)
	}
	// Paralel adres: aynı kaynak id'si ile biten slug (dil öneki ayrı).
	suffix := fmt.Sprintf("-%d", kaynak.ID)
	if !strings.HasSuffix(slug, suffix) {
		slug += suffix
	}

	// Referanslar DOĞRULANIR: silinmiş kategori/medya/yazar çeviri kaydını
	// FK hatasıyla patlatıyordu.
	//
	// Sahadan gelen gerçek vaka: `contents_category_id_fkey` ihlali, kullanıcıya
	// ham SQL hatası olarak gösteriliyordu ("insert or update on table
	// contents violates foreign key constraint"). Aynı hata arka plan
	// kuyruğunda da kaydı sürekli düşürüyordu (3 denemeden sonra atlanıyordu).
	catID := gecerliKategori(ctx, pool, kaynak.CategoryID)
	mediaID := gecerliMedya(ctx, pool, kaynak.CoverMediaID)
	mansetID := gecerliMedya(ctx, pool, kaynak.MansetMediaID)
	authorID := gecerliYazar(ctx, pool, kaynak.AuthorID)

	nc := &repo.Content{
		TypeKey:       kaynak.TypeKey,
		Status:        durum,
		Title:         c.Baslik,
		Spot:          c.Spot,
		Detail:        Bloklar(hedefP),
		Slug:          &slug,
		CategoryID:    catID,
		AuthorID:      authorID,
		CoverMediaID:  mediaID,
		MansetMediaID: mansetID,
		SourceURL:     kaynak.SourceURL,
		SourceName:    kaynak.SourceName,
		CommentStatus: "kapali",
		Locale:        hedef,
		TranslationOf: &kok,
		SiteID:        kaynak.SiteID,
	}
	// Faz 193c — SEO alanları çeviri yolunda da sınırlanır.
	//
	// ÖLÇÜM: 31 çeviride seo_title 70, 71'inde meta açıklama 165 karakteri
	// aşıyordu (en uzun 311). `ceviriBaslik` istemde 30-60 istiyor ama yanıtı
	// doğrulamıyordu; kırpma tek yerde (platform.SeoKirp) yapılır.
	if s := platform.SeoKirp(temiz(c.SeoBaslik), platform.SeoBaslikSertMax); s != "" {
		nc.SeoTitle = &s
	}
	if s := platform.SeoKirp(temiz(c.SeoAciklama), platform.SeoAciklamaSertMax); s != "" {
		nc.SeoDesc = &s
	}
	if durum == "yayinda" {
		now := time.Now()
		nc.PublishedAt = &now
	}
	return repo.CreateContent(ctx, pool, nc)
}

// UretTekrar çeviriyi en çok iki kez dener.
//
// Neden: sağlayıcı yanıtları zaman zaman token sınırında kesiliyor
// ("unexpected end of JSON input", izole logunda görüldü). Tek denemede
// bırakmak içeriği kalıcı çevirisiz bırakıyordu.
func UretTekrar(ctx context.Context, aiclient *ai.Client, c repo.Content, hedef string, logger *slog.Logger) (Ceviri, error) {
	cev, err := Uret(ctx, aiclient, c, hedef)
	if err == nil {
		return cev, nil
	}
	if ctx.Err() != nil {
		return Ceviri{}, err
	}
	logger.Info("ceviri: yeniden deneniyor", "icerik", c.ID, "dil", hedef, "ilk_hata", err.Error())
	return Uret(ctx, aiclient, c, hedef)
}

// hataSayaci kalıcı başarısız içerikleri işaretler.
//
// Neden gerekli: kayıt FK/veri hatasıyla sürekli düşerse kuyruğun başında
// kalır ve HER turda yeniden denenir (her deneme AI maliyeti). Üç
// denemeden sonra atlanır; süreç yeniden başlarsa sayaç sıfırlanır.
type hataSayaci struct {
	mu   sync.Mutex
	sayi map[int64]int
}

func (h *hataSayaci) asildi(id int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sayi[id] >= 3
}

func (h *hataSayaci) basarisiz(id int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sayi[id]++
}

func (h *hataSayaci) basarili(id int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sayi, id)
}

// gecerliKategori kimlik mevcutsa döndürür, değilse nil.
//
// Silinmiş kategoriye işaret eden bir kaynak, çeviri kaydını FK hatasıyla
// patlatıyordu. Kategoriyi boş bırakmak çeviriyi kurtarır; editör sonradan
// atayabilir.
func gecerliKategori(ctx context.Context, pool *pgxpool.Pool, id *int64) *int64 {
	if id == nil || *id <= 0 || pool == nil {
		return nil
	}
	var bir int
	if err := pool.QueryRow(ctx, `SELECT 1 FROM categories WHERE id=$1`, *id).Scan(&bir); err != nil {
		return nil
	}
	return id
}

// gecerliMedya kimlik mevcutsa döndürür, değilse nil.
func gecerliMedya(ctx context.Context, pool *pgxpool.Pool, id *int64) *int64 {
	if id == nil || *id <= 0 || pool == nil {
		return nil
	}
	var bir int
	if err := pool.QueryRow(ctx, `SELECT 1 FROM media_files WHERE id=$1`, *id).Scan(&bir); err != nil {
		return nil
	}
	return id
}

// gecerliYazar kimlik mevcutsa döndürür; yoksa 1 (sistem yazarı).
//
// AuthorID NOT NULL: nil verilemez, bu yüzden geçerli bir yedek şart.
func gecerliYazar(ctx context.Context, pool *pgxpool.Pool, id int64) int64 {
	if pool == nil {
		return 1
	}
	var bir int
	if err := pool.QueryRow(ctx, `SELECT 1 FROM users WHERE id=$1`, id).Scan(&bir); err != nil {
		return 1
	}
	return id
}

// Calistir bir tur çeviri yapar ve çevrilen sayısını döndürür.
func Calistir(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, siteID int64, a Ayarlar, limit int) int {
	return calistir(ctx, pool, aiclient, logger, siteID, a, limit, &hataSayaci{sayi: map[int64]int{}})
}

func calistir(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, siteID int64, a Ayarlar, limit int, hata *hataSayaci) int {
	if !a.Aktif || aiclient == nil || !aiclient.Enabled() {
		return 0
	}
	if limit <= 0 {
		limit = TurLimiti
	}
	yapilan := 0
	for _, dil := range a.Diller {
		if ctx.Err() != nil {
			return yapilan
		}
		liste, err := Cevrilecekler(ctx, pool, siteID, dil, limit)
		if err != nil {
			logger.Warn("ceviri: kuyruk okunamadı", "dil", dil, "error", err)
			continue
		}
		if len(liste) == 0 {
			continue
		}
		yapilan += tur(ctx, pool, aiclient, logger, dil, liste, a.Durum, hata)
	}
	return yapilan
}

// tur tek bir dil için listeyi eşzamanlı çevirir.
func tur(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, dil string, liste []eksikIcerik, durum string, hata *hataSayaci) int {
	sema := make(chan struct{}, VarsayilanEszamanli)
	sonuc := make(chan bool, len(liste))
	baslatilan := 0
	for _, e := range liste {
		if ctx.Err() != nil {
			break
		}
		if hata != nil && hata.asildi(e.ID) {
			continue
		}
		sema <- struct{}{}
		baslatilan++
		go func(e eksikIcerik) {
			defer func() { <-sema }()
			c, err := repo.GetContent(ctx, pool, e.ID)
			if err != nil {
				logger.Warn("ceviri: içerik okunamadı", "içerik", e.ID, "error", err)
				sonuc <- false
				return
			}
			cev, err := UretTekrar(ctx, aiclient, c, dil, logger)
			if err != nil {
				logger.Warn("ceviri: üretilemedi", "içerik", e.ID, "dil", dil, "error", err)
				if hata != nil {
					hata.basarisiz(e.ID)
				}
				sonuc <- false
				return
			}
			id, err := Kaydet(ctx, pool, c, dil, cev, durum)
			if err != nil {
				logger.Warn("ceviri: kaydedilemedi", "içerik", e.ID, "dil", dil, "error", err)
				if hata != nil {
					hata.basarisiz(e.ID)
				}
				sonuc <- false
				return
			}
			if hata != nil {
				hata.basarili(e.ID)
			}
			logger.Info("ceviri: yazıldı", "kaynak", e.ID, "yeni", id, "dil", dil,
				"durum", durum, "sağlayıcı", cev.Provider, "model", cev.Model)
			sonuc <- true
		}(e)
	}
	n := 0
	for i := 0; i < baslatilan; i++ {
		select {
		case ok := <-sonuc:
			if ok {
				n++
			}
		case <-ctx.Done():
			return n
		}
	}
	return n
}

// istekEsZamanli, editör isteklerini işleyen eşzamanlı işçi sayısıdır.
//
// Neden 1 değil: ölçümde tek işçi, uzun süren (ya da sağlayıcı hatasıyla
// yeniden denenen) bir isteğin arkasında yeni isteği 6 dakika bekletti.
// Neden 3+ değil: her işçi ayrı AI çağrısı yapar; otomatik kuyruk zaten 3
// eşzamanlı çalışıyor, sağlayıcıyı daha fazla zorlamak hataları artırır.
const istekEsZamanli = 2

// Bekleme hedef tempoyu tutturmak için beklenecek süredir.
func Bekleme(dakika, islenen int, gecen time.Duration) time.Duration {
	if islenen <= 0 {
		return 0
	}
	hedef := time.Duration(float64(time.Minute) * float64(islenen) / float64(sinirDakika(dakika)))
	if hedef <= gecen {
		return 0
	}
	return hedef - gecen
}

// Run sürekli çeviri döngüsü.
func Run(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, siteID int64) {
	hata := &hataSayaci{sayi: map[int64]int{}}
	logger.Info("ceviri başladı", "varsayılan_dakika", VarsayilanDakika)
	// Editör istekleri AYRI bir döngüde işlenir.
	//
	// Neden aynı döngüde değil: otomatik tur dakikalarca sürebiliyor (3
	// eşzamanlı AI çağrısı, uzun haberlerde 120 sn zaman aşımı). Aynı
	// döngüde işlendiğinde editörün isteği tur bitene kadar "bekliyor"
	// kalıyordu (ölçüm: 2 dk sonra hâlâ bekliyor). İki iş birbirini
	// beklememeli.
	go istekDongusu(ctx, pool, aiclient, logger)

	for {
		if ctx.Err() != nil {
			logger.Info("ceviri durdu")
			return
		}
		aiVar := aiclient != nil && aiclient.Enabled()

		a := AyarlariOku(ctx, pool)
		if !a.Aktif || !aiVar {
			if !bekle(ctx, time.Minute) {
				return
			}
			continue
		}
		bas := time.Now()
		n := calistir(ctx, pool, aiclient, logger, siteID, a, TurLimiti, hata)
		if n > 0 {
			logger.Info("ceviri turu", "cevrilen", n, "dakika_hedefi", a.Dakika)
		}
		if sure := Bekleme(a.Dakika, n, time.Since(bas)); sure > 0 {
			if !bekle(ctx, sure) {
				return
			}
		} else if n == 0 {
			if !bekle(ctx, time.Minute) {
				return
			}
		}
	}
}

// istekDongusu editör çeviri isteklerini yoklar ve işler.
//
// Ayrı goroutine: otomatik tur ne kadar sürerse sürsün editör isteği
// beklemez.
//
// EŞZAMANLILIK: tek tüketici yetmedi. Ölçüm: 28 paragraflık bir haberin
// isteği 3-4 dk sürerken ya da sağlayıcı hatasıyla yeniden denerken,
// kuyruğa yeni gelen istek 6 dakika boyunca `bekliyor` kaldı (deneyde
// 361 sn). İki işçi, iki editörün birbirini beklemesini engeller; üst sınır
// bilinçli olarak küçük tutulur, sağlayıcı yükü artmasın.
func istekDongusu(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger) {
	// Süreç çeviri ortasında öldüyse "calisiyor" satırı kalır; panel
	// sonsuza kadar yoklar. Başlangıçta geri alınır.
	if pool != nil {
		if _, err := pool.Exec(ctx,
			`UPDATE ceviri_istek SET durum = 'bekliyor', updated_at = now()
			  WHERE durum = 'calisiyor'`); err != nil {
			logger.Warn("ceviri: yarım kalan istekler sıfırlanamadı", "error", err)
		}
	}
	go bayatIstekTemizle(ctx, pool, logger)
	for i := 0; i < istekEsZamanli; i++ {
		go istekIsci(ctx, pool, aiclient, logger)
	}
}

// istekIsci tek bir tüketicidir: sıradaki isteği alır, işler, tekrarlar.
//
// Boş kuyrukta 2 sn bekler. `FOR UPDATE SKIP LOCKED` sayesinde iki işçi
// (ya da iki süreç) aynı satırı almaz.
func istekIsci(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger) {
	for {
		if ctx.Err() != nil {
			return
		}
		if aiclient == nil || !aiclient.Enabled() {
			if !bekle(ctx, 2*time.Second) {
				return
			}
			continue
		}
		is, err := istekAl(ctx, pool)
		if err != nil {
			logger.Warn("ceviri: istek alınamadı", "error", err)
			if !bekle(ctx, 5*time.Second) {
				return
			}
			continue
		}
		if is == nil {
			if !bekle(ctx, 2*time.Second) {
				return
			}
			continue
		}
		istekIsle(ctx, pool, aiclient, logger, *is, "taslak")
	}
}

// bayatIstekTemizle, sahibi ölmüş "calisiyor" istekleri geri alır.
//
// Neden: başlangıçtaki sıfırlama yalnız süreç YENİDEN BAŞLARKEN çalışır.
// İşçi goroutine'i panikle ölürse (ya da işlem `calisiyor` yazdıktan sonra
// kesilirse) satır sonsuza kadar "çalışıyor" kalır ve panel hiç durmadan
// yoklar. 20 dakika, en uzun çeviriden (ölçüm: 231 sn) kat kat uzundur;
// dolayısıyla canlı bir işin altından satır çalınmaz.
func bayatIstekTemizle(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	if pool == nil {
		return
	}
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		etiket, err := pool.Exec(ctx, `
			UPDATE ceviri_istek
			   SET durum = 'bekliyor', updated_at = now()
			 WHERE durum = 'calisiyor'
			   AND updated_at < now() - interval '20 minutes'`)
		if err != nil {
			logger.Warn("ceviri: bayat istek temizlenemedi", "error", err)
			continue
		}
		if n := etiket.RowsAffected(); n > 0 {
			logger.Warn("ceviri: bayat istekler geri alındı", "adet", n)
		}
	}
}

func bekle(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
