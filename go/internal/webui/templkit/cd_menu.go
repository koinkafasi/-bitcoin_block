package templkit

import "strings"

// CoinDesk mega menüsü — veri modeli.
//
// Yapı UYDURULMADI: `docs/COINDESK-SPEC.md` §3 "Üst gezinme (birebir)"
// bölümündeki ölçümden birebir çıkarıldı. Başlıklar, sıra ve gruplama
// ölçülen hâliyle korunur; yalnız URL'ler §4'teki desene bağlanır.
//
// Tasarım kararı: menü İÇERİĞİ burada sabit (ölçülen yapı), menü
// GÖRÜNÜMÜ cd-menu.css'te. cdFooterColumns ile aynı desen.

// CDMenuItem bir açılır menü satırı.
//
// Kind alanı satırın rolünü taşır; şablon buna göre biçim verir:
//
//	""        normal bağlantı
//	"baslik"  grup başlığı (tıklanmaz)
//	"tumu"    "tümünü gör" satırı (ayraçtan sonra, vurgulu)
type CDMenuItem struct {
	Label string
	Href  string
	Kind  string
}

// CDMenuColumn açılır menüdeki tek sütun.
type CDMenuColumn struct {
	Items []CDMenuItem
}

// CDMenuGroup üst gezinmedeki tek başlık ve (varsa) açılır menüsü.
//
// Columns boşsa başlık düz bağlantıdır (Research, Events — ölçümde bu
// ikisinde açılır menü yok).
type CDMenuGroup struct {
	Label   string
	Href    string
	Columns []CDMenuColumn
}

// cdMenu üst gezinmenin tamamı (COINDESK-SPEC.md §3).
var cdMenu = []CDMenuGroup{
	{
		Label: "News", Href: "/latest-crypto-news",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Latest Crypto News", Href: "/latest-crypto-news"},
				{Label: "Markets", Href: "/markets"},
				{Label: "Finance", Href: "/business"},
				{Label: "Tech", Href: "/tech"},
				{Label: "Policy", Href: "/policy"},
				{Label: "Focus", Href: "/focus"},
				{Label: "Opinion", Href: "/opinion"},
			}},
			{Items: []CDMenuItem{
				{Label: "Newsletters", Kind: "baslik"},
				{Label: "CoinDesk Headlines", Href: "/newsletters/coindesk-headlines"},
				{Label: "Crypto Daybook Americas", Href: "/newsletters/crypto-daybook-americas"},
				{Label: "State of Crypto", Href: "/newsletters/state-of-crypto"},
				{Label: "Crypto Long & Short", Href: "/newsletters/crypto-long-short"},
				{Label: "Crypto for Advisors", Href: "/newsletters/crypto-for-advisors"},
				{Label: "Tokenization Weekly", Href: "/newsletters/tokenization-weekly"},
				{Label: "CoinDesk Insider", Href: "/newsletters/coindesk-insider"},
			}},
			{Items: []CDMenuItem{
				{Label: "Crypto News", Kind: "baslik"},
				{Label: "Bitcoin News", Href: "/tag/bitcoin"},
				{Label: "Ethereum News", Href: "/tag/ethereum"},
				{Label: "XRP News", Href: "/tag/xrp"},
				{Label: "Solana News", Href: "/tag/solana"},
				{Label: "News Analysis", Href: "/news-analysis"},
				{Label: "Press Releases", Href: "/press-release"},
			}},
		},
	},
	{
		Label: "Video", Href: "/video",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Videos", Href: "/video"},
				{Label: "CoinDesk Daily", Href: "/video/coindesk-daily"},
				{Label: "Spotlight", Href: "/video/spotlight"},
				{Label: "Markets Outlook", Href: "/video/markets-outlook"},
			}},
			{Items: []CDMenuItem{
				{Label: "Privacy in Motion", Href: "/video/privacy-in-motion"},
				{Label: "Public Keys", Href: "/video/public-keys"},
				{Label: "Shorts", Href: "/video/shorts"},
				{Label: "The Policy Show", Href: "/video/the-policy-show"},
			}},
		},
	},
	{
		Label: "Prices", Href: "/price",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Latest Crypto Prices", Kind: "baslik"},
				{Label: "Bitcoin (BTC)", Href: "/price/btc"},
				{Label: "Ethereum (ETH)", Href: "/price/eth"},
				{Label: "XRP (XRP)", Href: "/price/xrp"},
				{Label: "Solana (SOL)", Href: "/price/sol"},
				{Label: "Hyperliquid (HYPE)", Href: "/price/hype"},
				{Label: "See all crypto prices", Href: "/price", Kind: "tumu"},
			}},
			{Items: []CDMenuItem{
				{Label: "Latest US Stock Prices", Kind: "baslik"},
				{Label: "S&P 500 (SPY)", Href: "/price/stock/spy"},
				{Label: "Bullish (BLSH)", Href: "/price/stock/blsh"},
				{Label: "Circle (CRCL)", Href: "/price/stock/crcl"},
				{Label: "Coinbase (COIN)", Href: "/price/stock/coin"},
				{Label: "Strategy (MSTR)", Href: "/price/stock/mstr"},
				{Label: "See all US stock prices", Href: "/price/stock", Kind: "tumu"},
			}},
		},
	},
	// Ölçümde bu ikisinde açılır menü YOK; düz bağlantı.
	{Label: "Research", Href: "/research"},
	{Label: "Events", Href: "/events"},
	{
		Label: "Data & Indices", Href: "/index",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "CoinDesk 20 (CD20)", Href: "/index/cd20"},
				{Label: "Bitcoin Index (XBX)", Href: "/index/xbx"},
				{Label: "Ether Index (ETX)", Href: "/index/etx"},
				{Label: "See all indices", Href: "/index", Kind: "tumu"},
			}},
		},
	},
	{
		Label: "Sponsored", Href: "/sponsored-content",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Thought Leadership", Href: "/sponsored-content"},
				{Label: "Press Releases", Href: "/press-release"},
			}},
		},
	},
}

// cdMenuGroups şablona menüyü verir. Şu an sabit; taksonomiden
// beslenmesi istenirse yalnız bu fonksiyon değişir, şablon değişmez.
func cdMenuGroups() []CDMenuGroup { return cdMenu }

// cdMenuID açılır menünün aria-controls/id değerini üretir.
// cdSlug yalnız harf/rakam bıraktığı için "Data & Indices" → "dataindices".
func cdMenuID(label string) string { return "cd-mega-" + cdSlug(label) }

// cdHasMenu başlığın açılır menüsü var mı.
func cdHasMenu(g CDMenuGroup) bool { return len(g.Columns) > 0 }

// cdMenuAktif, üst başlığın aktif olup olmadığını s.Nav'dan türetir.
//
// CDShell'de "geçerli yol" alanı yok; aktiflik, handler'ın s.Nav içinde
// işaretlediği bağlantıdan okunur. Böylece mega menü, CDShell'e yeni alan
// eklemeden ve handler'lara dokunmadan mevcut aria-current davranışını
// korur. Karşılaştırma son ek üzerinden yapılır: s.Nav yolları dil ön eki
// taşıyabilir ("/tr/markets"), menü tanımı taşımaz ("/markets").
func cdMenuAktif(nav []CDNavLink, href string) bool {
	if href == "" {
		return false
	}
	for _, l := range nav {
		if !l.Active || l.Href == "" {
			continue
		}
		if l.Href == href || strings.HasSuffix(l.Href, href) {
			return true
		}
	}
	return false
}
