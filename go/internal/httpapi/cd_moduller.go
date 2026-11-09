package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// cdModuller ana sayfanın modüler yerleşimini düzen kaydından okur.
//
// KAYNAK: `page_layouts` (page_key = "home"), sayfa tasarımcısının zaten
// yazdığı yer. Yeni tasarımın alanları `cd_*` tipleriyle aynı tabloda
// duruyor; ayrı bir tablo/ayar eklemek iki kaynak yaratırdı.
//
// DAVRANIŞ:
//   - düzen YOKSA ya da hiç `cd_*` bloğu yoksa → nil döner ve şablon
//     VARSAYILAN sırayı basar (mevcut görünüm birebir korunur),
//   - varsa → yalnız `cd_*` blokları, GÖRÜNÜRLÜK ve SIRA korunarak döner,
//   - tanınmayan `cd_*` tipi sessizce atlanır (yazım kayması sayfayı
//     kırmasın; `TestCDModulTipleriPaletle` paleti kilitliyor).
func (s *Server) cdModuller(ctx context.Context, r *http.Request) []templkit.CDModul {
	site := s.siteByDomain(ctx, r.Host)
	raw, err := repo.GetLayout(ctx, s.pool, site.ID, "home")
	if err != nil || len(raw) == 0 {
		return nil
	}
	var bloklar []struct {
		Type    string `json:"type"`
		Visible *bool  `json:"visible"`
		// Count üst düzeyde DEĞİL, `opts` içinde yazılıyor (tasarımcı
		// biçimi). İkisi de okunur: ileride üst düzeye taşınırsa sessizce
		// sıfıra düşmesin.
		Count int `json:"count"`
		Opts  struct {
			Count int `json:"count"`
		} `json:"opts"`
	}
	if err := json.Unmarshal(raw, &bloklar); err != nil {
		return nil
	}
	var out []templkit.CDModul
	for _, b := range bloklar {
		if !templkit.CDModulGecerli(b.Type) {
			continue
		}
		// `visible` alanı yoksa (eski kayıt) GÖRÜNÜR kabul edilir: tasarımcı
		// yeni blok eklerken alanı yazmayabilir; varsayılanın "gizli" olması
		// kullanıcının eklediği modülün kaybolması demek olurdu.
		goster := true
		if b.Visible != nil {
			goster = *b.Visible
		}
		adet := b.Count
		if adet == 0 {
			adet = b.Opts.Count
		}
		out = append(out, templkit.CDModul{Tip: b.Type, Goster: goster, Adet: adet})
	}
	return out
}
