package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Request tek bir tamamlamanın sağlayıcıdan bağımsız girdisidir.
type Request struct {
	Messages    []Msg
	Model       string
	JSONMode    bool
	MaxTokens   int
	Temperature float64
}

// Response sağlayıcı yanıtı + kullanım/maliyet temelidir.
type Response struct {
	Text             string
	Provider         string
	Model            string
	PromptTokens     int
	CompletionTokens int
	LatencyMS        int
}

// Completer Chat'ten daha zengin (model, JSON modu, usage) tamamlama sunan
// sağlayıcıların arayüzüdür. Mevcut Chat arayüzü geriye dönük korunur.
type Completer interface {
	Name() string
	Complete(ctx context.Context, req Request) (Response, error)
}

// Complete sırayla sağlayıcıları dener; hepsi düşerse mock'a düşer.
func (c *Client) Complete(ctx context.Context, req Request) (Response, error) {
	var lastErr error
	for _, p := range c.providers {
		cp, ok := p.(Completer)
		if !ok {
			continue
		}
		start := time.Now()
		resp, err := cp.Complete(ctx, req)
		if err == nil {
			resp.LatencyMS = int(time.Since(start).Milliseconds())
			if resp.Provider == "" {
				resp.Provider = p.Name()
			}
			c.logger.Info("ai complete", "provider", p.Name(), "model", resp.Model,
				"prompt_tokens", resp.PromptTokens, "completion_tokens", resp.CompletionTokens,
				"latency_ms", resp.LatencyMS)
			return resp, nil
		}
		lastErr = err
		c.logger.Warn("ai provider başarısız, sıradakine geçiliyor", "provider", p.Name(), "error", err)
	}
	return Response{}, fmt.Errorf("tüm AI sağlayıcıları başarısız: %w", lastErr)
}

// CompleteStrict mock'a DÜŞMEZ. Editör/research gibi gerçek çıktı gerektiren
// akışlarda kullanılır; sağlayıcı yoksa veya hata verirse açık hata döner.
func (c *Client) CompleteStrict(ctx context.Context, req Request) (Response, error) {
	var lastErr error
	for _, p := range c.providers {
		if p.Name() == "mock" {
			continue
		}
		cp, ok := p.(Completer)
		if !ok {
			continue
		}
		start := time.Now()
		resp, err := cp.Complete(ctx, req)
		if err == nil {
			resp.LatencyMS = int(time.Since(start).Milliseconds())
			if resp.Provider == "" {
				resp.Provider = p.Name()
			}
			c.logger.Info("ai complete", "provider", p.Name(), "model", resp.Model,
				"prompt_tokens", resp.PromptTokens, "completion_tokens", resp.CompletionTokens,
				"latency_ms", resp.LatencyMS)
			return resp, nil
		}
		lastErr = err
		c.logger.Warn("ai provider başarısız", "provider", p.Name(), "error", err)
	}
	if lastErr == nil {
		return Response{}, fmt.Errorf("gerçek AI sağlayıcısı yapılandırılmadı")
	}
	return Response{}, fmt.Errorf("AI sağlayıcısına ulaşılamadı: %w", lastErr)
}

type completionReq struct {
	Model          string          `json:"model"`
	Messages       []apiMsg        `json:"messages"`
	Stream         bool            `json:"stream"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	// ReasoningEffort muhakeme tabanlı modellerde düşünme bütçesini kısar.
	//
	// ÖLÇÜM (6 Kasım 2026, evren/deepseek-v4.1-flash, OmniRoute): 4.000
	// karakterlik bir istemde model `max_tokens` bütçesinin TAMAMINI
	// muhakemeye harcayıp `finish_reason=length` ile BOŞ içerik döndürdü —
	// otomatik yazım işleri "boş yanıt" hatasıyla düşüyordu. `low` ile aynı
	// istem tam JSON üretti (108-162 sn). Sağlayıcı parametreyi desteklemezse
	// boş bırakılır (env ile kapatılabilir).
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

func (p *httpProvider) Complete(ctx context.Context, req Request) (Response, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = p.model
	}
	body, err := json.Marshal(completionReq{
		Model:           model,
		Messages:        toAPIMsgs(req.Messages),
		Stream:          false,
		MaxTokens:       req.MaxTokens,
		Temperature:     req.Temperature,
		ReasoningEffort: p.reasoningEffort,
	})
	if err != nil {
		return Response{}, err
	}
	// JSON modu bazı OpenAI-uyumlu uçlarda desteklenmez; yalnız istendiğinde ekle.
	if req.JSONMode {
		body, err = json.Marshal(completionReq{
			Model: model, Messages: toAPIMsgs(req.Messages), Stream: false,
			MaxTokens: req.MaxTokens, Temperature: req.Temperature,
			ResponseFormat:  &responseFormat{Type: "json_object"},
			ReasoningEffort: p.reasoningEffort,
		})
		if err != nil {
			return Response{}, err
		}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(p.baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("AI uç noktası %d: %s", resp.StatusCode, clip(raw, 300))
	}
	var cr chatCompletionResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		return Response{}, err
	}
	if len(cr.Choices) == 0 {
		return Response{}, fmt.Errorf("AI boş yanıt")
	}
	return Response{
		Text:             cr.Choices[0].Message.Content,
		Provider:         p.name,
		Model:            model,
		PromptTokens:     cr.Usage.PromptTokens,
		CompletionTokens: cr.Usage.CompletionTokens,
	}, nil
}

func (p *mockProvider) Complete(ctx context.Context, req Request) (Response, error) {
	out, err := p.Chat(ctx, req.Messages)
	if err != nil {
		return Response{}, err
	}
	return Response{
		Text:             out,
		Provider:         "mock",
		Model:            "mock",
		PromptTokens:     estimateTokens(req.Messages),
		CompletionTokens: len(strings.Fields(out)),
	}, nil
}
