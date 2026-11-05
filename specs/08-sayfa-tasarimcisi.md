# theblock — Sayfa Tasarımcısı (Modüler Blok / Manşet & Slider Yönetimi)
> Elementor/WordPress-widget benzeri, sürükle-bırak modüler sayfa düzeni. Manşet Editörü'nün (görsel kapak üretimi) üstüne **düzen** katmanı ekler.

**Sürüm:** 1.0 — 2026-10-15
**İlişkili:** `02` (frontend), `07` (önyüz/alanlar), `04-DoD`, `05` (çoklu site)

---

## 1. Amaç
Ana sayfada **birden fazla manşet/slider alanı** olur ve bunlar **modüler bloklar** hâlinde sürükle-bırak ile dizilir, seçeneklerle değiştirilir. Bloklar WordPress/Elementor mantığında: **ekle · sırala · yapılandır · gizle · çoğalt · sil**.

## 2. Veri modeli
- **`page_layouts`** (yeni): `site_id`, `page_key` (ör. `home`), `blocks JSONB`, `updated_at` · `UNIQUE(site_id, page_key)`.
- **blocks**: sıralı dizi; her öğe `{id, type, visible, opts}`.
- `opts`: bloğa özel seçenekler (aşağıda). İstemci her tür için uygun alanları üretir; sunucu **tipli** görünüme çevirip render eder.
- **Varsayılan layout** seed edilir; kayıt yoksa mevcut sabit ana sayfa gösterilir (geri uyum).

## 3. Blok tipleri (MVP)
| Tip | Açıklama | Seçenekler |
|---|---|---|
| `slider` | Otomatik geçişli görsel slider (kapak+başlık) | `count`, `autoplay`, `interval` (sn), `dots`, `arrows`, `kategori` (slug, ops) |
| `manset` | Manşet ızgarası (1 büyük + yandakiler) | `count`, `only_badges`? |
| `son_dakika` | Kayan son dakika bandı | `count` |
| `one_cikanlar` | Öne çıkanlar kart listesi (ızgara) | `count`, `kategori` |
| `kategori_izgara` | Kategori kartları (ikon + içerik sayısı) | `count` |
| `trend` | Gündem/trend etiketleri | `count` |
| `reklam` | Reklam yerleşimi (slot) | `slot` (slot_key), `label` |
| `servis_bandi` | Hava + Piyasa bandı | — |
| `baslik` | Bölüm başlığı (ayraç) | `text` |

## 4. Düzenleyici (Editor) — etkileşim modeli
- **Blok paleti** (sol): blok tipleri; tıkla → diziye eklenir.
- **Tuval** (sağ): bloklar kart olarak listelenir; **sürükle-bırak** ile sıralanır (HTML5 drag + dokunmatik fallback), **yukarı/aşağı** düğmeleri, **görünürlük**, **çoğalt**, **sil**.
- **Denetçi**: seçili bloğun seçenek formu (tipe göre alanlar) — anında önizlemeye yansır.
- **Önizleme**: bloklar sayfa içinde gerçek SSR çıktısıyla önizlenir (`Önizle` → `/?onizleme=1`).
- **Kaydet / Sıfırla**: layout'u `page_layouts`'a yazar; sıfırla → varsayılan.

## 5. Genel (public) render
- `handleHome`: `page_key=home` layout'unu yükler → **tipli blok listesi**; blokları SSR ile sırayla basar.
- Slider: saf JS (bağımlılıksız) — otomatik geçiş, ok/nokta, klavye, dokunmatik kaydırma, `prefers-reduced-motion`.
- Layout yoksa: mevcut sabit düzen (geri uyum).
- SEO: bloklar SSR (crawl edilebilir); slider ilk slayt LCP için eager, kalanı lazy.

## 6. Kabul kriterleri
- ≥6 blok tipi eklenip sıralanabiliyor; sıralama sürükle-bırak ile değişiyor.
- Slider **otomatik geçiş + oklar + noktalar** ile çalışıyor; seçeneklerden kapatılabiliyor.
- Layout kaydedilince ana sayfa **kaydedilen sıraya** göre render ediyor; varsayılana sıfırlanabiliyor.
- Çoklu site: layout `site_id` ile izole.
- `go vet`/`test`/smoke yeşil; üretim 200.

## 7. Kapsam dışı (sonraki)
- Serbest konumlu canvas (absolute drag) — bu sürüm **akış (flow) tabanlı** modüler dizilim.
- Blok-içi zengin metin editörü, sürümleme/geri alma geçmişi, A/B.
- Manşet Editörü görsel belgesi ile blok başına kapak şablonu eşleme (sonraki adım).

## 8. İlişki: Manşet Editörü ↔ Sayfa Tasarımcısı
- **Manşet Editörü**: tek bir **görsel kapak** üretir (insan/AI başlık + karartma → JPEG/WebP).
- **Sayfa Tasarımcısı**: kapakların ve içerik bloklarının **sayfadaki düzenini** yönetir.
- Sonraki adım: bir bloğa/seçilen içeriğe **kapak şablonu** atayıp otomatik uygulama.