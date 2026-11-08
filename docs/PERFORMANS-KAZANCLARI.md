# Performans kazançları — ölçüm raporu

**Tarih:** 8 Kasım 2026 · **Makine:** Ryzen 5 7500F, 12 vCPU, WSL 20 GB
**Yöntem:** canlı servise yük testi + gerçek profil + PostgreSQL sorgu sayımı.
Aşağıdaki her sayı bu makinede ölçüldü.

İlgili analiz: [`RUST-GECIS-ANALIZI.md`](RUST-GECIS-ANALIZI.md)

---

## 1. Sonuç tablosu

Aynı yük testi (50 eşzamanlı istemci × 10 sn, 6 sayfa, aynı betik):

| Metrik | Önce | Sonra | Fark |
|---|---|---|---|
| Toplam istek | 4.347 | **18.099** | **4,2×** |
| Saniyedeki istek (rps) | 435 | **1.810** | **4,2×** |
| p50 gecikme | 96 ms | **25 ms** | **3,8× daha hızlı** |
| p90 | 182 ms | **41 ms** | 4,4× |
| p95 | 211 ms | **51 ms** | 4,1× |
| p99 | 270 ms | **59 ms** | 4,6× |
| max | 382 ms | **95 ms** | 4,0× |
| Hata | 0 | **0** | — |

> **Ölçüm varyansı:** aynı betikle tekrar koşulduğunda 1.630 rps / p50 28 ms
> ölçüldü (arka planda başka bir iş koşuyordu). Yani kazanç 4,2× değil
> **en az 3,7×** diye okunmalı; her koşuda bir önceki servisin GC/bağlantı
> durumu sonucu birkaç puan oynatıyor. Karşılaştırmalı sonuçlar (önce/sonra)
> aynı oturumda art arda alındı.

Sayfa bazında (yük altında, p50):

| Sayfa | Önce | Sonra |
|---|---|---|
| `/` (ana sayfa) | 191 ms | **51 ms** |
| `/price` | 163 ms | **21 ms** |
| `/markets` | 87 ms | **26 ms** |
| `/videos` | 95 ms | **33 ms** |
| `/canli-yayin` | 91 ms | **21 ms** |
| `/business` | 87 ms | **26 ms** |

Tek istek (boşta, `curl`): ana sayfa **26 ms → 12 ms**.

### Bellek

| | Önce | Sonra |
|---|---|---|
| Boşta RSS | 34,6 MB | **29–32 MB** |
| Yük altında tepe | 44,4 MB | 50,9 MB |

Tepe değer yükseldi çünkü **aynı 10 saniyede 4,2 kat fazla istek** işlendi
(uçuştaki istek sayısı arttı). İstek başına maliyet düştü; boşta RSS de azaldı.

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
Artık açık: `DB_MAX_CONNS=16`, `DB_MIN_CONNS=2`, yaşam süresi 30 dk,
boşta 5 dk (PostgreSQL'de `idle in transaction` birikmesin).

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
| `BenchmarkResizeToWidth*`, `BenchmarkCreateThumb*` | kazanç ölçülebilir kalır |

Tüm paket testleri geçiyor: `go test -short ./...` → 23 paket **ok**,
`gofmt -l .` boş, `go vet ./...` temiz.

---

## 4. Sırada ne var (ölçümle doğrulanmış adaylar)

1. **`/price` sayfası**: yük altında p50 21 ms'e indi ama istek başına 3,35
   transaction görülüyor — fiyat/endeks uçları için tek batch'e aday.
2. **`/media/thumb` ve `/media/variant`**: ilk istekte üretiliyor, sonra
   diskten. Medya tablosu şu an boş; içerik dolduğunda `image-svc`
   kazancı (1,8×) doğrudan kullanıcıya yansıyacak.
3. **`platform/render.go` (583 satır, JSON tasarım → JPEG)**: manşet raster'ı
   hâlâ Go'da. `image-svc` ile aynı kütüphane etkisi bekleniyor; ayrı süreç
   olarak Rust'a taşınmaya en uygun parça (bkz. analiz §6, Faz B).
4. **`GOMEMLIMIT`**: bellek artık ölçülüyor; 256 MB hedefiyle GC baskısı
   ayarlanabilir.

---

## 5. Yeniden üretme

```bash
# yük testi (50 eşzamanlı × 10 sn)
python3 /mnt/c/Users/alper/AppData/Local/Temp/opencode/load2.py

# görsel hattı karşılaştırması
IMAGE_SVC_URL=http://127.0.0.1:8172 \
  go test -run XXX -bench ResizeToWidth -benchtime 10x ./internal/platform

# istek başına sorgu sayımı
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/perquery2.sh

# profil
curl -sS -o /tmp/cpu.pprof "http://127.0.0.1:6060/debug/pprof/profile?seconds=8"
go tool pprof -top bin/site /tmp/cpu.pprof
```
