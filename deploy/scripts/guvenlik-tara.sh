#!/usr/bin/env bash
# guvenlik-tara.sh — bitcosar güvenlik kapıları. TEK KAYNAK.
#
# CI (.github/workflows/ci.yml) bu betiği çağırır; yerelde `make guvenlik-tara`
# de aynısını koşar. Böylece "CI'da kırıldı, yerelde geçiyordu" olmaz.
#
# Kullanım:
#   bash deploy/scripts/guvenlik-tara.sh           # kapı (kırmızıysa exit 1)
#   bash deploy/scripts/guvenlik-tara.sh --rapor   # yalnız rapor (exit 0)
#   bash deploy/scripts/guvenlik-tara.sh --imaj    # ayrıca Docker imajlarını tara
#
# YIĞIN ALGILAMA: proje yığını henüz kesinleşmedi. Betik go.mod /
# package-lock.json / Cargo.toml varlığına bakar. Adım atlanırsa bunu
# AÇIKÇA yazar — sessizce yeşil göstermez.
set -uo pipefail

TRIVY_SURUM="${TRIVY_SURUM:-0.76.1}"
SEMGREP_SURUM="${SEMGREP_SURUM:-1.183.0}"
ZIZMOR_SURUM="${ZIZMOR_SURUM:-1.32.0}"
GOSEC_SURUM="${GOSEC_SURUM:-v2.30.0}"
GOVULN_SURUM="${GOVULN_SURUM:-v1.9.0}"

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$KOK"

IMAJ=0
RAPOR=0
for a in "$@"; do
  case "$a" in
    --imaj)  IMAJ=1 ;;
    --rapor) RAPOR=1 ;;
    *) echo "bilinmeyen seçenek: $a" >&2; exit 2 ;;
  esac
done
KOD=1
[ "$RAPOR" = 1 ] && KOD=0

BASARISIZ=0
bolum() { echo; echo "──────── $* ────────"; }
atlandi() { echo "  ATLANDI: $*"; }

# ---------------------------------------------------------------- trivy
trivy_kur() {
  local hedef="$HOME/.local/bin"
  mkdir -p "$hedef"
  if [ -x "$hedef/trivy" ] && "$hedef/trivy" --version 2>/dev/null | grep -q "$TRIVY_SURUM"; then
    return 0
  fi
  echo "==> trivy $TRIVY_SURUM kuruluyor"
  local d
  d=$(mktemp -d)
  (
    cd "$d" || exit 1
    local t="trivy_${TRIVY_SURUM}_Linux-64bit.tar.gz"
    local b="https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_SURUM}"
    curl -fsSLO "$b/$t" || exit 1
    curl -fsSLO "$b/trivy_${TRIVY_SURUM}_checksums.txt" || exit 1
    # Sağlama ZORUNLU: doğrulanmamış binary kurulmaz.
    sha256sum -c --ignore-missing "trivy_${TRIVY_SURUM}_checksums.txt" || exit 1
    tar xzf "$t" || exit 1
    install -m 0755 trivy "$hedef/trivy" || exit 1
  ) || { echo "trivy kurulamadı" >&2; rm -rf "$d"; exit 1; }
  rm -rf "$d"
}

if command -v trivy >/dev/null 2>&1 && trivy --version 2>/dev/null | grep -q "$TRIVY_SURUM"; then
  TRIVY=trivy
else
  trivy_kur
  TRIVY="$HOME/.local/bin/trivy"
fi

ATLA=(--skip-dirs node_modules --skip-dirs .git --skip-dirs target
      --skip-dirs .runtime --skip-dirs storage --skip-dirs data
      --skip-dirs dist --skip-dirs .next)

# ---------------------------------------------------------------- 1) bağımlılık CVE
bolum "1/6  bağımlılık CVE"
if [ -f go.mod ] || [ -f go/go.mod ] || [ -f package-lock.json ] || [ -f Cargo.lock ] || [ -f rust/Cargo.lock ]; then
  "$TRIVY" fs -q --scanners vuln --severity CRITICAL,HIGH --ignore-unfixed \
    --exit-code "$KOD" --table-mode summary "${ATLA[@]}" . || BASARISIZ=1
else
  atlandi "bağımlılık dosyası yok (go.mod / package-lock.json / Cargo.lock)"
fi

# ---------------------------------------------------------------- 2) yapılandırma
bolum "2/6  yapılandırma (Dockerfile + k8s manifest)"
if [ -d deploy ]; then
  "$TRIVY" config -q --severity CRITICAL,HIGH --table-mode summary \
    --ignorefile .trivyignore.yaml \
    --exit-code "$KOD" deploy/ || BASARISIZ=1
else
  atlandi "deploy/ yok"
fi

# ---------------------------------------------------------------- 3) sır taraması
bolum "3/6  sır taraması (gömülü anahtar/jeton)"
"$TRIVY" fs -q --scanners secret --exit-code "$KOD" \
  --table-mode summary "${ATLA[@]}" . || BASARISIZ=1

# ---------------------------------------------------------------- 4) semgrep
bolum "4/6  semgrep SAST (ERROR kapı, WARNING rapor)"
if command -v semgrep >/dev/null 2>&1; then
  SEMGREP=semgrep
elif [ -x "$HOME/.local/bin/semgrep" ]; then
  SEMGREP="$HOME/.local/bin/semgrep"
else
  SEMGREP=""
  atlandi "semgrep yok (kurulum: uv tool install semgrep==$SEMGREP_SURUM)"
fi
if [ -n "$SEMGREP" ]; then
  KONF=(--config p/golang --config p/javascript --config p/secrets)
  HEDEF=()
  for d in go src app collab-server editor-src e2e web; do
    [ -d "$d" ] && HEDEF+=("$d")
  done
  if [ ${#HEDEF[@]} -eq 0 ]; then
    atlandi "taranacak kaynak dizini yok"
  else
    "$SEMGREP" scan "${KONF[@]}" --metrics=off --quiet --severity ERROR \
      --error --exclude node_modules --exclude target --exclude dist \
      "${HEDEF[@]}" || BASARISIZ=1
    echo "ERROR seviyesi: temiz"
    echo "--- WARNING (rapor, kapı değil) ---"
    "$SEMGREP" scan "${KONF[@]}" --metrics=off --quiet --severity WARNING \
      --exclude node_modules --exclude target --exclude dist \
      "${HEDEF[@]}" 2>/dev/null \
      | grep -E '^\s+❯❱|Findings:' | sed 's/^/    /' | head -20
  fi
fi

# ---------------------------------------------------------------- 5) dil-özel SAST
bolum "5/6  dil-özel SAST (gosec / govulncheck / npm audit / cargo audit)"
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

if [ -f go.mod ] || [ -f go/go.mod ]; then
  GODIZ=$( [ -f go/go.mod ] && echo go || echo . )
  if ! command -v go >/dev/null 2>&1; then
    atlandi "go yok"
  else
    # gosec `go` komutunu alt süreç olarak çağırır; PATH'te yoksa
    # "0 bulgu" der ve YANLIŞ GÜVEN verir — bu yüzden PATH açıkça kuruldu.
    GOSEC="$HOME/go/bin/gosec"
    if [ ! -x "$GOSEC" ]; then
      echo "==> gosec $GOSEC_SURUM kuruluyor"
      go install "github.com/securego/gosec/v2/cmd/gosec@$GOSEC_SURUM" >/dev/null 2>&1 || true
    fi
    if [ -x "$GOSEC" ]; then
      ( cd "$GODIZ" && "$GOSEC" -quiet -severity high -confidence medium \
          -nosec-require-rules -nosec-require-justification \
          -exclude-dir=vendor ./... ) || BASARISIZ=1
    else
      atlandi "gosec kurulamadı"
    fi

    GOVULN="$HOME/go/bin/govulncheck"
    if [ ! -x "$GOVULN" ]; then
      echo "==> govulncheck $GOVULN_SURUM kuruluyor"
      go install "golang.org/x/vuln/cmd/govulncheck@$GOVULN_SURUM" >/dev/null 2>&1 || true
    fi
    if [ -x "$GOVULN" ]; then
      ( cd "$GODIZ" && "$GOVULN" ./... ) || BASARISIZ=1
    else
      atlandi "govulncheck kurulamadı"
    fi
  fi
else
  atlandi "go.mod yok — gosec/govulncheck çalışmadı"
fi

if [ -f package-lock.json ]; then
  if command -v npm >/dev/null 2>&1; then
    npm audit --audit-level=high --omit=dev || BASARISIZ=1
  else
    atlandi "npm yok"
  fi
else
  atlandi "package-lock.json yok — npm audit çalışmadı"
fi

if [ -f Cargo.toml ] || [ -f rust/Cargo.toml ]; then
  if command -v cargo >/dev/null 2>&1; then
    for d in . rust; do
      [ -f "$d/Cargo.toml" ] && ( cd "$d" && cargo audit ) || true
    done
  else
    atlandi "cargo yok"
  fi
else
  atlandi "Cargo.toml yok — cargo audit çalışmadı"
fi

# ---------------------------------------------------------------- 6) workflow
bolum "6/6  GitHub Actions workflow güvenliği (zizmor)"
if command -v zizmor >/dev/null 2>&1; then
  ZIZMOR=zizmor
elif [ -x "$HOME/.local/bin/zizmor" ]; then
  ZIZMOR="$HOME/.local/bin/zizmor"
else
  ZIZMOR=""
  atlandi "zizmor yok (kurulum: uv tool install zizmor==$ZIZMOR_SURUM)"
fi
if [ -n "$ZIZMOR" ] && [ -d .github/workflows ]; then
  "$ZIZMOR" --min-severity medium --no-progress .github/workflows/ || BASARISIZ=1
fi

# ---------------------------------------------------------------- imaj taraması
if [ "$IMAJ" = 1 ]; then
  bolum "ek  kap imajları (derle + tara)"
  shopt -s nullglob
  for df in deploy/docker/*.Dockerfile; do
    ad=$(basename "$df" .Dockerfile)
    echo "--- $ad derleniyor"
    docker build -q -f "$df" -t "bitcosar-$ad:tara" . >/dev/null \
      || { echo "derleme başarısız: $ad" >&2; BASARISIZ=1; continue; }
    "$TRIVY" image -q --scanners vuln --severity CRITICAL,HIGH --ignore-unfixed \
      --exit-code "$KOD" --table-mode summary "bitcosar-$ad:tara" || BASARISIZ=1
  done
  shopt -u nullglob
fi

echo
if [ "$BASARISIZ" = 0 ]; then
  echo "✓ güvenlik kapıları geçti"
else
  echo "✗ güvenlik kapısı kırıldı"
  [ "$RAPOR" = 1 ] && echo "  (--rapor modunda exit 0 döner)"
fi
[ "$RAPOR" = 1 ] && exit 0
exit "$BASARISIZ"
