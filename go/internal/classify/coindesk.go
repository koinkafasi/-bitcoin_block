package classify

import "strings"

// CoinDesk bölüm sınıflandırması (bitcosar).
//
// NEDEN AYRI: `classify` theblock'un Türkçe taksonomisine (gundem, ekonomi,
// piyasalar…) eşliyor. Kripto portalında bölümler CoinDesk'in bölümleri
// (docs/COINDESK-SPEC.md §4). Aynı paketi iki taksonomiye zorlamak yerine
// ayrı ve açık bir eşleme tutuluyor.
//
// GİRDİ: kaynağın GERÇEK RSS <category> değerleri. Ölçüm (5 Kasım 2026,
// 179 içerik): CoinDesk "Markets/Policy/Regulation/Tech", Decrypt
// "Bitcoin/DeFi/NFT", The Block "Markets/Companies" gönderiyor.
//
// KURAL: en özgül bölüm kazanır (policy > tech > business > markets).
// Hiçbiri eşleşmezse "markets" — kripto portalında varsayılan bölüm.
func CoinDeskSection(etiketler []string, baslik, spot string) string {
	metin := strings.ToLower(strings.Join(etiketler, " ") + " " + baslik + " " + spot)
	metin = turkceKucult(metin)

	// Sıra önemli: ilk eşleşen kazanır (en özgülden en gene).
	kurallar := []struct {
		slug  string
		words []string
	}{
		{"policy", []string{
			"policy", "regulation", "regulator", "government", "congress", "senate",
			"law", "lawsuit", "court", "legal", "sec ", "cftc", "doj", "tax",
			"legislation", "bill ", "sanction", "compliance", "etf approval",
			"düzenleme", "regülasyon", "yasa", "mahkeme", "dava", "vergi", "spk",
		}},
		{"tech", []string{
			"tech", "technology", "protocol", "blockchain", "upgrade", "hack",
			"exploit", "cybersecurity", "artificial intelligence", " ai ",
			"layer 2", "layer2", "rollup", "zero-knowledge", "zk-", "smart contract",
			"developer", "open source", "node", "consensus", "scaling", "ının",
			"teknoloji", "protokol", "güvenlik açığı", "siber",
		}},
		{"business", []string{
			"business", "finance", "company", "companies", "funding", "raise",
			"acquisition", "merger", "ipo", "earnings", "exchange", "bank",
			"institutional", "custody", "payment", "stablecoin issuer", "mining",
			"treasury", "etf inflow", "asset manager", "broker",
			"şirket", "borsa şirketi", "fon", "yatırım", "banka", "ödeme",
		}},
		{"opinion", []string{
			"opinion", "commentary", "column", "editorial", "görüş", "köşe",
		}},
		{"research", []string{
			"research", "report", "analysis deep", "araştırma", "rapor",
		}},
		{"markets", []string{
			"market", "markets", "price", "trading", "trader", "bitcoin", "btc",
			"ethereum", "eth", "xrp", "solana", "defi", "nft", "altcoin",
			"liquidity", "derivative", "futures", "options", "rally", "dump",
			"piyasa", "fiyat", "kripto", "alım satım",
		}},
	}
	for _, k := range kurallar {
		for _, w := range k.words {
			if strings.Contains(metin, w) {
				return k.slug
			}
		}
	}
	return "markets"
}

// turkceKucult Türkçe I/İ sorununu ele alır: strings.ToLower "I" harfini
// "i" yapar ama Türkçe'de "I" → "ı". Karşılaştırma için ikisini de "i"
// kabul etmek yeterli; burada yalnız arama metni üretiliyor.
func turkceKucult(s string) string {
	r := strings.NewReplacer("ı", "i", "İ", "i", "I", "i", "ş", "s", "ğ", "g",
		"ü", "u", "ö", "o", "ç", "c")
	return r.Replace(strings.ToLower(s))
}
