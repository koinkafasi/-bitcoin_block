# bitcosar

Kripto haber portalı. **CoinDesk tasarımı** + CoinDesk kategori/içerik
yapısı, theblock modülleri üzerinde. Çok dilli, AI destekli, performans ve
SEO odaklı.

- Depo: <https://github.com/koinkafasi/-bitcoin_block>
- Yerel geliştirme: WSL (Ubuntu), servis `127.0.0.1:8090`
- Yayın: `https://demo.bitcosar.com` (Cloudflare Tunnel)

## Belgeler

| Belge | İçerik |
|---|---|
| [`docs/COINDESK-SPEC.md`](docs/COINDESK-SPEC.md) | CoinDesk'ten **ölçülen** tasarım/yapı (token, nav, URL, modül sırası) |
| [`docs/PLAN.md`](docs/PLAN.md) | 6 fazlık uygulama planı |
| [`docs/KAYNAK-MERKEZI.md`](docs/KAYNAK-MERKEZI.md) | Kripto RSS/API kataloğu + **fiilen test sonuçları** |
| [`docs/guvenlik-hatti.md`](docs/guvenlik-hatti.md) | Güvenlik kapıları, sürüm sabitleme, sır politikası |

## Mimari (özet)

```
Internet → Cloudflare (WAF, cache, demo.bitcosar.com)
         → Tunnel "bitcosar-demo"
         → Go 1.27 monoliti (127.0.0.1:8090, WSL)
             • SSR (templ) — JS'siz makale sayfası
             • /api/* JSON
             • i18n (tr kök, en …)
             • sitemap / news-sitemap / RSS / robots
             • AI hattı (özet, çeviri, sınıflandırma, kümeleme)
         → PostgreSQL 17 (:5434, ayrı container) + pgvector
         → Redis (önbellek)
         → image-svc (Rust, AVIF/WebP) + arama (search-svc)
         → storage-private (ham kaynak arşivi) + storage (medya)
```

## Çalıştırma

```bash
# veritabanı (ayrı container, theblock'a dokunmaz)
docker start bitcosar-pg            # 127.0.0.1:5434

# uygulama
systemctl --user status bitcosar-site
systemctl --user restart bitcosar-site
curl -sS http://127.0.0.1:8090/readyz    # {"status":"ready","db":"ok"}

# göçler
source ~/.config/bitcosar/db.env
DATABASE_URL="postgres://bitcosar:$POSTGRES_PASSWORD@127.0.0.1:5434/bitcosar?sslmode=disable" \
  go run ./cmd/migrate        # go/ içinden

# güvenlik kapıları (CI ile birebir aynı)
make guvenlik-tara
```

## Sırlar

Repoda **hiçbir sır yok**. Hepsi `~/.config/bitcosar/` altında `0600`:

| Dosya | İçerik |
|---|---|
| `secrets.env` | GitHub PAT, Cloudflare API jetonu, hesap/zone kimlikleri |
| `cloudflared-token` | Tünel jetonu (argv'de değil, dosyada) |
| `db.env` | PostgreSQL parolası |
| `app.env` | `DATABASE_URL`, depolama yolları, port |

`git log --all -p | grep -c 'ghp_\|cfut_'` → **0** (doğrulandı).

## Durum

Faz 0 tamam. Faz 1 (CoinDesk tasarım sistemi) sürüyor — ayrıntı:
[`docs/PLAN.md`](docs/PLAN.md).
