package transcribe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromConfigRequiresAPIKey(t *testing.T) {
	assert.Nil(t, FromConfig("assemblyai", "", "", ""))
	assert.Nil(t, FromConfig("assemblyai", "   ", "http://example.invalid", "best"))

	p := FromConfig("assemblyai", "test-key", "", "best")
	require.NotNil(t, p)
	assert.Equal(t, "assemblyai", p.Name())
}

func TestSpeechModelsMapping(t *testing.T) {
	cases := map[string][]string{
		"":                  {"universal-3-5-pro", "universal-2"},
		"best":              {"universal-3-5-pro", "universal-2"},
		"BEST":              {"universal-3-5-pro", "universal-2"},
		"auto":              {"universal-3-5-pro", "universal-2"},
		"nano":              {"universal-2"},
		"fast":              {"universal-2"},
		"universal-2":       {"universal-2"},
		"universal-3-pro":   {"universal-3-pro"},
		"universal-3-5-pro": {"universal-3-5-pro"},
		"bilinmeyen":        {"universal-3-5-pro", "universal-2"},
	}
	for model, want := range cases {
		p := NewAssemblyAI("k", "", model)
		assert.Equalf(t, want, p.speechModels(), "model %q", model)
	}
}

func TestAssemblyAITranscribe(t *testing.T) {
	var uploads, submits, polls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/upload":
			atomic.AddInt32(&uploads, 1)
			if got := r.Header.Get("Authorization"); got != "test-key" {
				t.Errorf("Authorization yanlış: %q", got)
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != "ID3TEST" {
				t.Errorf("yüklenen ses gövdesi yanlış: %q", string(body))
			}
			_, _ = io.WriteString(w, `{"upload_url":"https://cdn.example/audio.mp3"}`)

		case r.Method == http.MethodPost && r.URL.Path == "/v2/transcript":
			atomic.AddInt32(&submits, 1)
			var params map[string]any
			if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
				t.Errorf("transcript isteği okunamadı: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if params["audio_url"] != "https://cdn.example/audio.mp3" {
				t.Errorf("audio_url yanlış: %v", params["audio_url"])
			}
			if params["language_code"] != "tr" {
				t.Errorf("language_code yanlış: %v", params["language_code"])
			}
			if params["speaker_labels"] != true {
				t.Errorf("speaker_labels açık değil: %v", params["speaker_labels"])
			}
			if params["speakers_expected"] != float64(2) {
				t.Errorf("speakers_expected yanlış: %v", params["speakers_expected"])
			}
			// Deprecated speech_model GÖNDERİLMEMELİ; liste gönderilmeli.
			if _, ok := params["speech_model"]; ok {
				t.Errorf("deprecated speech_model gönderilmiş")
			}
			models, ok := params["speech_models"].([]any)
			if !ok || len(models) == 0 {
				t.Errorf("speech_models listesi eksik: %v", params["speech_models"])
			}
			if params["punctuate"] != true || params["format_text"] != true {
				t.Errorf("noktalama/biçim ayarları eksik: %v", params)
			}
			_, _ = io.WriteString(w, `{"id":"tr_123","status":"queued"}`)

		case r.Method == http.MethodGet && r.URL.Path == "/v2/transcript/tr_123":
			n := atomic.AddInt32(&polls, 1)
			switch n {
			case 1:
				_, _ = io.WriteString(w, `{"id":"tr_123","status":"queued"}`)
			case 2:
				_, _ = io.WriteString(w, `{"id":"tr_123","status":"processing"}`)
			default:
				_, _ = io.WriteString(w, `{
					"id":"tr_123","status":"completed",
					"text":"Merhaba dünya. Nasılsın?",
					"language_code":"tr","audio_duration":12.5,
					"utterances":[
						{"start":0,"end":4000,"speaker":"A","text":"Merhaba dünya.","confidence":0.98},
						{"start":4000,"end":9000,"speaker":"B","text":"Nasılsın?","confidence":0.95}
					]
				}`)
			}

		default:
			t.Errorf("beklenmeyen istek: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := NewAssemblyAI("test-key", srv.URL, "best")
	p.pollInterval = time.Millisecond

	var progress []int
	res, err := p.Transcribe(context.Background(), Request{
		Audio: strings.NewReader("ID3TEST"), Language: "tr", SpeakersExpected: 2,
	}, func(percent int, providerJobID string) {
		progress = append(progress, percent)
	})
	require.NoError(t, err)

	assert.Equal(t, int32(1), atomic.LoadInt32(&uploads))
	assert.Equal(t, int32(1), atomic.LoadInt32(&submits))
	assert.Equal(t, int32(3), atomic.LoadInt32(&polls))
	assert.Equal(t, []int{10, 20, 25, 55, 90}, progress)

	assert.Equal(t, "Merhaba dünya. Nasılsın?", res.Text)
	assert.Equal(t, "tr", res.Language)
	assert.Equal(t, int64(12_500), res.DurationMS)
	assert.Equal(t, "assemblyai", res.Provider)
	assert.Equal(t, "best", res.Model)
	assert.Equal(t, "tr_123", res.ProviderJobID)
	assert.Equal(t, []string{"A", "B"}, res.Speakers)
	require.Len(t, res.Segments, 2)
	assert.Equal(t, int64(0), res.Segments[0].StartMS)
	assert.Equal(t, int64(4_000), res.Segments[0].EndMS)
	assert.Equal(t, "A", res.Segments[0].Speaker)
	assert.Equal(t, "Merhaba dünya.", res.Segments[0].Text)
	assert.InDelta(t, 0.98, res.Segments[0].Confidence, 0.0001)
	assert.Equal(t, "B", res.Segments[1].Speaker)
}

func TestAssemblyAITranscribeProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/upload":
			_, _ = io.WriteString(w, `{"upload_url":"https://cdn.example/audio.mp3"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/transcript":
			_, _ = io.WriteString(w, `{"id":"tr_err","status":"queued"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/transcript/tr_err":
			_, _ = io.WriteString(w, `{"id":"tr_err","status":"error","error":"Ses anlaşılamadı"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := NewAssemblyAI("test-key", srv.URL, "")
	p.pollInterval = time.Millisecond
	_, err := p.Transcribe(context.Background(), Request{
		Audio: strings.NewReader("ID3TEST"), Language: "tr",
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Ses anlaşılamadı")
}

func TestAssemblyAIAPIErrorSurfacesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v2/upload":
			_, _ = io.WriteString(w, `{"upload_url":"https://cdn.example/a.mp3"}`)
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"The speech_model parameter is deprecated."}`)
		}
	}))
	defer srv.Close()

	p := NewAssemblyAI("k", srv.URL, "best")
	_, err := p.Transcribe(context.Background(), Request{Audio: strings.NewReader("x")}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deprecated")
}

func TestAssemblyAITranscribeRequiresAudio(t *testing.T) {
	p := NewAssemblyAI("test-key", "", "")
	_, err := p.Transcribe(context.Background(), Request{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ses verisi boş")
}

func TestConvertTranscriptGroupsWordsWithoutUtterances(t *testing.T) {
	var tr assemblyTranscript
	require.NoError(t, json.Unmarshal([]byte(`{
		"text":"Merhaba dünya. Nasılsın?",
		"language_code":"tr",
		"audio_duration":12.5,
		"words":[
			{"text":"Merhaba","start":0,"end":400},
			{"text":"dünya.","start":400,"end":800},
			{"text":"Nasılsın?","start":900,"end":1400}
		]
	}`), &tr))

	res := convertTranscript(tr, "tr_words", "tr", "assemblyai", "best")
	require.Len(t, res.Segments, 2)
	assert.Equal(t, "Merhaba dünya.", res.Segments[0].Text)
	assert.Equal(t, int64(0), res.Segments[0].StartMS)
	assert.Equal(t, int64(800), res.Segments[0].EndMS)
	assert.Equal(t, "Nasılsın?", res.Segments[1].Text)
	assert.Equal(t, int64(900), res.Segments[1].StartMS)
	assert.Equal(t, int64(1_400), res.Segments[1].EndMS)
	assert.Equal(t, int64(12_500), res.DurationMS)
}

func TestConvertTranscriptSortsSpeakers(t *testing.T) {
	var tr assemblyTranscript
	require.NoError(t, json.Unmarshal([]byte(`{
		"text":"İki konuşmacı",
		"utterances":[
			{"start":0,"end":1000,"speaker":"B","text":"İkinci"},
			{"start":1000,"end":2000,"speaker":"A","text":"Birinci"},
			{"start":2000,"end":3000,"speaker":"B","text":"Tekrar ikinci"}
		]
	}`), &tr))

	res := convertTranscript(tr, "tr_speakers", "tr", "assemblyai", "best")
	assert.Equal(t, []string{"A", "B"}, res.Speakers)
	assert.Equal(t, int64(3_000), res.DurationMS)
	require.Len(t, res.Segments, 3)
}
