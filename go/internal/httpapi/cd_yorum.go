package httpapi

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ---- Yeni tasarımda yorum bölümü ----
//
// DURUM: yorum altyapısı (tablo, ekleme ucu, beğeni, moderasyon paneli,
// yanıt zinciri, IP hız sınırı) yazılıydı ama yalnız ESKİ şablonda
// görünüyordu — paywall'la aynı durum. Yeni CoinDesk tasarımındaki
// makale sayfasında yorum hiç basılmıyordu.
//
// SIRALAMA "EN BEĞENİLEN": ilk yorumlar genelde en zayıf olanlar oluyor
// ve "en yeni" sırası sayfayı onlarla açıyor. Beğeni sırası, okunmaya
// değer yorumu üste taşıyor; eşitlikte eski yorum önce gelir ki sıra
// her yenilemede oynamasın.
//
// SIFIR YORUMDA "0" YAZILMAZ: boş bir "Yorumlar (0)" başlığı "burası
// ıssız" sinyali veriyor. Onun yerine doğrudan davet basılır.

// cdYorumVerisi makale sayfasının yorum bölümü.
func (s *Server) cdYorumVerisi(r *http.Request, c repo.Content) templkit.CDYorumBolumu {
	bolum := templkit.CDYorumBolumu{
		ContentID: itoa64(c.ID),
		Acik:      c.CommentStatus != "kapali",
	}
	if !bolum.Acik {
		return bolum
	}

	// Oturum durumu: form mu yoksa giriş daveti mi basılacak.
	if u, err := s.currentUser(r); err == nil && u.ID > 0 {
		bolum.UyeGirisli = true
		bolum.UyeAdi = strings.TrimSpace(u.Name)
	}

	ham, err := repo.ListApprovedComments(r.Context(), s.pool, c.ID)
	if err != nil || len(ham) == 0 {
		return bolum
	}

	// Kökler ve yanıtlar ayrılır; yanıtlar kendi köküne bağlanır.
	yanitlar := map[int64][]repo.Comment{}
	var kokler []repo.Comment
	for _, y := range ham {
		if y.ParentID > 0 {
			yanitlar[y.ParentID] = append(yanitlar[y.ParentID], y)
			continue
		}
		kokler = append(kokler, y)
	}

	sort.SliceStable(kokler, func(i, j int) bool {
		if kokler[i].Likes != kokler[j].Likes {
			return kokler[i].Likes > kokler[j].Likes
		}
		return kokler[i].CreatedAt.Before(kokler[j].CreatedAt)
	})

	simdi := time.Now()
	for _, k := range kokler {
		kart := cdYorumKart(k, simdi)
		for _, y := range yanitlar[k.ID] {
			kart.Yanitlar = append(kart.Yanitlar, cdYorumKart(y, simdi))
		}
		bolum.Yorumlar = append(bolum.Yorumlar, kart)
	}
	bolum.Sayi = len(ham)
	return bolum
}

func cdYorumKart(c repo.Comment, simdi time.Time) templkit.CDYorum {
	ad := strings.TrimSpace(c.AuthorName)
	if ad == "" {
		ad = "Üye"
	}
	return templkit.CDYorum{
		ID:      itoa64(c.ID),
		Ad:      ad,
		BasHarf: cdBasHarfAl(ad),
		Metin:   c.Body,
		Zaman:   trGoreliZaman(c.CreatedAt, simdi),
		Begeni:  c.Likes,
	}
}

// cdBasHarfAl avatar yerine kullanılan harf. Yorumcularda görsel yok;
// baş harf, satırları birbirinden ayırmaya yetiyor.
func cdBasHarfAl(ad string) string {
	for _, r := range ad {
		if r != ' ' {
			return strings.ToUpper(string(r))
		}
	}
	return "Ü"
}
