// Package config uygulama yapılandırmasını ortam değişkenlerinden okur.
// Sırlar yalnızca .env/sır deposunda bulunur; repoya asla düşmez.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env         string
	Debug       bool
	ListenAddr  string // ör. ":8070"
	BaseURL     string
	Port        string
	DatabaseURL string
	RedisURL    string
	// Sesli haber (OpenAI TTS) — anahtar yoksa özellik kapalı kalır.
	OpenAIKey  string
	OpenAIBase string
	TTSModel   string
	TTSVoice   string
	// Röportaj transkripsiyonu (AssemblyAI / Deepgram) — anahtar yoksa özellik kapalı.
	AssemblyAIKey   string
	AssemblyAIBase  string
	AssemblyAIModel string
	DeepgramKey     string
	DeepgramBase    string
	DeepgramModel   string
	// TranscribeProvider hangi ASR sağlayıcısının kullanılacağı.
	TranscribeProvider string
	// Private ses depolama ve ffmpeg araçları.
	PrivateStorageDir string
	FFmpegBin         string
	FFprobeBin        string
	// Sözlük tabanlı yazım denetimi (isteğe bağlı). Boşsa yalnız kural
	// tabanlı denetim çalışır.
	HunspellBin  string
	HunspellDict string

	// CryptoQuant zincir-üstü veri anahtarı. Boşsa zincir-üstü modüller
	// (MVRV, SOPR, borsa akışı) hiç basılmaz — uydurma veri üretilmez.
	CryptoQuantKey string

	DBMaxConns        int
	DBMinConns        int
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration

	// Kapanışta uçuştaki istekler için tanınan süre (SIGTERM → Shutdown).
	ShutdownTimeout time.Duration

	PprofAddr string

	GzipLevel int

	SayfaOnbellekTTL   time.Duration
	SayfaOnbellekBayat time.Duration
}

func Load() Config {
	cfg := Config{
		Env:         getEnv("APP_ENV", "production"),
		Debug:       strings.EqualFold(getEnv("APP_DEBUG", "false"), "true"),
		ListenAddr:  ":" + getEnv("PORT", "8070"),
		BaseURL:     getEnv("APP_BASE_URL", "https://bitcosar.com"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", ""),
		OpenAIKey:   getEnv("OPENAI_API_KEY", ""),
		OpenAIBase:  getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		TTSModel:    getEnv("TTS_MODEL", "tts-1"),
		TTSVoice:    getEnv("TTS_VOICE", "alloy"),

		AssemblyAIKey:      getEnv("ASSEMBLYAI_API_KEY", ""),
		AssemblyAIBase:     getEnv("ASSEMBLYAI_BASE_URL", "https://api.assemblyai.com"),
		AssemblyAIModel:    getEnv("ASSEMBLYAI_MODEL", "best"),
		DeepgramKey:        getEnv("DEEPGRAM_API_KEY", ""),
		DeepgramBase:       getEnv("DEEPGRAM_BASE_URL", "https://api.deepgram.com"),
		DeepgramModel:      getEnv("DEEPGRAM_MODEL", "nova-3"),
		TranscribeProvider: getEnv("TRANSCRIBE_PROVIDER", "assemblyai"),
		PrivateStorageDir:  getEnv("PRIVATE_STORAGE_DIR", ""),
		FFmpegBin:          getEnv("FFMPEG_BIN", "ffmpeg"),
		FFprobeBin:         getEnv("FFPROBE_BIN", "ffprobe"),
		HunspellBin:        getEnv("HUNSPELL_BIN", ""),
		HunspellDict:       getEnv("HUNSPELL_DICT", "tr_TR"),
		CryptoQuantKey:     getEnv("CRYPTOQUANT_KEY", ""),

		DBMaxConns:         getInt("DB_MAX_CONNS", 0),
		DBMinConns:         getInt("DB_MIN_CONNS", 0),
		DBMaxConnLifetime:  getDuration("DB_MAX_CONN_LIFETIME", 0),
		DBMaxConnIdleTime:  getDuration("DB_MAX_CONN_IDLE_TIME", 0),
		ShutdownTimeout:    getDuration("SHUTDOWN_TIMEOUT", 25*time.Second),
		PprofAddr:          getEnv("PPROF_ADDR", ""),
		GzipLevel:          getInt("GZIP_LEVEL", 0),
		SayfaOnbellekTTL:   getDuration("SAYFA_ONBELLEK_TTL", 0),
		SayfaOnbellekBayat: getDuration("SAYFA_ONBELLEK_BAYAT", 0),
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getInt ortam değişkenini int olarak okur. Bozuk/eksikse varsayılana döner:
// yapılandırma hatası yüzünden süreç açılmasın (varsayılan güvenli tarafta).
func getInt(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return def
	}
	return n
}

// getDuration ortam değişkenini süre olarak okur ("25s", "2m", "1h").
// Bozuk/eksikse varsayılana döner.
func getDuration(key string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return def
	}
	return d
}
