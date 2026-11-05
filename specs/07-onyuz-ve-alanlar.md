# theblock — Önyüz (Frontend) + Kategoriler + İçerik Alanları Speci
> Kantan.News referansıyla hizalanmış önyüz ve alan modeli (spec `06`'nın uygulama detayı).

**Sürüm:** 1.0 — 2026-10-14
**Kaynak (canlı, salt-okunur kamu uçları):** `https://kantan.news/api/categories`, `/api/news`, `/api/trends`
**İlişkili:** `06` (referans), `01` §A (kategori), `02` (frontend), `04-DoD`

---

## 1. Kategori modeli (kantan `/api/categories` → theblock `categories`)

| Kantan alanı | Anlam | theblock karşılığı | Durum |
|---|---|---|---|
| `id` | kimlik | `id` | ✅ |
| `name` | TR ad | `name` | ✅ |
| `name_en` | EN ad | `name_en` | ✅ |
| `name_es` | ES ad | `name_es` | ✅ |
| `slug` | URL | `slug` | ✅ |
| `min_score` | yayın AI-skor eşiği | `min_score` | ✅ |
| `homepage_min_score` | anasayfa eşiği | `homepage_min_score` | ✅ |
| `show_in_menu` | üst menüde göster | `in_menu` (ve `in_header`) | ✅ |
| `menu_order` | menü sırası | `sort_order` | ✅ |
| `icon` | ikon anahtarı | `icon` | ✅ |
| `image_service` | otomatik görsel (pexels) | `image_service` | ✅ |
| `label` | kısa görünen etiket | **`label`** | ➕ eklendi (0035) |

**Menü kuralı:** üst menü = `show_in_menu/in_menu=1` olan ve `menu_order/sort_order` ile sıralı kategoriler (kantan `menu_categories` eşdeğeri). theblock'ta `in_header` üst menü, `in_menu` yan menü; ikisi de desteklenir.

## 2. İçerik (haber) alanları (kantan `/api/news` → theblock `contents`)

| Kantan alanı | Anlam | theblock karşılığı | Durum |
|---|---|---|---|
| `id, slug, title, excerpt, published_at, created_at` | temel | `id, slug, title, spot, published_at, created_at` | ✅ |
| `category` | kategori adı | `category_id` (join) | ✅ |
| `image_url, images` | kapak/galeri | `cover_media_id`, galeri | ✅ |
| `image_credit` | **görsel kredisi** | **`image_credit`** | ➕ eklendi |
| `original_link` | kaynak bağlantısı | `source_url` | ✅ |
| `summary` | **AI özet** | **`ai_summary`** | ➕ eklendi |
| `social_summary` | **sosyal paylaşım metni** | **`social_summary`** | ➕ eklendi |
| `tags` | etiketler | `content_tags` | ✅ |
| `views, total_views` | okunma | `counters_hit.total` | ✅ |
| `scroll_view_count` | **kaydırma derinliği** | **`content_metrics.scroll_count`** | ➕ eklendi |
| `audio_url, audio_seconds, audio_status` | **makale TTS sesi** | **`audio_url, audio_seconds, audio_status`** | ➕ eklendi |
| `has_translation` | çeviri var mı | `translation_of` / lang refs | ✅ |
| `author_name, author_avatar` | yazar | `author_id` (join, `users`) | ✅ |
| `like_count, favorite_count, comment_count` | etkileşim | bookmarks/yorum sayaçları | ✅ (like hariç) |
| `user_liked, user_favorited` | oturum bazlı | bookmark oturumu | ✅ (favorite) |

**Ek alanlar (0035 migration):** `contents.image_credit`, `contents.social_summary`, `contents.ai_summary`, `contents.audio_url`, `contents.audio_seconds`, `contents.audio_status`, `content_metrics(content_id, scroll_count)`.

## 3. Önyüz (sayfa sayfa)

> theblock **Go SSR + edge cache** (kantan React SPA değil; SEO/performans avantajı). Görünüm kantan'ın dark-first, kompakt, kart-tabanlı akışından esinlenir.

- **Ortak:** dark/light/system tema, font ölçekleme (a11y), PWA (maskable), sabit üst menü, kategori nav (menü kategorileri).
- **Ana sayfa (`/`):** son dakika bandı · manşet ızgarası · trend etiketleri · kategori akış kartları · hava/piyasa/etkinlik şeridi · reklam alanları.
- **Kategori (`/kategori/{slug}`):** kategori başlığı + **trend etiketi** + sayfalı kart akışı.
- **Makale (`/{slug}-{id}`):** kapak + **görsel kredisi** + spot + gövde + **AI özet kutusu** (`ai_summary`) + **kaynak bağlantısı** (`source_url`) + **sesli dinle** (`audio_url`) + etiketler + **İlgili içerikler** (backlink ağı) + yorumlar.
- **Arama / arşiv / servis sayfaları:** mevcut (borsa, fikstür, astroloji, hava, namaz, eczane, gazete manşetleri, sinema, resmi ilan, anket, etkinlik).

## 4. Kabul kriterleri
- Kategori formunda **`label`** alanı düzenlenebilir; menü `sort_order` ile sıralı görünür.
- İçerik formunda **görsel kredisi / sosyal özet / AI özet** kaydedilir ve makale sayfasında görünür.
- Makale sayfası: `image_credit` + `source_url` bağlantısı + (varsa) `audio_url` oynatıcı + `ai_summary` kutusu.
- Kaydırma derinliği (`scroll_count`) ziyaretçi kaydırdıkça artar; içerik raporunda görünür.
- Tüm alanlar mevcut sayfaları bozmaz; `go vet`/`go test`/smoke yeşil.

## 5. Uygulama adımları (bu spec ile)
1. Migration `0035_kantan_fields` (kategori `label` + içerik alanları + `content_metrics`).
2. `repo/content_fields.go` (oku/yaz/ses/kaydırma) + `Category.Label`.
3. Panel: içerik formu (kredi/sosyal özet/AI özet), kategori formu (`label`).
4. Önyüz: makale kredisi/kaynak/AI özet/ses oynatıcı + kaydırma beacon'ı.
5. Kanıt + `docs/ILERLEME.md` kaydı.

## 6. Bilinçli farklar (theblock ≠ kantan)
- **Reklam:** theblock AdSense + çoklu site; kantan reklamsız.
- **Render:** Go SSR (kantan SPA) — SEO/edge avantajı.
- **İçerik:** theblock editör+bot; kantan AI-derleme. Ortak payda **skorlu yayın kapısı**.
- `like` (beğeni) alanı kapsam dışı; kaydetme (bookmark) + yorum kullanılıyor.