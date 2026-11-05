package transcribe

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// FFmpeg ffmpeg/ffprobe CLI sarmalayıcısıdır. CGO/ffmpeg kütüphanesi yerine
// sistemdeki araçlar kullanılır; böylece dağıtım ve test daha şeffaftır.
type FFmpeg struct {
	Bin      string
	ProbeBin string
}

// NewFFmpeg boş değerlerde PATH'teki ffmpeg/ffprobe'u kullanır.
func NewFFmpeg(bin, probeBin string) *FFmpeg {
	if strings.TrimSpace(bin) == "" {
		bin = "ffmpeg"
	}
	if strings.TrimSpace(probeBin) == "" {
		probeBin = "ffprobe"
	}
	return &FFmpeg{Bin: bin, ProbeBin: probeBin}
}

// MediaInfo ses dosyasının süresi ve temel akış bilgisidir.
type MediaInfo struct {
	DurationMS int64
	Format     string
	SampleRate int
	Channels   int
	HasAudio   bool
}

type probeOutput struct {
	Format struct {
		Duration   string `json:"duration"`
		FormatName string `json:"format_name"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		SampleRate string `json:"sample_rate"`
		Channels   int    `json:"channels"`
	} `json:"streams"`
}

// Probe ses dosyasını ffprobe ile inceler.
func (f *FFmpeg) Probe(ctx context.Context, path string) (MediaInfo, error) {
	cmd := exec.CommandContext(ctx, f.ProbeBin,
		"-v", "error", "-print_format", "json", "-show_format", "-show_streams", path)
	out, err := cmd.Output()
	if err != nil {
		return MediaInfo{}, fmt.Errorf("ffprobe: %w", err)
	}
	var po probeOutput
	if err := json.Unmarshal(out, &po); err != nil {
		return MediaInfo{}, fmt.Errorf("ffprobe çıktısı: %w", err)
	}
	info := MediaInfo{Format: po.Format.FormatName}
	if sec, err := strconv.ParseFloat(po.Format.Duration, 64); err == nil && sec > 0 {
		info.DurationMS = int64(sec * 1000)
	}
	for _, s := range po.Streams {
		if s.CodecType != "audio" {
			continue
		}
		info.HasAudio = true
		info.SampleRate, _ = strconv.Atoi(s.SampleRate)
		info.Channels = s.Channels
		break
	}
	if !info.HasAudio {
		return info, fmt.Errorf("dosyada ses akışı bulunamadı")
	}
	return info, nil
}

// Normalize sesi ASR için 16 kHz mono MP3'e indirir. Video akışları atılır;
// böylece 90 dakikalık bir röportaj ~40 MB'a iner.
func (f *FFmpeg) Normalize(ctx context.Context, src, dst string) error {
	cmd := exec.CommandContext(ctx, f.Bin,
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", src,
		"-vn", "-map_metadata", "-1",
		"-ac", "1", "-ar", "16000",
		"-c:a", "libmp3lame", "-b:a", "64k",
		dst)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("ffmpeg normalize: %s", msg)
	}
	st, err := os.Stat(dst)
	if err != nil {
		return fmt.Errorf("normalize çıktısı: %w", err)
	}
	if st.Size() == 0 {
		return fmt.Errorf("normalize çıktısı boş")
	}
	return nil
}
