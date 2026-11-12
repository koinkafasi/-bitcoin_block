#!/usr/bin/env bash
# bitcosar-manifest-uret.sh — theblock manifestlerinden bitcosar sürümü üretir.
#
# NEDEN TÜRETİYORUZ, KOPYALAMIYORUZ:
# deploy/k8s/*.yaml tek kaynak. İkinci bir elle bakımlı kopya tutulsaydı
# güvenlik ayarları (seccomp, RBAC, NetworkPolicy, Kyverno, kaynak
# tavanı) iki yerde ayrı ayrı güncellenir, biri er geç unutulurdu.
# Bu betik dönüşümü her çalıştırmada baştan yapar.
#
# NE ÜRETİR: deploy/k8s-bitcosar/ altına namespace ve ad önekleri
# değiştirilmiş manifestler. UYGULAMAZ — sevk ayrı ve bilinçli bir adım.
#
# Kullanım:
#   bash deploy/scripts/bitcosar-manifest-uret.sh
#   less deploy/k8s-bitcosar/50-ingress.yaml          # inceleme
#   NAMESPACE=bitcosar-site bash deploy/scripts/k8s-sevk.sh sha-XXXXXXX
set -uo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
KAYNAK="$KOK/deploy/k8s"
HEDEF="$KOK/deploy/k8s-bitcosar"

ESKI_NS="theblock"
YENI_NS="bitcosar-site"
ESKI_ALAN="theblock.com.tr"
YENI_ALAN="bitcosar.com"

[ -d "$KAYNAK" ] || { echo "HATA: $KAYNAK yok" >&2; exit 1; }
mkdir -p "$HEDEF"

echo "==> $KAYNAK  ->  $HEDEF"
echo "    namespace: $ESKI_NS -> $YENI_NS"
echo "    alan adi : $ESKI_ALAN -> $YENI_ALAN"
echo

for f in "$KAYNAK"/*.yaml; do
  ad=$(basename "$f")
  cikti="$HEDEF/$ad"

  # Sira onemli: once alan adlari (theblock.com.tr icinde "theblock"
  # geciyor), sonra namespace referanslari, en son kalan adlar.
  sed \
    -e "s|www\.${ESKI_ALAN}|www.${YENI_ALAN}|g" \
    -e "s|k8s\.${ESKI_ALAN}|k8s.${YENI_ALAN}|g" \
    -e "s|${ESKI_ALAN}|${YENI_ALAN}|g" \
    -e "s|namespace: ${ESKI_NS}|namespace: ${YENI_NS}|g" \
    -e "s|^  name: ${ESKI_NS}$|  name: ${YENI_NS}|" \
    -e "s|namespaces: \[${ESKI_NS}\]|namespaces: [${YENI_NS}]|g" \
    -e "s|kubernetes.io/metadata.name: ${ESKI_NS}|kubernetes.io/metadata.name: ${YENI_NS}|g" \
    -e "s|part-of: ${ESKI_NS}|part-of: bitcosar|g" \
    -e "s|${ESKI_NS}|bitcosar|g" \
    "$f" > "$cikti"

  # Imaj adi DEGISMEZ: iki site ayni Go binary'sini calistiriyor,
  # icerik veritabanindan geliyor. Depo adi tarihsel.
  sed -i "s|ghcr.io/koinkafasi/bitcosar-|ghcr.io/koinkafasi/theblock-|g" "$cikti"

  echo "  + $ad"
done

cat > "$HEDEF/README.md" <<'MD'
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
MD

echo
echo "==> uretildi: $HEDEF"

# Dogrulama: YORUM OLMAYAN satirda kalan referans bir hatadir.
# Kyverno'da `namespaces: [theblock]` kalsaydi guvenlik politikalari yeni
# namespace'e hic uygulanmazdi — sessiz ve tehlikeli bir bosluk.
echo "==> dogrulama: yorum disi 'theblock' referansi"
KALAN=$(grep -rn "theblock" "$HEDEF"/*.yaml 2>/dev/null \
  | grep -v "ghcr.io" \
  | grep -vE ":[0-9]+:[[:space:]]*#" || true)
if [ -n "$KALAN" ]; then
  echo "$KALAN"
  echo
  echo "HATA: yukaridaki satirlar donusturulmedi." >&2
  exit 1
fi
echo "    temiz (yalniz aciklama satirlarinda geciyor)"
echo
echo "UYGULANMADI. Incelemek icin: less $HEDEF/50-ingress.yaml"
