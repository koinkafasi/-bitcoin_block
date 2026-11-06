# OpenCode yapılandırması — yetenekler ve gerekçeleri

6 Kasım 2026. Hepsi kurulup **doğrulandı**; doğrulama yöntemi her başlıkta
yazılı. Tarayıcı yeteneğinin ayrıntısı ayrı belgede:
`OPENCODE-TARAYICI.md`.

Aktif yapılandırma: `C:\Users\alper\.config\opencode\opencode.json`
(WSL'den çağrılsa bile OpenCode bunu okuyor — ikili Windows tarafında).

## Maliyet modeli — kararların dayanağı

`opencode stats`: 14,1 milyar token / 46,7k adım → **adım başına ~302k
token** context. Buna göre:

| | Şema | Payı |
|---|---|---|
| Yalın MCP (browser, 25 araç) | ~5.265 token | %1,7 |
| Context7 (2 araç) | ~1.304 token | %0,4 |
| ruflo (500+ araç) | ~607k token | **%200 — eski yavaşlığın sebebi** |

Skill'ler çağrılana kadar bedava; ajanlar araç setini **daraltır**, yani
maliyeti düşürür. O yüzden ağırlık skill ve ajanlarda, MCP'de değil.

## MCP sunucuları

| Ad | Araç | Durum | Ne için |
|---|---|---|---|
| `browser` | 25 | açık | Playwright; gerçek Chrome, ölçüm, mobil, koyu tema |
| `docs` | 2 | açık | Context7; güncel kütüphane dokümanı |
| `ruflo` | 500+ | **kapalı** | yavaşlığın kök sebebi; açılmamalı |
| `rea` | 170 | kapalı | tersine mühendislik; gerektiğinde açılır |

Doğrulama: `opencode mcp list` → `✓ browser connected`, `✓ docs connected`.

## Formatter — gofmt köprüsü

```
formatter.gofmt-wsl  →  powershell -NoProfile -ExecutionPolicy Bypass
                        -File .config/opencode/bin/gofmt-wsl.ps1 $FILE
formatter.gofmt      →  disabled (yerleşik; Windows'ta gofmt yok)
```

**Niye köprü:** OpenCode Windows'ta koşuyor, dosyalara `\\wsl$\Ubuntu\...`
UNC yoluyla erişiyor; Go araç takımı yalnız WSL'de. Yerleşik `gofmt`
formatter'ı Windows PATH'inde `gofmt` aradığı için hiç çalışmıyordu.

**Niye önemli:** CI'ın **ilk** kapısı `test -z "$(gofmt -l .)"`. Biçim
ihlali derlemeye hiç gelmeden kırar; adım 14 saniyede düşer. Bu depoda
CI bir gün boyunca tam bundan kırmızıydı (`etiket.go` dosya sonu boş
satırı, `rss_sanitize.go` struct etiket hizası).

**Niye Windows'a Go kurmadım:** CI, WSL'deki go 1.27.2'yi kullanıyor.
İkinci bir toolchain sürüm farkı riski doğurur ve kapı birebir biçim
istiyor. Köprü CI ile **aynı ikiliyi** çağırır.

Betiğin biçimini belirleyen dört ders (hepsi yaşandı, yorumlarda yazılı):
mutlak yol (PATH'e güvenmek sessizce başarısız oldu), tek satır komut
(çok satırlı `bash -c` ayrıştırma hatası verdi), toleranslı ters bölü
(aktarımda biri yeniyor), konumsal ve `Mandatory` olmayan parametre
(bağlanamayınca PowerShell konsolsuz ortamda istem açıp çıkış 1 veriyordu).

**Doğrulama:** bozuk bir `.go` dosyası OpenCode'a düzenletildi. Köprüden
önce dosya bozuk kalıyordu; sonra OpenCode'un kendi raporu:
*"the edit tool's format-on-edit hook kept reformatting the file"*.

## LSP — BİLİNÇLİ OLARAK KURULMADI

Yerleşik `gopls` Windows'ta `go` komutu arar; Windows'ta Go yok. WSL'deki
gopls'i `wsl.exe` ile başlatmak da çözmez: LSP protokolü dosya URI'si
taşır, gopls'e `\\wsl$\...` URI'si gider ve çözemez. URI çeviren bir
vekil gerekir.

Düzgün çözüm iki tane, ikisi de karar gerektiriyor:

1. **WSL içine native OpenCode kur** — her şey POSIX olur, formatter
   köprüsü de gerekmez. Bedeli: ikinci kurulum, kendi oturum geçmişi.
2. **Windows'a Go kur** — yerleşik `lsp: true` çalışır. Bedeli: ikinci
   toolchain, ve gopls'in UNC üzerinden modül indekslemesi yavaş/kırılgan.

Yanlış yapılandırıp "kurdum" demek yerine boş bırakıldı.

## İzin kuralları

**Kritik ayrım:** `opencode run --auto` "açıkça reddedilmeyen" izinleri
otomatik onaylar. Yani `--auto` altında **`ask` korumaz**; gerçek koruma
`deny` ile olur. Liste buna göre iki kademeli.

`deny` (8) — geri dönüşü yok:
`git push --force*` · `git push -f*` · `git filter-branch*` ·
`rm -rf /*` · `rm -rf ~*` · `rm -fr /*` · `dropdb*` ·
`psql*DROP DATABASE*`

`ask` (7) — etkileşimli kullanımda sorulsun:
`kubectl apply*` · `kubectl delete*` · `kubectl rollout undo*` ·
`talosctl*` · `git reset --hard*` · `git rebase*` ·
`git push*--force-with-lease*`

**Doğrulama:** atılabilir bir depoda `git push --force` denendi.
OpenCode'un raporu: *"The shell tool itself was blocked by a permission
rule before `git` was ever invoked"* — ve etrafından dolaşmayı reddetti.

`kubectl apply` neden listede: `deploy/k8s/40-site.yaml` bir şablon,
`${IMAJ_ETIKETI}` taşıyor; doğrudan apply edilince `InvalidImageName` ile
pod kalkmıyor. Bu bir kez yapıldı ve **10 dakikalık üretim kesintisi**
oldu. Doğru yol `deploy/scripts/k8s-sevk.sh`.

## Ajanlar

`~/.config/opencode/agents/*.md` — dosya adı ajan adı. Hepsi `subagent`.

| Ajan | Yetki | İş |
|---|---|---|
| `tasarim` | salt okunur + `browser_*` | Canlıyı ölç, tasarım spec'i ile farkı raporla |
| `inceleme` | salt okunur | Kod incelemesi; bulguyu dosya:satır ile bildirir, düzeltmez |
| `dagitim` | `kubectl get/logs/rollout status`, `curl`; **`apply`/`delete` kapalı**, `deploy/k8s/*` düzenlemesi kapalı | Sevk betiğiyle dağıtım ve canlı doğrulama |
| `guvenlik` | salt okunur + `make guvenlik-*` | Kapıları koştur, bulguyu bildir; sırrın değerini yazmaz |

Ajan araç setini daralttığı için **maliyeti düşürür** ve yanlış araç
seçimini engeller.

Doğrulama: `opencode debug agents` → dördü de `mode=subagent`.

## Komutlar

| Komut | Ne yapar |
|---|---|
| `/dagit` | Sevk betiğiyle dağıtım; manifesti doğrudan apply etmeme uyarısıyla |
| `/guvenlik-tara` | `make guvenlik-tara`, bulguları dosya:satır ile |
| `/tasarim-fark` | Tarayıcıyla ölç, `COINDESK-TASARIM-SPEC.md` ile karşılaştır |
| `/ci-durum` | Son CI koşuları; ilk kırılan adımı bul (önce gofmt kapısına bak) |

## Skill'ler

`~/.agents/skills/` — çağrılana kadar bedava.

| Skill | İçerik |
|---|---|
| `browser-measure` | Ölçüm disiplini: hesaplanmış stili ölç, toplu tara, DPR'yi hesaba kat, `browser_resize` ile mobil, `browser_emulate_media` ile koyu tema, ölçmediğini söyle, araç depoya yazmasın |
| `bitcosar-is-akisi` | Derleme/doğrulama/commit akışı ve beş tuzak: `templ generate` kökten, üretilmiş dosyalar commit edilir, çalışma ağacı yeşil ≠ HEAD yeşil (worktree doğrulaması), araç depoya yazmasın, üretim manifesti şablondur |

İkisi de genel tavsiye değil: içlerinde bu depoda **gerçekten yapılmış**
hatalar ve ölçülen karşılıkları var.

Ajanlarla örtüşen skill yazılmadı (dağıtım/güvenlik/tasarım bilgisi ajan
yönergesinde duruyor); aynı bilgiyi iki yere koymak bakım yükü olurdu.

## Diğer

- `small_model`: `omniroute/auto/best-coding-fast` — başlık/özet üretimi
  524k bağlamlı modele gitmiyor. Önceden ana modelle aynıydı.
- `websearch`: `{"provider": "random"}` — önceden yapılandırılmamıştı.
- `instructions`: proje belgeleri ortam talimatı
  (`COINDESK-TASARIM-SPEC.md`, `COINDESK-SPEC.md`, `guvenlik-hatti.md`,
  `OPENCODE-TARAYICI.md`, bu belge).

## Yedekler

Her değişiklikten önce alındı:
`~/.config/opencode/opencode.json.yedek-<zaman>`

## Açık kalan

- **LSP** — yukarıdaki iki seçenekten biri seçilmeli.
- **OmniRoute anahtarı döndürülmeli.** WSL'deki eski yapılandırma düz
  metin anahtar taşıyordu; dosya düzeltildi ama anahtar bir süre diskte
  durdu.
