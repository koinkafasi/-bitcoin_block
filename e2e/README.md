# Testler

Üç katman, hepsi ücretsiz araçlarla:

| Katman | Araç | Ne zaman | Ne test eder |
|---|---|---|---|
| **Unit** | `go test` + testify | her `make check` | Saf mantık, ayrıştırma, önbellek/ayar katmanı (mock'lu) |
| **Integration** | `go test -tags=integration` | CI `go` job'ında | Gerçek Postgres + tam HTTP yığını, yetkilendirme, responsive CSS regresyonu |
| **E2E** | Puppeteer + `node:test` | CI `e2e` job'ında | Gerçek tarayıcıda taşma, manşet, fikstür, admin koruması |

## Hızlı başlangıç

```bash
make test        # birim testleri
make test-race   # + yarış tespiti
make check       # vet + test + build
make verify      # tüm Go katmanları
```

## Mock'lar

Dış bağımlılıklar mock'lanır, gerçek servis çağrısı yapılmaz:

- **Veritabanı** — `internal/repo/store.go` içindeki `DB` arayüzü. `*pgxpool.Pool`
  bu arayüzü sağlar; testlerde `fakeDB` (`internal/repo/dbmock_test.go`) kullanılır.
  Çağrı noktaları değişmemiştir.
- **HTTP** — ESPN/TheSportsDB uçları paket düzeyinde `var`; testlerde
  `httptest` sunucusuna yönlendirilir (`internal/repo/services_http_test.go`).
  Timeout, HTTP 4xx/5xx, bozuk JSON, rate limit senaryoları kapsanır.
- **Tarayıcı dış ağı** — E2E'de çapraz kaynak istekleri engellenir
  (`lib/harness.js`), böylece testler deterministik ve hızlıdır.

## Integration testleri

```bash
createdb theblock_test
TEST_DATABASE_URL=postgres://... make test-int
```

`TEST_DATABASE_URL` tanımlı değilse testler **skip** edilir; üretim
veritabanına dokunulmaz.

## E2E testleri

> **Uyarı — port 8070 canlı siteyi besler.** Bu makinede
> `theblock-site.service` 8070'te çalışır ve `cloudflared-theblock.service`
> onu theblock.com.tr'ye tünneller. E2E dosyalarının büyük bölümü admin
> paneline POST eder, yani 8070'e koşan bir paket **üretim veritabanına**
> içerik yazıp silerdi. `make e2e` bu durumda kendini durdurur
> (`deploy/scripts/e2e-uretim-korumasi.sh`).

İzole koşum — önerilen yol:

```bash
make e2e-install   # puppeteer + chrome (bir kez)
make e2e-izole     # ayrı veritabanı + ayrı port + ayrı ikili
```

`make e2e-izole` CI'ın `e2e` işini yerelde birebir kurar ve üretimden dört
noktada ayrışır:

1. **Ayrı veritabanı** — `theblock_e2e` sıfırdan yaratılır, migration'lar
   uygulanır, `deploy/e2e_seed.sql` yüklenir.
2. **Ayrı ikili** — `/tmp/theblock-e2e/site`. `go/bin/site`'ın üzerine
   *yazılmaz*; o dosya çalışan servisin ikilisidir ve `make build` onu
   değiştirir.
3. **Ayrı port ve depolama** — 8099 ve `/tmp/theblock-e2e/storage`.
4. **`deploy/.env` yüklenmez** — `env -i` ile yalnız CI'ın verdiği dört
   değişken geçer (`DATABASE_URL`, `PORT`, `APP_ENV`, `STORAGE_DIR`), böylece
   üretim Redis'i ve gerçek ASR/AI anahtarları devrede olmaz.

Tek dosya koşmak için:

```bash
bash deploy/scripts/e2e-izole.sh tests/smoke.test.js
```

Ayarlanabilir: `E2E_IZOLE_PORT`, `E2E_IZOLE_DB`, `E2E_IZOLE_DIR`.
Betik 8070'i port olarak kabul etmez.

Kendi test örneğiniz zaten ayaktaysa doğrudan hedef verebilirsiniz:

```bash
make e2e E2E_BASE_URL=http://127.0.0.1:8099
```

## Responsive regresyon koruması

Geçmişte mobilde gerçek bir hata vardı: grid sütunları çıplak `1fr`
(`minmax(auto,1fr)`) olduğu için içerik 390px ekranda 930px'e şişiyor,
`overflow-x:hidden` ile kırpılıyordu.

Bu iki yerden korunuyor:

1. `internal/httpapi/integration_test.go` → `TestPublicCSSHasNoBareFrTracks`
   sunulan `public.css` içinde çıplak `fr` grid sütunu kalmadığını doğrular.
2. `e2e/tests/responsive.test.js` → 320–1440px arası her sayfada yatay taşma,
   viewport dışına taşan görsel ve manşet/pager taşması kontrolü.

## CI

`.github/workflows/ci.yml`:

- `go` — vet, unit (`-race`), build, migration, integration
- `e2e` — Postgres servisi + site + Puppeteer
- `rust`, `smoke`, `guvenlik` — mevcut işler

Depo **özel** olduğu için Actions koşumları hesabın dakika kotasından
düşer; kota dolduğunda ya da ödeme/harcama limiti sorununda işler
*başlamadan* düşer ("The job was not started because recent account payments
have failed or your spending limit needs to be increased"). Bu durumda
doğrulamayı `make e2e-izole` ile yerelde yapın.

Tetikleyici: `push` yalnız `main` dalında, ayrıca her `pull_request` ve
`workflow_dispatch`. **Özellik dalını push etmek CI'ı çalıştırmaz** — PR
açmak (ya da `gh workflow run`) gerekir.
