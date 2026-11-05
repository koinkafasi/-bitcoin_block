package transcribe

import (
	"context"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeaksFromPCM(t *testing.T) {
	// 4 örnek: 0, +16000, -8000, +32000 → 2 kova.
	pcm := make([]byte, 0, 8)
	for _, v := range []int16{0, 16000, -8000, 32000} {
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], uint16(v))
		pcm = append(pcm, b[:]...)
	}
	peaks := peaksFromPCM(pcm, 2)
	require.Len(t, peaks, 2)
	// Kova 0: |0|,|16000| → 16000/32768 ≈ 0.488, normalize → 0.5
	assert.InDelta(t, 0.5, peaks[0], 0.01)
	// Kova 1: |−8000|,|32000| → 32000 en yüksek → normalize 1.0
	assert.InDelta(t, 1.0, peaks[1], 0.01)
}

func TestPeaksFromPCMEmpty(t *testing.T) {
	assert.Nil(t, peaksFromPCM(nil, 10))
	assert.Nil(t, peaksFromPCM([]byte{0, 0}, 0))
}

func TestPeaksFromPCMSilence(t *testing.T) {
	// Tamamen sessiz: normalize edilemez → nil (oynatıcı düz çubuğa düşer).
	pcm := make([]byte, 64)
	assert.Nil(t, peaksFromPCM(pcm, 8))
}

func TestPeaksFromPCMBucketsCappedToSamples(t *testing.T) {
	pcm := make([]byte, 8) // 4 örnek
	pcm[0] = 0xFF
	pcm[1] = 0x7F
	peaks := peaksFromPCM(pcm, 100)
	assert.LessOrEqual(t, len(peaks), 4)
}

func TestWaveformJSON(t *testing.T) {
	got, err := WaveformJSON([]float64{0, 0.5, 1})
	require.NoError(t, err)
	assert.Equal(t, "[0.000,0.500,1.000]", got)

	empty, err := WaveformJSON(nil)
	require.NoError(t, err)
	assert.Equal(t, "[]", empty)
}

func TestWaveformWithFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg PATH'te yok")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "test.wav")
	// Değişken genlikli ses üret (sine + tremolo etkisi için volume).
	gen := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-af", "volume='0.2+0.8*abs(sin(2*PI*t))':eval=frame", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("test sesi üretilemedi: %v (%s)", err, string(out))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ff := NewFFmpeg("", "")
	peaks := ff.Waveform(ctx, src, 50)
	require.NotNil(t, peaks, "dalga formu üretilmeli")
	assert.Len(t, peaks, 50)
	max := 0.0
	for _, p := range peaks {
		assert.GreaterOrEqual(t, p, 0.0)
		assert.LessOrEqual(t, p, 1.0)
		if p > max {
			max = p
		}
	}
	assert.InDelta(t, 1.0, max, 0.001, "en yüksek tepe 1.0'a normalize edilmeli")
}

func TestWaveformMissingFileReturnsNil(t *testing.T) {
	ff := NewFFmpeg("", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assert.Nil(t, ff.Waveform(ctx, filepath.Join(os.TempDir(), "olmayan-ses-xyz.mp3"), 20))
}
