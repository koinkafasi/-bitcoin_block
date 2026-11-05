package repo

import "strings"

// Kripto alaka filtresi — resmî kaynaklar için.
//
// SORUN (ölçüm, 5 Kasım 2026): ana sayfada manşet
// "Federal Reserve Board issues enforcement actions with former employee of
// Northstar Bank…" idi. SEC/Fed/ECB her gün onlarca **kripto dışı** bülten
// yayımlıyor (banka yaptırımı, personel cezası, para politikası).
//
// KURAL:
//   - Toplayıcı kripto medyası (aggregator=true): CoinDesk, Decrypt, The Block…
//     yalnız kripto yayımlar → filtre YOK.
//   - Resmî kurum kaynakları (aggregator=false): SEC, Fed, ECB, SPK…
//     metinde KRİPTOYA ÖZGÜ terim YOKSA içerik yayına alınmaz.
//
// İKİNCİ ÖLÇÜM (ağ önemli): ilk sürümde `exchange`, `token`, `custody`,
// `mining`, `wallet` da listede vardı. Bunlar **geleneksel finansta** da
// geçiyor ("securities exchange", "custody of client assets"); SEC'in 25
// bülteninin 25'i kapıyı geçti — filtre işlevsizdi. Artık yalnız kriptoya
// özgü terimler var; "token" yerine "tokeniz/tokenis", "exchange" yerine
// "crypto exchange"/"digital asset exchange" aranıyor.

// kriptoGucluTerimler kriptoya ÖZGÜ terimler (küçük harf, normalize).
//
// Bir terimi kaçırmak gerçek bir kripto haberini kaybettirir; fazla geniş
// tutmak alakasız bülteni geçirir. Bu liste ikinci hatayı önlemek için dar.
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

// kriptoAlakali metin kriptoya özgü bir terim içeriyor mu?
//
// Türkçe karakterler normalize edilir (aksi hâlde "dijital varlık"
// eşleşmez). Terim listesi de aynı normalizasyondan geçirilir.
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

// kaynakAlakaGerekli kaynak için alaka kapısı uygulanmalı mı?
//
// `aggregator=true` → kripto medyası, kapı YOK.
// `aggregator=false` → resmî kurum, kapı VAR.
func kaynakAlakaGerekli(aggregator bool) bool { return !aggregator }
