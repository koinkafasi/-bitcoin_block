package config

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (c Config) PoolConfig() (*pgxpool.Config, error) {
	pc, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL ayrıştırılamadı: %w", err)
	}
	if c.DBMaxConns > 0 {
		// #nosec G115 -- havuz boyutu yapılandırma değeri; int32 aralığını
		// aşan bir değer (2 milyar bağlantı) zaten anlamsız ve pgx reddeder.
		pc.MaxConns = int32(c.DBMaxConns)
	}
	// MinConns, MaxConns'ten büyük olamaz: pgx bunu reddeder. Kırpıyoruz.
	if c.DBMinConns > 0 {
		// #nosec G115 -- bkz. yukarıdaki MaxConns gerekçesi.
		pc.MinConns = int32(c.DBMinConns)
		if pc.MaxConns > 0 && pc.MinConns > pc.MaxConns {
			pc.MinConns = pc.MaxConns
		}
	}
	if c.DBMaxConnLifetime > 0 {
		pc.MaxConnLifetime = c.DBMaxConnLifetime
	}
	if c.DBMaxConnIdleTime > 0 {
		pc.MaxConnIdleTime = c.DBMaxConnIdleTime
	}
	return pc, nil
}

func (c Config) ShutdownWindow() time.Duration {
	if c.ShutdownTimeout > 0 {
		return c.ShutdownTimeout
	}
	return 25 * time.Second
}
