# Performans kazançları — ölçüm raporu

**Tarih:** 8 Kasım 2026 · **Makine:** Ryzen 5 7500F, 12 vCPU, WSL 20 GB
**Yöntem:** canlı servise yük testi + gerçek profil + PostgreSQL sorgu sayımı.
Aşağıdaki her sayı bu makinede ölçüldü.

İlgili analiz: [`RUST-GECIS-ANALIZI.md`](RUST-GECIS-ANALIZI.md)

---

## 1. Sonuç tablosu

Aynı yük testi (50 eşzamanlı istemci × 10 sn, 6 sayfa, aynı betik):

| Metrik | Önce | 1. tur | 2. tur | **3. tur** |
|---|---|---|---|---|
| rps | 435 | 1.810 | 1.515 | **1.945** |
| p50 | 96 ms | 25 ms | 30 ms | **24 ms** |
| p95 | 211 ms | 51 ms | 57 ms | **45 ms** |
| p99 | 270 ms | 59 ms | 72 ms | **57 ms** |
| Hata | 0 | 0 | 0 | **0** |

> Turlar arasında ölçüm koşulları değişti (2. turda arka planda başka iş
> vardı, ayrıca çalışma ağacına başka bir geliştirme girdi). Bu yüzden
> "1. tur" ile "3. tur"u doğrudan kıyaslamak yerine her turun kendi
> **önce/sonra** ölçümü esas alınmalı.

Ana sayfa, aynı yük altında (p50): **191 ms → 51 ms → 52 ms → 35 ms**.

Sayfa bazında (3. tur, yük altında, p50):

| Sayfa | Önce | 3. tur |
|---|---|---|
| `/` (ana sayfa) | 191 ms | **35 ms** |
| `/price` | 163 ms | **25 ms** |
| `/markets` | 87 ms | **23 ms** |
| `/videos` | 95 ms | **28 ms** |
| `/canli-yayin` | 91 ms | **22 ms** |
| `/business` | 87 ms | **18 ms** |

Tek istek (boşta, `curl`): ana sayfa **26 ms → 9 ms**.

### Bellek

| | Önce | Sonra |
|---|---|---|
| Boşta RSS | 34,6 MB | **29–33 MB** |
| Yük altında tepe | 44,4 MB | 55–74 MB |

Tepe değer yükseldi çünkü **aynı 10 saniyede 4 kat fazla istek** işlendi
(uçuştaki istek sayısı arttı). İstek başına maliyet düştü; boşta RSS de azaldı.
3. turdaki 73,8 MB tepe, önbelleğin ilk dolduğu anda 50 eşzamanlı isteğin
aynı veriyi beklerken tuttuğu ara belleğe denk geliyor; GOMEMLIMIT (256MiB)
bu türden bir sıçramanın üstünde duruyor.

---

## 2. Ne yapıldı

### 2.1 `pprof` açıldı — ayrı dinleyicide

`PPROF_ADDR=127.0.0.1:6060`. **Ana mux'a bağlanmadı**: pprof çıktısı heap
içeriği, goroutine yığınları ve dosya yolları taşır; site `demo.bitcosar.com`
tünelinden erişilebilir olduğu için genel mux'a eklenmesi bunları herkese
açardı. Doğrulandı: `:6060/debug/pprof/` → 200, `:8090/debug/pprof/` → **404**.
Boş bırakılırsa hiç açılmaz (üretimde varsayılan kapalı).

Bununla ilk gerçek profil alındı; aşağıdaki iki N+1'in kaynağı buradan ve
`log_statement=all` sayımından çıktı.

### 2.2 Ana sayfada N+1: kategori eşiği SQL'e taşındı

`HomeHighlights`, son 48 içeriği çektikten sonra **her içerik için ayrı sorgu**
ile `homepage_min_score` eşiğini kontrol ediyordu (`homepageAllowed`).
48 içerik = 48 ek gidiş-dönüş.

Kapı tek sorguya alındı (`recentPublishedHome`):

```sql
AND (COALESCE(c.ai_score, 0) >= COALESCE(cat.homepage_min_score, 0))
```

Eşdeğerlik korundu: eski kod kategori bulunamazsa **izin veriyordu**;
`COALESCE(...,0)` bu davranışı birebir sürdürür. Canlı veriyle doğrulandı:
eski algoritma 48, yeni tek sorgu 48 satır döndürdü.

### 2.3 Ana sayfada N+1: YouTube ayarları 30 sorgudan 1'e

Ana sayfada üç video modülü (video / klipler / canlı) aynı ayarları soruyordu
ve `AyarlariOku` her alanı **ayrı sorguyla** okuyordu: 10 alan × 3 modül =
**istek başına 30 sorgu**.

Ölçüm (`log_statement=all`, tek ana sayfa isteği): toplam 54 sorgunun 30'u
buydu. İki katmanlı düzeltme:

1. `repo.GetSettingsText` — bölümün tamamı **tek sorguda**; saklanan jsonb
   sarmalı (`"bitcosar"` → `bitcosar`) burada çözülür, böylece tek sorguya
   geçen her çağıran aynı çözümü tekrar yazmaz.
2. `cachedYtAyar` — üç modülün üç okuması 10 sn TTL'li tek okumaya iner.
   Panelde kaydetme `invalidateContentCache()` çağırır, değişiklik anında görünür.

Sonuç: tek ana sayfa isteğinde **54 → 24 sorgu**, istek başına PostgreSQL
transaction sayısı **≈295 → 2**.

### 2.4 Rust `image-svc` devreye alındı

Servis aylardır çalışıyordu ama Go tarafında **tek bir referansı yoktu**.
Bağlandı (`IMAGE_SVC_URL`), kendi systemd birimi ve portu (8172) ile — theblock
deposunun servisine (8170) dokunmadan.

Ölçüm (1600×900 JPEG → hedef genişlik, JPEG q82):

| İş | Yerel Go | image-svc | Fark |
|---|---|---|---|
| `ResizeToWidth` 1200 px (HTTP turu dahil) | 79,2 ms | **46,0 ms** | 1,7× |
| `CreateThumb` 480 px kare | 39,3 ms | **22,3 ms** | 1,8× |
| Servis içi işlem (1200 px) | 109 ms | **45 ms** | 2,4× |

Fark dil değil kodlayıcı: Go'nun `image/jpeg`'i saf Go; Rust tarafı SIMD'li
`zune-jpeg` kullanıyor. Çıktı kalitesi birebir kalsın diye servise `quality`
parametresi eklendi (yerel yoldaki q82) ve ölçekleme çekirdeği Go'daki
`CatmullRom` ile aynıya alındı (`Triangle` srcset varyantları arasında
keskinlik farkı yaratıyordu).

**Güvenlik ve dayanıklılık:**
- Yalnız **JPEG** girdi gönderilir. PNG/WebP yerel yolda kalır: servis
  saydamlığı beyaz zemine çevirmiyor, renk değişirdi.
- Her hata durumunda **yerel Go yoluna düşer** — servis çökmesi görsel
  üretimini durdurmaz. Testle korunuyor.
- Servis hata verdiğinde **30 sn devre dışı** kalır; yoksa ölü servis her
  görselde 5 sn zaman aşımı ekleyip kazancı kayba çevirirdi. Testle korunuyor.

### 2.5 Veritabanı havuzu

`config/pool.go` yazılmış ama `app.env`'de hiçbir değer tanımlı değildi →
pgx varsayılanı `max(4, NumCPU)` = 12 bağlantı, 20+ arka plan işçisi varken.
Artık açık ayarlı. Aynı yük testiyle boyut denendi:

| `DB_MAX_CONNS` | rps | p50 | p95 |
|---|---|---|---|
| 16 | 1.252 | 33,0 ms | 69,4 ms |
| **24** | **1.478** | **29,5 ms** | **61,5 ms** |
| 32 | 1.502 | 28,7 ms | 62,9 ms |

24 seçildi: 32 ile fark ölçüm gürültüsü içinde ve PostgreSQL
`max_connections=100` sınırında arka plan işçilerine yer bırakıyor.

### 2.6 Ana sayfa okumaları paralelleştirildi

Profil şunu gösterdi: istek başına **CPU yalnızca ~4 ms**, kalan süre
PostgreSQL gidiş-dönüşlerinin **beklenmesi**. Ana sayfa ~24 sorguyu sırayla
çalıştırıyordu; her tur ~2 ms.

Birbirine bağımlı olmayan 10 okuma (düzen, menü, footer, hava, kurlar,
şehirler, etkinlikler, kategoriler, navigasyon, manşet havuzu) artık birlikte
uçuyor; ardından gelen 3 okuma (en çok okunan + iki galeri) da kendi içinde
paralel. Sıralı derinlik 24 turdan 2 tura indi.

| Eşzamanlı istemci | Ana sayfa p50 | p95 |
|---|---|---|
| 1 | **9,0 ms** | 10,9 ms |
| 4 | **10,0 ms** | 13,0 ms |
| 12 | **17,5 ms** | 23,3 ms |

Tek istekte 26 ms → **9 ms**. 50 eşzamanlı istemcide ise kazanç görünmüyor
(p50 55 ms): orada darboğaz artık **bağlantı havuzu**, istek içi sıralama
değil. Bu yüzden 50-eşzamanlı tablodaki asıl kazanç 2.2–2.4'ten geliyor;
paralelleştirme gecikmeyi tek/az istemcide düşürüyor.

**Görsel regresyon kontrolü:** eski sürüm (commit `2fa02fd`) ayrı worktree'de
derlenip 8091'de çalıştırıldı ve 5 sayfanın HTML'i bayt bayt karşılaştırıldı.
Sonuç: **tüm sayfalar aynı boyutta ve aynı içerikte**; tek fark süreç başına
değişen statik varlık sürümü (`?v=...`). Paralelleştirme çıktıyı değiştirmedi.

`go test -race` ile de doğrulandı (eşzamanlı okumalar veri yarışı üretmiyor).

### 2.7 CD yüzeyine 10 sn'lik süreç içi önbellek + singleflight

Profil ve `pg_stat_user_tables` sayımı, asıl kalan maliyetin **aynı veriyi
tekrar tekrar okumak** olduğunu gösterdi. Ölçüm (10 istek başına `contents`
tablosu taraması):

| | Önce | Sonra |
|---|---|---|
| Ana sayfa | **34 tarama/istek** | **4** |
| `/markets` | **35 tarama/istek** | **~0** |

İki katman eklendi (`internal/httpapi/ttlcache.go`):

1. **`ttlcache[T]`** — jenerik, TTL'li, süreç içi önbellek.
2. **Singleflight** — aynı anahtar için eşzamanlı çağrılar **tek** yükleme
   yapar, diğerleri onu bekler. 50 eşzamanlı yükte önbellek boşken oluşan
   sürü (thundering herd) tam olarak yük testinde görülen şeydi.

Önbelleğe alınanlar:
- **Kabuk verisi** (popüler etiketler + menü coinleri + bölüm doluluğu):
  her CD sayfasında aynı, istekten bağımsız. Üç okuma ayrıca kendi içinde
  paralel çalışıyor (önbellek boşken 3 sıralı tur yerine 1 tur).
- **Ana sayfa içerik listesi** (60 satır).
- **Bölüm listeleri** (bölüm + sayfa anahtarıyla).

**Tazelik:** HTTP yanıtı zaten `max-age=60, s-maxage=300`. 10 sn'lik süreç içi
katman kullanıcıya görünen tazeliği değiştirmez. Panelden yapılan
ayar/içerik değişiklikleri `invalidateContentCache()` ile anında yansır.

**Hata davranışı:** başarısız yükleme (ok=false) **önbelleğe yazılmaz** —
geçici bir DB hatası 10 sn boyunca boş sayfa olarak donmaz, sonraki istek
yeniden dener. Testle korunuyor.

**İptal:** bekleyen bir çağıranın bağlamı iptal olursa beklemeden döner;
terk edilmiş istek, yavaş sorgu bitene kadar goroutine tutmaz. Testle korunuyor.

**Doğrulama:** soğuk (önbellek boş) ve sıcak (önbellekten) yanıtlar 5 sayfada
**bayt bayt aynı** — önbellek render çıktısını değiştirmiyor.

---

## 3. Eklenen testler

| Test | Neyi korur |
|---|---|
| `TestResizeToWidthServisAdresiniKullanir` | servis devredeyken iş oraya gider, `quality=82` ve doğru `ratio` geçer |
| `TestImageSvcHataYerelYolaDuser` | servis 500 dönerse çıktı yine üretilir |
| `TestImageSvcHatadanSonraTekrarDenenmez` | ölü servis her görselde zaman aşımı eklemez |
| `TestImageSvcPNGYereldeKalir` | saydamlık bozulmaz |
| `TestImageSvcKapaliVarsayilan` | adres yoksa hiçbir şey değişmez |
| `TestGetSettingsTextSarmaliCozer` | jsonb sarmalı çözülür (`"x"` → `x`) |
| `TestGetSettingsTextTekSorgu` | N+1 geri gelmez (tek sorgu) |
| `TestTTLCacheHitYukleyiciyiTekrarCagirmaz` | taze değer varken DB'ye gidilmez |
| `TestTTLCacheSuresiDoluncaYenidenYukler` | TTL geçince tazelenir |
| `TestTTLCacheBasarisizSonucOnbellegeYazilmaz` | DB hatası 10 sn donmaz |
| `TestTTLCacheSingleflight` | 20 eşzamanlı çağrı = 1 yükleme |
| `TestTTLCacheIptalBeklemeyiBiraktirir` | iptal edilen istek beklemede kalmaz |
| `BenchmarkResizeToWidth*`, `BenchmarkCreateThumb*` | görsel kazancı ölçülebilir kalır |
| `BenchmarkRenderDesign` | manşet raster'ının maliyeti (10,1 ms) takip edilir |

Tüm paket testleri geçiyor: `go test -short ./...` → 23 paket **ok**,
`go test -race` (httpapi/repo/youtube/platform) temiz, `gofmt -l .` boş,
`go vet ./...` temiz.
Ek olarak eski sürümle **HTML karşılaştırması** yapıldı: kabuk değişiklikleri
(piyasa şeridi, arama çipleri) başka bir geliştirmeden geliyor; benim
değişikliklerim bunların dışında çıktıyı değiştirmedi.

---

## 4. Yapıldı: bellek emniyeti (kazanç değil, koruma)

`GOMEMLIMIT` hem yerel systemd biriminde (256MiB) hem k8s manifestinde
(220MiB) ayarlandı. Pod limiti 256Mi; Go konteyner limitini görmediği için
limit aşımı GC'ye değil **OOMKill**'e gidiyordu. Normal kullanımın çok
üstünde duruyor — kırpma değil, emniyet kemeri.

## 5. Sırada ne var (ölçümle doğrulanmış adaylar)

1. **Şablon render'ı artık listenin başında.** Sorgu sayısı düştükten sonra
   ana sayfada kalan sürenin büyük kısmı templ render + CSP hash üretimi.
   Profilde `indexFold` (CSP taraması) tek başına ~%9 CPU alıyor.
   Ölçülebilir aday: CSP hash'lerini gövde başına **bir kez** üretmek yerine
   yanıtla birlikte önbelleğe almak (aynı HTML → aynı hash).
2. **`/media/thumb` ve `/media/variant`**: ilk istekte üretiliyor, sonra
   diskten. Medya tablosu şu an boş; içerik dolduğunda `image-svc`
   kazancı (1,8×) doğrudan kullanıcıya yansıyacak.
3. **`pgx.Batch`**: ana sayfada hâlâ ~20 ayrı sorgu var (artık önbellekli,
   ama önbellek boşken hepsi tek turda uçuyor). Tek batch'e indirmek soğuk
   isteği kısaltır; sıcak yolda etkisi yok.
4. **`platform/render.go`** (583 satır, JSON tasarım → JPEG): ölçüldü
   **10,1 ms/raster**, ama **sıcak yolda değil** (kapak/manşet üretimi, panel
   ve arka plan işleri). Rust'a taşımak için uygun bir öğrenme parçası;
   performans gerekçesi zayıf. Karar: ihtiyaç doğunca.

---

## 6. Yeniden üretme

```bash
# yük testi (50 eşzamanlı × 10 sn)
python3 /mnt/c/Users/alper/AppData/Local/Temp/opencode/load2.py

# görsel hattı karşılaştırması
IMAGE_SVC_URL=http://127.0.0.1:8172 \
  go test -run XXX -bench ResizeToWidth -benchtime 10x ./internal/platform

# istek başına sorgu sayımı
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/perquery2.sh

# düşük eşzamanlılık (istek içi paralelliği görür)
python3 /mnt/c/Users/alper/AppData/Local/Temp/opencode/lowload.py

# görsel regresyon: eski commit'i ayrı portta çalıştırıp HTML diff
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/diffhtml.sh

# önbellek doğrulaması: soğuk (önbellek boş) = sıcak (önbellekten)
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/cacheverify.sh

# contents tablosu tarama sayısı (istek başına DB yükü)
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/scans.sh

# profil
curl -sS -o /tmp/cpu.pprof "http://127.0.0.1:6060/debug/pprof/profile?seconds=8"
go tool pprof -top bin/site /tmp/cpu.pprof
```
