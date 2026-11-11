package repo

import (
	"context"
	"encoding/json"

	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Anketler (spec 02 §26 / 01 §H14) — contents_extra içinde saklanır ----

// PollOption tek bir anket seçeneği.
type PollOption struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Votes int    `json:"votes"`
}

// Poll bir anketin seçenekleri.
type Poll struct {
	Options []PollOption `json:"options"`
}

// Total toplam oy sayısı.
func (p Poll) Total() int {
	n := 0
	for _, o := range p.Options {
		n += o.Votes
	}
	return n
}

// GetPoll içeriğin anket verisini okur.
func GetPoll(ctx context.Context, pool *pgxpool.Pool, contentID int64) (Poll, error) {
	var raw []byte
	err := pool.QueryRow(ctx, `SELECT extra FROM contents_extra WHERE content_id=$1`, contentID).Scan(&raw)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Poll{}, err
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	var p Poll
	if m != nil {
		if opts, ok := m["options"]; ok {
			_ = json.Unmarshal(opts, &p.Options)
		}
	}
	return p, nil
}

// SavePoll anket seçeneklerini yazar (diğer extra anahtarlarını korur).
func SavePoll(ctx context.Context, pool *pgxpool.Pool, contentID int64, p Poll) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := writePollExtra(ctx, tx, contentID, p); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// VotePoll bir seçeneğe oy ekler (satır kilidi ile güvenli) ve güncel anketi döndürür.
func VotePoll(ctx context.Context, pool *pgxpool.Pool, contentID int64, optionID string) (Poll, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Poll{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT extra FROM contents_extra WHERE content_id=$1 FOR UPDATE`, contentID).Scan(&raw); err != nil {
		return Poll{}, ErrNotFound
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	var p Poll
	if m != nil {
		if opts, ok := m["options"]; ok {
			_ = json.Unmarshal(opts, &p.Options)
		}
	}
	found := false
	for i := range p.Options {
		if p.Options[i].ID == optionID {
			p.Options[i].Votes++
			found = true
			break
		}
	}
	if !found {
		return Poll{}, ErrNotFound
	}
	if err := writePollExtra(ctx, tx, contentID, p); err != nil {
		return Poll{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Poll{}, err
	}
	return p, nil
}

// writePollExtra extra JSON'unu options anahtarını güncelleyerek upsert eder.
func writePollExtra(ctx context.Context, tx pgx.Tx, contentID int64, p Poll) error {
	var raw []byte
	_ = tx.QueryRow(ctx, `SELECT extra FROM contents_extra WHERE content_id=$1`, contentID).Scan(&raw)
	m := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	m["options"] = p.Options
	b, _ := json.Marshal(m)
	_, err := tx.Exec(ctx, `
		INSERT INTO contents_extra (content_id, extra) VALUES ($1, $2::jsonb)
		ON CONFLICT (content_id) DO UPDATE SET extra = EXCLUDED.extra`, contentID, b)
	return err
}

// ListPolls siteye ait yayındaki anketleri (içerik + seçenekler) döndürür.
type PollItem struct {
	Content Content
	Poll    Poll
}

// ListPolls yayındaki anket içeriklerini seçenekleriyle döndürür.
func ListPolls(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]PollItem, error) {
	if limit <= 0 {
		limit = 50
	}
	items, _, err := ListContents(ctx, pool, ListFilter{TypeKey: "anket", Status: "yayinda", SiteID: siteID, PerPage: limit, Page: 1})
	if err != nil {
		return nil, err
	}
	out := make([]PollItem, 0, len(items))
	for _, c := range items {
		p, _ := GetPoll(ctx, pool, c.ID)
		out = append(out, PollItem{Content: c, Poll: p})
	}
	return out, nil
}
