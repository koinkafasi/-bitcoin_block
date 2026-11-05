package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Conversation AI sohbet başlığıdır.
type Conversation struct {
	ID        int64
	UserID    int64
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Message tek sohbet mesajıdır.
type Message struct {
	ID             int64
	ConversationID int64
	Role           string
	Content        string
	CreatedAt      time.Time
}

// AITemplate hazır sohbet şablonudur.
type AITemplate struct {
	ID        int64
	Kind      string
	Title     string
	Prompt    string
	Icon      string
	SortOrder int
}

// CreateConversation yeni sohbet açar.
func CreateConversation(ctx context.Context, pool *pgxpool.Pool, userID int64, title string) (int64, error) {
	if title == "" {
		title = "Yeni sohbet"
	}
	var id int64
	err := pool.QueryRow(ctx,
		`INSERT INTO ai_conversations (user_id, title) VALUES ($1,$2) RETURNING id`, userID, title).Scan(&id)
	return id, err
}

// ListConversations kullanıcının sohbetlerini (en yeni üstte) döndürür.
func ListConversations(ctx context.Context, pool *pgxpool.Pool, userID int64) ([]Conversation, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, user_id, title, created_at, updated_at
		FROM ai_conversations WHERE user_id = $1 ORDER BY updated_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetConversation sahiplik kontrolüyle sohbeti döndürür.
func GetConversation(ctx context.Context, pool *pgxpool.Pool, userID, id int64) (Conversation, error) {
	var c Conversation
	err := pool.QueryRow(ctx,
		`SELECT id, user_id, title, created_at, updated_at FROM ai_conversations WHERE id = $1 AND user_id = $2`,
		id, userID).Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	return c, err
}

// AddMessage sohbete mesaj ekler ve sohbet sırasını günceller.
func AddMessage(ctx context.Context, pool *pgxpool.Pool, convID int64, role, content string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO ai_messages (conversation_id, role, content) VALUES ($1,$2,$3) RETURNING id`,
		convID, role, content).Scan(&id)
	if err != nil {
		return 0, err
	}
	_, err = pool.Exec(ctx, `UPDATE ai_conversations SET updated_at = now(),
		title = CASE WHEN title = 'Yeni sohbet' THEN left($2, 60) ELSE title END WHERE id = $1`, convID, content)
	return id, err
}

// ListMessages sohbet mesajlarını (bağlam: son N) döndürür.
func ListMessages(ctx context.Context, pool *pgxpool.Pool, convID int64, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 40
	}
	rows, err := pool.Query(ctx, `
		SELECT id, conversation_id, role, content, created_at
		FROM ai_messages WHERE conversation_id = $1 ORDER BY created_at ASC`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		all = append(all, m)
	}
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, rows.Err()
}

// DeleteConversation kullanıcının sohbetini (mesajlarıyla) siler.
func DeleteConversation(ctx context.Context, pool *pgxpool.Pool, userID, id int64) error {
	tag, err := pool.Exec(ctx,
		`DELETE FROM ai_conversations WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListAITemplates hazır sohbet şablonlarını döndürür.
func ListAITemplates(ctx context.Context, pool *pgxpool.Pool, kind string) ([]AITemplate, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, kind, title, prompt, icon, sort_order FROM ai_templates WHERE kind = $1 ORDER BY sort_order, id`,
		kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AITemplate
	for rows.Next() {
		var t AITemplate
		if err := rows.Scan(&t.ID, &t.Kind, &t.Title, &t.Prompt, &t.Icon, &t.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
