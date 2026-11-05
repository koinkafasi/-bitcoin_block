#!/usr/bin/env bash
# sevk-head.sh — YALNIZ commit'lenmiş işi (HEAD) canlıya alır.
#
# Neden ayrı betik: `dagit.sh` çalışma ağacını sevk eder. Depoda birden fazla
# ajan/kişi çalışırken ağaçta her zaman yarım iş bulunuyor; onu sevk etmek
# bitmemiş özelliği canlıya atmak demek. Nitekim bir kez `dagit.sh` tam da
# bu yüzden (gofmt'lenmemiş yarım dosya) durdu.
#
# Burada HEAD ayrı bir git worktree'sine çıkarılır, orada derlenip test
# edilir; ana çalışma ağacına hiç dokunulmaz. Böylece sevk edilen şey
# commit geçmişinde birebir görünür.
#
# Kullanım (WSL içinden):  bash deploy/scripts/sevk-head.sh
#          Windows'tan:    wsl -d ubuntu -- bash -lc "cd /home/alper/theblock && bash deploy/scripts/sevk-head.sh"
#
# Seçenekler:
#   --kuru    yalnız derle ve test et; servise dokunma (sevk öncesi deneme)
set -uo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
WT=/tmp/tb-sevk-head
SERVIS=theblock-site.service
PORT="${DAGIT_PORT:-8070}"
KURU=0
[ "${1:-}" = "--kuru" ] && KURU=1

cd "$KOK"
HEADC=$(git rev-parse --short HEAD)
echo "==> 1/5 HEAD: $HEADC  $(git log -1 --format=%s | cut -c1-56)"
KIRLI=$(git status --porcelain | wc -l)
if [ "$KIRLI" -gt 0 ]; then
  echo "    not: ağaçta $KIRLI commit'lenmemiş dosya var — SEVK EDİLMEYECEK"
fi

git worktree remove --force "$WT" >/dev/null 2>&1
rm -rf "$WT"
git worktree add --detach "$WT" HEAD >/dev/null 2>&1 \
  || { echo "    worktree kurulamadı" >&2; exit 1; }
temizle() { cd "$KOK"; git worktree remove --force "$WT" >/dev/null 2>&1; }
trap temizle EXIT

cd "$WT/go"

echo "==> 2/5 biçim ve vet"
B=$(gofmt -l ./internal/ ./cmd/)
if [ -n "$B" ]; then
  echo "    DURDURULDU: gofmt'lenmemiş dosyalar:" >&2
  echo "$B" | sed 's/^/      /' >&2
  exit 1
fi
if ! go vet ./... >/tmp/sevk-head-vet.log 2>&1; then
  echo "    DURDURULDU: go vet temiz değil" >&2
  head -20 /tmp/sevk-head-vet.log >&2
  exit 1
fi
echo "    temiz"

echo "==> 3/5 testler"
if ! go test ./... >/tmp/sevk-head-test.log 2>&1; then
  echo "    DURDURULDU: HEAD testleri geçmiyor (ayrıntı /tmp/sevk-head-test.log)" >&2
  grep -E '^(FAIL|--- FAIL)' /tmp/sevk-head-test.log | head -10 >&2
  exit 1
fi
echo "    geçen paket: $(grep -c '^ok' /tmp/sevk-head-test.log)"

echo "==> 4/5 derleme"
G=$(mktemp /tmp/site-head-XXXXXX)
if ! go build -o "$G" ./cmd/site; then
  rm -f "$G"
  echo "    DERLEME BAŞARISIZ — çalışan ikiliye dokunulmadı" >&2
  exit 1
fi
echo "    ikili: $(stat -c%s "$G") bayt"

if [ "$KURU" -eq 1 ]; then
  rm -f "$G"
  echo
  echo "KURU KOŞU TEMİZ — sevk geçer ($HEADC)"
  exit 0
fi

echo "==> 5/5 servis"
YEDEK="/tmp/site-yedek-$(date +%H%M%S)"
cp "$KOK/go/bin/site" "$YEDEK" 2>/dev/null && echo "    yedek: $YEDEK"
ONCE=$(pgrep -f "$KOK/go/bin/site" | head -1)
mv "$G" "$KOK/go/bin/site"
chmod +x "$KOK/go/bin/site"
systemctl --user restart "$SERVIS" || { echo "    RESTART BAŞARISIZ" >&2; exit 1; }

HAZIR=0
for _ in $(seq 1 60); do
  curl -fsS -o /dev/null "http://127.0.0.1:${PORT}/healthz" 2>/dev/null && { HAZIR=1; break; }
done
echo "    pid $ONCE -> $(pgrep -f "$KOK/go/bin/site" | head -1) · $(systemctl --user is-active $SERVIS)"
if [ "$HAZIR" -ne 1 ]; then
  echo "    SAĞLIK GELMEDİ — son günlük:" >&2
  journalctl --user -u "$SERVIS" --since '2 min ago' --no-pager 2>/dev/null | tail -20 >&2
  echo "    geri almak için: cp $YEDEK $KOK/go/bin/site && systemctl --user restart $SERVIS" >&2
  exit 1
fi
for u in /healthz / /gazete /static/gazete-baski.css /static/gazete-editor.js; do
  printf "    %-28s HTTP %s\n" "$u" \
    "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${PORT}$u")"
done

echo
echo "SEVK TAMAM ($HEADC) · geri alma: cp $YEDEK $KOK/go/bin/site && systemctl --user restart $SERVIS"
