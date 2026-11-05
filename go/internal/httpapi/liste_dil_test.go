package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Faz 192 — içerik listesinde dil tekilleştirmesi.
//
// Sahadaki sorun (ölçüm, üretim): "Yayında" sekmesinin ilk 40 satırının 39'u
// EN çeviriydi; TR asılları sonraki sayfalara düşüyordu. Yani editör Türkçe
// içerikleri liste üzerinden bulamıyordu. Aynı makale her dilde ayrı satır
// olarak tekrarlanıyordu (492 makale = 984 satır).
//
// Bu testler sözleşmeyi korur: tek satır + Dil kolonu (Tip'ten ÖNCE) +
// etikete tıklayınca o dilin editörü.

// TestListeDilKolonuTipdenOnce: Dil kolonu Tip'in solunda olmalı.
func TestListeDilKolonuTipdenOnce(t *testing.T) {
	tpl := okuDosya(t, "../webui/templkit/icerikler.templ")

	iDil := strings.Index(tpl, `<th class="dil-kolon">Dil</th>`)
	iTip := strings.Index(tpl, "<th>Tip</th>")
	require.Greater(t, iDil, 0, `th.dil-kolon yok`)
	require.Greater(t, iTip, 0, "<th>Tip</th> yok")
	assert.Less(t, iDil, iTip, "Dil kolonu Tip'ten ÖNCE gelmeli (kullanıcı isteği)")

	// Boş durum sütun sayısı başlık sayısıyla eşleşmeli (10).
	assert.Contains(t, tpl, `@TabloBos(10,`)

	// Satırda üç etiket durumu: bu satır / var / yok.
	assert.Contains(t, tpl, `class="dil-pill is-aktif"`)
	assert.Contains(t, tpl, `class="dil-pill is-var"`)
	assert.Contains(t, tpl, `class="dil-pill is-yok"`)
	// "yok" etiketi AI kuyruğuna yazmalı (Faz 190 ucu).
	assert.Contains(t, tpl, `hx-post={ "/admin/icerikler/" + l.Kaynak + "/cevir/" + l.Kod }`)
	assert.Contains(t, tpl, `hx-vals={ `+"`"+`{"ai":"1"}`+"`"+` }`)
}

// TestListeKokBazliSorgu: sorgu kök bazlı olmalı, arama tüm dilleri görmeli.
func TestListeKokBazliSorgu(t *testing.T) {
	src := okuDosya(t, "../repo/contents.go")

	assert.Contains(t, src, "KokBazli bool")
	assert.Contains(t, src, "func listContentsKokBazli(")
	// Kök başına TEK satır.
	assert.Contains(t, src, "SELECT DISTINCT ON (COALESCE(c.translation_of, c.id))")
	// Temsilci: önce kök dil, sonra en yeni.
	assert.Contains(t, src, "(c.locale = $1) DESC")
	// Arama TÜM dillerde olmalı: yoksa İngilizce başlıkla arayan bulamaz.
	assert.Contains(t, src, "q.title ILIKE $")
	// Dil filtresi "bu dilde sürümü olan makaleler" demek.
	assert.Contains(t, src, "AND l.locale = $")

	// Dil etiketleri TEK sorgu (N+1 yok).
	assert.Contains(t, src, "func DillerOf(")
	assert.Contains(t, src, "WHERE id = ANY($1) OR translation_of = ANY($1)")
}

// TestListeDilEtiketleri: etiket kuralları.
func TestListeDilEtiketleri(t *testing.T) {
	v := okuDosya(t, "icerikler_view.go")

	// Satırın kendi dili her zaman ilk ve aktif.
	assert.Contains(t, v, "func (s *Server) dilEtiketleri(")
	assert.Contains(t, v, "Aktif: true,")
	// Eksik dile AI kaynağı (kök kimliği) yazılır.
	assert.Contains(t, v, `kok := strconv.FormatInt(c.TranslationRootID(), 10)`)
	assert.Contains(t, v, "Kaynak: kok")
	// 4 etiketten fazlası basılmaz (satır şişmesin).
	assert.Contains(t, v, "if len(out) >= 4 {")

	// Handler kök bazlı sorgular ve etiket verisini toplar.
	h := okuDosya(t, "admin_handlers.go")
	assert.Contains(t, h, "KokBazli: true, KokDil: \"tr\", Locale: dilFiltre")
	assert.Contains(t, h, "repo.DillerOf(ctx, s.pool, kokler)")
	assert.Contains(t, h, "Diller:  diller[items[i].TranslationRootID()],")
}

// TestEditorDilSeridi: makaleyi açınca üstte dil geçişi olmalı.
func TestEditorDilSeridi(t *testing.T) {
	tpl := okuDosya(t, "../webui/templkit/icerikform.templ")

	i := strings.Index(tpl, `class="icerik-dil-seridi"`)
	require.Greater(t, i, 0, "editörde dil şeridi yok")
	serit := tpl[i:]
	serit = serit[:strings.Index(serit, "</div>")+6]

	assert.Contains(t, serit, "is-aktif", "bu sayfanın dili")
	assert.Contains(t, serit, "is-var", "var olan çeviri bağlantısı")
	assert.Contains(t, serit, "is-yok", "eksik dil")
	assert.Contains(t, serit, `"/admin/icerikler/" + itoaT64(l.ContentID)`)

	// CSS: etiket boyutu satır yüksekliğini bozmamalı.
	css := okuDosya(t, "../webui/static/admin-only.css")
	assert.Contains(t, css, ".dil-pill")
	assert.Contains(t, css, ".dil-pill.is-aktif")
	assert.Contains(t, css, ".dil-pill.is-yok")
}

// TestDilKolonuTabloHucresiKalir — Faz 192c.
//
// Kullanıcı bildirimi: "liste dil kolon diğer satırlardan yükseklik ve kolon
// hizalı değil".
//
// Ölçüm (tarayıcı): hücre `display:flex` idi → tablo hücresi yerleşiminden
// çıkıyordu:
//
//	satır 138px · dil hücresi 40px · üst kenar 434 (diğerleri 433)
//	dikey ortalama YOK
//
// Bir <td>'ye `display:flex` vermek tablo yerleşimini bozar. Hücre table-cell
// kalmalı; etiketler satır içi öğe olarak dizilmeli.
func TestDilKolonuTabloHucresiKalir(t *testing.T) {
	css := okuDosya(t, "../webui/static/admin-only.css")

	i := strings.Index(css, ".dil-hucre {")
	require.Greater(t, i, 0, ".dil-hucre kuralı yok")
	kural := css[i:]
	kural = kural[:strings.Index(kural, "}")]

	assert.NotContains(t, kural, "display: flex",
		"<td> üzerinde flex tablo hücresi yerleşimini bozar (ölçüm: 40px vs 138px)")
	assert.NotContains(t, kural, "display:flex")
	assert.Contains(t, kural, "white-space: nowrap")

	// Sabit kolon genişliği: sekmeler arasında zıplamasın (ölçüm: 126 ↔ 128).
	assert.Contains(t, css, ".dil-kolon,")
	assert.Contains(t, css, "td.dil-hucre { width: 126px; max-width: 126px; }")

	// Etiket satır içi ve dikey ortalanmış olmalı.
	j := strings.Index(css, ".dil-pill {")
	require.Greater(t, j, 0, ".dil-pill kuralı yok")
	pill := css[j:]
	pill = pill[:strings.Index(pill, "}")]
	assert.Contains(t, pill, "display: inline-flex")
	assert.Contains(t, pill, "vertical-align: middle")

	// Eski flex `gap` gitti; boşluk kardeş seçiciyle verilir.
	assert.Contains(t, css, ".dil-pill + .dil-pill { margin-left: 3px; }")
}
