package httpapi

import (
	"net/http"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
	"theblock/site/internal/tts"
)

// handleContentTTS, yazının metnini OpenAI TTS ile MP3'e çevirip içeriğe bağlar.
// Yönetimden "Sesli haber oluştur" düğmesiyle tetiklenir.
func (s *Server) handleContentTTS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	c, ok := s.editableContent(w, r, id)
	if !ok {
		return
	}
	if !s.tts.Enabled() {
		s.ttsResult(w, r, c, "error", "TTS kapalı: deploy/.env içine OPENAI_API_KEY ekleyin.")
		return
	}

	text := tts.HTMLToText(string(s.blocksToHTML(c.Detail, nil)))
	if len([]rune(text)) < 40 {
		s.ttsResult(w, r, c, "error", "Seslendirilecek metin çok kısa.")
		return
	}

	chunks := tts.SplitText(text, 3500)
	var mp3 []byte
	for _, ch := range chunks {
		part, err := s.tts.Speech(ctx, ch)
		if err != nil {
			s.logger.Error("tts üretilemedi", "content", c.ID, "error", err)
			s.ttsResult(w, r, c, "error", "Ses üretilemedi: "+err.Error())
			return
		}
		mp3 = append(mp3, part...)
	}
	if len(mp3) == 0 {
		s.ttsResult(w, r, c, "error", "Ses üretilemedi (boş yanıt).")
		return
	}

	stored, err := platform.SaveMedia(mp3, "sesli-haber-"+itoa(int(c.ID))+".mp3")
	if err != nil {
		s.ttsResult(w, r, c, "error", "Ses dosyası kaydedilemedi: "+err.Error())
		return
	}
	u := userFromContext(ctx)
	sz := int64(len(mp3))
	_, err = repo.CreateMedia(ctx, s.pool, &repo.Media{
		SiteID: c.SiteID, UserID: u.ID, Kind: "audio",
		Filename: "sesli-haber-" + itoa(int(c.ID)) + ".mp3",
		Path:     stored, Mime: "audio/mpeg", SizeBytes: &sz, Source: "tts",
	})
	if err != nil {
		s.ttsResult(w, r, c, "error", "Medya kaydı oluşturulamadı: "+err.Error())
		return
	}

	secs := tts.EstimateSeconds(text)
	if err := repo.SetContentAudio(ctx, s.pool, c.ID, publicMediaURL(&stored), secs, "hazir"); err != nil {
		s.ttsResult(w, r, c, "error", "Ses bilgisi yazılamadı: "+err.Error())
		return
	}
	s.audit(r, "content_tts", itoa(int(c.ID))+" parça="+itoa(len(chunks)))
	s.ttsResult(w, r, c, "info", "Sesli haber hazır ("+itoa(len(chunks))+" parça, ~"+itoa(int(secs))+" sn).")
}

// ttsResult, HTMX'te içerik formunu tazeler; normal istekte düzenleme sayfasına döner.
func (s *Server) ttsResult(w http.ResponseWriter, r *http.Request, c repo.Content, kind, text string) {
	if IsHTMX(r) {
		d := s.buildContentForm(w, r, &c, false)
		d.Flash = &flashMsg{Kind: kind, Text: text}
		s.renderIcerikFormPanelV2(w, r, d)
		return
	}
	s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(c.ID)), kind, text)
}
