// Command dbq — salt-okunur SQL sorgusu çalıştırıp sonucu tablo olarak basar.
//
// Kullanım:
//
//	DATABASE_URL=... go run ./cmd/dbq "SELECT id, title FROM contents LIMIT 5"
//	go run ./cmd/dbq -f sorgu.sql
//
// Yalnız SELECT/WITH kabul eder; yazma işlemleri reddedilir (yanlışlıkla veri
// bozmayı önlemek için). Operasyon/debug için pratik bir yardımcıdır.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	file := flag.String("f", "", "SQL dosyası")
	flag.Parse()

	sql := strings.TrimSpace(flag.Arg(0))
	if *file != "" {
		b, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "dosya okunamadı:", err)
			os.Exit(1)
		}
		sql = strings.TrimSpace(string(b))
	}
	if sql == "" {
		fmt.Fprintln(os.Stderr, "kullanım: go run ./cmd/dbq \"SELECT ...\" | -f dosya.sql")
		os.Exit(1)
	}
	low := strings.ToLower(sql)
	if !strings.HasPrefix(low, "select") && !strings.HasPrefix(low, "with") {
		fmt.Fprintln(os.Stderr, "yalnız SELECT/WITH sorguları desteklenir (salt-okunur)")
		os.Exit(1)
	}

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL boş")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bağlantı:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close(ctx) }()

	rows, err := conn.Query(ctx, sql)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sorgu:", err)
		os.Exit(1)
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	fmt.Println(strings.Join(names, " | "))
	fmt.Println(strings.Repeat("-", 60))
	n := 0
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			fmt.Fprintln(os.Stderr, "satır:", err)
			continue
		}
		cells := make([]string, len(vals))
		for i, v := range vals {
			cells[i] = fmtValue(v)
		}
		fmt.Println(strings.Join(cells, " | "))
		n++
	}
	if err := rows.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "okuma:", err)
	}
	fmt.Printf("(%d satır)\n", n)
}

func fmtValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch x := v.(type) {
	case time.Time:
		return x.Local().Format("2006-01-02 15:04:05")
	case []byte:
		return truncate(string(x), 200)
	default:
		return truncate(fmt.Sprint(x), 200)
	}
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
