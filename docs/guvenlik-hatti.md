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
