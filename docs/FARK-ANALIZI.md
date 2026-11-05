# CoinDesk ↔ bitcosar fark analizi

Karşılaştırma temeli: `docs/COINDESK-SPEC.md` (CoinDesk'ten **ölçülen** yapı,
5 Kasım 2026) ↔ `https://demo.bitcosar.com` (aynı gün ölçüldü).

Durum kodları: ✅ tam · 🟡 kısmi · ⛔ eksik · 🔒 dış veri bekliyor

---

## 1. Ana sayfa modülleri (§6, 23 madde)

| # | CoinDesk modülü | bitcosar | Not |
|---|---|---|---|
| 1 | Üst nav + arama + dil + oturum | ✅ | nav + arama + dil seçici + Giriş/Üye ol; mega menü hâlâ düz nav |
| 2 | Fiyat şeridi (Markets Live / Crypto Stocks) | 🟡 | kripto şeridi ✅ (7 varlık, canlı); **Crypto Stocks sekmesi boş** |
| 3 | Promo banner (etkinlik geri sayımı) | ⛔ | ayar alanı hazır (`cd-promo`), içerik yok |
| 4 | Today's Videos (oynatıcı + liste) | ⛔ | video verisi/modeli yok |
| 5 | **Latest News** (saat damgalı liste) | ✅ | 12 satır, "3 saat önce" biçimi |
| 6 | Thought Leadership (sponsorlu) | ⛔ | sponsorlu içerik türü var, kayıt yok |
| 7 | Newsletters (e-posta kayıt) | ⛔ | `/bulten` sayfası var, ana sayfada modül yok |
| 8 | **Manşet bloğu** (1 büyük + 3 yan) | ✅ | gerçek kapak görselli |
| 9 | Press Releases (3 madde) | ⛔ | kategori var, içerik yok |
| 10 | Featured Research | ⛔ | araştırma içeriği yok |
| 11 | Today's Videos (2. blok) | ⛔ | — |
| 12 | Today's Clips (dikey 9:16 karusel) | ⛔ | — |
| 13 | **Reklam yuvası** | ✅ | CLS için yükseklik ayrılmış |
| 14 | **Prices** (ilk 10) | ✅ | CoinGecko canlı, 60 sn önbellek |
| 15 | Stocks (ABD hisseleri) | 🔒 | veri kaynağı bağlanmadı |
| 16 | Exchange Benchmark (sıralı tablo) | ⛔ | kendi sıralamamız üretilecek |
| 17 | Videos (5 kart) | ⛔ | — |
| 18 | CoinDesk Indices (CD5/CD20…) | ⛔ | kendi endeksimiz yok |
| 19 | Podcasts (kapak + bölüm) | ⛔ | — |
| 20 | **Opinion** (yazar avatarlı) | 🟡 | şablon ✅, veri 1 içerik |
| 21 | **Most Read** (1 büyük + 5 satır) | ✅ | `hit_daily` son 7 gün ağırlıklı |
| 22 | **Bölüm blokları** (Markets/Policy/Tech/Finance) | ✅ | 1 büyük + 2 alt + 3 satır |
| 23 | Alt bilgi (5 sütun) | ✅ | ölçülen başlıklarla |

**Sayı:** 7 ✅ · 3 🟡 · 1 🔒 · 12 ⛔ → **%52 tam, %65 işlevsel**

## 2. Sayfalar

| Sayfa | CoinDesk | bitcosar | Not |
|---|---|---|---|
| Ana sayfa | `/` | ✅ | 27,7 KB |
| Bölüm | `/{section}` | ✅ | 6 bölüm dolu (markets 23, policy 77, business 40, tech 40, opinion 1) |
| Makale | `/{section}/{y}/{m}/{d}/{slug}` | ✅ | CoinDesk deseni; eski `slug-id` adresler 301 ile taşınıyor |
| Fiyat | `/price` | ✅ | 25+ varlık |
| Tek varlık | `/price/{coin}` | ✅ | `/price/btc` → $80.216 +2,19% |
| Etiket | `/tag/{tag}` | ✅ | 23 kart (Markets) |
| Arama | `/search?q=` | ✅ | `/arama?q=` (Türkçe yol) |
| Video | `/video/{slug}` | ⛔ | — |
| Podcast | `/podcasts/{slug}` | ⛔ | — |
| Araştırma | `/research/{slug}` | ⛔ | — |
| Bülten | `/newsletters/{slug}` | 🟡 | `/bulten` var, tekil bülten yok |
| Yazar | `/author/…` | ⛔ | yazar sayfası yok |
| Endeks | `/index/{cd20}` | ⛔ | — |

## 3. SEO altyapısı (§9)

| Öğe | CoinDesk | bitcosar | Not |
|---|---|---|---|
| `sitemap-index.xml` | ✅ | 🟡 | tek `sitemap.xml` (33 KB); dil başına dizin yok |
| News sitemap (48 saat) | ✅ | ✅ | 79 KB |
| `robots.txt` | ✅ | ✅ | 2,9 KB |
| `hreflang` + `x-default` | 15 dil | 🟡 | `tr` + `x-default` (EN içerik yok) |
| `NewsArticle` JSON-LD | ✅ | ✅ | zorunlu alanlar dolu |
| `BreadcrumbList` | ✅ | ✅ | makale: Ana sayfa → Bölüm → Başlık |
| `Organization`/`WebSite` | ✅ | ✅ | `@graph` içinde |
| `Person` (yazar) | ✅ | ⛔ | yazar profili yok |
| Kanonik | dil başına | ✅ | |
| 410 (kaldırılan) | ✅ | ✅ | theblock'tan devralındı |
| IndexNow | — | ✅ | theblock'tan devralındı |
| RSS | ✅ | ✅ | 32 KB |
| AMP | kapatılmış | ✅ (yok) | bilinçli |

## 4. Tasarım dili (§2)

| Kural | Durum |
|---|---|
| Token'lar (renk/tipografi/ritim) | ✅ birebir (`cd.css`) |
| Gölge yok, ayırma çizgiyle | ✅ testle korunuyor (`cd_tasarim_test.go`) |
| Köşe 2 px, modül arası 40 px | ✅ |
| `proxima-nova` | 🟡 sistem fontuna düşüyor (lisans gerekli) |
| Açık tema varsayılan, karanlık medya sorgusu | ✅ |
| Mega menü (açılır panel) | ✅ 5 grup, JS yok (`:hover`/`:focus-within`) |
| Mobil nav (hamburger) | ✅ `<details>` tabanlı, JS yok, klavye erişilebilir |

## 5. Performans (ölçüldü)

| Metrik | Hedef | bitcosar |
|---|---|---|
| Lighthouse performans | ≥ 90 | **96** |
| Lighthouse SEO | ≥ 95 | **100** |
| Lighthouse a11y | ≥ 90 | **96** |
| Lighthouse best-practices | ≥ 90 | **96** |
| LCP | < 1,5 s | **1,4 s** |
| CLS | < 0,05 | **0** |
| TBT | < 100 ms | **0 ms** |
| Ana sayfa HTML | < 60 KB | 27,7 KB |
| Makale HTML | < 60 KB | 7,4 KB |

## 6. Yapısal farklar (bilinçli sapmalar)

| Konu | CoinDesk | bitcosar | Gerekçe |
|---|---|---|---|
| Render | Astro + Vercel | **Go + templ** | aynı SSR çıktısı, tek binary, 1 süreç |
| Görsel | Sanity CDN | kaynak CDN + `referrerpolicy` | telif: kopyalamıyoruz |
| İçerik | kendi muhabirleri | **toplama + atıf** | kaynak atfı zorunlu kapı |
| Yorum | var | yok (kripto portalında Faz 3) | moderasyon yükü |
| AMP | kapatılmış | hiç yapılmadı | Google Haber gerekmiyor |

---

## 7. Öncelik sırası (sıradaki iş)

### P0 — kırık/eksik işlevsellik ✅ (hepsi kapandı)
1. ~~Makale URL'sinde tarih~~ ✅ `/{section}/{y}/{m}/{d}/{slug}` + eski `slug-id` 301.
2. ~~`BreadcrumbList` JSON-LD~~ ✅ makale + bölüm sayfalarında.
3. ~~Dil seçici + Giriş/Üye ol~~ ✅ kabukta; dil listesi yalnız gerçek dilleri gösterir.
4. ~~Mega menü + mobil hamburger~~ ✅ 5 grup + veriye bağlı sütunlar; mobilde JS'siz `<details>` panel.

### P1 — ana sayfa modülleri (veri bağımlı)
5. Bülten modülü (mevcut `/bulten` altyapısıyla, veri gerekmiyor) — **en kolay kazanç**
6. Promo banner (ayar tabanlı, veri gerekmiyor)
7. Videos / Podcasts / Clips (önce içerik türü + yükleme akışı)
8. Stocks tablosu (veri kaynağı seçimi)
9. Indices / Exchange Benchmark (kendi metodolojimiz)

### P2 — çok dillik
10. EN içerik hattı (AI çeviri + `hreflang` genişletme + dil başına sitemap)
11. Dil başına `sitemap-index.xml`

### P3 — içerik derinliği
12. Yazar sayfası + `Person` şeması
13. Makalede paylaşım düğmeleri + yazar kutusu + editör politikası kutusu
14. AI özet/çeviri hattı (Faz 3)

---

## 8. Düzeltilen gerçek hatalar (bu tur)

| Hata | Etki | Düzeltme |
|---|---|---|
| `/tag/{slug}` rota yok | makale etiketleri 404 | `handleCDTag` + `TagIcerikleri` |
| `/price/{coin}` rota yok | fiyat tablosu satırları 404 | `handleCDCoin` |
| `contentCols` + `image_url` uyumsuzluğu | ana sayfa boş | ikinci tarama yeri güncellendi |
| `cdMostRead` boş dilimde panik | 500 | koşul blok içine alındı |
| `classify` theblock taksonomisi | 179 içerik tek bölümde | `CoinDeskSection` |
| mux literal desende `PathValue` boş | bölüm sayfası 404 | slug closure ile sabitlendi |
| `.gitignore` `vendor/` | 55 statik dosya repo dışı, CI kırmızı | `/go/vendor/` |
| Lighthouse Chrome yolu | kapı sessizce atlanıyordu | Puppeteer Chrome otomatik bulunur |
| **Menüde 29 ölü bağlantı** | menü yapısı tam, site yarım (404) | menüye yalnız **var olan** sayfa yazılır + test kara listesi |
| **`/videos` iki kez tanımlı** | mux panic, servis 53 kez yeniden başladı | bölüm listesinden çıkarıldı, tek sahip tür handler'ı |
| Menüde "News News" | etiket adı zaten "News" ile bitiyordu | `etiketMenuAdi` son ek kuralı |
| **Etiket sütunu yalnız liste sayfalarında** | menü sayfadan sayfaya değişiyordu | veri `cdShellCtx`'e taşındı, her sayfada aynı |
| **Mobilde gezinme yok** | hamburger yazılmamıştı, menü gizliydi | JS'siz `<details>` panel |

## 9. Menü sözleşmesi (kalıcı kural)

> **Menüye yalnız VAR OLAN sayfa yazılır.**

Ölü bağlantı kullanıcıya "site yarım" der; eksik menü satırı yalnızca "daha
az" der. Bu yüzden CoinDesk ölçümündeki şu satırlar **bilinçli olarak yok**:

| CoinDesk satırı | Neden yok |
|---|---|
| `/events` | kripto etkinliğimiz yok |
| `/index`, `/index/cd20`, `/index/xbx`, `/index/etx` | kendi endeks metodolojimiz yok — uydurma endeks yanlış veri olur |
| `/price/stock/*` (SPY, COIN, MSTR…) | ABD hisse veri kaynağı bağlanmadı |
| `/video/coindesk-daily`, `/video/public-keys` … | CoinDesk'in kendi programları |
| `/newsletters/coindesk-headlines` … | bülten serileri henüz yok (tek bülten var) |

Yerine **veriye bağlı** sütunlar: etiketler (içeriği olan, ≥2 içerik) ve
canlı ilk 5 coin. Veri yoksa sütun hiç basılmaz.

Koruma: `templkit/cd_menu_test.go` — ölü hedef kara listesi, hedef biçimi,
"veri yoksa sütun yok", "veri gelince dolar", aktif başlık, `aria-controls`
tekilliği. Yeni ölü hedef eklendiği anda test kırmızıya döner.
