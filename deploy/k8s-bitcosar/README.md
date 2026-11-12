# bitcosar-site manifestleri (ÜRETİLMİŞ — elle düzenlemeyin)

`deploy/scripts/bitcosar-manifest-uret.sh` tarafından `deploy/k8s/`
içinden türetilir. Elle düzenleme bir sonraki üretimde kaybolur;
değişiklik `deploy/k8s/` altında yapılmalı.

| | theblock | bitcosar |
|---|---|---|
| namespace | `theblock` | `bitcosar-site` |
| alan adı | theblock.com.tr | bitcosar.com |
| imaj | ghcr.io/koinkafasi/theblock-site | (aynı) |

## Sevk

```bash
bash deploy/scripts/bitcosar-manifest-uret.sh
NAMESPACE=bitcosar-site bash deploy/scripts/k8s-sevk.sh sha-XXXXXXX
```
