package transcribe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Deepgram ikinci ASR sağlayıcısıdır. REST "pre-recorded" akışı kullanılır:
// ses dosyası tek istekte gönderilir, sağlayıcı yanıtı iş bitince döner.
// Uzun kayıtlarda worker'ın kendi iptal/ilerleme mantığı korunur.
//
// Neden ikinci sağlayıcı: AssemblyAI kesintisinde işler tamamen durmasın ve
// Türkçe doğruluk benchmark'ı iki sağlayıcıyla karşılaştırılabilsin.
type Deepgram struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewDeepgram istemciyi kurar. baseURL boşsa Deepgram varsayılanı kullanılır.
func NewDeepgram(apiKey, baseURL, model string) *Deepgram {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = "https://api.deepgram.com"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "nova-3"
	}
	return &Deepgram{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(base, "/"),
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Minute},
	}
}

// Name sağlayıcı adını döndürür.
func (d *Deepgram) Name() string { return "deepgram" }

// deepgramResponse pre-recorded yanıtının kullandığımız alt kümesi.
type deepgramResponse struct {
	Metadata struct {
		Duration  float64 `json:"duration"`
		RequestID string  `json:"request_id"`
	} `json:"metadata"`
	Results struct {
		Channels []struct {
			DetectedLanguage string `json:"detected_language"`
			Alternatives     []struct {
				Transcript string  `json:"transcript"`
				Confidence float64 `json:"confidence"`
				Words      []struct {
					Word       string  `json:"word"`
					Start      float64 `json:"start"`
					End        float64 `json:"end"`
					Confidence float64 `json:"confidence"`
					Speaker    *int    `json:"speaker"`
					Punctuated string  `json:"punctuated_word"`
				} `json:"words"`
			} `json:"alternatives"`
		} `json:"channels"`
	} `json:"results"`
	ErrCode string `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

// Transcribe sesi Deepgram'a gönderir ve sonucu döndürür. Sağlayıcı tek
// istekte yanıt verdiği için ilerleme yalnız kaba aşamalarla bildirilir.
func (d *Deepgram) Transcribe(ctx context.Context, req Request, progress ProgressFunc) (Result, error) {
	if req.Audio == nil {
		return Result{}, fmt.Errorf("ses verisi boş")
	}
	lang := strings.TrimSpace(req.Language)
	if lang == "" {
		lang = "tr"
	}
	if progress != nil {
		progress(10, "")
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("audio", filenameOr(req.Filename, "audio.mp3"))
	if err != nil {
		return Result{}, err
	}
	if _, err := io.Copy(part, req.Audio); err != nil {
		return Result{}, fmt.Errorf("ses okunamadı: %w", err)
	}
	if err := mw.Close(); err != nil {
		return Result{}, err
	}

	q := url.Values{}
	q.Set("model", d.model)
	q.Set("language", lang)
	q.Set("punctuate", "true")
	q.Set("smart_format", "true")
	q.Set("diarize", "true")
	if req.SpeakersExpected > 0 {
		// Deepgram "diarize" ile konuşmacıları kendi bulur; beklenen sayı
		// yalnız ipucu olarak gönderilir.
		q.Set("utterances", "true")
	}

	endpoint := d.baseURL + "/v1/listen?" + q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return Result{}, err
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	httpReq.Header.Set("Authorization", "Token "+d.apiKey)

	if progress != nil {
		progress(35, "")
	}
	resp, err := d.client.Do(httpReq)
	if err != nil {
		return Result{}, fmt.Errorf("Deepgram isteği: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))

	var dr deepgramResponse
	if err := json.Unmarshal(raw, &dr); err != nil {
		return Result{}, fmt.Errorf("Deepgram yanıtı okunamadı (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(dr.ErrMsg)
		if msg == "" {
			msg = clipText(string(raw), 300)
		}
		return Result{}, fmt.Errorf("Deepgram (%d): %s", resp.StatusCode, msg)
	}
	if dr.ErrMsg != "" {
		return Result{}, fmt.Errorf("Deepgram: %s", dr.ErrMsg)
	}
	if progress != nil {
		progress(90, dr.Metadata.RequestID)
	}

	res := convertDeepgram(dr, lang, d.Name(), d.model)
	if strings.TrimSpace(res.Text) == "" {
		return Result{}, fmt.Errorf("Deepgram boş transkript döndürdü")
	}
	return res, nil
}

// convertDeepgram yanıtı sağlayıcıdan bağımsız Result'a çevirir.
func convertDeepgram(dr deepgramResponse, requestedLang, provider, model string) Result {
	res := Result{
		Language:      requestedLang,
		Provider:      provider,
		Model:         model,
		ProviderJobID: dr.Metadata.RequestID,
	}
	if dr.Metadata.Duration > 0 {
		res.DurationMS = int64(dr.Metadata.Duration * 1000)
	}
	if len(dr.Results.Channels) == 0 || len(dr.Results.Channels[0].Alternatives) == 0 {
		return res
	}
	ch := dr.Results.Channels[0]
	if dl := strings.TrimSpace(ch.DetectedLanguage); dl != "" {
		res.Language = dl
	}
	alt := ch.Alternatives[0]
	res.Text = strings.TrimSpace(alt.Transcript)

	// Deepgram yalnız kelime dizisi döndürür (utterance yok). Her kelimeyi
	// ayrı segment yapmak panelde binlerce tek kelimelik satır üretirdi;
	// bunun yerine AssemblyAI ile aynı gruplayıcıdan geçirilir: konuşmacı
	// değişimi, cümle sonu veya 40 kelimede segment kapanır. Kelime zamanları
	// segmentin içinde korunur.
	speakerSet := map[string]bool{}
	words := make([]wordItem, 0, len(alt.Words))
	for _, w := range alt.Words {
		text := strings.TrimSpace(w.Punctuated)
		if text == "" {
			text = strings.TrimSpace(w.Word)
		}
		if text == "" {
			continue
		}
		sp := ""
		if w.Speaker != nil {
			sp = speakerLabel(*w.Speaker)
			speakerSet[sp] = true
		}
		words = append(words, wordItem{
			Text:       text,
			Start:      int64(w.Start * 1000),
			End:        int64(w.End * 1000),
			Speaker:    sp,
			Confidence: w.Confidence,
		})
	}
	res.Segments = groupWordItems(words, res.DurationMS)
	if len(res.Segments) == 0 && res.Text != "" {
		res.Segments = []Segment{{StartMS: 0, EndMS: res.DurationMS, Text: res.Text}}
	}
	if res.DurationMS == 0 && len(res.Segments) > 0 {
		res.DurationMS = res.Segments[len(res.Segments)-1].EndMS
	}
	for sp := range speakerSet {
		res.Speakers = append(res.Speakers, sp)
	}
	sort.Strings(res.Speakers)
	return res
}

// speakerLabel Deepgram'ın 0 tabanlı konuşmacı indeksini A/B/C etiketine
// çevirir; AssemblyAI etiketleriyle aynı biçimi korur.
func speakerLabel(n int) string {
	if n < 0 {
		n = 0
	}
	if n > 25 {
		n = 25
	}
	return string(rune('A' + n))
}

func filenameOr(name, def string) string {
	if strings.TrimSpace(name) == "" {
		return def
	}
	return name
}

func clipText(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
