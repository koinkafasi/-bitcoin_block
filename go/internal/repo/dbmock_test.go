package repo

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dbCall struct {
	Kind string // "queryrow" | "query" | "exec"
	SQL  string
	Args []any
}

// fakeDB, DB arayüzünü implemente eder. Yanıtlar verilen fonksiyonlarla
// belirlenir; tanımlanmazsa "satır yok" / boş sonuç döner.
type fakeDB struct {
	// rowFn, Scan hedeflerine yazmak zorundadır.
	rowFn   func(sql string, args []any, dest ...any) error
	queryFn func(sql string, args []any) ([][]any, error)
	execFn  func(sql string, args []any) error

	mu    sync.Mutex
	calls []dbCall
}

var _ DB = (*fakeDB)(nil)

func (f *fakeDB) record(kind, sql string, args []any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, dbCall{Kind: kind, SQL: sql, Args: args})
}

// Calls kaydedilen tüm çağrıları döndürür.
func (f *fakeDB) Calls() []dbCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dbCall, len(f.calls))
	copy(out, f.calls)
	return out
}

// ExecCalls yalnızca Exec çağrılarını döndürür.
func (f *fakeDB) ExecCalls() []dbCall {
	var out []dbCall
	for _, c := range f.Calls() {
		if c.Kind == "exec" {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeDB) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	f.record("queryrow", sql, args)
	if f.rowFn == nil {
		return fakeRow{err: pgx.ErrNoRows}
	}
	return fakeRow{scan: func(dest ...any) error { return f.rowFn(sql, args, dest...) }}
}

func (f *fakeDB) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.record("query", sql, args)
	if f.queryFn == nil {
		return &fakeRows{}, nil
	}
	rows, err := f.queryFn(sql, args)
	if err != nil {
		return &fakeRows{err: err}, err
	}
	return &fakeRows{rows: rows}, nil
}

func (f *fakeDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.record("exec", sql, args)
	if f.execFn != nil {
		if err := f.execFn(sql, args); err != nil {
			return pgconn.CommandTag{}, err
		}
	}
	return pgconn.CommandTag{}, nil
}

// fakeRow tek satırlık sorgu sonucunu temsil eder.
type fakeRow struct {
	scan func(dest ...any) error
	err  error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.scan != nil {
		return r.scan(dest...)
	}
	return r.err
}

// fakeRows çok satırlı sonuçları temsil eder.
type fakeRows struct {
	rows [][]any
	i    int
	err  error
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return r.err }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) TypeMap() *pgtype.Map                         { return nil }

func (r *fakeRows) Next() bool {
	if r.err != nil {
		return false
	}
	r.i++
	return r.i <= len(r.rows)
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.i == 0 || r.i > len(r.rows) {
		return pgx.ErrNoRows
	}
	cur := r.rows[r.i-1]
	for i, d := range dest {
		if d == nil {
			continue
		}
		var v any
		if i < len(cur) {
			v = cur[i]
		}
		if err := assignValue(d, v); err != nil {
			return err
		}
	}
	return nil
}

func (r *fakeRows) Values() ([]any, error) {
	if r.i == 0 || r.i > len(r.rows) {
		return nil, pgx.ErrNoRows
	}
	return r.rows[r.i-1], nil
}

// assignValue bir any değerini Scan hedefine (pointer) yazar.
func assignValue(dest, val any) error {
	dv := reflect.ValueOf(dest)
	if dv.Kind() != reflect.Pointer || dv.IsNil() {
		return fmt.Errorf("scan hedefi pointer degil: %T", dest)
	}
	elem := dv.Elem()
	if val == nil {
		elem.Set(reflect.Zero(elem.Type()))
		return nil
	}
	vv := reflect.ValueOf(val)
	if !vv.Type().AssignableTo(elem.Type()) {
		if !vv.Type().ConvertibleTo(elem.Type()) {
			return fmt.Errorf("tip uyusmadi: %T -> %s", val, elem.Type())
		}
		vv = vv.Convert(elem.Type())
	}
	elem.Set(vv)
	return nil
}

// bytesRow belirli bir payload döndüren QueryRow yanıtı kurar.
func bytesRow(payload string) func(string, []any, ...any) error {
	return func(_ string, _ []any, dest ...any) error {
		return assignValue(dest[0], []byte(payload))
	}
}

// ---- önbellek (services_cache) ------------------------------------

func TestCacheGet(t *testing.T) {
	cases := []struct {
		name    string
		rowFn   func(string, []any, ...any) error
		want    string
		wantErr bool
	}{
		{"veri doner", bytesRow(`{"rows":[{"rank":1}]}`), `{"rows":[{"rank":1}]}`, false},
		{"bos payload", bytesRow(``), ``, false},
		{"satir yok -> hata", func(string, []any, ...any) error { return pgx.ErrNoRows }, "", true},
		{"baglanti hatasi", func(string, []any, ...any) error { return assert.AnError }, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := &fakeDB{rowFn: c.rowFn}
			got, err := CacheGet(context.Background(), db, "puan", "L:superlig")
			if c.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, string(got))
			calls := db.Calls()
			require.Len(t, calls, 1)
			assert.Equal(t, "queryrow", calls[0].Kind)
			assert.Contains(t, calls[0].SQL, "FROM services_cache")
			assert.Equal(t, []any{"puan", "L:superlig"}, calls[0].Args)
		})
	}
}

func TestCacheSet(t *testing.T) {
	cases := []struct {
		name    string
		execFn  func(string, []any) error
		wantErr bool
	}{
		{"basarili", nil, false},
		{"yazma hatasi yutulmaz", func(string, []any) error { return assert.AnError }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := &fakeDB{execFn: c.execFn}
			err := CacheSet(context.Background(), db, "puan", "L:superlig", []byte(`{}`))
			if c.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			execs := db.ExecCalls()
			require.Len(t, execs, 1)
			assert.Contains(t, execs[0].SQL, "INSERT INTO services_cache")
			assert.Contains(t, execs[0].SQL, "ON CONFLICT", "önbellek yazımı idempotent olmalı")
			assert.Equal(t, []any{"puan", "L:superlig"}, execs[0].Args[:2])
		})
	}
}

func TestCacheList(t *testing.T) {
	rows := [][]any{
		{"hava", "6", []byte(`{"sicaklik":21}`), nil},
		{"hava", "34", []byte(`{"sicaklik":25}`), nil},
	}
	db := &fakeDB{queryFn: func(string, []any) ([][]any, error) { return rows, nil }}
	out, err := CacheList(context.Background(), db, "hava")
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, "6", out[0].Key)
	assert.Equal(t, "34", out[1].Key)
	assert.JSONEq(t, `{"sicaklik":21}`, string(out[0].Data))
}

func TestCacheListError(t *testing.T) {
	db := &fakeDB{queryFn: func(string, []any) ([][]any, error) { return nil, assert.AnError }}
	out, err := CacheList(context.Background(), db, "hava")
	assert.Error(t, err)
	assert.Nil(t, out)
}

func TestCacheListEmpty(t *testing.T) {
	db := &fakeDB{}
	out, err := CacheList(context.Background(), db, "yok-boyle-bir-kind")
	require.NoError(t, err)
	assert.Empty(t, out, "kayit yoksa bos liste, hata degil")
}

// ---- ayarlar ------------------------------------------------------

func TestGetSetting(t *testing.T) {
	cases := []struct {
		name    string
		rowFn   func(string, []any, ...any) error
		want    string
		wantErr error
	}{
		// JSON string olarak saklanan metin sarmalayıcısız döner (bkz. settingText).
		{"deger var", bytesRow(`"123"`), `123`, nil},
		{"ham json sayi", bytesRow(`7`), `7`, nil},
		{"ham json nesne", bytesRow(`{"a":1}`), `{"a":1}`, nil},
		{"yok -> ErrNotFound", func(string, []any, ...any) error { return pgx.ErrNoRows }, "", ErrNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := &fakeDB{rowFn: c.rowFn}
			got, err := GetSetting(context.Background(), db, "servis", "fikstur_key")
			if c.wantErr != nil {
				assert.ErrorIs(t, err, c.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, string(got))
		})
	}
}

func TestSetSetting(t *testing.T) {
	var got []any
	db := &fakeDB{execFn: func(_ string, args []any) error { got = args; return nil }}
	require.NoError(t, SetSetting(context.Background(), db, "servis", "fikstur_key", []byte(`"abc"`)))
	require.NotNil(t, got)
	assert.Equal(t, "servis", got[0])
	assert.Equal(t, "fikstur_key", got[1])
}

func TestSetSettingError(t *testing.T) {
	db := &fakeDB{execFn: func(string, []any) error { return assert.AnError }}
	assert.Error(t, SetSetting(context.Background(), db, "servis", "k", []byte(`1`)))
}

// ---- sayfa düzeni (page_layouts) ---------------------------------

func TestGetLayout(t *testing.T) {
	cases := []struct {
		name  string
		rowFn func(string, []any, ...any) error
		want  string
	}{
		{"duzen kayitli", bytesRow(`[{"type":"slider"}]`), `[{"type":"slider"}]`},
		{"kayit yok -> bos dizi (hata degil)", func(string, []any, ...any) error { return pgx.ErrNoRows }, `[]`},
		{"veritabani hatasi -> bos dizi", func(string, []any, ...any) error { return assert.AnError }, `[]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := &fakeDB{rowFn: c.rowFn}
			got, err := GetLayout(context.Background(), db, 1, "home")
			require.NoError(t, err, "duzen okunamasa bile varsayılan kullanılmalı")
			assert.JSONEq(t, c.want, string(got))
			calls := db.Calls()
			require.Len(t, calls, 1)
			assert.Equal(t, []any{int64(1), "home"}, calls[0].Args)
		})
	}
}

func TestSaveLayout(t *testing.T) {
	cases := []struct {
		name    string
		execFn  func(string, []any) error
		wantErr bool
	}{
		{"basarili", nil, false},
		{"yazma hatasi yutulmaz", func(string, []any) error { return assert.AnError }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := &fakeDB{execFn: c.execFn}
			err := SaveLayout(context.Background(), db, 1, "home", []byte(`[]`))
			if c.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			execs := db.ExecCalls()
			require.Len(t, execs, 1)
			assert.Contains(t, execs[0].SQL, "ON CONFLICT (site_id, page_key)")
		})
	}
}

func TestDeleteLayout(t *testing.T) {
	db := &fakeDB{}
	require.NoError(t, DeleteLayout(context.Background(), db, 1, "kategori"))
	execs := db.ExecCalls()
	require.Len(t, execs, 1)
	assert.Contains(t, execs[0].SQL, "DELETE FROM page_layouts")
	assert.Equal(t, []any{int64(1), "kategori"}, execs[0].Args)
}
