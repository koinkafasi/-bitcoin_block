package templkit

import "html/template"

// CDKurumsalSayfa.SafeHTML alanı için yardımcı tip.
// templ dosyasında `templ.SafeHTML` yerine `template.HTML` kullanılıyor:
// templ paketinin kendi adıyla çakışmasın diye burada yeniden dışa açılır.
type SafeHTML = template.HTML
