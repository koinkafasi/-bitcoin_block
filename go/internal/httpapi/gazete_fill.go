package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
)

// ---- Gazete otomatik doldurma & baskı üretimi (Faz 3/4) ----
//
// Sayfa blokları JSON ile saklanır:
//
//	{id, tip, x, y, w, h, baslik, stil:{...}, kaynak:{tip, id, limit, icerikler},
//	 metin, gorsel, icerik:{haberler:[...]}}
//
// x/y/w/h sayfa yüzdesi (%); otomatik doldurma kaynakları çözer ve "icerik"i
// doldurur. Baskı (PDF) ve flip görüntüleyici aynı blok JSON'undan render eder.

// gazeteArticle bloklara yerleşen haber özeti.
type gazeteArticle struct {
	ID       int64  `json:"id"`
	Baslik   string `json:"baslik"`
	Spot     string `json:"spot"`
	Kategori string `json:"kategori"`
	Kaynak   string `json:"kaynak"`
	Gorsel   string `json:"gorsel"`
	Tarih    string `json:"tarih"`
	URL      string `json:"url"`
}

// gazeteBlock tek sayfa bloğu.
type gazeteBlock struct {
	ID     string  `json:"id"`
	Tip    string  `json:"tip"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	W      float64 `json:"w"`
	H      float64 `json:"h"`
	Baslik string  `json:"baslik"`
	// Z boyama sirasi; 0 en altta. Eski kayitlarda yok, o zaman dizi
	// sirasi kullanilir (gazeteBloklariSirala).
	Z      int             `json:"z"`
	Stil   map[string]any  `json:"stil"`
	Kaynak *gazeteKaynak   `json:"kaynak"`
	Metin  string          `json:"metin"`
	Gorsel string          `json:"gorsel"`
	Icerik *gazeteIcerik   `json:"icerik"`
	Raw    json.RawMessage `json:"-"`
}

// gazeteKaynak bloğun haber kaynağını tanımlar.
type gazeteKaynak struct {
	Tip       string  `json:"tip"` // son | kategori | manset | sabit
	ID        int64   `json:"id"`
	Limit     int     `json:"limit"`
	Icerikler []int64 `json:"icerikler"`
}

// gazeteIcerik doldurma sonucu (bloğa yerleşen haberler).
type gazeteIcerik struct {
	Haberler []gazeteArticle `json:"haberler"`
}

// gazetePageDoc tek sayfa dokümanı.
type gazetePageDoc struct {
	No     int           `json:"no"`
	Ad     string        `json:"ad"`
	Blocks []gazeteBlock `json:"bloklar"`
}

// defaultGazetePages 10 sayfalık hazır gazete düzeni (şablon yoksa).
func defaultGazetePages() []gazetePageDoc {
	blk := func(id, tip string, x, y, w, h float64, baslik string, k *gazeteKaynak) gazeteBlock {
		return gazeteBlock{ID: id, Tip: tip, X: x, Y: y, W: w, H: h, Baslik: baslik, Kaynak: k}
	}
	son := func(n int) *gazeteKaynak { return &gazeteKaynak{Tip: "son", Limit: n} }
	return []gazetePageDoc{
		{No: 1, Ad: "Kapak", Blocks: []gazeteBlock{
			blk("p1m", "manset", 2, 3, 96, 52, "", &gazeteKaynak{Tip: "manset", Limit: 1}),
			blk("p1l", "liste", 2, 57, 46, 40, "GÜNDEM", son(5)),
			blk("p1k", "kart", 52, 57, 46, 19, "", son(1)),
			blk("p1k2", "kart", 52, 78, 46, 19, "", &gazeteKaynak{Tip: "son", Limit: 2}),
			blk("p1kunye", "kunye", 2, 97, 96, 2, "", nil),
		}},
		{No: 2, Ad: "Gündem", Blocks: []gazeteBlock{
			blk("p2b", "baslik", 2, 2, 96, 6, "GÜNDEM", nil),
			blk("p2m", "manset", 2, 9, 47, 55, "", &gazeteKaynak{Tip: "manset", Limit: 1}),
			blk("p2l", "liste", 52, 9, 46, 55, "SON HABERLER", son(7)),
			blk("p2l2", "liste", 2, 66, 96, 32, "GÜNDEMİN ÖZETİ", son(6)),
		}},
		{No: 3, Ad: "Ekonomi", Blocks: []gazeteBlock{
			blk("p3b", "baslik", 2, 2, 96, 6, "EKONOMİ & FİNANS", nil),
			blk("p3l", "liste", 2, 9, 60, 88, "", son(10)),
			blk("p3k", "kart", 65, 9, 33, 28, "PİYASALAR", son(2)),
			blk("p3m", "metin", 65, 39, 33, 20, "NOT", nil),
			blk("p3l2", "liste", 65, 61, 33, 36, "KISA KISA", son(5)),
		}},
		{No: 4, Ad: "Siyaset", Blocks: []gazeteBlock{
			blk("p4b", "baslik", 2, 2, 96, 6, "SİYASET", nil),
			blk("p4m", "manset", 2, 9, 96, 45, "", &gazeteKaynak{Tip: "manset", Limit: 1}),
			blk("p4l", "liste", 2, 56, 47, 42, "MECLİS GÜNDEMİ", son(6)),
			blk("p4l2", "liste", 52, 56, 46, 42, "PARTİLER", son(6)),
		}},
		{No: 5, Ad: "Dünya", Blocks: []gazeteBlock{
			blk("p5b", "baslik", 2, 2, 96, 6, "DÜNYA", nil),
			blk("p5l", "liste", 2, 9, 47, 88, "DIŞ POLİTİKA", son(8)),
			blk("p5l2", "liste", 52, 9, 46, 88, "ULUSLARARASI", son(8)),
		}},
		{No: 6, Ad: "Spor", Blocks: []gazeteBlock{
			blk("p6b", "baslik", 2, 2, 96, 6, "SPOR", nil),
			blk("p6m", "manset", 2, 9, 60, 50, "", &gazeteKaynak{Tip: "manset", Limit: 1}),
			blk("p6l", "liste", 65, 9, 33, 50, "TRANSFER & LİG", son(6)),
			blk("p6l2", "liste", 2, 61, 96, 37, "SPOR GÜNDEMİ", son(7)),
		}},
		{No: 7, Ad: "Kültür-Sanat", Blocks: []gazeteBlock{
			blk("p7b", "baslik", 2, 2, 96, 6, "KÜLTÜR & SANAT", nil),
			blk("p7k", "kart", 2, 9, 31, 42, "", son(1)),
			blk("p7k2", "kart", 34.5, 9, 31, 42, "", &gazeteKaynak{Tip: "son", Limit: 2}),
			blk("p7k3", "kart", 67, 9, 31, 42, "", &gazeteKaynak{Tip: "son", Limit: 3}),
			blk("p7l", "liste", 2, 54, 96, 44, "ETKİNLİKLER", son(6)),
		}},
		{No: 8, Ad: "Teknoloji", Blocks: []gazeteBlock{
			blk("p8b", "baslik", 2, 2, 96, 6, "TEKNOLOJİ & BİLİM", nil),
			blk("p8l", "liste", 2, 9, 60, 88, "YENİLİKLER", son(9)),
			blk("p8k", "kart", 65, 9, 33, 28, "ÖNE ÇIKAN", son(2)),
			blk("p8l2", "liste", 65, 39, 33, 58, "BİLİM", son(6)),
		}},
		{No: 9, Ad: "Yaşam", Blocks: []gazeteBlock{
			blk("p9b", "baslik", 2, 2, 96, 6, "YAŞAM & SAĞLIK", nil),
			blk("p9l", "liste", 2, 9, 47, 88, "GÜNLÜK YAŞAM", son(8)),
			blk("p9l2", "liste", 52, 9, 46, 88, "SAĞLIK", son(8)),
		}},
		{No: 10, Ad: "Arka Kapak", Blocks: []gazeteBlock{
			blk("p10b", "baslik", 2, 2, 96, 6, "GÜNÜN ÖZETİ", nil),
			blk("p10l", "liste", 2, 9, 96, 55, "KAÇIRMAYIN", son(10)),
			blk("p10m", "metin", 2, 66, 47, 30, "KÜNYE", nil),
			blk("p10k", "kart", 52, 66, 46, 30, "YARINKİ GAZETE", son(1)),
		}},
	}
}

// gazeteArticlesFor kaynaktan haberleri çözer (doldurma).
func (s *Server) gazeteArticlesFor(ctx context.Context, siteID int64, k *gazeteKaynak) []gazeteArticle {
	if k == nil {
		return nil
	}
	limit := k.Limit
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	var rows []gazeteArticle
	var err error
	switch k.Tip {
	case "kategori":
		rows, err = s.gazeteQuery(ctx, siteID, &k.ID, limit)
	case "manset":
		rows, err = s.gazeteQuery(ctx, siteID, &k.ID, 1)
		if len(rows) == 0 {
			rows, _ = s.gazeteQuery(ctx, siteID, nil, 1)
		}
	case "sabit":
		for _, cid := range k.Icerikler {
			if a, e := s.gazeteOne(ctx, siteID, cid); e == nil {
				rows = append(rows, a)
			}
			if len(rows) >= limit {
				break
			}
		}
	default: // son
		rows, err = s.gazeteQuery(ctx, siteID, nil, limit)
	}
	_ = err
	return rows
}

// gazeteQuery yayındaki son haberleri kategori filtresiyle döndürür.
func (s *Server) gazeteQuery(ctx context.Context, siteID int64, categoryID *int64, limit int) ([]gazeteArticle, error) {
	q := `
		SELECT c.id, c.title, c.spot, COALESCE(cat.name, ''), COALESCE(c.source_name, ''),
		       COALESCE(m.path, ''), COALESCE(c.published_at, c.created_at), COALESCE(c.slug, '')
		FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN media_files m ON m.id = c.cover_media_id
		WHERE c.site_id=$1 AND c.status='yayinda'
		  AND ($2::bigint IS NULL OR c.category_id = $2)
		  AND c.type_key <> 'e-gazete'
		ORDER BY COALESCE(c.published_at, c.created_at) DESC
		LIMIT $3`
	rows, err := s.pool.Query(ctx, q, siteID, categoryID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gazeteArticle
	for rows.Next() {
		var a gazeteArticle
		var t time.Time
		var slug string
		if err := rows.Scan(&a.ID, &a.Baslik, &a.Spot, &a.Kategori, &a.Kaynak, &a.Gorsel, &t, &slug); err != nil {
			return nil, err
		}
		a.Tarih = t.Format("02.01.2006")
		a.URL = "/haber/" + slug
		out = append(out, a)
	}
	return out, rows.Err()
}

// gazeteOne tek haberi döndürür (sabit içerik blokları için).
func (s *Server) gazeteOne(ctx context.Context, siteID, contentID int64) (gazeteArticle, error) {
	list, err := s.gazeteQueryIDs(ctx, siteID, []int64{contentID})
	if err != nil {
		return gazeteArticle{}, err
	}
	if len(list) == 0 {
		return gazeteArticle{}, repo.ErrNotFound
	}
	return list[0], nil
}

// gazeteQueryIDs id listesinden haberleri getirir (sıra korunur).
func (s *Server) gazeteQueryIDs(ctx context.Context, siteID int64, ids []int64) ([]gazeteArticle, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := `
		SELECT c.id, c.title, c.spot, COALESCE(cat.name, ''), COALESCE(c.source_name, ''),
		       COALESCE(m.path, ''), COALESCE(c.published_at, c.created_at), COALESCE(c.slug, '')
		FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN media_files m ON m.id = c.cover_media_id
		WHERE c.site_id=$1 AND c.id = ANY($2)`
	rows, err := s.pool.Query(ctx, q, siteID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[int64]gazeteArticle{}
	for rows.Next() {
		var a gazeteArticle
		var t time.Time
		var slug string
		if err := rows.Scan(&a.ID, &a.Baslik, &a.Spot, &a.Kategori, &a.Kaynak, &a.Gorsel, &t, &slug); err != nil {
			return nil, err
		}
		a.Tarih = t.Format("02.01.2006")
		a.URL = "/haber/" + slug
		byID[a.ID] = a
	}
	var out []gazeteArticle
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			out = append(out, a)
		}
	}
	return out, rows.Err()
}

// gazeteAutoFill sayının sayfalarını şablondan kurar ve kaynakları doldurur.
// Mevcut elle düzenlenmiş sayfalar korunur; yalnızca boş/eksik sayfalar doldurulur.
func (s *Server) gazeteAutoFill(ctx context.Context, siteID, issueID int64) error {
	iss, err := repo.GetGazeteIssue(ctx, s.pool, issueID)
	if err != nil {
		return err
	}
	// Şablon sayfaları
	pages := s.gazeteSayiSayfaDuzeni(ctx, iss)
	existing, _ := repo.ListGazetePages(ctx, s.pool, issueID)
	have := map[int]bool{}
	for _, p := range existing {
		have[p.PageNo] = true
	}
	for _, doc := range pages {
		if have[doc.No] {
			continue // elle düzenlenmiş sayfaya dokunma
		}
		for i := range doc.Blocks {
			if doc.Blocks[i].ID == "" {
				doc.Blocks[i].ID = fmt.Sprintf("b%d-%d", doc.No, i+1)
			}
			if doc.Blocks[i].Kaynak != nil {
				doc.Blocks[i].Icerik = &gazeteIcerik{Haberler: s.gazeteArticlesFor(ctx, siteID, doc.Blocks[i].Kaynak)}
			}
		}
		raw, err := json.Marshal(doc.Blocks)
		if err != nil {
			return err
		}
		if err := repo.SaveGazetePage(ctx, s.pool, issueID, doc.No, doc.Ad, raw); err != nil {
			return err
		}
	}
	return nil
}

// ---- Blok render (baskı PDF + flip görüntüleyici ortak) ----

// gazeteBlockHTML bloğu baskı HTML'ine çevirir (mutlak konumlu, sayfa yüzdesi).
// gazeteBlockHTML blogu konumlandirilmis kapsayiciyla uretir (baski yolu).
func gazeteBlockHTML(b gazeteBlock) string {
	style := fmt.Sprintf("left:%.2f%%;top:%.2f%%;width:%.2f%%;height:%.2f%%;", b.X, b.Y, b.W, b.H)
	// data-* nitelikleri dogrulama olcumu icin: rapor hangi blogun tastigini
	// soyleyebilmeli. Baskida gorunmezler.
	// Sarmalayici sinifi "gb-t-<tip>"; ic govde siniflariyla (.gb-metin,
	// .gb-liste ...) CARPISMAMALI. Carpisinca icerik kurallari konumlanmis
	// sarmalayiciya da uygulaniyordu: .gb-metin'in column-count'u sabit
	// yukseklikli sarmalayiciyi cok sutunlu kaba cevirip metni alt satir
	// yerine YANA akitiyor, overflow:hidden da kirpiyordu. Uzun metin
	// bloklari baskida sessizce kayboluyordu.
	// Stil jetonlari beyaz listeden gecer; serbest CSS asla enjekte edilmez.
	return `<div class="gb gb-t-` + b.Tip + gazeteStilSinif(b.Stil) +
		`" data-id="` + html.EscapeString(b.ID) +
		`" data-tip="` + html.EscapeString(b.Tip) + `" style="` + style +
		gazeteStilDeger(b.Stil) + `">` + gazeteBlockInnerHTML(b) + `</div>`
}

// gazeteBlockInnerHTML blogun yalniz ic govdesini uretir.
// Editor kendi kapsayicisini kurup tutamak eklediği için ici ayrica gerekiyor;
// boylece editor ile baski ayni govdeyi gosterir (SPEC K1 - tek render yolu).
func gazeteBlockInnerHTML(b gazeteBlock) string {
	var inner string
	switch b.Tip {
	case "baslik":
		inner = `<div class="gb-baslik">` + html.EscapeString(b.Baslik) + `</div>`
	case "manset":
		if b.Icerik != nil && len(b.Icerik.Haberler) > 0 {
			a := b.Icerik.Haberler[0]
			inner = `<div class="gb-manset">`
			if a.Gorsel != "" {
				inner += `<img src="` + html.EscapeString(mediaAbs(a.Gorsel)) + `" alt="">`
			}
			inner += `<h1>` + html.EscapeString(a.Baslik) + `</h1><p>` + html.EscapeString(a.Spot) + `</p>` +
				`<div class="gb-kucuk">` + html.EscapeString(a.Kategori) + ` · ` + html.EscapeString(a.Kaynak) + ` · ` + html.EscapeString(a.Tarih) + `</div></div>`
		}
	case "liste":
		inner = ""
		if b.Baslik != "" {
			inner += `<div class="gb-baslik gb-baslik-k">` + html.EscapeString(b.Baslik) + `</div>`
		}
		inner += `<ul class="gb-liste">`
		if b.Icerik != nil {
			for _, a := range b.Icerik.Haberler {
				inner += `<li><strong>` + html.EscapeString(a.Baslik) + `</strong><span>` +
					html.EscapeString(firstLine(a.Spot, 120)) + `</span></li>`
			}
		}
		inner += `</ul>`
	case "kart":
		inner = `<div class="gb-kartlar">`
		if b.Icerik != nil {
			for _, a := range b.Icerik.Haberler {
				inner += `<div class="gb-kart">`
				if a.Gorsel != "" {
					inner += `<img src="` + html.EscapeString(mediaAbs(a.Gorsel)) + `" alt="">`
				}
				inner += `<h3>` + html.EscapeString(a.Baslik) + `</h3><p>` + html.EscapeString(firstLine(a.Spot, 100)) + `</p></div>`
			}
		}
		inner += `</div>`
	case "metin":
		inner = ""
		if b.Baslik != "" {
			inner += `<div class="gb-baslik gb-baslik-k">` + html.EscapeString(b.Baslik) + `</div>`
		}
		inner += `<div class="gb-metin">` + html.EscapeString(b.Metin) + `</div>`
	case "gorsel":
		if b.Gorsel != "" {
			inner = `<img class="gb-gorsel" src="` + html.EscapeString(mediaAbs(b.Gorsel)) + `" alt="">`
		}
	case "cizgi":
		inner = `<hr class="gb-cizgi">`
	case "kunye":
		inner = `<div class="gb-kunye">theblock Gazete · Abonelik: /gazete</div>`
	case "reklam":
		inner = `<div class="gb-reklam">REKLAM</div>`
	}
	return inner
}

func mediaAbs(p string) string {
	p = strings.TrimLeft(p, "/")
	if strings.HasPrefix(p, "media/") {
		return "/" + p
	}
	if p == "" {
		return ""
	}
	return "/" + p
}

// firstLine metni tek satira indirip en fazla max KARAKTERE kisaltir.
// Bayt bazli kesme Turkce metinde cok baytli bir harfi ortadan bolup
// baskiya gecersiz UTF-8 yaziyordu; bu yuzden rune bazli sayilir.
func firstLine(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	r := []rune(s)
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// gazeteBloklariSirala bloklari boyama sirasina gore dizer.
//
// Katman paneli ile tuvalin ayni sirayi gostermesi sart: once z, esitlikte
// dizideki yer. Eski kayitlarda z yok (hepsi 0), o durumda dizi sirasi
// korunur, yani gorunum degismez.
func gazeteBloklariSirala(bloklar []gazeteBlock) []gazeteBlock {
	out := make([]gazeteBlock, len(bloklar))
	copy(out, bloklar)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Z < out[j].Z })
	return out
}

// gazetePageHTML tek sayfayı render eder (flip HTML sayfaları için).
func gazetePageHTML(p repo.GazetePage, pageLabel string) string {
	var blocks []gazeteBlock
	_ = json.Unmarshal(p.Blocks, &blocks)
	// Boyama sirasi z'ye gore; ustteki blok sonra basilir (ortusme dogru olur).
	blocks = gazeteBloklariSirala(blocks)
	var sb strings.Builder
	sb.WriteString(`<div class="gazete-sayfa" data-sayfa="` + strconv.Itoa(p.PageNo) +
		`"><div class="gb-kunye-ust">` + html.EscapeString(pageLabel) + `</div>`)
	for _, b := range blocks {
		sb.WriteString(gazeteBlockHTML(b))
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

// gazetePrintHTML tüm baskıyı tek HTML dokümanı olarak üretir (Chromium baskı girişi).
func (s *Server) gazetePrintHTML(ctx context.Context, siteID int64, iss repo.GazeteIssue) (string, error) {
	return s.gazetePrintHTMLOpt(ctx, siteID, iss, false)
}

// gazetePrintHTMLOpt baski dokumanini uretir; olcum true ise sayfaya tasma
// olcum betigini de ekler (yalniz dogrulama kosusunda).
func (s *Server) gazetePrintHTMLOpt(ctx context.Context, siteID int64, iss repo.GazeteIssue, olcum bool) (string, error) {
	pages, err := repo.ListGazetePages(ctx, s.pool, iss.ID)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	for _, p := range pages {
		body.WriteString(gazetePageHTML(p, fmt.Sprintf("Sayfa %d", p.PageNo)))
	}
	title := iss.Title
	if title == "" {
		title = "Gazete — " + iss.IssueDate
	}
	// Sayfa boyu ve tipografi olcegi sablonun ozelligidir (SPEC K2).
	belge := s.gazeteSayiBelgesi(ctx, iss)
	boyut := gazeteBoyut(belge.Sayfa.Boyut)
	tipOlcek := belge.Sayfa.TipOlcek
	return "<!doctype html><html lang=\"tr\"><head><meta charset=\"utf-8\"><title>" +
		html.EscapeString(title) + "</title><style>" +
		gazeteSayfaDegiskenleriTam(boyut, tipOlcek, belge.Sayfa.YaziTipi) + "\n" + gazeteBaskiCSS() +
		"</style></head><body>" + body.String() + olcumBetigi(olcum) + "</body></html>", nil
}

// gazeteSayfaBoyut baskı sayfası ölçüsü ve tipografi tabanı.
// Tipografi tabanı türetilmez, bildirilir (SPEC-GAZETE-DUZENLEYICI.md K2):
// A3 ve A4 gazeteler birbirinin ölçekli kopyası değildir; oransal küçültme
// A4'te okunamayan 6-7pt gövde üretir. Taban 9.5pt iken gazete-baski.css
// çarpanları eski sabit puntolara birebir denk gelir, yani A3 çıktısı bozulmaz.
type gazeteSayfaBoyut struct {
	Ad     string // "A4", "A3"
	W      string // sayfa genişliği, CSS uzunluğu
	H      string // sayfa yüksekliği
	Kenar  string // sayfa iç boşluğu
	TipTab string // gövde metni punto tabanı
}

var gazeteBoyutlar = map[string]gazeteSayfaBoyut{
	"A4": {Ad: "A4", W: "210mm", H: "297mm", Kenar: "5mm", TipTab: "8.5pt"},
	"A3": {Ad: "A3", W: "297mm", H: "420mm", Kenar: "6mm", TipTab: "9.5pt"},
}

// gazeteBoyut adı verilen sayfa boyutunu döner; bilinmeyen ad A4'e düşer.
// Varsayılan A4: referans tasarım ve baskı ihtiyacı A4 (210x297mm).
func gazeteBoyut(ad string) gazeteSayfaBoyut {
	if b, ok := gazeteBoyutlar[strings.ToUpper(strings.TrimSpace(ad))]; ok {
		return b
	}
	return gazeteBoyutlar["A4"]
}

// gazeteBaskiCSSBir gömülü baskı tasarımını bir kez okur.
var gazeteBaskiCSSBir struct {
	bir sync.Once
	css string
}

// gazeteBaskiCSS baskı tasarımını gömülü statik dosyadan okur.
// Chromium baskıyı file:// üzerinden açtığı için /static/... çözümlenemiyor;
// bu yüzden aynı dosya satır içi gömülür. Editör ise onu HTTP ile yükler.
// Tek render yolu (SPEC K1): editörde görünen ile basılan aynı CSS'ten gelir.
func gazeteBaskiCSS() string {
	gazeteBaskiCSSBir.bir.Do(func() {
		b, err := webui.FS.ReadFile("static/gazete-baski.css")
		if err != nil {
			gazeteBaskiCSSBir.css = "/* gazete-baski.css okunamadi: " + err.Error() + " */"
			return
		}
		gazeteBaskiCSSBir.css = string(b)
	})
	return gazeteBaskiCSSBir.css
}

// gazeteSayfaDegiskenleri sayfa kutusunu ve tipografi tabanını bildirir.
// @page içinde custom property güvenilir çalışmadığı için ölçü oraya
// doğrudan yazılır; geri kalan tasarım gazete-baski.css'te.
func gazeteSayfaDegiskenleri(b gazeteSayfaBoyut, tipOlcek float64) string {
	return gazeteSayfaDegiskenleriTam(b, tipOlcek, "")
}

// gazeteSayfaDegiskenleriTam sayfa kutusunu, tipografi tabanini ve yazi
// tipini bildirir. Yazi tipi sunucuda kurulu olanlar arasindan secilir
// (bkz. gazete_yazitipi.go); eskiden Georgia isteniyor ama kurulu olmadigi
// icin sessizce DejaVu Serif'e dusuyordu.
func gazeteSayfaDegiskenleriTam(b gazeteSayfaBoyut, tipOlcek float64, yaziTipi string) string {
	if tipOlcek <= 0 {
		tipOlcek = 1
	}
	// body sifirlamasi yalniz baskida: paylasilan CSS admin panelinde de
	// yuklendigi icin orada genel secici kullanilmiyor (bkz. gazete-baski.css).
	return fmt.Sprintf(
		"@page { size: %s %s; margin: 0; }\n"+
			"body{margin:0}\n"+
			":root{--gz-sayfa-w:%s;--gz-sayfa-h:%s;--gz-kenar:%s;--gz-tip-tab:%s;"+
			"--gz-tip-olcek:%g;--gz-yazi:%s;}",
		b.W, b.H, b.W, b.H, b.Kenar, b.TipTab, tipOlcek, gazeteYaziTipiCSS(yaziTipi))
}

// gazeteBaskiMedyaMutlak baski HTML'indeki /media/... yollarini file:// mutlak
// yoluna cevirir.
//
// Baski dokumani Chromium'a file:// uzerinden veriliyor; orada "/media/x"
// dosya sistemi kokune (file:///media/x) cozuluyordu ve depolama
// {STORAGE_DIR}/media altinda oldugu icin hicbir gorsel yuklenmiyordu --
// yani uretilen gazete PDF'leri gorselsiz cikiyordu. Yalniz baski yolunda
// uygulanir; editor ayni HTML'i HTTP uzerinden aldigi icin orada /media/...
// zaten dogru cozuluyor.
func gazeteBaskiMedyaMutlak(doc string) string {
	kok, err := filepath.Abs(platform.StorageDir())
	if err != nil {
		return doc
	}
	onek := "file://" + filepath.ToSlash(kok) + "/media/"
	return strings.ReplaceAll(doc, `src="/media/`, `src="`+onek)
}

// gazeteSayiBelgesi sayinin sablon belgesini cozer. Sablon yoksa ya da
// cozulemezse varsayilan duzen dondurulur; sayi uretimi bir sablon
// bozuklugu yuzunden durmamali.
func (s *Server) gazeteSayiBelgesi(ctx context.Context, iss repo.GazeteIssue) gazeteSablonDoc {
	vars := gazeteSablonDoc{
		Surum:    1,
		Sayfa:    gazeteSablonVarsayilanSayfa(),
		Izgara:   gazeteSablonVarsayilanIzgara(),
		Sayfalar: defaultGazetePages(),
	}
	if iss.TemplateID == nil {
		return vars
	}
	t, err := repo.GetGazeteTemplate(ctx, s.pool, *iss.TemplateID)
	if err != nil {
		return vars
	}
	d, err := gazeteSablonCoz(t.Pages)
	if err != nil || len(d.Sayfalar) == 0 {
		if err != nil {
			s.logger.Warn("gazete sablonu cozulemedi", "sablon", t.ID, "error", err)
		}
		return vars
	}
	return d
}

// gazeteSayiSayfaDuzeni sayinin uygulayacagi sayfa duzenini dondurur.
func (s *Server) gazeteSayiSayfaDuzeni(ctx context.Context, iss repo.GazeteIssue) []gazetePageDoc {
	return s.gazeteSayiBelgesi(ctx, iss).Sayfalar
}

// gazeteMmYuzde mm olcusunu sayfa genisligine gore yuzdeye cevirir.
// Kilavuz cizgileri blok koordinatlariyla ayni birimde olmali.
func gazeteMmYuzde(mm float64, sayfaOlcusu string) float64 {
	genislik := strings.TrimSuffix(strings.TrimSpace(sayfaOlcusu), "mm")
	f, err := strconv.ParseFloat(genislik, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return mm / f * 100
}

// ---- Tasma dogrulamasi (SPEC K3) ----

// gazeteTasma bir blogun sayfaya sigmadigini bildirir. Olculen degerler
// CSS pikselidir; mm karsiligi oneri uretmek icin kullanilir.
type gazeteTasma struct {
	Sayfa  int     `json:"sayfa"`
	BlokID string  `json:"blok"`
	Tip    string  `json:"tip"`
	Dikey  float64 `json:"dikey"` // px, icerik yuksekligi - kutu yuksekligi
	Yatay  float64 `json:"yatay"` // px
	KutuY  float64 `json:"kutuY"` // px, blogun yuksekligi
	Oneri  string  `json:"oneri"`
}

// gazeteOlcumJS tarayicida calisip tasan bloklari rapor eden betik.
// Sunucu tarafinda olculemez: tasma, yazi tipi ve satir kirilmasi gercekten
// dizildikten sonra belli olur. Bu yuzden baskiyi uretecek olan AYNI
// Chromium'da olculur -- "editorde sigdi ama PDF'te tasti" ancak boyle kapanir.
const gazeteOlcumJS = `<script>
(function(){
  function rapor(){
    var out = [];
    document.querySelectorAll('.gazete-sayfa').forEach(function(sf){
      var no = parseInt(sf.getAttribute('data-sayfa') || '0', 10);
      sf.querySelectorAll('.gb').forEach(function(el){
        var d = el.scrollHeight - el.clientHeight;
        var y = el.scrollWidth - el.clientWidth;
        if (d > 1 || y > 1) {
          out.push({
            sayfa: no,
            blok: el.getAttribute('data-id') || '',
            tip: el.getAttribute('data-tip') || '',
            dikey: Math.round(d * 100) / 100,
            yatay: Math.round(y * 100) / 100,
            kutuY: el.clientHeight
          });
        }
      });
    });
    var d = document.createElement('div');
    d.id = 'gz-tasma-raporu';
    d.textContent = JSON.stringify(out);
    document.body.appendChild(d);
  }
  if (document.readyState === 'complete') { rapor(); }
  else { window.addEventListener('load', rapor); }
})();
</script>`

func olcumBetigi(acik bool) string {
	if !acik {
		return ""
	}
	return gazeteOlcumJS
}

// gazeteTasmaOneri tasma miktarina gore somut bir duzeltme onerir.
// Yuzde cinsinden yukseklik artisi ve alternatif olarak olcek kucultme.
func gazeteTasmaOneri(t gazeteTasma) string {
	if t.KutuY <= 0 {
		return "blogu buyutun veya icerigi kisaltin"
	}
	oran := (t.KutuY + t.Dikey) / t.KutuY
	if t.Dikey <= 0 {
		return "blok genisligini artirin veya icerigi kisaltin"
	}
	return fmt.Sprintf("yuksekligi %%%.0f artirin veya tipografi olcegini %.2f yapin",
		(oran-1)*100, 1/oran)
}

// gazeteDogrula baskiyi olcum betigiyle Chromium'da acar ve tasan bloklari
// dondurur. PDF uretmez; yalniz olcer.
func (s *Server) gazeteDogrula(ctx context.Context, siteID, issueID int64) ([]gazeteTasma, error) {
	iss, err := repo.GetGazeteIssue(ctx, s.pool, issueID)
	if err != nil {
		return nil, err
	}
	htmlDoc, err := s.gazetePrintHTMLOpt(ctx, siteID, iss, true)
	if err != nil {
		return nil, err
	}
	chrome := findChrome()
	if chrome == "" {
		return nil, fmt.Errorf("Chromium bulunamadi (GAZETE_CHROME ayarlayin)")
	}
	tmpDir, err := os.MkdirTemp("", "gazete-dogrula-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	htmlPath := filepath.Join(tmpDir, "olcum.html")
	// #nosec G703 -- Yol, os.MkdirTemp'in ürettiği geçici dizin + SABİT
	// dosya adı ("olcum.html"). Dış girdi yola girmiyor.
	if err := os.WriteFile(htmlPath, []byte(gazeteBaskiMedyaMutlak(htmlDoc)), 0o644); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, chrome, // #nosec G702 -- sabit bayraklar + MkdirTemp yolu (yukarıdaki G703 notu)
		"--headless=new", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage",
		"--virtual-time-budget=15000", "--dump-dom", "file://"+htmlPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("olcum basarisiz: %w", err)
	}
	return gazeteTasmaAyikla(string(out))
}

// gazeteTasmaAyikla dump-dom ciktisindan olcum raporunu cikarir.
func gazeteTasmaAyikla(dom string) ([]gazeteTasma, error) {
	const bas = `id="gz-tasma-raporu">`
	i := strings.Index(dom, bas)
	if i < 0 {
		return nil, fmt.Errorf("olcum raporu uretilmedi (sayfa yuklenemedi olabilir)")
	}
	rest := dom[i+len(bas):]
	j := strings.Index(rest, "</div>")
	if j < 0 {
		return nil, fmt.Errorf("olcum raporu eksik")
	}
	ham := html.UnescapeString(rest[:j])
	var list []gazeteTasma
	if err := json.Unmarshal([]byte(ham), &list); err != nil {
		return nil, fmt.Errorf("olcum raporu cozulemedi: %w", err)
	}
	for i := range list {
		list[i].Oneri = gazeteTasmaOneri(list[i])
	}
	sort.SliceStable(list, func(a, b int) bool {
		if list[a].Sayfa != list[b].Sayfa {
			return list[a].Sayfa < list[b].Sayfa
		}
		return list[a].Dikey > list[b].Dikey
	})
	return list, nil
}

// ---- PDF üretimi (Chromium headless) ----

// findChrome baskı motoru için kullanılacak Chromium ikilisini arar.
func findChrome() string {
	if v := os.Getenv("GAZETE_CHROME"); v != "" {
		// #nosec G703 -- Yol, sunucu yöneticisinin ayarladığı ortam
		// değişkeninden gelir; HTTP girdisi değil. Yalnız var mı diye
		// bakılır, içeriği okunmaz.
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	candidates := []string{
		"google-chrome", "google-chrome-stable", "chromium", "chromium-browser",
	}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	// Puppeteer tarayıcı önbelleği (e2e ile kurulmuş olabilir).
	home, _ := os.UserHomeDir()
	patterns := []string{
		filepath.Join(home, ".cache", "puppeteer", "chrome", "linux-*", "chrome-linux64", "chrome"),
		filepath.Join(home, ".cache", "puppeteer", "chrome", "mac-*", "chrome*", "chrome"),
	}
	for _, pat := range patterns {
		if matches, err := filepath.Glob(pat); err == nil {
			sort.Strings(matches)
			for i := len(matches) - 1; i >= 0; i-- {
				if st, err := os.Stat(matches[i]); err == nil && !st.IsDir() {
					return matches[i]
				}
			}
		}
	}
	return ""
}

// gazeteGeneratePDF baskıyı PDF'e çevirir, depolamaya yazar ve yolunu kaydeder.
func (s *Server) gazeteGeneratePDF(ctx context.Context, siteID, issueID int64) (string, error) {
	iss, err := repo.GetGazeteIssue(ctx, s.pool, issueID)
	if err != nil {
		return "", err
	}
	htmlDoc, err := s.gazetePrintHTML(ctx, siteID, iss)
	if err != nil {
		return "", err
	}
	chrome := findChrome()
	if chrome == "" {
		return "", fmt.Errorf("Chromium bulunamadı (GAZETE_CHROME ayarlayın)")
	}
	tmpDir, err := os.MkdirTemp("", "gazete-pdf-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	htmlPath := filepath.Join(tmpDir, "baski.html")
	pdfPath := filepath.Join(tmpDir, "gazete.pdf")
	// Gorseller file:// altinda cozulebilsin diye mutlak yola cevrilir.
	// #nosec G703 -- Yol, os.MkdirTemp'in geçici dizini + SABİT dosya adı
	// ("baski.html"). Dış girdi yola girmiyor.
	if err := os.WriteFile(htmlPath, []byte(gazeteBaskiMedyaMutlak(htmlDoc)), 0o644); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, chrome, // #nosec G702 -- sabit bayraklar + MkdirTemp yolu (G703 notuyla aynı gerekçe)
		"--headless=new", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage",
		"--no-pdf-header-footer", "--virtual-time-budget=15000",
		"--print-to-pdf="+pdfPath, "file://"+htmlPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("PDF üretilemedi: %w (%s)", err, firstLine(string(out), 200))
	}
	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		return "", err
	}
	key := platform.GazeteKey(siteID, iss.IssueDate, "gazete.pdf")
	store := s.storeForSite(ctx, siteID)
	if _, err := store.Put(key, pdf, "application/pdf"); err != nil {
		return "", err
	}
	if err := repo.SetGazeteIssuePDF(ctx, s.pool, issueID, key); err != nil {
		return "", err
	}
	// Flip/minik görseller (pdftoppm varsa) — sessiz geç.
	s.gazetePageImages(ctx, siteID, iss, pdfPath)
	return key, nil
}

// gazetePageImages PDF sayfalarını PNG'ye çevirip depolamaya yazar (varsa).
func (s *Server) gazetePageImages(ctx context.Context, siteID int64, iss repo.GazeteIssue, pdfPath string) {
	pdftoppm, err := exec.LookPath("pdftoppm")
	if err != nil {
		return
	}
	tmpDir, err := os.MkdirTemp("", "gazete-img-*")
	if err != nil {
		return
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	prefix := filepath.Join(tmpDir, "sayfa")
	cmd := exec.CommandContext(ctx, pdftoppm, // #nosec G702 -- sabit bayraklar; pdfPath/prefix MkdirTemp kaynaklı
		"-png", "-r", "60", pdfPath, prefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		s.logger.Warn("gazete sayfa görselleri", "error", err, "out", firstLine(string(out), 120))
		return
	}
	matches, _ := filepath.Glob(prefix + "-*.png")
	sort.Strings(matches)
	store := s.storeForSite(ctx, siteID)
	for i, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		key := platform.GazeteKey(siteID, iss.IssueDate, fmt.Sprintf("sayfa-%02d.png", i+1))
		_, _ = store.Put(key, data, "image/png")
	}
}

// handleGazetePDF POST /admin/gazete/sayilar/{id}/pdf — baskı PDF üret.
func (s *Server) handleGazetePDF(w http.ResponseWriter, r *http.Request) {
	if !s.requireGazete(w, r, gazeteActionEdit) {
		return
	}
	id := parseID(r.PathValue("id"))
	sid := s.curSiteID(r)
	// Tasma gecidi (SPEC K3): tasan blok varsa PDF sessizce uretilmez.
	// Kullanici yine de uretmek isterse zorla=1 ile gecer -- karar onun,
	// ama bilerek verilsin.
	if r.FormValue("zorla") != "1" {
		if tasmalar, derr := s.gazeteDogrula(r.Context(), sid, id); derr != nil {
			// Olcum yapilamadiysa uretimi engelleme; sebebi gunluge yaz.
			s.logger.Warn("gazete dogrulama", "error", derr)
		} else if len(tasmalar) > 0 {
			s.gazeteResult(w, r, "error", gazeteTasmaOzeti(tasmalar))
			return
		}
	}
	key, err := s.gazeteGeneratePDF(r.Context(), sid, id)
	if err != nil {
		s.logger.Error("gazete pdf", "error", err)
		s.gazeteResult(w, r, "error", "PDF üretilemedi: "+err.Error())
		return
	}
	s.audit(r, "gazete_pdf", strconv.FormatInt(id, 10))
	s.gazeteResult(w, r, "info", "PDF üretildi: "+key)
}

// gazeteTasmaOzeti tasma listesini kullaniciya gosterilecek tek satira indirir.
func gazeteTasmaOzeti(t []gazeteTasma) string {
	if len(t) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d blok sayfaya sigmiyor, PDF uretilmedi. ", len(t))
	n := len(t)
	if n > 3 {
		n = 3
	}
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteString(" · ")
		}
		fmt.Fprintf(&sb, "s.%d %s (%s): %s", t[i].Sayfa, t[i].BlokID, t[i].Tip, t[i].Oneri)
	}
	if len(t) > n {
		fmt.Fprintf(&sb, " · ve %d blok daha", len(t)-n)
	}
	sb.WriteString(". Yine de uretmek icin \"Zorla PDF\" kullanin.")
	return sb.String()
}
