# Statik analiz hattı (SonarQube)

CVE taraması **ayrı** hatta: `docs/guvenlik-hatti.md` + `make guvenlik-tara`
(Trivy · govulncheck · cargo audit · npm audit · Semgrep · gosec · zizmor).

Bu doküman **kod kalitesi** hattını anlatır: SonarQube ile statik analiz.

## Neden iki ayrı hat

| | Güvenlik hattı | Kalite hattı |
|---|---|---|
| Soru | "Bu kodda sömürülebilir açık var mı?" | "Bu kod bakımı zor mu, hatalı mı?" |
| Araç | Trivy, govulncheck, gosec, Semgrep | SonarQube |
| Kapı | CI'da (`make guvenlik-tara`) | Yerel SonarQube |
| Hız | ~20 sn | ~30 sn tarama + 5 dk ilk kurulum |

İkisi **birbirinin yerine geçmez**: gosec `SELECT` enjeksiyonunu bulur ama
"bu fonksiyon 40 dal içeriyor" demez; Sonar tersini yapar.

## Kurulum

```bash
make sonar-kur      # konteyner + parola + token + proje (idempotent)
```

Kurulum ayrıntıları ve nedenleri:

| Karar | Gerekçe |
|---|---|
| Port **9100** | 9000 arastam-dev tarafından kullanılıyor |
| `sonarqube:community` | LGPL, ücretsiz, yeterli (Go/JS/TS/Python/k8s/Docker) |
| Veri `~/sonarqube/{data,extensions,logs}` | Konteyner silinse de analiz geçmişi kalır |
| Token `~/.sonar-token` (600) | **Repoya yazılmaz**; `sonar-tara.sh` oradan okur |
| Parola `~/.sonar-admin-pass` (600) | Aynı gerekçe; varsayılan `admin` değiştirilir |
| `--ulimit nofile/nproc` | Elasticsearch "max file descriptors" hatasını önler |

Betik **idempotent**: makine açılışında `make sonar-kur` demek yeterli —
mevcut konteyneri başlatır, token'ı korur, yeniden kurmaz.

## Tarama

```bash
make sonar-tara     # kapsamı üret + tara
make sonar-durum    # kapı + açık bulgular
```

`sonar-tara.sh` önce `kapsam-uret.sh` çağırır: bayat `coverage.out` yanlış
rapor üretir.

## Kapsam dışı bırakılanlar (ve neden)

`sonar-project.properties` içinde gerekçesiyle yazılı. Özet:

| Hariç | Neden |
|---|---|
| `rust/**/target/**` | Derleme çıktısı, yüz binlerce dosya |
| `go/internal/webui/templkit/*_templ.go` | `templ` tarafından **üretilir** |
| `node_modules`, `dist`, `vendor` | Bağımlılık ağacı |
| `.runtime/`, `.git/` | Veri ve geçmiş |
| **`.html` (Web eklentisi)** | Aşağıdaki bölüm |

### HTML analizi neden kapalı — 152 yanlış pozitif

Depoda 42 `.html` var; **40'ı Go html/template** (`{{define "..."}}`),
yalnız 2'si gerçek HTML (biri test fixture, biri `.runtime/` çıktısı).

Sonar Web eklentisi bunları HTML sanıp şunları üretti:

| Bulgu | Adet | Gerçek |
|---|---|---|
| "Add a `<title>` tag" | 73 | `<title>` `public_head` **partial**'ında |
| "Input without label" | 41 | Nitelikler `{{...}}` ifadesiyle geliyor |
| "Duplicate id" | 38 | Her `{{define}}` **ayrı sayfa**; `id="main"` her birinde bir kez |

Toplam **152 yanlış pozitif** — açık bulguların %88'i. Go template'i HTML
olarak ayrıştırmak yapısal olarak imkânsız; doğru çözüm türü yanlış
tanımamak: `sonar.html.file.suffixes=.xhtml`.

Erişilebilirlik garantisi kaybolmadı: yeni tasarım `cd_*` dosyaları
`templkit` testleriyle (başlık hiyerarşisi, `aria-*`, dokunma hedefi)
korunuyor.

## Kalite kapısı — "bitcosar"

Varsayılan **"Sonar way" kullanılmıyor.** O kapı yeni kod için %80 kapsam
ister; bu proje %23,5'te (DB/HTTP gerektiren paketler birim testinde
koşmuyor). Kalıcı kırmızı kapı, theblock'ta yaşandığı gibi "kimse bakmıyor"
durumuna yol açar — `docs/guvenlik-hatti.md`'deki yamasız CVE kararıyla
aynı mantık.

`deploy/scripts/sonar-kapi-kur.sh` şu koşulları kurar:

| Koşul | Eşik | Neden |
|---|---|---|
| `new_bugs` | > 0 | Yeni kod hatalı gelmesin |
| `new_vulnerabilities` | > 0 | Yeni açık gelmesin |
| `new_code_smells` | > 0 | Yeni borç gelmesin |
| `new_coverage` | < 7 | **Çıt (ratchet)**: bugünkü 7,8'in hemen altı |
| `new_duplicated_lines_density` | > 3 | Kopyala-yapıştır sınırı |
| `bugs` / `vulnerabilities` | > 0 | Mevcut 0 korunsun |
| `security_rating` / `reliability_rating` | > 1 | A derecesi korunsun |

`new_coverage` eşiği **hedef değil çıt**: test eklendikçe yükseltilir
(7 → 10 → 20 → 40). Bugünkü değerin üstünde bir eşik koymak kapıyı
anlamsız kılar; altında koymak düşüşü engeller.

### Bu Sonar sürümünün API tuzağı

`api/qualitygates/list` ve `show` yanıtlarında kapı **kimliği (id) yok** —
bu yüzden `api/qualitygates/select` (gateId ister) kullanılamıyor. Tek
projeli kurulumda doğru yol `set_as_default`. Çok projeli bir kuruluma
geçilirse `select` için id başka bir uçtan alınmalı.

Koşullar da metrik adıyla silinemiyor ("id parameter is missing"); her
koşulun bir UUID'si var. `sonar-kapi-kur.sh` bu yüzden önce mevcut
koşulları okur, sonra `update_condition` (id ile) veya `create_condition`
kullanır — betik idempotent.

## İlk taramanın sonucu (10 Kasım 2026)

| Ölçü | İlk | Sonra |
|---|---|---|
| Güvenlik açığı | 13 | **0** |
| Hata | 172 | **0** |
| Kod kokusu | 42 | **0** |
| Güvenilirlik derecesi | E (5.0) | **A (1.0)** |
| Güvenlik derecesi | C (3.0) | **A (1.0)** |
| Borç | 189 dk | **0 dk** |
| Kapsam | %0 | **%23,5** |
| ncloc | 114.426 | 104.615 |

### Düzeltilen gerçek bulgular

**Güvenlik (12 k8s + 1 JS):**

1. `automountServiceAccountToken: false` — 6 pod'a eklendi. Pod'lar
   Kubernetes API'sini kullanmıyor; token mount edilirse ele geçirilen bir
   container kümeye konuşabilir.
2. `ephemeral-storage` limit + request — 9 container'a eklendi. **Yedek
   Job'ları 8Gi** aldı (DB dump'ı büyük), site 4Gi, mikroservisler 1Gi,
   postgres/redis küçük. Request eklemek scheduler'ın düğüm diskini önceden
   hesaba katmasını sağlar.
3. `Math.random` → sayaç + `crypto.randomUUID` (editor.js). Öneri kimliği
   gizlilik değeri taşımıyor **ama** zayıf PRNG'yi "kimlik" gerekçesiyle
   kod tabanında bırakmak, bir gün yetki jetonu için kopyalanmasının önünü
   açar.

**Hatalar (16 JS):**

4. Yakalanmayan Promise'ler (`S9383`). Hepsi gerçek kullanıcı etkisi
   taşıyordu:
   - `cd-daha.js` — zincirin sonundaki `.then` reddi yakalamıyordu
     (`.finally`'ye çevrildi)
   - `manset.js` — medya yükleme sessizce başarısız oluyordu; **dışa
     aktarma düğmesi ağ hatasında "…" (devre dışı) kalıyordu**, kullanıcı
     bir daha deneyemiyordu
   - `manset.js` — AI isteği ve şablon silme sessiz kalıyordu
   - `tabcats.js` — sıfırlama POST'u 500 dönerse düğme hiçbir şey
     yapmıyormuş gibi görünüyordu
   - `editor.js` — `fetchMeta`, `runSearch`, `uploadEditorFiles`, not
     ekleme/çözme/silme; ayrıca **`init()`**: editör kurulumu patlarsa
     sayfa "hazır" görünürken editör çalışmıyordu

**Kod kokuları (42 JS/Python/CSS):**

5. `parseInt`/`parseFloat` → `Number.parseInt`/`Number.parseFloat`
   (42 yer) — gizli global bağımlılığı kaldırır.
6. `ikon-uret.py` — `[ \t]*\n[ \t]*` belirsizdi (`\s` yeni satırı da
   eşler, geri izleme patlar) → `\s+` tek geçiş.
7. `dictation.js` — `/[ \t]+$/` → düz döngü (süper-doğrusal geri izleme
   riski).
8. `pano.css` — SVG 2 `r:` geometri özelliğine `NOSONAR` + gerekçe.

### Ek olarak bulunan (Sonar dışı)

9. **Depo kökünde 24 adet `C:\Users\...` dizini (108 MB)** — Lighthouse
   Windows yolunu `TMPDIR` sanıp açmış. Git izlemiyordu ama disk yiyordu
   ve taramayı kırıyordu (JS analizcisi `opendir` hatası verdi).
   `.gitignore`'a `C:*` eklendi.

## Komut özeti

```bash
make sonar-kur       # ilk kurulum / makine açılışı
make sonar-tara      # kapsam + tarama
make sonar-durum     # kapı + bulgular
```

Pano: <http://127.0.0.1:9100/dashboard?id=bitcosar>
