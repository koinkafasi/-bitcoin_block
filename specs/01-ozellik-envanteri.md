# theblock — Özellik Envanteri v0.1 (kaynak: Haber Yazılımı AI)

**Tarih:** 2026-10-14
**Kaynaklar:**
- (S1) https://www.haberyazilimiai.com/#ozellikler — özellik sayfası
- (S2) D:\ozelikler — YouTube CC altyazıları (5 video: Keşfet, Etkinlikler, Raporlama, Google Trends, Reklam Geliri)
- (S3) songazete.com canlı incelemesi (önceden yapıldı — servisler menüsü)

> Not: Bu envanter "ne var" listesidir. Öncelik sıralaması ve uygulama planı ayrı dokümandadır.

---

## A. Çekirdek İçerik Üretimi & Yönetimi
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| A1 | Haber ekleme/düzenleme/taslak/zamanlama | S1 | İleri tarihli yayın |
| A2 | Makale kavramı | S1 | Haberden ayrı içerik tipi |
| A3 | Foto galeri sistemi | S1 | Slayt gösterimi, PhotoSwipe |
| A4 | Video galeri sistemi | S1 | YouTube/yerel video |
| A5 | Köşe yazarları | S1 | Yazar sayfaları |
| A6 | Biyografi sayfası/modül | S1 | Kişi biyografileri |
| A7 | Mahreç yönetimi + mahreç raporlama | S1 | Ajans/mahreç etiketi + analiz |
| A8 | Kategorileme | S1 | Çoklu kategori, alt kategori |
| A9 | Etiket sistemi | S1 | Haber etiketleri |
| A10 | Ortam kütüphanesi / medya arşivi | S1 | Upload, arama, yönetim |
| A11 | WebP desteği | S1 | Görsel dönüşüm (Rust image-svc) |
| A12 | Kopya içerikler | S1 | Orijinal/kopya işaretleme, kaynak link |
| A13 | İçerik linkleri | S1 | Haber içi link yönetimi (iç linkleme) |
| A14 | Günün haberleri / günün manşetleri | S1 | Editör seçimi |
| A15 | Editör içerikleri | S1 | Editör tarafından öne çıkarılanlar |
| A16 | Habere özel meta kod | S1 | Per-haber custom meta/script |
| A17 | Kalıcı bağlantı (permalink) ayarları | S1 | URL yapısı tanımlama |
| A18 | Çöp kutusu | S1 | Geri alınabilir silme |
| A19 | Taslak | S1 | Kaydedilmemiş/ön yayın |
| A20 | Sesli haber yazabilme | S1 | Dikte ile haber girişi |

## B. Yönetim Paneli — Dashboard & Modüller
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| B1 | Keşfet (Discover) ekranı | S1,S2 | Sayılar: haber/makale/galeri/video, çöp, onay bekleyen yorum, okunmamış mesaj |
| B2 | Etkinlikler modülü | S1,S2 | Etkinlik ekleme, tarih/saat, hatırlatma günleri, AI'ya gönderme |
| B3 | Raporlama (GA API) | S1,S2 | Gerçek zamanlı (3 dk güncelleme): aktif kullanıcı, sayfa, ülke, 5/10/15 dk hareket; geçmiş (7/15/30 gün): ziyaretçi, cihaz, oturum, PV, süre, bounce, vs. |
| B4 | Google Trends modülü | S1,S2 | ~5 dk güncelleme, yükselen aramalar, kaynaklara git, AI'ya gönder |
| B5 | Ulusal Gündem modülü | S1 | Kaynak takibi, tek tıkla AI'ya |
| B6 | Reklam Geliri (AdSense API) | S1,S2 | 1 tık bağlantı, 3 saatte rapor: bugün/son3s/dün/ay, gösterim, tıklama, CTR, RPM, CPC |
| B7 | Modül yönetimi | S1 | Modül aç/kapa |
| B8 | Kullanıcı seviye yönetimi | S1 | Rol/yetki seviyeleri |
| B9 | Çalışan editör analizi | S1 | Editör performans metrikleri |

## C. Yapay Zekâ (AI Editör Asistanı) — panelde
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| C1 | Panel içi AI asistanı | S1,S2 | Sağ tık → "araştır ve yaz", saniyelerde taslak |
| C2 | Başlık alternatifi / özet / SEO açıklama / etiket çıkarma | S1 | Yayın asistanı görevleri |
| C3 | Editör kontrol listesi (AI) | S1 | Yayın öncesi kontrol |
| C4 | Varsayılan prompt şablonları + özel prompt | S1 | Ayarlardan kullanıcı tanımlı |
| C5 | Kaynak entegrasyonu: Trends/Etkinlik/Ulusal Gündem → AI | S1,S2 | Tek tıkla konu → taslak |
| C6 | AI motor desteği | S1 | ChatGPT, Gemini, Claude — kullanıcının kendi API anahtarı |
| C7 | İnternette arama yaptırma | S1 | AI'ya web araştırması |
| C8 | Google serp/SEO uyumlu çıktı | S2,S1 | Üretilen metin SEO kurallarıyla |

## D. SEO, Arama Görünürlüğü & Yapısal Veri
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| D1 | %100 Schema uyumlu | S1 | NewsArticle, BreadcrumbList vb. JSON-LD |
| D2 | Anlık/gelişmiş SEO skoru | S1 | Başlık/özet/görsel/etiket puanlanır |
| D3 | SEO analiz + özel SEO ayarları | S1 | Meta alanları, canonical, vs. |
| D4 | Google News & Discover uyumu | S1 | Teknik + içerik kuralları |
| D5 | Sitemaps + Google News Sitemap XML | S1 | Otomatik üretim |
| D6 | Robot ayarları, robots.txt & ads.txt düzenleme | S1 | Dosya editörü |
| D7 | URL 301 yönlendirme (redirect) | S1 | Kalıcı yönlendirme yönetimi |
| D8 | Core Web Vitals / PageSpeed optimizasyonu | S1 | Performans odaklı yapı |
| D9 | Çapraz backlink sistemi | S1 | Site-içi backlink ağı |
| D10 | 404 sayfa kontrolü | S1 | Hata yönetimi |
| D11 | Google Indexing API | S1 | Anlık index isteği |
| D12 | Google AMP | S1 | AMP sürümü |
| D13 | GEO / AI arama görünürlüğü | S1 | AI motorları (ChatGPT/Gemini/Claude) için içerik mimarisi |

## E. Reklam & Gelir
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| E1 | Reklam yönetimi (alanlar) | S1 | Kod/alan tanımlama, yerleşim |
| E2 | Manşet reklamları | S1 | Manşet bölgesi reklam |
| E3 | Haberler arası reklamlar | S1 | İçerik arası yerleşim |
| E4 | Google AdSense API raporlama | S1,S2 | Panelde gelir analizi |
| E5 | Özel kod alanları | S1 | AdScript/management injection |

## F. Kullanıcı, Üyelik & Etkileşim
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| F1 | Üyelik sistemi | S1 | Kullanıcı kayıt/giriş/profil |
| F2 | Abonelik sistemi | S1 | Üyelik paketleri, abonelik |
| F3 | Yorum sistemi | S1 | Onay/red, moderasyon, SEO |
| F4 | Sesli yorum | S1 | Sesli geri bildirim |
| F5 | 2FA (çift faktör) | S1 | Panel güvenliği |
| F6 | WhatsApp paylaş / ihbar butonu / mesaj | S1 | Ziyaretçi etkileşim |
| F7 | Sosyal medya otomatik paylaşım | S1 | Facebook/X/Telegram vs. yayın sonrası |

## G. Tasarım & Tema
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| G1 | Responsive (Tailwind) tasarım | S1 | Mobil öncelikli |
| G2 | Sürükle-bırak tema yönetimi | S1 | Modül yerleşimi |
| G3 | Ön yüzden site tasarlama | S1 | Canlı önizleme |
| G4 | Header/Footer seçenekleri | S1 | Farklı şablonlar |
| G5 | Her kategoriye özel tasarım | S1 | Kategori bazlı tema |
| G6 | Infinity scroll | S1 | Sonsuz kaydırma |
| G7 | Manşet yönetim sistemi | S1 | Ana manşet seçimi, sıralama |
| G8 | Manşet görsel oluşturma/düzenleme | S1 | Grafik editörü |
| G9 | Manşet sabitleme | S1 | İstenen süre sabit |
| G10 | Düzenlenebilir manşet | S1 | Metin/görsel düzenle |
| G11 | Menü yönetimi | S1 | Çoklu menü |
| G12 | Panel içi grafik editörü (şablon kaydet) | S1 | Manşet/görsel, harici program gerekmez |
| G13 | Hikaye (Story) | S1 | Web story / kısa içerik |

## H. Servis Sayfaları & Yardımcı Modüller (haber siteleri)
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| H1 | Resmi İlanlar (BİK) uyumu | S1 | Basın İlan Kurumu süreç |
| H2 | E-Gazete PDF sistemi | S1 | Günlük gazete PDF |
| H3 | Gazete manşetleri / günlük gazeteler | S1,S3 | Güncel manşet görselleri |
| H4 | İmsakiye / Namaz vakitleri | S1,S3 | Şehir bazlı |
| H5 | Trafik durumu | S1,S3 | İstanbul vb. |
| H6 | Nöbetçi eczaneler | S1,S3 | İl/ilçe bazlı |
| H7 | Sinemalar (fragmanlı) | S1,S3 | Şehir, salon, seans |
| H8 | Astroloji / burç yorumları | S1,S3 | Günlük/haftalık |
| H9 | Döviz kurları / altın / borsalar / kripto | S1,S3 | Canlı piyasa verisi |
| H10 | Puan durumu & lig fikstürü | S1,S3 | Spor |
| H11 | Vefat edenler | S1,S3 | İlan |
| H12 | Yerel haberler + muhabir yönetimi | S1,S3 | Mahreç/ilçe |
| H13 | Arşiv / gazete arşivi / arşivleme | S1,S3 | Tarih bazlı |
| H14 | Anketler | S1,S3 | Oylama |
| H15 | RSS okuyucu + RSS oluşturma | S1,S3 | İçerik alma/verme |
| H16 | Canlı haber akışı | S1,S3 | Son dakika şeridi, otomatik yenileme |
| H17 | Ajans botları | S1 | Ajans RSS otomatik çekim |
| H18 | Otomatik döviz çevirici sayfalar | S1 | Kur çevirme |
| H19 | Tüm dillere çeviri | S1 | Çok dilli |
| H20 | Sesli okuma (TTS) | S1 | Makale seslendirme |

## I. Teknik Altyapı & Entegrasyonlar
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| I1 | Google API entegrasyonları | S1 | Analytics, Search Console, Indexing, Recaptcha, YouTube |
| I2 | Yandex Metrica API | S1 | Alternatif analitik |
| I3 | Arama yönetimi (Algolia) | S1 | Site arama motoru |
| I4 | Tarayıcı bildirimleri | S1 | OneSignal / Bildirt / yerel |
| I5 | PWA | S1 | manifest + service worker |
| I6 | Cloudflare cache | S1 | Kenar önbelleği |
| I7 | Ücretsiz SSL | S1 | TLS |
| I8 | Otomatik sayfa yenileme | S1 | Son dakika otomatik |
| I9 | Sesli arama | S1 | Arama dikte |
| I10 | Görsel CDN + dönüşüm | S1 (WebP) | image-svc ile |

## J. Geçiş / Veri Taşıma (rakip ürünlerden)
| # | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| J1 | 100k'ya kadar içerik taşıma | S1 | Haber/kategori/yazar/SEO |
| J2 | 301 ile link koruma | S1 | Eski URL'ler yönlendirilir |
| J3 | SEO/URL yapısı koruma | S1 | Otorite düşmez |

---

## K. Çoklu Site Ağı & AdSense Geliri (spec `05` — iş modeli)
| ID | Özellik | Plan | Açıklama |
|---|---|---|---|
| K1 | `sites` (tenant) CRUD + panel site seçici | P5-A | marka/domain/dil/tema; tek panel çoklu yayın |
| K2 | `site_id` izolasyonu | P5-A | içerik/kategori/medya/ayar/reklam/analitik kayıtlarında; geçiş DEFAULT 1 |
| K3 | Domain→site eşleme; site bazlı robots/ads.txt/sitemap | P5-A/E | host sorgusuyla site çözümü |
| K4 | Konu kümesi (`keyword_clusters`) + içerik eşleme | P5-B | pillar/cluster, arama niyeti |
| K5 | İçerik çakışması (cannibalization) uyarısı | P5-B | aynı cluster'da benzer başlık/kelime ikazı |
| K6 | İç link önerileri + canonical kontrolü | P5-B | cluster içi linkler |
| K7 | Evergreen + AI güncelleme önerisi taslağı | P5-B/C | yaşlı içerik uyarısı; onaylı güncelleme |
| K8 | Özgünlük + kaynak doğrulama ekranı | P5-C | benzerlik ölçüsü; kaynak listesi |
| K9 | YMYL rozeti (finans/sağlık doğrulama alanı) | P5-C | onaysız otomatik yayın kapalı |
| K10 | AdSense/GA site+URL bazlı gelir raporu (RPM/gelir/gösterim/CTR) | P5-D | P2 §AdSense kartının çoklu-site hâli |
| K11 | İçerik başına kâr = gelir − AI maliyeti | P5-D | `contents_extra/revenue` + `ai_usage_logs` |
| K12 | PV×RPM gelir simülatörü (3-site planlama) | P5-D/G | panel hesaplayıcı |
| K13 | Hesaplayıcı içerik türü (finans/tadilat widget) | P5-F | JSON şema, frontend widget |
| K14 | KVKK/Gizlilik/Kullanım seed sayfaları + telif/kaynak/yazar kontrol listesi | P5-E | durum amacıyla denetim |
| K15 | Reklam yerleşimi + YMYL yoğunluk uyarısı, site bazlı | P5-E | ads alanları site_id |
| K16 | 3-Site Pilot Dashboard (trafik×RPM×gelir×maliyet karşılaştırma) | P5-G | büyüt/iyileştir/birleştir kararı |

---

## L. Kantan.News Referansı (spec `06`)
| ID | Özellik | Kaynak | Açıklama |
|---|---|---|---|
| L1 | Kategori **AI-skor kapıları** (`min_score`/`homepage_min_score`) | (K) | içerik kalite eşiği; ana sayfa eşiği ayrı |
| L2 | Kategori menü/ikon/`image_service` alanları | (K) | `show_in_menu`, `menu_order`, `icon`, pexels otomatik görsel |
| L3 | **Trends API** (kategori×trend_tag, günlük) | (K) | `/api/trends` — bot üretimini besler |
| L4 | Çok dilli kategori adları | (K) | `name_en/name_es` (theblock 24 dil) |
| L5 | Günlük **/ping** (özet) + **/podcast** serisi | (K) | `YYYY-MM-DD` arşivli günlük çıktılar |
| L6 | **Bülten** (`/bulten`, `/spor-bulteni`, API) | (K) | e-posta newsletter + sitemap-bulten |
| L7 | **Bookmarks / Follow / Notifications** | (K) | kişiselleştirilmiş `/feed` |
| L8 | **Gamification** (points/achievements/onboarding) | (K) | bağlılık katmanı |
| L9 | **Article JSON-LD + SearchAction + Google News sitemap** | (K) | SEO açığımızı kapatır (hreflang 24 dil) |
| L10 | **Content-Signal/robots + AI-bot kuralları + tuzak yol** | (K) | `/arsiv/dizin` trap, GPTBot/ClaudeBot engeli |
| L11 | **A11y font ölçekleme** + dark/light/system + PWA maskable | (K) | erişilebilirlik + PWA olgunluğu |
| L12 | AI **asistan** (okurken sor/özetle) + `safety`/`feedback` | (K) | içerik güvenliği/şikâyet |

## EKSİK / DOĞRULANACAK ALANLAR
- (S2) "Ulusal Gündem" ve "7- Hazırlanıyor" videolarının altyazısı henüz yok — kullanıcı verecek
- Panel kullanım kılavuzu (panel-kullanim-kilavuzu.html) taranmadı — ayrı kaynak
- Haber içi ileri özellikler (paywall, e-posta bülten) sayfada net değil
- Yorum SEO, üyelik detayları video ister

## İZLENECEK (kullanıcı adım adım verecek)
- Diğer YouTube videolarının CC altyazıları
- Görseller (reklam/marka örnekleri)
