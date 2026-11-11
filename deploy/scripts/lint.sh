#!/usr/bin/env bash
# lint.sh — Go statik analizi (golangci-lint).
#
# Kullanım:
#   bash deploy/scripts/lint.sh          # TAM depo raporu (çıkış her zaman 0)
#   bash deploy/scripts/lint.sh --kapi   # YENİ KOD kapısı (bulgu varsa 1)
#
# NEDEN İKİ MOD: depodaki eski borç bir gecede kapanmaz. Tam depoyu kapı
# yapmak CI'ı sürekli kırmızı başlatır; tamamen kapatmak ise yeni koda
# göz yumar. Bu yüzden CI'da kapı `--new-from-rev` ile YENİ satırlara
# bakar, tam depo ise yalnız raporlanır (docs/tedarik-zinciri.md).
#
# SÜRÜM SABİT: `go install ...@latest` her koşuda kural setini değiştirir
# (yerelde yeşil, CI'da kırmızı). Aynı sürüm resmî ikiliden kurulur.
set -uo pipefail

SURUM="${GOLANGCI_SURUM:-v2.15.0}"
KAPI=0
[ "${1:-}" = "--kapi" ] && KAPI=1

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$KOK" || exit 1

export PATH="$HOME/.local/bin:/usr/local/go/bin:$HOME/go/bin:$PATH"

if ! command -v go >/dev/null 2>&1; then
  echo "go bulunamadı — lint koşturulamadı" >&2
  exit 1
fi

LINT="$HOME/.local/bin/golangci-lint"
if [ ! -x "$LINT" ]; then
  echo "==> golangci-lint $SURUM kuruluyor"
  GOBIN="$HOME/.local/bin" go install \
    "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$SURUM" || {
    echo "golangci-lint kurulamadı" >&2
    exit 1
  }
fi

# Yapılandırma depo kökünde (.golangci.yml); linter `go/` içinden koşar
# ama dosyayı yukarı doğru arar.
cd "$KOK/go" || exit 1

if [ "$KAPI" = 1 ]; then
  TABAN="${LINT_TABAN:-origin/main}"
  if ! git -C "$KOK" rev-parse --verify "$TABAN" >/dev/null 2>&1; then
    echo "UYARI: '$TABAN' yok — son commit'e göre karşılaştırılıyor." >&2
    TABAN="HEAD~1"
  fi
  echo "==> kapı: yeni kod ($TABAN'e göre)"
  "$LINT" run --timeout 12m --new-from-rev="$TABAN"
  KOD=$?
  if [ "$KOD" = 0 ]; then
    echo "yeni kodda bulgu yok"
  else
    echo "YENİ KODDA BULGU VAR (yukarıda)." >&2
  fi
  exit "$KOD"
fi

echo "==> tam depo raporu (bloklamaz)"
"$LINT" run --timeout 12m --output.text.print-issued-lines=false > /tmp/lint-rapor.txt 2>&1
grep -E '^\* |^[0-9]+ issues|bulgu yok' /tmp/lint-rapor.txt | sed 's/^/  /'
echo
echo "ayrıntı: /tmp/lint-rapor.txt"
echo "NOT: bu mod BLOKLAMAZ. Kapı: make lint-yeni"
