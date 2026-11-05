package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/transcribe"
	"theblock/site/internal/webui/templkit"
)

// ASR ayarları sayfası (Faz 125).
//
// Amaç: API anahtarını/sağlayıcıyı .env'i düzenleyip servisi yeniden
// başlatmadan değiştirebilmek. Değerler settings/asr altında saklanır ve
// .env yalnız varsayılan/başlangıç değeri sağlar.

// asrStore sunucunun ASR ayar deposudur (dinamik sağlayıcıyla aynı kaynak).
func (s *Server) asrStore() *transcribe.SettingsStore {
	if s.asrSettings == nil {
		s.asrSettings = transcribe.NewSettingsStore(s.pool, transcribe.DefaultsFromValues(
			s.cfg.TranscribeProvider,
			s.cfg.AssemblyAIKey, s.cfg.AssemblyAIBase, s.cfg.AssemblyAIModel,
			s.cfg.DeepgramKey, s.cfg.DeepgramBase, s.cfg.DeepgramModel,
		))
	}
	return s.asrSettings
}

// handleASRSettingsGet ayar sayfasını gösterir.
func (s *Server) handleASRSettingsGet(w http.ResponseWriter, r *http.Request) {
	s.renderASRSettings(w, r, "")
}

// handleASRSettingsPost ayarları kaydeder ve sayfayı tazeler.
func (s *Server) handleASRSettingsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderASRSettings(w, r, "Form okunamadı.")
		return
	}
	ctx := r.Context()
	store := s.asrStore()

	// "Anahtarı temizle"
	if r.FormValue("clear_key") != "" {
		if err := store.ClearAPIKey(ctx); err != nil {
			s.renderASRSettings(w, r, "Anahtar temizlenemedi: "+err.Error())
			return
		}
		s.audit(r, "asr_ayar", "anahtar temizlendi")
		s.invalidateContentCache()
		s.renderASRSettings(w, r, "Panelde kayıtlı API anahtarı silindi; varsa .env değeri kullanılacak.")
		return
	}

	cfg := transcribe.Config{
		Provider: strings.TrimSpace(r.FormValue("provider")),
		APIKey:   strings.TrimSpace(r.FormValue("api_key")),
		BaseURL:  strings.TrimSpace(r.FormValue("base_url")),
		Model:    strings.TrimSpace(r.FormValue("model")),
	}
	if err := store.Save(ctx, cfg); err != nil {
		s.renderASRSettings(w, r, "Ayarlar kaydedilemedi: "+err.Error())
		return
	}
	s.invalidateContentCache()
	s.audit(r, "asr_ayar", transcribe.ProviderName(cfg.Provider)+" kaydedildi")

	// "Bağlantıyı sına"
	if r.FormValue("test") != "" {
		msg := s.testASRConnection(ctx)
		s.renderASRSettings(w, r, msg)
		return
	}
	s.renderASRSettings(w, r, "Ayarlar kaydedildi. Yeni anahtar/sağlayıcı bir sonraki işte kullanılacak.")
}

// testASRConnection etkin yapılandırmayla sağlayıcıya küçük bir istek atar.
// Tam transkripsiyon yapmaz (maliyet/timeout); yalnız kimlik doğrulamayı
// sınar.
func (s *Server) testASRConnection(ctx context.Context) string {
	cfg := s.asrStore().Load(ctx)
	if cfg.Empty() {
		return "Bağlantı sınaması: API anahtarı yok."
	}
	tctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if cfg.Provider == "deepgram" {
		req, err := http.NewRequestWithContext(tctx, http.MethodGet, strings.TrimRight(baseOr(cfg.BaseURL, "https://api.deepgram.com"), "/")+"/v1/auth/token", nil)
		if err != nil {
			return "Bağlantı sınaması: istek kurulamadı."
		}
		req.Header.Set("Authorization", "Token "+cfg.APIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "Bağlantı sınaması başarısız: " + err.Error()
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized && false {
			return "Deepgram bağlantısı başarılı."
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return "Deepgram: anahtar geçersiz (HTTP " + strconv.Itoa(resp.StatusCode) + ")."
		}
		return "Deepgram yanıtı: HTTP " + strconv.Itoa(resp.StatusCode) + " (anahtar geçerli olabilir)."
	}

	// AssemblyAI: /v2/transcript?limit=1 kimlik doğrulamayı sınar.
	base := strings.TrimRight(baseOr(cfg.BaseURL, "https://api.assemblyai.com"), "/")
	req, err := http.NewRequestWithContext(tctx, http.MethodGet, base+"/v2/transcript?limit=1", nil)
	if err != nil {
		return "Bağlantı sınaması: istek kurulamadı."
	}
	req.Header.Set("Authorization", cfg.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "Bağlantı sınaması başarısız: " + err.Error()
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return "AssemblyAI bağlantısı başarılı; anahtar geçerli."
	case http.StatusUnauthorized, http.StatusForbidden:
		return "AssemblyAI: anahtar geçersiz (HTTP " + strconv.Itoa(resp.StatusCode) + ")."
	}
	return "AssemblyAI yanıtı: HTTP " + strconv.Itoa(resp.StatusCode) + "."
}

func baseOr(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// renderASRSettings sayfayı (HTMX'te paneli) yazar.
func (s *Server) renderASRSettings(w http.ResponseWriter, r *http.Request, flash string) {
	ctx := r.Context()
	store := s.asrStore()
	cur := store.Load(ctx)
	def := store.Defaults()

	v := templkit.SesAyarlariView{
		Provider:     cur.Provider,
		Model:        cur.Model,
		BaseURL:      cur.BaseURL,
		KeySet:       false,
		EnvKeySet:    !def.Empty(),
		EnvProvider:  def.Provider,
		Ready:        !cur.Empty(),
		WorkerNote:   workerNote(cur),
		SpellEnabled: strings.TrimSpace(s.cfg.HunspellBin) != "",
		SpellBin:     s.cfg.HunspellBin,
		SpellDict:    s.cfg.HunspellDict,
	}
	if cur.Provider == "deepgram" {
		v.TestURL = "https://console.deepgram.com/"
	} else {
		v.TestURL = "https://www.assemblyai.com/dashboard"
	}
	// Panelde kayıtlı anahtar var mı? env'den farklıysa "panel" sayılır.
	if raw, err := repo.GetSetting(ctx, s.pool, "asr", "api_key"); err == nil && len(raw) > 0 {
		if k := strings.TrimSpace(string(raw)); k != "" {
			v.KeySet = true
			v.KeyHint = maskKey(k)
		}
	}
	if v.Model == "" {
		v.Model = defaultModelFor(cur.Provider)
	}
	v.EffectiveProvider = cur.Provider
	v.EffectiveModel = cur.Model
	if v.EffectiveModel == "" {
		v.EffectiveModel = defaultModelFor(cur.Provider)
	}

	// İstatistik + son işler.
	var total, completed, failed int
	var totalMS int64
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM transcription_jobs`).Scan(&total)
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM transcription_jobs WHERE status='completed'`).Scan(&completed)
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM transcription_jobs WHERE status='failed'`).Scan(&failed)
	_ = s.pool.QueryRow(ctx, `SELECT COALESCE(sum(duration_ms),0) FROM transcription_jobs WHERE status='completed'`).Scan(&totalMS)
	v.TotalJobs = strconv.Itoa(total)
	v.CompletedJobs = strconv.Itoa(completed)
	v.FailedJobs = strconv.Itoa(failed)
	v.TotalAudio = audioTimeLabel(totalMS)

	if rows, err := s.pool.Query(ctx, `
		SELECT j.id, COALESCE(m.filename,''), j.status, j.provider, j.created_at, j.error
		FROM transcription_jobs j
		LEFT JOIN media_files m ON m.id = j.media_id
		ORDER BY j.id DESC LIMIT 8`); err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var media, status, provider, errMsg string
			var created time.Time
			if err := rows.Scan(&id, &media, &status, &provider, &created, &errMsg); err != nil {
				continue
			}
			v.Jobs = append(v.Jobs, templkit.SesAyarJob{
				ID: strconv.FormatInt(id, 10), Media: media, Status: status,
				StatusTR: audioStatusTR(status), Provider: provider,
				When:  created.Local().Format("02.01 15:04"),
				Error: truncateRunes(errMsg, 70),
			})
		}
	}

	v.FlashKind, v.FlashText = "info", flash
	if strings.Contains(strings.ToLower(flash), "başarısız") || strings.Contains(strings.ToLower(flash), "geçersiz") {
		v.FlashKind = "error"
	}

	if IsHTMX(r) {
		s.renderTemplPanel(w, r, templkit.SesAyarlariPanel(v))
		return
	}
	d := s.baseView("tr", "/admin/ses-ayarlari", "Ses Ayarları")
	d.User = ptr(userFromContext(ctx))
	s.renderTemplPage(w, r, d, templkit.SesAyarlariPanel(v))
}

// workerNote etkin yapılandırmanın işleri alıp alamayacağını açıklar.
func workerNote(c transcribe.Config) string {
	if c.Empty() {
		return "anahtar yok — yükleme yapılırsa iş açık hatayla başarısız olur."
	}
	return "worker her işte ayarları yeniden okur; değişiklik anında geçerli."
}

// defaultModelFor sağlayıcının önerilen model adı.
func defaultModelFor(provider string) string {
	if provider == "deepgram" {
		return "nova-3"
	}
	return "best"
}

// maskKey anahtarın yalnız ilk 6 karakterini gösterir.
func maskKey(k string) string {
	r := []rune(strings.TrimSpace(k))
	if len(r) <= 6 {
		return "••••"
	}
	return string(r[:6]) + "…"
}
