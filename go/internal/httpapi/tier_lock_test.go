package httpapi

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// applyTierLock: form değeri > kategori varsayılanı > is_exclusive sırası.

func tierLockTest(t *testing.T, form url.Values, c *repo.Content, categoryDefault int) {
	t.Helper()
	r := httptest.NewRequest("POST", "/admin/icerikler", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())
	applyTierLock(r, c, categoryDefault)
}

func TestTierLockKategoriVarsayilaniDevralinir(t *testing.T) {
	c := &repo.Content{} // yeni içerik (ID=0)
	tierLockTest(t, url.Values{}, c, 3)
	assert.Equal(t, 3, c.MinTierWeight)
	assert.True(t, c.IsExclusive, "katman > 0 ise içerik aboneye özel olmalı")
}

func TestTierLockFormDegeriKategoriyiEzer(t *testing.T) {
	// Açıkça 0: kategori 3 olsa da içerik açık kalır.
	c := &repo.Content{}
	tierLockTest(t, url.Values{"min_tier_weight": []string{"0"}}, c, 3)
	assert.Equal(t, 0, c.MinTierWeight)
	assert.False(t, c.IsExclusive)

	// Açıkça 5: kategori 1 olsa da 5 uygulanır.
	c2 := &repo.Content{}
	tierLockTest(t, url.Values{"min_tier_weight": []string{"5"}}, c2, 1)
	assert.Equal(t, 5, c2.MinTierWeight)
	assert.True(t, c2.IsExclusive)
}

func TestTierLockMevcutIcerikDegeriKorunur(t *testing.T) {
	// Güncelleme (ID>0) ve formda alan yok: mevcut kademe KORUNUR
	// (istemeden kademe düşürmeyi önler).
	c := &repo.Content{ID: 42, MinTierWeight: 2, IsExclusive: true}
	tierLockTest(t, url.Values{}, c, 5)
	assert.Equal(t, 2, c.MinTierWeight, "mevcut kademe korunmalı")
	assert.True(t, c.IsExclusive)

	// Güncellemede form değeri yine kazanır.
	c2 := &repo.Content{ID: 42, MinTierWeight: 2, IsExclusive: true}
	tierLockTest(t, url.Values{"min_tier_weight": []string{"4"}}, c2, 5)
	assert.Equal(t, 4, c2.MinTierWeight)
}

func TestTierLockExclusiveVarsayilanBir(t *testing.T) {
	c := &repo.Content{IsExclusive: true}
	tierLockTest(t, url.Values{}, c, 0)
	assert.Equal(t, 1, c.MinTierWeight, "kategori kuralı yoksa en düşük katman")
	assert.True(t, c.IsExclusive)
}

func TestTierLockAcikIcerik(t *testing.T) {
	c := &repo.Content{}
	tierLockTest(t, url.Values{}, c, 0)
	assert.Equal(t, 0, c.MinTierWeight)
	assert.False(t, c.IsExclusive)
}

func TestTierLockGecersizDegerYokSayilir(t *testing.T) {
	for _, v := range []string{"abc", "6", "-2"} {
		c := &repo.Content{}
		tierLockTest(t, url.Values{"min_tier_weight": []string{v}}, c, 2)
		assert.Equal(t, 2, c.MinTierWeight,
			"geçersiz form değeri yok sayılıp kategori devralınmalı: "+v)
	}
}

func TestTierLockGuncellemedeGecersizDegerKorunur(t *testing.T) {
	// Güncellemede bozuk form değeri kademeyi sıfırlamamalı; handler mevcut
	// değeri DB'den yükler, applyTierLock da onu korur.
	c := &repo.Content{ID: 42, MinTierWeight: 3, IsExclusive: true}
	tierLockTest(t, url.Values{"min_tier_weight": []string{"abc"}}, c, 0)
	assert.Equal(t, 3, c.MinTierWeight, "geçersiz form değeri mevcut kademeyi düşürmemeli")
	assert.True(t, c.IsExclusive)
}
