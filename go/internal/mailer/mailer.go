// Package mailer — scheduler gibi httpapi dışı paketlerin SMTP ile
// e-posta göndermesi için ince yardımcı (ayarlar section='mail').
package mailer

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// Config ayarlardan SMTP yapılandırmasını okur (httpapi.smtpConfig ile aynı).
func Config(ctx context.Context, pool *pgxpool.Pool) platform.SMTPConfig {
	gs, _ := repo.GetSettings(ctx, pool, "mail")
	return ParseConfig(gs)
}

func ParseConfig(gs map[string][]byte) platform.SMTPConfig {
	str := func(k string) string {
		b, ok := gs[k]
		if !ok {
			return ""
		}
		var v string
		if json.Unmarshal(b, &v) == nil {
			return v
		}
		return strings.Trim(string(b), `"`)
	}
	port := 587
	if p := str("smtp_port"); p != "" {
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				n = 0
				break
			}
			n = n*10 + int(r-'0')
		}
		if n > 0 && n <= 65535 {
			port = n
		}
	}
	return platform.SMTPConfig{
		Host:        str("smtp_host"),
		Port:        port,
		User:        str("smtp_user"),
		Pass:        str("smtp_pass"),
		From:        str("smtp_from"),
		FromName:    str("smtp_from_name"),
		ImplicitTLS: str("smtp_tls") == "1",
	}
}

// Send kullanıcı e-postası gönderir (SMTP yoksa sessizce döner).
func Send(ctx context.Context, pool *pgxpool.Pool, to, subject, html string) error {
	return SendConfig(Config(ctx, pool), to, subject, html)
}

func SendConfig(cfg platform.SMTPConfig, to, subject, html string) error {
	if !cfg.Configured() || strings.TrimSpace(to) == "" {
		return nil
	}
	return cfg.SendMail(to, subject, html)
}
