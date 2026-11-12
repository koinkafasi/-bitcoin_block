package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Yeniden yazım durumları (DB'de metin).
const (
	YenidenBekliyor   = "bekliyor"
	YenidenKuyrukta   = "kuyrukta"
	YenidenHazir      = "hazir"
	YenidenHata       = "hata"
	YenidenUygulandi  = "uygulandi"
	YenidenReddedildi = "reddedildi"
)

const (
	YenidenModYeniden     = "yeniden"
	YenidenModArastirmali = "arastirmali"
)

type YenidenKaynak struct {
	Ref       string `json:"ref,omitempty"`
	Baslik    string `json:"baslik"`
	URL       string `json:"url"`
	Yayinci   string `json:"yayinci,omitempty"`
	Ozet      string `json:"ozet,omitempty"`
	YayinTari string `json:"yayin_tari,omitempty"`
}

const YenidenAzamiDeneme = 3

// YenidenAday yayındaki ince bir içeriktir.
type YenidenAday struct {
	ContentID int64
	RootID    int64
	Locale    string
	Title     string
	Blok      int
	Karakter  int
	Version   int
	Hit       int
}

// YenidenYazim tek bir içerik/dil için AI yeniden yazım önerisidir.
type YenidenYazim struct {
	ID             int64
	SiteID         int64
	ContentID      int64
	RootID         int64
	Locale         string
	Status         string
	ContentVersion int
	Mod            string
	// Kaynaklar yalnız 'arastirmali' modda dolu.
	Kaynaklar []YenidenKaynak

	OldTitle    string
	OldSpot     string
	OldDetail   []byte
	OldSeoTitle string
	OldSeoDesc  string
	NewTitle    string
	NewSpot     string
	NewDetail   []byte
	NewSeoTitle string
	NewSeoDesc  string

	Warnings         []string
	Error            string
	Provider         string
	Model            string
	PromptTokens     int
	CompletionTokens int
	Attempts         int

	CreatedAt time.Time
	UpdatedAt time.Time
	AppliedAt *time.Time

	// join (panel için)
	GuncelBaslik string
	GuncelSlug   string
	Kategori     string
}

// YenidenYazimIs işçinin sahiplendiği öneridir (içerik metni dahil).
type YenidenYazimIs struct {
	ID        int64
	ContentID int64
	Locale    string
	Mod       string
	Version   int
	// Hit sıralaması için: bu içeriğin toplam görüntülenmesi.
	Hit        int
	Title      string
	Spot       string
	Detail     []byte
	SeoTitle   string
	SeoDesc    string
	SourceURL  string
	SourceName string
}

func GovdeParagraflari(detail []byte) []string {
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
			if t := strings.TrimSpace(b.Text); t != "" {
				out = append(out, t)
			}
		}
		return out
	}
	var probe struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(detail, &probe) == nil && probe.Type == "doc" {
		var doc struct {
			Content []any `json:"content"`
		}
		if err := json.Unmarshal(detail, &doc); err == nil {
			var out []string
			for _, blok := range doc.Content {
				if m, ok := blok.(map[string]any); ok {
					var sb strings.Builder
					gezMetin(m, &sb)
					if t := strings.TrimSpace(sb.String()); t != "" {
						out = append(out, t)
					}
				}
			}
			return out
		}
	}
	return nil
}

// GovdeMetni gövdeyi tek metin olarak döndürür (paragraflar boş satırla).
func GovdeMetni(detail []byte) string {
	return strings.Join(GovdeParagraflari(detail), "\n\n")
}

func YenidenYazimAdaylari(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale string, enFazlaKarakter int, aktifOlanlarHaric bool, limit int) ([]YenidenAday, error) {
	if siteID <= 0 {
		siteID = 1
	}
	if enFazlaKarakter <= 0 {
		enFazlaKarakter = 400
	}
	if limit <= 0 {
		limit = 2000
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, COALESCE(c.translation_of, c.id), c.locale, c.title, c.detail, c.version,
		       COALESCE(h.hit, 0)
		  FROM contents c
		  LEFT JOIN (
				SELECT content_id, sum(hits)::int AS hit
				  FROM hit_daily
				 WHERE day >= DATE '2020-01-01'
				 GROUP BY content_id) h ON h.content_id = c.id
		 WHERE c.status = 'yayinda'
		   AND c.site_id = $1
		   AND ($2 = '' OR c.locale = $2)
		   AND COALESCE(c.title,'') <> ''
		   AND c.detail <> '[]'::jsonb
		   AND ($4 = false OR NOT EXISTS (
				SELECT 1 FROM yeniden_yazim y
				 WHERE y.content_id = c.id
				   AND y.status IN ('bekliyor','kuyrukta','hazir','hata')))
		 ORDER BY COALESCE(h.hit,0) DESC, c.id
		 LIMIT $3`, siteID, locale, limit, aktifOlanlarHaric)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []YenidenAday
	for rows.Next() {
		var a YenidenAday
		var detail []byte
		if err := rows.Scan(&a.ContentID, &a.RootID, &a.Locale, &a.Title, &detail, &a.Version, &a.Hit); err != nil {
			return nil, err
		}
		paragraflar := GovdeParagraflari(detail)
		a.Blok = len(paragraflar)
		a.Karakter = len([]rune(strings.Join(paragraflar, " ")))
		if a.Blok > 1 && a.Karakter >= enFazlaKarakter {
			continue
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func YenidenYazimPilot(ctx context.Context, pool *pgxpool.Pool, siteID int64, kokSayisi int, mod string) (int, error) {
	if kokSayisi <= 0 {
		kokSayisi = 10
	}
	if mod != YenidenModYeniden {
		mod = YenidenModArastirmali
	}
	adaylar, err := YenidenYazimAdaylari(ctx, pool, siteID, "", 400, true, 4000)
	if err != nil {
		return 0, err
	}
	// Kökleri sırayla topla (ilk görülme sırası korunur).
	var kokler []int64
	gorulen := map[int64]bool{}
	for _, a := range adaylar {
		if gorulen[a.RootID] {
			continue
		}
		// Kökün TR satırı olmalı: kaynak dil Türkçe.
		trVar := false
		for _, b := range adaylar {
			if b.RootID == a.RootID && b.Locale == "tr" {
				trVar = true
				break
			}
		}
		if !trVar {
			continue
		}
		gorulen[a.RootID] = true
		kokler = append(kokler, a.RootID)
		if len(kokler) >= kokSayisi {
			break
		}
	}
	if len(kokler) == 0 {
		return 0, nil
	}
	secili := map[int64]bool{}
	for _, k := range kokler {
		secili[k] = true
	}

	olusan := 0
	for _, a := range adaylar {
		if !secili[a.RootID] {
			continue
		}
		c, err := GetContent(ctx, pool, a.ContentID)
		if err != nil {
			continue // arada silinmiş olabilir
		}
		eklendi, err := YenidenYazimAc(ctx, pool, c, mod)
		if err != nil {
			return olusan, err
		}
		if eklendi {
			olusan++
		}
	}
	return olusan, nil
}

func YenidenYazimAc(ctx context.Context, pool *pgxpool.Pool, c Content, mod string) (bool, error) {
	if mod != YenidenModYeniden {
		mod = YenidenModArastirmali
	}
	root := c.ID
	if c.TranslationOf != nil && *c.TranslationOf > 0 {
		root = *c.TranslationOf
	}
	site := c.SiteID
	if site <= 0 {
		site = 1
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO yeniden_yazim
			(site_id, content_id, root_id, locale, status, content_version, mod,
			 old_title, old_spot, old_detail, old_seo_title, old_seo_description)
		VALUES ($1,$2,$3,$4,'bekliyor',$5,$6,$7,$8,$9::jsonb,$10,$11)
		ON CONFLICT DO NOTHING
		RETURNING id`,
		site, c.ID, root, c.Locale, c.Version, mod,
		c.Title, c.Spot, string(c.Detail), derefStr(c.SeoTitle), derefStr(c.SeoDesc),
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func YenidenYazimSahiplen(ctx context.Context, pool *pgxpool.Pool, limit int) ([]YenidenYazimIs, error) {
	if limit <= 0 {
		limit = 1
	}
	if _, err := pool.Exec(ctx, `
		UPDATE yeniden_yazim
		   SET status = 'bekliyor', updated_at = now()
		 WHERE status = 'kuyrukta'
		   AND updated_at < now() - interval '10 minutes'
		   AND attempts < $1`, YenidenAzamiDeneme); err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		WITH sec AS (
			SELECT id FROM yeniden_yazim
			 WHERE status = 'bekliyor'
			 ORDER BY id
			 LIMIT $1
			 FOR UPDATE SKIP LOCKED)
		UPDATE yeniden_yazim y
		   SET status = 'kuyrukta', updated_at = now()
		  FROM sec
		 WHERE y.id = sec.id
		RETURNING y.id, y.content_id, y.locale, y.mod, y.content_version`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []YenidenYazimIs
	for rows.Next() {
		var is YenidenYazimIs
		if err := rows.Scan(&is.ID, &is.ContentID, &is.Locale, &is.Mod, &is.Version); err != nil {
			return nil, err
		}
		out = append(out, is)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// İçerik metni ayrı okunur: güncel hâl (editör arada kaydetmiş olabilir).
	for i := range out {
		var title, spot, seoTitle, seoDesc string
		var detail []byte
		var sourceURL, sourceName *string
		err := pool.QueryRow(ctx, `
			SELECT c.title, COALESCE(c.spot,''), c.detail,
			       COALESCE(c.seo_title,''), COALESCE(c.seo_description,''),
			       c.source_url, c.source_name
			  FROM contents c WHERE c.id = $1`, out[i].ContentID,
		).Scan(&title, &spot, &detail, &seoTitle, &seoDesc, &sourceURL, &sourceName)
		if err != nil {
			return nil, err
		}
		out[i].Title = title
		out[i].Spot = spot
		out[i].Detail = detail
		out[i].SeoTitle = seoTitle
		out[i].SeoDesc = seoDesc
		out[i].SourceURL = derefStr(sourceURL)
		out[i].SourceName = derefStr(sourceName)
	}
	return out, nil
}

// YenidenYazimKuyrukBirak işlenemeyen öneriyi kuyruktan çıkarır.
func YenidenYazimKuyrukBirak(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `
		UPDATE yeniden_yazim
		   SET status = 'bekliyor', updated_at = now()
		 WHERE id = $1 AND status = 'kuyrukta'`, id)
	return err
}

// YenidenYazimSonucu AI'dan gelen yeni metindir.
type YenidenYazimSonucu struct {
	Spot     string
	Paragraf []string
	SeoTitle string
	SeoDesc  string
}

// YenidenYazimKaydet kalite kapısından geçen öneriyi "hazir" yapar.
func YenidenYazimKaydet(ctx context.Context, pool *pgxpool.Pool, id int64, s YenidenYazimSonucu, uyarilar []string, kaynaklar []YenidenKaynak, provider, model string, promptTok, completionTok int) error {
	detay, err := yazimDetay(s.Paragraf)
	if err != nil {
		return err
	}
	if uyarilar == nil {
		uyarilar = []string{}
	}
	uyariJSON, err := json.Marshal(uyarilar)
	if err != nil {
		return err
	}
	if kaynaklar == nil {
		kaynaklar = []YenidenKaynak{}
	}
	kaynakJSON, err := json.Marshal(kaynaklar)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		UPDATE yeniden_yazim SET
			status = 'hazir',
			new_title = '',
			new_spot = $2,
			new_detail = $3::jsonb,
			new_seo_title = $4,
			new_seo_description = $5,
			warnings = $6::jsonb,
			sources = $7::jsonb,
			error = '',
			provider = $8,
			model = $9,
			prompt_tokens = $10,
			completion_tokens = $11,
			updated_at = now()
		 WHERE id = $1 AND status = 'kuyrukta'`,
		id, s.Spot, detay, s.SeoTitle, s.SeoDesc, string(uyariJSON), string(kaynakJSON),
		provider, model, promptTok, completionTok)
	return err
}

// YenidenYazimHata öneriyi hata durumuna alır ve denemeyi sayar.
func YenidenYazimHata(ctx context.Context, pool *pgxpool.Pool, id int64, mesaj string) error {
	if len(mesaj) > 500 {
		mesaj = mesaj[:500]
	}
	_, err := pool.Exec(ctx, `
		UPDATE yeniden_yazim SET
			status = 'hata',
			error = $2,
			attempts = attempts + 1,
			updated_at = now()
		 WHERE id = $1`, id, mesaj)
	return err
}

func YenidenYazimGeciciHata(ctx context.Context, pool *pgxpool.Pool, id int64, mesaj string) (bool, error) {
	if len(mesaj) > 500 {
		mesaj = mesaj[:500]
	}
	var durum string
	err := pool.QueryRow(ctx, `
		UPDATE yeniden_yazim SET
			attempts = attempts + 1,
			status = CASE WHEN attempts + 1 < $2 THEN 'bekliyor' ELSE 'hata' END,
			error = $3,
			updated_at = now()
		 WHERE id = $1
		RETURNING status`, id, YenidenAzamiDeneme, mesaj).Scan(&durum)
	if err != nil {
		return false, err
	}
	return durum == YenidenBekliyor, nil
}

// YenidenYazimTek tek öneriyi (panel detayı için) okur.
func YenidenYazimTek(ctx context.Context, pool *pgxpool.Pool, id int64) (YenidenYazim, error) {
	rows, err := pool.Query(ctx, yenidenSelect+` WHERE y.id = $1`, id)
	if err != nil {
		return YenidenYazim{}, err
	}
	defer rows.Close()
	liste, err := yenidenTara(rows)
	if err != nil {
		return YenidenYazim{}, err
	}
	if len(liste) == 0 {
		return YenidenYazim{}, ErrNotFound
	}
	return liste[0], nil
}

// YenidenYazimListele panel listesi (yeni önce).
func YenidenYazimListele(ctx context.Context, pool *pgxpool.Pool, siteID int64, status, locale string, limit int) ([]YenidenYazim, error) {
	if siteID <= 0 {
		siteID = 1
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := pool.Query(ctx, yenidenSelect+`
		 WHERE y.site_id = $1
		   AND ($2 = '' OR y.status = $2)
		   AND ($3 = '' OR y.locale = $3)
		 ORDER BY y.id DESC
		 LIMIT $4`, siteID, status, locale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return yenidenTara(rows)
}

const yenidenSelect = `
	SELECT y.id, y.site_id, y.content_id, y.root_id, y.locale, y.status, y.content_version,
	       y.mod, y.sources,
	       y.old_title, y.old_spot, y.old_detail, y.old_seo_title, y.old_seo_description,
	       y.new_title, y.new_spot, y.new_detail, y.new_seo_title, y.new_seo_description,
	       y.warnings, y.error, y.provider, y.model, y.prompt_tokens, y.completion_tokens,
	       y.attempts, y.created_at, y.updated_at, y.applied_at,
	       COALESCE(c.title,''), COALESCE(c.slug,''), COALESCE(cat.name,'')
	  FROM yeniden_yazim y
	  JOIN contents c ON c.id = y.content_id
	  LEFT JOIN categories cat ON cat.id = c.category_id`

func yenidenTara(rows pgx.Rows) ([]YenidenYazim, error) {
	var out []YenidenYazim
	for rows.Next() {
		var y YenidenYazim
		var uyariJSON, kaynakJSON []byte
		if err := rows.Scan(&y.ID, &y.SiteID, &y.ContentID, &y.RootID, &y.Locale, &y.Status, &y.ContentVersion,
			&y.Mod, &kaynakJSON,
			&y.OldTitle, &y.OldSpot, &y.OldDetail, &y.OldSeoTitle, &y.OldSeoDesc,
			&y.NewTitle, &y.NewSpot, &y.NewDetail, &y.NewSeoTitle, &y.NewSeoDesc,
			&uyariJSON, &y.Error, &y.Provider, &y.Model, &y.PromptTokens, &y.CompletionTokens,
			&y.Attempts, &y.CreatedAt, &y.UpdatedAt, &y.AppliedAt,
			&y.GuncelBaslik, &y.GuncelSlug, &y.Kategori); err != nil {
			return nil, err
		}
		if len(uyariJSON) > 0 {
			_ = json.Unmarshal(uyariJSON, &y.Warnings)
		}
		if len(kaynakJSON) > 0 {
			_ = json.Unmarshal(kaynakJSON, &y.Kaynaklar)
		}
		out = append(out, y)
	}
	return out, rows.Err()
}

func YenidenYazimOnayla(ctx context.Context, pool *pgxpool.Pool, id int64, userID *int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var contentID int64
	var status string
	var surum int
	var yeniSpot, yeniSeoTitle, yeniSeoDesc string
	var yeniDetay []byte
	err = tx.QueryRow(ctx, `
		SELECT content_id, status, content_version, new_spot, new_detail, new_seo_title, new_seo_description
		  FROM yeniden_yazim WHERE id = $1 FOR UPDATE`, id,
	).Scan(&contentID, &status, &surum, &yeniSpot, &yeniDetay, &yeniSeoTitle, &yeniSeoDesc)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != YenidenHazir {
		return errors.New("öneri onaya hazır değil")
	}
	if len(yeniDetay) == 0 || string(yeniDetay) == "[]" {
		return errors.New("öneri gövdesi boş")
	}

	// 1) Eski hâli arşivle (aynı sürüm iki kez yazılmaz).
	if _, err := tx.Exec(ctx, `
		INSERT INTO content_revisions
			(content_id, version, title, spot, detail, snapshot, user_id, source, ai_job_id)
		SELECT c.id, c.version, c.title, COALESCE(c.spot,''), c.detail, to_jsonb(c), $2, 'yeniden_yazim', NULL
		FROM contents c WHERE c.id=$1
		ON CONFLICT (content_id, version) DO NOTHING`,
		contentID, nullableInt64P(userID)); err != nil {
		return err
	}

	// 2) Yeni metni yaz (sürüm korumalı).
	komut, err := tx.Exec(ctx, `
		UPDATE contents SET
			spot = $2,
			detail = $3::jsonb,
			seo_title = NULLIF($4,''),
			seo_description = NULLIF($5,''),
			version = version + 1,
			updated_at = now()
		 WHERE id = $1 AND version = $6`,
		contentID, yeniSpot, yeniDetay, yeniSeoTitle, yeniSeoDesc, surum)
	if err != nil {
		return err
	}
	if komut.RowsAffected() == 0 {
		return ErrConflict
	}

	// 3) Öneriyi kapat.
	if _, err := tx.Exec(ctx, `
		UPDATE yeniden_yazim
		   SET status = 'uygulandi', applied_at = now(), updated_at = now()
		 WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// YenidenYazimReddet öneriyi kapatır; içerik dokunulmaz.
func YenidenYazimReddet(ctx context.Context, pool *pgxpool.Pool, id int64, not string) error {
	if len(not) > 300 {
		not = not[:300]
	}
	_, err := pool.Exec(ctx, `
		UPDATE yeniden_yazim
		   SET status = 'reddedildi', error = $2, updated_at = now()
		 WHERE id = $1 AND status IN ('hazir','hata','bekliyor')`, id, not)
	return err
}

// YenidenYazimYenidenDenen hatalı/reddedilmiş öneriyi kuyruğa geri koyar.
func YenidenYazimYenidenDenen(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `
		UPDATE yeniden_yazim
		   SET status = 'bekliyor', error = '', attempts = 0, updated_at = now()
		 WHERE id = $1 AND status IN ('hata','reddedildi')`, id)
	return err
}

// YenidenYazimOzet paneldeki sayaçlar (durum → adet).
func YenidenYazimOzet(ctx context.Context, pool *pgxpool.Pool, siteID int64) (map[string]int, error) {
	if siteID <= 0 {
		siteID = 1
	}
	rows, err := pool.Query(ctx, `
		SELECT status, count(*) FROM yeniden_yazim
		 WHERE site_id = $1 GROUP BY 1`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var s string
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out[s] = n
	}
	return out, rows.Err()
}

// YenidenYazimAdaySayisi panelde "kaç içerik ince" göstergesi.
func YenidenYazimAdaySayisi(ctx context.Context, pool *pgxpool.Pool, siteID int64) int {
	adaylar, err := YenidenYazimAdaylari(ctx, pool, siteID, "", 400, false, 4000)
	if err != nil {
		return 0
	}
	return len(adaylar)
}
