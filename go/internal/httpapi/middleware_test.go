package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- flash mesajı: başlık enjeksiyonu koruması -------------------

func TestSanitizeFlash(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"duz metin", "Kaydedildi.", "Kaydedildi."},
		{"turkce karakter korunur", "İşlem başarılı: şükr", "İşlem başarılı: şükr"},
		{" satir sonu temizlenir", "satir1\nsatir2", "satir1 satir2"},
		{"donus temizlenir", "satir1\rsatir2", "satir1 satir2"},
		{"cift tirnak kirpilmaz", `de"ger`, `de"ger`},
		{"NUL ve kontrol karakterleri", "a\x00b\x1fc", "a b c"},
		{"sekme", "a\tb", "a b"},
		{"bos", "", ""},
		{"sadece satir sonu", "\n", " "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeFlash(c.in)
			assert.Equal(t, c.want, got)
			assert.NotContains(t, got, "\n", "yanıt başlığı enjeksiyonu satır sonu içermemeli")
			assert.NotContains(t, got, "\r", "yanıt başlığı enjeksiyonu taşıma içermemeli")
		})
	}
}

func TestSanitizeFlashRemovesAllControlChars(t *testing.T) {
	for r := rune(0); r < 32; r++ {
		got := sanitizeFlash("a" + string(r) + "b")
		assert.NotContains(t, got, string(r), "kontrol karakteri %d sızmamalı", r)
	}
}

// ---- flash çerezi gidiş-dönüş -------------------------------------

func TestFlashRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		kind string
		text string
	}{
		{"bilgi", "info", "Kaydedildi."},
		{"hata", "error", "Kaydedilemedi."},
		{"turkce + emoji", "info", "İşlem tamam ✅"},
		{"ayrac iceren metin", "info", "a|b|c"},
		{"bos metin", "info", ""},
		{"cok uzun metin", "info", strings.Repeat("x", 500)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Server{}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)

			s.flash(rec, req, c.kind, c.text)

			cookies := rec.Result().Cookies()
			require.Len(t, cookies, 1)
			ck := cookies[0]
			assert.Equal(t, flashCookie, ck.Name)
			assert.True(t, ck.HttpOnly, "flash çerezi HttpOnly olmalı")
			assert.Equal(t, http.SameSiteLaxMode, ck.SameSite)
			assert.Equal(t, 60, ck.MaxAge)

			// çerezi isteğe tak ve geri oku
			req2 := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req2.AddCookie(&http.Cookie{Name: ck.Name, Value: ck.Value})

			msg := s.consumeFlash(req2)
			require.NotNil(t, msg)
			assert.Equal(t, c.kind, msg.Kind)
			assert.Equal(t, c.text, msg.Text)
		})
	}
}

func TestConsumeFlashInvalidCases(t *testing.T) {
	s := &Server{}
	cases := []struct {
		name  string
		value string
	}{
		{"yok", ""},
		{"bozuk base64", "!!!gecersiz!!!"},
		{"ayracsiz", "djkflasdkj"},
		{"bos deger", " "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if c.value != "" {
				req.AddCookie(&http.Cookie{Name: flashCookie, Value: c.value})
			}
			assert.Nil(t, s.consumeFlash(req), "geçersiz flash çerezi sessizce yok sayılmalı")
		})
	}
}

func TestConsumeFlashNoCookie(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	assert.Nil(t, s.consumeFlash(req), "çerez yoksa flash olmamalı")
}

func TestFlashIsSafeAgainstHeaderInjection(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)

	// saldırgan girdisi: başlık enjeksiyonu denemesi
	s.flash(rec, req, "info", "x\r\nSet-Cookie: admin=1")

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1, "tek çerez üretilmeli, başlık enjeksiyonu olmamalı")
	assert.Equal(t, flashCookie, cookies[0].Name)
	assert.Equal(t, flashCookie, cookies[0].Name, "enjeksiyonla ikinci çerez eklenmemeli")
	assert.NotContains(t, cookies[0].Value, "\n")
}

// ---- flash + yönlendirme ------------------------------------------

func TestFlashRedirect(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/servisler/ayar", nil)

	s.flashRedirect(rec, req, "/admin/servisler", "info", "Anahtar kaydedildi.")

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/admin/servisler", rec.Header().Get("Location"))

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, flashCookie, cookies[0].Name)

	req2 := httptest.NewRequest(http.MethodGet, "/admin/servisler", nil)
	req2.AddCookie(cookies[0])
	msg := s.consumeFlash(req2)
	require.NotNil(t, msg)
	assert.Equal(t, "info", msg.Kind)
	assert.Equal(t, "Anahtar kaydedildi.", msg.Text)
}

func TestClearFlashExpiresCookie(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()

	s.clearFlash(rec)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, flashCookie, cookies[0].Name)
	assert.Equal(t, "", cookies[0].Value)
	assert.Negative(t, cookies[0].MaxAge, "çerez hemen sona ermeli")
}

// ---- istek kimliği + durum kaydı ----------------------------------

// genID kriptografik rastgelelik kullanır: 12 bayt → 24 hex karakter.
// (Eskiden zaman tabanlı üretiliyordu ve tahmin edilebilirdi.)
func TestGenID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := genID()
		assert.Len(t, id, 24)
		for _, r := range id {
			assert.True(t,
				(r >= 'a' && r <= 'f') || (r >= '0' && r <= '9'),
				"beklenmeyen karakter: %q", r)
		}
		seen[id] = true
	}
	assert.Greater(t, len(seen), 1, "kimlikler tekrarlanmamalı")
}

func TestStatusWriterRecordsCode(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec, status: http.StatusOK}

	assert.Equal(t, http.StatusOK, sw.status, "varsayılan 200")
	sw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, sw.status)
	assert.Equal(t, http.StatusNotFound, rec.Code, "alt yazıcıya da iletilmeli")
}
