package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageLayoutDefaults(t *testing.T) {
	for _, c := range []struct {
		page        string
		wantContent int
	}{
		{"home", 1280},
		{"kategori", 1280},
		{"kategori:ekonomi", 1280},
		{"article", 960},
	} {
		v := pageLayoutViewFrom(c.page, pageLayoutConfig{}, false, false)
		assert.True(t, v.Sidebar, c.page+": sağ sütun varsayılan açık")
		assert.Equal(t, 33, v.SidebarWidth)
		assert.Equal(t, 67, v.MainWidth)
		assert.Equal(t, c.wantContent, v.ContentWidth)
		assert.False(t, v.HasCustom)
		assert.False(t, v.UsingGlobal)
	}
}

func TestPageLayoutNoSidebar(t *testing.T) {
	off := false
	v := pageLayoutViewFrom("kategori", pageLayoutConfig{Sidebar: &off}, true, false)
	assert.False(t, v.Sidebar)
	assert.Equal(t, 0, v.SidebarWidth, "sağ sütun kapalıyken genişlik sıfır")
	assert.Equal(t, 100, v.MainWidth, "tek sütun tüm genişliği alır")
	assert.True(t, v.HasCustom)
}

// TestPageLayoutMergeAlanBazliMiras kategori kayıtlarının yalnız kendi
// yazdığı alanlarda genel kategori düzenini ezmekle yükümlü olduğunu;
// diğer alanların genel ayardan geldiğini doğrular. Böylece "genel kategori
// düzeninde gizle" tüm kategorilere yansır.
func TestPageLayoutMergeAlanBazliMiras(t *testing.T) {
	on := true
	off := false
	base := pageLayoutConfig{HideBreadcrumb: &on, HideTitle: &on}
	over := pageLayoutConfig{HideBreadcrumb: &off, SidebarWidth: ptrInt(20)}

	m := mergePageLayout(base, over)
	require.NotNil(t, m.HideBreadcrumb)
	assert.False(t, *m.HideBreadcrumb, "kategori kendi alanında istisna yapabilmeli")
	require.NotNil(t, m.HideTitle)
	assert.True(t, *m.HideTitle, "yazılmayan alan genel ayardan gelmeli")
	require.NotNil(t, m.SidebarWidth)
	assert.Equal(t, 20, *m.SidebarWidth)

	// Boş kayıt her şeyi miras alır.
	m2 := mergePageLayout(base, pageLayoutConfig{})
	require.NotNil(t, m2.HideTitle)
	assert.True(t, *m2.HideTitle)
	require.NotNil(t, m2.HideBreadcrumb)
	assert.True(t, *m2.HideBreadcrumb)
}

func ptrInt(v int) *int { return &v }

// TestPageLayoutHideHeader kategori sayfası üst alanının (başlık, açıklama,
// kırıntı) sayfaya özel gizlenebildiğini ve varsayılanın görünür olduğunu
// doğrular.
func TestPageLayoutHideHeader(t *testing.T) {
	// Varsayılan: hepsi görünür.
	v := pageLayoutViewFrom("kategori:ekonomi", pageLayoutConfig{}, true, false)
	assert.False(t, v.HideTitle)
	assert.False(t, v.HideDesc)
	assert.False(t, v.HideBreadcrumb)

	hide := true
	v = pageLayoutViewFrom("kategori:ekonomi", pageLayoutConfig{
		HideTitle: &hide, HideBreadcrumb: &hide,
	}, true, false)
	assert.True(t, v.HideTitle, "kategori başlığı gizlenebilmeli")
	assert.False(t, v.HideDesc, "açıklama bağımsız kalabilmeli")
	assert.True(t, v.HideBreadcrumb, "kırıntı gizlenebilmeli")

	show := false
	v = pageLayoutViewFrom("kategori:ekonomi", pageLayoutConfig{
		HideTitle: &show, HideDesc: &hide, HideBreadcrumb: &show,
	}, true, false)
	assert.False(t, v.HideTitle, "açık bırakılan alan gizlenmemeli")
	assert.True(t, v.HideDesc)
	assert.False(t, v.HideBreadcrumb)
}

func TestPageLayoutClamp(t *testing.T) {
	on := true
	width := 90
	content := 200
	v := pageLayoutViewFrom("article", pageLayoutConfig{
		Sidebar: &on, SidebarWidth: &width, ContentWidth: &content,
	}, true, false)
	assert.Equal(t, 50, v.SidebarWidth, "sağ sütun en fazla %50")
	assert.Equal(t, 50, v.MainWidth)
	assert.Equal(t, 680, v.ContentWidth, "içerik en az 680 px")

	width = 5
	content = 5000
	v = pageLayoutViewFrom("article", pageLayoutConfig{
		Sidebar: &on, SidebarWidth: &width, ContentWidth: &content,
	}, true, false)
	assert.Equal(t, 20, v.SidebarWidth, "sağ sütun en az %20")
	assert.Equal(t, 80, v.MainWidth)
	assert.Equal(t, 1600, v.ContentWidth, "içerik en fazla 1600 px")
}

func TestPageLayoutCategoryInheritanceFlags(t *testing.T) {
	width := 40
	v := pageLayoutViewFrom("kategori:ekonomi", pageLayoutConfig{SidebarWidth: &width}, false, true)
	assert.Equal(t, 40, v.SidebarWidth)
	assert.Equal(t, 60, v.MainWidth)
	assert.False(t, v.HasCustom, "bu kategoriye özel kayıt yok")
	assert.True(t, v.UsingGlobal, "genel kategori şablonu kullanılıyor")
	assert.Equal(t, "kategori", pageLayoutBaseKey("kategori:ekonomi"))
	assert.Equal(t, "article", pageLayoutBaseKey("article"))
}
