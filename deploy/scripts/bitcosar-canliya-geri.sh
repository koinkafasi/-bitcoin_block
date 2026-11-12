#!/usr/bin/env bash
# bitcosar-canliya-geri.sh — bitcosar.com trafiğini WordPress'e geri verir.
#
# Kullanım: bash deploy/scripts/bitcosar-canliya-geri.sh [yedek-dizini]
set -uo pipefail

KUBECTL="/mnt/c/Program Files/Docker/Docker/resources/bin/kubectl.exe"
KC="${KUBECONFIG_WIN:-D:\\talos\\.kube\\config}"
kctl() { "$KUBECTL" --kubeconfig "$KC" "$@"; }

YEDEK="${1:-}"
if [ -z "$YEDEK" ]; then
  YEDEK=$(ls -dt "$HOME"/bitcosar-gecis/canli-geri-* 2>/dev/null | head -1)
fi
[ -n "$YEDEK" ] && [ -f "$YEDEK/wp-ingress.yaml" ] || {
  echo "HATA: WordPress ingress yedeği bulunamadı." >&2
  echo "      Kullanım: $0 <yedek-dizini>" >&2
  exit 1
}

echo "=== yedek: $YEDEK ==="
echo "=== yeni ingress kaldırılıyor ==="
kctl -n bitcosar-site delete ingress bitcosar-site bitcosar-collab --ignore-not-found

echo "=== WordPress ingress geri yükleniyor ==="
kctl apply -f "$(wslpath -w "$YEDEK/wp-ingress.yaml")"

echo "=== WordPress pod'u geri açılıyor ==="
R=$(cat "$YEDEK/wp-replicas.txt" 2>/dev/null || echo 1)
[ "$R" -ge 1 ] 2>/dev/null || R=1
kctl -n bitcosar scale deploy/wordpress --replicas="$R"
kctl -n bitcosar rollout status deploy/wordpress --timeout=180s || true

echo
echo "=== doğrulama ==="
sleep 6
for i in 1 2 3 4 5; do
  kod=$(curl -s -o /dev/null -w '%{http_code}' --max-time 25 https://bitcosar.com/)
  echo "  / -> $kod"
  [ "$kod" = "200" ] && break
  sleep 8
done
echo "WordPress üretimde. Yeni site k8s.bitcosar.com'da çalışmaya devam ediyor."
