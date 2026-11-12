package transcribe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// Service transkripsiyon kuyruğunu yürütür.
type Service struct {
	pool     *pgxpool.Pool
	provider Provider
	ff       *FFmpeg
	logger   *slog.Logger
	workRoot string
}

// NewService servisi kurar. provider nil olabilir; bu durumda işler açık bir
// hata ile başarısız işaretlenir.
func NewService(pool *pgxpool.Pool, provider Provider, ff *FFmpeg, logger *slog.Logger) *Service {
	if ff == nil {
		ff = NewFFmpeg("", "")
	}
	return &Service{pool: pool, provider: provider, ff: ff, logger: logger, workRoot: os.TempDir()}
}

func StartWorker(ctx context.Context, pool *pgxpool.Pool, provider Provider, ff *FFmpeg, logger *slog.Logger, interval time.Duration) {
	if provider == nil {
		logger.Warn("transkripsiyon worker kapalı: ASR sağlayıcı tanımlı değil")
		return
	}
	if _, dynamic := provider.(*Dynamic); !dynamic {
		if ds, ok := provider.(interface{ Resolve(context.Context) Config }); ok && ds.Resolve(ctx).Empty() {
			logger.Warn("transkripsiyon worker kapalı: ASR anahtarı tanımlı değil")
			return
		}
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	svc := NewService(pool, provider, ff, logger)
	if n, err := repo.RecoverStaleTranscriptionJobs(ctx, pool, 30*time.Minute); err != nil {
		logger.Warn("transkripsiyon stale temizliği", "error", err)
	} else if n > 0 {
		logger.Info("transkripsiyon stale işleri kuyruğa aldı", "adet", n)
	}
	concurrency := envInt("TRANSCRIBE_WORKER_CONCURRENCY", 1)
	if concurrency <= 0 {
		concurrency = 1
	}
	logger.Info("transkripsiyon worker başladı", "concurrency", concurrency, "interval", interval)
	for i := 0; i < concurrency; i++ {
		go svc.workerLoop(ctx, interval)
	}
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := repo.RecoverStaleTranscriptionJobs(ctx, pool, 30*time.Minute); err == nil && n > 0 {
					logger.Info("transkripsiyon stale işleri kuyruğa aldı", "adet", n)
				}
			}
		}
	}()
}

func (s *Service) workerLoop(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for {
				ok, err := s.ProcessNext(ctx)
				if err != nil {
					s.logger.Warn("transkripsiyon iş hatası", "error", err)
				}
				if !ok {
					break
				}
			}
		}
	}
}

// ProcessNext kuyruktaki bir işi sahiplenip yürütür.
func (s *Service) ProcessNext(ctx context.Context) (bool, error) {
	job, ok, err := repo.ClaimTranscriptionJob(ctx, s.pool)
	if err != nil || !ok {
		return false, err
	}
	if err := s.ProcessJob(ctx, job.ID); err != nil {
		return true, err
	}
	return true, nil
}

// ProcessJob tek işi yürütür. Aynı iş için idempotenttir; terminal iş
// yeniden çalıştırılmaz.
func (s *Service) ProcessJob(ctx context.Context, jobID int64) error {
	job, err := repo.GetTranscriptionJob(ctx, s.pool, jobID)
	if err != nil {
		return err
	}
	switch job.Status {
	case "completed", "failed", "cancelled":
		return nil
	case "queued":
		ok, err := repo.StartTranscriptionJob(ctx, s.pool, jobID)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		job.Status = "running"
	case "running":
		// worker claim etti veya doğrudan çalıştırıldı.
	default:
		return fmt.Errorf("transkripsiyon işi beklenmeyen durumda: %s", job.Status)
	}

	if s.provider == nil {
		return s.fail(job, "transkripsiyon sağlayıcısı yapılandırılmadı (ASR anahtarı eksik)")
	}
	media, err := repo.GetMedia(ctx, s.pool, job.MediaID)
	if err != nil {
		return s.fail(job, "ses kaydı bulunamadı")
	}
	if media.Visibility != "private" {
		return s.fail(job, "ses kaydı private görünürlükte değil; güvenlik için işlem durduruldu")
	}
	srcPath, err := platform.PrivateFilePath(media.Path)
	if err != nil {
		return s.fail(job, "ses dosyası yolu geçersiz")
	}
	if _, err := os.Stat(srcPath); err != nil {
		return s.fail(job, "ses dosyası diskte bulunamadı")
	}

	workDir, err := os.MkdirTemp(s.workRoot, fmt.Sprintf("theblock-asr-%d-", job.ID))
	if err != nil {
		return s.fail(job, "geçici çalışma dizini oluşturulamadı")
	}
	defer func() { _ = os.RemoveAll(workDir) }()
	normalized := filepath.Join(workDir, "audio.mp3")

	if err := s.ff.Normalize(ctx, srcPath, normalized); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return s.fail(job, "ses normalize edilemedi: "+err.Error())
	}
	info, err := s.ff.Probe(ctx, normalized)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return s.fail(job, "ses süresi okunamadı: "+err.Error())
	}
	if info.DurationMS > 0 {
		_ = repo.UpdateTranscriptionJobProgress(ctx, s.pool, job.ID, 5, "", info.DurationMS)
	}

	f, err := os.Open(normalized)
	if err != nil {
		return s.fail(job, "normalize edilmiş ses açılamadı")
	}
	defer f.Close()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var mu sync.Mutex
	lastPct, lastUpdate := 0, time.Time{}
	progress := func(p int, providerJobID string) {
		if p < 1 {
			p = 1
		}
		if p > 99 {
			p = 99
		}
		mu.Lock()
		now := time.Now()
		shouldWrite := p != lastPct || now.Sub(lastUpdate) >= 2*time.Second
		if shouldWrite {
			lastPct, lastUpdate = p, now
		}
		mu.Unlock()
		cur, gerr := repo.GetTranscriptionJob(runCtx, s.pool, job.ID)
		if gerr == nil && cur.Status == "cancelled" {
			cancel()
			return
		}
		if shouldWrite {
			_ = repo.UpdateTranscriptionJobProgress(runCtx, s.pool, job.ID, p, providerJobID, info.DurationMS)
		}
	}

	res, err := s.provider.Transcribe(runCtx, Request{
		Audio:            f,
		Filename:         "audio.mp3",
		Language:         job.Language,
		SpeakersExpected: job.SpeakersExpected,
		Model:            job.Model,
	}, progress)
	if err != nil {
		if ctx.Err() != nil {
			// Süreç kapanıyor; işi failed yapma, stale kurtarma kuyruğa alsın.
			return ctx.Err()
		}
		cur, gerr := repo.GetTranscriptionJob(context.Background(), s.pool, job.ID)
		if gerr == nil && cur.Status == "cancelled" {
			return nil
		}
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return s.fail(job, err.Error())
	}

	durationMS := res.DurationMS
	if durationMS <= 0 {
		durationMS = info.DurationMS
	}
	text := strings.TrimSpace(res.Text)
	if text == "" && len(res.Segments) > 0 {
		parts := make([]string, 0, len(res.Segments))
		for _, seg := range res.Segments {
			if t := strings.TrimSpace(seg.Text); t != "" {
				parts = append(parts, t)
			}
		}
		text = strings.Join(parts, " ")
	}
	if text == "" {
		return s.fail(job, "sağlayıcı boş transkript döndürdü")
	}
	segs := make([]repo.AudioSegment, 0, len(res.Segments))
	for _, seg := range res.Segments {
		var words []repo.AudioWord
		if len(seg.Words) > 0 {
			words = make([]repo.AudioWord, 0, len(seg.Words))
			for _, w := range seg.Words {
				words = append(words, repo.AudioWord{
					StartMS: w.StartMS, EndMS: w.EndMS,
					Text: w.Text, Confidence: w.Confidence,
				})
			}
		}
		segs = append(segs, repo.AudioSegment{
			StartMS: seg.StartMS, EndMS: seg.EndMS, Speaker: seg.Speaker,
			Text: strings.TrimSpace(seg.Text), Confidence: seg.Confidence,
			Words: words,
		})
	}
	// Dalga formu: normalize edilmiş sesten tepe noktaları (SoundCloud benzeri
	// oynatıcı). Üretilemezse boş kalır; oynatıcı düz çubuğa düşer.
	waveform := s.ff.Waveform(ctx, normalized, WaveformBucketCount)
	if _, err := repo.UpsertAudioTranscript(ctx, s.pool, &repo.AudioTranscript{
		SiteID: job.SiteID, MediaID: job.MediaID, ContentID: job.ContentID, JobID: &job.ID,
		Language: res.Language, Provider: res.Provider, Model: res.Model,
		DurationMS: durationMS, Text: text, Segments: segs, Speakers: res.Speakers,
		Waveform: waveform,
	}); err != nil {
		return s.fail(job, "transkript kaydedilemedi: "+err.Error())
	}
	if err := repo.FinishTranscriptionJob(ctx, s.pool, job.ID, "completed", "",
		res.Provider, res.Model, durationMS); err != nil {
		return err
	}
	return nil
}

func (s *Service) fail(job repo.TranscriptionJob, msg string) error {
	fctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	provider := ProviderNameOf(fctx, s.provider)
	if provider == "" {
		provider = "assemblyai"
	}
	_ = repo.FinishTranscriptionJob(fctx, s.pool, job.ID, "failed", msg, provider, job.Model, 0)
	return errors.New(msg)
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
