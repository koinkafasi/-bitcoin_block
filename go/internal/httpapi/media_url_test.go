package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMediaURLUploadRejectsPrivateHost(t *testing.T) {
	s := &Server{}
	form := url.Values{"url": {"http://127.0.0.1/private.png"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/dosyalar/url?json=1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.handleMediaURLUpload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("özel ağ adresi reddedilmeli, durum=%d gövde=%s", rec.Code, rec.Body.String())
	}
}

func TestMediaURLUploadRejectsNonHTTP(t *testing.T) {
	s := &Server{}
	form := url.Values{"url": {"file:///etc/passwd"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/dosyalar/url?json=1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.handleMediaURLUpload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("file şeması reddedilmeli, durum=%d gövde=%s", rec.Code, rec.Body.String())
	}
}
