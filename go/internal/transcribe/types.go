package transcribe

import (
	"context"
	"errors"
	"io"
	"strings"
)

// ErrDisabled sağlayıcı anahtarı yoksa döner.
var ErrDisabled = errors.New("transkripsiyon sağlayıcısı yapılandırılmadı")

type Word struct {
	StartMS    int64   `json:"start_ms"`
	EndMS      int64   `json:"end_ms"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence,omitempty"`
}

// Segment zaman damgalı tek konuşma bölümüdür.
type Segment struct {
	StartMS    int64   `json:"start_ms"`
	EndMS      int64   `json:"end_ms"`
	Speaker    string  `json:"speaker,omitempty"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence,omitempty"`
	// Words segmentin kelime kelime zaman damgalarıdır. Sağlayıcı kelime
	// döndürmezse boş kalır; arayüz segment seviyesine düşer.
	Words []Word `json:"words,omitempty"`
}

// Request sağlayıcıya verilen normalize edilmiş ses isteğidir.
type Request struct {
	Audio            io.Reader
	Filename         string
	Language         string
	SpeakersExpected int
	Model            string
}

// Result sağlayıcıdan dönen transkripttir.
type Result struct {
	Text          string
	Language      string
	DurationMS    int64
	Segments      []Segment
	Speakers      []string
	Provider      string
	Model         string
	ProviderJobID string
}

type ProgressFunc func(percent int, providerJobID string)

// Provider tek bir ASR sağlayıcısıdır.
type Provider interface {
	Name() string
	Transcribe(ctx context.Context, req Request, progress ProgressFunc) (Result, error)
}

// FromConfig anahtar varsa seçilen sağlayıcıyı kurar; yoksa nil döner.
// provider boşsa "assemblyai" varsayılır (mevcut davranış korunur).
func FromConfig(provider, apiKey, baseURL, model string) Provider {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	if ProviderName(provider) == "deepgram" {
		return NewDeepgram(apiKey, baseURL, model)
	}
	return NewAssemblyAI(apiKey, baseURL, model)
}

// ProviderName verilen adı doğrulanmış sağlayıcı adına çevirir.
func ProviderName(name string) string {
	if strings.EqualFold(strings.TrimSpace(name), "deepgram") {
		return "deepgram"
	}
	return "assemblyai"
}
