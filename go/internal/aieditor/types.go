package aieditor

import "time"

// Section sonuç panelinde başlıklı metin bloğudur.
type Section struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Alternative başlık/açı gibi seçilebilir alternatiflerdir.
type Alternative struct {
	Text string `json:"text"`
	Note string `json:"note,omitempty"`
}

// Claim doğrulama sonucudur. Status Türkçe etikettir; DB'de de bu şekilde
// saklanır (supported -> Destekleniyor).
type Claim struct {
	Text        string   `json:"text"`
	Excerpt     string   `json:"excerpt,omitempty"`
	Status      string   `json:"status"`
	Evidence    string   `json:"evidence,omitempty"`
	Explanation string   `json:"explanation,omitempty"`
	Action      string   `json:"action,omitempty"`
	Sources     []string `json:"sources,omitempty"`
}

// Source araştırmada kullanılan gerçek kaynaktır.
type Source struct {
	Ref       string     `json:"ref,omitempty"`
	Title     string     `json:"title"`
	URL       string     `json:"url"`
	Publisher string     `json:"publisher,omitempty"`
	Snippet   string     `json:"snippet,omitempty"`
	Content   string     `json:"-"` // model bağlamı; kullanıcıya gösterilmez
	Published *time.Time `json:"published_at,omitempty"`
}

// Result AI işinin kullanıcıya gösterilen ve onay bekleyen sonucudur.
// Hiçbir alan doğrudan contents tablosuna yazılmaz.
type Result struct {
	Action       string            `json:"action"`
	Title        string            `json:"title,omitempty"`
	Sections     []Section         `json:"sections,omitempty"`
	Alternatives []Alternative     `json:"alternatives,omitempty"`
	Claims       []Claim           `json:"claims,omitempty"`
	Sources      []Source          `json:"sources,omitempty"`
	Fields       map[string]string `json:"fields,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	// Quotes doğrudan alıntılar (ses taslağı). Alıntı kapısı bunları kaynak
	// transkripte karşı doğrular.
	Quotes []string `json:"quotes,omitempty"`
	// UnverifiedQuotes kaynakta bulunamayan alıntılar (kapı sonucu).
	UnverifiedQuotes []string `json:"unverified_quotes,omitempty"`
	// UnverifiedNumbers kaynakta bulunamayan sayılar (olgu kapısı sonucu).
	UnverifiedNumbers []string `json:"unverified_numbers,omitempty"`
	// QuoteSources alıntıların kaynak segmentleri (Faz 124).
	QuoteSources []QuoteSource `json:"quote_sources,omitempty"`
	// MediaID alıntıların dinlenebileceği ses kaydı.
	MediaID  int64  `json:"media_id,omitempty"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

// JobView handler/template için iş + sonuç görünümüdür.
type JobView struct {
	JobID  int64
	Result Result
}
