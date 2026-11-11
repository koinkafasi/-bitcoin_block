# Tedarik zinciri ve kalite hattı

Bu belge **11 Kasım 2026** turunda kurulan hatları ve gerekçelerini anlatır.
Kural: burada yazan bir kapı, `make` hedefi ya da CI işi olarak gerçekten
koşmuyorsa belge yanlıştır — öyle bir satır bulursanız düzeltin.

---

## 1. Hatların haritası

```
kod yazıldı
   │
   ├─ ci.yml / guvenlik    Trivy(fs,config,secret) · govulncheck · gosec
   │                       Semgrep · zizmor · npm audit · cargo audit
   ├─ ci.yml / uygulama    gofmt · vet · test · derleme
   ├─ kalite.yml / lint    golangci-lint — YENİ KOD kapısı
   ├─ kalite.yml / kapsam  test + kapsam çıtı (%16,5)
   ├─ codeql.yml           CodeQL security-and-quality (Go, JS)
   │
   ▼
main'e birleşti
   │
   ├─ imaj.yml             derle → Trivy imaj kapısı → CycloneDX SBOM
   │                       → cosign keyless imza → GHCR (digest)
   │                       → dogrula işi: imzayı GERÇEKTEN doğrular
   │
   ▼
küme
   ├─ Kyverno              imza zorunlu · :latest yasak · pod sertleştirme
   ├─ NetworkPolicy        varsayılan reddet · DNS istisnası · segmentasyon
   ├─ ServiceAccount       token mount YOK · RBAC YOK
   └─ PDB                  drain sırasında neyin tahliye edilebileceği
```

Haftalık: `scorecard.yml` (depo sağlığı), `codeql.yml` (yeni kurallar),
`renovate` (bağımlılık PR'ları).

---

## 2. Neden bunlar AYRI hatlar

Her araç farklı soru sorar; birini diğerinin yerine koymak sessiz boşluk
üretir:

| Araç | Sorduğu soru | Diğerinin kaçırdığı |
|---|---|---|
| Trivy (fs/config/secret) | Bilinen CVE, yanlış yapılandırma, repoda sır var mı? | Kod akışını görmez |
| govulncheck | Bu CVE'ye **giden kod yolu** var mı? | CVE dışı desenleri görmez |
| gosec | Güvenlik deseni (SQL, zayıf kripto, taint) | Ölü kod, hata akışı |
| Semgrep | Dile gevşek bağlı desen | Tip bilgisi yok |
| golangci-lint | Yakalanmayan hata, ölü kod, kullanılmayan atama | Zafiyet sınıflandırması yok |
| CodeQL | Derin veri akışı (taint tracking) | Hızlı geri bildirim değil |
| Sonar | Kod kokusu + kapsam + tekrar | Güvenlik CVE'si değil |
| Scorecard | Depoyu güvenli geliştiriyor musun? | Kodun kendisi |

`guvenlik-tara.sh` ile `lint.sh` bu yüzden **ayrı** dosyalar: birinin yeşili
diğerini geçersiz kılmaz.

---

## 3. golangci-lint: neden "yeni kod" kapısı

İlk tam tarama (11 Kasım 2026) **236 bulgu** verdi: 50 ölü bildirim,
38 yakalanmayan hata, 37 gosec, 48 yazım (Türkçe kelimeler), 27 staticcheck,
11 nilerr, 8 errorlint, 8 kullanılmayan atama.

Yapılanlar:

- **Mekanik olanların tamamı düzeltildi**: kullanılmayan atamalar → `var`,
  gereksiz dönüşümler, etkisiz atamalar, boşluklar, goimports, `%v` → `%w`,
  `==` → `errors.Is`, `defer X()` → `defer func(){ _ = X() }()`.
- **50 ölü bildirim silindi.** Silmeden önce her biri grep ile doğrulandı.
  İki yerde aşırı silme oldu (`hisseHizSecenekleri` gövdesi,
  `YayinKaydi` tipi) — `go build` yakaladı, geri getirildi. Ders: toplu
  silme **derleme + test** olmadan bitmiş sayılmaz.
- **Türkçe kelimeler** `extra-words` ile serbest bırakıldı: `performans`,
  `paralel`, `adres`, `personel`, `EUROPEN` (borsa kaydı böyle; `EUROPEAN`
  yapmak **veriyi bozar**).
- **`nilerr`** 11 yerde bilinçli "bulunamadı → varsayılan" davranışı:
  her birine tek satır gerekçeli `//nolint:nilerr` yazıldı. Bunlar gerçek
  borç: `GetLayout`, `GetContentExtra`, `GetTrendSources` gibi okumalar
  "kayıt yok" ile "okuma hatası"nı ayırmıyor (ileride `ErrNotFound`).
- **`ST1005`** kapalı: hata metinleri Türkçe ve büyük harfle başlıyor.
- **`SA1019`** kapalı: `webpush` hâlâ `crypto/elliptic`in eski API'sini
  kullanıyor; yeni API'ye geçiş ayrı iş (bilinen borç).
- **`gosec` eşiği** `guvenlik-tara.sh` ile aynı: yalnız yüksek önem + orta
  güven. Farklı eşik, o kapıdan geçen kodun burada kırılması demekti.

Kalan bulgu: yalnız 2 (`canli_hub.go` içinde henüz bağlanmamış sayaç
yardımcıları — canlı piyasa işi sürüyor).

**Kapı neden yeni kodla sınırlı:** tam depoyu kapı yapmak 236 bulguyla
kırmızı başlar ve "kapıyı kapat" kararına götürür. Ters yönde taviz yok:
yeni eklenen her satır temiz olmalı. `main` üzerinde tam depo **raporlanır**
(iş özeti + artifact), bloklamaz. Borç büyürse rapor büyür — görünür kalır.

---

## 4. İmaj hattı: imzala, sonra DOĞRULA

`imaj.yml` her `main` push'unda 4 imajı derler:

1. `trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1`
   — yaması olan kritik/yüksek varsa imaj GHCR'a **gitmez**.
   `--ignore-unfixed` bilinçli: yamasız bulgu kapıyı kalıcı kırmızıya çevirir
   ve kimse bakmaz.
2. `trivy image --format cyclonedx` → SBOM artifact (90 gün).
3. `cosign sign` — **keyless**: uzun ömürlü özel anahtar yok, kimlik GitHub
   OIDC jetonundan, imza Rekor şeffaflık günlüğünde.
4. `cosign attest --type cyclonedx` — SBOM imzaya **bağlanır**. Ayrı iki
   dosya olsaydı SBOM değiştirilip imza geçerli kalırdı.
5. `dogrula` işi: 4 imaj için `cosign verify` + `verify-attestation` koşar.
   "İmzaladık" demek yetmez; doğrulanmış olması gerekir.

Yerelde sevk öncesi aynı doğrulama: `make tedarik-dogrula`.

### Kyverno: imza iddiasını kümede zorunlu kılar

İmza, küme **kontrol etmedikçe** dekorasyondur. `deploy/k8s/05-kyverno.yaml`:

| Politika | Etki |
|---|---|
| `bitcosar-imza-zorunlu` | `ghcr.io/koinkafasi/theblock-*` imzasız/digest'sizse pod **reddedilir** |
| `bitcosar-latest-yasak` | `:latest` yasak (initContainer dahil) |
| `bitcosar-pod-sertlestirme` | root yasak, `allowPrivilegeEscalation: false` |

Kurulum (bir kez):

```bash
helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update
helm install kyverno kyverno/kyverno -n kyverno --create-namespace \
  --set replicas=1 --version 3.4.4
kubectl apply -f deploy/k8s/05-kyverno.yaml
```

`k8s-sevk.sh` bu adımı **atlamaz**: CRD yoksa uyarı basar ve sevki
durdurmaz (küme Kyverno'suz da çalışabilmeli), ama uyarı görünür kalır.

---

## 5. Ağ: varsayılan reddet

`deploy/k8s/06-networkpolicy.yaml` önce **her şeyi kapatır**, sonra yalnız
gereken akışı açar. En sık tuzak DNS: `Egress` seçildiğinde DNS de kapanır
ve her şey "name resolution failed" verir — bu yüzden DNS ayrı ve ilk kural.

Dış çıkış `site` için açık (CoinGecko, Binance, RSS, SMTP, AI) ama `except`
ile özel ağlar **ve** `169.254.169.254` (bulut metadata kimliği — SSRF'in en
değerli hedefi) kapalı. IP listesi yazmak kırılgan: CDN IP'leri değişir.

---

## 6. Postgres TLS: `sslmode=verify-full`

Önce: `sslmode=disable`. Aynı düğümde bile ağ katmanı açıktı; yedek trafiği
(`pg_dump`) düz metindi.

Şimdi:

- `deploy/scripts/pg-tls-uret.sh` CA + sunucu sertifikasını üretir
  (SAN'lar `theblock-postgres`, `.theblock`, `.svc`, `.svc.cluster.local`,
  `localhost` — `verify-full` CN'e değil SAN'a bakar), Secret'a yazar.
- Postgres yalnız `hostssl` kabul eder; `hostnossl` satırları **reject**
  (`pg_hba.conf` Secret'ın içinde).
- `site` ve yedek `pg_dump` → `PGSSLMODE=verify-full` + `sslrootcert`.
- Secret yoksa pod **başlamaz** (fail closed): şifresiz bağlanmaktansa
  hiç bağlanmamak doğru.

Rotasyon: `bash deploy/scripts/pg-tls-uret.sh --yenile` → postgres, sonra
site rollout (sıra önemli: ikisi aynı CA'yı görmeli).

---

## 7. Kimlik: ServiceAccount + PDB

- 7 adlandırılmış ServiceAccount; hiçbirine **RBAC bağı yok** ve
  `automountServiceAccountToken: false`. Pod'lar Kubernetes API'sini
  kullanmıyor; yetki vermemek, yetkiyi yönetmekten güvenli.
- Adlandırılmış SA denetim kaydında pod kimliğini görünür kılar.
- PDB: `site` ve mikroservisler `minAvailable: 0` (tek replika — gerçek
  koruma replika artırılınca başlar, bu yüzden yorumda yazılı), postgres/
  redis `maxUnavailable: 1` (RWO disk + tek replika: kesinti kaçınılmaz).

---

## 8. Renovate: bağımlılık güncellemesi

`renovate.json` — GitHub Actions SHA'sı ile sabitli olduğu için Renovate
SHA'yı **ve** yanındaki `# vX.Y.Z` yorumunu birlikte günceller (yoksa yorum
yalan söyler). Güvenlik düzeltmeleri zamanlamayı atlar. Go'da yalnız yama
otomatik; minor/major insan incelemesi ister.

Not: aynı işi `deploy/scripts/action-sha-guncelle.sh` yerelde yapar (gh ile);
Renovate kurulana kadar o kullanılıyordu, sonrasında da yedek yol olarak kalır.

---

## 9. Dal koruması ve depo ayarları

`make dal-koruma` (göstermek) / `bash deploy/scripts/dal-koruma.sh --uygula`:

- `main`'e doğrudan push yok, kuvvetli push yok, dal silme yok.
- Birleşme için 4 kontrol yeşil olmalı: `guvenlik kapıları`, `derleme ve
  test`, `golangci-lint (yeni kod)`, `kapsam çıtı`.
- Yöneticiler de kurala tabi (`enforce_admins=true`) — kendi kuralını
  atlayabilen bir yönetici, kuralın olmadığı anlamına gelir.
- Gizli tarama + push koruması + Dependabot güvenlik güncellemeleri açık.

Depo **public** olduğu için CodeQL ve secret scanning ücretsizdir.

---

## 10. Bilinen borç (kapatılacak)

| Borç | Neden ertelendi |
|---|---|
| `webpush` eski `crypto/elliptic` API'si (SA1019) | VAPID anahtar üretimi/doğrulaması yeniden yazılmalı |
| `GetLayout`/`GetContentExtra`/`GetTrendSources` "yok" ile "hata"yı ayırmıyor | Çağıranların hepsi güncellenmeli (`ErrNotFound`) |
| `canli_hub.go` 2 kullanılmayan sayaç yardımcısı | Canlı piyasa işi sürüyor |
| SonarQube yalnız yerelde | CI'dan erişilemez; SonarCloud token'ı gerekir |
| E2E testleri CI'da değil | Puppeteer + gerçek tarayıcı; runner'a Chrome kurulumu gerekir |
| GitOps (Argo CD/Flux) yok | Sevk elle: `k8s-sevk.sh` |

---

## 11. Ölçüm (11 Kasım 2026)

| Ölçüm | Değer |
|---|---|
| `make guvenlik-tara` | 6/6 kapı yeşil |
| `golangci-lint` tam depo | 236 → **2** bulgu |
| `golangci-lint` yeni kod | **0** |
| Go testleri | tümü geçti |
| Kapsam | %16,9 (çıt %16,5) |
| SonarQube kalite kapısı | OK — 0 hata, 0 açık, 0 koku, güvenlik A |
| zizmor | 0 bulgu (5 workflow) |
| Action SHA doğrulaması | 23/23 geçerli |
