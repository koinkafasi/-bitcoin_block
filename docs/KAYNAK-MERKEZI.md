# Haber kaynak merkezi — kripto RSS/API kataloğu

Toplayıcının beslendiği kaynaklar. **Her satır 5 Kasım 2026'da bu makineden
fiilen test edildi**; çalışmayanlar da yazılı (neden + çözüm).

Ölçüm betiği: `deploy/scripts/kaynak-test.sh`

---

## A. Çalışan kaynaklar (doğrudan kullanılabilir)

| Kaynak | URL | Durum | Öğe | Not |
|---|---|---|---|---|
| **CoinDesk** | `https://www.coindesk.com/arc/outboundfeeds/rss/` | 200 · 30 KB | 25 | Arc XP feed; tam metin yok, spot var |
| **Decrypt** | `https://decrypt.co/feed` | 200 · 40 KB | 36 | Zengin içerik, görsel var |
| **The Block** | `https://www.theblock.co/rss.xml` | 200 · 34 KB | 20 | Sektörün en hızlı habercisi |
| **CryptoBriefing** | `https://cryptobriefing.com/feed/` | 200 · 34 KB | 30 | |
| **CryptoSlate** | `https://cryptoslate.com/feed/` | 200 · 17 KB | 10 | Sayfalama sınırlı |

## B. Bağlantı sıfırlanan kaynaklar (TLS parmak izi)

Bu siteler `curl`/Go varsayılan TLS'i ile **`Connection reset by peer`**
döndürüyor. Engelleme IP'ye değil **TLS ClientHello parmak izine** (JA3)
bakıyor; tarayıcı başlıkları yetmiyor.

| Kaynak | URL | Ölçüm |
|---|---|---|
| Cointelegraph | `https://cointelegraph.com/rss` | `curl (35) reset` |
| Bitcoin Magazine | `https://bitcoinmagazine.com/feed` | `curl (35) reset` |
| BeInCrypto | `https://beincrypto.com/feed/` | `curl (35) reset` |
| NewsBTC | `https://www.newsbtc.com/feed/` | bağlantı yok |
| Bitcoinist | `https://bitcoinist.com/feed/` | bağlantı yok |
| CryptoPotato | `https://cryptopotato.com/feed/` | bağlantı yok |
| AMBCrypto | `https://ambcrypto.com/feed/` | bağlantı yok |
| CoinGape | `https://coingape.com/feed/` | bağlantı yok |
| CoinJournal | `https://coinjournal.net/feed/` | bağlantı yok |
| Cryptonews | `https://cryptonews.com/news/feed/` | bağlantı yok |

**Çözüm (Faz 3'te):** Go tarafında `utls` (browser TLS fingerprint) veya
HTTP/2 + Chrome ALPN taklidi. `refraction-networking/utls` ile
`HelloChrome_Auto` kullanıldığında bu sitelerin çoğu açılıyor. Alternatif:
kaynak başına isteğe bağlı HTTP proxy (yalnız bu feed'ler için).

**Karar:** Faz 1'de A grubu (5 kaynak, ~120 öğe/saat) yeterli. B grubu
adaptör olarak yazılır, `enabled=false` seed edilir; utls gelince açılır.

## C. Resmî / kurumsal kaynaklar (yüksek güven)

| Kaynak | URL / yöntem | Not |
|---|---|---|
| SEC basın | `https://www.sec.gov/news/pressreleases.rss` | UA'da e-posta zorunlu |
| SEC EDGAR (şirket) | `https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=…` | oran sınırı 10 istek/sn |
| Federal Reserve | `https://www.federalreserve.gov/feeds/press_all.xml` | |
| ECB | `https://www.ecb.europa.eu/rss/press.html` | |
| Binance duyuru | `https://www.binance.com/bapi/composite/v1/public/cms/article/list/query?type=1&pageNo=1&pageSize=20` | JSON |
| Coinbase blog | `https://www.coinbase.com/blog/rss.xml` | |
| Tether | `https://tether.to/en/news/feed/` | |
| SPK (TR) | `https://spk.gov.tr/rss` | Türkçe regülasyon |

## D. Fiyat / piyasa verisi

| Kaynak | Uç | Sınır |
|---|---|---|
| CoinGecko | `/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc` | ücretsiz 10–30/dk |
| Binance | `/api/v3/ticker/24hr` | 1200 ağırlık/dk |
| Coinbase | `/products/{pair}/ticker` | 10/sn |
| Alternative.me | `/fng/` (Korku/Açlık endeksi) | günlük |
| DefiLlama | `/protocols`, `/chains` | ücretsiz |

## E. Toplama kuralları (hukuk + nezaket)

1. **Tam metin yayınlanmaz.** Kaynaktan gelen `description`/`summary`
   alanı **özet** olarak kullanılır; gövde AI ile yeniden yazılır ve
   **kaynağa atıf** verilir (`Kaynak: CoinDesk`).
2. `robots.txt`'e uyulur; feed URL'leri zaten izinli.
3. Kaynak başına **en az 15 dakika** aralık + `ETag`/`If-Modified-Since`.
4. `429`/`503` → üstel geri çekilme (1→2→4→8 dk, en çok 1 saat).
5. Her çekim **ham olarak** object storage'a yazılır (denetim + yeniden işleme).
6. Kaynak kaldırma talebi (`iletişim`) → `enabled=false` + içerik `arsiv`.

## F. Tekilleştirme / kümeleme

Aynı haber 5 kaynakta birden çıkıyor. Strateji:

1. **URL parmak izi**: kanonik URL → SHA-256 (birebir kopya)
2. **Başlık parmak izi**: normalize (küçük harf, noktalama sil) + simhash
   → Hamming mesafesi ≤ 3 ise aynı
3. **Embedding** (Faz 4): `pgvector`, kosinüs ≥ 0.92 → aynı hikâye
4. Küme → `stories` satırı; kaynaklar `story_articles` altında listelenir.
   Sayfada **tek kanonik URL**, altında "Bu haberi yazan kaynaklar".

## G. Seed

`go/migrations/0221_kripto_kaynaklar.up.sql` — A + C + D grupları
`feeds` tablosuna yazılır (`aggregator=true`, `trust` 60–95).

---

## Spam / reklam ayıklaması — ölçüm ve karar (6 Kasım 2026)

Dört canlı kaynak indirildi, **97 öğe** tanıtım işaretleri için tarandı.

| kaynak | öğe | PR/sponsor yolu | sponsor ifşası | teşvik/scam | affiliate |
|---|---|---|---|---|---|
| cryptobriefing | 30 | 0 | 1* | 0 | 0 |
| cryptoslate | 10 | 0 | 0 | 0 | 0 |
| decrypt | 37 | 0 | 0 | 0 | 0 |
| theblock | 20 | 0 | 0 | 0 | 0 |

\* Tek isabet **yanlış pozitifti**: *"North Korea reportedly rewards top
hackers with Pyongyang apartments after Binance heist"* — gerçek haber.
Onu yakalayan geniş desen kaldırıldı ve regresyon testi yazıldı.

**Sonuç: mevcut kaynaklarda spam ve reklam pratikte yok.** Dördü de ciddi
yayın; beslemeleri temiz. Buna rağmen kapı yazıldı
(`internal/repo/reklam.go`) — bugün hiçbir şeyi düşürmüyor, amacı daha
gürültülü bir kaynak eklendiğinde hazır olmak (basın bülteni toplayıcısı,
borsa blogu, Türkçe kripto siteleri).

### Kapı NİYE dar — ikinci ölçüm

Aynı 97 öğe projenin mevcut `kriptoGucluTerimler` listesiyle de süzüldü:

| kaynak | öğe | alakalı | **alakasız** | oran |
|---|---|---|---|---|
| cryptobriefing | 30 | 30 | 0 | 0% |
| cryptoslate | 10 | 10 | 0 | 0% |
| decrypt | 37 | 21 | **16** | **43%** |
| theblock | 20 | 10 | **10** | **50%** |
| **toplam** | 97 | 71 | **26** | **27%** |

Düşenlere bakıldığında hepsi **gerçek kripto haberi**:

- *"Zcash's major network upgrade NU7 goes live on mainnet"*
- *"Ark Invest buys more Coinbase as shares plunge 10%"*
- *"Hyperliquid Strategies files to become CFTC-registered FCM"*

Terim listesinde Zcash, Coinbase, Hyperliquid, Kalshi yok. Yani **alaka
kapısı toplayıcılara uygulanırsa meşru haberin %27'si kaybolur.** Bu,
kapının yalnız resmî kurum kaynaklarıyla sınırlı tutulmasının (alaka.go)
ölçülmüş gerekçesidir.

Dolayısıyla reklam kapısı **alaka filtresi değildir** ve öyle
genişletilmemelidir. `reklam_test.go` içindeki `TestAlakaFiltresiDegil`
bunu sabitliyor: kripto dışı ama meşru haber (petrol fiyatı, Meta davası)
geçmek zorunda.

### Kapının kullandığı sinyaller

Güvenilirlik sırasıyla:

1. **Adres yolu** — `/press-release`, `/sponsored`, `/partner-content`,
   `/advertorial`, `/paid-post`, `/basin-bulteni`, `/sponsorlu`…
   En güvenilir: yayıncı kendi URL şemasında ayırıyor.
2. **Yönlendirme parametresi** — `?ref=`, `aff=`, `affiliate=`,
   `referral=`, `invite=`, `promo=`.
   `utm_campaign` **bilinçli olarak yok**: meşru haber bağlantılarında da
   bulunuyor, ayırt etmiyor (test: `TestUtmCampaignReklamDegil`).
3. **Açık sponsor ifşası** — tam ifadeler ("this article is sponsored",
   "sponsorlu içerik"…), başlıkta ya da gövdenin **ilk 600 baytında**.
   `"in partnership with"` **çıkarıldı**: gerçek haberde geçiyor
   ("Samsung Wallet, Solana ile ortaklıkla…").
   `"press release"` yalnız **yolda** aranır; gövdede geçmesi haberi
   reklam yapmaz ("şirket basın bülteninde açıkladı").
4. **Başlıkta teşvik kalıbı** — "guaranteed returns", "use code",
   "claim your free", "next 100x gem", "garantili kazanç"…
   Tek başına **"airdrop" / "presale" / "token sale" YOK**: bunlar gerçek
   kripto haberinin konusu (test: `TestAirdropPresaleHaberiIsaretlenmez`).

Kapı tetiklendiğinde öğe `crawled_items`'a yazılır (her turda yeniden
denenmesin) ve besleme özetine **gerekçesiyle** not düşülür:
`reklam kapısı: 3 öğe tanıtım (adres yolu: /press-release)`.

### Yan bulgu — CoinDesk feed'i bozuk DEĞİL

Ölçüm sırasında CoinDesk adresi `308` döndü ve 0 öğe göründü; önce bozuk
sanıldı. **Doğrusu:** `.../rss/` adresi `.../rss`'e kalıcı yönlendirme
yapıyor ve izlenince `200` ile 25 öğe geliyor. Toplayıcının istemcisi
(`platform.SafeFetchClient`) yönlendirmeyi **izliyor** — en çok 3, her
hedefi SSRF'e karşı doğrulayarak. Ölçüm betiği `curl -L` kullanmadığı
için yanıltmıştı; katalogdaki satır doğru.

İsteğe bağlı küçük iyileştirme: seed'deki adresten sondaki eğik çizgi
atılırsa (`0111_kripto_kaynaklar.up.sql`) her tur bir HTTP gidiş-dönüşü
kazanılır. Davranış değişmez.
