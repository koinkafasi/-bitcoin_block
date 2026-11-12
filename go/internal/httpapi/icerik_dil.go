package httpapi

import (
	"context"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// dilDurumTR dil satırındaki durum etiketini verir.
func dilDurumTR(state string) string {
	switch state {
	case "yayinda":
		return "yayında"
	case "taslak":
		return "taslak"
	case "arsiv":
		return "arşiv"
	}
	return "çeviri yok"
}

func (s *Server) langRows(ctx context.Context, c repo.Content, isNew bool) []templkit.IcerikLang {
	enabled := s.i18n.Enabled()
	mevcut := map[string]repo.Content{}
	root := c.TranslationRootID()
	if !isNew {
		if trs, err := repo.TranslationsOf(ctx, s.pool, root); err == nil {
			for _, t := range trs {
				mevcut[t.Locale] = t
			}
		}
	}
	kaynak := c.Locale
	if kaynak == "" {
		kaynak = "tr"
	}

	var satirlar []templkit.IcerikLang
	for _, l := range enabled {
		satir := templkit.IcerikLang{
			Code:   l.Code,
			Native: l.NativeName,
			Label:  l.NativeName,
		}
		if l.Code == kaynak {
			satir.Active = true
			satir.State = c.Status
			satir.StateTR = dilDurumTR(c.Status)
			satir.ContentID = c.ID
			satir.URL = "/admin/icerikler/" + itoa64(c.ID)
			satirlar = append(satirlar, satir)
			continue
		}
		if t, ok := mevcut[l.Code]; ok {
			satir.ContentID = t.ID
			satir.State = t.Status
			satir.StateTR = dilDurumTR(t.Status)
			satir.URL = publicArticleURL(t.Locale, t.DisplaySlug(), t.ID)
			satir.SourceID = root
			satir.Once = true
		} else {
			satir.State = "yok"
			satir.StateTR = dilDurumTR("yok")
			satir.SourceID = root
		}
		satirlar = append(satirlar, satir)
	}

	// Mevcut çeviriler öne (kaynak ilk satırda kalır).
	onlar := make([]templkit.IcerikLang, 0, len(satirlar))
	var sonra []templkit.IcerikLang
	for _, r := range satirlar {
		if r.Active || r.Once {
			onlar = append(onlar, r)
		} else {
			sonra = append(sonra, r)
		}
	}
	return append(onlar, sonra...)
}

// ceviriBaglam çeviri düzenleme bandı için kaynak bilgisini toplar.
func (s *Server) ceviriBaglam(ctx context.Context, c repo.Content) (kaynakID, baslik, spot, govde string) {
	if c.TranslationOf == nil {
		return "", "", "", ""
	}
	k, err := repo.GetContent(ctx, s.pool, *c.TranslationOf)
	if err != nil {
		return "", "", "", ""
	}
	if len(k.Detail) > 0 {
		govde = string(s.blocksToHTML(k.Detail, nil))
	}
	return itoa64(k.ID), k.Title, k.Spot, govde
}

// dilAdiTR dil kodunun yerel adını verir.
func (s *Server) dilAdiTR(code string) string {
	if s.i18n == nil {
		return code
	}
	for _, l := range s.i18n.Enabled() {
		if l.Code == code {
			return l.NativeName
		}
	}
	return strings.ToUpper(code)
}
