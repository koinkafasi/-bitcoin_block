package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Role modeli konuşma rolleridir.
type Role string

const (
	RoleSystem Role = "system"
	RoleUser   Role = "user"
)

// Msg tekileti konuşmada rol+igeriktir.
type Msg struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Provider yapay zekâ uç noktası soyutlamasıdır.
type Provider interface {
	Name() string
	Chat(ctx context.Context, messages []Msg) (string, error)
}

// Client sağlayıcı listesini tutar (fallback sıralı).
type Client struct {
	logger     *slog.Logger
	providers  []Provider
	httpClient *http.Client
	// kota sağlayıcının günlük kota durumunu hatırlar: tükendiyse boşa
	// HTTP çağrısı yapılmaz (eskiden her iş için 429 alınıyordu).
	kota *KotaIzleyici
}

// NewClient env'den sağlayıcıları kurar; anahtar yoksa Mock ekler.
func NewClient(logger *slog.Logger) *Client {
	c := &Client{
		logger:     logger,
		httpClient: &http.Client{Timeout: aiHTTPTimeout()},
		kota:       NewKotaIzleyici(),
	}
	key := os.Getenv("OMNIROUTE_API_KEY")
	base := os.Getenv("OMNIROUTE_BASE_URL")
	if base == "" {
		base = "https://omniroute.example.invalid/v1"
	}
	model := os.Getenv("OMNIROUTE_MODEL")
	if model == "" {
		model = os.Getenv("OMNIROUTE_DEFAULT_MODEL")
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	if key != "" {
		c.providers = append(c.providers, &httpProvider{
			name: "omniroute", baseURL: base, apiKey: key, model: model, client: c.httpClient, logger: logger,
			reasoningEffort: strings.TrimSpace(os.Getenv("OMNIROUTE_REASONING_EFFORT")),
		})
	}
	c.providers = append(c.providers, &mockProvider{client: c})
	return c
}

func aiHTTPTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("AI_HTTP_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 10 && n <= 600 {
			return time.Duration(n) * time.Second
		}
	}
	// OLCUM (14 Kasim 2026): BASARILI islerin p90'i 194.7 sn, en uzunu
	// 239.1 sn. 120 sn tek denemede saglikli yanitlari kesiyordu.
	return 180 * time.Second
}

func (c *Client) Enabled() bool {
	if c == nil || len(c.providers) == 0 {
		return false
	}
	_, onlyMock := c.providers[0].(*mockProvider)
	return !onlyMock
}

// Providers adlı listeyi döndürür. Nil alıcı güvenli.
func (c *Client) Providers() []string {
	if c == nil {
		return nil
	}
	var out []string
	for _, p := range c.providers {
		out = append(out, p.Name())
	}
	return out
}

const chatAttemptBudget = 15 * time.Second

// Chat mesajları sırayla sağlayıcılarda dener; hepsi düşerse mock'a düşer.
func (c *Client) Chat(ctx context.Context, messages []Msg) (string, error) {
	if c == nil {
		return "", fmt.Errorf("AI istemcisi yapılandırılmamış")
	}
	// KOTA KISA DEVRESİ: kota tükenmişse hiç çağrı yapma. Eskiden her iş
	// için istek atılıyor, 429 alınıyor ve iş kaybediliyordu; ölçümde
	// 01:00-02:00 arası 60 çağrının TAMAMI böyle boşa gitti.
	simdi := time.Now()
	if c.kota != nil {
		if c.kota.Temizle(simdi) {
			c.logger.Info("ai kotası döndü (sıfırlama geçildi)")
		}
		if c.kota.Tukendi(simdi) {
			return "", fmt.Errorf("%w; sıfırlamaya kalan: %s",
				ErrKotaDoldu, c.kota.BeklemeSuresi(simdi).Round(time.Minute))
		}
	}

	var lastErr error
	var kotaErr error // kota hatası ayrı tutulur: zincirde kaybolmasın
	for _, p := range c.providers {
		if err := ctx.Err(); err != nil {
			lastErr = err
			break
		}
		attemptCtx, cancel := context.WithTimeout(ctx, chatAttemptBudget)
		out, err := p.Chat(attemptCtx, messages)
		cancel()
		if err == nil {
			c.logger.Info("ai chat", "provider", p.Name(), "tokens_est", estimateTokens(messages))
			return out, nil
		}
		lastErr = err
		if errors.Is(err, ErrKotaDoldu) {
			kotaErr = err
			if c.kota != nil {
				c.kota.Isaretle(simdi)
			}
		}
		c.logger.Warn("ai provider başarısız, sıradakine geçiliyor", "provider", p.Name(), "error", err)
	}
	// Kota hatası varsa ONU döndür: çağıran taraf işi kuyrukta bırakmalı,
	// başarısız saymamalı. Başka bir hatayla sarmalamak bu ayrımı yok eder.
	if kotaErr != nil {
		return "", kotaErr
	}
	return "", fmt.Errorf("tüm AI sağlayıcıları başarısız: %w", lastErr)
}

// ---- HTTP provider (OpenAI-uyumlu) ----

type httpProvider struct {
	name            string
	baseURL         string
	apiKey          string
	model           string
	reasoningEffort string
	client          *http.Client
	logger          *slog.Logger
}

func (p *httpProvider) Name() string { return p.name }

type chatCompletionReq struct {
	Model    string   `json:"model"`
	Messages []apiMsg `json:"messages"`
	Stream   bool     `json:"stream"`
}

type apiMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResp struct {
	Choices []struct {
		// FinishReason "length" ise çıktı token sınırında kesilmiştir.
		// Eskiden okunmuyordu: kesik JSON ayrıştırıcıya gidiyor ve
		// hata sebebi yanlış yeri gösteriyordu.
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content string `json:"content"`
			// Reasoning: düşünen modeller çıktıyı buraya yazıp
			// content'i boş bırakıyor. Bkz. icerik.go.
			Reasoning string `json:"reasoning"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (p *httpProvider) Chat(ctx context.Context, messages []Msg) (string, error) {
	body, err := json.Marshal(chatCompletionReq{
		Model:    p.model,
		Stream:   false,
		Messages: toAPIMsgs(messages),
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		// KOTA AYRI BİR HATA: 429/402 geldiğinde iş BAŞARISIZ değildir —
		// içerik bir sonraki sıfırlamada üretilebilir. Eskiden hepsi tek
		// tip hataydı ve çağıran taraf işi `failed` işaretleyip atıyordu;
		// ölçümde 24 saatte 118 iş (toplamın ~%22'si) böyle kayboldu.
		if kerr := kotaHatasi(resp.StatusCode, clip(raw, 200)); kerr != nil {
			return "", kerr
		}
		return "", fmt.Errorf("AI uç noktası %d: %s", resp.StatusCode, clip(raw, 300))
	}
	var cr chatCompletionResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		return "", err
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("AI boş yanıt")
	}
	ch := cr.Choices[0]
	// kesikHataSay=false: burası etkileşimli düz metin yolu (sohbet,
	// geri bildirim yanıtı, admin aracı). Kesik yanıt kullanıcıya yarım
	// cümle gösterir ama hiç yanıt göstermemekten iyidir; yine de sessiz
	// kalmasın diye uyarıya yazılır. Boş içerik ise her yolda hatadır.
	out, err := metinCikar(ch.FinishReason, ch.Message.Content,
		ch.Message.Reasoning, cr.Usage.CompletionTokens, false)
	if err != nil {
		return "", err
	}
	if kesikMi(ch.FinishReason) && p.logger != nil {
		p.logger.Warn("ai yanıtı token sınırında kesildi",
			"provider", p.name, "model", p.model,
			"completion_tokens", cr.Usage.CompletionTokens)
	}
	return out, nil
}

func toAPIMsgs(msgs []Msg) []apiMsg {
	out := make([]apiMsg, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, apiMsg{Role: string(m.Role), Content: m.Content})
	}
	return out
}

// ---- Mock provider (anahtarsız demo; kural tabanlı) ----

type mockProvider struct {
	client *Client
}

func (p *mockProvider) Name() string { return "mock" }

func (p *mockProvider) Chat(ctx context.Context, messages []Msg) (string, error) {
	user := lastUser(messages)
	if strings.HasPrefix(user, "JSON:") {
		return p.structured(user), nil
	}
	return p.plain(user), nil
}

func (p *mockProvider) plain(user string) string {
	user = stripInstruction(user)
	words := strings.Fields(user)
	if len(words) == 0 {
		return "İçerik alınamadı."
	}
	n := len(words)
	head := strings.Trim(strings.Join(words[:min(n, 40)], " "), " .,")
	return fmt.Sprintf(
		"Mock AI yanıtı — OmniRoute anahtarı tanımlı değil. Gelen metnin özeti: %s … Sayfa, anahtar eklendiğinde gerçek sağlayıcıya geçer (AYARLAR → Yapay Zeka).",
		head,
	)
}

func (p *mockProvider) structured(user string) string {
	// "JSON:{type} {input}" — alan bazlı üretim
	rest := strings.TrimPrefix(user, "JSON:")
	kind := ""
	if i := strings.IndexAny(rest, " \n"); i >= 0 {
		kind, rest = rest[:i], rest[i+1:]
	}
	input := strings.Trim(strings.TrimSpace(rest), `"`)
	switch kind {
	case "title":
		// Geçerli JSON dizi üret: girişteki tırnak/kaçış karakterleri
		// (ör. haber başlığındaki alıntılar) sonucu bozmamalı.
		b, err := json.Marshal([]string{
			input + " — Mock başlık A",
			input + ": Mock bakış",
			"En güncel haliyle: " + input,
		})
		if err != nil {
			return input
		}
		return string(b)
	case "spot":
		// Düz metin: alana doğrudan yazılır, JSON tırnağı gerekmez.
		return input + " konusundaki haberde tüm detaylar masaya yatırıldı. Amacımız okuyucuya hızlı ve güvenilir bilgi aktarmak. Devamı haberimizin içeriğinde."
	case "seo_description":
		return input + " ile ilgili en güncel gelişmeler, analiz ve ayrıntılı haber — theblock.com.tr'de."
	case "seo_title":
		return input + " | Haber ve Güncel Gelişmeler — theblock.com.tr"
	case "rewrite":
		return strings.TrimSpace(input) + "\n\n— (Mock AI yeniden yazımının ilk paragrafı. OmniRoute anahtarı eklenince gerçek üretim aktifleşir.)"
	case "summarize":
		return fmt.Sprintf("Mock özet: metin %d kelime; ana başlık korunarak 3 maddelik özet üretildi. (Gerçek sağlayıcı için anahtar gerekli)", len(strings.Fields(input)))
	case "trend_brief":
		first := strings.TrimSpace(strings.SplitN(input, "\n", 2)[0])
		first = strings.TrimPrefix(first, "Terim: ")
		return "Mock trend brifingi — " + first + ":\n" +
			"• Konu son 24 saatte Google Trends'te yükseliyor; haber başlıklarını doğrulayın.\n" +
			"• Site içeriğinde kapsama zayıf: kısa, kaynaklı ve özgün bir haber taslağı hazırlayın.\n" +
			"• Başlık önerisi: \"" + first + " hakkında son gelişmeler ve merak edilenler\".\n" +
			"• Kaynak ve görseli haber öğesinden alın; doğrulanmamış iddia kullanmayın."
	default:
		return "Mock yanıt."
	}
}

func lastUser(msgs []Msg) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleUser {
			return msgs[i].Content
		}
	}
	return ""
}

func stripInstruction(s string) string {
	for _, p := range []string{"JSON:", "ÜRETİM:", "KISALT:", "YENİDENYAZ:"} {
		if strings.HasPrefix(s, p) {
			return strings.TrimSpace(strings.TrimPrefix(s, p))
		}
	}
	return s
}

func estimateTokens(msgs []Msg) int {
	t := 0
	for _, m := range msgs {
		t += len(strings.Fields(m.Content))
	}
	return t
}

func clip(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "..."
	}
	return string(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
