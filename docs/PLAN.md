# bitcosar — uygulama planı

Hedef: **CoinDesk tasarımı + CoinDesk kategori/içerik yapısı**, theblock
modülleri üzerinde, çok dilli, AI destekli, performans ve SEO odaklı kripto
haber portalı.

Ölçülen CoinDesk yapısı: [`COINDESK-SPEC.md`](COINDESK-SPEC.md)
Kaynak kataloğu: [`KAYNAK-MERKEZI.md`](KAYNAK-MERKEZI.md)
Güvenlik hattı: [`guvenlik-hatti.md`](guvenlik-hatti.md)

---

## 0. Durum (5 Kasım 2026)

| Bileşen | Durum |
|---|---|
| `~/bitcosar` deposu, `main`, CI | ✅ |
| Güvenlik hattı (6 kapı, tek betik) | ✅ |
| `demo.bitcosar.com` → Tunnel → WSL | ✅ |
| theblock kodu taşındı (`go/`, derleniyor + testler geçiyor) | ✅ |
| Ayrı PostgreSQL (`bitcosar-pg`, :5434) | ✅ |
| CoinDesk taksonomisi (`0110`) | ✅ 16 bölüm |
| Kripto kaynak seed'i (`0111`) | ✅ 32 kaynak |
| CoinDesk tasarım sistemi (`cd.css` + kabuk) | ✅ |
| **Ana sayfa: 23 modülün çekirdeği** | ✅ §3, §5, §8, §13, §14, §20, §21, §22 |
| **Makale sayfası (§7)** | ✅ breadcrumb, görsel, kaynak atfı, etiket, ilgili |
| **Bölüm sayfaları (§8)** | ✅ /markets /business /tech /policy … |
| **`/price` + fiyat şeridi** | ✅ CoinGecko, 60 sn önbellek |
| Gerçek görsel + gerçek etiket | ✅ 119/179 görselli, 109 etiket |
| `hreflang` + Organization/WebSite/NewsArticle JSON-LD | ✅ |
| Sitemap + news sitemap + RSS + robots | ✅ |
| Lighthouse bütçe kapısı | ✅ `make performans` — **/ 100 · /markets 99 · /price 100**, SEO 100, BP 96, LCP 0.4–0.9 s, CLS 0 |
| **Mega menü (§3)** | ✅ 5 grup, açılır panel, JS yok; etiket+coin sütunları canlı veriden |
| **Mobil gezinme** | ✅ JS'siz `<details>` hamburger, masaüstünde gizli |
| **Menüde ölü bağlantı** | ✅ 0 (ölçüm: 49 hedefin 29'u 404'tü); testle korunuyor |
| Kaynak görsel proxy (`/kg/{hash}`) | ✅ üçüncü taraf çerez kapandı; dış CDN bağımlılığı yok |
| Kripto alaka kapısı | ✅ resmî kaynaklarda kripto dışı bülten sızmıyor (SEC 25 → 3) |
| **Video / podcast / endeks / borsa tablosu** | 🟡 sayfalar hazır (`/videos`, `/podcasts`), **içerik ve veri kaynağı yok** → dürüst boş durum basar |
| EN dil | ⏳ |

---

## Faz 1 — CoinDesk tasarım sistemi (1. hafta)

Tasarımı **yeniden yorumlamak yok**; ölçülen yapıya birebir oturmak.

### 1.1 Tasarım token'ları
`go/internal/webui/static/cd.css` — mevcut `public.css`'in yerine geçmez,
**üstüne** biner; eski site çalışmaya devam ederken yeni yüzey açılır.

```css
:root{
  --bg:#fff; --fg:#000; --surface-soft:#f8f8f8; --surface-alt:#f4f4f4;
  --border:#e9e9e9; --border-strong:#d8d8d8;
  --muted:#696969; --muted-strong:#4d4d4d;
  --dark:#262626; --dark-2:#2e3644;
  --link:#4d90fe; --link-hover:#3860be;
  --up:#68b631; --down:#ff0062; --warn:#ffd200;
  --font:"proxima-nova",system-ui,-apple-system,"Segoe UI",sans-serif;
  --max:1280px; --radius:2px;
}
```
Kurallar: gölge yok, **çizgiyle** ayırma, 2 px köşe, sıkı başlık aralığı
(1.15), gövde 1.7, `proxima-nova` yoksa sistem fontu (FOUT yok).

### 1.2 Kabuk (layout)
- `header`: üst bar (nav + arama + dil + oturum), altında **fiyat şeridi**
  (Markets Live / Crypto Stocks sekmeli, yapışkan, 40 px).
- `footer`: 5 sütun, ölçülen başlıklarla.
- Mobil: nav hamburger + şerit kaydırmalı.

### 1.3 Şablonlar (templ)
| Şablon | Dosya |
|---|---|
| `cd_head` (meta, hreflang, JSON-LD) | `templkit/cd_head.templ` |
| `cd_header` / `cd_ticker` / `cd_footer` | `templkit/cd_shell.templ` |
| `cd_home` (23 modül) | `templkit/cd_home.templ` |
| `cd_section` (bölüm sayfası) | `templkit/cd_section.templ` |
| `cd_article` (makale) | `templkit/cd_article.templ` |
| `cd_price` / `cd_tag` | `templkit/cd_misc.templ` |

### 1.4 Ana sayfa modülleri (sıra birebir)
fiyat şeridi → video hero → latest news → sponsorlu → bülten → manşet
(1+2+liste) → basın bültenleri → araştırma → video 2 → dikey klipler →
reklam → fiyatlar (10) → hisseler (10) → borsa sıralaması → video ızgarası
→ endeksler → podcast → görüş → en çok okunan → 4 bölüm bloğu → footer.

**Faz 1 kapsamı:** 1.1–1.3 + ana sayfada *manşet, latest news, fiyatlar,
bölüm blokları* (ölçülen 23 modülün 12'si). Kalan modüller Faz 2.

---

## Faz 2 — Kripto toplama hattı (2. hafta)

theblock'ta `feeds`, `crawled_items`, `repo.RefreshFeeds` **hazır**. Yapılacak:

1. **Adaptör katmanı** (`repo/kaynak_*.go`): RSS/Atom ayrıştırıcı + kaynak
   başına eşleme (başlık, spot, görsel, dil, yazar, yayın zamanı).
2. **utls istemcisi**: Cointelegraph/Bitcoin Magazine/BeInCrypto gibi
   TLS parmak izi engelleyen siteler için Chrome ClientHello taklidi.
   (`refraction-networking/utls`)
3. **Kalıcı ham arşiv**: her yanıt `storage-private/kaynak/{kaynak}/{tarih}.xml`.
4. **Tekilleştirme**: URL SHA-256 + başlık simhash (Hamming ≤ 3).
5. **Kümeleme**: `stories` + `story_articles`; sayfada tek kanonik URL,
   altında "bu haberi yazan kaynaklar".
6. **Telif kuralı (kod düzeyinde kapı)**: toplayıcı kaynağın tam gövdesi
   **yayınlanamaz**; `contents.detail` AI özetinden üretilir, `source_url`
   + `source_name` + görünür atıf zorunlu. Atıf yoksa yayın reddedilir.
7. **Görsel hattı**: kaynak görseli → `image-svc` → AVIF/WebP, 3 genişlik.

---

## Faz 3 — AI katmanı (3. hafta)

| İş | Model | Kural |
|---|---|---|
| Kategori/sınıflandırma | ucuz, toplu | mevcut `classify` paketi |
| TR başlık + spot + 120 kelime özet | orta | **yalnız kaynak metninden**, JSON şemalı |
| EN çeviri | orta | kripto terim sözlüğü zorunlu |
| Embedding (kümeleme, ilgili haber) | küçük | `pgvector` |
| Uydurma denetimi | kural + LLM | kaynakta geçmeyen sayı/özel isim → reddet |

AI çıktısı **her zaman `taslak`**; insan onayı olmadan yayına girmez
(theblock `aieditor` akışı aynen kullanılır).

---

## Faz 4 — Çok dillik + SEO (4. hafta)

1. `tr` (kök) + `en`; URL deseni `/en/{section}/{y}/{m}/{d}/{slug}`.
2. `hreflang` **yalnız gerçek çeviriler** için (theblock dersi).
3. `sitemap-index.xml` + dil başına + **news sitemap** (48 saat).
4. `NewsArticle` + `BreadcrumbList` + `Organization` + `Person` JSON-LD.
5. IndexNow ping (Google Indexing API haber için kullanılmaz).
6. Kaldırılan içerik **410**, eski slug **301**.

---

## Faz 5 — Performans bütçesi (5. hafta)

| Metrik | Hedef | Yöntem |
|---|---|---|
| TTFB (edge) | < 80 ms | Cloudflare `s-maxage=60, stale-while-revalidate=600` |
| TTFB (origin) | < 20 ms | tam HTML önbelleği (Redis) + materialized view |
| LCP p75 | < 1.5 s | kritik CSS inline, görsel `srcset`, AVIF |
| HTML gzip | < 60 KB | makalede JS ~0 |
| CLS | < 0.05 | boyut yazılı görsel/iframe |
| Lighthouse | ≥ 95 (perf/SEO/a11y) | CI kapısı |

Önbellek invalidation **tek noktadan**: yayın → `purge(url, tag)`.

---

## Faz 6 — Fiyat/piyasa ve sonrası

- `/price`, `/price/{coin}` (CoinGecko + Binance, 60 sn önbellek)
- `/index` (kendi endekslerimiz — sonra)
- Bildirim (web push), bülten, API (`/api/v1/…`)

---

## Yapılmayacaklar (bilinçli)

| Şey | Neden |
|---|---|
| AMP | Google Haber için gerekmiyor; bakım yükü |
| Yjs/collab, gazete/baskı hattı, TTS/transkript | Kripto portalında kullanılmıyor (kod duruyor, yüzeyde yok) |
| Astro/Java ikinci runtime | Aynı SSR çıktısı Go'da; 2. runtime = 2× bakım |
| Kaynak tam metni | Telif riski; özet + atıf |
