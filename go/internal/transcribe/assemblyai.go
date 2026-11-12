package transcribe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type AssemblyAI struct {
	apiKey       string
	baseURL      string
	model        string
	client       *http.Client
	pollInterval time.Duration
}

// NewAssemblyAI istemciyi kurar. baseURL boşsa varsayılan kullanılır.
func NewAssemblyAI(apiKey, baseURL, model string) *AssemblyAI {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = "https://api.assemblyai.com"
	}
	return &AssemblyAI{
		apiKey:       strings.TrimSpace(apiKey),
		baseURL:      strings.TrimRight(base, "/"),
		model:        strings.TrimSpace(model),
		client:       &http.Client{Timeout: 30 * time.Minute},
		pollInterval: 3 * time.Second,
	}
}

// Name sağlayıcı adını döndürür.
func (a *AssemblyAI) Name() string { return "assemblyai" }

// speechModels yapılandırılan model adını API'nin beklediği listeye çevirir.
// Boş/"best" → en yüksek doğruluk + kapsama için iki model (API önerisi).
func (a *AssemblyAI) speechModels() []string {
	switch strings.ToLower(a.model) {
	case "", "best", "auto":
		return []string{"universal-3-5-pro", "universal-2"}
	case "nano", "fast", "universal-2":
		return []string{"universal-2"}
	case "universal-3-pro":
		return []string{"universal-3-pro"}
	case "universal-3-5-pro":
		return []string{"universal-3-5-pro"}
	}
	// Bilinmeyen değer: API'yi kırmak yerine güvenli varsayılana düş.
	return []string{"universal-3-5-pro", "universal-2"}
}

// assemblyUploadResp upload yanıtı.
type assemblyUploadResp struct {
	UploadURL string `json:"upload_url"`
}

// assemblyTranscript POST/GET yanıtının kullandığımız alanları.
type assemblyTranscript struct {
	ID            string              `json:"id"`
	Status        string              `json:"status"`
	Text          string              `json:"text"`
	LanguageCode  string              `json:"language_code"`
	AudioDuration float64             `json:"audio_duration"`
	Error         string              `json:"error"`
	Speakers      []string            `json:"-"`
	Utterances    []assemblyUtterance `json:"utterances"`
	Words         []assemblyWord      `json:"words"`
}

// assemblyUtterance konuşmacı ayrımı açıkken dönen tek konuşma bölümüdür.
type assemblyUtterance struct {
	Start      int64   `json:"start"`
	End        int64   `json:"end"`
	Text       string  `json:"text"`
	Speaker    string  `json:"speaker"`
	Confidence float64 `json:"confidence"`
}

// assemblyWord AssemblyAI'nin kelime nesnesidir. Hem konuşmacı ayrımı yokken
// segment üretmek hem de utterance'lara kelime zamanı bağlamak için kullanılır.
type assemblyWord struct {
	Text       string  `json:"text"`
	Start      int64   `json:"start"`
	End        int64   `json:"end"`
	Speaker    string  `json:"speaker"`
	Confidence float64 `json:"confidence"`
}

// Transcribe sesi yükler, transkript işini başlatır ve tamamlanana kadar
// durum sorgular.
func (a *AssemblyAI) Transcribe(ctx context.Context, req Request, progress ProgressFunc) (Result, error) {
	if req.Audio == nil {
		return Result{}, fmt.Errorf("ses verisi boş")
	}
	lang := strings.TrimSpace(req.Language)
	if lang == "" {
		lang = "tr"
	}

	uploadURL, err := a.upload(ctx, req.Audio)
	if err != nil {
		return Result{}, fmt.Errorf("AssemblyAI yükleme: %w", err)
	}
	if progress != nil {
		progress(10, "")
	}

	params := map[string]any{
		"audio_url":      uploadURL,
		"speech_models":  a.speechModels(),
		"language_code":  lang,
		"speaker_labels": true,
		"punctuate":      true,
		"format_text":    true,
	}
	if req.SpeakersExpected > 0 {
		params["speakers_expected"] = req.SpeakersExpected
	}

	var submitted assemblyTranscript
	if err := a.postJSON(ctx, "/v2/transcript", params, &submitted); err != nil {
		return Result{}, fmt.Errorf("AssemblyAI iş başlatma: %w", err)
	}
	id := strings.TrimSpace(submitted.ID)
	if id == "" {
		return Result{}, fmt.Errorf("AssemblyAI iş kimliği dönmedi")
	}
	if progress != nil {
		progress(20, id)
	}

	interval := a.pollInterval
	if interval <= 0 {
		interval = 3 * time.Second
	}
	for {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-time.After(interval):
		}
		var t assemblyTranscript
		if err := a.getJSON(ctx, "/v2/transcript/"+id, &t); err != nil {
			return Result{}, fmt.Errorf("AssemblyAI durum sorgusu: %w", err)
		}
		switch strings.ToLower(t.Status) {
		case "queued":
			if progress != nil {
				progress(25, id)
			}
		case "processing":
			if progress != nil {
				progress(55, id)
			}
		case "completed":
			if progress != nil {
				progress(90, id)
			}
			return convertTranscript(t, id, lang, a.Name(), a.model), nil
		case "error":
			msg := strings.TrimSpace(t.Error)
			if msg == "" {
				msg = "bilinmeyen sağlayıcı hatası"
			}
			return Result{}, fmt.Errorf("AssemblyAI: %s", msg)
		default:
			if progress != nil {
				progress(35, id)
			}
		}
	}
}

// upload sesi /v2/upload uçuna akıtır ve dosya URL'sini döndürür.
func (a *AssemblyAI) upload(ctx context.Context, r io.Reader) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v2/upload", r)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", a.apiKey)
	httpReq.Header.Set("Content-Type", "application/octet-stream")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiError(raw))
	}
	var out assemblyUploadResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("yanıt okunamadı: %w", err)
	}
	if strings.TrimSpace(out.UploadURL) == "" {
		return "", fmt.Errorf("upload_url dönmedi")
	}
	return out.UploadURL, nil
}

func (a *AssemblyAI) postJSON(ctx context.Context, path string, body, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	return a.do(httpReq, out)
}

func (a *AssemblyAI) getJSON(ctx context.Context, path string, out any) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+path, nil)
	if err != nil {
		return err
	}
	return a.do(httpReq, out)
}

func (a *AssemblyAI) do(req *http.Request, out any) error {
	req.Header.Set("Authorization", a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiError(raw))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("yanıt okunamadı: %w", err)
	}
	return nil
}

// apiError API'nin {"error": "..."} gövdesinden mesajı çıkarır.
func apiError(raw []byte) string {
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &e); err == nil && strings.TrimSpace(e.Error) != "" {
		return e.Error
	}
	return clipText(string(raw), 300)
}

// convertTranscript API yanıtını sağlayıcıdan bağımsız Result'a çevirir.
func convertTranscript(t assemblyTranscript, providerJobID, requestedLang, provider, model string) Result {
	res := Result{
		Text:          strings.TrimSpace(t.Text),
		Language:      strings.TrimSpace(t.LanguageCode),
		Provider:      provider,
		Model:         model,
		ProviderJobID: providerJobID,
	}
	if res.Language == "" {
		res.Language = requestedLang
	}
	if t.AudioDuration > 0 {
		res.DurationMS = int64(t.AudioDuration * 1000)
	}

	speakerSet := map[string]bool{}
	for _, u := range t.Utterances {
		seg := Segment{
			StartMS:    u.Start,
			EndMS:      u.End,
			Speaker:    strings.TrimSpace(u.Speaker),
			Text:       strings.TrimSpace(u.Text),
			Confidence: u.Confidence,
		}
		if seg.Text == "" {
			continue
		}
		if seg.Speaker != "" {
			speakerSet[seg.Speaker] = true
		}
		res.Segments = append(res.Segments, seg)
	}
	// Kelime zamanları. Utterance'lardan segment kurulduysa kelimeleri ilgili
	// segmente dağıt (karaoke vurgu, kelimeye tıklayıp sarma, altyazı).
	if len(res.Segments) > 0 && len(t.Words) > 0 {
		attachWords(res.Segments, toWordItems(t.Words))
	}
	if len(res.Segments) == 0 && len(t.Words) > 0 {
		res.Segments = groupWords(t.Words, res.DurationMS)
	}
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

// wordItem transkript kelimesinin dönüşüm için gereken alanları.
type wordItem struct {
	Text       string
	Start      int64
	End        int64
	Speaker    string
	Confidence float64
}

func groupWords(raw []assemblyWord, durationMS int64) []Segment {
	return groupWordItems(toWordItems(raw), durationMS)
}

// toWordItems sağlayıcıya özgü kelime nesnelerini ortak wordItem'a çevirir.
func toWordItems(raw []assemblyWord) []wordItem {
	words := make([]wordItem, 0, len(raw))
	for _, w := range raw {
		words = append(words, wordItem(w))
	}
	return words
}

func attachWords(segs []Segment, words []wordItem) {
	wi := 0
	for i := range segs {
		for wi < len(words) {
			w := words[wi]
			mid := w.Start + (w.End-w.Start)/2
			if mid < segs[i].StartMS {
				wi++
				continue
			}
			if mid > segs[i].EndMS {
				break
			}
			if text := strings.TrimSpace(w.Text); text != "" {
				segs[i].Words = append(segs[i].Words, Word{
					StartMS: w.Start, EndMS: w.End,
					Text: text, Confidence: w.Confidence,
				})
			}
			wi++
		}
	}
}

func groupWordItems(words []wordItem, durationMS int64) []Segment {
	const maxWords = 40
	var out []Segment
	var cur Segment
	var count int
	flush := func() {
		cur.Text = strings.TrimSpace(cur.Text)
		if cur.Text != "" {
			out = append(out, cur)
		}
		cur = Segment{}
		count = 0
	}
	for _, w := range words {
		text := strings.TrimSpace(w.Text)
		if text == "" {
			continue
		}
		speaker := strings.TrimSpace(w.Speaker)
		// Konuşmacı değiştiyse segmenti kapat: tek satırda iki kişi olmaz.
		if count > 0 && speaker != cur.Speaker {
			flush()
		}
		if count == 0 {
			cur.StartMS = w.Start
			cur.Speaker = speaker
			cur.Confidence = w.Confidence
		}
		if cur.Text != "" {
			cur.Text += " "
		}
		cur.Text += text
		cur.EndMS = w.End
		cur.Words = append(cur.Words, Word{
			StartMS: w.Start, EndMS: w.End,
			Text: text, Confidence: w.Confidence,
		})
		count++
		if count >= maxWords || strings.ContainsAny(text, ".!?") {
			flush()
		}
	}
	flush()
	if len(out) == 0 && durationMS > 0 {
		return nil
	}
	return out
}
