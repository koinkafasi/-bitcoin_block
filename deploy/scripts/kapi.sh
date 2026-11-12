#!/usr/bin/env bash
# kapi.sh — her değişiklikte geçilecek TEK kapı.
#
# NEDEN TEK BETİK:
# Kapılar dağınıktı: testler `go test`, lint `make lint-yeni`, güvenlik
# `make guvenlik-tara`, e2e ayrı, kapsam ayrı. Her biri ayrı ayrı
# hatırlanmak zorundaydı ve biri unutulduğunda CI'da öğreniliyordu.
# Burası tek giriş: yerelde geçen, CI'da da geçer.
#
# SIRA RASTGELE DEĞİL — ucuzdan pahalıya:
#   derleme (saniyeler) → unit test → lint → güvenlik → e2e (dakikalar)
# Erken kırılma geri bildirimi hızlandırır; derlenmeyen kodu taramak
# zaman kaybı.
#
# Kullanım:
#   bash deploy/scripts/kapi.sh            # hızlı kapı (e2e hariç)
#   bash deploy/scripts/kapi.sh --tam      # + e2e + imaj taraması
#   bash deploy/scripts/kapi.sh --rapor    # hiçbir şeyi kırma, yalnız göster
#   make kapi / make kapi-tam
set -uo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$KOK"
export PATH="/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH"

TAM=0
RAPOR=0
for a in "$@"; do
  case "$a" in
    --tam)   TAM=1 ;;
    --rapor) RAPOR=1 ;;
    -h|--yardim) sed -n '2,20p' "${BASH_SOURCE[0]}" | sed 's|^# \?||'; exit 0 ;;
    *) echo "bilinmeyen seçenek: $a" >&2; exit 2 ;;
  esac
done

KAPSAM_ESIGI="${KAPSAM_ESIGI:-16.5}"
BASARISIZ=0
declare -a SONUC

adim() { echo; echo "═══════ $* ═══════"; }
gecti() { SONUC+=("  ✓ $1"); }
kirildi() {
  SONUC+=("  ✗ $1")
  [ "$RAPOR" = 1 ] || BASARISIZ=1
}

# ---------------------------------------------------------------- 1) derleme
adim "1/5  derleme (templ + go build)"
if make templ >/dev/null 2>&1 && (cd go && go build ./... 2>&1 | head -5); then
  # `go build` sessizse başarılı; çıktı varsa hata.
  if (cd go && go build ./... 2>&1 | grep -q .); then
    kirildi "derleme"
  else
    gecti "derleme"
  fi
else
  kirildi "derleme"
fi

# ---------------------------------------------------------------- 2) unit test + kapsam
adim "2/5  unit test + kapsam çıtı (${KAPSAM_ESIGI}%)"
# Kapsam ORİJİNAL profilden okunur: `go/` öneki silinmiş kopyada
# `cmd/backfill` stdlib sanılıp "is not in std" hatası veriyordu.
if (cd go && go vet ./... 2>&1 | head -5 | grep -q .); then
  kirildi "go vet"
else
  gecti "go vet"
fi

if (cd go && go test -count=1 -covermode=set -coverprofile=/tmp/kapi-cover.out ./... > /tmp/kapi-test.log 2>&1); then
  TOPLAM=$(cd go && go tool cover -func=/tmp/kapi-cover.out 2>/dev/null | tail -1 | awk '{print $NF}' | tr -d '%')
  echo "  kapsam: ${TOPLAM:-?}%   çıt: ${KAPSAM_ESIGI}%"
  if [ -n "${TOPLAM:-}" ] && awk -v t="$TOPLAM" -v e="$KAPSAM_ESIGI" 'BEGIN { exit (t < e) }'; then
    gecti "unit test + kapsam (${TOPLAM}%)"
  else
    echo "  kapsam çıtın altında" >&2
    kirildi "kapsam çıtı (${TOPLAM:-?}% < ${KAPSAM_ESIGI}%)"
  fi
else
  grep -E '^(FAIL|--- FAIL)' /tmp/kapi-test.log | head -8 >&2
  kirildi "unit test"
fi

# ---------------------------------------------------------------- 3) lint
adim "3/5  lint (yeni kodda 0 bulgu)"
if bash deploy/scripts/lint.sh --kapi >/tmp/kapi-lint.log 2>&1; then
  gecti "lint"
else
  tail -12 /tmp/kapi-lint.log >&2
  kirildi "lint"
fi

# ---------------------------------------------------------------- 4) güvenlik
adim "4/5  güvenlik (CVE, SAST, sır, workflow)"
GUV_BAYRAK=()
[ "$RAPOR" = 1 ] && GUV_BAYRAK+=(--rapor)
[ "$TAM" = 1 ] && GUV_BAYRAK+=(--imaj)
if bash deploy/scripts/guvenlik-tara.sh "${GUV_BAYRAK[@]}" > /tmp/kapi-guv.log 2>&1; then
  grep -E '^(──────|  ulaşılabilir|✓)' /tmp/kapi-guv.log | tail -12
  gecti "güvenlik"
else
  tail -25 /tmp/kapi-guv.log >&2
  kirildi "güvenlik"
fi

# ---------------------------------------------------------------- 5) e2e
adim "5/5  e2e"
if [ "$TAM" = 1 ]; then
  # E2E ÜRETİME KOŞMAZ: e2e-uretim-korumasi.sh hedefi doğrular.
  if bash deploy/scripts/e2e-izole.sh > /tmp/kapi-e2e.log 2>&1; then
    gecti "e2e"
  else
    tail -20 /tmp/kapi-e2e.log >&2
    kirildi "e2e"
  fi
else
  echo "  atlandı (--tam ile çalışır; ~dakikalar sürer)"
  SONUC+=("  · e2e atlandı")
fi

# ---------------------------------------------------------------- özet
echo
echo "═══════ ÖZET ═══════"
printf '%s\n' "${SONUC[@]}"
echo
if [ "$BASARISIZ" = 0 ]; then
  echo "✓ kapı geçti"
  [ "$TAM" = 0 ] && echo "  (e2e ve imaj taraması için: --tam)"
else
  echo "✗ kapı kırıldı — yukarıdaki ✗ satırlarına bakın"
  [ "$RAPOR" = 1 ] && echo "  (--rapor modunda çıkış 0)"
fi
[ "$RAPOR" = 1 ] && exit 0
exit "$BASARISIZ"
