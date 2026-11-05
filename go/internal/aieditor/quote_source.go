package aieditor

// QuoteSource alıntının kaynak transkript segmentidir (Faz 124).
type QuoteSource struct {
	Quote    string  `json:"quote"`
	StartMS  int64   `json:"start_ms"`
	EndMS    int64   `json:"end_ms"`
	Speaker  string  `json:"speaker,omitempty"`
	Verified bool    `json:"verified"`
	Overlap  float64 `json:"overlap,omitempty"`
}
