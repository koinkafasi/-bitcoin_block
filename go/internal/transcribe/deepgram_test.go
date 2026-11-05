package transcribe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromConfigSelectsProvider(t *testing.T) {
	assert.Nil(t, FromConfig("deepgram", "", "", ""))
	assert.Nil(t, FromConfig("assemblyai", "  ", "", ""))

	a := FromConfig("", "k", "", "")
	require.NotNil(t, a)
	assert.Equal(t, "assemblyai", a.Name())

	d := FromConfig("Deepgram", "k", "", "")
	require.NotNil(t, d)
	assert.Equal(t, "deepgram", d.Name())
}

func TestProviderName(t *testing.T) {
	assert.Equal(t, "deepgram", ProviderName("deepgram"))
	assert.Equal(t, "deepgram", ProviderName(" DeepGram "))
	assert.Equal(t, "assemblyai", ProviderName(""))
	assert.Equal(t, "assemblyai", ProviderName("bilinmeyen"))
}

func TestDeepgramTranscribe(t *testing.T) {
	var gotQuery string
	var gotAuth string
	var gotBodyLen int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/listen" {
			t.Errorf("beklenmeyen istek: %s %s", r.Method, r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBodyLen = len(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"metadata":{"duration":12.5,"request_id":"req-123"},
			"results":{"channels":[{"detected_language":"tr","alternatives":[{
				"transcript":"Yatırım planı 2027 yılında başlayacak.",
				"confidence":0.96,
				"words":[
					{"word":"yatırım","punctuated_word":"Yatırım","start":0.0,"end":0.4,"confidence":0.98,"speaker":0},
					{"word":"planı","punctuated_word":"planı","start":0.4,"end":0.8,"confidence":0.97,"speaker":0},
					{"word":"başlayacak","punctuated_word":"başlayacak.","start":0.8,"end":1.6,"confidence":0.95,"speaker":1}
				]
			}]}]}
		}`)
	}))
	defer srv.Close()

	d := NewDeepgram("dg-key", srv.URL, "")
	var progress []int
	res, err := d.Transcribe(context.Background(), Request{
		Audio: strings.NewReader("ID3TEST"), Filename: "audio.mp3",
		Language: "tr", SpeakersExpected: 2,
	}, func(p int, id string) { progress = append(progress, p) })
	require.NoError(t, err)

	assert.Equal(t, "Token dg-key", gotAuth)
	assert.Contains(t, gotQuery, "model=nova-3")
	assert.Contains(t, gotQuery, "language=tr")
	assert.Contains(t, gotQuery, "diarize=true")
	assert.Contains(t, gotQuery, "punctuate=true")
	assert.Contains(t, gotQuery, "utterances=true")
	assert.Greater(t, gotBodyLen, 0, "multipart gövde gönderilmeli")

	assert.Equal(t, []int{10, 35, 90}, progress)
	assert.Equal(t, "deepgram", res.Provider)
	assert.Equal(t, "nova-3", res.Model)
	assert.Equal(t, "req-123", res.ProviderJobID)
	assert.Equal(t, "tr", res.Language)
	assert.Equal(t, int64(12_500), res.DurationMS)
	assert.Equal(t, "Yatırım planı 2027 yılında başlayacak.", res.Text)
	assert.Equal(t, []string{"A", "B"}, res.Speakers)
	// Deepgram utterance döndürmez, yalnız kelime dizisi. Kelimeler konuşmacı
	// değişiminde ve cümle sonunda gruplanır — her kelime ayrı segment olmaz.
	require.Len(t, res.Segments, 2)

	assert.Equal(t, "Yatırım planı", res.Segments[0].Text)
	assert.Equal(t, "A", res.Segments[0].Speaker)
	assert.Equal(t, int64(0), res.Segments[0].StartMS)
	assert.Equal(t, int64(800), res.Segments[0].EndMS)

	assert.Equal(t, "başlayacak.", res.Segments[1].Text)
	assert.Equal(t, "B", res.Segments[1].Speaker)
	assert.Equal(t, int64(800), res.Segments[1].StartMS)

	// Kelime zamanları segmentin içinde korunur.
	require.Len(t, res.Segments[0].Words, 2)
	assert.Equal(t, "Yatırım", res.Segments[0].Words[0].Text)
	assert.Equal(t, int64(0), res.Segments[0].Words[0].StartMS)
	assert.Equal(t, int64(400), res.Segments[0].Words[0].EndMS)
	assert.Equal(t, "planı", res.Segments[0].Words[1].Text)
	assert.Equal(t, int64(400), res.Segments[0].Words[1].StartMS)
	require.Len(t, res.Segments[1].Words, 1)
	assert.Equal(t, "başlayacak.", res.Segments[1].Words[0].Text)
}

func TestDeepgramTranscribeHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"err_code":"INVALID_AUTH","err_msg":"Invalid credentials"}`)
	}))
	defer srv.Close()

	d := NewDeepgram("bad", srv.URL, "nova-3")
	_, err := d.Transcribe(context.Background(), Request{Audio: strings.NewReader("x")}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid credentials")
}

func TestDeepgramRequiresAudio(t *testing.T) {
	d := NewDeepgram("k", "", "")
	_, err := d.Transcribe(context.Background(), Request{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ses verisi boş")
}

func TestDeepgramEmptyTranscript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"metadata":{"duration":3},"results":{"channels":[{"alternatives":[{"transcript":""}]}]}}`)
	}))
	defer srv.Close()

	d := NewDeepgram("k", srv.URL, "")
	_, err := d.Transcribe(context.Background(), Request{Audio: strings.NewReader("x")}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boş transkript")
}

// Konuşmacı indeksi A/B etiketine çevrilmeli (AssemblyAI ile aynı biçim).
func TestSpeakerLabel(t *testing.T) {
	assert.Equal(t, "A", speakerLabel(0))
	assert.Equal(t, "B", speakerLabel(1))
	assert.Equal(t, "C", speakerLabel(2))
	assert.Equal(t, "A", speakerLabel(-1))
	assert.Equal(t, "Z", speakerLabel(99))
}

func TestConvertDeepgramWithoutWords(t *testing.T) {
	var dr deepgramResponse
	require.NoError(t, json.Unmarshal([]byte(`{
		"metadata":{"duration":4},
		"results":{"channels":[{"alternatives":[{"transcript":"Merhaba dünya."}]}]}
	}`), &dr))

	res := convertDeepgram(dr, "tr", "deepgram", "nova-3")
	assert.Equal(t, "Merhaba dünya.", res.Text)
	require.Len(t, res.Segments, 1)
	assert.Equal(t, int64(4_000), res.DurationMS)
}
