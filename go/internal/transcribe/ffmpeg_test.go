package transcribe

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFFmpegNormalizeAndProbe gerçek ffmpeg/ffprobe ile normalizasyon
// sözleşmesini doğrular: 16 kHz mono MP3 ve okunabilir süre.
func TestFFmpegNormalizeAndProbe(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg PATH'te yok")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe PATH'te yok")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "kaynak.wav")
	gen := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1.2", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("test sesi üretilemedi: %v (%s)", err, string(out))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ff := NewFFmpeg("", "")

	info, err := ff.Probe(ctx, src)
	require.NoError(t, err)
	assert.True(t, info.HasAudio)
	assert.InDelta(t, 1200, info.DurationMS, 150)

	dst := filepath.Join(dir, "normalize.mp3")
	require.NoError(t, ff.Normalize(ctx, src, dst))

	st, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Greater(t, st.Size(), int64(0))

	out, err := ff.Probe(ctx, dst)
	require.NoError(t, err)
	assert.True(t, out.HasAudio)
	assert.Equal(t, 16_000, out.SampleRate)
	assert.Equal(t, 1, out.Channels)
	assert.InDelta(t, 1200, out.DurationMS, 200)
}

func TestFFmpegProbeRejectsFileWithoutAudio(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe PATH'te yok")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "bos.txt")
	require.NoError(t, os.WriteFile(path, []byte("ses yok"), 0o600))

	ff := NewFFmpeg("", "")
	_, err := ff.Probe(context.Background(), path)
	require.Error(t, err)
}
