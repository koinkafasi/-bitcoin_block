#!/usr/bin/env bash
# imaj-dogrula.sh — GHCR'daki imajların İMZASINI ve SBOM'unu doğrular.
#
# NEDEN: `imaj.yml` imzalıyor; ama "imzaladık" iddiası DOĞRULANMADIKÇA
# değersizdir. Bu betik sevk öncesi (veya bir olay sonrası) şu soruyu
# yanıtlar: "kümede koşacak imaj gerçekten bizim CI'ımızın imzaladığı
# baytlar mı, SBOM'u imzaya bağlı mı?"
#
# Kullanım:
#   bash deploy/scripts/imaj-dogrula.sh              # HEAD'in sha-xxxxxxx etiketi
#   bash deploy/scripts/imaj-dogrula.sh sha-abc1234
#
# Gereksinimler: cosign (yoksa kurulur), gh CR kimliği.
set -uo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$KOK" || exit 1

REPO="koinkafasi/-bitcoin_block"
REGISTRY="ghcr.io/koinkafasi"
IS_AKISI="refs/heads/main"
ETIKET="${1:-}"
if [ -z "$ETIKET" ]; then
  ETIKET="$(git rev-parse --short HEAD)"
fi

COSIGN="$HOME/.local/bin/cosign"
if [ ! -x "$COSIGN" ]; then
  echo "==> cosign kuruluyor (v2.5.0)"
  GOBIN="$HOME/.local/bin" go install github.com/sigstore/cosign/v2/cmd/cosign@v2.5.0 || {
    echo "cosign kurulamadı — doğrulama YAPILAMADI" >&2
    exit 1
  }
fi

echo "==> GHCR kimliği"
if ! grep -q "ghcr.io" "$HOME/.config/gh/hosts.yml" 2>/dev/null; then
  echo "UYARI: gh ile ghcr.io oturumu görünmüyor. Önce: gh auth refresh -s write:packages"
fi

BASARISIZ=0
for ad in site image-svc search-svc collab; do
  IMJ="$REGISTRY/theblock-$ad:$ETIKET"
  echo
  echo "=== $ad ($IMJ)"
  if ! "$COSIGN" verify \
      --certificate-identity-regexp "^https://github.com/$REPO/.github/workflows/imaj.yml@${IS_AKISI}$" \
      --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
      "$IMJ"; then
    echo "  İMZA DOĞRULANAMADI — bu imaj sevk EDİLMEMELİ." >&2
    BASARISIZ=1
    continue
  fi
  # SBOM attestation: CycloneDX yüklemesi imzaya bağlı mı?
  if "$COSIGN" verify-attestation --type cyclonedx \
      --certificate-identity-regexp "^https://github.com/$REPO/.github/workflows/imaj.yml@${IS_AKISI}$" \
      --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
      "$IMJ" >/dev/null 2>&1; then
    echo "  SBOM attestation: DOĞRULANDI"
  else
    echo "  SBOM attestation YOK/DOĞRULANAMADI" >&2
    BASARISIZ=1
  fi
done

echo
if [ "$BASARISIZ" = 0 ]; then
  echo "tüm imzalar ve SBOM attestation'ları geçerli: $ETIKET"
else
  echo "DOĞRULAMA BAŞARISIZ — sevk etmeyin (yukarıdaki satırlar)." >&2
fi
exit "$BASARISIZ"
