package transcribe

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
)

const WaveformBucketCount = 400

func (f *FFmpeg) Waveform(ctx context.Context, audioPath string, buckets int) []float64 {
	if buckets <= 0 {
		buckets = WaveformBucketCount
	}
	cmd := exec.CommandContext(ctx, f.Bin,
		"-hide_banner", "-loglevel", "error",
		"-i", audioPath,
		"-ac", "1", "-ar", "8000",
		"-f", "s16le", "-acodec", "pcm_s16le",
		"pipe:1",
	)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[waveform] ffmpeg: %v (%s)\n", err, strings.TrimSpace(errBuf.String()))
		return nil
	}
	return peaksFromPCM(out, buckets)
}

// peaksFromPCM 16-bit little-endian mono PCM verisinden kova tepeleri üretir.
func peaksFromPCM(pcm []byte, buckets int) []float64 {
	const bytesPerSample = 2
	samples := len(pcm) / bytesPerSample
	if samples == 0 || buckets <= 0 {
		return nil
	}
	if buckets > samples {
		buckets = samples
	}
	peaks := make([]float64, buckets)
	per := float64(samples) / float64(buckets)
	maxPeak := 0.0
	for i := 0; i < buckets; i++ {
		start := int(float64(i) * per)
		end := int(float64(i+1) * per)
		if end > samples {
			end = samples
		}
		if end <= start {
			end = start + 1
		}
		peak := 0.0
		for j := start; j < end; j++ {
			v := int16(binary.LittleEndian.Uint16(pcm[j*bytesPerSample : j*bytesPerSample+bytesPerSample]))
			a := math.Abs(float64(v)) / 32768.0
			if a > peak {
				peak = a
			}
		}
		peaks[i] = peak
		if peak > maxPeak {
			maxPeak = peak
		}
	}
	if maxPeak <= 0 {
		return nil
	}
	// En yüksek tepeye göre normalize et; sessiz kayıtları da görünür kıl.
	for i := range peaks {
		peaks[i] = math.Round(peaks[i]/maxPeak*1000) / 1000
	}
	return peaks
}

// WaveformJSON dalga formunu JSON dizisine çevirir (DB'ye yazmak için).
func WaveformJSON(peaks []float64) (string, error) {
	if len(peaks) == 0 {
		return "[]", nil
	}
	var b []byte
	b = append(b, '[')
	for i, p := range peaks {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, []byte(fmt.Sprintf("%.3f", p))...)
	}
	b = append(b, ']')
	return string(b), nil
}
