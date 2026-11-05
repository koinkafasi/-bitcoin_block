#!/usr/bin/env bash
# action-sha-guncelle.sh — workflow'lardaki SHA sabitlemelerini tazeler.
#
# Workflow'larda action'lar commit SHA'sı ile sabitlenmiştir (`@v4` gibi
# hareketli etiketler, etiket sahibi onları taşıdığında bizim CI'mızda
# habersiz kod koşturur). Sabitlemenin bedeli: güncellemeler elle gelir.
# Bu betik her satırın yanındaki `# vX.Y.Z` yorumuna bakar, o etiketin
# GÜNCEL SHA'sını sorar ve farklıysa bildirir.
#
# Kullanım:
#   bash deploy/scripts/action-sha-guncelle.sh          # yalnız göster
#   bash deploy/scripts/action-sha-guncelle.sh --yaz    # dosyaları güncelle
#
# `gh` ile kimlik doğrulaması gerekir (gh auth login).
set -uo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
YAZ=0
[ "${1:-}" = "--yaz" ] && YAZ=1

command -v gh >/dev/null 2>&1 || { echo "gh gerekli" >&2; exit 1; }

degisen=0
for wf in "$KOK"/.github/workflows/*.yml; do
  # Desen:  uses: owner/repo@<40 hex> # vX.Y.Z   (ya da # stable)
  while IFS= read -r satir; do
    repo=$(echo "$satir" | sed -nE 's#.*uses: ([^@]+)@[0-9a-f]{40}.*#\1#p')
    mevcut=$(echo "$satir" | sed -nE 's#.*@([0-9a-f]{40}).*#\1#p')
    ref=$(echo "$satir" | sed -nE 's@.*#[[:space:]]*([^[:space:]]+)[[:space:]]*$@\1@p')
    [ -z "$repo" ] || [ -z "$mevcut" ] || [ -z "$ref" ] && continue

    yeni=$(gh api "repos/$repo/commits/$ref" --jq .sha 2>/dev/null)
    if [ -z "$yeni" ]; then
      printf '  ? %-34s %s  SHA alinamadi\n' "$repo" "$ref"
      continue
    fi
    if [ "$yeni" = "$mevcut" ]; then
      printf '  = %-34s %s  guncel\n' "$repo" "$ref"
    else
      printf '  ↑ %-34s %s  %s -> %s\n' "$repo" "$ref" "${mevcut:0:12}" "${yeni:0:12}"
      degisen=1
      if [ "$YAZ" = 1 ]; then
        sed -i "s#${mevcut}#${yeni}#g" "$wf"
      fi
    fi
  done < <(grep -E 'uses: [^@]+@[0-9a-f]{40}' "$wf" | sort -u)
done

echo
if [ "$degisen" = 0 ]; then
  echo "tum sabitlemeler guncel"
elif [ "$YAZ" = 1 ]; then
  echo "dosyalar guncellendi — degisiklikleri gozden gecirip commit'leyin"
  echo "ONEMLI: yeni SHA'nin icerigini de kontrol edin; sabitlemenin amaci"
  echo "        kor guncelleme YAPMAMAK."
else
  echo "guncelleme var — yazmak icin: --yaz"
fi
