package repo

import "strings"

var kriptoGucluTerimler = []string{
	"crypto", "cryptocurrenc", "crypto-asset", "crypto asset",
	"bitcoin", "btc", "ethereum", "ether", " xrp", "solana", "dogecoin",
	"stablecoin", "altcoin", "memecoin", "defi", "decentralized finance",
	"nft", "web3", "blockchain", "distributed ledger", "digital asset",
	"virtual asset", "vasp", "casp", "cbdc", "digital euro", "digital dollar",
	"tokeniz", "tokenis", "crypto exchange", "digital asset exchange",
	"crypto wallet", "cold wallet", "hot wallet", "mining pool",
	"kripto", "kripto para", "dijital varlik", "dijital para", "blok zinciri",
}

func kriptoAlakali(metin string) bool {
	m := kriptoNormalize(metin)
	for _, t := range kriptoGucluTerimler {
		if strings.Contains(m, kriptoNormalize(t)) {
			return true
		}
	}
	return false
}

// kriptoNormalize karşılaştırma biçimi: küçük harf + Türkçe aksan sadeleştirme.
func kriptoNormalize(s string) string {
	s = strings.ToLower(s)
	return strings.NewReplacer(
		"ı", "i", "İ", "i", "ş", "s", "ğ", "g",
		"ü", "u", "ö", "o", "ç", "c", "â", "a", "î", "i",
	).Replace(s)
}

func kaynakAlakaGerekli(aggregator bool) bool { return !aggregator }
