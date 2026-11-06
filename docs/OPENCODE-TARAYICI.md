# OpenCode tarayıcı yeteneği — kurulum ve kullanım

6 Kasım 2026'da kuruldu ve uçtan uca doğrulandı. OpenCode artık gerçek
Chrome açıp sayfayı okuyabiliyor, ölçebiliyor, tıklayıp yazabiliyor.

## Ne kuruldu

| | |
|---|---|
| Paket | `@playwright/mcp@0.0.84` (global npm) |
| Shim | `C:\Users\alper\AppData\Roaming\npm\playwright-mcp.cmd` |
| Tarayıcı | **kurulu Chrome** (`--browser chrome`) — ~300 MB ek indirme yok |
| Yapılandırma | `C:\Users\alper\.config\opencode\opencode.json` → `mcp.servers.browser` |
| Araç sayısı | **25** (çekirdek), şema ~**5.265 token** |
| Mod | **görünür** (headed) — kullanıcı ne yaptığını görür |

Yapılandırmadaki bayraklar ve gerekçeleri:

```
--browser chrome              kurulu Chrome; indirme yok
--snapshot-boxes              snapshot'ta [box=x,y,w,h] → JS'siz geometri
--viewport-size 1440x900      makul varsayılan
--user-data-dir  %LOCALAPPDATA%\opencode-browser\profil     DEPO DIŞINDA
--output-dir     %LOCALAPPDATA%\opencode-browser\cikti      DEPO DIŞINDA
--ignore-roots                kök sorgusunda takılmayı önler
--timeout-navigation 45000
```

Windows'ta `.cmd` shim doğrudan spawn edilemez; komut
`cmd.exe /c <tam yol>` biçiminde yazıldı.

**Profil ve çıktı dizinleri bilinçli olarak depo dışında.** Daha önce bir
Lighthouse koşusu Windows yolunu `TMPDIR` sanıp depo kökünde
`C:\Users\...` adlı dizinler açmıştı: 40 MB Chrome profili ve **165
dosyası commit edilmiş** durumdaydı (bkz. `1687d92` öncesi). Aynı hata
tekrarlanmasın.

### Yetenek grupları — niye çekirdekte kalındı

Ölçüldü:

| `--caps` | araç | ~token |
|---|---|---|
| **(yok / çekirdek)** | **25** | **5.265** |
| vision | 31 | 6.037 |
| devtools | 38 | 7.304 |
| vision,devtools | 44 | 8.075 |

Çekirdek zaten gerekli her şeyi içeriyor — `browser_console_messages` ve
`browser_network_requests` dahil. `devtools` yalnız video/trace/annotate
ekliyor, `vision` ise koordinatla fare. Gerekirse sonradan eklenebilir.

Bu önemli: OpenCode'un eski yavaşlığının kök sebebi istek başına ~607k
token MCP şemasıydı. 5,3k kabul edilebilir; büyüterek o hataya dönmeyelim.

## Çekirdek 25 araç

```
browser_navigate · browser_navigate_back · browser_snapshot
browser_take_screenshot · browser_click · browser_type · browser_fill_form
browser_press_key · browser_hover · browser_select_option · browser_drag
browser_drop · browser_file_upload · browser_evaluate · browser_find
browser_wait_for · browser_tabs · browser_close · browser_resize
browser_emulate_media · browser_console_messages · browser_network_requests
browser_network_request · browser_handle_dialog · browser_run_code_unsafe
```

**İki araç bu projede özellikle değerli, çünkü Claude'un Chrome
eklentisinde çalışmıyordu:**

- `browser_resize` → **gerçek mobil ölçümü**. Daha önce duyarlı davranış
  yalnız sınıf adlarından ve `sizes` özniteliğinden çıkarılabiliyordu.
- `browser_emulate_media` → `prefers-color-scheme` emülasyonu, yani
  **koyu tema ölçümü** sitenin kendi anahtarını aramadan.

## Skill

`browser-measure` (`~/.agents/skills/browser-measure/SKILL.md`) — ölçüm
disiplinini öğretir: hesaplanmış stili ölç (kaynak koddan tahmin etme),
tek çağrıda toplu tara, duyarlı davranışı `browser_resize` ile ölç,
DPR'yi hesaba kat, ölçmediğini söyle, araç deponun içine yazmasın,
doğrulamayı ayrı worktree'de yap.

İçinde bu projede gerçekten yapılmış dört yanlış iddia ve ölçülen
karşılıkları var — soyut tavsiye değil.

## Doğrulama (yapıldı)

1. `opencode mcp list` → `✓ browser connected`
2. Uçtan uca: demo.bitcosar.com açıldı, `browser_snapshot` alındı,
   **NAVCOUNT 5 / News, Video, Prices, Research, Sponsored** döndü —
   bağımsız ölçümle birebir aynı.
3. Mobil: `browser_resize 390x844` çalıştı, mega menü `display: none`,
   5 `<details>` bulundu.

## İlk gerçek bulgu — mobilde yatay taşma (AÇIK)

Yeni yetenekle ilk koşuda çıktı. 390×844'te ölçüm:

```
innerWidth                390
documentElement.clientW   375
documentElement.scrollW   568      → 193px TAŞMA
sayfa yatay kayıyor:      true
```

Şerit suçlu **değil**: `.cd-ticker__inner` `overflow-x: auto` ile kendi
içinde doğru kayıyor (1318 içerik / 327 görünür).

Gerçek suçlu **başlığın sağ aksiyonları** — kırpılmayan taşan öğeler:

| Öğe | left | right |
|---|---|---|
| `button.cd-tema` | 347 | 377 |
| `span.cd-lang__tek` | 389 | **404** |
| `a.cd-btn--ghost` "Giriş" | 416 | **474** |
| `a.cd-btn` "Ücretsiz üye ol" | 486 | **568** |

`scrollWidth` 568 = "Ücretsiz üye ol" düğmesinin sağ kenarı.
`.cd-topbar__actions` mobilde ne sarıyor ne hamburgere katlanıyor.

Seçenekler: mobilde ikincil aksiyonları hamburger içine taşı · `flex-wrap`
ver · "Giriş"/"Ücretsiz üye ol" metinlerini ikona indir · dil göstergesini
menüye al.

> Not: `8ecf27f` commit mesajı "mobil 390px … yatay taşma YOK" diyor.
> O doğrulama gezinmeyi kontrol etmiş, aksiyon şeridini değil. Taşma
> ölçülebilir ve yukarıda kaynağıyla duruyor.

## Kullanım

```
opencode run --auto "<görev>"            # tek seferlik
opencode                                  # etkileşimli
```

Yapılandırma değiştirilirse servisi yeniden başlat, yoksa eski katalog
kalır:

```
opencode service stop && opencode service start
opencode mcp list
```

## Yan temizlik

`~/.config/opencode/opencode.json` (WSL) ölü bir yapılandırmaydı: port
`20129` kapalı ve **düz metin API anahtarı** taşıyordu. Aktif config
Windows tarafında olduğu için bu dosya okunmuyor, ama anahtar diskte
duruyordu. `baseURL` küme adresine çevrildi, anahtar
`{env:OMNIROUTE_BITCOSAR_KEY}` referansına dönüştürüldü, yedek alındı.

**Anahtar yine de döndürülmeli** — bir süre düz metin olarak diskte
durdu.
