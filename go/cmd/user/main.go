// theblock — kullanıcı CLI.
// Kullanım:
//
//	go run ./cmd/user create --email E --name AD --role ROL [--password P]
//	go run ./cmd/user reset-password --email E --password P [--role ROL]
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/auth"
	"theblock/site/internal/config"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

type args struct {
	email, name, role, password string
}

func main() {
	logger := platform.NewLogger(slog.LevelInfo)
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		logger.Error("DATABASE_URL boş")
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("bağlanılamadı", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	ctx := context.Background()

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	a, err := parseArgs(os.Args[2:])
	if err != nil {
		logger.Error("parametre hatası", "error", err)
		usage()
		os.Exit(2)
	}

	if a.email == "" {
		logger.Error("--email gerekli")
		usage()
		os.Exit(2)
	}
	a.email = strings.ToLower(a.email)

	switch cmd {
	case "create":
		if !validRole(a.role) {
			a.role = "Editör"
		}
		if a.password == "" {
			a.password = randomPass(16)
			fmt.Printf("üretilen şifre: %s\n", a.password)
		}
		hash, err := auth.HashPassword(a.password)
		if err != nil {
			logger.Error("hash hatası", "error", err)
			os.Exit(1)
		}
		name := a.name
		if name == "" {
			name = strings.SplitN(a.email, "@", 2)[0]
		}
		id, err := repo.CreateUser(ctx, pool, name, a.email, a.role, hash)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") {
				logger.Error("kullanıcı zaten var", "email", a.email)
				os.Exit(2)
			}
			logger.Error("kullanıcı oluşturulamadı", "error", err)
			os.Exit(1)
		}
		fmt.Printf("kullanıcı oluşturuldu: id=%d email=%s rol=%s\n", id, a.email, a.role)

	case "reset-password":
		if a.password == "" {
			logger.Error("--password gerekli")
			os.Exit(2)
		}
		u, err := repo.GetUserByEmail(ctx, pool, a.email)
		if err != nil {
			logger.Error("kullanıcı bulunamadı", "email", a.email)
			os.Exit(1)
		}
		hash, err := auth.HashPassword(a.password)
		if err != nil {
			logger.Error("hash hatası", "error", err)
			os.Exit(1)
		}
		if err := repo.UpdateUserPassword(ctx, pool, u.ID, hash); err != nil {
			logger.Error("şifre güncellenemedi", "error", err)
			os.Exit(1)
		}
		if a.role != "" && validRole(a.role) {
			if err := repo.UpdateUserRole(ctx, pool, u.ID, a.role); err != nil {
				logger.Error("rol güncellenemedi", "error", err)
				os.Exit(1)
			}
			fmt.Printf("rol güncellendi: %s\n", a.role)
		}
		fmt.Printf("şifre güncellendi: %s\n", u.Email)

	default:
		logger.Error("bilinmeyen komut", "komut", cmd)
		usage()
		os.Exit(2)
	}
}

func parseArgs(in []string) (args, error) {
	var a args
	i := 0
	next := func() (string, error) {
		if i >= len(in) {
			return "", fmt.Errorf("değer eksik")
		}
		v := in[i]
		i++
		return v, nil
	}
	for i < len(in) {
		flag := in[i]
		i++
		var v string
		var err error
		switch flag {
		case "--email":
			if v, err = next(); err == nil {
				a.email = v
			}
		case "--name":
			if v, err = next(); err == nil {
				a.name = v
			}
		case "--role":
			if v, err = next(); err == nil {
				a.role = v
			}
		case "--password":
			if v, err = next(); err == nil {
				a.password = v
			}
		default:
			return a, fmt.Errorf("bilinmeyen parametre: %s", flag)
		}
		if err != nil {
			return a, err
		}
	}
	return a, nil
}

func validRole(r string) bool {
	switch r {
	case "Yönetici", "Editör", "Muhabir", "Yazar", "Üye":
		return true
	}
	return false
}

func usage() {
	fmt.Fprintln(os.Stderr, `kullanım:
  go run ./cmd/user create --email E --name AD --role ROL [--password P]
  go run ./cmd/user reset-password --email E --password P [--role ROL]
rol: Yönetici | Editör | Muhabir | Yazar | Üye`)
}

func randomPass(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, n)
	rb := make([]byte, n)
	if _, err := rand.Read(rb); err != nil {
		for i := range b {
			b[i] = chars[(i*7+3)%len(chars)]
		}
		return string(b)
	}
	for i := range b {
		b[i] = chars[int(rb[i])%len(chars)]
	}
	return string(b)
}
