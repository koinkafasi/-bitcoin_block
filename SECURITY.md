# Güvenlik Politikası

bitcosar (depo: `koinkafasi/-bitcoin_block`) bir haber portalıdır. Bu belge
güvenlik açığını **nasıl bildireceğinizi** ve bildirimin nasıl işleneceğini
tanımlar.

## Açık bildirimi

**Lütfen herkese açık issue açmayın.** Açık, düzeltme yayınlanmadan önce
sömürülebilir hâle gelir.

- Tercih edilen yol: deponun **Security → Report a vulnerability** düğmesi
  (GitHub özel güvenlik danışmanlığı). Bildirim gizli kalır, düzeltme aynı
  danışmanlık altında yürür.
- Alternatif: `guvenlik@bitcosar.com` adresine e-posta.

Bildirime ekleyin:

1. Etkilenen adres/özellik ve sürüm (commit SHA'sı yeterli).
2. Yeniden üretme adımları (istek/yanıt, curl komutu, ekran görüntüsü).
3. Etki: hangi veriye erişim, hangi yetki, hangi kullanıcı.
4. Varsa önerdiğiniz düzeltme.

**Sırları bildirime yazmayın.** Sızmış bir anahtar bulduysanız yalnız
"nerede ve ne tür" olduğunu söyleyin; değeri göndermeyin.

## Kapsam

Kapsam içi:

- `go/` altındaki sunucu kodu, şablonlar ve istemci betikleri
- `deploy/` altındaki Kubernetes manifestleri, betikler ve CI iş akışları
- Kimlik doğrulama, oturum, yetkilendirme, ödeme (Stripe) ve medya yükleme
  yolları

Kapsam dışı:

- Üçüncü taraf hizmetlerin (CoinGecko, Binance, Stripe, SMTP, AI sağlayıcı)
  kendi açıkları — onlara bildirin.
- Yalnız `bitcosar.com` dışındaki ortamlar (demo tüneli) üzerinden erişilen
  ve üretimde kapalı olan hata ayıklama yolları.
- Otomatik tarayıcı çıktıları (Trivy/gosec/Semgrep) — bunlar zaten CI'da
  kapı; yeni bir şey söylemiyorsa issue açmayın.

## Yanıt süresi

| Aşama | Hedef |
|---|---|
| Alındı bildirimi | 3 iş günü |
| İlk değerlendirme (etki + önem) | 10 iş günü |
| Düzeltme / azaltma | Öneme göre: kritik 7 gün, yüksek 30 gün, diğer 90 gün |

Süre taşarsa gerekçesiyle birlikte haber verilir; sessiz kalmak yok.

## Bu depoda zaten koşan hatlar

Bildirim yapmadan önce bakmanız işinizi hızlandırır:

| Hat | Nerede | Ne yapar |
|---|---|---|
| `ci.yml` → güvenlik | her push/PR | Trivy (fs/config/secret), govulncheck, gosec, Semgrep, zizmor, npm/cargo audit |
| `ci.yml` → uygulama | her push/PR | gofmt, vet, test, derleme |
| `kalite.yml` | her push/PR | golangci-lint (yeni kod kapısı), kapsam çıtı |
| `codeql.yml` | push/PR/haftalık | CodeQL `security-and-quality` (Go + JS) |
| `imaj.yml` | `main` | imaj derle → Trivy kapısı → CycloneDX SBOM → cosign keyless imza |
| `scorecard.yml` | haftalık | OpenSSF Scorecard (tedarik zinciri sağlığı) |
| `sonar` | yerel | SonarQube kalite kapısı (`make sonar-durum`) |

Küme tarafında Kyverno, yalnız **imzalı ve digest'i doğrulanmış** imajların
pod olarak kalkmasına izin verir (`deploy/k8s/05-kyverno.yaml`).

## Sırlar

Sırlar repoya yazılmaz; `deploy/.env`'den üretilen Kubernetes Secret'larında
tutulur. Sızmış bir anahtar fark edildiğinde öncelik **anahtarı döndürmektir**
(geçmişi temizlemek yetmez: kopyalar alınmış olabilir).

## Teşekkür

Geçerli bir açık bildiren ve düzeltmeyi beklemeden yayınlamayan herkese
teşekkür ederiz; isterseniz danışmanlık kaydında adınız anılır.
