#!/bin/bash
# Go kapsam raporu uret — Sonar'in bekledigi bicim (coverage.out, depo koku).
#
# Kullanim:  bash deploy/scripts/kapsam-uret.sh
# Sonra:     bash deploy/scripts/sonar-tara.sh
#
# NOT: DB gerektiren paketler (httpapi, repo) burada kosmaz — nil pool ile
# panikleyen testler var (kapak_test.go). `-run` suzgeci yerine paket
# listesi kullaniliyor: DB'siz paketler ANLAMLI kapsam verir.
set -uo pipefail
export PATH="$HOME/.local/bin:/usr/local/go/bin:$PATH"
KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$KOK/go" || exit 1

echo "==> kapsam toplaniyor"
go test -count=1 -covermode=set -coverprofile=/tmp/cover.out ./... > /tmp/cover.log 2>&1
TEST_KOD=$?

if [ ! -s /tmp/cover.out ]; then
  echo "KAPSAM DOSYASI BOS — testler hic kosmadi" >&2
  tail -20 /tmp/cover.log >&2
  exit 1
fi

# Sonar depo kokune gore yol bekler: "theblock/site/internal/..." -> "go/internal/..."
sed -e 's|^theblock/site/|go/|' /tmp/cover.out > "$KOK/coverage.out"

echo "==> kapsam: $(go tool cover -func=/tmp/cover.out 2>/dev/null | tail -1 | awk '{print $NF}')"
echo "==> $KOK/coverage.out  ($(wc -l < "$KOK/coverage.out") satir)"

# Test kirmiziysa kapsam yine yazildi ama kullanici BILMELI.
if [ "$TEST_KOD" != "0" ]; then
  echo "UYARI: bazi testler basarisiz (kapsam yine uretildi):" >&2
  grep -E '^(FAIL|--- FAIL)' /tmp/cover.log | head -5 >&2
fi
exit 0
