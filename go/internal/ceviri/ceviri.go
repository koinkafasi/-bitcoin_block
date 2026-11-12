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

const (
	parcaKarakter = 2200
	parcaBoyu     = 6 // kısa paragraflarda istek sayısı artmasın
)

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
		JSONMode:    true,
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
		JSONMode:    true,
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

func Kaydet(ctx context.Context, pool *pgxpool.Pool, kaynak repo.Content, hedef string, c Ceviri, durum string) (int64, error) {
	if kaynak.ID <= 0 {
		return 0, errors.New("geçersiz kaynak içerik")
	}
	if c.Baslik == "" {
		return 0, errors.New("çeviri başlığı boş")
	}
	kok := kaynak.TranslationRootID()

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
