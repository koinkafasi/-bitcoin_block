# CoinDesk yapı spesifikasyonu (ölçüm, 5 Kasım 2026)

Bu belge **uydurma değil, ölçüm**: `https://www.coindesk.com/` sayfasından,
`robots.txt`'ten ve CSS parçalarından çıkarıldı. bitcosar'ın hedef yapısı budur.

Ölçüm kaynağı: sayfa HTML'i (1,32 MB), `robots.txt`, CSS öbekleri
(`3683addgw5ehx.css`), sitemap dizinleri.

---

## 1. Teknoloji (gözlem)

| Katman | CoinDesk'te | bitcosar'da |
|---|---|---|
| Render | Astro (SSR/SSG) + Vercel | **Go + templ** (aynı çıktı, tek binary) |
| CDN | Vercel Edge + Cloudflare | **Cloudflare** (tunnel + cache) |
| Görsel CDN | Sanity (`cdn.sanity.io`) | kendi `image-svc` + R2/S3 |
| Fiyat verisi | CryptoCompare + kendi index'leri | CoinGecko/Binance + kendi önbellek |
| Video | JWPlayer | YouTube gömme / kendi HLS |
| Reklam | Playwire, OneTrust | AdSense + kendi yuvalar |

> CoinDesk'in Astro kullanması **tasarım hedefi** açısından bilgi; mimari
> tercihimizi değiştirmiyor (aynı SSR HTML'i Go üretiyor). Ayrıntı: `PLAN.md`.

---

## 2. Tasarım token'ları (CSS'ten çıkarıldı)

```
--bg            #ffffff        --fg           #000000
--surface-soft  #f8f8f8        --surface-alt  #f4f4f4
--border        #e9e9e9        --border-strong #d8d8d8
--muted         #696969        --muted-strong #4d4d4d
--dark          #262626        --dark-2       #2e3644
--link          #4d90fe        --link-hover   #3860be
--up            #68b631        --down         #ff0062
--warn          #ffd200
--font          "proxima-nova", system-ui, -apple-system, "Segoe UI", sans-serif
--theme-color   rgb(255,255,255)
```

Gözlem: **açık tema varsayılan**, siyah metin, ince gri ayırıcılar, mavi
bağlantı, yeşil/magenta fiyat yönü, sarı vurgu. Köşe yarıçapı küçük (0–4 px),
gölge yok, kartlar **çizgiyle** ayrılır (gölgeyle değil).

Tipografi ölçeği (CSS'te görülen): `12px`, `.875rem`, `1rem`, `1.125rem`
gövde; başlıklar 1.6–2.4rem arası, sıkı satır aralığı (`1.15–1.25`).

---

## 3. Üst gezinme (birebir)

```
News ▾        │ Video ▾      │ Prices ▾      │ Research │ Events │ Data & Indices ▾ │ Sponsored ▾
 Latest Crypto News            Videos            Latest Crypto Prices
 Markets                       CoinDesk Daily    Bitcoin (BTC)
 Finance                       Spotlight         Ethereum (ETH)
 Tech                          Markets Outlook   XRP (XRP)
 Policy                        Privacy in Motion SOL (SOL)
 Focus                         Public Keys…      Hyperliquid (HYPE)
 Newsletters ▾                 Shorts            See all crypto prices
 CoinDesk Headlines            The Policy…       Latest US Stock Prices
 Crypto Daybook Americas                         S&P 500 (SPY)
 State of Crypto                                 Bullish (BLSH)
 Crypto Long & Short                             Circle (CRCL)
 Crypto for Advisors                             Coinbase (COIN)
 Tokenization Weekly                             Strategy (MSTR)
 CoinDesk Insider                                See all US stock prices
 Crypto News
  Bitcoin News / Ethereum News / XRP News / Solana News
```

Sağ üst: arama, dil seçici (`en`), **Sign In / Sign Up**.

Altında **fiyat şeridi** (yapışkan):

```
[MARKETS LIVE | CRYPTO STOCKS]
CD20 $2,195.44 ▲3.77%   BTC $80,342.62 ▲3.48%   ETH $2,418.95 ▲5.61%
XRP $1.31 ▲2.46%   SOL $95.80 ▲3.39%   HYPE $83.28 ▼1.42%   TRX $0.31 ▼0.50%
```

---

## 4. Bölümler (sections) ve URL desenleri

| Bölüm | Nav adı | URL deseni |
|---|---|---|
| Markets | Markets | `/{locale}/markets/{YYYY}/{MM}/{DD}/{slug}` |
| Business | **Finance** | `/business/{tarih}/{slug}` |
| Tech | Tech | `/tech/{tarih}/{slug}` |
| Policy | Policy | `/policy/{tarih}/{slug}` |
| Focus | Focus | `/focus/{tarih}/{slug}` |
| Opinion | Opinion | `/opinion/{tarih}/{slug}` |
| News Analysis | — | `/news-analysis/{tarih}/{slug}` |
| Press Release | Press Releases | `/press-release/{tarih}/{slug}` |
| Sponsored | Thought Leadership | `/sponsored-content/{slug}` |
| Video | Video | `/video/{slug}` |
| Podcast | Podcasts | `/podcasts/{slug}` |
| Research | Research | `/research/{slug}` |
| Price | Prices | `/price`, `/price/{coin}`, `/price/stock/{ticker}` |
| Index | Data & Indices | `/index/{cd20…}` |
| Tag | — | `/tag/{tag}` |
| Latest | Latest Crypto News | `/latest-crypto-news` |
| Newsletter | Newsletters | `/newsletters/{slug}` |
| Live blog | — | aynı makale deseni + `canli` tipi |

**Kritik gözlem:** Sitenin **kendi** haberleri `/markets/2026/11/05/...`
deseninde; haber **toplayan** bir site için bu desen SEO'da tarih
kanonikliği sağlar (Google News `lastmod` ile uyumlu). bitcosar bunu
**birebir** kullanacak.

---

## 5. Diller (15 + varsayılan)

`en` (kök, öneksiz), `es`, `uk`, `ru`, `fil`, `it`, `fr`, `pt-br`, `de`,
`nl`, `ko`, `zh`, **`tr`**, `id`, `ja`

Desen: `https://www.coindesk.com/tr/...` — her dil için ayrı
`sitemap-index.xml` ve ayrı `robots.txt` yönlendirmesi.

bitcosar faz 1'de: **tr (kök) + en**. Sonra `de`, `es`, `ru`, `zh`, `ja`.

> Not: theblock'ta öğrenilen ders — **çevirisi olmayan dil hreflang'e
> yazılmaz**. Aksi hâlde Google "aynı içerik" cezası verir.

---

## 6. Ana sayfa modül sırası (birebir)

1. Üst nav + arama + dil + oturum
2. **Fiyat şeridi** (Markets Live / Crypto Stocks sekmeli)
3. Promo banner (etkinlik geri sayımı)
4. **Today's Videos** — büyük oynatıcı + oynatma listesi
5. **Latest News** — saat damgalı düz liste ("17 minutes ago")
6. **Thought Leadership** (sponsorlu)
7. **Newsletters** — e-posta kayıt
8. **Manşet bloğu** — 1 büyük + 2 ikincil + metin liste
9. **Press Releases** (3 madde, sponsor etiketli)
10. **Featured Research** (istatistik + başlık)
11. **Today's Videos** (2. blok)
12. **Today's Clips** — dikey (9:16) kısa video karusel
13. Reklam yuvası
14. **Prices** — ilk 10 kripto (sıra, logo, fiyat, %)
15. **Stocks** — ilk 10 ABD hissesi
16. **Exchange Benchmark** — sıralı tablo (1–5 + tam sıralama)
17. **Videos** — 5 kart ızgarası
18. **CoinDesk Indices** — CD5/CD20/CD80/CD100/CDMEME
19. **Podcasts** — kapak + bölüm sayısı
20. **Opinion** — yazar avatarı + başlık
21. **Most Read** — 1 büyük + 5 satır
22. **Bölüm blokları** — Markets, Policy, Technology, Finance
    (her biri: 1 büyük görsel + başlık, 2 alt başlık, 3 satır)
23. **Footer** — 5 sütun (About, Contact, Newsletters, News, Videos,
    Podcasts, Cryptocurrencies, Data & Indices, Events, Sponsored, Research)

---

## 7. Makale sayfası yapısı

```
Breadcrumb: Home / Markets
Kicker:      MARKET (bölüm etiketi, mavi, büyük harf)
Başlık:      H1, 2.2–2.8rem, sıkı
Spot:        gri, 1.1rem
Yazar satırı: avatar + ad + "Nov 5, 2026 at 9:14 a.m. UTC" + okuma süresi
Sosyal:      X / Facebook / LinkedIn / kopyala
Gövde:       tek sütun, 680–720 px, 1.7 satır aralığı
  • ara başlık (H2), liste, alıntı (blockquote), kod, gömülü X/grafik
  • görsel + alt yazı ("Image 23: Fairshake PAC …")
  • "Kaynak: …" atıf satırı (toplayıcı sitede ZORUNLU)
Etiketler:   /tag/bitcoin, /tag/ethereum
İlgili:      "Related" 3 kart
En altta:    "Disclosure" kutusu + yazar kutusu
```

## 8. Bölüm (kategori) sayfası

```
Başlık (H1): bölüm adı + tek satır açıklama
Öne çıkan: 1 büyük görsel kart
Liste:      görsel + başlık + spot + zaman (sonsuz değil, sayfalı)
Sağ ray:    Most Read + reklam
```

## 9. SEO altyapısı (birebir)

| Öğe | CoinDesk | bitcosar |
|---|---|---|
| Sitemap | `sitemap-index.xml` + dil başına | aynı + **news sitemap** (son 48 saat) |
| robots | `/pf/api/v3/*`, `/api/`, `/?s=`, `/?q=` kapalı | aynı mantık |
| AI botları | GPTBot/ClaudeBot/PerplexityBot için ayrı bloklar | aynı (yapılandırılabilir) |
| AMP | `/*/amp` (kapatılmış) | **yapmayacağız** (Google haber için gerekmiyor) |
| Şema | `NewsArticle`, `BreadcrumbList`, `Organization` | aynı + `Person` (yazar) |
| Kanonik | dil başına kendi URL | aynı |
| `hreflang` | 15 dil + `x-default` | yalnız gerçek çeviriler |

---

## 10. İçerik tipleri (CMS modeli)

`haber`, `analiz` (news-analysis), `gorus` (opinion), `canli` (live blog),
`basin-bulteni` (press-release), `sponsorlu` (sponsored), `video`,
`podcast`, `arastirma` (research), `fiyat-sayfasi` (price), `endeks` (index).

theblock'ta `content_types` tablosu zaten var; CoinDesk tipleri **ek** olarak
seed edilecek (mevcut tipler silinmez, `aktif=false` yapılır).
