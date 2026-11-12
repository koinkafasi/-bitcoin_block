package repo

import (
	"context"
	"time"
)

// ---- İletişim formu mesajları (Faz 145) ----

// ContactMessage iletişim formundan gelen okuyucu mesajı.
type ContactMessage struct {
	ID        int64
	SiteID    int64
	Name      string
	Email     string
	Phone     string
	Subject   string
	Body      string
	Status    string
	IP        string
	UserAgent string
	CreatedAt time.Time
}

// AddContactMessage yeni mesaj kaydeder.
func AddContactMessage(ctx context.Context, pool DB, m ContactMessage) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO contact_messages (site_id, name, email, phone, subject, message, ip, user_agent)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		m.SiteID, m.Name, m.Email, m.Phone, m.Subject, m.Body, m.IP, m.UserAgent).Scan(&id)
	return id, err
}

// ListContactMessages duruma göre mesajları (en yeni önce) döndürür.
func ListContactMessages(ctx context.Context, pool DB, siteID int64, status string, limit int) ([]ContactMessage, error) {
	if limit <= 0 {
		limit = 200
	}
	q := `SELECT id, site_id, name, email, COALESCE(phone,''), subject, message,
	             status, COALESCE(ip,''), COALESCE(user_agent,''), created_at
	      FROM contact_messages WHERE site_id = $1`
	args := []any{siteID}
	if status != "" && status != "tumu" {
		args = append(args, status)
		q += ` AND status = $2`
	}
	q += ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)+1)
	args = append(args, limit)
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContactMessage
	for rows.Next() {
		var m ContactMessage
		if err := rows.Scan(&m.ID, &m.SiteID, &m.Name, &m.Email, &m.Phone, &m.Subject,
			&m.Body, &m.Status, &m.IP, &m.UserAgent, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CountContactMessages durum başına mesaj sayısı.
func CountContactMessages(ctx context.Context, pool DB, siteID int64) (yeni, okundu, arsiv int, err error) {
	rows, err := pool.Query(ctx,
		`SELECT status, COUNT(*) FROM contact_messages WHERE site_id=$1 GROUP BY status`, siteID)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int64
		if err := rows.Scan(&st, &n); err != nil {
			return 0, 0, 0, err
		}
		switch st {
		case "yeni":
			yeni = int(n)
		case "okundu":
			okundu = int(n)
		case "arsiv":
			arsiv = int(n)
		}
	}
	return yeni, okundu, arsiv, rows.Err()
}

// SetContactMessageStatus mesajın durumunu günceller.
func SetContactMessageStatus(ctx context.Context, pool DB, id int64, status string) error {
	_, err := pool.Exec(ctx, `UPDATE contact_messages SET status=$1 WHERE id=$2`, status, id)
	return err
}

// DeleteContactMessage mesajı siler.
func DeleteContactMessage(ctx context.Context, pool DB, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM contact_messages WHERE id=$1`, id)
	return err
}

func ReplaceTextInPages(ctx context.Context, pool DB, from, to string) (int64, int64, error) {
	if from == "" || from == to {
		return 0, 0, nil
	}
	tag, err := pool.Exec(ctx, `
		UPDATE pages
		SET detail = replace(detail::text, $1, $2)::jsonb, updated_at = now()
		WHERE detail::text LIKE '%' || $1 || '%'`, from, to)
	if err != nil {
		return 0, 0, err
	}
	pages := tag.RowsAffected()
	tag2, err := pool.Exec(ctx, `
		UPDATE page_texts SET value = replace(value, $1, $2), updated_at = now()
		WHERE value LIKE '%' || $1 || '%'`, from, to)
	if err != nil {
		return pages, 0, err
	}
	return pages, tag2.RowsAffected(), nil
}
