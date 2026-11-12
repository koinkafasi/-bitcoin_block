// Package tts, OpenAI uyumlu /audio/speech uç noktasıyla metni MP3'e çevirir
// ve uzun metinleri güvenli parçalara böler.
package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Client OpenAI TTS istemcisi.
type Client struct {
	Key   string
	Base  string
	Model string
	Voice string
	HTTP  *http.Client
}

// New istemci kurar; anahtar boşsa Enabled false döner.
func New(key, base, model, voice string) *Client {
	if strings.TrimSpace(base) == "" {
		base = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(model) == "" {
		model = "tts-1"
	}
	if strings.TrimSpace(voice) == "" {
		voice = "alloy"
	}
	return &Client{
		Key:   strings.TrimSpace(key),
		Base:  strings.TrimRight(strings.TrimSpace(base), "/"),
		Model: model,
		Voice: voice,
		HTTP:  &http.Client{Timeout: 120 * time.Second},
	}
}

// Enabled TTS kullanılabilir mi (API anahtarı var mı).
func (c *Client) Enabled() bool { return c != nil && c.Key != "" }

// Speech tek bir metin parçasını MP3 olarak döndürür.
func (c *Client) Speech(ctx context.Context, text string) ([]byte, error) {
	if !c.Enabled() {
		return nil, errors.New("TTS anahtarı tanımlı değil (OPENAI_API_KEY)")
	}
	body, err := json.Marshal(map[string]any{
		"model":           c.Model,
		"input":           text,
		"voice":           c.Voice,
		"response_format": "mp3",
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("TTS hatası (%d): %s", resp.StatusCode, strings.TrimSpace(string(data[:min(len(data), 300)])))
	}
	if len(data) < 128 || !looksLikeMP3(data) {
		return nil, errors.New("TTS yanıtı ses verisi değil")
	}
	return data, nil
}

func looksLikeMP3(b []byte) bool {
	if len(b) >= 3 && string(b[:3]) == "ID3" {
		return true
	}
	return len(b) >= 2 && b[0] == 0xFF && (b[1]&0xE0) == 0xE0
}

func SplitText(text string, maxRunes int) []string {
	text = strings.Join(strings.Fields(text), " ")
	if maxRunes <= 0 {
		maxRunes = 3500
	}
	r := []rune(text)
	if len(r) <= maxRunes {
		if len(r) == 0 {
			return nil
		}
		return []string{text}
	}
	// Cümlelere böl (nokta, ünlem, soru, üç nokta + boşluk).
	re := regexp.MustCompile(`(?s)[^.!?…]+[.!?…]+[\s]*|[^.!?…]+$`)
	parts := re.FindAllString(text, -1)
	var out []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		pr := []rune(p)
		if len(pr) > maxRunes {
			flush()
			for len(pr) > maxRunes {
				out = append(out, strings.TrimSpace(string(pr[:maxRunes])))
				pr = pr[maxRunes:]
			}
			if len(pr) > 0 {
				cur.WriteString(string(pr))
			}
			continue
		}
		if cur.Len() > 0 && len([]rune(cur.String()))+1+len(pr) > maxRunes {
			flush()
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(p)
	}
	flush()
	return out
}

// HTMLToText blok HTML'ini düz metne çevirir (script/style atılır).
func HTMLToText(h string) string {
	// RE2 geri referans desteklemez: her etiket için ayrı desen.
	for _, tag := range []string{"script", "style", "noscript"} {
		re := regexp.MustCompile(`(?is)<` + tag + `[^>]*>.*?</` + tag + `>`)
		h = re.ReplaceAllString(h, " ")
	}
	reBlock := regexp.MustCompile(`(?i)</?(p|div|br|li|h[1-6]|blockquote|tr|td|section|article)[^>]*>`)
	h = reBlock.ReplaceAllString(h, "\n")
	reTag := regexp.MustCompile(`(?s)<[^>]*>`)
	h = reTag.ReplaceAllString(h, " ")
	h = html.UnescapeString(h)
	// Etiket boşlukları noktalama öncesinde kalmasın ("paragraf ." → "paragraf.").
	h = regexp.MustCompile(`\s+([.,!?;:…%])`).ReplaceAllString(h, "$1")
	lines := strings.Split(h, "\n")
	var b strings.Builder
	for _, ln := range lines {
		ln = strings.Join(strings.FieldsFunc(ln, unicode.IsSpace), " ")
		if ln == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(ln)
	}
	return strings.TrimSpace(b.String())
}

// EstimateSeconds Türkçe okuma hızıyla (~14 karakter/sn) süre tahmini.
func EstimateSeconds(text string) float64 {
	n := len([]rune(text))
	if n <= 0 {
		return 0
	}
	s := float64(n) / 14.0
	if s < 1 {
		s = 1
	}
	return float64(int(s*10)) / 10
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
