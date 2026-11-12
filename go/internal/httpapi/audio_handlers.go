package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/aieditor"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
	"theblock/site/internal/transcribe"
	"theblock/site/internal/webui/templkit"
)

// maxAudioUploadBytes röportaj kaydı üst sınırı. Normalizasyon öncesi ham
// dosya private depolamaya akıtılır; belleğe tamamı alınmaz.
const maxAudioUploadBytes = 500 << 20

// handleAudioUpload private ses/video kaydını yükler, media_files ve
// transcription_jobs kayıtlarını oluşturur. İçeriğe hiçbir metin yazmaz.
func (s *Server) handleAudioUpload(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	c, ok := s.editableContent(w, r, id)
	if !ok {
		return
	}
	siteID := c.SiteID
	fail := func(msg string) { s.renderAudioPanel(w, r, siteID, id, "error", msg) }
	if strings.TrimSpace(s.asrAPIKey()) == "" {
		fail(s.asrMissingKeyMessage())
		return
	}
	// Sunucunun genel ReadTimeout değeri büyük dosyalar için kısa kalabilir;
	// yalnızca bu istek için okuma süresini uzat.
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(30 * time.Minute))
	r.Body = http.MaxBytesReader(w, r.Body, maxAudioUploadBytes+(8<<20))
	mr, err := r.MultipartReader()
	if err != nil {
		fail("Geçersiz yükleme isteği.")
		return
	}

	var savedPath, filename string
	var size int64
	var consent bool
	speakers := 0
	lang := "tr"
	cleanup := func() {
		if savedPath != "" {
			_ = platform.DeletePrivate(savedPath)
			savedPath = ""
		}
	}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			cleanup()
			fail("Yükleme okunamadı: " + err.Error())
			return
		}
		switch part.FormName() {
		case "ses":
			if part.FileName() == "" {
				_ = part.Close()
				continue
			}
			if savedPath != "" {
				_ = part.Close()
				cleanup()
				fail("Tek seferde yalnızca bir ses dosyası yükleyin.")
				return
			}
			filename = filepath.Base(part.FileName())
			if !audioUploadAllowed(filename) {
				_ = part.Close()
				cleanup()
				fail("Desteklenmeyen ses/video türü: " + filename + " (mp3, m4a, aac, wav, ogg, opus, flac, webm, mp4, mov).")
				return
			}
			savedPath, size, err = platform.SavePrivateAudio(part, filename)
			_ = part.Close()
			if err != nil {
				cleanup()
				fail("Ses dosyası kaydedilemedi: " + err.Error())
				return
			}
		case "onay":
			consent = readPartBool(part)
		case "konusmaci":
			speakers = parseSpeakerCount(readPartString(part))
		case "dil":
			lang = readPartString(part)
		}
		_ = part.Close()
	}
	if savedPath == "" {
		fail("Ses dosyası seçilmedi.")
		return
	}
	if !consent {
		cleanup()
		fail("Kayıt için kişilerin onayını teyit etmelisiniz.")
		return
	}
	if lang != "tr" && lang != "en" {
		lang = "tr"
	}

	ctx := r.Context()
	uid := userFromContext(ctx).ID
	mimeType := audioMime(filename)
	mediaID, err := repo.CreateMedia(ctx, s.pool, &repo.Media{
		SiteID: siteID, UserID: uid, Kind: "audio", Filename: filename,
		Path: savedPath, Mime: mimeType, SizeBytes: &size, Title: filename,
		Category: "ses", Source: "upload", Visibility: "private",
	})
	if err != nil {
		cleanup()
		s.logger.Error("ses medya kaydı", "error", err)
		fail("Ses kaydı veritabanına yazılamadı.")
		return
	}
	now := time.Now()
	_, err = repo.CreateTranscriptionJob(ctx, s.pool, &repo.TranscriptionJob{
		SiteID: siteID, MediaID: mediaID, ContentID: &id, UserID: &uid,
		Provider: s.asrProviderName(), Language: lang, SpeakersExpected: speakers, ConsentAt: &now,
	})
	if err != nil {
		_ = repo.DeleteMedia(ctx, s.pool, mediaID)
		cleanup()
		s.logger.Error("transkripsiyon işi", "error", err)
		fail("Transkripsiyon işi oluşturulamadı.")
		return
	}
	s.audit(r, "ses_yukle", filename)
	s.renderAudioPanel(w, r, siteID, id, "info", "Ses kaydı yüklendi; transkripsiyon kuyruğa alındı. Durum birkaç saniye içinde güncellenir.")
}

// handleAudioStatus panelin tamamını yeniden üretir; UI çalışan işleri yoklar.
func (s *Server) handleAudioStatus(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	c, ok := s.editableContent(w, r, id)
	if !ok {
		return
	}
	s.renderTemplPanel(w, r, templkit.SesPanelBody(s.audioPanelView(r, c.SiteID, id)))
}

// handleAudioRetry başarısız/iptal işi yeniden kuyruğa alır.
func (s *Server) handleAudioRetry(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	c, ok := s.editableContent(w, r, id)
	if !ok {
		return
	}
	mid := parseID(r.PathValue("media"))
	ca, err := repo.GetAudioForContentMedia(r.Context(), s.pool, id, mid)
	if err != nil {
		s.renderAudioPanel(w, r, c.SiteID, id, "error", "Ses kaydı bulunamadı.")
		return
	}
	ok, err = repo.RetryTranscriptionJob(r.Context(), s.pool, ca.Job.ID)
	if err != nil {
		s.renderAudioPanel(w, r, c.SiteID, id, "error", "İş yeniden kuyruğa alınamadı.")
		return
	}
	if !ok {
		s.renderAudioPanel(w, r, c.SiteID, id, "error", "İş zaten çalışıyor veya yeniden denenemez.")
		return
	}
	s.audit(r, "ses_yeniden", ca.Media.Filename)
	s.renderAudioPanel(w, r, c.SiteID, id, "info", "Transkripsiyon yeniden kuyruğa alındı.")
}

// handleAudioDelete işi iptal eder, kaydı ve private dosyayı siler.
func (s *Server) handleAudioDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	c, ok := s.editableContent(w, r, id)
	if !ok {
		return
	}
	mid := parseID(r.PathValue("media"))
	ctx := r.Context()
	ca, err := repo.GetAudioForContentMedia(ctx, s.pool, id, mid)
	if err != nil {
		s.renderAudioPanel(w, r, c.SiteID, id, "error", "Ses kaydı bulunamadı.")
		return
	}
	if ca.Job.Status == "queued" || ca.Job.Status == "running" {
		_, _ = repo.CancelTranscriptionJob(ctx, s.pool, ca.Job.ID)
	}
	if err := repo.DeleteMedia(ctx, s.pool, ca.Media.ID); err != nil {
		s.renderAudioPanel(w, r, c.SiteID, id, "error", "Ses kaydı silinemedi; içerikte kullanılıyor olabilir.")
		return
	}
	_ = platform.DeletePrivate(ca.Media.Path)
	s.audit(r, "ses_sil", ca.Media.Filename)
	s.renderAudioPanel(w, r, c.SiteID, id, "info", "Ses kaydı ve transkripti silindi.")
}

// handleAudioDraft transkriptten AI taslağı üretir. Sonuç yalnızca AI
// panelinde gösterilir; editör uygulamadan içeriğe yazılmaz.
func (s *Server) handleAudioDraft(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	mid := parseID(r.PathValue("media"))
	ca, err := repo.GetAudioForContentMedia(r.Context(), s.pool, id, mid)
	if err != nil || ca.Transcript == nil {
		s.renderAIEditorError(w, r, "Bu ses kaydı için transkript bulunamadı.")
		return
	}
	text := audioTranscriptForDraft(ca.Transcript)
	if text == "" {
		s.renderAIEditorError(w, r, "Transkript boş; taslak oluşturulamaz.")
		return
	}
	act, ok := aieditor.ActionByID("audio_draft")
	if !ok {
		s.renderAIEditorError(w, r, "Ses taslağı eylemi tanımlı değil.")
		return
	}
	ctx := r.Context()
	jobID, err := s.aiEditor.Enqueue(ctx, aieditor.Request{
		ContentID: id,
		User:      userFromContext(ctx),
		Action:    act.ID,
		Selection: text,
		Context: map[string]any{
			"transcript_id": ca.Transcript.ID,
			"media_id":      ca.Media.ID,
			"language":      ca.Transcript.Language,
			"duration_ms":   ca.Transcript.DurationMS,
			"speakers":      ca.Transcript.Speakers,
			"provider":      ca.Transcript.Provider,
		},
	})
	if err != nil {
		s.logger.Warn("ses taslağı kuyruğa alınamadı", "content", id, "media", mid, "error", err)
		s.renderAIEditorError(w, r, aiErrorText(err))
		return
	}
	s.renderAIEditorPending(w, r, id, act, jobID, "Kuyrukta")
}

// handlePrivateAudioStream private ses kaydını Range destekli akıtır.
// Public /media/ ağacından bilinçli olarak ayrıdır.
func (s *Server) handlePrivateAudioStream(w http.ResponseWriter, r *http.Request) {
	mid := parseID(r.PathValue("id"))
	if mid <= 0 {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()
	m, err := repo.GetMedia(ctx, s.pool, mid)
	if err != nil || m.Visibility != "private" {
		http.NotFound(w, r)
		return
	}
	job, err := repo.GetLatestTranscriptionJobForMedia(ctx, s.pool, mid)
	if err != nil || job.ContentID == nil {
		http.NotFound(w, r)
		return
	}
	c, err := repo.GetContent(ctx, s.pool, *job.ContentID)
	if err != nil || m.SiteID != c.SiteID || !canEditContent(userFromContext(ctx), c) {
		http.Error(w, "yetkisiz", http.StatusForbidden)
		return
	}
	f, err := platform.OpenPrivate(m.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", m.Mime)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, m.Filename, st.ModTime(), f)
}

// audioPanelView panelin veri modelini üretir.
func (s *Server) audioPanelView(r *http.Request, siteID, contentID int64) templkit.IcerikFormView {
	cid := strconv.FormatInt(contentID, 10)
	v := templkit.IcerikFormView{
		ContentID:    cid,
		HasContent:   true,
		SesUploadURL: "/admin/icerikler/" + cid + "/ses",
	}
	audios, err := repo.ListAudioForContent(r.Context(), s.pool, siteID, contentID)
	if err != nil {
		s.logger.Warn("ses listesi", "content", contentID, "error", err)
		return v
	}
	v.Audios = s.icerikAudioViews(contentID, audios)
	for _, a := range v.Audios {
		if a.IsActive {
			v.SesPollURL = "/admin/icerikler/" + cid + "/ses/durum"
			break
		}
	}
	return v
}

// renderAudioPanel panel gövdesini flash mesajıyla birlikte yazar.
func (s *Server) renderAudioPanel(w http.ResponseWriter, r *http.Request, siteID, contentID int64, kind, text string) {
	v := s.audioPanelView(r, siteID, contentID)
	v.SesFlashKind, v.SesFlashText = kind, text
	s.renderTemplPanel(w, r, templkit.SesPanelBody(v))
}

// icerikAudioViews repo kayıtlarını templ görünümüne çevirir.
func (s *Server) icerikAudioViews(contentID int64, audios []repo.ContentAudio) []templkit.IcerikAudio {
	cid := strconv.FormatInt(contentID, 10)
	out := make([]templkit.IcerikAudio, 0, len(audios))
	for _, a := range audios {
		mid := strconv.FormatInt(a.Media.ID, 10)
		item := templkit.IcerikAudio{
			MediaID:     mid,
			Filename:    a.Media.Filename,
			StreamURL:   "/admin/ses/" + mid + "/stream",
			Status:      a.Job.Status,
			StatusLabel: audioStatusTR(a.Job.Status),
			Progress:    a.Job.Progress,
			CreatedAt:   a.Media.CreatedAt.Format("02.01.2006 15:04"),
			DraftURL:    "/admin/icerikler/" + cid + "/ses/" + mid + "/taslak",
			RetryURL:    "/admin/icerikler/" + cid + "/ses/" + mid + "/yeniden",
			DeleteURL:   "/admin/icerikler/" + cid + "/ses/" + mid + "/sil",
			CanRetry:    a.Job.Status == "failed" || a.Job.Status == "cancelled",
			IsActive:    a.Job.Status == "queued" || a.Job.Status == "running",
			Error:       a.Job.Error,
		}
		if a.Job.Status == "completed" {
			item.Progress = 100
		}
		if a.Media.SizeBytes != nil {
			item.Size = humanBytes(*a.Media.SizeBytes)
		}
		durationMS := a.Job.DurationMS
		if a.Transcript != nil {
			if a.Transcript.DurationMS > 0 {
				durationMS = a.Transcript.DurationMS
			}
			item.TranscriptText = strings.TrimSpace(a.Transcript.Text)
			item.Speakers = strings.Join(a.Transcript.Speakers, ", ")
			item.Waveform = a.Transcript.Waveform
			item.WaveformJSON = waveformJSON(a.Transcript.Waveform)
			for _, seg := range a.Transcript.Segments {
				text := strings.TrimSpace(seg.Text)
				if text == "" {
					continue
				}
				item.Segments = append(item.Segments, templkit.IcerikAudioSegment{
					StartMS:    seg.StartMS,
					StartLabel: audioTimeLabel(seg.StartMS),
					Speaker:    strings.TrimSpace(seg.Speaker),
					Text:       text,
					Words:      audioWords(seg.Words),
				})
			}
			item.CanDraft = a.Job.Status == "completed" && item.TranscriptText != ""
			// Altyazı yalnız zaman damgalı bölüm varsa üretilebilir.
			if len(a.Transcript.Segments) > 0 {
				base := "/admin/icerikler/" + cid + "/ses/" + mid + "/altyazi/"
				item.SrtURL, item.VttURL = base+"srt", base+"vtt"
			}
		}
		item.Duration = audioTimeLabel(durationMS)
		out = append(out, item)
	}
	return out
}

func (s *Server) handleTranscriptSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		s.renderTemplPanel(w, r, templkit.SesAramaSonuclari(nil, ""))
		return
	}
	siteID := s.curSiteID(r)
	hits, err := repo.SearchAudioTranscripts(ctx, s.pool, siteID, q, 30)
	if err != nil {
		s.logger.Warn("transkript arama", "q", q, "error", err)
		s.renderTemplPanel(w, r, templkit.SesAramaSonuclari(nil, q))
		return
	}
	out := make([]templkit.IcerikSesArama, 0, len(hits))
	for _, h := range hits {
		if h.ContentID == nil {
			continue
		}
		cid := strconv.FormatInt(*h.ContentID, 10)
		title := strings.TrimSpace(h.ContentTitle)
		if title == "" {
			title = "İçerik #" + cid
		}
		out = append(out, templkit.IcerikSesArama{
			URL:          "/admin/icerikler/" + cid + "#sesPanel",
			ContentTitle: title,
			Snippet:      h.Snippet,
			Duration:     audioTimeLabel(h.DurationMS),
		})
	}
	s.renderTemplPanel(w, r, templkit.SesAramaSonuclari(out, q))
}

// asrAPIKey seçili sağlayıcının anahtarını döndürür.
func (s *Server) asrAPIKey() string {
	if transcribe.ProviderName(s.cfg.TranscribeProvider) == "deepgram" {
		return s.cfg.DeepgramKey
	}
	return s.cfg.AssemblyAIKey
}

// asrProviderName panelde gösterilecek sağlayıcı adı.
func (s *Server) asrProviderName() string {
	return transcribe.ProviderName(s.cfg.TranscribeProvider)
}

// asrMissingKeyMessage anahtar yokken editöre gösterilecek açıklama.
func (s *Server) asrMissingKeyMessage() string {
	if s.asrProviderName() == "deepgram" {
		return "Transkripsiyon sağlayıcısı yapılandırılmadı. DEEPGRAM_API_KEY ekleyip sunucuyu yeniden başlatın."
	}
	return "Transkripsiyon sağlayıcısı yapılandırılmadı. ASSEMBLYAI_API_KEY ekleyip sunucuyu yeniden başlatın."
}

func (s *Server) handleAudioSubtitle(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	format := strings.ToLower(strings.TrimSpace(r.PathValue("bicim")))
	if format != "srt" && format != "vtt" {
		http.NotFound(w, r)
		return
	}
	mid := parseID(r.PathValue("media"))
	ca, err := repo.GetAudioForContentMedia(r.Context(), s.pool, id, mid)
	if err != nil || ca.Transcript == nil {
		http.Error(w, "Bu ses kaydı için transkript bulunamadı.", http.StatusNotFound)
		return
	}
	cues := transcribe.BuildCues(transcribeSegments(ca.Transcript.Segments))
	if len(cues) == 0 {
		http.Error(w, "Transkript altyazıya çevrilemedi: zaman damgalı bölüm yok.",
			http.StatusUnprocessableEntity)
		return
	}

	body, ctype := transcribe.SRT(cues), "application/x-subrip; charset=utf-8"
	if format == "vtt" {
		body, ctype = transcribe.WebVTT(cues), "text/vtt; charset=utf-8"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+subtitleFilename(ca.Media.Filename, format)+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.WriteString(w, body)
}

// transcribeSegments repo segmentlerini altyazı üreticisinin beklediği
// alan adı bağımsız tipe çevirir.
func transcribeSegments(segs []repo.AudioSegment) []transcribe.Segment {
	if len(segs) == 0 {
		return nil
	}
	out := make([]transcribe.Segment, 0, len(segs))
	for _, seg := range segs {
		var words []transcribe.Word
		if len(seg.Words) > 0 {
			words = make([]transcribe.Word, 0, len(seg.Words))
			for _, w := range seg.Words {
				words = append(words, transcribe.Word{
					StartMS: w.StartMS, EndMS: w.EndMS,
					Text: w.Text, Confidence: w.Confidence,
				})
			}
		}
		out = append(out, transcribe.Segment{
			StartMS: seg.StartMS, EndMS: seg.EndMS, Speaker: seg.Speaker,
			Text: seg.Text, Confidence: seg.Confidence, Words: words,
		})
	}
	return out
}

func subtitleFilename(mediaName, ext string) string {
	base := strings.TrimSuffix(mediaName, filepath.Ext(mediaName))
	var b strings.Builder
	for _, r := range base {
		if tr, ok := turkishASCII[r]; ok {
			b.WriteString(tr)
			continue
		}
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	name := strings.Trim(b.String(), "-.")
	if name == "" {
		name = "altyazi"
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name + "." + ext
}

// turkishASCII dosya adındaki Türkçe harflerin ASCII karşılıkları.
var turkishASCII = map[rune]string{
	'ç': "c", 'Ç': "C", 'ğ': "g", 'Ğ': "G", 'ı': "i", 'İ': "I",
	'ö': "o", 'Ö': "O", 'ş': "s", 'Ş': "S", 'ü': "u", 'Ü': "U",
}

// audioWords repo kelimelerini şablon tipine çevirir. Boş metinli kelimeler
// atılır; hiç kelime yoksa nil döner ve şablon segmenti düz metin basar.
func audioWords(words []repo.AudioWord) []templkit.IcerikAudioWord {
	if len(words) == 0 {
		return nil
	}
	out := make([]templkit.IcerikAudioWord, 0, len(words))
	for _, w := range words {
		text := strings.TrimSpace(w.Text)
		if text == "" {
			continue
		}
		out = append(out, templkit.IcerikAudioWord{
			StartMS: w.StartMS, EndMS: w.EndMS, Text: text,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// waveformJSON dalga formunu HTML data-özniteliği için JSON'a çevirir.
// Boşsa boş dize döner (oynatıcı düz çubuğa düşer).
func waveformJSON(peaks []float64) string {
	if len(peaks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, p := range peaks {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(p, 'f', 3, 64))
	}
	b.WriteByte(']')
	return b.String()
}

// audioTranscriptForDraft transkripti AI'ya konuşmacı etiketleriyle verir.
// Segment yoksa düz metne düşer; böylece eski kayıtlar da çalışır.
func audioTranscriptForDraft(t *repo.AudioTranscript) string {
	if t == nil {
		return ""
	}
	var b strings.Builder
	for _, seg := range t.Segments {
		line := strings.TrimSpace(seg.Text)
		if line == "" {
			continue
		}
		if sp := strings.TrimSpace(seg.Speaker); sp != "" {
			b.WriteString("Konuşmacı ")
			b.WriteString(sp)
			b.WriteString(": ")
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if b.Len() == 0 {
		return strings.TrimSpace(t.Text)
	}
	return strings.TrimSpace(b.String())
}

// audioStatusTR durum etiketleri.
func audioStatusTR(status string) string {
	switch status {
	case "queued":
		return "Sırada"
	case "running":
		return "Metne çevriliyor"
	case "completed":
		return "Tamamlandı"
	case "failed":
		return "Başarısız"
	case "cancelled":
		return "İptal edildi"
	default:
		return status
	}
}

// audioTimeLabel milisaniyeyi mm:ss / hh:mm:ss biçimine çevirir.
func audioTimeLabel(ms int64) string {
	if ms <= 0 {
		return ""
	}
	sec := ms / 1000
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// audioUploadAllowed röportaj kaydı için kabul edilen uzantılar.
func audioUploadAllowed(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp3", ".m4a", ".aac", ".wav", ".ogg", ".oga", ".opus", ".weba",
		".webm", ".flac", ".mp4", ".mov", ".m4v":
		return true
	}
	return false
}

// audioMime tarayıcı oynatıcısı ve akış için açık MIME eşlemesi.
func audioMime(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp3":
		return "audio/mpeg"
	case ".m4a", ".m4v":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".wav":
		return "audio/wav"
	case ".ogg", ".oga", ".opus":
		return "audio/ogg"
	case ".weba", ".webm":
		return "audio/webm"
	case ".flac":
		return "audio/flac"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	}
	return "application/octet-stream"
}

func readPartString(part io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(part, 64<<10))
	return strings.TrimSpace(string(b))
}

func readPartBool(part io.Reader) bool {
	switch strings.ToLower(readPartString(part)) {
	case "1", "on", "true", "evet", "yes":
		return true
	}
	return false
}

func parseSpeakerCount(v string) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 {
		return 0
	}
	if n > 10 {
		return 10
	}
	return n
}
