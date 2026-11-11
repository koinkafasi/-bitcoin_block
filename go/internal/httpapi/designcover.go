package httpapi

import (
	"context"
	"strconv"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// systemUserID oturumsuz sistemsel üretim (blok kapak şablonu) için
// kullanılacak ilk aktif yönetici kullanıcıyı döndürür (önbellekli).
// Bulunamazsa 0 döner — o durumda medya kaydı yapılamaz, kapak sessizce atlanır.
func (s *Server) systemUserID(ctx context.Context) int64 {
	s.sysMu.Lock()
	defer s.sysMu.Unlock()
	if s.sysUserID != 0 {
		return s.sysUserID
	}
	var id int64
	if err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE role='Yönetici' ORDER BY id LIMIT 1`).Scan(&id); err != nil {
		return 0
	}
	s.sysUserID = id
	return id
}

// ---- Blok başına kapak şablonu eşlemesi (Faz 3) ----
//
// Tasarımcıda bir blok (slider/manset6/manset/one_cikanlar/ust_manset) bir
// Manşet Editörü şablonu (headline_designs) seçebilir. Bu durumda bloğun
// içerikleri kendi görselini kullanmak yerine seçilen şablonun üzerinden
// otomatik kapak ile render edilir (başlık/özet/kategori/görsel içerikten
// doldurularak). Sonuç medya olarak saklanır ve (içerik_id, şablon_id) ile
// önbelleğe alınır; içerik güncellenince otomatik yeniden üretilir.

// applyDesignCover bloğun Slider/Items görsellerini seçilen şablonla üretilen
// kapak medyasıyla değiştirir. Şablon yoksa veya içerik hatası çıkarsa içeriğin
// kendi görseli bozulmadan kalır (sessiz hata, görsel yolundan düşme).
func (s *Server) applyDesignCover(ctx context.Context, b *homeBlock) {
	if b.KapakTasarimID <= 0 {
		return
	}
	s.applyCoverToItems(ctx, b.Slides, b.KapakTasarimID)
	s.applyCoverToItems(ctx, b.Items, b.KapakTasarimID)
}

// applyCoverToItems dilimindeki her içeriğin CoverMediaID'sini şablon kapak
// medyasıyla (önbellekli üretim) değiştirir.
func (s *Server) applyCoverToItems(ctx context.Context, items []repo.Content, designID int64) {
	for i := range items {
		mid, err := s.ensureContentDesignCover(ctx, items[i], designID)
		if err != nil || mid == nil {
			continue
		}
		items[i].CoverMediaID = mid
	}
}

// ensureContentDesignCover bir içerik için şablon kapak görselini üretir ve
// (içerik, şablon) önbelleğine yazar. Daha önce üretilmişse ve içerik
// güncellenmemişse önbellekten okur.
func (s *Server) ensureContentDesignCover(ctx context.Context, c repo.Content, designID int64) (*int64, error) {
	if designID <= 0 || c.ID <= 0 {
		return nil, nil
	}
	cached, err := repo.GetContentDesignCover(ctx, s.pool, c.ID, designID)
	if err == nil && !c.UpdatedAt.After(cached.ContentUpdatedAt) {
		return &cached.MediaID, nil
	}

	hd, err := repo.GetHeadlineDesign(ctx, s.pool, designID)
	if err != nil {
		return nil, err
	}
	cat := ""
	if c.CategoryName != nil {
		cat = *c.CategoryName
	}
	spot := c.Spot
	if strings.TrimSpace(spot) == "" && c.IsSD {
		spot = ""
	}
	designed, err := platform.ApplyContentDesign(hd.Design, platform.ContentDesignInput{
		Title:      c.Title,
		Spot:       spot,
		Badge:      cat,
		Category:   cat,
		CoverMedia: c.CoverMediaID,
	})
	if err != nil {
		return nil, err
	}
	raw, err := platform.RenderDesign(designed, s.mediaRaw)
	if err != nil {
		return nil, err
	}
	stored, err := platform.SaveMedia(raw, "manset-kapak.jpg")
	if err != nil {
		return nil, err
	}
	uid := s.systemUserID(ctx)
	if uid <= 0 {
		return nil, nil // sistem kullanıcısı yok → medya kaydı yapılamaz, sessizce atla
	}
	sz := int64(len(raw))
	mid, err := repo.CreateMedia(ctx, s.pool, &repo.Media{
		UserID: uid, Kind: "image", Filename: "manset-kapak-" + strconv.FormatInt(c.ID, 10) + ".jpg", Path: stored,
		Mime: "image/jpeg", SizeBytes: &sz, Source: "render",
	})
	if err != nil {
		return nil, err
	}
	if err := repo.SaveContentDesignCover(ctx, s.pool, c.ID, designID, mid, c.UpdatedAt); err != nil {
		return nil, err
	}
	return &mid, nil
}
