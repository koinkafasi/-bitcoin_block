# theblock — Eksikler Denetimi ve Yapılacaklar Spec'i

**Sürüm:** 1.0 — 28 Ekim 2026
**Kapsam:** Depo genelinde kanıta dayalı teknik denetim; `04-DoD.md` "73 kalem → 69 ✅" iddiasının
kodla/canlıyla doğrulanması; kalan işlerin fazlara bölünmesi.
**İlişkili:** `03-spec-plan.md` (dalga haritası) · `04-DoD.md` (kabul listesi) · `docs/ILERLEME.md` (Faz 1–124)

> **Yöntem:** Her bulgu ya bir komut çıktısıyla ya da üretim günlüğüyle kanıtlanmıştır. Kanıtsız
> hiçbir kalem bu listeye girmemiştir. Bulgular `E-n` kimliğiyle numaralanır; her birinin
> **kök nedeni**, **çözümü** ve **kabul kriteri** vardır — doğrudan faz girdisi olarak kullanılabilir.

---

## 1. Yönetici özeti

`04-DoD.md` özellik kapsamı için büyük ölçüde doğrudur: ürün gerçekten geniş ve kod disiplinli
(gofmt temiz, `go vet` temiz, 569 Go test fonksiyonu, 56 E2E dosyası, kodda TODO/FIXME borcu yok).
**Ancak DoD'nin "Global kapılar" tablosu (§1) bugün gerçeği yansıtmıyor:** 8 kapıdan 3'ü kırmızı.

Eksikler özellik eksikliği değil; **işletme (operasyon), doğrulama ve kapanmamış iş** eksikliği.

| Katman | Durum | Kanıt |
|---|---|---|
| Go statik analiz + derleme | ✅ temiz | `go vet ./...`, `go build` çıkış 0 |
| Go birim testleri | ✅ yeşil | `go test ./...` çıkış 0, 569 test fonksiyonu |
| Biçim (gofmt) | ✅ temiz | `gofmt -l` boş |
| Rust image-svc / search-svc | ✅ derleniyor, clippy temiz | `cargo check/clippy --locked` temiz |
| Go integration testleri | ❌ **yerelde çalışmıyor** | SQLSTATE 28P01 — `make test-int` kırılmış (E-11) |
| E2E | ⚠️ **170/192 geçti, 15 hata** | seed taksonomisi düzeltildi (E-12, E-23) |
| Canlı smoke (`deploy/smoke.sh`) | ❌ **2 hata** | `SONUÇ: 2 HATA` (E-4) |
| Otomatik yedekleme | ❌ **hiç çalışmamış** | yedek dizini yok, timer kurulu değil (E-1) |
| RSS/ajans botu | ❌ **çekimlerin %55'i hatalı** | 10.296 hata / 8.293 başarı (3 gün) (E-2) |
| Arka plan işleri (`scheduler`) | ⚠️ **0 test** | 1.224 satır, 8 dosya, test dosyası yok (E-10) |

**En kritik üç kalem:** (1) yedekleme hiç çalışmıyor → veri kaybı riski, (2) RSS botu yarı yarıya
çalışmıyor → içerik akışı aksıyor, (3) commit edilmemiş 57 dosyalık Faz 125 işi → kaybolma/CI körlüğü.

### Önem dereceleri
| Kod | Anlam |
|---|---|
| **S1** | Veri kaybı veya canlı servis riski — hemen |
| **S2** | İşlev bozuk / kullanıcıya veya editöre yansıyor |
| **S3** | Kalite, test, CI, bakım borcu |
| **S4** | Doküman / süreç tutarlılığı |

---

## 2. Denetim kanıt tablosu

| # | Komut | Sonuç |
|---|---|---|
| 1 | `cd go && go vet ./...` | çıkış 0, uyarı yok |
| 2 | `cd go && go build -mod=mod -o bin/site ./cmd/site` | çıkış 0 |
| 3 | `cd go && go test ./...` | çıkış 0; 13 paket `ok`, 7 paket `[no test files]` |
| 4 | `cd go && gofmt -l .` | boş (üretilen `_templ.go` hariç) |
| 5 | `cargo check/clippy/test --locked` (iki servis) | temiz; image-svc **0 test**, search-svc **1 test** |
| 6 | `go test -tags=integration ./internal/httpapi/` | **FAIL** — SQLSTATE 28P01 (kimlik/port) |
| 7 | `bash deploy/smoke.sh https://theblock.com.tr` | **2 HATA** |
| 8 | `systemctl --user list-timers` | yedek timer **yok**; `theblock-wake.service` **failed** |
| 9 | `ls /home/alper/theblock-backups` | **dizin yok** → hiç yedek alınmamış |
| 10 | `journalctl -u theblock-site --since '3 days ago'` | 713.296 INFO · 11.770 WARN · 55 ERROR |
| 11 | `git status --short` | 42 değişmiş + 15 izlenmeyen dosya |
| 12 | `grep -rn 'TODO\|FIXME\|HACK' go rust` | gerçek borç kaydı yok (yalnız `\uXXXX` gibi yanlış eşleşmeler) |

---

## 3. Bulgular

### A — Üretim / işletme (S1–S2)

#### E-1 · S1 · Otomatik yedekleme hiç kurulmamış
**Kanıt:** `deploy/systemd/theblock-backup.service` ve `.timer` depoda var (25 Ekim),
ancak `systemctl --user list-unit-files 'theblock*'` yalnızca 6 birim listeliyor ve
**backup birimi aralarında yok**; `list-timers --all` çıktısında yedek timer'ı yok;
`BACKUP_ROOT` varsayılanı `/home/alper/theblock-backups` **dizini hiç oluşmamış**.
Ayrıca `deploy/scripts/theblock-backup.sh` izni `-rw-r--r--` → çalıştırılamaz.
**Risk:** Canlı sitede 203 MB `storage` + 12 MB `storage-private` (röportaj sesleri) + PostgreSQL
tamamı yedeksiz. Tek disk arızası = toplam kayıp.
**Kök neden:** Betik ve unit dosyaları yazıldı, kurulum adımı (`systemctl --user enable --now`) hiç yapılmadı; `chmod +x` atlandı.
**Çözüm:**
1. `chmod +x deploy/scripts/theblock-backup.sh deploy/scripts/omniroute-watcher.sh`
2. `systemctl --user enable --now theblock-backup.timer`
3. Betiğe **saklama politikası** (son N yedek) ve **makine dışı kopya** (rclone/scp) ekle — aynı diskteki yedek yedek değildir.
4. Betiğe geri yükleme provası ekle: `pg_restore` ile boş DB'ye açılabildiği doğrulanmalı.
**Kabul:** `systemctl --user list-timers` yedek timer'ını gösterir · `ls $BACKUP_ROOT` en az 1 tarihli
dizin içerir · `pg_restore --list db.dump` okunabilir çıktı verir · geri yükleme provası
`docs/ILERLEME.md`'ye kanıtla yazılır.

#### E-2 · S2 · RSS/ajans botu: çekimlerin %55'i başarısız
**Kanıt (3 gün):** `msg="feed çekilemedi"` **10.296** · `msg="feed çekildi"` **8.293**.
Her denemede hata veren 9 kaynak: Hürriyet Dünya (990), DHA (983), İHA (982), Technopat (982),
Sporx (982), Fanatik (982), Bigpara (982), NTV Spor (982), DW Türkçe (982).
Hata dağılımı (1 gün): `attribute name without = in element` 1.782 · `invalid UTF-8` 367 ·
`invalid character entity &l` 367 · `invalid character entity &` 365+87.
**Kök neden** — `go/internal/repo/bot.go:183` `xml.Unmarshal([]byte(doc), &feed)`:
| Belirti | Neden |
|---|---|
| `invalid character entity &` | Katı XML modu; kaçırılmamış `&` yaygın. `Decoder.Strict = false` ayarlanmamış |
| `invalid UTF-8` | `CharsetReader` tanımlı değil; windows-1254 / ISO-8859-9 Türkçe feed'ler çözülemiyor |
| `attribute name without = in element` (satır 1) | Yanıt XML değil (bot koruması / HTML hata sayfası / onay duvarı). `Content-Type` doğrulaması yok |
| satır 27/42/111 hataları | `io.LimitReader(resp.Body, 2<<20)` → 2 MB'ı aşan feed **sessizce ortadan kesiliyor** |
Depo genelinde `Strict` veya `CharsetReader` kullanımı **yok** (grep ile doğrulandı).
**Çözüm:** `parseRSS`'i `xml.NewDecoder` + `Strict=false` + `CharsetReader`
(`golang.org/x/net/html/charset`) ile yeniden yaz; `&`→`&amp;` ön temizliği uygula;
`Content-Type` XML değilse ayrı hata tipi döndür; limiti 8 MB'a çıkar ve **kırpma olduğunda açıkça hata ver**
(sessiz kırpma yok); hata sebebini `feeds.last_error`'a tiple yaz.
**Kabul:** Önce kırmızı test — 9 başarısız kaynağın gerçek gövdeleri `testdata/` altına alınır,
her biri için ayrıştırma testi yazılır (silinmez) · düzeltme sonrası `feed çekilemedi` oranı
24 saatte **< %5** · panelde `/admin/bot` her kaynağın son hata tipini gösterir.

#### E-3 · S2 · `theblock-wake.service` başlamıyor (203/EXEC)
**Kanıt:** `Active: failed (Result: exit-code)`, `status=203/EXEC`,
`Unable to locate executable '/home/alper/theblock/deploy/scripts/omniroute-watcher.sh': Permission denied`.
Dosya izni `-rw-r--r--`. Timer her gün 03:00'te tetikliyor ve her gün başarısız oluyor.
**Çözüm:** `chmod +x`; ayrıca unit'e `ExecStart=/bin/bash <betik>` savunması veya depoda
izin bitinin korunduğunu doğrulayan bir kontrol (`git ls-files -s` mod 100755) ekle.
**Kabul:** `systemctl --user start theblock-wake.service` → `Active: inactive (dead)` başarıyla;
bir sonraki 03:00 tetiklemesinde günlükte hata yok.

#### E-4 · S2 · Canlı smoke 2 hata veriyor → CI `main`'de kırmızı
**Kanıt:** `FAIL /hesabim/rozetler 302 (beklenen 200) başarımlar` ve
`FAIL / içerik yok: hikayeData` → `SONUÇ: 2 HATA ❌`.
`.github/workflows/ci.yml` içinde `smoke` işi `needs: [go, rust]` ile zorunlu bir kapı; yani
**her push'ta CI bu yüzden kırmızı.**
**Kök neden (iki ayrı sorun):**
1. `/hesabim/rozetler` `server.go:330`'da **`s.uyeRoute`** ile tanımlı (üye zorunlu) — oturumsuz istek
   doğru davranışla 302 veriyor. Smoke beklentisi (`200`) hatalı; `deploy/smoke.sh:51`.
2. `hikayeData` yalnızca `.StoriesJSON` boş değilse basılıyor (`site_public.html:1735`). Üretimde
   hikaye şeridi dolu değil → iddia **değişken içeriğe bağlı**, kalıcı kapı olamaz.
**Çözüm:** Smoke'u iki katmana ayır — (a) *altyapı* iddiaları (200/301/başlık/şema, içerikten bağımsız),
(b) *içerik* iddiaları ayrı, bilgilendirici ve CI'yı kırmayan bir kümede. `/hesabim/rozetler` için
beklentiyi 302'ye çevir veya oturumlu istekle doğrula.
**Kabul:** `bash deploy/smoke.sh https://theblock.com.tr` → `0 HATA` · CI `smoke` işi yeşil ·
boş veritabanlı taze kurulumda da smoke geçer.

#### E-5 · S2 · Veritabanı check-constraint ihlalleri (üretimde)
**Kanıt (3 gün, `level=ERROR`):**
- 3× `msg="stripe okuyucu kaydı" error=... violates check constraint "users_role_check" (SQLSTATE 23514)`
- 3× `msg="media kaydı" error=... violates check constraint "media_files_kind_check" (SQLSTATE 23514)`
**Anlam:** Kod, şemanın kabul etmediği bir `role` ve bir `kind` değeri yazmaya çalışıyor — yani
abonelik (Stripe) akışında kullanıcı kaydı ve bir medya kaydı **sessizce düşüyor**.
**Çözüm:** İki değeri tespit et (`users_role_check` ve `media_files_kind_check` tanımları vs. yazan kod);
ya migration ile izinli değer kümesini genişlet ya da çağıran kodu düzelt. Her ikisi için önce
kırmızı integration testi.
**Kabul:** İki senaryonun kırmızı testi yeşile döner · 7 gün boyunca günlükte 23514 yok.

#### E-6 · S2 · Servis kararlılığı: 3 günde 143 başlangıç ve bir restart döngüsü
**Kanıt:** `msg="theblock site dinliyor"` 3 günde **143** kez. 27 Ekim 06:34'te
`Main process exited, status=1/FAILURE` → `Scheduled restart ... counter is at 1` → `at 2` zinciri.
26 Ekim 22:37'de `status=2/INVALIDARGUMENT`. Ayrıca bir kez
`migration bağlantısı kurulamadı ... the database system is starting up`.
**Anlam:** Dağıtımlar elle restart'la yapılıyor olsa bile bu sıklık normal değil; en az bir gerçek
çökme döngüsü ve bir **başlangıç sırası yarışı** (site, Postgres hazır olmadan migration deniyor) var.
**Çözüm:** `theblock-site.service`'e Postgres'e `After=`/`Requires=` veya migration'da geri çekilmeli
yeniden deneme (exponential backoff) ekle; `status=1`/`status=2` çıkışlarının sebebini günlükten
çıkar ve her biri için kırmızı test/koruma yaz.
**Kabul:** 7 gün boyunca plansız restart = 0 · Postgres yeniden başlatıldığında site kendini
migration hatasıyla düşürmüyor.

#### E-7 · S3 · Günlük gürültüsü sinyali boğuyor
**Kanıt:** 3 günde 713.296 INFO, 11.770 WARN. ERROR'ların çoğu gerçek hata değil:
35× `templ panel error="context canceled"`, 11× `sohbet mesaj parçası basılamadı error="context canceled"`,
2× `broken pipe` / `connection reset by peer`. Bunlar istemcinin sekmeyi kapatmasıdır.
**Çözüm:** `errors.Is(err, context.Canceled)` ve `net.ErrClosed`/`EPIPE` durumlarını DEBUG'a indir;
RSS "feed çekildi" gibi yüksek hacimli INFO'ları örnekle veya sayaç metriğine çevir.
**Kabul:** Normal günde ERROR satırı < 5 · `journalctl | grep ERROR` çıktısı eyleme değer satır içerir
(DoD §1 G7 kapısı anlamlı hale gelir).

### B — Kapanmamış iş (S1–S3)

#### E-8 · S1 · Faz 125 işi commit edilmemiş: 57 dosya havada
**Kanıt:** `git status` → 42 değişmiş + 15 izlenmeyen dosya; `git diff --stat` 2.292 ekleme / 1.583 silme.
İzlenmeyenler arasında **kalıcı ürün kodu** var: `go/internal/transcribe/{config,naming,waveform,waveform_test}.go`,
`go/internal/httpapi/asr_settings_handlers.go`, `go/internal/webui/templkit/ses_ayarlari.templ`,
`go/internal/webui/static/ses_player.js`, **`go/migrations/0088_audio_waveform.{up,down}.sql`**.
`docs/ILERLEME.md` en son Faz 124'te duruyor — bu iş günlüğe de girmemiş.
**Risk:** Migration dosyası commit'siz → başka bir ortamda şema uyuşmazlığı. CI bu kodu hiç görmedi.
**Çözüm:** İşi "Faz 125 — dalga formu + ASR sağlayıcı ayarları" olarak tamamla, `ILERLEME.md`'ye
kanıt bloğu yaz, `make check` sonrası commit et.
**Kabul:** `git status` temiz · CI yeşil · `ILERLEME.md`'de Faz 125 kanıt bloğu var.

#### E-9 · S3 · Geçici dosyalar ve ikili dosya depo kökünde
**Kanıt:** `go/dbq` izlenmeyen **ELF 64-bit yürütülebilir** dosya (`.gitignore` `go/bin/` kapsıyor,
kök seviyedeki bu ikiliyi kapsamıyor). `go/cmd/tmpwave/main.go` başlığı: *"geçici: mevcut bir
transkript için dalga formunu yeniden üretir"*. `go/cmd/tmpsession` ve `e2e/dbg_player.js` de
hata ayıklama artığı. `go/scripts/whisper_transcribe.py` izlenmiyor.
**Çözüm:** `.gitignore`'a `go/dbq` ve derlenmiş çıktı deseni ekle; `tmpwave`/`tmpsession`'ı ya
`cmd/admintool` gibi kalıcı bir ada taşı ve test et ya da sil; `whisper_transcribe.py`'ye karar ver
(ürün parçası mı, deneme mi) ve ona göre commit et veya sil.
**Kabul:** `git status --untracked-files=all` yalnızca kasıtlı dosyaları gösterir · `cmd/` altında
"geçici" ibareli komut kalmaz.

### C — Test ve CI kapıları (S3)

#### E-10 · S3 · `scheduler` paketi testsiz — hatanın yaşadığı yer tam burası
**Kanıt:** `go/internal/scheduler/` → 8 kaynak dosya, **1.224 satır, 0 test dosyası**.
Bu paket RSS çekimi, trend skorlama, podcast üretimi, servis poller'ı, planlı yayın ve abonelik
hatırlatmalarını yürütüyor. E-2'deki 10.296 hatalı çekim burada yaşıyor.

**Korumasızlığın somut örneği:** `scheduler.go:128` `"abonelik hatırlatma listesi alınamadı"`
uyarısı 26 Ekim 06:06 – 13:32 arasında **890 kez** oluştu:
`failed to encode args[0]: unable to encode 3 into text format for text (OID 25)` — yani sorguya
`text` kolonu için `int` parametre geçilmiş (tip hatası). Son 24 saatte **0 kez** görüldüğü için
bu hata sonradan (muhtemelen tesadüfen, başka bir değişiklikle) kapanmış görünüyor —
**ama onu yakalayan bir test yok**, yani geri gelebilir. Testsiz bir paketin 7,5 saat boyunca
bir işi hiç çalıştırmadan sessizce dönebildiğini gösteren tam olarak bu vaka.
**Çözüm:** İş fonksiyonlarını saf (pure) çekirdek + ince kabuk olacak şekilde ayır; çekirdeğe
tablo-tabanlı testler yaz. Asgari kapsam: feed ayrıştırma (E-2 `testdata/`), planlı yayın geçişi,
çöp 30 gün temizliği, abonelik hatırlatma sorgusu, poller geri çekilme davranışı.
**Kabul:** `scheduler` paketi için `go test` en az 15 test fonksiyonuyla yeşil ·
abonelik hatırlatma sorgusu için parametre tiplerini doğrulayan bir test var (geri dönüşü engeller) ·
her periyodik iş için "hata durumunda ne olur" senaryosu test edilmiş.

#### E-11 · S3 · `make test-int` / `make verify` yerelde kırık
**Kanıt:** `Makefile:19` → `TEST_DATABASE_URL ?= postgres://theblock:theblock@localhost:**5432**/theblock_test`.
Ancak `deploy/compose.yml` PostgreSQL'i **5433**'te yayınlıyor (`ss -ltnp` doğruladı) ve
`theblock:theblock` kimliği geçmiyor: `failed SASL auth: FATAL: password authentication failed` (28P01).
Yani integration katmanı **yalnızca CI'da** çalışıyor; geliştirici `make verify` çalıştıramıyor.
**Çözüm:** `deploy/scripts/gen-env.sh`'a test veritabanı hazırlama adımı ekle (rol + `theblock_test` DB);
`Makefile` varsayılanını compose portuyla hizala; `make test-int` DB yoksa anlaşılır bir mesajla
kurulum komutunu söyleyerek çıksın.
**Kabul:** Taze ortamda `make test-int` ve `make verify` yeşil · README/docs'ta tek komutluk kurulum yazılı.

#### E-12 · S3 · E2E paketi doğrulanmadı — ve **yerelde çalıştırılamaz**
**Not:** 56 E2E test dosyası var, CI'da `e2e` işi tanımlı; ancak hiç çalıştırılmadı.
Durumu bilinmiyor — yeşil varsayılmamalı.

> **DÜZELTME (28 Ekim 2026).** Bu maddenin ilk hâli "`make e2e-install && make e2e`
> bir kez çalıştırılsın" diyordu. **Bu öneri yanlıştı ve tehlikeliydi.**
>
> **E2E YEREL ÇALIŞTIRILAMAZ:** `Makefile`'da `E2E_BASE_URL ?= http://127.0.0.1:$(PORT)`
> ve `PORT ?= 8070`. O portta **canlı üretim servisi** çalışıyor
> (`theblock-site.service`). 56 E2E dosyasının **40'ı** POST ediyor / panele veri
> yazıyor. Yerel `make e2e` canlı sitede içerik oluşturup silerdi.
>
> E-11'de "integration testleri yalnız CI'da çalışıyor" diye not düşülmüştü; E2E
> için durum daha keskin — yerel çalıştırma yalnızca zahmetli değil, **zararlı**.

**Çözüm:** E2E yalnız iki yerde koşmalı:
1. **CI** (önerilen) — `.github/workflows/ci.yml` kendi PostgreSQL'ini kurar,
   migration'ları uygular, `e2e_seed.sql` yükler, **ayrı bir site örneği** başlatır
   ve E2E'yi ona karşı koşar. Üretime dokunmaz. Tek gereken: dalı push etmek.
2. Ayrı porta kurulmuş yerel test örneği + `theblock_test` veritabanı (E-11 ile
   birlikte çözülür).

**Engel (28 Ekim 2026):** Çalışma dalı `faz-125-dalga-formu-asr-ayarlari` ve
**230 commit gönderilmemiş**. Yani bugüne kadar yapılan hiçbir işte CI çalışmadı.
Push, hem E-12'yi çözer hem de tüm geçmişin tek makinede durması riskini kapatır.

**Kabul:** CI'daki `e2e` işi yeşil; sonuç `ILERLEME.md`'ye kanıtla yazılmış.
`Makefile`'a yerel çalıştırmanın üretime vuracağını söyleyen bir uyarı eklenmiş
(veya `E2E_BASE_URL` varsayılanı üretim portundan ayrılmış).

> **KOŞULDU (28 Ekim 2026, öğleden sonra).** E2E paketi ilk kez uçtan uca
> çalıştırıldı. CI üzerinden olmadı — aşağıdaki iki engel yüzünden yerelde
> izole bir ortam kurularak yapıldı.
>
> **Engel 1 — dalı push etmek CI'ı tetiklemiyor.** `ci.yml` tetikleyicisi
> `push: branches: [main]`, `pull_request` ve `workflow_dispatch`. Bu maddenin
> "tek gereken: dalı push etmek" cümlesi eksikti; PR gerekiyor. Dal push edildi
> (231 commit) ve **PR #1** açıldı.
>
> **Engel 2 — Actions faturalandırma.** Koşum `40677564619` 4 saniyede, hiçbir
> iş başlamadan düştü: *"The job was not started because recent account
> payments have failed or your spending limit needs to be increased."* Depo
> özel olduğu için Actions dakikaları hesaptan düşüyor. Kod tarafında
> çözülemez; GitHub Billing ayarı gerekiyor. Düzeltilince
> `gh run rerun 40677564619`.
>
> **Yapılan:** `deploy/scripts/e2e-izole.sh` (+ `make e2e-izole`) CI'ın `e2e`
> işini yerelde birebir kurar ve üretimden dört noktada ayrışır: ayrı
> veritabanı (`theblock_e2e`), ayrı ikili (`go/bin/site`'ın **üzerine yazmaz**
> — o çalışan servisin ikilisidir), ayrı port + depolama, ve `deploy/.env`
> yüklenmez (`env -i`, yalnız CI'ın verdiği dört değişken). Ayrıca `make e2e`
> artık canlı portu hedeflerse kendini durduruyor
> (`deploy/scripts/e2e-uretim-korumasi.sh`, `FORCE=1` ile geçilir) — bu
> maddenin "Kabul" şartındaki uyarı koşulu böylece karşılandı.

**Sonuç (izole koşum, 28 Ekim):** `192 test · 147 geçti · 38 düştü · 7 atlandı`
(9 dk 15 sn). Yani maddenin gerçek içeriği "E2E hiç koşmadı" değil:
**paket mevcut hâliyle CI'da da geçmeyecekti.** 38 hata üç kök nedene ayrılıyor
ve hiçbiri üretim regresyonu değil:

| # | Kök neden | Kanıt | Durum |
|---|---|---|---|
| 1 | **Testlerde sabit üretim adresi** | `seo.test.js` ve `rss.test.js` içinde 5 yerde `http://127.0.0.1:8070` gömülü; `E2E_BASE_URL` yok sayılıyor. İzole koşumda bunlar `net::ERR_FAILED` verdi — harness'ın çapraz kaynak engeli şans eseri canlı siteye gitmelerini önledi. Varsayılan `make e2e`'de **doğrudan üretime** giderlerdi. | ✅ düzeltildi; `seo.test.js` yeşile döndü |
| 2 | **`e2e_seed.sql` eski** | Seed'e en son 19 Ekim'de (`2040801`) dokunuldu; kategori/tasarım testleri Faz 114–132'de eklendi. Seed'de tek `INSERT INTO categories` var ve **hiç alt kategori yok**, `contents` yalnız 3 satır. Bu yüzden "Ekonomi mega menüsü", "Borsa alt kategori olmalı", "`/kategori/gundem` → HTTP 404", "Gündem akışı olmalı", kategori akış/modül/manşet testleri düşüyor. | ⏳ açık — hataların çoğunluğu |
| 3 | **Eksik yan servis** | `editor.test.js` `.ProseMirror-yjs-cursor` arıyor, yani Yjs collab sunucusunu (port 1234) gerektiriyor. Ne izole ortam ne **CI** `COLLAB_URL`/`COLLAB_SECRET` geçiyor → bu test CI'da da düşerdi. | ⏳ açık — ya CI'a collab servisi eklenmeli ya test `COLLAB_URL` yoksa `skip` etmeli |


**Son durum (28 Ekim, seed düzeltmelerinden sonra):**
`192 test · 170 geçti · 15 düştü · 7 atlandı`. Başlangıç `147 geçti · 38 düştü`
idi; hataların %60'ı kapandı. Kapananların tamamı fikstür/veri kaynaklıydı,
düzeltme `deploy/e2e_seed.sql` içinde kaldı — ürün kodu değişmedi.

Kalan 15 hatanın tasnifi:

| Adet | Kök neden | Kime ait |
|---|---|---|
| 7 | Kategori sayfası düzen modülleri: üçlü manşet, `kategori_4lu`/`kategori_slider` önizlemesi, manşet stil paneli, `module_category` varsayılan akış | fikstür mü ürün mü ayrımı yapılmadı; her biri tek tek incelenmeli |
| 2 | `#kpList` kaldırıldı, testler güncellenmedi | **E-24** — refaktörü yapan |
| 2 | collab (Yjs) ve search-svc ayakta değil; CI de başlatmıyor | **E-25** |
| 1 | `ust_manset_styles`: "Bloomberg II sol boşluğu 16px olmalı", gelen 12px | ürün/test anlaşmazlığı, tasarım kararı |
| 1 | `ai_source`: kategori planı düğmesi sayısı | veri (plan kayıtları) |
| 1 | `builder.test.js` | incelenmedi |
| 1 | `tts.test.js` mobil mini oynatıcı | incelenmedi |

Önemli: bu 15'in hiçbiri **üretim regresyonu olarak doğrulanmadı**, ama
hiçbiri "yeşil" de sayılmamalı. E-24 somut bir regresyon (bugünkü refaktör),
E-25 CI yapılandırma eksiği.

**Kalan iş:** (2) seed'i Faz 114–132 testlerinin beklediği veriyle zenginleştir
(alt kategoriler: Ekonomi→Borsa; `gundem` kategorisi ve kategori başına yeterli
içerik; servis bandı / RSS kaynak satırları), (3) collab bağımlı testi koşullu
hâle getir. Bunlar yapılmadan "E2E yeşil" denemez.

#### E-13 · S3 · CI kapıları yumuşak veya eksik
**Kanıt (`.github/workflows/ci.yml`):**
| Satır | Sorun |
|---|---|
| 181 | `govulncheck ./... \|\| true` — güvenlik bulgusu **asla** CI'yı kırmaz |
| 190 | `cargo audit \|\| true` — aynı |
| — | `gofmt -l` kontrolü yok (şu an temiz; koruma yok) |
| — | `templ generate` **drift kontrolü yok**: üretilen `_templ.go` dosyaları commit'li, elle senkron riski (şu anda 25+ `_templ.go` değişmiş durumda) |
| — | `editor-src` (bundle) ve `collab-server` derlenmiyor/test edilmiyor |
| — | `image-svc` **0 test** içerdiği için `cargo test` boş geçiyor; yanlış güven veriyor |
**Ek:** `templ` ikilisi yerel ortamda **kurulu değil** → `make templ` çalışmaz.
**Çözüm:** `govulncheck`/`cargo audit`'i bilinen-istisna listesiyle **zorunlu** yap;
`gofmt -l` boş değilse başarısız ol; `templ generate` sonrası `git diff --exit-code` ile drift yakala;
`editor-src` ve `collab-server` için build/lint işi ekle; `image-svc`'ye kırpma/WebP dönüşümü için
en az 3 birim testi yaz.
**Kabul:** CI tüm işlerde yeşil ve **hiçbir adımda `|| true` yok** · `templ` drift'i CI'da yakalanıyor.

#### E-14 · S3 · Testsiz paketler ve Rust tarafı
**Kanıt:** Test dosyası olmayan Go paketleri — `config` (76 satır), `db` (222), `i18n` (139),
`mailer` (62), `searchsvc` (167), `webui` (7), `scheduler` (1.224 → E-10).
Rust: `image-svc/src/main.rs` 197 satır **0 test**; `search-svc` 1 test.
**Çözüm:** Öncelik sırası: `scheduler` (E-10) → `searchsvc` (fallback davranışı) → `db` (bağlantı/retry)
→ `i18n` (çeviri çözümleme) → `mailer` (SMTP zarf oluşturma) → `image-svc` (16:9 kırpma + WebP).
**Kabul:** Her paket için en az kritik yolu kapsayan test · `make cover` raporunda bu paketler %0 değil.

### D — Doküman ve süreç (S4)

#### E-15 · S4 · `docs/ADR.md` yok
`docs/README.md:7` "ADR.md — mimari karar günlüğü (**oluşturulacak**)" diyor; dosya 14 Ekim'den
beri oluşturulmamış. Oysa kayda değer kararlar alınmış: Laravel'in fiilen düşürülmesi, templ'e geçiş,
HTMX'e geçiş (Faz 49–99), AssemblyAI'ın SDK yerine REST ile kullanılması
(gerekçe `assemblyai.go` yorumunda gömülü kalmış), Deepgram'ın ikinci ASR sağlayıcısı olması.
**Kabul:** `docs/ADR.md` oluşur; en az bu 5 karar *bağlam / karar / sonuç* biçiminde yazılır.

#### E-16 · S4 · `AGENTS.md` yok ama spec ona atıfta
`03-spec-plan.md` §7 başlığı "Kapı & Onay Süreci (**AGENTS.md uyumu**)" ve `04-DoD.md` başlığı
"`AGENTS.md` kapıları" diyor; depo kökünde ne `AGENTS.md` ne `CLAUDE.md` var.
**Kabul:** Kapı kuralları (kırmızı test zorunluluğu, `ILERLEME.md` kanıt bloğu, sır yönetimi,
`main`'e birleşme onayı) `AGENTS.md`'de yazılı — spec'teki atıf karşılığını bulur.

#### E-17 · S4 · `docs/README.md` indeksi güncel değil
39 doküman var, indekste **20** satır → 19 doküman listelenmemiş
(ADMIN-*, GAZETE-*, KART-*, KATEGORI-MENU, UYE-SISTEMI, PERFORMANS, SEO-DENETIM, TEMA, RSS,
HEADER-FOOTER, YAZARLAR-EDITORLER-SPEC, AI-KAYNAK-MERKEZI, MODUL-STILLERI, SESLI-HABER-TTS vb.).
**Kabul:** İndeks 39 dokümanı kapsar · indeksi tazeleyen küçük bir kontrol (CI veya `make docs-check`) eklenir.

#### E-18 · S4 · `docs/ILERLEME.md` 284 KB / 124 faz — tek dosya sınırına dayandı
Dosya artık rahat okunamıyor/aranamıyor ve her fazda çatışma riski taşıyor.
**Çözüm:** Faz aralıklarına göre böl (`ILERLEME/2026-10-faz-001-050.md` …) ve `ILERLEME.md`'yi
özet + bağlantı indeksine çevir. Kanıt blokları korunur (DoD §4 gereği).
**Kabul:** Hiçbir ilerleme dosyası > 80 KB · eski kanıtlar erişilebilir · `ILERLEME.md` indeks olarak çalışır.

#### E-19 · S4 · Laravel: doküman ile gerçek uyuşmuyor
`README.md:3` "Laravel panel (kademeli emeklilik)" ve `03-spec-plan.md` §2'de Laravel bir katman
olarak listeli; ancak `laravel/` dizininde **yalnızca README.md** var — panel hiç yazılmadı, tüm
panel Go+templ+HTMX ile yapıldı. `.gitignore` hâlâ `laravel/vendor`, `laravel/node_modules` satırlarını taşıyor.
**Çözüm:** Kararı E-15'teki ADR'ye yaz ("Laravel geçiş paneli yapılmadı; panel Go/templ/HTMX'te"),
`README.md` ve `03` §2'yi gerçeğe çek, `laravel/` dizinini kaldır veya ADR'ye işaret eden tek satırlık
bir not bırak.
**Kabul:** Mimari dokümanlarında artık yapılmamış bir katman aktif gösterilmiyor.

#### E-20 · S4 · `03-spec-plan.md` kutucukları ile `04-DoD.md` çelişiyor
`03` §4'teki P0–P5 kapsam maddelerinin **tamamı hâlâ `[ ]`** işaretsiz, oysa `04-DoD.md` aynı
kalemleri ✅ sayıyor ve DoD §4 "ilgili satırlar yeşile dönmüş olmalı" diyor. İki kaynak çeliştiği
için ikisi de güvenilmez hale geliyor.
**Çözüm:** Tek doğruluk kaynağı `04-DoD.md` olsun; `03`'teki kutucuklar DoD'ye atıfla işaretlensin
veya kaldırılsın. `04-DoD.md` §1 global kapı tablosu da bu denetimin sonuçlarıyla güncellenmeli
(bugün 3 kapı kırmızı ama tablo hepsini ✅ gösteriyor).
**Kabul:** Bir kalemin durumu tek yerde yazılı; `04-DoD.md` §1 gerçek kapı durumunu yansıtıyor.

### E — Mimari bakım (S3)

#### E-21 · S3 · Aşırı büyümüş dosyalar
| Dosya | Satır |
|---|---|
| `go/internal/httpapi/admin_handlers.go` | 2.901 |
| `go/internal/httpapi/builder_handlers.go` | 2.562 |
| `go/internal/httpapi/public_handlers.go` | 1.841 |
| `go/cmd/seed/main.go` | 1.270 |
| `go/internal/httpapi/tmpl.go` | 1.269 |
| `go/internal/repo/trend_center.go` | 1.054 |
| `go/internal/repo/services_fetch.go` | 1.016 |
**Çözüm:** Konuya göre böl (`admin_content.go`, `admin_media.go`, `admin_settings.go` …). Davranış
değişmez, bu yüzden mevcut testler koruma sağlar — bölme öncesi/sonrası `go test ./...` aynı kalmalı.
**Kabul:** Tek dosya > 1.200 satır kalmaz · test sonucu bölme öncesiyle birebir aynı.

#### E-22 · S3 · DoD'deki 4 "⚠️ kısmi" kalem hâlâ açık
`04-DoD.md` §3: (1) AdSense gerçek gelir çekimi, (2) GA4 Data API rapor çekimi,
(3) Nöbetçi eczane gerçek kaynak, (4) Google Indexing API servis hesabı.
Bunların üçü **dış kimlik bilgisi** bekliyor (kod hazır) — teknik iş değil, **senin aksiyonun**.
`repo/revenue.go:36` yorumu da bunu doğruluyor: *"ADSupported gerçek AdSense API bağlı mı (şimdilik false)"*.
**Çözüm:** Kimlik bilgisi gerektirenleri ayrı bir "dış bağımlılık bekleyenler" listesine taşı ki
mühendislik kuyruğunu tıkamasın. Nöbetçi eczane için gerçek kaynak URL'si bulunabilir (kod
`servis/eczane_kaynak` ayarıyla `{plaka}` şablonunu zaten destekliyor — `services_fetch.go:240`).
**Kabul:** Her kalem ya bağlanmış ya "dış bağımlılık bekliyor" olarak etiketli; demo veri
kullanıcıya "demo" rozetiyle gösteriliyor.

---

## 4. Faz planı (Faz 125 → 136)

Sıra, bağımlılık ve risk sırasına göre. Her faz **tek oturumda bitebilir** büyüklükte tutuldu ve
`04-DoD.md` üçlü kanıt kuralına (vet/test temiz · uçtan uca kanıt · `ILERLEME.md` kaydı) uyar.

| Faz | İş | Bulgular | Önem | Neden bu sırada |
|---|---|---|---|---|
| **125** | Bekleyen ses/dalga formu işini kapat ve commit et | E-8 | S1 | 57 dosya havada; her şeyin üstüne yazılıyor |
| **126** | Yedekleme kur: `chmod +x`, timer, saklama, makine dışı kopya, **geri yükleme provası** | E-1, E-3 | S1 | Tek veri kaybı riski; başka her iş bunu bekler |
| **127** | RSS ayrıştırıcıyı sağlamlaştır (`Strict=false`, `CharsetReader`, Content-Type, limit) + `testdata/` kırmızı testleri | E-2 | S2 | İçerik akışının yarısı kayıp |
| **128** | Smoke'u altyapı/içerik olarak ayır; iki FAIL'i gider → CI yeşile dön | E-4 | S2 | CI kırmızıyken sonraki fazların kanıtı zayıf |
| **129** | İki check-constraint ihlalini kırmızı testle yakala ve düzelt | E-5 | S2 | Sessiz veri kaybı (Stripe kaydı + medya) |
| **130** | `scheduler` paketine test tabanı (periyodik işlerin hata senaryoları dahil) | E-10 | S3 | 1.224 satır korumasız; 127/129 buraya dayanıyor |
| **131** | Servis kararlılığı: başlangıç sırası, migration retry, restart sebepleri | E-6 | S2 | 130'daki testler yarışları açığa çıkarır |
| **132** | Günlük seviyelerini düzelt (context canceled → DEBUG, INFO örnekleme) | E-7 | S3 | DoD G7 kapısını anlamlı yapar |
| **133** | CI kapılarını sıkılaştır: `\|\| true` kaldır, gofmt, templ drift, editor/collab, image-svc testleri | E-13, E-14, E-12 | S3 | Önceki fazların kazanımını kalıcılaştırır |
| **134** | `make test-int`/`make verify` yerelde çalışsın; test DB kurulumu betikle | E-11 | S3 | Katkı akışını açar |
| **135** | Depo hijyeni: `go/dbq` ignore, `tmpwave`/`tmpsession` karar, `whisper_transcribe.py` | E-9 | S3 | Küçük, 133 ile birlikte doğrulanır |
| **136** | Doküman borcu: ADR.md, AGENTS.md, docs indeksi, ILERLEME bölme, Laravel/`03`↔`04` tutarlılığı | E-15…E-20 | S4 | Kod kararları netleştikten sonra yazılmalı |
| **137+** | Dosya bölme (admin/builder/public handlers) | E-21 | S3 | Davranış değişmeden, testler koruma altındayken |
| **(dış)** | AdSense / GA4 / Indexing API / eczane kaynağı | E-22 | — | Kimlik bilgisi sende; mühendislik kuyruğunu tıkamaz |

### Bağımlılıklar
```
125 (commit) ──► 126 (yedek) ──► 127 (RSS) ──► 130 (scheduler testleri) ──► 131 (kararlılık)
                      │              │
                      └──► 128 (smoke/CI) ──► 133 (CI sıkılaştırma) ──► 134 (yerel kapılar)
                                   │
                           129 (constraint) ──┘          132 (günlük) ──► 136 (doküman)
```

### Hemen sıradaki 3 adım
1. **Faz 125** — bekleyen işi `make check` sonrası commit et, `ILERLEME.md`'ye kanıt bloğu yaz.
2. **Faz 126** — yedekleme kur ve **geri yükleme provasını yap**. Alınmamış yedek yedek değildir;
   açılabildiği doğrulanmamış yedek de yedek değildir.
3. **Faz 127** — RSS ayrıştırıcısı. Önce 9 başarısız kaynağın gövdelerini `testdata/`'ya al, kırmızı
   testleri yaz, sonra düzelt (DoD "her bilinen hata için önce kırmızı test" kuralı).

---

## 5. `04-DoD.md`'ye eklenecek satırlar

Bu spec kabul edilirse DoD §1 tablosu gerçeğe çekilmeli ve şu kapılar eklenmeli:

| # | Kapı | Komut | Bugünkü durum |
|---|---|---|---|
| G9 | Yedek alınıyor ve **geri yüklenebiliyor** | `ls $BACKUP_ROOT && pg_restore --list .../db.dump` | ❌ hiç yedek yok |
| G10 | Feed sağlığı | 24 saatte `feed çekilemedi` oranı < %5 | ❌ ~%55 |
| G11 | Biçim | `cd go && gofmt -l . \| grep -v _templ.go` boş | ✅ |
| G12 | templ drift | `templ generate && git diff --exit-code` | ❓ doğrulanamadı (`templ` kurulu değil) |
| G13 | Güvenlik taraması **zorunlu** | `govulncheck ./...` (`\|\| true` yok) | ⚠️ şu an yumuşak |

Ayrıca mevcut DoD §1 tablosundaki şu satırlar **düzeltilmeli**: G4 "0038 uygulandı" yazıyor ama
depoda **0088**'e kadar migration var; G6 "canlı sağlık ✅ yeşil" yazıyor ama smoke 2 hata veriyor;
G2 yeşil ✅ (doğru) ama integration katmanı yerelde çalışmıyor (E-11).

---

## 6. Riskler

| Risk | Olasılık | Etki | Azaltma |
|---|---|---|---|
| Disk/VM kaybı → tüm içerik ve röportaj sesleri gider | orta | **kritik** | Faz 126 (makine dışı kopya + prova) |
| Bekleyen 57 dosya kaybolur / migration 0088 ortam uyuşmazlığı yaratır | orta | yüksek | Faz 125 hemen |
| RSS hataları kalıcılaşıp kaynaklar sessizce ölü sayılır | yüksek | orta | Faz 127 + G10 kapısı |
| CI kırmızı olduğu için ekip kapıları görmezden gelmeye alışır | yüksek | yüksek | Faz 128 → 133 |
| `scheduler` testsiz kaldığı sürece her düzeltme yeni regresyon riski | yüksek | orta | Faz 130 |
| Doküman çelişkisi (03 ↔ 04) yüzünden "bitti" sanılan iş bitmemiş olur | orta | orta | Faz 136, tek doğruluk kaynağı |

---

#### E-23 · S2 · Sıfırdan kurulan veritabanında kategori ağacı oluşmuyor

**Kanıt (28 Ekim 2026, izole E2E veritabanında bulundu).** Taze bir veritabanına
migration'lar uygulandığında Ekonomi, Dünya, Yaşam ve Eğitim-kültür altında
**hiçbir alt kategori oluşmuyor**; üst menü de eksik kalıyor. Üretimde sorun
görünmüyor çünkü oradaki kategoriler migration'lardan önce elle girilmişti.

Kök neden, hiyerarşi migration'larının üst kategorinin **zaten var olduğunu**
varsayması:

| Migration | Koşul | Taze veritabanında sonuç |
|---|---|---|
| `0052_category_hierarchy` | `WHERE p.slug = 'ekonomi'` | `ekonomi` yok → 12 alt kategorinin hiçbiri eklenmez |
| `0054_category_gzt_hierarchy` | `WHERE p.slug = 'ekonomi'` / `'spor'` | aynı → 18 alt kategori eklenmez |
| `0055_gundem_rename` | `WHERE p.slug = 'gundem'` | `gundem` yok → 6 alt kategori eklenmez |
| `0056_other_subcategories` | `JOIN categories p ON p.slug = v.parent_slug` | yalnız `teknoloji` var (0038'den) → Dünya/Yaşam/Eğitim-kültür çocukları eklenmez |

Ayrıca `asayis`, `ilceler`, `turizm`, `sanat` kategorileri **hiçbir
migration'da yaratılmıyor** — yalnız 0054'te taşınıyor ve 0058'de çekirdek
taksonomiye dahil ediliyor. Üretimde elle oluşturulmuşlar.

**Etki:** theblock yeni bir ortama (yeni müşteri, yeni site, felaket sonrası
geri yükleme) kurulduğunda mega menü boş gelir, `/kategori/<alt>` sayfaları 404
döner. Yani bu yalnız test altyapısı sorunu değil, **ürün kurulum hatası**.

**Şimdilik yapılan (28 Ekim):** düzeltme yalnız test tarafında —
`deploy/e2e_seed.sql` üst kategorileri kurduktan sonra 0052/0054/0055/0056'nın
listelerini ve 0058'in görünürlük beyaz listesini kendisi uyguluyor. Bununla
`category_menu.test.js` 8/8 yeşile döndü (önce 7'si düşüyordu).

**Önerilen kalıcı çözüm:** yeni bir migration (`0090_category_taxonomy_repair`)
eksik üst kategorileri ve ardından alt kategori ağacını **idempotent** biçimde
kursun. Üretim için güvenli olması adına yalnız `INSERT ... ON CONFLICT DO
NOTHING` ve `parent_id IS NULL` koşullu `UPDATE` kullanmalı; mevcut satırların
`in_menu`/`in_header` değerlerine dokunmamalı (editörün elle yaptığı
görünürlük ayarları ezilmesin).

**Bu oturumda uygulanmadı:** migration zinciri üretim verisine yazar ve o karar
kullanıcıya ait. Seed düzeltmesi testleri açtığı için acil değil.

**Kabul:** boş bir veritabanına yalnız `migrate up` uygulandığında üst menü 8
kategoriyi ve Ekonomi mega paneli en az 10 alt kategoriyi gösteriyor;
`category_menu.test.js` seed'in hiyerarşi bloğu olmadan da geçiyor.

#### E-24 · S3 · Refaktör E2E testlerini geride bıraktı (`#kpList`)

**Kanıt (28 Ekim 2026).** `e2e/tests/kategori_panelleri.test.js` ve
`e2e/tests/gastro_paneli.test.js` hâlâ
`page.waitForSelector('#kpList .kp-row')` ile bekliyor ve 15 saniye sonra
zaman aşımıyla düşüyor. `#kpList` ve `.kp-row` **kod tabanının hiçbir yerinde
yok**:

```
git log -S'kpList' --all
  4b83350 feat(tasarim): kategori panelleri tek tek modüle ayrıldı   ← sildi
  27966d5 Kategori panelleri: Haberturk GASTRO vitrini duzeni
  9d36f86 Kategori panelleri: Haberturk Ekonomi vitrini duzeni
  a03e277 feat(yonetim): Kategori Panelleri Sayfa Tasarimcisindan yonetilebilir
```

`4b83350` (28 Ekim) tek parça "Kategori Panelleri" modülünü ayrı modüllere
böldü ve `site_builder.html`'den `#kpList` listesini kaldırdı; iki E2E dosyası
güncellenmedi.

**Etki:** iki test kalıcı olarak kırmızı. CI faturalandırma yüzünden
çalışmadığı için refaktör anında fark edilmedi — E-12 ve E-13'ün neden önemli
olduğunun somut örneği.

**Çözüm:** testler yeni modül yapısının seçicilerine göre yeniden yazılmalı
(ya da modül başına ayrılmalı). Yeni arayüzün karşılığını yalnız refaktörü
yapan bilir; bu oturumda tahminle yeniden yazılmadı.

**Kabul:** iki dosya da `make e2e-izole` içinde geçiyor.

#### E-25 · S3 · E2E, CI'da ayakta olmayan yan servisleri gerektiriyor

**Kanıt.** İzole koşumda iki test, veri eksiğinden değil **servis
yokluğundan** düşüyor:

| Test | Aradığı | Gereken servis |
|---|---|---|
| `editor.test.js` | `.ProseMirror-yjs-cursor` | `theblock-collab` (Yjs/Hocuspocus, 1234) |
| `search_public.test.js` | Tantivy sonuçları | `theblock-search-svc` (Rust) |

`.github/workflows/ci.yml`'deki `e2e` işi siteyi yalnız dört değişkenle
başlatıyor (`DATABASE_URL`, `PORT`, `APP_ENV`, `STORAGE_DIR`); ne `COLLAB_URL`
veriyor ne de bu iki servisi ayağa kaldırıyor. Yani bu testler **CI'da da
düşerdi** — izole ortamın eksiği değil.

**Çözüm:** ya CI'a bu iki servis eklenmeli, ya testler ilgili ortam değişkeni
(`COLLAB_URL` / `SEARCH_SVC_URL`) yoksa `t.skip()` etmeli. İkincisi daha ucuz
ve dürüst: atlanan test "yeşil" gibi görünmez.

**Kabul:** E2E paketi, yan servisler olmadan da kırmızı vermiyor; servisler
varsa ilgili testler gerçekten koşuyor.

## 7. Bu denetimin kapsamadıkları

Dürüstlük adına, aşağıdakiler **doğrulanmadı** — "yeşil" varsayılmamalı:
- **E2E paketi çalıştırılmadı** (56 dosya) — E-12.
- **Integration testleri çalıştırılamadı** (yerel DB kimliği) — E-11; CI'da geçtiği varsayılıyor.
- Üretimdeki **işlevsel** doğruluk (örn. AI çıktı kalitesi, manşet editörü davranışı) elle
  denenmedi; yalnızca smoke, günlükler ve statik analiz temel alındı.
- **Güvenlik denetimi yapılmadı**: `govulncheck`/`cargo audit` bu oturumda çalıştırılmadı
  (CI'da yumuşak modda koşuyor — E-13). Yetkilendirme/çok-kiracılı izolasyon incelemesi de
  kapsam dışıydı; `/admin` yetki matrisi ve `site_id` izolasyonu ayrı bir denetimi hak ediyor.
- `editor-src` ve `collab-server` derlenmedi.
