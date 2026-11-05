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
