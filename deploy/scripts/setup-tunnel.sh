#!/usr/bin/env bash
# theblock Cloudflare Tunnel kurulumu (sistem servisi, coinhit deseniyle aynı).
#
# KULLANIM:
#   1) Cloudflare Dashboard > Zero Trust > Networks > Tunnels'da
#      "theblock" adında bir tünel oluşturun ve RUN TOKEN'ını alın
#      (>200 karakter, eyJ... ile başlar). Bu "tunnel run token"ıdır,
#      girdiğiniz "cfut_..." API token'ı DEĞİLDİR.
#   2)     sudo bash deploy/scripts/setup-tunnel.sh <run-token-dosyası-veya-düz>
#      örn: sudo bash deploy/scripts/setup-tunnel.sh ~/.secrets/theblock-tunnel.token
#
#   Ingress (host→servis) Dashboard'dan tanımlanır:
#     theblock.com.tr      → http://localhost:8070
#     www.theblock.com.tr  → http://localhost:8070
set -euo pipefail

TOKEN_SRC="${1:?Kullanım: setup-tunnel.sh <run-token> veya <token-dosyası>}"
UNIT=/etc/systemd/system/cloudflared-theblock.service
TOKEN_FILE=/etc/cloudflared/theblock-token

# Kaynağa göre token'ı hazırla
if [ -f "$TOKEN_SRC" ]; then
  tmp_token=$(cat "$TOKEN_SRC")
else
  tmp_token="$TOKEN_SRC"
fi

case "$tmp_token" in
  ey*) ;;            # geçerli run token formatı (base64 JWT)
  *)
    echo "UYARI: token 'ey...' ile başlamıyor — run token olmayabilir!"
    ;;
esac

echo "Token hedef: $TOKEN_FILE"
printf '%s' "$tmp_token" | sudo tee "$TOKEN_FILE" >/dev/null
sudo chown root:root "$TOKEN_FILE"
sudo chmod 600 "$TOKEN_FILE"
unset tmp_token

echo "Unit yazılıyor: $UNIT"
sudo tee "$UNIT" >/dev/null <<EOF
[Unit]
Description=Cloudflare Tunnel - theblock.com.tr (WSL Go+Rust)
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/bin/cloudflared --no-autoupdate tunnel run --token-file $TOKEN_FILE
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now cloudflared-theblock.service
sleep 6
sudo systemctl status cloudflared-theblock.service --no-pager | head -12
echo
echo "=== Log önizleme ==="
sudo journalctl -u cloudflared-theblock.service -n 15 --no-pager
echo
echo "=== NOT: Dashboard'da tünel ingress'ini ayarlayın ==="
echo "  theblock.com.tr      -> http://localhost:8070"
echo "  www.theblock.com.tr  -> http://localhost:8070"
