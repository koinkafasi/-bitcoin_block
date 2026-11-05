package httpapi

import (
	"context"

	"theblock/site/internal/repo"
)

// Video süresi kartlarda (Faz 21)
//
// `contents.video_seconds` yalnız video içeriklerinde anlamlıdır ve çoğu liste
// sorgusu onu okumaz. Burada, EKRANA BASILACAK kartlar için süreler TEK sorguda
// getirilip içerik kopyalarına yerinde yazılır. Böylece:
//   - kart şablonları doğrudan `.VideoSeconds` okur (şablon işlevi gerekmez,
//     hatta FormatDurationTR bile şablonda çağrılabilir),
//   - kart sayısı ne olursa olsun sorgu sayısı 1 kalır (N+1 yok).

// videoSureHedefleri verilen içerik dilimindeki videoları, sonradan doldurulmak
// üzere kimlik -> içerik işaretçisi haritasına toplar.
func videoSureHedefleri(items []repo.Content, out map[int64][]*repo.Content) {
	for i := range items {
		c := &items[i]
		if c.TypeKey != "video" || c.ID <= 0 || c.VideoSeconds != nil {
			continue
		}
		out[c.ID] = append(out[c.ID], c)
	}
}

// fillVideoDurations bir içerik diliminin video sürelerini yerinde doldurur
// (kategori/arşiv/arama listeleri). Dilim yerinde (in-place) güncellenir.
func (s *Server) fillVideoDurations(ctx context.Context, items []repo.Content) {
	hedef := map[int64][]*repo.Content{}
	videoSureHedefleri(items, hedef)
	s.sureleriUygula(ctx, hedef)
}

// fillBlockVideoDurations blok kartlarındaki video sürelerini doldurur
// (manşet, galeri, canlı yayın, kategori paneli/listesi blokları).
func (s *Server) fillBlockVideoDurations(ctx context.Context, blocks []homeBlock) {
	hedef := map[int64][]*repo.Content{}
	for i := range blocks {
		b := &blocks[i]
		videoSureHedefleri(b.Items, hedef)
		videoSureHedefleri(b.Slides, hedef)
		for j := range b.CatSections {
			videoSureHedefleri(b.CatSections[j].Items, hedef)
		}
		for j := range b.CatBlocks {
			cb := &b.CatBlocks[j]
			videoSureHedefleri(cb.Items, hedef)
			if cb.Lead.TypeKey == "video" && cb.Lead.ID > 0 && cb.Lead.VideoSeconds == nil {
				hedef[cb.Lead.ID] = append(hedef[cb.Lead.ID], &cb.Lead)
			}
		}
	}
	s.sureleriUygula(ctx, hedef)
}

// sureleriUygula tek sorguyla süreleri getirip hedef içeriklere yazar.
func (s *Server) sureleriUygula(ctx context.Context, hedef map[int64][]*repo.Content) {
	if len(hedef) == 0 {
		return
	}
	ids := make([]int64, 0, len(hedef))
	for id := range hedef {
		ids = append(ids, id)
	}
	sureler := repo.VideoDurations(ctx, s.pool, ids)
	for id, ptrler := range hedef {
		sec, ok := sureler[id]
		if !ok || sec <= 0 {
			continue
		}
		for _, p := range ptrler {
			v := sec
			p.VideoSeconds = &v
		}
	}
}
