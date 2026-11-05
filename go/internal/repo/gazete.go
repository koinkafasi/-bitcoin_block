package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Gazete modülü (Faz 1: veri modeli) ----

// SiteFeature site bazlı özellik lisansı (ticari paket anahtarı).
type SiteFeature struct {
	SiteID    int64
	Key       string
	Enabled   bool
	Config    []byte
	UpdatedAt time.Time
}

// SiteFeatureEnabled sitenin özelliğinin açık olup olmadığını döndürür (kayıt yoksa false).
func SiteFeatureEnabled(ctx context.Context, pool DB, siteID int64, key string) (bool, error) {
	var on bool
	err := pool.QueryRow(ctx, `
		SELECT enabled FROM site_features WHERE site_id=$1 AND feature_key=$2`, siteID, key).Scan(&on)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return on, err
}

// ListSiteFeatures sitenin tüm özellik lisanslarını döndürür.
func ListSiteFeatures(ctx context.Context, pool DB, siteID int64) ([]SiteFeature, error) {
	rows, err := pool.Query(ctx, `
		SELECT site_id, feature_key, enabled, config, updated_at
		FROM site_features WHERE site_id=$1 ORDER BY feature_key`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SiteFeature
	for rows.Next() {
		var f SiteFeature
		if err := rows.Scan(&f.SiteID, &f.Key, &f.Enabled, &f.Config, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// SetSiteFeature özellik lisansını açar/kapatır (config opsiyonel JSON).
func SetSiteFeature(ctx context.Context, pool DB, siteID int64, key string, enabled bool, config []byte) error {
	if len(config) == 0 {
		config = []byte("{}")
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO site_features (site_id, feature_key, enabled, config, updated_at)
		VALUES ($1,$2,$3,$4::jsonb, now())
		ON CONFLICT (site_id, feature_key) DO UPDATE SET enabled=EXCLUDED.enabled, config=EXCLUDED.config, updated_at=now()`,
		siteID, key, enabled, config)
	return err
}

// ---- Site depolama yapılandırması (Cloudflare R2 / S3 / yerel) ----

// SiteStorage site bazlı nesne depolama yapılandırmasıdır.
type SiteStorage struct {
	SiteID        int64
	Provider      string // local | r2 | s3 | minio
	Endpoint      string
	Bucket        string
	Region        string
	AccessKeyID   string
	SecretEnc     string // şifreli gizli anahtar (düz metin asla saklanmaz)
	PublicBaseURL string
	PathPrefix    string
	Enabled       bool
	UpdatedAt     time.Time
}

// GetSiteStorage site depolama yapılandırmasını döndürür; yoksa varsayılan (local) döner.
func GetSiteStorage(ctx context.Context, pool DB, siteID int64) (SiteStorage, error) {
	var s SiteStorage
	err := pool.QueryRow(ctx, `
		SELECT site_id, provider, endpoint, bucket, region, access_key_id, secret_enc,
		       public_base_url, path_prefix, enabled, updated_at
		FROM site_storage WHERE site_id=$1`, siteID).
		Scan(&s.SiteID, &s.Provider, &s.Endpoint, &s.Bucket, &s.Region, &s.AccessKeyID,
			&s.SecretEnc, &s.PublicBaseURL, &s.PathPrefix, &s.Enabled, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SiteStorage{SiteID: siteID, Provider: "local", Region: "auto"}, nil
		}
		return SiteStorage{}, err
	}
	return s, nil
}

// UpsertSiteStorage depolama yapılandırmasını yazar/günceller.
func UpsertSiteStorage(ctx context.Context, pool DB, s *SiteStorage) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO site_storage (site_id, provider, endpoint, bucket, region, access_key_id,
			secret_enc, public_base_url, path_prefix, enabled, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now())
		ON CONFLICT (site_id) DO UPDATE SET provider=EXCLUDED.provider, endpoint=EXCLUDED.endpoint,
			bucket=EXCLUDED.bucket, region=EXCLUDED.region, access_key_id=EXCLUDED.access_key_id,
			secret_enc=EXCLUDED.secret_enc, public_base_url=EXCLUDED.public_base_url,
			path_prefix=EXCLUDED.path_prefix, enabled=EXCLUDED.enabled, updated_at=now()`,
		s.SiteID, s.Provider, s.Endpoint, s.Bucket, defRegion(s.Region), s.AccessKeyID,
		s.SecretEnc, s.PublicBaseURL, s.PathPrefix, s.Enabled)
	return err
}

func defRegion(r string) string {
	if r == "" {
		return "auto"
	}
	return r
}

// ---- Gazete şablonları ----

// GazeteTemplate 10 sayfalık gazete düzeni şablonudur.
type GazeteTemplate struct {
	ID          int64
	SiteID      int64
	Name        string
	Description string
	Pages       []byte // JSONB: [{no, ad, tip, bloklar:[...]}]
	ThumbURL    string
	IsDefault   bool
	CreatedBy   *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ListGazeteTemplates site şablonlarını döndürür.
func ListGazeteTemplates(ctx context.Context, pool DB, siteID int64) ([]GazeteTemplate, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, name, description, pages, thumb_url, is_default, created_by, created_at, updated_at
		FROM gazete_templates WHERE site_id=$1 ORDER BY is_default DESC, id DESC`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GazeteTemplate
	for rows.Next() {
		var t GazeteTemplate
		if err := rows.Scan(&t.ID, &t.SiteID, &t.Name, &t.Description, &t.Pages, &t.ThumbURL,
			&t.IsDefault, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetGazeteTemplate şablonu id ile döndürür.
func GetGazeteTemplate(ctx context.Context, pool DB, id int64) (GazeteTemplate, error) {
	var t GazeteTemplate
	err := pool.QueryRow(ctx, `
		SELECT id, site_id, name, description, pages, thumb_url, is_default, created_by, created_at, updated_at
		FROM gazete_templates WHERE id=$1`, id).
		Scan(&t.ID, &t.SiteID, &t.Name, &t.Description, &t.Pages, &t.ThumbURL,
			&t.IsDefault, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GazeteTemplate{}, ErrNotFound
		}
		return GazeteTemplate{}, err
	}
	return t, nil
}

// CreateGazeteTemplate yeni şablon ekler.
func CreateGazeteTemplate(ctx context.Context, pool DB, t *GazeteTemplate) (int64, error) {
	if len(t.Pages) == 0 {
		t.Pages = []byte("[]")
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO gazete_templates (site_id, name, description, pages, thumb_url, is_default, created_by)
		VALUES ($1,$2,$3,$4::jsonb,$5,$6,$7) RETURNING id`,
		t.SiteID, t.Name, t.Description, t.Pages, t.ThumbURL, t.IsDefault, t.CreatedBy).Scan(&id)
	return id, err
}

// UpdateGazeteTemplate şablonu günceller.
func UpdateGazeteTemplate(ctx context.Context, pool DB, t *GazeteTemplate) error {
	if len(t.Pages) == 0 {
		t.Pages = []byte("[]")
	}
	_, err := pool.Exec(ctx, `
		UPDATE gazete_templates SET name=$1, description=$2, pages=$3::jsonb, thumb_url=$4,
			is_default=$5, updated_at=now() WHERE id=$6`,
		t.Name, t.Description, t.Pages, t.ThumbURL, t.IsDefault, t.ID)
	return err
}

// DeleteGazeteTemplate şablonu siler.
func DeleteGazeteTemplate(ctx context.Context, pool DB, id int64) error {
	tag, err := pool.Exec(ctx, `DELETE FROM gazete_templates WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetDefaultGazeteTemplate şablonu sitenin varsayılanı yapar (diğerlerini kaldırır).
func SetDefaultGazeteTemplate(ctx context.Context, pool DB, siteID, id int64) error {
	if _, err := pool.Exec(ctx, `UPDATE gazete_templates SET is_default=false, updated_at=now() WHERE site_id=$1`, siteID); err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, `UPDATE gazete_templates SET is_default=true, updated_at=now() WHERE id=$1 AND site_id=$2`, id, siteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- Gazete sayıları (baskılar) ----

// GazeteIssue tarih damgalı gazete baskısıdır.
type GazeteIssue struct {
	ID               int64
	SiteID           int64
	TemplateID       *int64
	Period           string // gunluk | haftalik | aylik
	IssueDate        string // YYYY-MM-DD
	Title            string
	Status           string // taslak | hazir | yayinda | arsiv
	AssignedEditorID *int64
	CoverMediaID     *int64
	PDFPath          string
	PageCount        int
	CreatedAt        time.Time
	UpdatedAt        time.Time
	PublishedAt      *time.Time
}

// ListGazeteIssues tarih aralığına (opsiyonel) göre baskıları döndürür.
func ListGazeteIssues(ctx context.Context, pool DB, siteID int64, period, from, to string, limit int) ([]GazeteIssue, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
		SELECT i.id, i.site_id, i.template_id, i.period, i.issue_date::text, i.title, i.status,
		       i.assigned_editor_id, i.cover_media_id, i.pdf_path,
		       (SELECT count(*) FROM gazete_pages p WHERE p.issue_id = i.id),
		       i.created_at, i.updated_at, i.published_at
		FROM gazete_issues i
		WHERE i.site_id=$1
		  AND ($2 = '' OR i.period = $2)
		  AND ($3 = '' OR i.issue_date >= $3::date)
		  AND ($4 = '' OR i.issue_date <= $4::date)
		ORDER BY i.issue_date DESC, i.id DESC LIMIT $5`, siteID, period, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GazeteIssue
	for rows.Next() {
		var i GazeteIssue
		if err := rows.Scan(&i.ID, &i.SiteID, &i.TemplateID, &i.Period, &i.IssueDate, &i.Title, &i.Status,
			&i.AssignedEditorID, &i.CoverMediaID, &i.PDFPath, &i.PageCount,
			&i.CreatedAt, &i.UpdatedAt, &i.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// GetGazeteIssue baskıyı id ile döndürür.
func GetGazeteIssue(ctx context.Context, pool DB, id int64) (GazeteIssue, error) {
	var i GazeteIssue
	err := pool.QueryRow(ctx, `
		SELECT i.id, i.site_id, i.template_id, i.period, i.issue_date::text, i.title, i.status,
		       i.assigned_editor_id, i.cover_media_id, i.pdf_path,
		       (SELECT count(*) FROM gazete_pages p WHERE p.issue_id = i.id),
		       i.created_at, i.updated_at, i.published_at
		FROM gazete_issues i WHERE i.id=$1`, id).
		Scan(&i.ID, &i.SiteID, &i.TemplateID, &i.Period, &i.IssueDate, &i.Title, &i.Status,
			&i.AssignedEditorID, &i.CoverMediaID, &i.PDFPath, &i.PageCount,
			&i.CreatedAt, &i.UpdatedAt, &i.PublishedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GazeteIssue{}, ErrNotFound
		}
		return GazeteIssue{}, err
	}
	return i, nil
}

// GetGazeteIssueByDate tarih+periyot ile baskıyı döndürür.
func GetGazeteIssueByDate(ctx context.Context, pool DB, siteID int64, period, date string) (GazeteIssue, error) {
	var i GazeteIssue
	err := pool.QueryRow(ctx, `
		SELECT i.id, i.site_id, i.template_id, i.period, i.issue_date::text, i.title, i.status,
		       i.assigned_editor_id, i.cover_media_id, i.pdf_path,
		       (SELECT count(*) FROM gazete_pages p WHERE p.issue_id = i.id),
		       i.created_at, i.updated_at, i.published_at
		FROM gazete_issues i WHERE i.site_id=$1 AND i.period=$2 AND i.issue_date=$3::date`,
		siteID, period, date).
		Scan(&i.ID, &i.SiteID, &i.TemplateID, &i.Period, &i.IssueDate, &i.Title, &i.Status,
			&i.AssignedEditorID, &i.CoverMediaID, &i.PDFPath, &i.PageCount,
			&i.CreatedAt, &i.UpdatedAt, &i.PublishedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GazeteIssue{}, ErrNotFound
		}
		return GazeteIssue{}, err
	}
	return i, nil
}

// UpsertGazeteIssue baskıyı oluşturur veya (site, periyot, tarih) üzerinde günceller.
func UpsertGazeteIssue(ctx context.Context, pool DB, i *GazeteIssue) (int64, error) {
	if i.Period == "" {
		i.Period = "gunluk"
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO gazete_issues (site_id, template_id, period, issue_date, title, status,
			assigned_editor_id, cover_media_id, pdf_path)
		VALUES ($1,$2,$3,$4::date,$5,$6,$7,$8,$9)
		ON CONFLICT (site_id, period, issue_date) DO UPDATE SET
			template_id=COALESCE(EXCLUDED.template_id, gazete_issues.template_id),
			title=EXCLUDED.title, status=EXCLUDED.status,
			assigned_editor_id=EXCLUDED.assigned_editor_id, cover_media_id=EXCLUDED.cover_media_id,
			pdf_path=EXCLUDED.pdf_path, updated_at=now()
		RETURNING id`,
		i.SiteID, i.TemplateID, i.Period, i.IssueDate, i.Title, i.Status,
		i.AssignedEditorID, i.CoverMediaID, i.PDFPath).Scan(&id)
	return id, err
}

// SetGazeteIssueStatus baskı durumunu günceller (yayında ise published_at damgalanır).
func SetGazeteIssueStatus(ctx context.Context, pool DB, id int64, status string) error {
	_, err := pool.Exec(ctx, `
		UPDATE gazete_issues SET status=$1, updated_at=now(),
			published_at = CASE WHEN $1='yayinda' AND published_at IS NULL THEN now() ELSE published_at END
		WHERE id=$2`, status, id)
	return err
}

// SetGazeteIssueEditor sayıya sorumlu editör atar (0 = atamayı kaldırır).
func SetGazeteIssueEditor(ctx context.Context, pool DB, id, editorID int64) error {
	if editorID == 0 {
		_, err := pool.Exec(ctx, `UPDATE gazete_issues SET assigned_editor_id=NULL, updated_at=now() WHERE id=$1`, id)
		return err
	}
	_, err := pool.Exec(ctx, `UPDATE gazete_issues SET assigned_editor_id=$1, updated_at=now() WHERE id=$2`, editorID, id)
	return err
}

// SetGazeteIssuePDF baskı PDF yolunu kaydeder.
func SetGazeteIssuePDF(ctx context.Context, pool DB, id int64, pdfPath string) error {
	_, err := pool.Exec(ctx, `UPDATE gazete_issues SET pdf_path=$1, updated_at=now() WHERE id=$2`, pdfPath, id)
	return err
}

// DeleteGazeteIssue baskıyı siler (sayfaları cascade).
func DeleteGazeteIssue(ctx context.Context, pool DB, id int64) error {
	tag, err := pool.Exec(ctx, `DELETE FROM gazete_issues WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- Gazete sayfaları ----

// GazetePage baskının tek sayfasıdır (yerleşim blokları JSONB).
type GazetePage struct {
	ID        int64
	IssueID   int64
	PageNo    int
	PageKey   string
	Blocks    []byte // JSONB: [{id,tip,x,y,w,h,opts,icerik...}]
	UpdatedAt time.Time
}

// ListGazetePages baskı sayfalarını sıra ile döndürür.
func ListGazetePages(ctx context.Context, pool DB, issueID int64) ([]GazetePage, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, issue_id, page_no, page_key, blocks, updated_at
		FROM gazete_pages WHERE issue_id=$1 ORDER BY page_no`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GazetePage
	for rows.Next() {
		var p GazetePage
		if err := rows.Scan(&p.ID, &p.IssueID, &p.PageNo, &p.PageKey, &p.Blocks, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SaveGazetePage sayfayı kaydeder/günceller (yoksa oluşturur).
func SaveGazetePage(ctx context.Context, pool DB, issueID int64, pageNo int, pageKey string, blocks []byte) error {
	if len(blocks) == 0 {
		blocks = []byte("[]")
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO gazete_pages (issue_id, page_no, page_key, blocks, updated_at)
		VALUES ($1,$2,$3,$4::jsonb, now())
		ON CONFLICT (issue_id, page_no) DO UPDATE SET page_key=EXCLUDED.page_key,
			blocks=EXCLUDED.blocks, updated_at=now()`,
		issueID, pageNo, pageKey, blocks)
	return err
}

// ---- Gazete abonelikleri ----

// GazeteSubscription özel gazete bülteni abonesidir.
type GazeteSubscription struct {
	ID        int64
	SiteID    int64
	Email     string
	Phone     string
	Channel   string // email | whatsapp
	Frequency string // gunluk | haftalik | aylik
	Status    string // aktif | iptal
	Token     string
	OptInAt   *time.Time
	OptInIP   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListGazeteSubscriptions site abonelerini döndürür (durum filtresi opsiyonel).
func ListGazeteSubscriptions(ctx context.Context, pool DB, siteID int64, status string, limit int) ([]GazeteSubscription, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, email, phone, channel, frequency, status, token, opt_in_at, opt_in_ip, created_at, updated_at
		FROM gazete_subscriptions WHERE site_id=$1 AND ($2='' OR status=$2)
		ORDER BY id DESC LIMIT $3`, siteID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GazeteSubscription
	for rows.Next() {
		var s GazeteSubscription
		if err := rows.Scan(&s.ID, &s.SiteID, &s.Email, &s.Phone, &s.Channel, &s.Frequency,
			&s.Status, &s.Token, &s.OptInAt, &s.OptInIP, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CountGazeteSubscriptions aktif abone sayısını döndürür.
func CountGazeteSubscriptions(ctx context.Context, pool DB, siteID int64) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM gazete_subscriptions WHERE site_id=$1 AND status='aktif'`, siteID).Scan(&n)
	return n, err
}

// CreateGazeteSubscription abone ekler; token yoksa üretir. KVKK opt-in kaydı tutulur.
func CreateGazeteSubscription(ctx context.Context, pool DB, s *GazeteSubscription) (int64, error) {
	if s.Channel == "" {
		s.Channel = "email"
	}
	if s.Frequency == "" {
		s.Frequency = "gunluk"
	}
	if s.Status == "" {
		s.Status = "aktif"
	}
	if s.Token == "" {
		s.Token = rndToken()
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO gazete_subscriptions (site_id, email, phone, channel, frequency, status, token, opt_in_at, opt_in_ip)
		VALUES ($1,$2,$3,$4,$5,$6,$7, COALESCE($8, now()), $9) RETURNING id`,
		s.SiteID, s.Email, s.Phone, s.Channel, s.Frequency, s.Status, s.Token, s.OptInAt, s.OptInIP).Scan(&id)
	return id, err
}

// GazeteSubscriptionByToken abonelik bağlantısı (iptal) için token ile arar.
func GazeteSubscriptionByToken(ctx context.Context, pool DB, token string) (GazeteSubscription, error) {
	var s GazeteSubscription
	err := pool.QueryRow(ctx, `
		SELECT id, site_id, email, phone, channel, frequency, status, token, opt_in_at, opt_in_ip, created_at, updated_at
		FROM gazete_subscriptions WHERE token=$1`, token).
		Scan(&s.ID, &s.SiteID, &s.Email, &s.Phone, &s.Channel, &s.Frequency,
			&s.Status, &s.Token, &s.OptInAt, &s.OptInIP, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GazeteSubscription{}, ErrNotFound
		}
		return GazeteSubscription{}, err
	}
	return s, nil
}

// SetGazeteSubscriptionStatus abone durumunu günceller (aktif/iptal).
func SetGazeteSubscriptionStatus(ctx context.Context, pool DB, id int64, status string) error {
	_, err := pool.Exec(ctx, `
		UPDATE gazete_subscriptions SET status=$1, updated_at=now() WHERE id=$2`, status, id)
	return err
}

// DeleteGazeteSubscription abone kaydını siler (KVKK silme talebi).
func DeleteGazeteSubscription(ctx context.Context, pool DB, id int64) error {
	tag, err := pool.Exec(ctx, `DELETE FROM gazete_subscriptions WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- Gönderim kayıtları ----

// GazeteDelivery baskı-abone gönderim kaydıdır.
type GazeteDelivery struct {
	ID             int64
	IssueID        int64
	SubscriptionID int64
	Channel        string
	Status         string // kuyrukta | gonderildi | hata
	Error          string
	CreatedAt      time.Time
	SentAt         *time.Time
}

// UpsertGazeteDelivery gönderim kaydını oluşturur/günceller (idempotent).
func UpsertGazeteDelivery(ctx context.Context, pool DB, d *GazeteDelivery) (int64, error) {
	if d.Channel == "" {
		d.Channel = "email"
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO gazete_deliveries (issue_id, subscription_id, channel, status, error, sent_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (issue_id, subscription_id) DO UPDATE SET status=EXCLUDED.status,
			error=EXCLUDED.error, sent_at=EXCLUDED.sent_at
		RETURNING id`,
		d.IssueID, d.SubscriptionID, d.Channel, d.Status, d.Error, d.SentAt).Scan(&id)
	return id, err
}

// ListGazeteDeliveries baskının gönderim kayıtlarını döndürür.
func ListGazeteDeliveries(ctx context.Context, pool DB, issueID int64) ([]GazeteDelivery, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, issue_id, subscription_id, channel, status, error, created_at, sent_at
		FROM gazete_deliveries WHERE issue_id=$1 ORDER BY id`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GazeteDelivery
	for rows.Next() {
		var d GazeteDelivery
		if err := rows.Scan(&d.ID, &d.IssueID, &d.SubscriptionID, &d.Channel, &d.Status, &d.Error,
			&d.CreatedAt, &d.SentAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CountGazeteDeliveries durum bazlı gönderim sayısını döndürür.
func CountGazeteDeliveries(ctx context.Context, pool DB, issueID int64, status string) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM gazete_deliveries WHERE issue_id=$1 AND ($2='' OR status=$2)`,
		issueID, status).Scan(&n)
	return n, err
}

// PruneGazeteDeliveries eski gönderim kayıtlarını temizler (KVKK saklama süresi).
func PruneGazeteDeliveries(ctx context.Context, pool DB, olderThanDays int) (int64, error) {
	if olderThanDays <= 0 {
		olderThanDays = 365
	}
	tag, err := pool.Exec(ctx, `
		DELETE FROM gazete_deliveries WHERE created_at < now() - ($1 || ' days')::interval`, strconv.Itoa(olderThanDays))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ---- Gazete ekip yetkileri (site bazlı editör ataması) ----

// GazeteTeamMember site gazetesinde görevli kullanıcıdır.
type GazeteTeamMember struct {
	SiteID    int64
	UserID    int64
	UserName  string
	UserEmail string
	UserRole  string
	Rol       string // sorumlu | editor | izleyici
	CreatedAt time.Time
}

// ListGazeteTeam sitenin gazete ekibini kullanıcı bilgileriyle döndürür.
func ListGazeteTeam(ctx context.Context, pool DB, siteID int64) ([]GazeteTeamMember, error) {
	rows, err := pool.Query(ctx, `
		SELECT g.site_id, g.user_id, u.name, u.email::text, u.role, g.rol, g.created_at
		FROM gazete_user_roles g JOIN users u ON u.id = g.user_id
		WHERE g.site_id=$1 ORDER BY g.rol, u.name`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GazeteTeamMember
	for rows.Next() {
		var m GazeteTeamMember
		if err := rows.Scan(&m.SiteID, &m.UserID, &m.UserName, &m.UserEmail, &m.UserRole, &m.Rol, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SetGazeteTeamRole kullanıcıya site gazetesinde rol atar (yoksa oluşturur).
func SetGazeteTeamRole(ctx context.Context, pool DB, siteID, userID int64, rol string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO gazete_user_roles (site_id, user_id, rol) VALUES ($1,$2,$3)
		ON CONFLICT (site_id, user_id) DO UPDATE SET rol=EXCLUDED.rol`, siteID, userID, rol)
	return err
}

// RemoveGazeteTeamRole atamayı kaldırır.
func RemoveGazeteTeamRole(ctx context.Context, pool DB, siteID, userID int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM gazete_user_roles WHERE site_id=$1 AND user_id=$2`, siteID, userID)
	return err
}

// GazeteTeamRole kullanıcının sitedeki gazete rolünü döndürür (yoksa boş).
func GazeteTeamRole(ctx context.Context, pool DB, siteID, userID int64) (string, error) {
	var rol string
	err := pool.QueryRow(ctx, `
		SELECT rol FROM gazete_user_roles WHERE site_id=$1 AND user_id=$2`, siteID, userID).Scan(&rol)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return rol, err
}

func rndToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
