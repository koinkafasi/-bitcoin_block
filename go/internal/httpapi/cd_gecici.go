package httpapi

import "theblock/site/internal/webui/templkit"

// GEÇİCİ DIŞ MODÜL VERİSİ — CoinDesk ana sayfasından alınan GERÇEK
// içerikler (6 Kasım 2026 ölçümü).
//
// NEDEN GEÇİCİ: sitemizde video, sponsorlu içerik, basın bülteni ve
// metrikli araştırma kartı üreten hat henüz yok (yayında 0). Kullanıcı
// isteği: "şimdilik CoinDesk linkleriyle doldur, sonra güncellenecek".
// Bu yüzden her kart kaynağına (coindesk.com) gider ve kaynak adı
// görünür; uydurma başlık/metrik basılmıyor.
//
// KALDIRMA: ilgili içerik türü kendi arşivimizde yayına girdiğinde bu
// dosya silinir; şablonlar aynı kalır, veri repo katmanından gelir.

func cdGeciciDusunce() *templkit.CDDusunce {
	return &templkit.CDDusunce{
		Baslik:  "Regulated Cold Wallets with Instant Transaction Signing",
		Sponsor: "Hex Trust",
		Image:   "https://cdn.sanity.io/images/s3y3vcno/production/041527e9627e692f4d3e0da86f22b21b127239f9-1200x630.png?auto=format&w=768&q=75",
		Href:    "https://www.coindesk.com/sponsored-content/regulated-cold-wallets-with-instant-transaction-signing",
	}
}

func cdGeciciVideolar() []templkit.CDVideo {
	return []templkit.CDVideo{
		{
			Etiket: "CoinDesk Interviews",
			Baslik: "CZ on Binance's $6.3B hack",
			Image:  "https://cdn.jwplayer.com/v2/media/OqtIIfvT/poster.jpg?width=720",
			Href:   "https://www.coindesk.com/video/cz-on-binances-63b-hack",
			Sure:   "15:22",
		},
		{
			Etiket: "CoinDesk Videos",
			Baslik: "Is the AI boom keeping bitcoin from its next bull run?",
			Image:  "https://cdn.sanity.io/images/s3y3vcno/production/ca105ab8ae95548a82008216fc07359b61ef94e6-720x406.jpg?auto=format&w=768&q=75",
			Href:   "https://www.coindesk.com/video/is-the-ai-boom-keeping-bitcoin-from-its-next-bull-run",
		},
		{
			Etiket: "CoinDesk Videos",
			Baslik: "Hunter Biden Says Democrats Have It Wrong on Crypto",
			Image:  "https://cdn.sanity.io/images/s3y3vcno/production/8600a1ead0abe1fd4f75a165dc7ffef9a78e9e54-720x406.jpg?auto=format&w=768&q=75",
			Href:   "https://www.coindesk.com/video/hunter-biden-says-democrats-have-it-wrong-on-crypto",
		},
	}
}

func cdGeciciArastirma() *templkit.CDArastirma {
	return &templkit.CDArastirma{
		Metrik:   "50.8%",
		Baslik:   "APAC-to-APAC corridor share",
		Aciklama: "Share of tracked stablecoin corridor volume that stayed within Asia-Pacific",
		Rapor:    "Global Markets, Asian Flows: Stablecoins and Tokenized Assets in 2026",
		Etiketler: []string{
			"RWA", "Macro & Market Trends", "Stablecoins",
		},
		Href: "https://www.coindesk.com/research/global-markets-asian-flows-stablecoins-and-tokenized-assets-in-2026",
	}
}

func cdGeciciKlipler() []templkit.CDKlip {
	return []templkit.CDKlip{{
		Baslik: "Why CZ expected more withdrawals post-hack",
		Image:  "https://cdn.sanity.io/images/s3y3vcno/production/a175f33097bf30b8d3bbb3c969f798c43972c797-720x1280.jpg?w=1080&h=1920&auto=format",
		Href:   "https://www.coindesk.com/videos/shorts",
	}}
}

func cdGeciciBasin() []templkit.CDBasin {
	return []templkit.CDBasin{
		{
			Baslik:  "Archblock LLC, et al. Notice of Deadline for Filing Proofs of Claim, Including Claims Asserted Under Section 503(b)(9) of the Bankruptcy Code",
			Sponsor: "Archblock LLC",
			Href:    "https://www.coindesk.com/press-release/2026/10/21/archblock-llc-et-al-notice-of-deadline-for-filing-proofs-of-claim-including-claims-asserted-under-section-503-b-9-of-the-bankruptcy-code",
		},
		{
			Baslik:  "CertiK Announces Brand Upgrade to Reflect Its Expanded Role in Digital Asset Risk Management",
			Sponsor: "CertiK",
			Href:    "https://www.coindesk.com/press-release/2026/10/19/certik-announces-brand-upgrade-to-reflect-its-expanded-role-in-digital-asset-risk-management",
		},
		{
			Baslik:  "DAG Wealth Announces XRP Lending Strategy",
			Sponsor: "DAG Wealth",
			Href:    "https://www.coindesk.com/press-release/2026/10/08/dag-wealth-announces-xrp-lending-strategy",
		},
	}
}
