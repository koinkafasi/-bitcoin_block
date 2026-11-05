package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Person, yazar/editör/muhabir profilidir (users tablosunun genişletilmiş hâli).
type Person struct {
	ID            int64
	Name          string
	Email         string
	Role          string
	Status        string
	Slug          string
	Title         string
	Bio           string
	AvatarMediaID *int64
	Socials       map[string]string
	// ArticleCount yayındaki yazı sayısı (yazar veya editör olarak).
	ArticleCount int
}

const personCols = `id, name, email, role, status, coalesce(slug,''), coalesce(title,''),
	coalesce(bio,''), avatar_media_id, socials`

func scanPerson(row pgx.Row, withCount bool) (Person, error) {
	var p Person
	var socials []byte
	dest := []any{&p.ID, &p.Name, &p.Email, &p.Role, &p.Status, &p.Slug, &p.Title, &p.Bio, &p.AvatarMediaID, &socials}
	if withCount {
		dest = append(dest, &p.ArticleCount)
	}
	err := row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Person{}, ErrNotFound
	}
	if err != nil {
		return Person{}, err
	}
	p.Socials = map[string]string{}
	if len(socials) > 0 {
		_ = json.Unmarshal(socials, &p.Socials)
	}
	return p, nil
}

// PersonSlug addan URL dostu slug üretir (Türkçe karakterler sadeleşir).
func PersonSlug(name string) string {
	repl := strings.NewReplacer(
		"ç", "c", "Ç", "c", "ğ", "g", "Ğ", "g", "ı", "i", "İ", "i",
		"ö", "o", "Ö", "o", "ş", "s", "Ş", "s", "ü", "u", "Ü", "u",
		"â", "a", "Â", "a", "î", "i", "Î", "i", "û", "u", "Û", "u",
	)
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(repl.Replace(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// ListPeople verilen rollerdeki aktif kişileri yazı sayısıyla listeler.
func ListPeople(ctx context.Context, pool *pgxpool.Pool, roles []string) ([]Person, error) {
	if len(roles) == 0 {
		roles = []string{"Yönetici", "Editör", "Muhabir", "Yazar"}
	}
	rows, err := pool.Query(ctx, `SELECT `+personCols+`, coalesce(x.cnt, 0)
		FROM users
		LEFT JOIN LATERAL (
			SELECT count(*) AS cnt FROM contents c
			WHERE c.status = 'yayinda' AND (c.author_id = users.id OR c.editor_id = users.id)
		) x ON true
		WHERE status = 'aktif' AND role = ANY($1)
		ORDER BY name`, roles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Person
	for rows.Next() {
		p, err := scanPerson(rows, true)
		if err != nil {
			return nil, err
		}
		if p.Slug == "" {
			p.Slug = "yazar-" + itoa64(p.ID)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListPeopleAdmin personel profillerini (pasifler dahil) yönetim için listeler.
func ListPeopleAdmin(ctx context.Context, pool *pgxpool.Pool) ([]Person, error) {
	rows, err := pool.Query(ctx, `SELECT `+personCols+`, coalesce(x.cnt, 0)
		FROM users
		LEFT JOIN LATERAL (
			SELECT count(*) AS cnt FROM contents c
			WHERE c.status = 'yayinda' AND (c.author_id = users.id OR c.editor_id = users.id)
		) x ON true
		WHERE role = ANY($1)
		ORDER BY name`, []string{"Yönetici", "Editör", "Muhabir", "Yazar"})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Person
	for rows.Next() {
		p, err := scanPerson(rows, true)
		if err != nil {
			return nil, err
		}
		if p.Slug == "" {
			p.Slug = "yazar-" + itoa64(p.ID)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPersonBySlug slug ile kişi döndürür.
func GetPersonBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (Person, error) {
	return scanPerson(pool.QueryRow(ctx,
		`SELECT `+personCols+` FROM users WHERE (slug = $1 OR $1 = 'yazar-'||id) AND status <> 'pasif'`, slug), false)
}

// GetPersonByID id ile kişi döndürür.
func GetPersonByID(ctx context.Context, pool *pgxpool.Pool, id int64) (Person, error) {
	return scanPerson(pool.QueryRow(ctx, `SELECT `+personCols+` FROM users WHERE id = $1`, id), false)
}

// UpdatePersonProfile profil alanlarını günceller.
func UpdatePersonProfile(ctx context.Context, pool *pgxpool.Pool, p Person) error {
	socials, err := json.Marshal(p.Socials)
	if err != nil {
		return err
	}
	slug := strings.TrimSpace(p.Slug)
	if slug == "" {
		slug = PersonSlug(p.Name)
	}
	if slug == "" {
		slug = "kisi"
	}
	_, err = pool.Exec(ctx, `UPDATE users SET
			name = $2, slug = $3, title = $4, bio = $5,
			avatar_media_id = $6, socials = $7, role = $8, status = $9
		WHERE id = $1`,
		p.ID, p.Name, slug, p.Title, p.Bio, p.AvatarMediaID, socials, p.Role, p.Status)
	return err
}

// EnsurePersonSlug slug'ı boş olan kullanıcıya adından üretir.
func EnsurePersonSlug(ctx context.Context, pool *pgxpool.Pool, id int64, name string) (string, error) {
	slug := PersonSlug(name)
	if slug == "" {
		return "", nil
	}
	// Çakışmada sona id eklenir (benzersizlik indeksi korunur).
	_, err := pool.Exec(ctx, `UPDATE users SET slug = $2 WHERE id = $1 AND (slug IS NULL OR slug = '')`,
		id, slug)
	if err != nil {
		_, err = pool.Exec(ctx, `UPDATE users SET slug = $2 WHERE id = $1 AND (slug IS NULL OR slug = '')`,
			id, slug+"-"+itoa64(id))
	}
	return slug, err
}

// PersonRef, yazı sayfası künyesinde gösterilecek hafif kişi referansıdır.
type PersonRef struct {
	ID   int64
	Name string
	Slug string
}

// PersonRefByID künye için id ile ad + slug döndürür.
func PersonRefByID(ctx context.Context, pool *pgxpool.Pool, id int64) (PersonRef, error) {
	var p PersonRef
	err := pool.QueryRow(ctx, `SELECT id, name,
			CASE WHEN coalesce(slug,'') = '' THEN 'yazar-'||id ELSE slug END
		FROM users WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return PersonRef{}, ErrNotFound
	}
	return p, err
}

// ContentEditor içeriğin editörünü (atanmışsa) döndürür.
func ContentEditor(ctx context.Context, pool *pgxpool.Pool, contentID int64) (*PersonRef, error) {
	var p PersonRef
	err := pool.QueryRow(ctx, `SELECT u.id, u.name,
			CASE WHEN coalesce(u.slug,'') = '' THEN 'yazar-'||u.id ELSE u.slug END
		FROM contents c JOIN users u ON u.id = c.editor_id WHERE c.id = $1`, contentID).
		Scan(&p.ID, &p.Name, &p.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SetContentEditor içeriğin editörünü atar (nil = kaldır).
func SetContentEditor(ctx context.Context, pool *pgxpool.Pool, contentID int64, editorID *int64) error {
	_, err := pool.Exec(ctx, `UPDATE contents SET editor_id = $2 WHERE id = $1`, contentID, editorID)
	return err
}

// SetContentAuthor içeriğin yazarını değiştirir.
func SetContentAuthor(ctx context.Context, pool *pgxpool.Pool, contentID, authorID int64) error {
	_, err := pool.Exec(ctx, `UPDATE contents SET author_id = $2 WHERE id = $1`, contentID, authorID)
	return err
}

// PersonContents kişinin yazdığı/düzenlediği yayındaki içerikleri döndürür.
func PersonContents(ctx context.Context, pool *pgxpool.Pool, personID int64, limit, offset int) ([]Content, int, error) {
	var total int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM contents
		WHERE status = 'yayinda' AND (author_id = $1 OR editor_id = $1)`, personID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := pool.Query(ctx, `SELECT id FROM contents
		WHERE status = 'yayinda' AND (author_id = $1 OR editor_id = $1)
		ORDER BY coalesce(published_at, created_at) DESC
		LIMIT $2 OFFSET $3`, personID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	out := make([]Content, 0, len(ids))
	for _, id := range ids {
		c, err := GetContent(ctx, pool, id)
		if err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, total, nil
}
