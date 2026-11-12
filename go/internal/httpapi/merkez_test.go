package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerkezPanoyaYonlendirir(t *testing.T) {
	s := &Server{} // handler yalnız yönlendirme yapıyor, bağımlılık yok
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/admin/merkez?sek=marka", nil)

	s.handleMerkezKaldirildi(w, r)

	resp := w.Result()
	require.Equal(t, http.StatusMovedPermanently, resp.StatusCode,
		"kalıcı yönlendirme bekleniyordu")
	assert.Equal(t, "/admin", resp.Header.Get("Location"))
}
