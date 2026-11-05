# theblock — Kantan.News Referans Analizi (frontend + özellikler)

**Sürüm:** 1.0 — 2026-10-14
**Kaynak:** https://kantan.news (canlı; robots.txt, sitemapindex, /api/categories, /api/trends, manifest.json, index HTML/CSS/JS bundle artıkları)
**İlişkili:** `01` §L, `02` (frontend), `03` (dalga eşleme), `05` (gelir modeli)

> Kantan.News, theblock'tan **farklı bir yayın modeli**: AI ile **güvenilir kaynaklardan derleyen**, **reklamsız** (AdSense/GA kaldırılmış), bülten+podcast+topluluk odaklı, çok dilli (tr/en/es) bir teknoloji/gündem platformu. Referans olarak alınacak güçlü yönler: **kategori-AI-skor kapıları**, **trends API**, **içerik derleme hattı**, **sitemap/hreflang/JSON-LD disiplini**, **PWA + erişilebilirlik (font ölçekleme)**.

---

## 1. Teknoloji & Frontend yığını (kanıt: index HTML/JS)
- **SPA**: React + **Vite** (hashed assets: `index-*.js`, `react-vendor`, `query-vendor`, `i18n-vendor`, `icons-vendor`, `date-vendor`)
- Durum/veri: **React Query** (sunucu durumu); router SPA (route'lar robots/sitemap'ten okunur)
- **i18n**: react-i18next benzeri; diller **tr/en/es**; URL stratejisi: kök=tr, `/en`, `/es`; sayfa düzeyinde de `?lang=en` (about/privacy…)
- Tailwind-utility sınıfları (`bg-black text-white min-h-screen theme-body`); `#root` (tam istemci render)
- **Tema**: `html class="dark"` varsayılan; `localStorage.theme` ∈ {dark|light|system}, sistem tercihi (prefers-color-scheme) desteği
- **Erişilebilirlik**: `localStorage.font_scale` ∈ {1, 1.25, 1.5} → `html.font-size` yüzde ölçek (125/150%)
- **PWA**: `manifest.json` (standalone, portrait, maskable 192/512 ikon, `theme_color #000`), apple-* meta, `mobile-web-app-capable`
- **Analitik**: yerel **Matomo** (consent-gated, `src/lib/consent.js`); **GA4 kaldırıldı (2026-08-21)**; **reklam altyapısı söküldü (2026-09-18)** → site reklamsız
- Cloudflare (tünel/edge + insights beacon); **bot tuzağı** `/arsiv/dizin/` (gerçek içerik değil)

## 2. Görünüm (appearance) çıkarımları
- **Dark-first, siyah zemin** (#121212/#000), yüksek kontrast, sade tipografi; light tema opsiyonel
- Sabit/kompakt üst menü + kategori nav (menüde yalnız `show_in_menu` işaretli kategoriler: Teknoloji, Oyun, Bilim, Otomobil), `menu_order` sıralaması
- Kart tabanlı akış; kategori bazlı **akış (feed)**; "trend takip" etiketleri
- Mobil öncelikli; **viewporfit=cover**, safe-area; PWA standalone
- Tema + font ölçekleme kullanıcı tercihi (a11y) — SSR'da FOUC önleyen inline script

## 3. İçerik derleme/iş modeli (kanıt: category "min_score", trends)
- **AI-küratörlü derleme**: güvenilir kaynaklardan toplanır; her **kategori** için:
  - `min_score` (yayın eşiği) ve `homepage_min_score` (anasayfaya çıkma eşiği) — **AI alaka puanı kapısı**
  - çok dilli adlar (`name`, `name_en`, `name_es`), `slug`, `icon`, `label`
  - `image_service` (ör. **pexels** — Spor/Otomobil için otomatik görsel)
- **Trends API** (`/api/trends`): kategori başına `trending_tag`, `post_count`, `updated_at`, `lang` — günlük güncellenen gündem etiketleri (kaynak: trend/gündem) → "kategoriyi trende göre besleme"
- **Günlük derleme çıktıları**: `/ping/YYYY-MM-DD` (günlük özet/ping), `/podcast/YYYY-MM-DD` (günlük podcast) — arşivlenen günlük seriler
- **Bülten**: `/bulten`, `/spor-bulteni` + `/api/bulten` (newsletter), sitemap-bulten
- **Video**: `/videolar`, `/videolar/<slug>`, `/api/videos`, sitemap-videos
- Kütüphane: `/api/news`, `/api/pages`, `/api/i18n`, `/api/categories`, `/api/social`, `/api/comments`, `/api/live`, `/api/digest`

## 4. Kullanıcı & topluluk özellikleri (kanıt: robots kapalı uçlar + rotalar)
- **Kimlik**: `/login`, `/oauth/` (sosyal giriş), oturum gerektiren yazma uçları (`auth`, `admin`, `v1`, `internal`, `track`, `m`, `settings`, `profile`)
- **Profil**: `/profil`, `/u/<kullanıcı>`, `profile` API'si; `delete-account` sayfası (KVKK uyumu)
- **Kaydetme**: `/bookmarks`, `/feed/bookmarks` — haber kaydetme
- **Takip**: `follow` API'si — kategori/yazar/etiket takibi → kişiselleştirilmiş `/feed`
- **Bildirimler**: `notifications` API'si
- **Etkileşim**: `interactions`, `comments` (`/api/comments`), `feedback`, `safety` (içerik güvenliği/şikâyet)
- **AI asistan**: `assistant` API/rotası — okurken AI'ya sor/özetle
- **Gamification**: `/achievements`, `points`, `onboarding` — okuma serisi/rozet/puan (bağlılık)
- **Tracker**: `track` (okuma/olay) + `m` (mobil?) uçları

## 5. SEO & içerik makinesi disiplini (kanıt: HTML/sitemap/robots)
- **hreflang**: tr/en/es + **x-default**; sayfa içi `?lang=` ve yol (`/en`) karışık ama tutarlı
- **Sitemap index**: core, articles (sayfalı ×5), videos, bulten, **news** (Google News), **news-tr**, **news-en**
- **JSON-LD**: `Organization` + `WebSite` + **SearchAction** (`/feed?q={q}`) — makale sayfalarında Article/NewsArticle (sitemap-articles)
- **Meta**: og:locale + alternates, twitter:card summary_large_image, canonical, `robots max-image-preview:large`
- **robots.txt**: Content-Signal (`search=yes, ai-train=no, use=reference`) + eğitim botları engelli (GPTBot, ClaudeBot, Google-Extended, CCBot, Amazonbot…) + AhrefsBot crawl-delay; **`/api/` kapatıp yalnız salt-okunur kamu uçlarını uzun-eşleşmeyle açma** (Search Console soft-404 dersleri)
- **Search Console** dersleri 02-03: SPA'da /api kapalıyken Googlebot boş sayfa görüyordu → **kamu uçlarını aç**, çift sitemap, günlük news sitemap

## 6. Kaynak yönetimi & politika
- **Content-Signal/RSL** ile AI kullanımına açık sınır çizme (search izin, train yasak) — telif/ajans hattı için ileri seviye referans
- Yasal sayfalar: privacy, terms, cookies, **KVKK**, support, delete-account (çok dilli)
- Cloudflare yönetimli content-signals bloğu

---

## 7. theblock eşlemesi — alınacak / uyarlanacak / farklı

### 7.1 Uyarlanacak (değerli, uygulanabilir)
| # | Kantan özelliği | theblock karşılığı / aksiyon |
|---|---|---|
| K-A | **Kategori AI-skor kapıları** (`min_score`/`homepage_min_score`) | `categories`'e `min_score`/`homepage_min_score`/`show_in_menu`/`menu_order`/`icon`/`image_service` ekle; bot/AI üretimini skorla kapıla |
| K-B | **`/api/trends` günlük kategori trendi** | Mevcut "Trend/Gündem"i kategori×trend_tag tablosuna yükselt (haber üretimini besler) |
| K-C | **Çok dilli kategori adları** (`name_en/name_es`) | `categories` i18n adları (24 dil altyapısı hazır) |
| K-D | **Kategori `image_service` (Pexels)** | Görsel yoksa kategoriden otomatik görsel (image-svc + sağlayıcı adaptörü) |
| K-E | **Günlük `/ping` + `/podcast` serisi** | Günlük özet sayfası + TTS podcast (mevcut hizmet mimarisine worker) |
| K-F | **Bülten** (`/bulten`, `/spor-bulteni`, API) | E-posta bülten modülü + sitemap-bulten |
| K-G | **Bookmarks/follow/notifications/points/achievements** | Topluluk + bağlılık katmanı (üye/paywall altyapımızın üstüne) |
| K-H | **hreflang + JSON-LD + SearchAction + Google News sitemap** | theblock'ta eksik: JSON-LD/News sitemap **ekle**, hreflang'i 24 dile genişlet |
| K-I | **Content-Signal/robots disiplini + bot tuzağı** | robots'a content-signal + AI-bot kuralları + tuzak yol |
| K-J | **A11y: font ölçekleme + dark/light/system + PWA maskable** | Font ölçekleme + manifest maskable ikonlar ekle |

### 7.2 Farklı kalan (theblock bilinçli ayrışır)
- Kantan **reklamsız** (bülten/podcast/topluluk geliri); theblock **AdSense + çoklu site** (spec `05`) — iki model birleştirilebilir: reklamsız "premium" site + reklamlı ağ siteleri.
- Kantan **React SPA**; theblock **Go SSR + edge cache** (SEO/performans avantajı korunur). SPA gerekiyorsa yalnız okuma-yoğun "feed/tv" ekranlarında ada bazlı.
- Kantan AI **derleme** (kaynaktan topla); theblock AI **editör asistanı + bot**. Ortak payda: **skorlu kapı**.

### 7.3 Uygulanabilir ilk 4 adım (öneri sırası)
1. `02`§4 ve `03` P0'a **JSON-LD (Article/NewsArticle) + Google News sitemap + hreflang** ekle (SEO açığı kapatır)
2. `categories`'e **AI-skor + menü/ikon/görsel servisi** alanları (`01` §A güncelle, migration)
3. **trends** tablosu + günlük worker (kategori×trend_tag) → bot üretimini besler
4. **Bülten + günlük özet (ping)** modülü (worker + sayfa + sitemap)

## 8. Kabul kriterleri (referans uyarlaması için)
- Kategori AI-skor eşiği altındaki içerik **yayınlanmaz/anasayfaya çıkmaz** (kapı testi)
- Trends kategori×gün güncellenir ve panelden "Haber Yaz" tetiklenebilir
- Her makalede geçerli **Article JSON-LD** + **canonical** + **hreflang** (24 dil) doğrulanır
- **Google News sitemap** üretilir; hreflang'li sitemapindex
- robots.txt **content-signal** içerir + `/api/` kamu uçları açık (SPA değiliz ama ilke: crawl edilebilir)
- Bülten kaydı + günlük özet sayfası (`/ping/YYYY-MM-DD`) üretilir ve arşivlenir

## 9. Açık sorular
- Kantan içerikleri **insan editör** mi yoksa tam AI-derleme mi (kaynak sayfası: config "güvenilir kaynak")?
- `/api/live`, `/api/digest`, `/api/social` veri şemaları (üretimde örnek isteyip dokümante et)
- Bülten/podcast gelir modeli ve abonelik ilişkisi
