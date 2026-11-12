package httpapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func TestCdYorumKartiBosAdiDoldurur(t *testing.T) {
	simdi := time.Now()
	k := cdYorumKart(repo.Comment{AuthorName: "  ", Body: "metin", CreatedAt: simdi}, simdi)

	assert.Equal(t, "Üye", k.Ad, "adsız yorum 'Üye' olarak gösterilmeli")
	assert.Equal(t, "Ü", k.BasHarf)
}

func TestCdBasHarfAl(t *testing.T) {
	assert.Equal(t, "A", cdBasHarfAl("Ahmet Yılmaz"))
	assert.Equal(t, "Ş", cdBasHarfAl("şule"))
	// Baştaki boşluk atlanır.
	assert.Equal(t, "M", cdBasHarfAl("  Mehmet"))
	// Hiç harf yoksa varsayılan.
	assert.Equal(t, "Ü", cdBasHarfAl(""))
}

func TestCdYorumKartiZamanVeBegeni(t *testing.T) {
	simdi := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	k := cdYorumKart(repo.Comment{
		ID: 7, AuthorName: "Ayşe", Body: "yorum",
		CreatedAt: simdi.Add(-3 * time.Hour), Likes: 4,
	}, simdi)

	assert.Equal(t, "7", k.ID)
	assert.Equal(t, "Ayşe", k.Ad)
	assert.Equal(t, 4, k.Begeni)
	assert.Equal(t, "3 saat önce", k.Zaman)
}

// Sıralama kuralı: en beğenilen üstte, eşitlikte eski yorum önce.
// Burada kuralın kendisi doğrulanıyor (veritabanı gerekmeden).
func TestYorumSiralamaKurali(t *testing.T) {
	simdi := time.Now()
	yorumlar := []repo.Comment{
		{ID: 1, Likes: 1, CreatedAt: simdi},
		{ID: 2, Likes: 9, CreatedAt: simdi.Add(-48 * time.Hour)},
		{ID: 3, Likes: 9, CreatedAt: simdi.Add(-72 * time.Hour)},
	}
	sirala := func(l []repo.Comment) []int64 {
		kopya := append([]repo.Comment(nil), l...)
		// cdYorumVerisi ile aynı ölçüt.
		for i := 1; i < len(kopya); i++ {
			for j := i; j > 0; j-- {
				a, b := kopya[j-1], kopya[j]
				daha := b.Likes > a.Likes || (b.Likes == a.Likes && b.CreatedAt.Before(a.CreatedAt))
				if !daha {
					break
				}
				kopya[j-1], kopya[j] = b, a
			}
		}
		var ids []int64
		for _, c := range kopya {
			ids = append(ids, c.ID)
		}
		return ids
	}

	ids := sirala(yorumlar)
	require.Len(t, ids, 3)
	// 9 beğenili ikisi önce; aralarında ESKİ olan (3) üstte.
	assert.Equal(t, int64(3), ids[0])
	assert.Equal(t, int64(2), ids[1])
	assert.Equal(t, int64(1), ids[2], "az beğenilen yeni yorum en altta")
}
