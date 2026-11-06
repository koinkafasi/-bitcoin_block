# Güvenlik hattı (bitcosar)

Tek kural: **kapı tek yerde tanımlıdır.** `deploy/scripts/guvenlik-tara.sh`
hem CI'da hem yerelde koşar. Ayrı bir "CI sürümü" yoktur; olsaydı ikisi
zamanla ayrışır ve "CI'da kırıldı, yerelde geçiyordu" başlardı.

```
make guvenlik-tara     # kapı — kırmızıysa exit 1
make guvenlik-rapor    # yalnız rapor
make guvenlik-imaj     # + Docker imaj taraması
```

## Kapılar

| # | Kapı | Araç | Eşik | Not |
|---|---|---|---|---|
| 1 | Bağımlılık CVE | Trivy fs | CRITICAL+HIGH, **yaması olan** | Yamasızlar kapı değil: düzeltilemeyen bir bulgu CI'yı kalıcı kırmızıya çevirir ve kimse bakmaz |
| 2 | Yapılandırma | Trivy config | CRITICAL+HIGH | Dockerfile, k8s manifest |
| 3 | Gömülü sır | Trivy secret | tümü | Anahtar/jeton commit'i |
| 4 | SAST | Semgrep | **ERROR** | WARNING raporlanır, kapı değil |
| 5 | Go SAST | gosec | **high** + gerekçeli `#nosec` | `-nosec-require-rules -nosec-require-justification` |
| 5 | Go CVE | govulncheck | tümü | Çağrı grafiği bazlı |
| 5 | Node | npm audit | high | Yalnız üretim bağımlılıkları |
| 5 | Rust | cargo audit | advisory | |
| 6 | Workflow | zizmor | MEDIUM+ | unpinned action, fazla yetki, şablon enjeksiyonu |

## Sürüm sabitleme

Tüm araçlar **sabit sürüm**: Trivy `0.76.1`, Semgrep `1.183.0`,
zizmor `1.32.0`, gosec `v2.30.0`, govulncheck `v1.9.0`.

Neden: `@latest` bir gün yeni kurallarla gelir ve kapıyı habersiz kırar.
Bu, theblock projesinde tam olarak yaşandı (gosec v2.30.0 altı taint kuralı
ekledi, bulgu 0 → 139). Sürüm yükseltmesi bilinçli bir commit olmalıdır.

### Araç KURULUMU da betiğin işi

Sürümü sabitlemek yetmiyor: araç **kurulu değilse** kapı sessizce atlanır.
6 Kasım 2026'da ölçüldü — CI'da `gosec` **kurulu değildi**, betik
"gosec yok — atlandı" deyip geçiyordu. Yani "6 kapı" gerçekte 5 kapıydı ve
hiç kimse fark etmiyordu (koşu yeşil).

Kural: **güvenlik aracı yoksa kapı KIRILIR.** Betik Trivy ve gosec'i sabit
sürümle kendisi kurar; kuramazsa `BASARISIZ=1`.

Sürüm doğrulaması ince bir tuzak taşıyor: `gosec -version` **kaynaktan
derlenen her kurulumda "dev" yazar** (sürüm yalnız resmî sürüm
ikililerine gömülü). Bu yüzden sürüm `go version -m` build info'sundan
okunur:

```bash
go version -m "$GOSEC" | grep -qE "mod[[:space:]]+github.com/securego/gosec/v2[[:space:]]+v2\.30\.0"
```

`-version` ile bakılsaydı kontrol hiç tutmaz ve her koşuda yeniden
kurulum yapılırdı.

## Yol güvenliği — `os.Root` hapishanesi

Dosya sistemine yazan her uç, kökü `os.OpenRoot` ile açar:

```go
kok, err := os.OpenRoot(filepath.Join(platform.StorageDir(), "kaynak"))
b, err := kok.ReadFile(filepath.Join(hash[:2], hash)) // kökten GÖRELİ
```

Neden: ad doğrulaması (32 hex) doğru olsa da güvenlik **tek katmana**
bağlı kalır; doğrulama bir gün gevşetilirse `..` ile depolama dışına
çıkılır. `os.Root` bunu çekirdek tarafından uygulanan yapısal garantiye
çevirir — kod ne yazarsa yazsın kök dışına çıkan işlem hata döner.
`gosec` G703 bulgusu (taint analizi doğrulama fonksiyonunu göremiyor) bu
değişiklikle kapandı.

## gosec istisnaları (`#nosec`)

- Çıplak `#nosec` **susturmaz**; kural kimliği **ve** gerekçe zorunlu.
- Doğru biçim:
  ```go
  // #nosec G703 -- Yol, os.MkdirTemp'in ürettiği geçici dizin + SABİT dosya adı.
  os.WriteFile(dosya, veri, 0o644)
  ```
- Gerekçe "yanlış pozitif" demek yetmez; **neden** güvenli olduğu yazılır.

## Yığın algılama

Proje yığını kesinleşene kadar betik `go.mod` / `package-lock.json` /
`Cargo.toml` varlığına bakar. Adım atlanırsa **açıkça** "ATLANDI" yazar;
sessizce yeşil göstermez. Yığın netleşince ilgisiz adımlar betikten
çıkarılmalıdır.

## CI

`.github/workflows/ci.yml`:

- `permissions: contents: read` — hat hiçbir şey yazmaz.
- Tüm action'lar **SHA ile sabit** (`@v4` değil).
- `persist-credentials: false` — checkout jetonu iş boyunca durmaz.
- `concurrency` ile aynı dalda eski koşu iptal edilir.

## Sırlar

Kod tarafında hiçbir sır yoktur. Yerel sırlar `~/.config/bitcosar/`
altında `0600` izinle durur ve `.gitignore` ile repo dışındadır:

| Dosya | İçerik |
|---|---|
| `secrets.env` | GitHub PAT, Cloudflare API jetonu, hesap/zone kimlikleri |
| `cloudflared-token` | Tünel jetonu (argv'de değil dosyada) |

CI'ya sır **eklenmedi**: hat yalnız okuma yapar, dağıtım yapmaz.
