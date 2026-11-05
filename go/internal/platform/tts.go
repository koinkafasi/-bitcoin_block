// TTS: espeak-ng ile konuşma sentezi → WAV; ffmpeg ile MP3'e çevrilir.
// (Günlük özet podcast'i — spec 06 K-E)
package platform

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNoTTS espeak-ng bulunamadığında döner.
var ErrNoTTS = errors.New("espeak-ng yok; ses üretilemedi")

// espeakVoice dile göre espeak-ng ses adı seçer.
func espeakVoice(locale string) string {
	switch strings.ToLower(locale) {
	case "en", "en-us", "en-gb":
		return "en"
	case "de":
		return "de"
	case "fr":
		return "fr"
	case "es":
		return "es"
	case "ar":
		return "ar"
	case "fa", "ur":
		return "ur"
	default:
		return "tr"
	}
}

// SynthesizeSpeech metni konuşmaya çevirir: önce WAV (espeak-ng), varsa MP3 (ffmpeg).
// outDir içine yazılır; üretilen dosyanın göreli yolunu döndürür.
func SynthesizeSpeech(text, locale, outDir, name string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("metin boş")
	}
	if len([]rune(text)) > 6000 {
		text = string([]rune(text)[:6000])
	}
	bin, err := exec.LookPath("espeak-ng")
	if err != nil {
		return "", ErrNoTTS
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	wav := filepath.Join(outDir, name+".wav")
	cmd := exec.Command(bin, "-v", espeakVoice(locale), "-s", "155", "-w", wav, text)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	if ff, err := exec.LookPath("ffmpeg"); err == nil {
		mp3 := filepath.Join(outDir, name+".mp3")
		c := exec.Command(ff, "-y", "-loglevel", "error", "-i", wav, "-codec:a", "libmp3lame", "-q:a", "4", mp3)
		if c.Run() == nil {
			_ = os.Remove(wav)
			return filepath.Base(mp3), nil
		}
	}
	return filepath.Base(wav), nil
}

// AudioContentType uzantıdan MIME döndürür.
func AudioContentType(path string) string {
	if strings.HasSuffix(path, ".mp3") {
		return "audio/mpeg"
	}
	return "audio/wav"
}
