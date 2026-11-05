package webui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Kayan "mürekkep hapı" testleri (Faz 155).
//
// Referans: Murmura'nın sekme geçişi. Hap, iki kenarı BAĞIMSIZ hareket eden
// kırpılmış bir katman; içinde etiketlerin zıt renkli kopyaları var. Kırpma
// penceresi etiketin üstüne geldiğinde renk tam kenarda değişiyor.
//
// Bu testler üç sessiz bozulmayı kilitliyor:
//   1. Hap kimliği sayfa yüklemesinde rastgele üretilirse geçiş kaydı
//      eşleşmez ve hap animasyonsuz, doğrudan hedefe oturur.
//   2. Tıklamada TIKLANAN sekmenin konumu kaydedilirse yeni sayfada o sekme
//      aktif olacağı için "eski = yeni" çıkar ve hap hiç kımıldamaz.
//   3. İki kenar aynı süre/eğriyle sürülürse "esneme" hissi kaybolur.

func hapJS(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", "admin-ui.js"))
	require.NoError(t, err)
	bas := strings.Index(string(b), "kayganHap")
	require.Greater(t, bas, 0, "hap modülü admin-ui.js içinde bulunamadı")
	return string(b)[bas:]
}

func TestHap_KimlikKararli(t *testing.T) {
	s := hapJS(t)
	assert.NotContains(t, s, "Math.random().toString(36).slice(2, 8)",
		"hap kimliği rastgele üretilmemeli — sayfalar arası geçiş eşleşmez")
	assert.Contains(t, s, "ray.dataset.hapId = kimlik(ray)",
		"kimlik kararlı bir fonksiyondan gelmeli")
	assert.Contains(t, s, "classList.contains('ws-links')",
		"çalışma alanı çubuğu her sayfada aynı kimliği almalı")
}

func TestHap_TiklamadaOncekiKonumKaydedilir(t *testing.T) {
	s := hapJS(t)
	// Kayıt, hapın anlık konumundan (__lr) yapılmalı.
	assert.Contains(t, s, "var anlik = pill.__lr",
		"tıklamada hapın anlık konumu okunmalı")
	assert.NotContains(t, s, "l: a.offsetLeft,",
		"tıklanan sekmenin konumu kaydedilmemeli — 'eski = yeni' olur")
}

func TestHap_IkiKenarBagimsiz(t *testing.T) {
	s := hapJS(t)
	// Önde giden kenar 550 ms, arkada kalan 360 ms: esneme bundan doğuyor.
	assert.Contains(t, s, "onde: 550", "önde giden kenar süresi")
	assert.Contains(t, s, "arkada: 360", "arkada kalan kenar süresi")
	assert.Contains(t, s, "sagaGidis ? sure.onde : sure.arkada",
		"yön değişince kenar rolleri de değişmeli")
	assert.Contains(t, s, "power3", "eğri ailesi belgelenmeli")
	// İki ayrı tween çağrısı olmalı (tek tween = tek kenar).
	assert.Equal(t, 2, strings.Count(s, "yurusun(onceki["),
		"iki kenar iki ayrı animasyonla sürülmeli")
}

func TestHap_ErisilebilirlikVeKademeliBozulma(t *testing.T) {
	s := hapJS(t)
	assert.Contains(t, s, "prefers-reduced-motion",
		"hareket hassasiyeti desteklenmeli")
	// JS çalışmazsa eski .on zemini görünür kalmalı: hap sınıfı yalnız JS
	// kurulduğunda ekleniyor.
	css, err := os.ReadFile(filepath.Join("static", "admin-pano.css"))
	require.NoError(t, err)
	assert.Contains(t, string(css), ".tb-hap-var > a.on",
		"aktif öğenin kendi zemini yalnız hap kurulunca kapatılmalı")
	assert.Contains(t, string(css), "pointer-events: none",
		"hap tıklamayı engellememeli")
}
