package httpapi

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"

	"theblock/site/internal/ai"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// sohbetData AI sohbet sayfası verisidir.
type sohbetData struct {
	Convs     []repo.Conversation
	ActiveID  int64
	Msgs      []repo.Message
	Templates []repo.AITemplate
}

// handleAIChatGet sohbet sayfası (aktif sohbet + mesajlar + şablonlar).
func (s *Server) handleAIChatGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildAISohbetData(r)
	if IsHTMX(r) {
		s.renderAISohbetPanelV2(w, r, d)
		return
	}
	s.renderAISohbetPageV2(w, r, d)
}

func (s *Server) buildAISohbetData(r *http.Request) *tmplData {
	ctx := r.Context()
	u := userFromContext(ctx)

	d := s.baseView("tr", "/admin", "AI Sohbet")
	d.User = ptr(u)
	d.Flash = s.consumeFlash(r)

	convs, err := repo.ListConversations(ctx, s.pool, u.ID)
	if err != nil {
		convs = nil
	}

	actID := parseID(r.URL.Query().Get("konu"))
	if actID == 0 && len(convs) > 0 {
		actID = convs[0].ID
	}

	sc := &sohbetData{Convs: convs, ActiveID: actID}
	if actID != 0 {
		if msgs, err := repo.ListMessages(ctx, s.pool, actID, 40); err == nil {
			sc.Msgs = msgs
		}
	}
	if tpls, err := repo.ListAITemplates(ctx, s.pool, "sohbet"); err == nil {
		sc.Templates = tpls
	}
	d.AISohbet = sc
	return d
}

// renderAISohbetPanel mevcut konuşmayı aktif tutarak paneli basar.
func (s *Server) renderAISohbetPanel(w http.ResponseWriter, r *http.Request, actID int64, kind, text string) {
	q := r.URL.Query()
	if actID != 0 {
		q.Set("konu", itoa(int(actID)))
	}
	r2 := r.Clone(r.Context())
	r2.URL.RawQuery = q.Encode()
	d := s.buildAISohbetData(r2)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderAISohbetPanelV2(w, r, d)
}

// renderAIChatMsgs konuşmadaki tüm mesajları tek parça olarak basar.
// errText doluysa parçanın başına hata rozeti ekler (HTMX hata akışı).
func (s *Server) renderAIChatMsgs(w http.ResponseWriter, r *http.Request, convID int64, errText string) {
	msgs, err := repo.ListMessages(r.Context(), s.pool, convID, 40)
	if err != nil {
		msgs = nil
	}
	view := aiSohbetViewFrom(&sohbetData{ActiveID: convID, Msgs: msgs}, nil)
	var sb strings.Builder
	if err := templkit.AISohbetMsgs(view.Msgs).Render(r.Context(), &sb); err != nil {
		s.logger.Error("sohbet mesaj parçası basılamadı", "error", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request")
	if errText != "" {
		_, _ = fmt.Fprintf(w, `<div class="flash error">%s</div>`, html.EscapeString(errText))
	}
	_, _ = w.Write([]byte(sb.String()))
}

// handleAIChatNew yeni sohbet açar.
func (s *Server) handleAIChatNew(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	id, err := repo.CreateConversation(r.Context(), s.pool, u.ID, "")
	if err != nil {
		s.renderAISohbetPanel(w, r, 0, "error", "Sohbet açılamadı.")
		return
	}
	if IsHTMX(r) {
		s.renderAISohbetPanel(w, r, id, "info", "Yeni sohbet açıldı.")
		return
	}
	http.Redirect(w, r, "/admin/ai-sohbet?konu="+itoa(int(id)), http.StatusFound)
}

// handleAIChatMsg kullanıcı mesajı → AI yanıtı (JSON).
func (s *Server) handleAIChatMsg(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := userFromContext(ctx)
	convID := parseID(r.PathValue("id"))

	var req struct {
		Metin string `json:"metin"`
	}
	if IsHTMX(r) {
		_ = r.ParseForm()
		req.Metin = r.FormValue("metin")
	} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz istek"})
		return
	}
	req.Metin = strings.TrimSpace(req.Metin)
	if req.Metin == "" {
		if IsHTMX(r) {
			s.renderAIChatMsgs(w, r, convID, "Mesaj boş.")
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "mesaj boş"})
		return
	}
	convWasZero := convID == 0
	if convID == 0 {
		nid, err := repo.CreateConversation(ctx, s.pool, u.ID, "")
		if err != nil {
			if IsHTMX(r) {
				s.renderAIChatMsgs(w, r, 0, "Sohbet açılamadı.")
				return
			}
			s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "sohbet açılamadı"})
			return
		}
		convID = nid
	}
	if _, err := repo.GetConversation(ctx, s.pool, u.ID, convID); err != nil {
		if IsHTMX(r) {
			s.renderAIChatMsgs(w, r, convID, "Sohbet bulunamadı.")
			return
		}
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "sohbet bulunamadı"})
		return
	}

	// kullanıcı mesajını kaydet
	if _, err := repo.AddMessage(ctx, s.pool, convID, "user", req.Metin); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "kayıt hatası"})
		return
	}

	// bağlam: son birkaç mesaj
	history, _ := repo.ListMessages(ctx, s.pool, convID, 8)
	msgs := []ai.Msg{{Role: ai.RoleSystem, Content: systemPrompt()}}
	for _, m := range history {
		r2 := ai.RoleUser
		if m.Role == "assistant" {
			r2 = ai.RoleSystem
		}
		if m.Role == "system" {
			r2 = ai.RoleSystem
		}
		msgs = append(msgs, ai.Msg{Role: r2, Content: m.Content})
	}

	out, err := s.ai.Chat(ctx, msgs)
	if err != nil {
		if IsHTMX(r) {
			s.renderAIChatMsgs(w, r, convID, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	// asistan yanıtını kaydet
	if _, err := repo.AddMessage(ctx, s.pool, convID, "assistant", out); err != nil {
		s.logger.Warn("asistan mesajı kaydı", "error", err)
	}
	provider := "mock"
	name := "mock"
	if s.ai.Enabled() {
		provider = s.ai.Providers()[0]
		name = provider
	}
	_ = provider
	if _, lerr := repo.LogAIUsage(ctx, s.pool, &repo.AIUsage{
		UserID: ptr(u.ID), Provider: name, Action: "sohbet",
		PromptTokens: len(req.Metin) / 4, CompletionTokens: len(out) / 4,
	}); lerr != nil {
		s.logger.Warn("ai kullanım logu", "error", lerr)
	}

	if IsHTMX(r) {
		if convWasZero {
			// Yeni sohbet: paneli değiştir ki yan liste + form yeni konuşmayı göstersin.
			s.renderAISohbetPanel(w, r, convID, "info", "Yeni sohbet başlatıldı.")
			return
		}
		s.renderAIChatMsgs(w, r, convID, "")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": out, "conv_id": convID})
}

// handleAIChatDelete sohbeti siler.
func (s *Server) handleAIChatDelete(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteConversation(r.Context(), s.pool, u.ID, id); err != nil {
		if IsHTMX(r) {
			s.renderAISohbetPanel(w, r, 0, "error", "Silinemedi.")
			return
		}
		s.flashRedirect(w, r, "/admin/ai-sohbet", "error", "Silinemedi.")
		return
	}
	if IsHTMX(r) {
		s.renderAISohbetPanel(w, r, 0, "info", "Sohbet silindi.")
		return
	}
	s.flashRedirect(w, r, "/admin/ai-sohbet", "info", "Sohbet silindi.")
}
