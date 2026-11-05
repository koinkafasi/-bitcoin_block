package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/scoring"
)

// Content veritabanı içerik kaydı.
type Content struct {
	ID            int64
	TypeKey       string
	Status        string
	Title         string
	Spot          string
	Detail        []byte
	Slug          *string
	CategoryID    *int64
	AuthorID      int64
	MahrepID      *int64
	CoverMediaID  *int64
	MansetMediaID *int64
	SocialMediaID *int64
	// VideoSeconds video içeriğinin süresi (saniye). Çoğu liste sorgusu bu
	// sütunu okumaz; yalnız EKRANA BASILACAK video kartları için tek sorguda
	// doldurulur (bkz. httpapi.fillVideoDurations) — N+1 olmasın diye.
	VideoSeconds  *int
	IsAM          bool
	IsUM          bool
	IsDM          bool
	IsSD          bool
	IsFH          bool
	IsHK          bool
	SourceURL     *string
	SourceName    *string
	CommentStatus string
	IsExclusive   bool
	// MinTierWeight içerik için gereken en düşük abonelik katmanı (0 = açık).
	MinTierWeight int
	SeoTitle      *string
	SeoDesc       *string
	CanonicalURL  *string
	PublishAt     *time.Time
	PublishedAt   *time.Time
	Version       int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Locale        string
	TranslationOf *int64
	SiteID        int64

	// join (opsiyonel dolu)
	CategoryName *string
	CategorySlug *string
	AuthorName   string
	MahrepName   *string
	TypeLabel    *string

	IsYMYL     bool
	YMYLUzman  *string
	YMYLDurumu string
	// AiScore AI alaka puanı (0-100) — kategori yayın kapısı (spec 06 K-A)
	AiScore int
}

// ListFilter içerik listeleme filtreleridir.
type ListFilter struct {
	Status     string
	TypeKey    string
	CategoryID int64
	Q          string
	Locale     string
	SiteID     int64
	Page       int
	PerPage    int
	// LastDays > 0 ise yalnız son N günde güncellenen içerikler (Faz 30 —
	// admin liste filtresi; sekme ne olursa olsun updated_at'e göre).
	LastDays int
	// Baslangic/Bitis özel tarih aralığı (Faz 47): updated_at >= baslangic
	// ve updated_at < bitis (bitiş dahil günün ertesi; *time.Time).
	Baslangic *time.Time
	Bitis     *time.Time
	// IncludeGallery false ise foto_galeri / video türleri listelenmez
	// (haber akışı ve "Son Haberler" gibi listeler için).
	IncludeGallery bool
	ExcludeGallery bool
	// Audio yalnız ses/röportaj kaydı olan içerikleri süzer (admin liste
	// filtresi): AudioHas → transkripti olan, AudioPending → işi süren/kuyrukta,
	// AudioFailed → başarısız işi olan. Diğer değerler filtre uygulamaz.
	Audio string
	// Sort liste sıralaması (beyaz liste anahtarı; bkz. triyaj.go).
	// Bilinmeyen değer varsayılana düşer, asla SQL'e yazılmaz.
	Sort string
	// MinScore 1..100 ise yalnız bu AI skorunun üstündeki içerikler.
	// 0 = eşik yok. Triyajda "yayına değer taslakları göster" için.
	MinScore int
	// KokBazli true ise aynı makalenin dilleri TEK satıra indirilir
	// (Faz 192). Admin listesi bunu kullanır.
	//
	// Ölçüm (üretim): 13.350 satırın 493'ü çeviri; "Yayında" sekmesinin ilk
	// 40 satırının 40'ı da EN çeviriydi, TR asılları sonraki sayfalara
	// düşüyordu. Yani editör Türkçe içerikleri listede göremiyordu.
	KokBazli bool
	// KokDil satırın hangi dilinin gösterileceğini belirler (varsayılan "tr").
	// Yalnız KokBazli iken anlamlı.
	KokDil string
}

// Audio filtresi değerleri.
const (
	AudioHas     = "var"
	AudioPending = "bekliyor"
	AudioFailed  = "basarisiz"
)

const defaultPerPage = 25

// contentCols içerik sorgularının ortak sütun listesi.
//
// `spot` NULL olabilir (şema `TEXT`, NOT NULL değil) ama `Content.Spot`
// düz `string`. COALESCE olmadan pgx "cannot scan NULL into *string" hatası
// veriyordu: spot'u boş bırakılmış bir içerik HİÇ okunamıyordu. Bu da
// scheduler indekslemesini, bot akışını ve detay/liste ekranlarını
// etkiliyordu (Faz 208'de ölçüldü).
const contentCols = `c.id, c.type_key, c.status, c.title, COALESCE(c.spot,''), c.detail, c.slug,
	c.category_id, c.author_id, c.mahrep_id, c.cover_media_id, c.manset_media_id, c.social_media_id,
	c.is_am, c.is_um, c.is_dm, c.is_sd, c.is_fh, c.is_hk,
	c.source_url, c.source_name, c.comment_status, c.is_exclusive,
	c.seo_title, c.seo_description, c.canonical_url,
	c.publish_at, c.published_at, c.version, c.created_at, c.updated_at, c.locale, c.translation_of, c.site_id,
	c.is_ymyl, c.ymyl_uzman, c.ymyl_durumu, c.ai_score,
	cat.name, cat.slug, u.name, m.name`

// ScanContent satırı Content'e aktarır.
func scanContent(row pgx.Row) (Content, error) {
	var c Content
	var catName, catSlug *string
	var mahrepName *string
	err := row.Scan(
		&c.ID, &c.TypeKey, &c.Status, &c.Title, &c.Spot, &c.Detail, &c.Slug,
		&c.CategoryID, &c.AuthorID, &c.MahrepID, &c.CoverMediaID, &c.MansetMediaID, &c.SocialMediaID,
		&c.IsAM, &c.IsUM, &c.IsDM, &c.IsSD, &c.IsFH, &c.IsHK,
		&c.SourceURL, &c.SourceName, &c.CommentStatus, &c.IsExclusive,
		&c.SeoTitle, &c.SeoDesc, &c.CanonicalURL,
		&c.PublishAt, &c.PublishedAt, &c.Version, &c.CreatedAt, &c.UpdatedAt, &c.Locale, &c.TranslationOf, &c.SiteID,
		&c.IsYMYL, &c.YMYLUzman, &c.YMYLDurumu, &c.AiScore,
		&catName, &catSlug, &c.AuthorName, &mahrepName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Content{}, ErrNotFound
	}
	if err != nil {
		return Content{}, err
	}
	c.CategoryName, c.CategorySlug = catName, catSlug
	c.MahrepName = mahrepName
	return c, nil
}

// SetContentCoverMedia içeriğin kapak görselini ayarlar (otomatik kapak üretimi).
func SetContentCoverMedia(ctx context.Context, pool *pgxpool.Pool, contentID, mediaID int64) error {
	_, err := pool.Exec(ctx, `UPDATE contents SET cover_media_id=$1 WHERE id=$2`, mediaID, contentID)
	return err
}

// GetContent id ile içerik döndürür.
func GetContent(ctx context.Context, pool *pgxpool.Pool, id int64) (Content, error) {
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.id = $1`
	c, err := scanContent(pool.QueryRow(ctx, q, id))
	if err != nil {
		return c, err
	}
	_ = pool.QueryRow(ctx, `SELECT min_tier_weight FROM contents WHERE id = $1`, id).Scan(&c.MinTierWeight)
	return c, nil
}

// ContentMinWeight içerik için gereken en düşük katman ağırlığını döndürür.
// Sorgu hatasi sessizce yutulmaz; güncellemede kademe korunacaksa hata
// cagirana bildirilir.
func ContentMinWeight(ctx context.Context, pool *pgxpool.Pool, id int64) (int, error) {
	var n int
	if err := pool.QueryRow(ctx, `SELECT min_tier_weight FROM contents WHERE id = $1`, id).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// ContentBySlugAndID yayın URL'si için slug+id ile içerik döndürür.
func ContentBySlugAndID(ctx context.Context, pool *pgxpool.Pool, slug string, id int64) (Content, error) {
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.id = $1`
	c, err := scanContent(pool.QueryRow(ctx, q, id))
	if err != nil {
		return Content{}, err
	}
	if slug != "" && (c.Slug == nil || *c.Slug != slug) {
		return Content{}, ErrNotFound
	}
	return c, nil
}

// CreateContent yeni içerik oluşturur; slug verildiyse çakışmaya karşı -2, -3… ekler.
func CreateContent(ctx context.Context, pool *pgxpool.Pool, c *Content) (int64, error) {
	slug, err := ensureSlug(ctx, pool, c.Slug)
	if err != nil {
		return 0, err
	}
	var id int64
	err = pool.QueryRow(ctx, `
		INSERT INTO contents
			(type_key, status, title, spot, detail, slug, category_id, author_id, mahrep_id,
			 cover_media_id, manset_media_id, social_media_id,
			 is_am, is_um, is_dm, is_sd, is_fh, is_hk,
			 source_url, source_name, comment_status, is_exclusive, min_tier_weight,
			 seo_title, seo_description, canonical_url, publish_at, published_at,
			 locale, translation_of, site_id, ai_score)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32)
		RETURNING id`,
		c.TypeKey, c.Status, c.Title, c.Spot, orEmptyJSON(c.Detail), slug,
		nullableInt(c.CategoryID), c.AuthorID, nullableInt(c.MahrepID),
		nullableInt(c.CoverMediaID), nullableInt(c.MansetMediaID), nullableInt(c.SocialMediaID),
		c.IsAM, c.IsUM, c.IsDM, c.IsSD, c.IsFH, c.IsHK,
		nullableText(c.SourceURL), nullableText(c.SourceName), c.CommentStatus, c.IsExclusive, c.MinTierWeight,
		nullableText(c.SeoTitle), nullableText(c.SeoDesc), nullableText(c.CanonicalURL),
		c.PublishAt, c.PublishedAt, c.Locale, nullableInt64(c.TranslationOf), siteIDOr1(c.SiteID),
		c.AiScore,
	).Scan(&id)
	return id, err
}

func siteIDOr1(n int64) int64 {
	if n <= 0 {
		return 1
	}
	return n
}

// UpdateContent var olan içeriği günceller (slug korunur; yayın zamanı güncellenebilir).
//
// c.Version > 0 ise iyimser kilitleme uygulanır: gönderilen sürüm veritabanındaki
// sürümle eşleşmezse ErrConflict döner. Version=0 eski çağrılar için koşulsuzdur.
func UpdateContent(ctx context.Context, pool *pgxpool.Pool, c *Content) error {
	var version int
	err := pool.QueryRow(ctx, `
		UPDATE contents SET
			type_key=$1, title=$2, spot=$3, detail=$4::jsonb, category_id=$5, mahrep_id=$6,
			cover_media_id=$7, manset_media_id=$8, social_media_id=$9,
			is_am=$10, is_um=$11, is_dm=$12, is_sd=$13, is_fh=$14, is_hk=$15,
			source_url=$16, source_name=$17, comment_status=$18, is_exclusive=$19,
			seo_title=$20, seo_description=$21, canonical_url=$22, publish_at=$23,
			is_ymyl=$24, ymyl_uzman=$25, ymyl_durumu=$26, version=version+1,
			min_tier_weight=$29
		WHERE id=$27 AND ($28=0 OR version=$28)
		RETURNING version`,
		c.TypeKey, c.Title, c.Spot, orEmptyJSON(c.Detail), nullableInt(c.CategoryID), nullableInt(c.MahrepID),
		nullableInt(c.CoverMediaID), nullableInt(c.MansetMediaID), nullableInt(c.SocialMediaID),
		c.IsAM, c.IsUM, c.IsDM, c.IsSD, c.IsFH, c.IsHK,
		nullableText(c.SourceURL), nullableText(c.SourceName), c.CommentStatus, c.IsExclusive,
		nullableText(c.SeoTitle), nullableText(c.SeoDesc), nullableText(c.CanonicalURL),
		c.PublishAt, c.IsYMYL, nullableText(c.YMYLUzman), c.YMYLDurumu, c.ID, c.Version,
		c.MinTierWeight,
	).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contents WHERE id=$1)`, c.ID).Scan(&exists)
		if !exists {
			return ErrNotFound
		}
		return ErrConflict
	}
	if err != nil {
		return err
	}
	c.Version = version

	// Faz 169: alaka puanını (ai_score) yeniden hesapla.
	//
	// Önceki durum: puan yalnız içerik OLUŞTURULURKEN ve "Uygunluğu hesapla"
	// düğmesine basıldığında yazılıyordu. Editör başlığı kısaltıp, spotu
	// silip, kapağı kaldırıp kaydettiğinde puan ESKİ hâlinde kalıyordu;
	// kategori yayın kapısı (categories.min_score) bayat bir sayıya bakıyordu.
	// (Faz 138'den beri bilinen borç.)
	//
	// Hata kaydı ENGELLEMEZ: skor ikincil bir alandır, kaydetmeyi düşürmek
	// editörün işini kaybetmesi demek olurdu.
	_ = tazeleAIScore(ctx, pool, c.ID, c.Title, c.Spot, c.CoverMediaID)
	return nil
}

// tazeleAIScore içeriğin alaka puanını (ai_score) yeniden hesaplar.
//
// Sinyaller scoring paketinden gelir (tek kaynak): başlık uzunluğu, spot
// uzunluğu, etiket sayısı, kapak ve kaynak varlığı. Etiket sayısı ve kaynak
// varlığı veritabanından TEK sorguda okunur (otomatik kaydetme her 800 ms'de
// bir çalışabildiği için sorgu sayısı önemli).
func tazeleAIScore(ctx context.Context, pool *pgxpool.Pool, id int64, title, spot string, coverID *int64) error {
	var etiket int
	var kaynak bool
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM content_tags WHERE content_id = $1),
			(COALESCE(source_url,'') <> '')
		FROM contents WHERE id = $1`, id).Scan(&etiket, &kaynak)
	if err != nil {
		return err
	}
	puan, _ := scoring.Explain(scoring.Input{
		Title:     title,
		Spot:      spot,
		TagCount:  etiket,
		HasCover:  coverID != nil,
		HasSource: kaynak,
	})
	_, err = pool.Exec(ctx, `UPDATE contents SET ai_score = $1 WHERE id = $2`, puan, id)
	return err
}

// EnsureUniqueSlug verilen istenen slug için çakışmasız sürüm döndürür.
func EnsureUniqueSlug(ctx context.Context, pool *pgxpool.Pool, base string, excludeID int64) (string, error) {
	rows, err := pool.Query(ctx,
		`SELECT slug FROM contents WHERE (slug = $1 OR slug LIKE $2) AND id <> $3`,
		base, base+"-%", excludeID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	used := map[string]bool{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return "", err
		}
		used[s] = true
	}
	candidate := base
	for n := 2; used[candidate]; n++ {
		candidate = fmt.Sprintf("%s-%d", base, n)
	}
	return candidate, rows.Err()
}

// ensureSlug ekleme sırasında slug'ı benzersiz yapar.
func ensureSlug(ctx context.Context, pool *pgxpool.Pool, desired *string) (*string, error) {
	if desired == nil || *desired == "" {
		return nil, nil
	}
	u, err := EnsureUniqueSlug(ctx, pool, *desired, 0)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SetContentStatus içerik durumunu geçirir ve zaman damgalarını yönetir.
func SetContentStatus(ctx context.Context, pool *pgxpool.Pool, id int64, status string) error {
	// yayınlama: published_at ilk kez set edilir; planlama: publish_at ileri tarih anlamında formda verilir
	tag, err := pool.Exec(ctx, `
		UPDATE contents SET
			status = $1,
			published_at = CASE WHEN $1 = 'yayinda' AND published_at IS NULL THEN now() ELSE published_at END,
			version = version + 1,
			updated_at = now()
		WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListContents filtrelerle içerik listesi döndürür.
func ListContents(ctx context.Context, pool *pgxpool.Pool, f ListFilter) ([]Content, int, error) {
	if f.KokBazli {
		return listContentsKokBazli(ctx, pool, f)
	}
	q := `SELECT ` + contentCols + `, count(*) OVER() FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE 1=1`
	args := []any{}
	n := 0
	add := func(cond string, v any) {
		n++
		q += " AND " + cond
		args = append(args, v)
	}
	if f.Status != "" {
		add("c.status = $"+itoa(n+1), f.Status)
	}
	if f.TypeKey != "" {
		add("c.type_key = $"+itoa(n+1), f.TypeKey)
	} else if f.ExcludeGallery {
		// haber listelerinde galeri içerikleri karışmasın
		q += " AND c.type_key NOT IN ('foto_galeri','video')"
	}
	if f.CategoryID != 0 {
		add("c.category_id = $"+itoa(n+1), f.CategoryID)
	}
	if f.SiteID > 0 {
		add("c.site_id = $"+itoa(n+1), f.SiteID)
	}
	if f.Locale != "" {
		add("c.locale = $"+itoa(n+1), f.Locale)
	}
	if f.Q != "" {
		add("(c.title ILIKE $"+itoa(n+1)+" OR c.spot ILIKE $"+itoa(n+1)+")", "%"+f.Q+"%")
	}
	if f.LastDays > 0 {
		add("c.updated_at >= $"+itoa(n+1), time.Now().AddDate(0, 0, -f.LastDays))
	}
	if f.Baslangic != nil {
		add("c.updated_at >= $"+itoa(n+1), *f.Baslangic)
	}
	if f.Bitis != nil {
		add("c.updated_at < $"+itoa(n+1), *f.Bitis)
	}
	// Ses/röportaj filtresi: içeriğe bağlı transkripsiyon işi var mı?
	switch f.Audio {
	case AudioHas:
		q += ` AND EXISTS (SELECT 1 FROM audio_transcripts at
			WHERE at.content_id = c.id AND COALESCE(at.text,'') <> '')`
	case AudioPending:
		q += ` AND EXISTS (SELECT 1 FROM transcription_jobs tj
			WHERE tj.content_id = c.id AND tj.status IN ('queued','running'))`
	case AudioFailed:
		q += ` AND EXISTS (SELECT 1 FROM transcription_jobs tj
			WHERE tj.content_id = c.id AND tj.status = 'failed')
			AND NOT EXISTS (SELECT 1 FROM audio_transcripts at
				WHERE at.content_id = c.id AND COALESCE(at.text,'') <> '')`
	}
	if ms := normalizeMinScore(f.MinScore); ms > 0 {
		add("COALESCE(c.ai_score,0) >= $"+itoa(n+1), ms)
	}
	// Sıralama beyaz listeden gelir; f.Sort doğrudan SQL'e girmez.
	q += " ORDER BY " + contentOrderBy(f.Sort)
	if f.PerPage <= 0 {
		f.PerPage = defaultPerPage
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	q += " LIMIT $" + itoa(n+1) + " OFFSET $" + itoa(n+2)
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Content
	total := 0
	for rows.Next() {
		c, err := scanContentWithTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		total = c.Total
		out = append(out, c.Content)
	}
	return out, total, rows.Err()
}

// listContentsKokBazli aynı makalenin dillerini tek satıra indirir.
//
// Neden ayrı fonksiyon: filtreler satır bazında uygulanıp SONRA gruplama
// yapılır (kullanıcı "yayında" dediğinde yayında olan sürüm temsilci olur).
// Filtreyi köke uygulamak, çevirisi yayında olan bir makaleyi "Taslak"
// sekmesinde gizlerdi.
//
// Sıra: filtre → kök başına DISTINCT ON (temsilci: önce kök dil, sonra en
// yeni) → temsilci satırın tam içeriği → sayfalama.
func listContentsKokBazli(ctx context.Context, pool *pgxpool.Pool, f ListFilter) ([]Content, int, error) {
	kokDil := f.KokDil
	if kokDil == "" {
		kokDil = "tr"
	}
	args := []any{kokDil}
	n := 1
	kok := `c.locale = $1`
	_ = kok // kök dil parametresi aşağıda sıralamada kullanılıyor
	kosul := "1=1"
	add := func(cond string, v any) {
		n++
		args = append(args, v)
		kosul += " AND " + cond + "$" + itoa(n)
	}
	if f.Status != "" {
		add("c.status = ", f.Status)
	}
	if f.TypeKey != "" {
		add("c.type_key = ", f.TypeKey)
	} else if f.ExcludeGallery {
		kosul += " AND c.type_key NOT IN ('foto_galeri','video')"
	}
	if f.CategoryID != 0 {
		add("c.category_id = ", f.CategoryID)
	}
	if f.SiteID > 0 {
		add("c.site_id = ", f.SiteID)
	}
	if f.Locale != "" {
		// Dil filtresi artık "bu dilde sürümü OLAN makaleler" demek: satır
		// yine temsilci dilde gösterilir.
		n++
		args = append(args, f.Locale)
		kosul += ` AND EXISTS (
			SELECT 1 FROM contents l
			 WHERE COALESCE(l.translation_of, l.id) = COALESCE(c.translation_of, c.id)
			   AND l.locale = $` + itoa(n) + `)`
	}
	if f.Q != "" {
		// Arama TÜM dillerde: yoksa İngilizce başlıkla arayan editör hiçbir
		// şey bulamaz (satır TR temsilci olduğu için başlık eşleşmezdi).
		n++
		args = append(args, "%"+f.Q+"%")
		kosul += ` AND EXISTS (
			SELECT 1 FROM contents q
			 WHERE COALESCE(q.translation_of, q.id) = COALESCE(c.translation_of, c.id)
			   AND (q.title ILIKE $` + itoa(n) + ` OR q.spot ILIKE $` + itoa(n) + `))`
	}
	if f.LastDays > 0 {
		add("c.updated_at >= ", time.Now().AddDate(0, 0, -f.LastDays))
	}
	if f.Baslangic != nil {
		add("c.updated_at >= ", *f.Baslangic)
	}
	if f.Bitis != nil {
		add("c.updated_at < ", *f.Bitis)
	}
	switch f.Audio {
	case AudioHas:
		kosul += ` AND EXISTS (SELECT 1 FROM audio_transcripts at
			WHERE at.content_id = c.id AND COALESCE(at.text,'') <> '')`
	case AudioPending:
		kosul += ` AND EXISTS (SELECT 1 FROM transcription_jobs tj
			WHERE tj.content_id = c.id AND tj.status IN ('queued','running'))`
	case AudioFailed:
		kosul += ` AND EXISTS (SELECT 1 FROM transcription_jobs tj
			WHERE tj.content_id = c.id AND tj.status = 'failed')
			AND NOT EXISTS (SELECT 1 FROM audio_transcripts at
				WHERE at.content_id = c.id AND COALESCE(at.text,'') <> '')`
	}
	if ms := normalizeMinScore(f.MinScore); ms > 0 {
		add("COALESCE(c.ai_score,0) >= ", ms)
	}

	if f.PerPage <= 0 {
		f.PerPage = defaultPerPage
	}
	if f.Page <= 0 {
		f.Page = 1
	}

	q := `
		WITH sec AS (
			SELECT DISTINCT ON (COALESCE(c.translation_of, c.id))
			       c.id
			  FROM contents c
			 WHERE ` + kosul + `
			 ORDER BY COALESCE(c.translation_of, c.id),
			          (c.locale = $1) DESC,
			          c.updated_at DESC,
			          c.id DESC
		)
		SELECT ` + contentCols + `, count(*) OVER()
		  FROM sec
		  JOIN contents c ON c.id = sec.id
		  LEFT JOIN categories cat ON cat.id = c.category_id
		  LEFT JOIN users u ON u.id = c.author_id
		  LEFT JOIN mahrets m ON m.id = c.mahrep_id
		 ORDER BY ` + contentOrderBy(f.Sort) + `
		 LIMIT $` + itoa(n+1) + ` OFFSET $` + itoa(n+2)
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Content
	total := 0
	for rows.Next() {
		c, err := scanContentWithTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		total = c.Total
		out = append(out, c.Content)
	}
	return out, total, rows.Err()
}

// DilSurum bir makalenin tek dil sürümüdür (liste rozetleri için).
type DilSurum struct {
	ID     int64
	Locale string
	Status string
}

// DillerOf verilen kök içeriklerin dil sürümlerini TEK sorguda döndürür.
//
// Neden tek sorgu: liste sayfası 40 satır; satır başına sorgu N+1 olurdu.
// Koşul `id = ANY(...) OR translation_of = ANY(...)` yazıldı: birincisi PK,
// ikincisi `ix_contents_translation` kısmi indeksini kullanır (ifade
// üzerinden yazmak ikisini de devre dışı bırakıyordu).
func DillerOf(ctx context.Context, pool *pgxpool.Pool, kokIDs []int64) map[int64][]DilSurum {
	out := map[int64][]DilSurum{}
	if len(kokIDs) == 0 {
		return out
	}
	rows, err := pool.Query(ctx, `
		SELECT COALESCE(translation_of, id) AS kok_id, id, locale, status
		  FROM contents
		 WHERE id = ANY($1) OR translation_of = ANY($1)
		 ORDER BY kok_id, locale`, kokIDs)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var kok int64
		var d DilSurum
		if err := rows.Scan(&kok, &d.ID, &d.Locale, &d.Status); err != nil {
			return out
		}
		out[kok] = append(out[kok], d)
	}
	return out
}

type contentRow struct {
	Content
	Total int
}

func scanContentWithTotal(row pgx.Row) (contentRow, error) {
	var c Content
	var catName, catSlug *string
	var mahrepName *string
	var total int
	err := row.Scan(
		&c.ID, &c.TypeKey, &c.Status, &c.Title, &c.Spot, &c.Detail, &c.Slug,
		&c.CategoryID, &c.AuthorID, &c.MahrepID, &c.CoverMediaID, &c.MansetMediaID, &c.SocialMediaID,
		&c.IsAM, &c.IsUM, &c.IsDM, &c.IsSD, &c.IsFH, &c.IsHK,
		&c.SourceURL, &c.SourceName, &c.CommentStatus, &c.IsExclusive,
		&c.SeoTitle, &c.SeoDesc, &c.CanonicalURL,
		&c.PublishAt, &c.PublishedAt, &c.Version, &c.CreatedAt, &c.UpdatedAt, &c.Locale, &c.TranslationOf, &c.SiteID,
		&c.IsYMYL, &c.YMYLUzman, &c.YMYLDurumu, &c.AiScore,
		&catName, &catSlug, &c.AuthorName, &mahrepName, &total,
	)
	if err != nil {
		return contentRow{}, err
	}
	c.CategoryName, c.CategorySlug = catName, catSlug
	c.MahrepName = mahrepName
	return contentRow{Content: c, Total: total}, nil
}

// CountByStatus durumlara göre adet (panel rozetleri için).
func CountByStatus(ctx context.Context, pool *pgxpool.Pool, siteID int64) (map[string]int, error) {
	out := map[string]int{}
	rows, err := pool.Query(ctx, `SELECT status, count(*) FROM contents WHERE site_id = $1 GROUP BY status`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

// PublishedBySlug kategori slug'ıyla içerik listesi (genel site).
func PublishedBySlug(ctx context.Context, pool *pgxpool.Pool, siteID int64, categorySlug, locale string, limit int) ([]Content, error) {
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.status = 'yayinda' AND c.locale = $1 AND c.site_id = $2
		  AND (cat.slug = $3 OR cat.parent_id = (
		        SELECT pc.id FROM categories pc WHERE pc.slug = $3 AND pc.site_id = $2 LIMIT 1))
		ORDER BY c.published_at DESC LIMIT $4`
	rows, err := pool.Query(ctx, q, locale, siteID, categorySlug, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// RecentPublished son yayınlananlar (anasayfa + JSON feed).

// ListVisualNews yayındaki, görseli olan haberleri en yeniden eskiye döner.
// Galeri içeriği türetmek için kullanılır (cmd/seed).
func ListVisualNews(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]Content, error) {
	if limit <= 0 {
		limit = 10
	}
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.site_id = $1 AND c.status = 'yayinda' AND c.type_key = 'haber'
		  AND c.cover_media_id IS NOT NULL
		ORDER BY c.published_at DESC NULLS LAST LIMIT $2`
	rows, err := pool.Query(ctx, q, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Content{}
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func RecentPublished(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale string, limit int) ([]Content, error) {
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.status = 'yayinda' AND c.locale = $1 AND c.site_id = $2
		ORDER BY c.published_at DESC LIMIT $3`
	rows, err := pool.Query(ctx, q, locale, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// LangRef sitemap hreflang için hafif dil referansı.
type LangRef struct {
	ID     int64
	RootID int64
	Locale string
	Slug   string
}

// PublishedLangRefs sitenin tüm yayınlanan içeriklerinin dil referanslarını döndürür (hreflang için tek sorgu).
func PublishedLangRefs(ctx context.Context, pool *pgxpool.Pool, siteID int64) ([]LangRef, error) {
	rows, err := pool.Query(ctx, `SELECT c.id, COALESCE(c.translation_of, c.id), c.locale, COALESCE(c.slug, '')
		FROM contents c WHERE c.status = 'yayinda' AND c.site_id = $1`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LangRef
	for rows.Next() {
		var r LangRef
		if err := rows.Scan(&r.ID, &r.RootID, &r.Locale, &r.Slug); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RecentPublishedAllLang tüm dillerdeki son yayınları döndürür (Google News sitemap).
func RecentPublishedAllLang(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]Content, error) {
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.status = 'yayinda' AND c.site_id = $1
		ORDER BY c.published_at DESC LIMIT $2`
	rows, err := pool.Query(ctx, q, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Highlights son dakika + manşet + trend kombinasyonu (ana sayfa).
type Highlights struct {
	Breaking []Content // is_sd
	Lead     []Content // is_am
	Recent   []Content
}

// HomeHighlights ana sayfa bloklarını toplar.
func HomeHighlights(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale string, leadLimit, breakingLimit, recentLimit int) (*Highlights, error) {
	h := &Highlights{}
	// Provizyon: tek tek sorgu (dinamik kolon güvenli: sabit whitesetle)
	run := func(col string, limit int) error {
		qq := `SELECT ` + contentCols + ` FROM contents c
			LEFT JOIN categories cat ON cat.id = c.category_id
			LEFT JOIN users u ON u.id = c.author_id
			LEFT JOIN mahrets m ON m.id = c.mahrep_id
			WHERE c.status = 'yayinda' AND c.locale = $1 AND c.site_id = $2 AND c.` + col + ` = true
			AND (COALESCE(cat.homepage_min_score, 0) <= c.ai_score)
			ORDER BY c.published_at DESC LIMIT $3`
		rows, err := pool.Query(ctx, qq, locale, siteID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		list := []Content{}
		for rows.Next() {
			c, err := scanContent(rows)
			if err != nil {
				return err
			}
			list = append(list, c)
		}
		switch col {
		case "is_sd":
			h.Breaking = list
		case "is_am":
			h.Lead = list
		}
		return rows.Err()
	}
	if err := run("is_sd", breakingLimit); err != nil {
		return nil, err
	}
	if err := run("is_am", leadLimit); err != nil {
		return nil, err
	}
	rec, err := RecentPublished(ctx, pool, siteID, locale, recentLimit)
	if err != nil {
		return nil, err
	}
	// anasayfa kapısı: kategori homepage_min_score eşiğinin altındaki içerikler çıkarılır
	if len(rec) > 0 {
		keep := rec[:0]
		for _, c := range rec {
			ok, err := homepageAllowed(ctx, pool, c)
			if err != nil || ok {
				keep = append(keep, c)
			}
		}
		rec = keep
	}
	h.Recent = rec
	return h, nil
}

// homepageAllowed içeriğin anasayfa eşiğini geçip geçmediğini kontrol eder.
func homepageAllowed(ctx context.Context, pool *pgxpool.Pool, c Content) (bool, error) {
	var min int
	err := pool.QueryRow(ctx, `SELECT COALESCE(homepage_min_score,0) FROM categories WHERE id = $1`, c.CategoryID).Scan(&min)
	if err != nil {
		return true, nil // kategori yok → engelleme
	}
	return c.AiScore >= min, nil
}

// DisplaySlug boşsa güvenli boş döndürür (şablon linkleri için).
func (c Content) DisplaySlug() string {
	if c.Slug == nil {
		return ""
	}
	return *c.Slug
}

// TranslationRootID çeviri zincirinin kök id'sini döndürür.
func (c Content) TranslationRootID() int64 {
	if c.TranslationOf == nil {
		return c.ID
	}
	return *c.TranslationOf
}

// TranslationsOf çeviri grubunu (kendisi + çevirileri) döndürür.
func TranslationsOf(ctx context.Context, pool *pgxpool.Pool, contentID int64) ([]Content, error) {
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.id = $1 OR c.translation_of = $1 ORDER BY c.locale`
	rows, err := pool.Query(ctx, q, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetContentCategory içeriğin kategorisini değiştirir (toplu taşıma — Faz 26).
// catID <= 0 ise kategori boşaltılır. Satır yoksa (içerik silinmiş olabilir)
// ErrNotFound döner; toplu işlem raporunda "işlendi" denmesin diye önemli.
func SetContentCategory(ctx context.Context, pool *pgxpool.Pool, id, catID int64) error {
	var v any
	if catID > 0 {
		v = catID
	}
	tag, err := pool.Exec(ctx, `UPDATE contents SET category_id=$2, updated_at=now() WHERE id=$1`, id, v)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SaveContentDraft yalnızca metin alanlarını (başlık/özet/gövde) yazar ve
// updated_at'i ilerletir (Faz 27 — düzenleyici otomatik taslak kaydetme).
//
// BİLEREK sade tutulur: durum değişmez, audit yazılmaz, otomatik kapak/AI/
// sosyal tetiklenmez. Amaç, tarayıcı çökerse metnin kaybolmaması — kaydet
// düğmesinin işi normal kaydetmedir. Sürüm yine artırılır ve beklenen sürüm
// kontrol edilir; böylece iki editörün autosave'i birbirini sessizce ezmez.
func SaveContentDraft(ctx context.Context, pool *pgxpool.Pool, id, expectedVersion int64, title, spot string, detail []byte) (int, error) {
	var version int
	err := pool.QueryRow(ctx, `
		UPDATE contents SET title=$3, spot=$4, detail=$5::jsonb, version=version+1, updated_at=now()
		WHERE id=$1 AND version=$2
		RETURNING version`, id, expectedVersion, title, spot, orEmptyJSON(detail)).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contents WHERE id=$1)`, id).Scan(&exists)
		if !exists {
			return 0, ErrNotFound
		}
		return 0, ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return version, nil
}

// SetContentPublishAt içeriğin planlanmış yayın saatini yazar (Faz 32 — toplu
// planla). t==nil ise publish_at NULL'a çekilir (planlama kaldırılır).
func SetContentPublishAt(ctx context.Context, pool *pgxpool.Pool, id int64, t *time.Time) error {
	var v any
	if t != nil {
		v = *t
	}
	tag, err := pool.Exec(ctx, `UPDATE contents SET publish_at=$2, version=version+1, updated_at=now() WHERE id=$1`, id, v)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetContentRozet içeriğin editorial rozetini (AM/UM/DM/SD/FH/HK) açar/kapatır
// (Faz 39 — toplu rozet). Kod sabit whitelist'ten; sütun dinamik birleştirilmez.
func SetContentRozet(ctx context.Context, pool *pgxpool.Pool, id int64, code string, on bool) error {
	col := ""
	switch code {
	case "AM":
		col = "is_am"
	case "UM":
		col = "is_um"
	case "DM":
		col = "is_dm"
	case "SD":
		col = "is_sd"
	case "FH":
		col = "is_fh"
	case "HK":
		col = "is_hk"
	default:
		return ErrNotFound
	}
	tag, err := pool.Exec(ctx, `UPDATE contents SET `+col+`=$2, updated_at=now() WHERE id=$1`, id, on)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteContent siler (üzere ya da kalıcı; handler karar verir).
func DeleteContent(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	tag, err := pool.Exec(ctx, `DELETE FROM contents WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ClearFuturePublish anında yayın için ileri planlama tarihini temizler.
func ClearFuturePublish(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `UPDATE contents SET publish_at = NULL, version = version + 1, updated_at = now() WHERE id = $1`, id)
	return err
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

func orEmptyJSON(b []byte) string {
	if len(b) == 0 {
		return "[]"
	}
	return string(b)
}

func nullableInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullableInt64(p *int64) any { return nullableInt(p) }

func nullableText(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
