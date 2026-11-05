#!/usr/bin/env bash
# Ikon sprite surumunu guncelle (Faz 181).
#
# NEDEN: sprite URL'si hic degismedigi icin Cloudflare/tarayici onbellegi
# eski kopyayi tutuyor. Olculdu: alan adi 8 gunluk, 41 sembolluk sprite'i
# "immutable" basligiyla sunuyordu; sunucu 641 sembol veriyordu. Eksik
# sembol HATA VERMEZ, sessizce bos kutu basilir.
#
# Cozum: URL'e icerik ozeti eklenir. Icerik degisince URL degisir, onbellek
# kendiliginden tazelenir. Surum eski kalirsa `TestIkonSurumuGuncel` kirmizi
# olur, yani unutulmasi mumkun degil.
set -euo pipefail
cd /home/alper/theblock

SPRITE=go/internal/webui/static/vendor/tabler-icons.svg
SURUM=$(sha1sum "$SPRITE" | cut -c1-8)

echo "sprite: $SPRITE"
echo "sembol: $(grep -c '<symbol' "$SPRITE")"
echo "surum:  $SURUM"

# Eski surumu temizle, yenisini yaz.
grep -rl 'tabler-icons\.svg' go/internal --include='*.templ' --include='*.html' \
  --include='*.js' --include='*.go' 2>/dev/null | grep -v '_templ.go' | while read -r f; do
  sed -i -E "s|tabler-icons\.svg\?v=[0-9a-f]+|tabler-icons.svg|g" "$f"
  sed -i -E "s|tabler-icons\.svg#|tabler-icons.svg?v=$SURUM#|g" "$f"
  sed -i -E "s|tabler-icons\.svg\"|tabler-icons.svg?v=$SURUM\"|g" "$f"
  echo "  $f"
done

echo
echo "referans: $(grep -rho 'tabler-icons\.svg?v=[0-9a-f]*' go/internal --include='*.templ' --include='*.html' --include='*.js' --include='*.go' | grep -v _templ.go | wc -l)"
echo "surumsuz: $(grep -rho 'tabler-icons\.svg[^?]' go/internal --include='*.templ' --include='*.html' --include='*.js' --include='*.go' | grep -v _templ.go | wc -l)"
