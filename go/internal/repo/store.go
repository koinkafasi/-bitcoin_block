package repo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB veri katmanının ihtiyaç duyduğu minimal yüzey.
//
// Neden arayüz: repo fonksiyonlarına `*pgxpool.Pool` geçirilmesi onların
// yalnızca gerçek Postgres'e karşı test edilmesini zorlar (yavaş, state'li).
// Bu arayüz `*pgxpool.Pool` tarafından sağlanır, dolayısıyla hiçbir çağrı
// noktası değişmez; testlerde sahte (mock) uygulama geçilebilir.
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// DB arayüzünü sağladığını derleyiciye doğrulat (mock'lar için).
var _ DB = (*pgxpool.Pool)(nil)
