# theblock — Çoklu Site Yayın Ağı & Google AdSense Gelir Modeli Spec

**Sürüm:** 1.0 — 2026-10-14
**Kaynak:** Kullanıcı iş-modeli analizi (sektör seçimi, AdSense RPM hesaplaması, çoklu site stratejisi)
**İlişkili:** `01-ozellik-envanteri` §K, `03-spec-plan` P5, `02` §18-19-20-21 (ayarlar/reklam/raporlama)

---

## 1. Amaç
theblock yalnız tek haber sitesi değil; **tek AI-CMS altyapısından çok sayıda konu kümesi sitesi** yönetebilen, her sitede Google organik trafiği ve AdSense reklam gelirini maksimize etmek için ölçüm/denetim araçları sunan bir **yayın ağı sistemi** hâline gelir.

**İş modeli ilkeleri (kullanıcı analizinden):**
- Kazanç, en çok içerik yayınlayandan değil, **ticari değeri yüksek ziyaretçi çeken + özgün içerik üreten** yayıncıdan gelir.
- 10 site açmak ≠ 10× gelir; **3 farklı yayın modelini kontrollü pilot** olarak test et: (1) B2B teknoloji/AI, (2) finans & ekonomi, (3) genel haber + sürekli-değerli rehberler.
- Tahmin modeli: **Gelir = Sayfa Görüntüleme × RPM / 1000** (AdSense sayfa RPM tanımı). Örnek kademeler: 100k PV @ $1–$8 RPM → $100–$800 ay.
- Google Ads tıklama maliyeti ≠ yayıncı geliri; RPM yayıncı tarafı metriğidir.
- **YMYL alanlarında** (finans/sağlık) sağlık/finansal doğruluk ve kaynak kalitesi şart; doğrulanmamış otomatik tavsiye üretilmez.
- Google: AI-üretim tek başına yasak değildir; **kullanıcıya değer katmayan toplu özgün olmayan sayfa** spamdir. Aynı içerik çoklu sitede çoğaltılmaz.

## 2. Hedef sektörler (içerik stratejisi)
| # | Sektör | İçerik türleri | Parasal fırsat | Risk/YMYL |
|---|---|---|---|---|
| 1 | Finans, bankacılık, kişisel ekonomi | kredi maliyeti, mevduat faizi, ücret karşılaştırma, enflasyon hesaplayıcı | Yüksek ticari niyet | Yüksek rekabet; doğruluk zorunlu → **YMYL** |
| 2 | B2B teknoloji, yapay zekâ, siber güvenlik | AI/hosting/VPS karşılaştırma, SaaS inceleme, dev aracı rehberleri | Yüksek; sponsorluk+affiliate | Düşük; CMS ile uyumlu |
| 3 | Gayrimenkul, konut, ev tadilatı | kira, tadilat maliyeti, ısı yalıtımı, yerel arama | Yerel trafik + lead | Orta |
| 4 | Eğitim, meslek, kariyer | sertifika, yazılım öğrenme, sınavlar | Eğitim reklamları | Düşük |
| 5 | Turizm, otel, seyahat | şehir rehberi, otel karşılaştırma, vize | Otel/ULAŞIM + OTA affiliate | Ülke/ziyaretçi profiline bağlı |
| 6 | Sağlık, beslenme, yaşam | bilim açıklaması, uyku, egzersiz | Orta | **YMYL**; kaynak şart |
| 7 | Tüketici teknolojisi | telefon/PC karşılaştırma, test, satın alma rehberi | Elektronik + affiliate | Gerçek test şart |

## 3. Teknoloji gereksinimleri (feature set → spec)

### A. Çoklu site / kiracı (network) altyapısı — `P5-A`
- `sites` (tenant) kaydı: ad, marka, alan adı (domain), varsayılan dil, tema + **tenant_id izolasyonu** (ilk tasarımdan; şema/seed'de `site_id DEFAULT 1`).
- **İzolasyon kuralları:** tüm içerik/kategori/medya/ayar/reklam/analitik kayıtları `site_id` taşır; panel üstte **site seçici** (varsayılan + "Tümü/karşılaştır"); kamuya açık çıktı yalnızca sitenin kayıtlarını yayınlar.
- Domain/baseURL eşleme: istek host'u → site çözümü (varsayılan site + yönlendirme; `APP_BASE_URL` per site).
- **Pilot modeli:** 3 site (teknoloji, finans, genel) için site bazlı hedef ve yayın akışı.

### B. Konu kümesi & SEO yönetimi — `P5-B`
- `keyword_clusters`: cluster (ana), partner/pillar sayfalar, arama niyeti, hedef anahtar kelime, `canonical` satırı; içerik ↔ cluster eşleme.
- **İçerik çakışması / cannibalization uyarısı**: aynı cluster'da benzer başlık/anahtar kelime tespiti (panelde ikaz).
- **İç link önerileri**: cluster içi pillar→child ve child→child link önerileri (SEO paneli), sitemap/canonical site bazlı.
- Sürekli-değerli (evergreen) içerik bayrağı + **güncelleme önerisi** (yaşlı içerik → AI güncelleme taslağı).

### C. AI editör geliştirmeleri — `P5-C`
- **Özgünlük kontrolü**: kaynak metne karşı fark ekranı + orijinallik indeksi (kural tabanlı benzerlik).
- **Kaynak doğrulama**: bağlantılı kaynaklar liste halinde saklanır (`article_sources` yeniden canlanır); http-erişilebilirlik/referans kuralı.
- **YMYL rozeti**: finans/sağlık türü + doğrulama alanı (uzman/veri kaynağı/durum) → otomatik üretim kısıtı.
- Güncelleme önerisi AI taslağı **daima editör onayına girer** (otomatik yayın yok — mevcut ilke).

### D. Gelir analitiği — `P5-D`
- **AdSense API** site + URL bazlı: gelir, gösterim, tıklama, CTR, RPM, CPC (mevcut P2 §AdSense kartı geliştirilir: per `site_id`).
- **Sayfa bazlı gelir**: URL ↔ `counters_hit` eşleme ile içerik başına RPM tahmini; **en-içerik-kazancı** listeleri.
- **İçerik maliyeti**: AI token maliyeti (mevcut `ai_usage_logs`) → içerik başına üretim maliyeti; **kâr = gelir − maliyet** satırı.
- **Gelir hesaplayıcı (planlama)**: PV × RPM × site sayısı simülatörü (panel raporu; kullanıcı analizindeki tablo ölçeğinde).
- CSV dışa aktarım (per site/URL).

### E. Yayın & politika denetimi — `P5-E`
- **Telif/sağlık notu**: içerik denetim listesinde telif, yazar, tarih, kaynak, kalite onayı.
- **Gizlilik/KVKK**: varsayılan `gizlilik` + `kvkk` + `kullanim-kosullari` kurumsal sayfa seed'leri (02 §17 sayfaları).
- **Reklam yerleşimi kontrolü** per site: `ads` + alanlar `site_id` ile; YMYL sayfalarında reklam yoğunluğu uyarısı.
- Per site `ads.txt` / `robots.txt` / sitemap.

### F. Hesaplayıcı içerik türü — `P5-F`
- Yeni `content_types` türü: **hesaplayici (calculator)** — widget JSON (alanlar/formül/çıktı etiketi); finans (mevduat/kredi/tadilat) ve karşılaştırma araçları.
- Frontend widget'ı: çalışan hesaplayıcı + sonuç; içerik formunda `contents_extra.extra` altında şema saklanır (mevcut JSONB).

### G. İş modeli araçları — `P5-G`
- **3-Site Pilot Dashboard**: site × trafik/RPM/gelir/maliyet karşılaştırma kartları; "kazananı büyüt, kaybedeni iyileştir/birleştir" karar desteği.
- Site/hedef ayarları (aylık PV hedefi, hedef RPM, içerik bütçesi).

## 4. Veri modeli eklentileri
```sql
-- P5 (spec 05) — yayın ağı
CREATE TABLE sites (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name         TEXT NOT NULL,
    domain       TEXT NOT NULL UNIQUE,
    brand        TEXT,
    default_locale TEXT NOT NULL DEFAULT 'tr',
    status       TEXT NOT NULL DEFAULT 'aktif',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- mevcut tablolara site_id sütunu (geçiş: DEFAULT 1, tek yazıcı Go):
--   contents.site_id, categories.site_id, media_files.site_id, settings.site_id,
--   ads.site_id, keyword_clusters.site_id
CREATE TABLE keyword_clusters (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL,
    name        TEXT NOT NULL,
    intent      TEXT,               -- bilgi/ticari/navigasyon
    pillar_id   BIGINT REFERENCES contents(id),   -- pillar içerik
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- content ↔ cluster
CREATE TABLE content_clusters (
    content_id BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    cluster_id BIGINT NOT NULL REFERENCES keyword_clusters(id) ON DELETE CASCADE,
    PRIMARY KEY (content_id, cluster_id)
);
-- gelir eşlemesi (AdSense/GA getirisi doldurur; manuel düzeltme de olur)
CREATE TABLE content_revenue (
    content_id BIGINT PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    revenue_est NUMERIC(12,4) NOT NULL DEFAULT 0,
    impressions BIGINT NOT NULL DEFAULT 0,
    rpm_est     NUMERIC(8,2) NOT NULL DEFAULT 0,
    cost_est    NUMERIC(12,4) NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```
> Göç stratejisi: `site_id DEFAULT 1` ile tek site kullanıcısı etkilenmez; P5'te çoklu site arayüzü gelince izolasyon aktifleşir. (Ön koşul: schema `01`/`03` core; şema sahibi Go.)

## 5. Kabul kriterleri
1. Panel üst site seçici: site değişince içerik/rapor/küme verileri **yalnız o siteye** döner (izolasyon testi).
2. İçerik, kayıt anında `site_id` alır; farklı sitelerin URL'leri birbirine karışmaz (domain eşleme testi).
3. AdSense/GA raporu **site ve URL bazında** (RPM, gelir) gösterilir; veri yoksa "demo" rozeti (mevcut P2 ilkesi).
4. İçerik kârı satırı = taşınan gelir − hesaplı AI/maliyet (en az kural tabanlı).
5. YMYL türlerinde AI taslağı "doğrulanmadı" rozeti ile gelir; editsör onayı şartsızdır (yayınlanmaz).
6. Hesaplayıcı içerik: kaydedilip yeniden açılınca formül/widget korunur ve sayfada çalışır.

## 6. Kapsam dışı (v1 için)
- Gerçek çoklu-veri merkezli mikroservis sıralaması (tek codebase, tenant modeli yeterli).
- Sağlık/finans alanında otomatik, doğrulanmamış tavsiye üretimi.
- Google'a spam niteliğinde toplu link/ikiz sayfa üretim araçları (politika gereği kapalı).

## 7. Açık sorular (netleştirilince plan daralır)
- AdSense otorizasyonu: GSC + AdSense API client (kullanıcı hesabı) — test kapsamı ne?
- Site başına ayrı tema mi yoksa tek tema + marka değişimi mi (v1 önerisi: tek tema + renk/logo/domain)?
- 3 pilotun alan adları ve başlangıç içerik bütçeleri.
