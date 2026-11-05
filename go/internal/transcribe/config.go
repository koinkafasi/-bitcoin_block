package transcribe

import (
	"context"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

// Config çalışma zamanı ASR yapılandırmasıdır. Admin panelinden
// değiştirilebilir; .env yalnız varsayılan/başlangıç değerini sağlar.
type Config struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

// Empty anahtar yok mu?
func (c Config) Empty() bool { return strings.TrimSpace(c.APIKey) == "" }

// Normalized sağlayıcı adını doğrular ve boş alanları varsayılanla doldurur.
func (c Config) Normalized() Config {
	c.Provider = ProviderName(c.Provider)
	c.APIKey = strings.TrimSpace(c.APIKey)
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	c.Model = strings.TrimSpace(c.Model)
	return c
}

// providerFor yapılandırmadan sağlayıcı kurar (anahtar yoksa nil).
func (c Config) providerFor() Provider {
	c = c.Normalized()
	if c.Empty() {
		return nil
	}
	return FromConfig(c.Provider, c.APIKey, c.BaseURL, c.Model)
}

// DefaultsFromValues .env/config değerlerinden başlangıç yapılandırmasını
// kurar (panel değerleri bunu geçersiz kılar).
func DefaultsFromValues(provider, asrKey, asrBase, asrModel, dgKey, dgBase, dgModel string) Config {
	if ProviderName(provider) == "deepgram" {
		return Config{
			Provider: "deepgram",
			APIKey:   dgKey,
			BaseURL:  dgBase,
			Model:    dgModel,
		}.Normalized()
	}
	return Config{
		Provider: "assemblyai",
		APIKey:   asrKey,
		BaseURL:  asrBase,
		Model:    asrModel,
	}.Normalized()
}

// ASR ayarlarının settings tablosundaki yeri.
const (
	settingsSection = "asr"
	keyProvider     = "provider"
	keyAPIKey       = "api_key"
	keyBaseURL      = "base_url"
	keyModel        = "model"
)

// SettingsStore ASR yapılandırmasını settings tablosunda tutar. .env
// değerleri varsayılan olarak kullanılır; panelden kaydedilen değerler
// önceliklidir. Böylece API anahtarı/sağlayıcı değişince yeniden başlatma
// gerekmez.
type SettingsStore struct {
	pool     *pgxpool.Pool
	defaults Config
}

// NewSettingsStore kurar. defaults .env'den gelen başlangıç değerleridir.
func NewSettingsStore(pool *pgxpool.Pool, defaults Config) *SettingsStore {
	return &SettingsStore{pool: pool, defaults: defaults.Normalized()}
}

// Load etkin yapılandırmayı döndürür: panel değeri varsa o, yoksa .env.
func (s *SettingsStore) Load(ctx context.Context) Config {
	cfg := s.defaults
	if s.pool == nil {
		return cfg
	}
	if v, err := repo.GetSetting(ctx, s.pool, settingsSection, keyProvider); err == nil && len(v) > 0 {
		cfg.Provider = string(v)
	}
	if v, err := repo.GetSetting(ctx, s.pool, settingsSection, keyAPIKey); err == nil && len(v) > 0 {
		cfg.APIKey = string(v)
	}
	if v, err := repo.GetSetting(ctx, s.pool, settingsSection, keyBaseURL); err == nil && len(v) > 0 {
		cfg.BaseURL = string(v)
	}
	if v, err := repo.GetSetting(ctx, s.pool, settingsSection, keyModel); err == nil && len(v) > 0 {
		cfg.Model = string(v)
	}
	return cfg.Normalized()
}

// Save panelden gelen değerleri yazar. Boş alan "varsayılana dön" demektir;
// API anahtarı boş bırakılırsa .env değeri kullanılmaya devam eder.
func (s *SettingsStore) Save(ctx context.Context, cfg Config) error {
	if s.pool == nil {
		return nil
	}
	cfg = cfg.Normalized()
	set := func(key, val string) error {
		return repo.SetSetting(ctx, s.pool, settingsSection, key, []byte(val))
	}
	if err := set(keyProvider, cfg.Provider); err != nil {
		return err
	}
	if err := set(keyBaseURL, cfg.BaseURL); err != nil {
		return err
	}
	if err := set(keyModel, cfg.Model); err != nil {
		return err
	}
	// Anahtar boş gönderilirse mevcut panel değerini SİLME, .env'e düş.
	if cfg.APIKey != "" {
		return set(keyAPIKey, cfg.APIKey)
	}
	return nil
}

// ClearAPIKey panelde kayıtlı anahtarı siler (.env değeri devreye girer).
func (s *SettingsStore) ClearAPIKey(ctx context.Context) error {
	if s.pool == nil {
		return nil
	}
	return repo.SetSetting(ctx, s.pool, settingsSection, keyAPIKey, []byte(""))
}

// Defaults .env kaynaklı varsayılanları döndürür (panelde ipucu olarak).
func (s *SettingsStore) Defaults() Config { return s.defaults }

// Dynamic ayarları her işte yeniden okuyan sağlayıcıdır. Panelden anahtar
// değişince worker yeniden başlatılmadan yeni sağlayıcı kullanılır.
type Dynamic struct {
	store *SettingsStore
	mu    sync.Mutex
	last  Config
	cur   Provider
}

// NewDynamic dinamik sağlayıcıyı kurar.
func NewDynamic(store *SettingsStore) *Dynamic {
	return &Dynamic{store: store}
}

// Name sabit ad döndürür; gerçek sağlayıcı adı iş kaydına yazılır.
func (d *Dynamic) Name() string { return "dynamic" }

// Resolve güncel yapılandırmayı döndürür.
func (d *Dynamic) Resolve(ctx context.Context) Config {
	if d.store == nil {
		return Config{}
	}
	return d.store.Load(ctx)
}

// Transcribe güncel yapılandırmayla sağlayıcıyı seçip çalıştırır. Anahtar
// yoksa ErrDisabled döner (iş açık hata ile başarısız olur, sessizce
// beklemez).
func (d *Dynamic) Transcribe(ctx context.Context, req Request, progress ProgressFunc) (Result, error) {
	cfg := d.Resolve(ctx)
	if cfg.Empty() {
		return Result{}, ErrDisabled
	}
	p := d.provider(cfg)
	if p == nil {
		return Result{}, ErrDisabled
	}
	return p.Transcribe(ctx, req, progress)
}

// provider yapılandırma değişmediyse önceki örneği yeniden kullanır.
func (d *Dynamic) provider(cfg Config) Provider {
	cfg = cfg.Normalized()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cur != nil && d.last == cfg {
		return d.cur
	}
	p := cfg.providerFor()
	d.cur = p
	d.last = cfg
	return p
}
