# Performans kazançları — ölçüm raporu

**Tarih:** 8 Kasım 2026 · **Makine:** Ryzen 5 7500F, 12 vCPU, WSL 20 GB
**Yöntem:** canlı servise yük testi + gerçek profil + PostgreSQL sorgu sayımı.
Aşağıdaki her sayı bu makinede ölçüldü.

İlgili analiz: [`RUST-GECIS-ANALIZI.md`](RUST-GECIS-ANALIZI.md)

---

## 1. Sonuç tablosu

> ### Ölçüm yöntemi düzeltmesi (9 Kasım 2026)
>
> İlk turlarda yük üreteci **Python thread'leriydi**. 1.500 rps'nin üstünde
> Python kendisi CPU'nun %79'unu yiyordu, yani ölçülen "rps" büyük ölçüde
> **istemciyi** ölçüyordu. Go ile yazılmış sabit-eşzamanlılık yük üreteci
> (`/tmp/loadgen`) ile aynı ortamda gerçek tavan **~1.400 rps** çıktı.
>
> Bu yüzden 1.400 rps üstündeki karşılaştırmalar **gürültüye** dayanıyor ve
> aşağıdaki tablodaki 1.810/1.945 değerleri **abartılı** okunmalıdır. Sağlam
> olan ölçümler:
>
> | Ölçüm | Değer | Neden güvenilir |
> |---|---|---|
> | Tek istek gecikmesi (ana sayfa) | 26 ms → **9 ms** | istemci etkisi yok |
> | İstek başına sunucu CPU'su | 2,53 → **2,30 ms** | `/proc/<pid>/stat` ile süreç bazlı |
> | DB taraması / istek | 34 → **4** | `pg_stat_user_tables` |
>
> 50 eşzamanlıda ölçülen rps **istemciye bağlı** olduğu için artık birincil
> metrik değil; yerine **istek başına sunucu CPU'su** kullanılıyor.

İlk ölçüm turu (50 eşzamanlı istemci × 10 sn, 6 sayfa, Python üreteç):

| Metrik | Önce | 1. tur | 2. tur | 3. tur |
|---|---|---|---|---|
| rps (istemci sınırlı) | 435 | 1.810 | 1.515 | 1.945 |
| p50 | 96 ms | 25 ms | 30 ms | 24 ms |
| p95 | 211 ms | 51 ms | 57 ms | 45 ms |
| p99 | 270 ms | 59 ms | 72 ms | 57 ms |
| Hata | 0 | 0 | 0 | 0 |

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

### 2.8 gzip seviyesi 6 → 4 — istek yolundaki en pahalı tek kalem

Sorgu sayısı ve CSP düştükten sonra alınan profil (10 sn, yük altında,
20 eşzamanlı) **332% CPU** örnekledi ve dağılım şöyleydi:

| Kalem | CPU payı |
|---|---|
| **`compress/flate` (gzip kodlama)** | **%31,4** |
| sistem çağrıları (DB/ağ) | %9,5 |
| `memmove` | %5,0 |
| GC taraması | ~%6 |
| `sha256` (CSP hash) | %2,0 |

Yani gzip, her HTML yanıtını yeniden sıkıştırmak istek yolunun **en pahalı
tek işi** hâline gelmişti.

Gerçek 77 KB sayfa gövdesi üzerinde seviyeler ölçüldü (60 tur):

| Seviye | Çıktı | Oran | CPU/istek |
|---|---|---|---|
| 1 (BestSpeed) | 18 KB | %23,8 | 0,37 ms |
| 3 | 17 KB | %22,7 | 0,43 ms |
| **4 (yeni varsayılan)** | **17 KB** | **%22,2** | **0,45 ms** |
| 6 (eski varsayılan) | 16 KB | %21,2 | 0,63 ms |
| 9 | 15 KB | %20,6 | 1,16 ms |

**Uçtan uca A/B** — aynı ikili, yalnız `GZIP_LEVEL` farklı, 8 eşzamanlı,
iki turda sıra değiştirilerek:

| | CPU/istek | rps | gzip gövde |
|---|---|---|---|
| seviye 6 | 4,67 ms | ~665 | 24.115 B |
| **seviye 4** | **4,30 ms** | **~689** | 25.312 B |
| fark | **−%7,9** | +%3,6 | +%5 |

Seviye 4 seçildi: **CPU'da %8 kazanç**, karşılığında gövdede ~1,2 KB.
Cloudflare önbelleği (`s-maxage=300`) ve kenar sıkıştırması düşünülünce bu
1,2 KB önemsiz; buna karşılık haber patlamalarında önbellek ıskalayan her
istek bu CPU'yu ödüyor.

**Uygulama notu — havuz tuzağı:** `gzip.Writer.Reset()` seviyeyi
*değiştirmez*; seviye writer oluşturulurken sabitlenir. Tek `sync.Pool`
kullanılsaydı seviye değiştiğinde havuzdan eski seviyeli writer alınır ve
istenen seviye sessizce yok sayılırdı. Bunu yazdığım test yakaladı
(`TestGzipSeviyeHavuzuKirletmez`: seviye 9 ile 1 aynı boyutu üretiyordu).
Çözüm: **seviye başına ayrı havuz** (1–9).

---

### 2.9 Tam sayfa önbelleği — render + CSP + gzip birlikte atlanıyor

Bir önceki bölümde kalan CPU'nun dağılımı şuydu: gzip %23, GC %10, sistem
çağrıları %10, şablon render + JSON ~%15. Yani **hiçbiri tek başına baskın
değildi**; hepsi aynı işin parçasıydı: aynı sayfayı sıfırdan üretmek.

Çözüm: herkese açık CD sayfalarını **10 saniye** süreç içi önbelleğe almak.
Zincirin **en dışına** yerleştirildi (gzip'in de üstüne), böylece isabet eden
istek render, şablon, CSP hash üretimi **ve** sıkıştırmayı birlikte atlıyor.

**Temiz A/B** (aynı HEAD, tek fark bu katman; ayrı portlar, 8 eşzamanlı,
iki tur):

| | CPU/istek | rps | p50 | p95 |
|---|---|---|---|---|
| önbelleksiz | 4,47 ms | ~658 | 11,7 ms | 15,0 ms |
| **önbellekli** | **0,157 ms** | **~8.040** | **0,8 ms** | **1,8 ms** |
| fark | **28× az** | **12× fazla** | **14× düşük** | 8× düşük |

Canlıda (20 eşzamanlı): `/` **11.536 rps / p50 1,4 ms**, `/markets`
**15.098 rps / p50 1,0 ms**, hata 0. Boşta RSS 33 MB, yük sonrası 42 MB
(tepe 57 MB) — 400 girişlik sınır bellek büyümesini kapatıyor.

**Doğruluk — en kritik kısım.** Önbellek isabeti ile taze render **bayt bayt
aynı** (aynı ikili; "taze" istek `Cache-Control: no-cache` ile önbelleği
atlıyor):

| Sayfa | Sonuç |
|---|---|
| `/`, `/markets`, `/price`, `/videos`, `/canli-yayin`, `/business`, `/tag/bitcoin` | **AYNI** |

**Kişiselleştirme — neden yalnız anonim.** Gövdeye oturum durumu giriyor
(`LoggedIn`, kullanıcı adı, okunmamış bildirim rozeti). Bu yüzden:

- `tb_session` çerezi olan istek **ne önbelleğe alınır ne önbellekten okunur**.
- `Authorization` başlığı taşıyan istek aynı şekilde atlanır.
- `Cache-Control: no-cache` gönderen istemci önbelleği atlar (panel/tazeleme).
- Yanıt `Set-Cookie` veya `Location` içeriyorsa **saklanmaz** (önbelleklenmiş
  oturum çerezi herkese dağılırdı).
- 200 dışı yanıtlar saklanmaz.

**Başlıklar beyaz liste ile kopyalanır.** `X-Request-Id` önbellekten gelmez —
gelseydi tüm istekler aynı kimliği taşır ve log/tracing bozulurdu; isabet
anında **yeni** kimlik üretilir. `Set-Cookie` listede yok.

**Gözlemlenebilirlik.** İsabette ara katman (`withRequestID`) atlandığı için
log satırı yazılmıyordu; trafik panolarda "kaybolmuş" görünürdü. Artık isabet
kendi log satırını `"cache":"hit"` alanıyla yazıyor — toplam istek sayısı
eskisiyle kıyaslanabilir kalıyor.

**Gzip/ham varyantı ayrı saklanır**: gzip kabul etmeyen istemciye
sıkıştırılmış bayt gitmez (anahtar kodlama varyantını içerir).

**Kapsam dışı (bilinçli):** `/admin`, `/api`, `/media`, `/kg`, `/canli`
(SSE), oturum yolları (`/giris`, `/hesabim`…), `/arama`, sitemap/robots.
Bunlar kişisel ya da zaten ucuz.

### 2.10 Önbelleğin açtığı güvenlik açığı ve kapanışı

Bu katman yazılırken **kendi elimle bir açık ürettim** ve ölçüm yakaladı.
Ara katman sırası şöyleydi:

```
sayfaÖnbellek → gzip → recover → requestID → güvenlikBaşlıkları
              → sameOrigin → IPBloğu → site → HTMLCSP → mux
```

Önbellek en dışta olduğu için **isabet eden istek `withIPBlock`'u hiç
çalıştırmıyordu**. Bot tuzağına yakalanıp bloklanmış bir IP, önbellekte
duran sayfayı **200** ile alabiliyordu — yani blok koruması delinmişti.

**Kanıt (canlı, aynı senaryo, iki ikili):**

| İkili | temiz IP | bloklu IP |
|---|---|---|
| kontrolsüz | 200 | **200** ← açık |
| fixli | 200 | **403** |

**Düzeltme:** kontrol isabet yolunda **tekrar** ediliyor
(`isIPBlocked` 30 sn bellek önbelleği kullanır: mutex + map, DB'ye gitmez).
Önbelleği `withIPBlock`'un içine almak da bir seçenekti ama o zaman
saklanan (önceden sıkıştırılmış) gövde ikinci kez gzip'ten geçerdi — bozuk
yanıt. Bu yüzden kontrol yerinde tekrarlandı.

`TestSayfaOnbellekBlokluIPIsabetteDeReddedilir` bunu kilitliyor.

**Diğer ara katmanlar gözden geçirildi:**
- `withSameOrigin` yalnız durum değiştiren metotlara bakar; önbellek yalnız GET.
- `withSecurityHeaders` başlıkları beyaz listeyle replay ediliyor (CSP dahil).
- `withHTMLCSP` hash'li politikası saklanan başlıktan aynen gidiyor.
- `withRequestID` isabet yolunda atlanıyor → isabet kendi log satırını yazıyor.
- Bot tuzağı yolları (`/wp-login.php` vb.) önbellek kapsamında değil.

### 2.11 Lighthouse bütçesi (değişiklik sonrası)

`make performans` (gerçek Chrome, canlı 8090):

| | Skor | Eşik |
|---|---|---|
| performance | **100** | 90 |
| seo | **100** | 95 |
| accessibility | 97 | 90 |
| best-practices | **100** | 90 |

LCP 0,7 s · CLS 0,005 · TBT 0 ms — sıkıştırma seviyesi değişikliği ve
önbellek görünür hiçbir metriği bozmadı.

### 2.12 Bayat sun (stale-while-revalidate) — TTL anındaki sıçrama

**Bulunan sorun.** İlk sürümde TTL dolduğu an kayıt **tamamen** geçersiz
sayılıyordu: gelen istek senkron render ediyor, TTL anına birden çok istek
denk gelirse hepsi sıraya giriyordu. Tek-uçuş (singleflight) bunu "tek
üretim"e indiriyordu ama **o tek istek yine ~9 ms bekliyordu** ve 50
eşzamanlıda kuyruk p-max 190 ms'ye çıkıyordu.

**Değişiklik.** TTL geçmiş gövde **hemen** sunulur (bayat), yenisini tek bir
arka plan goroutine'i üretir. Bayat pencere 30 sn; HTTP `max-age=60`'ın
altında olduğu için tarayıcı/kenar zaten daha eskisini sunmuyor. Arka plan
işi `context.WithoutCancel` + 20 sn bütçe ile çalışır: istemci gitse de
önbellek tazelenir.

**A/B (TTL=1s, 50 eşzamanlı, 8 sn, aynı makine, sıra iki yönde de denendi):**

| | rps | p50 | p99 | **max** | **render** |
|---|---|---|---|---|---|
| önce | 13 925 | 2,8 ms | 13,6 ms | **190,0 ms** | **50** |
| sonra | 14 974 | 2,7 ms | 12,7 ms | **49,2 ms** | **10** |

Render sayısı loglardan sayıldı (her isabet `"cache":"hit"` yazar, kalan her
satır bir üretim). TTL=1s ile 8 sn'de ~8 süre sonu var; önce her süre sonu
bir senkron üretim + kuyruk, sonra yalnız arka plan: **50 → 10 render**,
p-max **190 → 49 ms**. Sıra ters çevrilince de aynı yön (max 114 ms vs
37 ms), yani ölçüm sıralamaya duyarlı değil.

Düşük eşzamanlılıkta (c=4) fark küçük ve beklenen yönde: p99 1,4 → 1,2 ms,
max 30 → 25 ms — bu katman zaten kullanıcı gecikmesini değil **sıçramayı**
hedefliyor.

**Canlı doğrulama (varsayılan 10 sn taze / 30 sn bayat).** TTL dolduktan
sonraki ilk istek **0,68 ms** (önbellek isabeti mertebesinde); eskiden bu
istek soğuk yoldu (~9 ms). Başlangıç logu yapılandırmayı yazıyor:
`"msg":"sayfa önbelleği","taze":"10s","bayat":"30s"`.

**Yapılandırma.** `SAYFA_ONBELLEK_TTL` / `SAYFA_ONBELLEK_BAYAT`; boşsa
10 sn / 30 sn. Bayat pencere verilmezse `3×TTL`. Geçersiz (≤0) değer yok
sayılır — yanlış yapılandırma önbelleği kapatmasın.

**Testler.** `TestSayfaOnbellekBayatHemenSunulurArkaPlandaYenilenir` (bayat
gövde anında + sonra yeni sürüm gelir), `TestSayfaOnbellekSingleflightTekUretim`
(20 eşzamanlı = 1 üretim, hepsi aynı gövdeyi alır),
`TestSayfaOnbellekArkaPlanBagimsizCalisir` (istek iptal edilse bile yenileme
tamamlanır).

### 2.13 Soğuk yolun (ıska) güncel taban çizgisi

Önbellek çalıştığı sürece soğuk yol yalnız ilk istekte ve her TTL sonunda
(bayat pencere kapattığı için artık kullanıcıya yansımıyor), ama DB
kesintisi/yeniden başlatma sonrası gerçek. Go yük üreteci, `-nocache`
(`Cache-Control: no-cache` → katman atlanır), 6 sn:

| eşzamanlılık | rps | p50 | p95 | p99 |
|---|---|---|---|---|
| c=1 | 111 | **7,8 ms** | 9,2 ms | 10,1 ms |
| c=4 | 410 | 9,4 ms | 12,5 ms | 15,4 ms |
| c=12 | 743 | 15,8 ms | 21,3 ms | 25,0 ms |

Sıcak yol aynı üreteçle: c=1 p50 **0,6 ms**, c=50 p50 2,9 ms / p99 13,2 ms.

Soğuk yol CPU profili (c=8, 10 sn, 29 s örnek): `Syscall6` %11,8,
`flate.(*fastEncL4).encode` %16,9 (gzip), `futex` %6,7, `memmove` %4,4.
Yani maliyetin çoğu **DB gidiş-dönüşü + gzip + kopyalama**; aritmetik değil.
Sıradaki aday ana sayfadaki ~20 sorguyu tek `pgx.Batch`'e indirmek.

### 2.14 Makale sayfaları da önbelleğe alındı — trafiğin çoğu burada

**Bulunan boşluk.** Önbellek kapsamı yalnız ana sayfa, bölüm, etiket ve
sabit sayfalardı. Bir haber sitesinde **trafiğin çoğu makale sayfasına**
gelir ve bunlar her istekte yeniden üretiliyordu: ölçüm **20–100 ms**
(tek istek 20 ms; yük altında 100 ms'ye çıkıyor).

**A/B (aynı makale, iki ikili, 6 sn, sıra iki yönde de):**

| | rps (c=8) | p50 (c=8) | p50 (c=1) |
|---|---|---|---|
| önce | 63 – 278 | 27 – 96 ms | 85 ms |
| sonra | **9 148** | **0,7 ms** | **0,4 ms** |

"Önce"nin aralık olması ölçüm gürültüsü değil, **bulgunun kendisi**: her
görüntülemede iki UPSERT (aşağıda) yapıldığı için sayfa, DB yüküne göre
27 ile 96 ms arasında salınıyor. "Sonra" iki koşuda da 0,4–0,8 ms.

**Kapsam nasıl belirleniyor.** Yol deseni zorunlu:
`/{bölüm}/{YYYY}/{MM}/{DD}/{slug}` veya `/{YYYY}/{MM}/{DD}/{slug}`. Son
segment denetlenmez (slug serbest metin); kapsamı daraltan tarih üçlüsü.
Ayrıca **hariç kökler** listesi var (`/api/`, `/admin`, `/media/`, `/kg/`,
`/canli/`, `/giris`, `/hesabim`, `/static/`, `/internal/`, `/webhooks/`,
`/amp/`, `/sitemap`, …): tarih desenine uysalar bile önbelleğe girmezler.
Bu ikinci liste bir test hatasıyla ortaya çıktı — `/api/2026/11/05/x`
desene uyuyordu; desen testi ile kapsam testi ayrıldı.

**Gövde değişmezliği kanıtlandı.** Önbellekten gelen gövde ile
`Cache-Control: no-cache` ile taze üretilen gövde **bayt bayt aynı**
(sha256; `/`, `/markets`, makale, `/tag/bitcoin`, `/price`). Yani makale
şablonunda istek başına rastgelelik (CSRF/jeton) yok; olsaydı önbellek
yanlış olurdu.

### 2.15 Okunma sayacı istek yolundan çıkarıldı (2 UPSERT → 2 sn'de bir tur)

**Sorun.** `repo.IncrementHit` her makale görüntülemesinde **iki ayrı,
sıralı UPSERT** atıyordu (`counters_hit` + `hit_daily`). Ölçüm: yalnız bu
iki yazım kaldırıldığında makale sayfası c=8'de **24 → 80 rps** (3,3×).
Sayaç bir iş metriği; istek yolunun parçası olmamalı.

**Değişiklik.** Okunmalar bellekte içerik başına biriktirilir, ~2 sn'de bir
**tek turda** yazılır (`unnest` ile dizi parametreleri → N UPSERT yerine 2
sorgu). Toplamlar kaybolmaz, yalnız gecikir (panel grafikleri dakika/gün
çözünürlüğünde). Kapanışta kalan sayılar da yazılır.

**Önbellekle çakışma — asıl tuzak.** Sayfa önbellekten gelince handler hiç
çalışmaz; sayaç handler'da olsaydı popülerlik verisi önbellek açıkken
**neredeyse dururdu**. Çözüm: handler içerik kimliğini istek bağlamına
işaretler, sayımı önbellek katmanı yapar — hem ıskada hem isabette, tek
yerden. Kimlik önbellek kaydında saklanır (`okunmaID`), böylece isabette
DB'ye sorgu gitmez. Arka plan yenilemesi **sayılmaz** (kullanıcı okuması
değil); bu, sahte okunma üretimini engeller.

**Kanıt (canlı).** Önbellek isabeti 0,4 ms iken 30 istek → `counters_hit`
**+30**, `hit_daily` **+30** (tam). Oturumlu istek önbelleği atlar ve
doğrudan sayılır (2 istek → +2).

**Testler.** `TestMakaleYoluMu` (desen + hariç kökler),
`TestSayfaOnbellekOkunmaIsabetteDeSayilir` (5 istek / 1 render / 5 okunma),
`TestSayfaOnbellekArkaPlanYenilemeSayilmaz` (3 istek → 3 okunma, yenileme
sayılmaz), `TestOkunmaBiriktiriciToplam`.

### 2.16 Sorgu sayısının gerçek dağılımı — ve iki N+1

**Önce ölçüm yöntemi yanlıştı.** Sorgu sayısını `log_statement='all'` +
`docker logs` ile saydım ve **89 / 62** gibi sayılar buldum. Bunlar
yanlıştı: çok satırlı sorgular log satırı olarak bölünüyor ve arka plan
işleri aynı pencereye düşüyor. Doğru cevap için `pgx.QueryTracer` ile her
sorguyu **çağıran fonksiyon:satır** ile yakalayan geçici bir araç yazdım
(`cmd/atrib`, commit'e girmedi).

**Doğru sayılar (soğuk istek, süreç içi önbellekler boş):**

| | sorgu | süre |
|---|---|---|
| ana sayfa | **28** | ~109 ms |
| makale | **38** | ~95 ms |

**N+1 #1 — ilgili haberler.** `repo.RelatedContents` her öneri için
`GetContent` çağırıyordu; `GetContent` ayrıca **ikinci** bir
`min_tier_weight` sorgusu atıyor. Yani öneri başına 2 gidiş-dönüş:
6 ilgili haber = **12 sorgu**, makale toplamının %32'si. Düzeltme:
`repo.ContentsByIDs` (tek sorgu + tek toplu `min_tier_weight`).
Sıra korunur — puanlama sırası `LinkSuggestions`'tan geliyor, map sırası
kullanılsaydı "en ilgili" haber rastgele olurdu.

**N+1 #2 — YouTube listesi.** Ana sayfa **aynı** listeyi 6 kez çekiyordu
(video · short · stream; modüller + "Explore" paneli ayrı ayrı). Tür
başına 10 sn önbellek (`yt_liste.go`), tavan 12 satır çekilip dilimlenir →
farklı `LIMIT` değerleri ayrı önbellek girdisi üretmez.

**Ayrıca** `kategoriID` (slug → id) 10 sn önbelleğe alındı: aynı 4 slug her
render'da çevriliyordu.

**Sonuç (aynı araç, 5 soğuk koşum):**

| | sorgu | gecikme |
|---|---|---|
| ana sayfa | 28 → **25** | ~109 → ~104 ms |
| makale | 38 → **28** | ~95 → ~93 ms |

**Dürüst yorum: sorgu sayısı %26 azaldı, gecikme yalnızca ~2 ms düzeldi.**
Bu sorgular aynı bağlantıda sırayla giden hafif okumalar (~0,2 ms/sorgu);
asıl maliyet gidiş-dönüş *sayısı* değil, sorguların kendisi (400 satırlık
alaka havuzu, gzip, şablon). Bu yüzden bu değişiklik "büyük hız kazancı"
değil, **DB yükü ve karmaşıklık azaltma** olarak değerlendirilmeli.
Toplu `pgx.Batch` fikrinin beklenen getirisi de bu ölçümle küçüldü: aynı
bağlantıda 5 sorguyu birleştirmek ~1 ms kazandırır.

**Doğruluk kanıtı.** Eski ve yeni ikilinin çıktısı 5 sayfada **bayt bayt
aynı** (`/`, `/markets`, makale, `/tag/bitcoin`, `/price`). N+1 kaldırmak
çıktıyı değiştirmedi.

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
| `TestCSPHashCacheAyniGovdeAyniBaslik` | CSP önbelleği başlığı birebir korur |
| `TestCSPHashCacheKapasiteTahliye` | önbellek süreç ömründe büyümez |
| `TestCSPHashCacheUretimMiddlewareIsabet` | gerçek zincirde 2. istek aynı başlık |
| `TestGzipSeviyeAyaruUygulanir` | seviye uygulanır, aralık dışı yok sayılır |
| `TestGzipSeviyeHavuzuKirletmez` | seviye değişince havuz yanlış writer vermez |
| `TestSayfaOnbellekIsabetRenderTekrarlanmaz` | isabette handler bir kez çalışır |
| `TestSayfaOnbellekOturumluIstekKisisel` | oturumlu istek önbelleği atlar |
| `TestSayfaOnbellekIstekKimligiTekrarlanmaz` | isabetler aynı kimliği taşımaz |
| `TestSayfaOnbellekCerezYazanYanitSaklanmaz` | `Set-Cookie`'li yanıt saklanmaz |
| `TestSayfaOnbellekHataSaklanmaz` | 200 dışı yanıt saklanmaz |
| `TestSayfaOnbellekYolVeMetotFiltresi` | yalnız herkese açık CD yolları + GET |
| `TestSayfaOnbellekGzipVeHamAyriSaklanir` | gzip kabul etmeyene ham gövde |
| `TestSayfaOnbellekCSPKorunur` | replay'de CSP başlığı aynı |
| `TestSayfaOnbellekBlokluIPIsabetteDeReddedilir` | **bloklu IP önbellekten 200 almaz** |
| `TestSayfaOnbellekBayatHemenSunulurArkaPlandaYenilenir` | bayat gövde anında, yenisi arka planda |
| `TestSayfaOnbellekSingleflightTekUretim` | 20 eşzamanlı = 1 üretim |
| `TestSayfaOnbellekArkaPlanBagimsizCalisir` | iptal edilen istek yenilemeyi durdurmaz |
| `TestMakaleYoluMu` | makale deseni + **hariç kökler** önbelleğe girmez |
| `TestSayfaOnbellekMakaleYoluUygun` | makale önbelleklenir, oturumlu istek yine kişisel |
| `TestSayfaOnbellekOkunmaIsabetteDeSayilir` | isabet sayaçı durdurmaz |
| `TestSayfaOnbellekArkaPlanYenilemeSayilmaz` | yenileme sahte okunma üretmez |
| `TestOkunmaBiriktiriciToplam` | biriktirici toplar, boşaltınca sıfırlar |
| `TestYtListeDilim` | YouTube listesi 3/5/6/12/20 isteklerinde doğru dilimlenir |
| `TestSayfaOnbellekIleUyum` | N+1 düzeltmeleri önbellek kapsamını değiştirmedi |
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

1. **Önbellek ıskası (soğuk istek).** Isabet 0,16 ms (makale: 0,4 ms);
   ıska c=1 p50 7,8 ms ana sayfa, 20–100 ms makale. Bayat sun ve makale
   önbelleği ile ıska artık kullanıcıya yansımıyor (bkz. §2.12, §2.14).
   Sorgu sayısı §2.16'da 28/25'e indi ama **gecikme kazancı ~2 ms**;
   toplu `pgx.Batch` artık düşük getirili.
2. **TTL ve tazelik ayarı — yapıldı** (`SAYFA_ONBELLEK_TTL` /
   `SAYFA_ONBELLEK_BAYAT`, §2.12). Varsayılan 10 sn taze + 30 sn bayat.
3. **`/media/thumb` ve `/media/variant`**: ilk istekte üretiliyor, sonra
   diskten. Medya tablosu şu an boş; içerik dolduğunda `image-svc`
   kazancı (1,8×) doğrudan kullanıcıya yansıyacak.
4. **`platform/render.go`** (583 satır, JSON tasarım → JPEG): ölçüldü
   **10,1 ms/raster**, ama **sıcak yolda değil** (kapak/manşet üretimi, panel
   ve arka plan işleri). Rust'a taşımak için uygun bir öğrenme parçası;
   performans gerekçesi zayıf. Karar: ihtiyaç doğunca.

---

## 6. Yeniden üretme

```bash
# YÜK ÜRETECİ: Go (Python thread'leri 1.500 rps üstünde kendisi darboğaz olur)
/tmp/loadgen -u http://127.0.0.1:8090/ -c 50 -d 10

# istek başına sunucu CPU'su (asıl metrik)
python3 /mnt/c/Users/alper/AppData/Local/Temp/opencode/gz_cpu.py

# gzip seviyesi A/B: aynı ikili, yalnız GZIP_LEVEL farklı
GZIP_LEVEL=6 ./bin/site &   # 8097
GZIP_LEVEL=4 ./bin/site &   # 8098

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

# bayat sun A/B: aynı commit, iki ikili, TTL=1s; render sayısı loglardan
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/ab_swr_run.sh
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/render_count.sh

# soğuk (ıska) / sıcak yol taban çizgisi — Go üreteç
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/cold_base.sh

# contents tablosu tarama sayısı (istek başına DB yükü)
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/scans.sh

# profil
curl -sS -o /tmp/cpu.pprof "http://127.0.0.1:6060/debug/pprof/profile?seconds=8"
go tool pprof -top bin/site /tmp/cpu.pprof
```
