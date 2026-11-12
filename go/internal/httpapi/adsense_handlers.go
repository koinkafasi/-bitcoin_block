package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- AdSense/GA OAuth (spec 05 §3 / 03 P5-D, P5-13) ----

const adsenseReadScope = "https://www.googleapis.com/auth/adsense.readonly"

type adsenseData struct {
	ClientID    string
	Connected   bool
	AccountName string
	TokenExpiry string
	RedirectURI string
	GAProperty  string
	SonSenkron  string
}

func (s *Server) adsenseSetting(ctx context.Context, key string) string {
	b, err := repo.GetSetting(ctx, s.pool, "adsense", key)
	if err != nil {
		return ""
	}
	var v string
	if json.Unmarshal(b, &v) == nil {
		return v
	}
	return strings.Trim(string(b), `"`)
}

func (s *Server) setAdsenseSetting(ctx context.Context, key, val string) {
	b, _ := json.Marshal(val)
	_ = repo.SetSetting(ctx, s.pool, "adsense", key, b)
}

func (s *Server) googleOAuth(ctx context.Context) platform.GoogleOAuth {
	return platform.GoogleOAuth{
		ClientID:     s.adsenseSetting(ctx, "client_id"),
		ClientSecret: s.adsenseSetting(ctx, "client_secret"),
		RedirectURI:  strings.TrimRight(s.cfg.BaseURL, "/") + "/admin/adsense/callback",
	}
}

// handleAdsenseGet panel: durum + ayarlar.
func (s *Server) handleAdsenseGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildAdsenseData(r)
	if IsHTMX(r) {
		s.renderAdsenseMerkezPanel(w, r, d)
		return
	}
	s.renderAdsenseMerkezPage(w, r, d)
}

func (s *Server) buildAdsenseData(r *http.Request) *tmplData {
	ctx := r.Context()
	d := s.baseView("tr", "/admin", "AdSense / GA (Gelir)")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	oauth := s.googleOAuth(ctx)
	exp := ""
	if e := s.adsenseSetting(ctx, "token_expiry"); e != "" {
		if n, err := strconv.ParseInt(e, 10, 64); err == nil && n > 0 {
			exp = time.Unix(n, 0).Local().Format("2006-01-02 15:04")
		}
	}
	d.Adsense = &adsenseData{
		ClientID:    oauth.ClientID,
		Connected:   s.adsenseSetting(ctx, "refresh_token") != "",
		AccountName: s.adsenseSetting(ctx, "account_name"),
		TokenExpiry: exp,
		RedirectURI: oauth.RedirectURI,
		GAProperty:  s.ga4PropertyID(ctx),
		SonSenkron:  s.adsenseSetting(ctx, "son_senkron"),
	}
	return d
}

func (s *Server) renderAdsensePanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildAdsenseData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderAdsenseMerkezPanel(w, r, d)
}

func (s *Server) adsenseResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderAdsensePanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/adsense", kind, text)
}

func (s *Server) ga4MulkuKaydet(ctx context.Context, ham string) string {
	ga := normalizeGA4PropertyID(ham)
	if ga == "" {
		return ""
	}
	_ = repo.SetTrendSetting(ctx, s.pool, "ga4_property", ga)
	s.setAdsenseSetting(ctx, "ga_property", ga)
	return ga
}

// handleAdsenseSave istemci kimlik bilgilerini kaydeder.
func (s *Server) handleAdsenseSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.setAdsenseSetting(ctx, "client_id", strings.TrimSpace(r.FormValue("client_id")))
	if c := r.FormValue("client_secret"); c != "" { // boş = koru
		s.setAdsenseSetting(ctx, "client_secret", c)
	}
	s.ga4MulkuKaydet(ctx, r.FormValue("ga_property"))
	s.audit(r, "adsense_ayar", "oauth")
	s.adsenseResult(w, r, "info", "OAuth bilgileri kaydedildi.")
}

// handleAdsenseConnect Google onay ekranına yönlendirir.
func (s *Server) handleAdsenseConnect(w http.ResponseWriter, r *http.Request) {
	oauth := s.googleOAuth(r.Context())
	if !oauth.Configured() {
		s.adsenseResult(w, r, "error", "Önce OAuth İstemci Kimliği ve Sırrı kaydedin.")
		return
	}
	state := rndSuffix() + rndSuffix()
	http.SetCookie(w, &http.Cookie{Name: "ga_state", Value: state, Path: "/", MaxAge: 600, HttpOnly: true, Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, oauth.AuthURL(state, adsenseReadScope), http.StatusFound)
}

// handleAdsenseCallback OAuth dönüşü: code → token.
func (s *Server) handleAdsenseCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ck, err := r.Cookie("ga_state")
	if err != nil || ck.Value == "" || ck.Value != r.URL.Query().Get("state") {
		s.adsenseResult(w, r, "error", "Geçersiz OAuth state (CSRF koruması).")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "ga_state", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookie()})
	if e := r.URL.Query().Get("error"); e != "" {
		s.adsenseResult(w, r, "error", "Google yetkilendirme reddedildi: "+e)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		s.adsenseResult(w, r, "error", "Yetkilendirme kodu gelmedi.")
		return
	}
	ts, err := s.googleOAuth(ctx).Exchange(ctx, code)
	if err != nil {
		s.logger.Error("adsense token değişimi", "error", err)
		s.adsenseResult(w, r, "error", "Token alınamadı: "+err.Error())
		return
	}
	s.setAdsenseSetting(ctx, "access_token", ts.AccessToken)
	if ts.RefreshToken != "" {
		s.setAdsenseSetting(ctx, "refresh_token", ts.RefreshToken)
	}
	s.setAdsenseSetting(ctx, "token_expiry", strconv.FormatInt(time.Now().Add(time.Duration(ts.ExpiresIn)*time.Second).Unix(), 10))
	s.audit(r, "adsense_baglan", "oauth")
	s.adsenseResult(w, r, "info", "AdSense bağlantısı kuruldu.")
}

// handleAdsenseSync access token'ı yeniler ve AdSense hesaplarını doğrular.
func (s *Server) handleAdsenseSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rt := s.adsenseSetting(ctx, "refresh_token")
	if rt == "" {
		s.adsenseResult(w, r, "error", "Bağlantı yok — önce 'Bağlan'.")
		return
	}
	ts, err := s.googleOAuth(ctx).Refresh(ctx, rt)
	if err != nil {
		s.adsenseResult(w, r, "error", "Token yenilenemedi: "+err.Error())
		return
	}
	s.setAdsenseSetting(ctx, "access_token", ts.AccessToken)
	s.setAdsenseSetting(ctx, "token_expiry", strconv.FormatInt(time.Now().Add(time.Duration(ts.ExpiresIn)*time.Second).Unix(), 10))

	// AdSense Management API v2: hesapları listele (token doğrulama + hesap adı)
	body, code, err := platform.GetJSON(ctx, ts.AccessToken, "https://adsense.googleapis.com/v2/accounts")
	if err != nil {
		s.adsenseResult(w, r, "error", "AdSense API çağrısı başarısız: "+err.Error())
		return
	}
	if code != 200 {
		s.adsenseResult(w, r, "error", "AdSense API HTTP "+itoa(code)+": "+string(body[:minInt(len(body), 200)]))
		return
	}
	var acc struct {
		Accounts []struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"accounts"`
	}
	_ = json.Unmarshal(body, &acc)
	name := ""
	if len(acc.Accounts) > 0 {
		name = acc.Accounts[0].DisplayName
		if name == "" {
			name = acc.Accounts[0].Name
		}
	}
	s.setAdsenseSetting(ctx, "account_name", name)
	s.setAdsenseSetting(ctx, "son_senkron", time.Now().Format("2006-01-02 15:04"))
	s.audit(r, "adsense_senkron", name)
	s.adsenseResult(w, r, "info", "AdSense senkronize edildi: "+name)
}

// handleAdsenseDisconnect token'ları temizler.
func (s *Server) handleAdsenseDisconnect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	for _, k := range []string{"refresh_token", "access_token", "token_expiry", "account_name", "son_senkron"} {
		s.setAdsenseSetting(ctx, k, "")
	}
	s.audit(r, "adsense_kes", "oauth")
	s.adsenseResult(w, r, "info", "AdSense bağlantısı kaldırıldı.")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
