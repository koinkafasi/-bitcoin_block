#!/usr/bin/env bash
# OmniRoute kota gözlemcisi — her gün 03:00'te (kota yenileme anı) çalışır.
# Görev: site sağlığını doğrula, durumu günlüğe yaz, uyandırma bayrağını bas.
set -u
LOG="$HOME/.omniroute-wake.log"
FLAG="$HOME/.omniroute-resume"
stamp() { date "+%Y-%m-%d %H:%M:%S"; }
echo "[$(stamp)] OmniRoute kota gözlemcisi çalıştı (03:00 kontrolü)" >> "$LOG"

HEALTH=$(curl -s -o /dev/null -w '%{http_code}' --max-time 6 http://127.0.0.1:8070/healthz 2>/dev/null || echo 000)
echo "[$(stamp)] site /healthz -> $HEALTH" >> "$LOG"
if [ "$HEALTH" = "200" ]; then SITE_OK="evet"; else SITE_OK="hayir"; fi

cat > "$FLAG" <<EOF
OmniRoute kota yenilendi (03:00).
Site durumu: sağlıklı=$SITE_OK
Devam: wsl bash -lc 'tail -60 /home/alper/theblock/docs/ILERLEME.md'
EOF
chmod 600 "$FLAG"

if command -v notify-send >/dev/null 2>&1; then
  notify-send "OmniRoute hazir" "Kota yenilendi; projeye devam edilebilir (site saglikli=$SITE_OK)." || true
fi
echo "[$(stamp)] uyandırma bayrağı yazıldı: $FLAG" >> "$LOG"
