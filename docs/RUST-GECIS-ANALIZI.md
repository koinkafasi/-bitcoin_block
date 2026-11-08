# Go → Rust geçiş analizi (ölçüme dayalı)

**Tarih:** 7 Kasım 2026 · **Kapsam:** `~/bitcosar` (Go monoliti + `rust/` mikroservisleri)
**Yöntem:** Kaynak taraması + canlı sistem ölçümü. Aşağıdaki her sayı bu makinede
alınmıştır; tahmin değildir.

---

## 0. Kısa cevap

**Tam geçiş yapma.** Talebi karşılayan sıcak yol (HTTP isteği) Rust'a taşınmıyor;
zaten darboğaz CPU değil. Buna karşılık **görsel işleme hattı ölçülmüş 2.4×
yavaş** ve Rust servis o hatta bağlı bile değil. Kazanç tam geçişte değil,
**dar ve ölçülebilir bir dilimde**.

---

## 1. Bugünkü tablo (ölçülen)

### Kod hacmi

| Kalem | Değer |
|---|---|
| Elle yazılmış Go | **96.454 satır** |
| templ üretilmiş | 43.257 satır (45 dosya) |
| Test | 38.232 satır / 293 dosya / **1.630 test fonksiyonu** |
| Paket | 22 iç paket + 12 komut |
| Doğrudan+dolaylı modül | 41 |
| `context.Context` kullanımı | 1.085 |
| SQL çağrısı (`Query/QueryRow/Exec`) | 866 |
| `json.Marshal`/`Unmarshal` | 351 |
| `map[string]any` | 436 |
| `go func` | 24 (errgroup yok — elle semafor) |

### Derleme / doğrulama

| İş | Süre |
|---|---|
| `go build ./cmd/site` (önbellek sıcak) | **4,6 sn** |
| `go vet ./...` | **0,96 sn** |
| `go test -short ./...` (1.630 test) | **14,8 sn** |
| Binary | 29 MB (LTO/strip yok) |

### Çalışma zamanı (canlı servis)

| Ölçüm | Değer |
|---|---|
| Boşta RSS | **34,6 MB** (tepe 35,4 MB) |
| 50 eşzamanlı istemci × 10 sn | 4.347 istek, **435 rps**, **0 hata** |
| Gecikme | p50 **96 ms** · p90 182 ms · p95 211 ms · p99 **270 ms** · max 382 ms |
| Yük altında tepe RSS | **44,4 MB** (GC sonrası 43,8 MB) |
| Tek istek (boşta) | `/` 19–28 ms · `/readyz` 0,8–1,4 ms |
| Sayfa bazında (yük altında) | `/` p50 **191 ms** · `/markets` p50 **87 ms** · `/price` 163 ms |
| Donanım | Ryzen 5 7500F, 12 vCPU, WSL'e 20 GB RAM |
| Önünde | Cloudflare + `Cache-Control: max-age=60, s-maxage=300` |

### Rust tarafı (hâlihazırda var)

| Servis | Satır | RSS | Bağlı mı? |
|---|---|---|---|
| `image-svc` (16:9 kırpma + ölçekleme + JPEG/PNG) | 448 | 2,3 MB | **HAYIR** — Go'da `IMAGE_SVC` referansı yok |
| `search-svc` (Tantivy, Türkçe indeks) | 444 | 12 MB | Evet (`SEARCH_SVC_URL`, opsiyonel) |

---

## 2. Kritik ölçüm: görsel hattı

Aynı iş: 1600×900 JPEG → 1200 px genişlik, 16:9 kırpma, JPEG q82.

| Uygulama | p50 |
|---|---|
| **Rust `image-svc`** (image crate / libjpeg-turbo) | **45 ms** |
| **Go** (`x/image/draw` CatmullRom + `image/jpeg`) | **109 ms** |

**2,4× fark.** Sebep dil değil, kütüphane: Go'nun `image/jpeg` kod çözücüsü/ kodlayıcısı
saf Go ve SIMD/libjpeg-turbo kullanmıyor. Rust tarafı C kütüphanesini çağırıyor.

Ama daha önemlisi: **bu servis çalışıyor ve Go onu hiç çağırmıyor.** Yani elinde
2,4× hızlandırıcı duruyor, kullanılmıyor.

---

## 3. Tam geçiş ne kazandırır? (dürüst hesap)

### Talep yolu (SSR + /api)

Sayfa süresinin bileşenleri kabaca:

```
istek → middleware/CSP → pgx sorguları (10-40 adet) → templ render → sıkıştırma → yanıt
        ~1 ms              ~%60-80 süre              ~%20        ~%5
```

**Sorgu süresi Rust ile değişmez.** `pgx` yerine `sqlx`/`tokio-postgres` yazsan
PostgreSQL'e giden aynı byte'lar gider. 191 ms'lik ana sayfanın baskın kısmı
veritabanı + N+1 sorgu sayısı, CPU değil. Bu yüzden:

> Talep yolunda tam geçişin beklenen kazancı **%5–15** aralığında; riski ise
> 96 bin satırın yeniden yazımı.

### Bellek

| | Go (bugün) | Rust (tahmini) |
|---|---|---|
| Boşta | 34,6 MB | 8–15 MB |
| Yük altında tepe | 44,4 MB | 12–20 MB |
| GC duraklaması | var (küçük, <1 ms) | yok |

Rust burada **kazanır ama kazanç anlamsız**: fark ~25 MB. WSL'de 20 GB RAM var,
13 GB'ı zaten başka şeyler (PostgreSQL, Docker, theblock) kullanıyor. **Senin
bellek tüketicin site değil.** Buna karşılık bu fark, 256 MB limitli bir
Kubernetes pod'una sıkıştırıldığında anlamlı olur — orada Rust'ın "GC yok"
avantajı gerçek.

### Maliyet

| Kalem | Tahmin |
|---|---|
| 96k satır + 43k templ + 1.630 testin yeniden yazımı | **6–12 ay** (tek kişi, yarı zamanlı: 1,5–2 yıl) |
| Özellik dondurma süresi | aynı süre |
| Regresyon riski | yüksek — 1.630 test = 1.630 gizli davranış sözleşmesi |
| templ eşdeğeri | yok. `askama`/`maud`/`minijinja` var ama templ'in tip güvenli component + HTMX modelini birebir vermiyor → 43k satırlık şablon katmanı yeniden tasarlanır |

**Sonuç:** Tam geçiş, ölçülebilir hiçbir metrikte mevcut durumu iyileştirmeden
6-12 ay boyunca ürün geliştirmeyi durdurur.

---

## 4. Rust'ın gerçekten kazandığı yerler (bu projede)

Aşağıdakiler **dil değil iş yükü** karakteri gereği Rust'a uygun. Hepsi
monolitten **ayrılabilir**, hepsinin sınırı net.

| # | İş | Bugün | Rust kazancı | Kanıt düzeyi |
|---|---|---|---|---|
| 1 | **Görsel hattı** (kırpma/ölçek/AVIF) | Go, 109 ms | **2,4×** | ölçüldü |
| 2 | **Manşet raster'ı** (`platform/render.go`, 583 satır) | Go, CPU + font | 2–3× (aynı libjpeg/libfreetype etkisi) | yüksek olasılık |
| 3 | **Metin işleme / AI ön-işleme** (tokenizasyon, kümeleme, benzerlik, RSS temizleme) | Go | 2–4× CPU, belirgin düşük bellek | orta |
| 4 | **Arama** (Tantivy) | Rust, zaten var | ✓ | kanıtlandı |
| 5 | **Video/transcode, thumbnail, ses normalizasyonu** | yok | Rust (ffmpeg bindings) doğal seçim | — |
| 6 | **Kalıcı arka plan işçileri** (uzun süreli, bellek sabit kalmalı) | Go | kazanç düşük, sadece pod limiti varsa | — |

Dikkat: **2 numara hariç hepsi zaten ayrı süreç olabilir.** Yani "Rust'a geçmek"
= "monoliti yeniden yazmak" değil, **"süreç sınırı çizip o sınırın arkasını Rust
yazmak"**.

---

## 5. Buna karşılık Go'da duran, Rust'tan büyük kazançlar

Aşağıdakiler **hiçbir dil geçişi gerektirmiyor** ve muhtemelen toplamda Rust'ın
vereceğinden fazlasını verir:

| # | Eksik | Etki |
|---|---|---|
| 1 | **pprof yok** (`/debug/pprof` → 404) | Körsün. Ana sayfanın 191 ms'ini nereye harcadığını bilmiyorsun. Profil olmadan hiçbir optimizasyon iddiası doğrulanamaz. **İlk iş bu.** |
| 2 | **GOMEMLIMIT / GOGC ayarsız** | GC hedefi bilinçli değil. `GOMEMLIMIT=256MiB` + `GOGC=off` denemesi tek satır. |
| 3 | **`DB_MAX_CONNS` boş** | `pool.go` yazılmış ama `app.env`'de tanımlı değil → pgx varsayılanı `max(4, NumCPU)` = 12. 20+ arka plan işçisi varken havuz dar. |
| 4 | **Ana sayfada N+1** | `homeView` + `buildBlocks`: highlights, trends, topread, galeri×2, hava, kurlar, şehirler, etkinlik, kategoriler… 10-40 ardışık sorgu. Tek `pgx.Batch` veya tek CTE ile %40-60 kesilebilir. **Rust'ın vereceğinden fazla.** |
| 5 | **`image-svc` bağlı değil** | Ölçülmüş 2,4× hazır bekliyor. Bağlamak ~1 gün. |
| 6 | **`errgroup` yok, elle semafor** | `go func` + `make(chan)` + 27 mutex. `errgroup` + `context` ile hata yayılımı ve iptal netleşir. |
| 7 | **`map[string]any` 436 yerde** | Her biri tahsis + tip kaybı. Şablon verisi tipli struct'a geçtikçe GC baskısı ve `json.Marshal` maliyeti düşer. |
| 8 | **`s-maxage=300` + Cloudflare** | Anonim trafiğin çoğu Go'ya hiç ulaşmıyor. 435 rps'lik ölçüm senaryosu gerçek trafikten çok daha ağır. |

---

## 6. Önerilen yol haritası

### Faz A — Ölçüm ve Go içi kazanç (1–2 hafta, dil değişikliği yok)

1. `net/http/pprof` + `runtime/metrics` aç, CPU ve heap profili al.
2. Ana sayfa sorgularını `pgx.Batch`'e indir; önce/sonra p50'yi yaz.
3. `GOMEMLIMIT` + `DB_MAX_CONNS` + `DB_MIN_CONNS` ayarla, ölç.
4. `image-svc`'yi Go'ya bağla: `platform/render.go` ve `headline_handlers.go`
   üzerinden POST → ölçülen 2,4× kazanç devreye girsin. Servis çökerse
   yerel Go yoluna düş (fallback).
5. Hedef: `/` p50 **191 ms → 90 ms altı**, RSS **44 MB → sabit 40 MB**.

### Faz B — Rust yüzeyini büyüt (1–2 ay, süreç sınırı korunarak)

6. `image-svc`'ye AVIF/WebP çıkışı ekle (bugün TODO). Cloudflare'a AVIF
   geçince bant genişliği düşer, LCP iyileşir.
7. `render-svc` (Rust): `platform/render.go`'nun (583 satır, JSON tasarım
   belgesi → JPEG) birebir portu. Sözleşme net, testleri var, ayrı süreç.
   **Rust öğrenmek için ideal ilk gerçek iş.**
8. `text-svc` (Rust): AI hattının ön-işlemesi — RSS temizleme, kümeleme,
   benzerlik, anahtar kelime. `internal/ceviri`, `internal/classify`,
   `internal/quality` içindeki saf fonksiyonlar buraya taşınabilir.

### Faz C — Yalnız ölçüm gerektiriyorsa

9. Uzun süreli işçileri (transkripsiyon, AI worker) ayrı sürece çıkar;
   gerekirse Rust. Burada kazanç dil değil **izolasyon**: bir işçinin OOM'u
   siteyi düşürmesin.
10. Dağıtım hedefi Kubernetes/Talos olursa (256–512 MB pod limiti) bellek
    disiplini için Rust yeniden gündeme gelir. **O zamana kadar değil.**

---

## 7. "Herkes Rust'a geçiyor" iddiası

Doğru değil. 2026 itibarıyla gerçek tablo:

- **Rust'ın kazandığı yer:** sistem katmanı, gömülü, tarayıcı motoru, veritabanı
  çekirdeği, görsel/ses/video kodek, ağ veri düzlemi, CLI araçları, kernel.
  Linux 6.1'den beri çekirdekte Rust var; Android, Windows, Cloudflare, AWS
  Firecracker, Discord (read states) — hepsi **dar, sıcak, CPU/ bellek kritik**
  modüller.
- **Go'nun koruduğu yer:** ağ servisleri, API arka uçları, orkestrasyon,
  DevOps altyapısı (Docker, Kubernetes, Terraform, Prometheus, etcd). Web
  backend'inde Go pazar payını **artırıyor**, azaltmıyor.
- **AI tarafında** gerçek geçiş Python → Rust değil; Python **orkestrasyon**
  katmanı olarak kalıyor, **inference/vektör/kodek** çekirdeği Rust/C++'a
  kayıyor (`candle`, `burn`, `ort`, `tokenizers`, `qdrant`). Yani senin
  `pgvector` + HTTP tabanlı AI hattın zaten bu modelin doğru tarafında.
- **Blog yazısı için malzeme tam burada:** "Go'dan Rust'a geçtik" değil,
  **"bir Go monolitinden Rust sidecar'a: nerede 2,4× kazandım, nerede hiçbir
  şey kazanmadım"** — bu ölçüm, trend yazılarından çok daha değerli ve
  doğrulanabilir. Elinde gerçek sayılar var.

---

## 8. Gelişim/portfolyo açısından

**Rust öğrenmeye değer mi?** Evet — ama gerekçesi performans değil:

| Gerekçe | Ağırlık |
|---|---|
| Sahiplik/borrow checker, veri yarışlarını derleme zamanında yakalamayı öğretir | **yüksek** — Go'daki 27 mutex + 24 `go func`'un hangi kısmının gerçekten güvenli olduğunu ancak bunu öğrenince görebilirsin |
| Elle yazılmış 96k satırlık Go kod tabanını "sıfırdan düşünme" fırsatı | yüksek |
| AI/makale konusu: ölçülmüş bir göç hikâyesi | yüksek |
| CV'de Rust | orta — "Go + Rust sidecar" profili, "Go'yu bıraktım" profilinden daha nadir ve daha değerli |
| Ürün hızı | **negatif** — Faz A boyunca hiçbir şey kazandırmaz |

**Doğru sıra:** Faz A (Go'da ölçüm + kazanç) → Faz B'de `render-svc` ile Rust
öğren → öğrendikçe Rust yüzeyini büyüt. Böylece Rust bilgin ürünü **yavaşlatmaz**,
ölçülmüş kazançla birlikte büyür.

---

## 9. Özet karar tablosu

| Soru | Cevap |
|---|---|
| Tam geçiş yapmalı mıyım? | **Hayır.** 6–12 ay, talep yolunda %5–15 kazanç, sıfır metrik iyileşmesi. |
| Bellek için geçmeli miyim? | **Hayır.** 34 MB → ~12 MB; WSL'de 20 GB var, site bellek tüketicisi değil. |
| Hiç geçmemeli miyim? | **Hayır.** Görsel/ses/video/metin ön-işleme hatlarında Rust ölçülmüş 2,4× kazandırıyor. |
| İlk yapılacak? | **pprof + ana sayfa N+1 + `image-svc`'yi bağla.** Üçü de Rust'sız, üçü de Rust'ın toplamından fazla kazanç. |
| Rust'ı nerede öğrenirim? | `render-svc` portu (`platform/render.go`, 583 satır, net sözleşme, testleri hazır). |
| Makale konusu? | "Nerede 2,4× kazandım, nerede kazanmadım" — ölçüm odaklı göç hikâyesi. |

---

### Ölçüm komutları (yeniden üretilebilir)

```bash
# yük testi
python3 /mnt/c/Users/alper/AppData/Local/Temp/opencode/load2.py

# görsel hattı karşılaştırması (Go tarafı)
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/imgbench.sh

# derleme / test süreleri
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/bench2.sh

# kod sayımı
bash /mnt/c/Users/alper/AppData/Local/Temp/opencode/go-stats.sh
```
