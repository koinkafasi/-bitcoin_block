package httpapi

import "theblock/site/internal/webui/templkit"

func cdGeciciDusunce() *templkit.CDDusunce {
	return &templkit.CDDusunce{
		Baslik:  "Regulated Cold Wallets with Instant Transaction Signing",
		Sponsor: "Hex Trust",
		Image:   "https://cdn.sanity.io/images/s3y3vcno/production/041527e9627e692f4d3e0da86f22b21b127239f9-1200x630.png?auto=format&w=768&q=75",
		Href:    "https://www.coindesk.com/sponsored-content/regulated-cold-wallets-with-instant-transaction-signing",
	}
}

func cdGeciciArastirma() []templkit.CDArastirma {
	return []templkit.CDArastirma{
		{
			Metrik:   "50.8%",
			Baslik:   "APAC-to-APAC corridor share",
			Aciklama: "Share of tracked stablecoin corridor volume that stayed within Asia-Pacific",
			Rapor:    "Global Markets, Asian Flows: Stablecoins and Tokenized Assets in 2026",
			Etiketler: []string{
				"RWA", "Macro & Market Trends", "Stablecoins",
			},
			Href: "https://www.coindesk.com/research/global-markets-asian-flows-stablecoins-and-tokenized-assets-in-2026",
		},
		{
			Metrik:   "3x Faster",
			Baslik:   "Zcash blok süresi 75 sn'den 25 sn'ye indi",
			Aciklama: "NU7 yükseltmesiyle Zcash blok hedefi 25 saniyeye düştü; özel ödemeler hızlanıyor.",
			Rapor:    "Zcash's NU7 Upgrade Goes Live as Developers Launch Quantum-Resistance Effort",
			Etiketler: []string{
				"Layer 1 Blockchains", "Macro & Market Trends",
			},
			Href: "https://www.coindesk.com/research",
		},
	}
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
