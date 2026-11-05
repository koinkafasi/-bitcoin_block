# bitcosar

Yeni proje. Depo: <https://github.com/koinkafasi/-bitcoin_block>

Yerel geliştirme WSL (Ubuntu) üzerinde koşar; `demo.bitcosar.com` Cloudflare
Tunnel ile bu makinedeki yerel servise bağlanır.

## Durum

| Bileşen | Durum |
|---|---|
| Depo + dal (`main`) | ✅ |
| Güvenlik hattı (CI + yerel tek kapı) | ✅ |
| `demo.bitcosar.com` → WSL | ✅ (Cloudflare Tunnel `bitcosar-demo`) |
| Uygulama kodu | ⏳ talimat bekliyor |

## Yerel servis ve alan adı

```
Internet → Cloudflare (demo.bitcosar.com)
         → Tunnel "bitcosar-demo"
         → http://localhost:8090  (WSL)
```

| Ne | Nerede |
|---|---|
| Yerel demo portu | `8090` |
| Tünel kimliği | `~/.config/bitcosar/tunnel_id` |
| Tünel jetonu (0600) | `~/.config/bitcosar/cloudflared-token` |
| Diğer sırlar (0600) | `~/.config/bitcosar/secrets.env` |
| systemd birimleri | `~/.config/systemd/user/bitcosar-{demo,tunnel}.service` |

Sırlar **repoda değil**, `~/.config/bitcosar/` altında ve `0600` izinle durur.

```bash
systemctl --user status bitcosar-demo bitcosar-tunnel
systemctl --user restart bitcosar-tunnel
curl -sS https://demo.bitcosar.com/saglik   # → ok
```

`demo.bitcosar.com` DNS'i, `*.bitcosar.com` joker kaydından **daha özgül**
bir CNAME'dir (`<tunnel-id>.cfargotunnel.com`), bu yüzden joker kaydı ezer.

## Güvenlik hattı

Tek kaynak: `deploy/scripts/guvenlik-tara.sh`. CI da yerel de **aynı** betiği
koşar; "CI'da kırıldı, yerelde geçiyordu" durumu olmaz.

```bash
make guvenlik-tara          # kapı: kırmızıysa çıkış kodu 1
make guvenlik-rapor         # yalnız rapor, kırmaz
```

| Kapı | Araç | Eşik |
|---|---|---|
| Bağımlılık CVE | Trivy fs | CRITICAL+HIGH, yaması olan |
| Yapılandırma | Trivy config | CRITICAL+HIGH |
| Gömülü sır | Trivy secret | tümü |
| SAST | Semgrep | ERROR |
| Go SAST | gosec | high (+ gerekçeli `#nosec`) |
| Go CVE | govulncheck | tümü |
| Workflow güvenliği | zizmor | MEDIUM+ |

Yığın henüz kesinleşmediği için betik **otomatik algılar**: `go.mod`,
`package-lock.json`, `Cargo.toml` varsa ilgili araç koşar; yoksa o adım
"atlandı" der (sessizce geçmez, açıkça yazar).

Ayrıntı: [`docs/guvenlik-hatti.md`](docs/guvenlik-hatti.md)
