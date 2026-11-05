// Command dbexec — verilen .sql dosyasını veritabanına uygular.
//
// Kullanım:
//
//	DATABASE_URL=postgres://... go run ./cmd/dbexec deploy/e2e_seed.sql
//
// CI'da test verisi yüklemek ve migration sonrası idempotent kontroller
// yapmak için kullanılır. Dış ağa çıkmaz; yalnızca verilen SQL'i çalıştırır.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL boş")
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "kullanım: go run ./cmd/dbexec <dosya.sql>")
		os.Exit(1)
	}

	for _, path := range os.Args[1:] {
		// #nosec G703 -- Bu bir YÖNETİCİ komut satırı aracıdır; dosya yolu
		// çağıran operatörün kendi argümanıdır. Ağdan/HTTP'den girdi yok.
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dosya okunamadı: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			cancel()
			fmt.Fprintf(os.Stderr, "veritabanına bağlanılamadı: %v\n", err)
			os.Exit(1)
		}

		if _, err := conn.Exec(ctx, string(b)); err != nil {
			conn.Close(ctx)
			cancel()
			fmt.Fprintf(os.Stderr, "SQL hatası (%s): %v\n", path, err)
			os.Exit(1)
		}
		conn.Close(ctx)
		cancel()
		fmt.Println("uygulandi:", path)
	}
}
