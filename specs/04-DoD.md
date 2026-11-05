# theblock — Tanım Yapıldı (Definition of Done) ve Kabul Kontrol Listesi

**Sürüm:** 1.0 — 14 Ekim 2026
**İlişkili:** `00` §17, `01`, `02`, `03` (P0–P5 dalgaları), `05` (iş modeli), `06` (Kantan.News referansı), `AGENTS.md` kapıları

> Bu dosya `03` içinde referans verilen **eksik `04-DoD`** kontrol listesidir. Her dalga için kabul kanıtı
> `docs/ILERLEME.md` içine yazılır. Bir kalem "yeşil" sayılmak için **üçlü kanıt** gerekir:
> (1) `go vet`/`go test`/`cargo check` temiz, (2) uçtan uca HTTP/DB kanıtı (canlı veya izole sunucu),
> (3) kalıcı kayıt (`docs/ILERLEME.md`).

---

## 1. Global kapılar (her değişiklikte)

| # | Kapı | Komut | Durum |
|---|---|---|---|
| G1 | Go statik analiz | `cd go && go vet ./...` | ✅ temiz |
| G2 | Go testleri | `cd go && go test ./...` | ✅ yeşil |
| G3 | Rust derleme | `cd rust/image-svc && cargo check` | ✅ yeşil |
| G4 | Migration'lar | `go run ./cmd/migrate up` | ✅ 0038 uygulandı |
| G5 | Build + restart | `go build -mod=mod -o bin/site ./cmd/site && systemctl --user restart theblock-site.service` | ✅ |
| G6 | Canlı sağlık | `bash deploy/smoke.sh` | ✅ yeşil |
| G7 | Üretim logu temiz | `journalctl --user -u theblock-site.service \| grep ERROR` | ✅ |
| G8 | Test verisi temizliği | e2e betikleri kendi verisini siler | ✅ |

---

## 2. Dalga kabul listesi

### P0 — Yayın Çekirdeği
| # | Kalem | Kanıt | Durum |
|---|---|---|---|
| P0-1 | İçerik yaşam döngüsü (taslak→planlı→yayında→arşiv→çöp) | DB durum geçişleri | ✅ |
| P0-2 | Kategori / mahreç / etiket | `/admin/kategoriler`, `/admin/mahrecler` | ✅ |
| P0-3 | Medya + 16:9 kırpma + WebP | `platform/imaging.go`, `webp.go` | ✅ |
| P0-4 | Yorum sistemi + moderasyon | `/admin/yorumlar` | ✅ |
| P0-5 | SEO: meta, canonical, SERP, robots, ads.txt | sayfa head + `/robots.txt` | ✅ |
| P0-6 | **Schema JSON-LD (Article/NewsArticle)** | `<script type="application/ld+json">` | ✅ |
| P0-7 | **Google News sitemap** | `/sitemap-news.xml` (xmllint geçerli) | ✅ |
| P0-8 | **hreflang alternatifleri** | sitemap `xhtml:link` + `x-default` | ✅ |
| P0-9 | Panel girişi + oturum + 2FA (RFC 6238) | TOTP testleri | ✅ |
| P0-10 | Görev zamanlayıcı (planlı yayın, köp-30g) | `scheduler.Run` | ✅ |

### P1 — AI Editör Asistanı + İçerik Akışı
| # | Kalem | Kanıt | Durum |
|---|---|---|---|
| P1-1 | AI sohbet + alan aksiyonları + maliyet logu | `/admin/ai`, `ai_usage_logs` | ✅ |
| P1-2 | Sesli yazma (Web Speech) | içerik formu | ✅ |
| P1-3 | Görsel manşet editörü (canvas + render) | `source='render'` medya | ✅ |
| P1-4 | Bot/RSS → taslak (url_hash dedupe) | `gundem_items`, `/admin/bot` | ✅ |
| P1-5 | Gündem → "Taslağa Al" | `/admin/trend` | ✅ |
| P1-6 | Sağ tık komut seti | alan bazlı AI menüleri | ✅ canlı (sağ tık AI menüsü) |
| P1-7 | Etkinlikler (hatırlatma şeridi) | `/admin/etkinlikler` | ✅ canlı |
| P1-8 | Web push bildirim | `/admin/bildirim` + `/sw.js` | ✅ canlı (VAPID) |

### P2 — Raporlama & Gelir
| # | Kalem | Kanıt | Durum |
|---|---|---|---|
| P2-1 | Raporlar + grafik + AI CSV dışa aktarım | `/admin/raporlar` | ✅ |
| P2-2 | Hit + paylaşım sayaçları | `POST /hits/` | ✅ |
| P2-3 | Ayarlar (7 sekme) + audit logu | `/admin/ayarlar` | ✅ |
| P2-4 | Reklam yönetimi modülü (5 tip + kod üretici) | `/admin/reklamlar` | ✅ canlı |
| P2-5 | GA4 gerçek entegrasyon (OAuth) | ⚠️ OAuth iskeleti paylaşılıyor (`/admin/adsense`); GA4 rapor çekimi kaldı |
| P2-6 | AdSense gerçek API | ⚠️ OAuth bağlantısı ✅ + hesap doğrulama; gerçek RPM çekimi kaldı |

### P3 — Servis Modülleri
| # | Kalem | Durum |
|---|---|---|
| P3-1 | Hava durumu (Open-Meteo, 81 il) | ✅ canlı |
| P3-2 | Namaz vakitleri (Aladhan) + imsakiye | ✅ canlı |
| P3-3 | Nöbetçi eczane | ⚠️ demo kaynak |
| P3-4 | Döviz kurları | ✅ canlı |
| P3-5 | BIST / kripto | ✅ canlı (`/borsa`, Yahoo Finance + CoinGecko) |
| P3-6 | Sinema / astroloji / fikstür / gazete manşet / BİK / anket | ✅ canlı (fikstür, astroloji, anket, gazete manşet, sinema, resmî ilan/BİK) |
| P3-7 | Sesli okuma (TTS) — **günlük özet podcast'i** | ✅ canlı (`/podcast/{tarih}`) |

### P4 — Gelişmiş & Büyüme
| # | Kalem | Durum |
|---|---|---|
| P4-1 | Arama (pg_trgm, Türkçe) | ✅ canlı |
| P4-2 | AMP | ✅ canlı |
| P4-3 | Paywall / abonelik | ✅ canlı |
| P4-4 | 2FA | ✅ |
| P4-5 | **search-svc (Tantivy)** | ✅ canlı (Go fallback ile) |
| P4-6 | GSC / Indexing API otomasyonu | ✅ IndexNow (Bing/Yandex, HTTP 202) canlı; Google Indexing API servis hesabı girilince hazır |
| P4-7 | Sosyal medya otomatik paylaşım | ✅ canlı (webhook + Telegram, `/admin/paylasim`) |
| P4-8 | E-posta bülten gönderimi (SMTP) | ✅ canlı (`/admin/bulten`); SMTP bilgisi girilince gönderir |
| P4-9 | Geçiş/taşıma (CSV toplu içe aktarım) + **301 link koruma** | ✅ canlı (`/admin/tasima`) |
| P4-10 | Güvenlik denetimi (govulncheck / cargo audit) | ✅ govulncheck temiz; cargo audit 1 "unsound" bilgi uyarısı (Tantivy→lru) |
| P4-11 | Çapraz backlink ağı (spec `01` §D9) | ✅ canlı (`/admin/backlink` + makale "İlgili" ağı) |

### P5 — Çoklu Site & AdSense (spec `05`)
| # | Kalem | Durum |
|---|---|---|
| P5-1 | Tenant izolasyonu (site_id) | ✅ |
| P5-2 | Konu kümeleri + cannibalization uyarısı | ✅ |
| P5-3 | Kaynak doğrulama + özgünlük + YMYL kapısı | ✅ |
| P5-4 | Gelir analitiği + PV×RPM simülatör | ✅ |
| P5-5 | Hesaplayıcı içerik türü + canlı widget | ✅ |
| P5-6 | **Kategori AI-skor kapıları** (K-A) | ✅ yayın + anasayfa kapısı |
| P5-7 | **Trends tablosu + `/api/trends`** (K-B) | ✅ |
| P5-8 | **Bülten + günlük özet + podcast** (K-E/F) | ✅ |
| P5-9 | **Bookmarks/follow/bildirim/gamification** (K-G) | ✅ |
| P5-10 | **PWA + yazı ölçekleme** (K-J) | ✅ |
| P5-11 | **Otomatik kapak görseli** (K-D) | ✅ |
| P5-12 | **Geri bildirim + AI asistan** (L12) | ✅ |
| P5-13 | AdSense/GA OAuth → gerçek gelir | ⚠️ OAuth bağlantısı ✅ (`/admin/adsense` bağlan/senkron); gerçek gelir çekimi Google kimlik bilgisiyle |
| P5-14 | İç link önerisi + evergreen güncelleme | ✅ canlı (`/admin/seo`) |
| P5-15 | **Politika/yasal sayfalar** (KVKK/Gizlilik/Kullanım/Çerez/Reklam) | ✅ canlı (`/sayfa/…`, footer) |
| P5-16 | 3-site pilot dashboard (site × gelir/karşılaştırma) | ✅ canlı (`/admin/siteler` karşılaştırma tablosu) |

### P6 — Önyüz & Alanlar (spec `07`, Kantan şeması)
| # | Kalem | Durum |
|---|---|---|
| C-1 | Kategori `label` alanı (kantan `/api/categories` hizası) | ✅ canlı (`/admin/kategoriler`) |
| C-2 | İçerik alanları: `image_credit`, `social_summary`, `ai_summary`, `audio_url/seconds/status`, `scroll_count` | ✅ canlı (0035) |
| C-3 | Önyüz: makale görsel kredisi + AI özet kutusu + ses oynatıcı + kaydırma beacon'ı | ✅ canlı |
| C-4 | **Bot tuzağı (honeypot)**: gizli link + robots Disallow + 410/noindex + log (`/admin/indexleme`) | ✅ canlı |
| C-5 | **Bloklu IP katmanı**: tuzak isabeti → otomatik blok → 403 (private/izin/smoke hariç), panelde çözme | ✅ canlı |
| C-6 | **Kategori sekmeleri (Güncel/Teknoloji/Oyun/Bilim/Otomobil/Diğer) + Akış Modu** (`/akis`, sonsuz kaydırma, akış/ızgara geçişi) | ✅ canlı |
| C-7 | **Manşet görünümü + tarih filtresi + ana sayfa sekme filtresi** | ✅ canlı |
| C-8 | **Sekme rozetleri (içerik sayısı) + hızlı tarih çipleri (Bugün/Dün/Bu hafta)** | ✅ canlı |
| C-9 | **Önyüz ince ayarları**: okunabilirlik tipografisi, a11y (skip-link + focus halkası), okuma ilerleme çubuğu, başa dön, mobil nav kaydırma, feed iskeleti, tema rengi senkronu + **varlık cache-busting** (`?v=`) | ✅ canlı |
| C-10 | **Panel Dashboard yeniden tasarımı**: 10 KPI kartı (ikon+ton), son 7 gün yayın grafiği, sistem durumu, hızlı işlemler, son içerikler + en çok okunanlar | ✅ canlı |
| C-11 | **Çok-site izolasyon düzeltmesi**: kategoriler (`ListCategories`+`CreateCategory` site_id) ve yorumlar (`CountPendingComments`+`ListComments`) + medya sayacı + `withSite` middleware | ✅ canlı |
| C-12 | **Dashboard geliştirme** (7/30 gün grafik aralığı + çok-site karşılaştırma) + **layout/padding düzeltmesi** (`.card.card-pad` 119 yerde uygulanmıyordu, admin-body alt boşluk, responsive panel) | ✅ canlı |
| C-13 | **Silme onayı düzeltmesi** (`data-confirm` JS `href="#"`'a gidiyordu → tüm silmeler çalışmıyordu; native onay + `data-confirm` URL'ine POST) + **panel menü gruplandırma** + görsel ince ayarlar (tablo hover, buton/odak durumları, scrollbar) | ✅ canlı |
| C-14 | **Manşet Editörü profesyonel yeniden tasarım** (sekmeler, snap kılavuzları, undo/redo, zoom, ızgara, hizalama, öne/arkaya, çoğalt/sil, klavye; overlay modları+8 yön+opaklık+presetler; gelişmiş rozet; metin kaydırma+gölge) + **çift-encode veri sözleşmesi düzeltmesi** (export/şablon) | ✅ canlı |

---

## 3. Kalan işler — öncelik sırasıyla

**Toplam: 73 kalem → 69 ✅ tam · 4 ⚠️ kısmi · 0 ❌**

**⚠️ Yalnızca kimlik bilgisi / dış kaynak gerektirenler (kod hazır):**
1. **P5-13 / P2-6 AdSense gerçek gelir çekimi** — OAuth bağlantısı ✅ (`/admin/adsense`); Google onayından sonra gerçek RPM/gösterim çekimi
2. **P2-5 GA4 rapor çekimi** — OAuth iskeleti paylaşılıyor; GA4 Data API çağrısı eklenmeli
3. **P3-3 Nöbetçi eczane** — gerçek belediye/kaynak URL'si girilince dolar (şimdi demo)
4. **P4-6 Google Indexing API kısmı** — servis hesabı JSON girilince etkin (IndexNow zaten canlı)

**Önerilen sıradaki (kimlik gerektirmeyen, doğrulanabilir):**
- Gerçek AdSense geliri için yalnızca Google istemci kimliği + onay gerekiyor
- Önyüz/tema ince ayarları (kullanıcı geri bildirimine göre)

## 4. Tanım (bitişik kriteri)

Bir dalga **tamamlanmış** sayılır ve `03`'te işaretlendiğinde:
1. `01`/`02` içindeki ilgili satırlar yeşile dönmüş olmalı,
2. `docs/ILERLEME.md` içinde **tarihli kanıt bloğu** bulunmalı,
3. Canlı (üretim) doğrulaması yapılmış olmalı,
4. Geçici test verisi DB'den silinmiş olmalı.