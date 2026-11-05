# theblock — Spec Plan (Öncelikli Dalgalar P0→P4)

**Sürüm:** 0.1 — 2026-10-14
**Kararlar (kullanıcı):**
- Kapsam: **Hepsi (P0→P4)** — tüm özellikler dalga dalga planlanır
- AI: **OmniRoute varsayılan** + "API Anahtarı Ekle" özelliği (çoklu provider)
- Mimari: Go merkez + Laravel panel (geçiş dönemi) + Rust sıcak yol (görsel/arama) + edge cache (`00-mimari-gecis`)

**Referans dokümanlar:**
- `01-ozellik-envanteri.md` — 108 özellik özet envanteri
- `02-detayli-ozellik-spec-panel-ve-frontend.md` — geliştirme referansı (panel + frontend)
- `00-mimari-gecis-laravel-go-rust.md` — hedef mimari

---

## 1. İlkeler
1. **Okuma-önce taşı, yazma-sonra taşı** (strangler). Önce ziyaretçiye görünen, edge'de cache'lenebilen yol.
2. **Her dalga canlı biter**; her dalga sonunda site çalışır durumdadır (theblock.com.tr hep açık).
3. **Yeni geliştirme Go'da**; Laravel/panel ekosistemi yalnızca geçiş dönemi için.
4. **Rust yalnız sıcak yol**: görsel dönüşüm (image-svc) + Türkçe arama (search-svc). İş mantığı bunlarda büyümez.
5. **Edge önbelleği 1. sınıf vatandaş**: her sayfada cache stratejisi tasarlanır.
6. **Sırlar repoda değil**: API anahtarları `.env`/sır deposunda, kullanıcı tarafından panelden yönetilir.

---

## 2. Mimari Eşleme (modül → sahiplenme)
| Katman | Sahip | Sorumluluk (bu planda) |
|---|---|---|
| **Go site+API** (`go/`) | Ana üretim | SSR sayfalar, REST API, oturum, içerik CRUD, SEO, dashboard, AI orchestration, önbellek taktikleri |
| **Go worker'lar** (`go/cmd/workers`) | Ana üretim | RSS/ajans bot, AI içerik üretimi, sitemap/feed, bildirim, indeksleme, piyasa/veri çeken (hava/namaz/eczane/kur) periyodik işler |
| **Rust image-svc** (`rust/image-svc`) | Sıcak yol | Upload kırpma (16:9 vb.), WebP/AVIF dönüşüm, srcset, EXIF temizlik, "İnternette Ara" görsel indirme işleme |
| **Rust search-svc** (`rust/search-svc`) | Sıcak yol | Türkçe tam metin arama (Tantivy), ilgili içerik, sıcak indeks |
| **Laravel** (`laravel/`) | Geçiş dönemi | Panel (Faz-0 geçişi yoksa; başlangıçta yalnızca legacy)". Kademeli emekli |
| **PostgreSQL + Redis** | Veri | Şema (tek yazıcı: Go), önbellek + kuyruk (Redis Streams) |
| **Edge (Cloudflare)** | Altyapı | Tam sayfa cache, WAF, TLS, purge otomasyonu (kuruldu) |

---

## 3. Temel Veri Modeli (ilk dalga çekirdeği)
> Migrasyon sahibi Go olacak; şema `go/migrations` altında (detailed: `02`§1-2).

- `users` (RBAC: Yönetici/Editör/Muhabir/Yazar/Üye; 2FA) — `02`§14
- `content_types` (12 tür) + `contents` (polimorfik ana kayıt: statu=Yayında/Taslak/Planlı/Arşiv/Çöp, tür, başlık, spot, detay(JSON-blocks), slug, hit, yazar, mahreç, rozet bitleri AM/ÜM/DM/SD/FH/HK, ileri tarih)
- `categories` (düz), `tags`, `mahrets`, `pages` (genel/kurumsal), `media_files` (dosya yöneticisi), `galleries` (görsel+sıra+başlık+açıklama), `videos`
- `contents_extra` (tür-özel: anket cevapları/oy, firma rehberi, seri ilan, BİK kodu, e-gazete tarih/sayı)
- `comments` (Onaylı/Onay Bekliyor, IP), `events` (etkinlik+hatırlatma), `ads` (5 tip+alanlar+program), `redirects` (301), `counters_hit`/`share` (okunma+paylaşım sayaçları), `logs` (panel işlem logu)
- `headline_designs` (görsel manşet tasarım belgeleri: JSON katmanlar + şablon adı + kategori/brand varsayılanı) — `02`§8
- `settings` (genel/görünüm/modüller/izinler/bağlantılar/AI)- `ai_templates` (sohbet + sağ tık), `ai_conversations`, `provider_keys` (çoklu provider API anahtarları)
- `jobs`/`crawled_items` (bot: kaynak, hash, durum, kelime değiştirici), `services_cache` (hava/namaz/piyasa vb. saklanan veri)
- **P5 yayın ağı (spec `05`)**: `sites` (tenant) + `contents/categories/media/settings/ads.site_id` izolasyonu · `keyword_clusters` + `content_clusters` (konu kümesi/SEO) · `content_revenue` (gelir, RSS hesaplayıcı tablosu) — bkz. §P5

---

## 4. DALGA HARİTASI (P0→P4)

### 🟢 P0 — Yayın Çekirdeği (ilk canlı)
**Hedef:** Haber sitesi temel işlevlerle yayında; editör panel üzerinden içerik girer; SEO'ya uygun; edge cache aktif.

**Kapsam (ref: `02`§1-4,16,17,18,19,20):**
- [ ] İçerik modeli (12 tür) + tek akıllı form (başlık/spot/detay, 3 görsel kutusu, 16:9 otomatik kırpma, Dosyalarımdan Seç) — `image-svc` kırpma entegrasyonu
- [ ] Yayın yaşam döngüsü: Yayında/Taslak/Planlı/Arşiv/Çöp(30 gün) + 3 sn geri sayım onayı + ileri tarih
- [ ] Kategori (düz, görünürlük), Mahreç, Etiket, Sayfalar (kurumsal kilitli)
- [ ] Manşet sistemi: AM/ÜM/DM/SD/FH/HK rozetleri + anasayfa manşet modülleri (Ana/Üst/Dev) + **Görsel Manşet Editörü çekirdeği**: zemin/rozet/3 başlık katmanı, JSON tasarım belgesi, şablon kaydet/yükle, PNG/WebP render (`image-svc`) — `02`§8
- [ ] Foto galeri (dikey/yatay) + Video (dosya/YouTube) + Kısayol gömme
- [ ] SEO çekirdeği: Schema JSON-LD, sitemap + Google News sitemap, robots/ads.txt, 301, meta/canonical, SERP önizleme
- [ ] SEO puanı (10 kriter ağırlıklı) — 02§4
- [ ] Yorum sistemi (manuel onay) + RBAC kullanıcılar (5 rol)
- [ ] Dosya yöneticisi (upload, resim düzenle, internet arama)
- [ ] Dashboard: Keşfet + Hızlı Ekle
- [ ] Frontend çekirdeği: header, kategori nav, anasayfa blokları (son dakika şeridi, manşetler, kategori blokları, son eklenenler), makale detay, yorum
- [ ] Edge cache + purge otomasyonu (yayın→purge), PWA
- **Çıktı:** Sitesi yayında, 2 yönetici hesabı, 50 test haberi, Google Search Console doğrulanmış
- **Kabul:** theblock.com.tr'de editör girişi→haber yayın→5 sn içinde sitede; PageSpeed ≥ 90; p99 TTFB kriteri

### 🟡 P1 — AI Editör Asistanı + İçerik Akışı
**Hedef:** Editörler yapay zekâ ile hızlı üretim; ajans/RSS otomasyonu.

**Kapsam (ref: `02`§8,9,10,11,12,19,22):**
- [ ] AI katmanı tasarımı (aşağı §6): **OmniRoute varsayılan + API anahtarı ekle** (OpenAI/Anthropic/Gemini/özel URL)
- [ ] AI sohbet ekranı (sohbet geçmişi, hazır şablonlar, →Haber Yap)
- [ ] Editör içi sağ tık 12 komut + prompt özelleştirme + özel şablon ekleme
- [ ] Alan bazlı AI menüleri (başlık/spot/detay özgünleştir vb.)
- [ ] Dev Manşet AI "Öner", başlık otomatik tamamlama (Google suggest), etiket önerisi, büyük/küçük harf
- [ ] Sesli yazma (Otomatik + Toplantı modu)
- [ ] Google Trends modülü (~5 dk) + Ulusal Gündem (kaynak gruplu) + Trend/Gündem sağ-tık menüsü (Sor/Başlık Öner/Haber Yaz/Araştıra-Yaz)
- [ ] **Bot modülü**: 10 ajans, otomatik çekim, kelime değiştirici, AI Kaydet, otomatik ekleme
- [ ] Etkinlikler (hatırlatma şeridi + AI)
- [ ] Web push bildirim + PWA derinleştirme
- **Çıktı:** Editör "Trend → Araştır & Haber Yaz → yayın" akışını <2 dk'da yapar
- **Kabul:** AI çağrıları provider soyutlamasında; anahtar panelden yönetilir; token maliyet loglanır

### 🟠 P2 — Raporlama & Gelir
**Hedef:** Veriye dayalı karar + reklam yönetimi.

**Kapsam (ref: `02`§16,19 + `01`§B3):**
- [ ] GA4 entegrasyonu: Gerçek Zamanlı (120 sn) + gelişmiş raporlama (7/15/30 gün, 7 metrik, % değişim, cihaz dağılımı)
- [ ] **İçerik performansı** (ilk-parti): haber bazında görüntülenme/okuma süresi/etkileşim; **en çok okunan haber · kategori · yazar** (tarih aralığı + % değişim) — `counters_hit` + GA4 karışımı, **çift sayım yok**
- [ ] **AI kullanım raporu**: sağlayıcı · model başına çağrı / token / ortalama gecikme / tahmini maliyet + aylık toplam; 7/30 gün + özel aralık; CSV export
- [ ] AdSense API bağlama + panel kartı (₺, gösterim, tıklama, CTR, RPM, CPC; 3 saatlik okuma)
- [ ] Reklam yönetimi: 5 tip (görsel/kod/slayt/splash/sticky) + Reklam Alanları + kategori/tarih + Kod Oluşturucu
- [ ] Hit + paylaşım sayaçları (+ web widget yönetimi), çalışan editör analizi
- [ ] Yandex Metrica API (opsiyonel)
- **Çıktı:** Editör panoda trafik + gelir + AI maliyeti; reklam alanları CF "cache tag" ile ayrışır
- **Kabul:** GA "reklamsız demo" metriği; AdSense raporu panelde 3 saatte tazelenir; içerik performansı ve AI kullanım raporu seçilen aralıkta **tutarlı, mükerrer olay sayılmaz** (tek olay-tek satır)

### 🔵 P3 — Servis Modülleri (haber siteleri paketleri)
**Hedef:** Yerel/ulusal haber sitesi beklenen tüm servisler.

**Kapsam (ref: `02`§20,21 + `01`§H):**
- [ ] Hava Durumu (81 il; şehir widget'ı) · Namaz vakitleri (iftara kalan) · Nöbetçi Eczane (il/ilçe) · İmsakiye
- [ ] Piyasalar: Döviz/Altın/Borsa (BİST)/Kripto canlı bant + otomatik çevirici sayfa
- [ ] Sinemalar (film listesi + fragman) · Astroloji (12 burç) · Trafik
- [ ] Puan durumu & lig fikstürü (takım görselleri)
- [ ] Gazete Manşetleri (~30 gazete thumb) · E-Gazete (PDF, sayfa görselleri) · Resmi İlanlar (BİK) · Vefat
- [ ] Firma Rehberi (harita) · Seri İlanlar · Anketler · Arşiv · Yerel haberler (muhabir) · Çok dilli (24 dil çeviri)
- [ ] Sesli okuma (TTS) · Story/Hikaye · Canlı yayın & Seçim sayfası widget'ları · Infinity scroll · "Gözden Kaçmasın / İlginizi Çekebilir / Trend / Son Eklenenler" widget'ları (İzinler tablosu)
- **Çıktı:** Servis sayfaları data-worker'larıyla güncellenir; hepsi edge cache'te
- **Kabul:** Her servis `02`§21 kriterine uyar; kapma/veri kaynağı sürekliliği (worker)

### ⚫ P4 — Gelişmiş & Büyüme
**Hedef:** Ölçek, keşfedilebilirlik, rekabet gücü.

**Kapsam (ref: `01`§D,I,J):**
- [ ] search-svc (Tantivy) Türkçe tam metin arama + Algolia uyumluluğu seçeneği
- [ ] Google Indexing API otomatik; AMP; Search Console API entegrasyonu
- [ ] Sosyal medya otomatik paylaşım
- [ ] Abonelik/paywall, üye profili geliştirme, e-posta bülten (gerektiyse)
- [ ] Çapraz backlink sistemi; GEO/AI arama görünürlüğü derinleştirme
- [ ] Geçiş paketi: 100k içerik taşıma araçları, 301 link koruma, CSV/içe aktarma
- [ ] 2FA zorunluluk modu, güvenlik denetimi (govulncheck/cargo audit), chaos testi
- **Çıktı:** P0→P3'ten geçmiş ürün + büyüme modülleri
- **Kabul:** `04-DoD` kontrol listesi yeşil (00-mimari §17 temelinde)

### 💰 P5 — Çoklu Site Yayın Ağı & AdSense Maksimizasyonu (spec `05`)
**Hedef:** Tek CMS altyapısından çok sayıda konu kümesi sitesi (3 pilot: teknoloji/AI, finans, genel+rehber) yönetip site/URL bazlı reklam gelirini ölçüp maksimize etmek.

**Kapsam (ref: `05` §3):**
- [ ] **P5-A tenant altyapısı**: `sites` tablosu + `site_id` izolasyonu (DEFAULT 1 geçişi), panel üstü site seçici, domain→site eşleme
- [ ] **P5-B konu kümesi SEO**: `keyword_clusters` + içerik eşleme, çakışma/cannibalization uyarısı, iç-link önerisi, evergreen + güncelleme önerisi
- [ ] **P5-C AI editör**: özgünlük/kaynak doğrulama ekranı, YMYL rozeti (finans/sağlık kısıtı), AI güncelleme taslağı (editör onaylı)
- [ ] **P5-D gelir analitiği**: AdSense+GA site/URL bazlı (RPM, gelir, gösterim), içerik başına kâr = gelir − AI maliyeti, PV×RPM simülatörü, CSV
- [ ] **P5-E politika denetimi**: telif/kaynak/yazar kontrol listesi, KVKK/Gizlilik/Kullanım sayfa seed'leri, reklam yerleşimi + YMYL yoğunluk uyarısı, site bazlı ads.txt/robots/sitemap
- [ ] **P5-F hesaplayıcı içerik türü**: finans/tadilat widget (JSON şema, `contents_extra`), çalışan frontend hesaplayıcı
- [ ] **P5-G 3-site pilot dashboard**: site × trafik/RPM/gelir/maliyet karşılaştırması + hedef yönetimi
- **Çıktı:** Aynı panelden ≥2 site yayında; site/URL gelir raporları; hesaplayıcı araçlı içerikler
- **Kabul:** izolasyon testi (site seçici → yalnız o sitenin verisi); YMYL taslağı onaysız yayınlanmaz; kâr satırı kural tabanlı doğru; AdSense verisi yoksa "demo" rozeti

---

## 5. Bağımlılık Haritası
```
P0 ────────► P1 ────────► P2 ────────► P3 ────────► P4
 ├ şema, form, SEO       ├ AI katmanı      ├ GA/AdSense   ├ search-svc
 ├ edge cache            ├ bot (worker)    ├ reklam       ├ indexing API
 └ image-svc(16:9)       └ trends (worker) │              └ taşıma
```
- P1, AI sohbeti olmadan da başlayabilir (önce Trends/bot altyapısı)
- P3'ün piyasa/hava servisleri P1 worker mimarisiyle (çekme/go worker) paralel gider
- search-svc (P4) image-svc (P0) ile aynı Rust servis yatağında; erken scaffold (healthz kuruldu)

---

## 6. AI Sağlayıcı Mimarisi (OmniRoute + çoklu anahtar)
```
Panel (Ayarlar → Yapay Zeka)
   ├─ Sağlayıcı seçimi: OmniRoute (varsayılan) · OpenAI · Anthropic · Gemini · Özel (OpenAI-uyumlu URL)
   ├─ API Anahtarı (sır deposunda; asla DB/log'da düz)
   └─ Model + prompt şablonları (yer tutucu: [ilce][sehir][tarih][site_adi] / {kaynak}{secilen})

Go AI aracısı (provider abstraction):
   Provider{ Name, BaseURL, Model, Key() }
   chat(ctx, provider, messages, tools) → response
   - OmniRoute: openai-compatible endpoint, /v1/chat/completions destekli
   - OpenAI: aynı uyum; Anthropic: messages API; Gemini: genai API
   - Tarama: "Araştır & Yaz" → web arama (provider tool veya arama api) + kaynak toplama
   - Token sayaç + maliyet logu; hata fallback: sağlayıcı_1 başarısız → sağlayıcı_2
```
**Gereksinimler:**
1. Varsayılan `OMNIROUTE_API_KEY` (`deploy/.env`), kullanıcı panelden ek anahtar ekleyebilir
2. `provider_keys` tablosu: tip (string enum), şifreli (AES/HSM) — Go tarafında şifre çözülür, loglanmaz
3. Prompt şablonları sürümlü (editörde "geçerli prompt" gösterimi; Varsayılanları Geri Yükle)

---

## 7. Kapı & Onay Süreci (AGENTS.md uyumu)
- Her dalga sonunda `ILERLEME.md`'ye kanıtlı giriş (komut + çıktı)
- `main`'e birleşme kullanıcı onayıyla; üretime deploy yalnız onayla
- Her bilinen hata için önce kırmızı test (silinmez)
- Sırlar yalnız `.env`/sır deposu (bu doküman ve spec'ler sır içermez)

---

## 8. Hemen Sıradaki Adımlar
1. **P0 şema+migration** (Go): `contents`, `categories`, `media_files`, RBAC, `settings`, `redirects` + ilk göç
2. **P0 form/API**: Go'da içerik CRUD (Laravel değil; yeni geliştirme Go)
3. **image-svc**: 16:9 kırpma + WebP dönüşüm (worker çağırır)
4. **SSR + edge cache**: makale detay + anasayfa (cache-tag + purge)
5. Seçenek: referans ürünün `panel-kullanim-kilavuzu` akışını P0 paneline birebir taşı
