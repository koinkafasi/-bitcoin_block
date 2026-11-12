#!/usr/bin/env bash
# bitcosar-canliya-al.sh — bitcosar.com trafiğini yeni Go sitesine devreder.
#
# ÖN KOŞUL: yeni site k8s.bitcosar.com'da sağlıklı (bu betik önce ölçer).
#
# YAPTIĞI:
#   1) Ön kontrol: yeni site 200 veriyor mu, pod'lar hazır mı
#   2) WordPress ingress'lerini YAML'a yedekle, deployment'ı 0'a çek (SİLME)
#   3) wp-mysql ÇALIŞMAYA DEVAM EDER (veri bozulmadan bekler)
#   4) Yeni ingress'i uygula
#   5) Doğrula; başarısızsa OTOMATİK GERİ DÖN
#
# GERİ DÖNÜŞ: bash deploy/scripts/bitcosar-canliya-geri.sh
set -uo pipefail

KUBECTL="/mnt/c/Program Files/Docker/Docker/resources/bin/kubectl.exe"
KC="${KUBECONFIG_WIN:-D:\\talos\\.kube\\config}"
kctl() { "$KUBECTL" --kubeconfig "$KC" "$@"; }

YEDEK="$HOME/bitcosar-gecis/canli-geri-$(date +%Y%m%d-%H%M)"
mkdir -p "$YEDEK"; chmod 700 "$YEDEK"

echo "=== 1) ON KONTROL: yeni site sağlıklı mı ==="
HAZIR=1
for y in / /healthz /markets; do
  kod=$(curl -s -o /dev/null -w '%{http_code}' --max-time 20 "https://k8s.bitcosar.com$y")
  printf "  %-12s %s\n" "$y" "$kod"
  [ "$kod" = "200" ] || HAZIR=0
done
PODLAR=$(kctl -n bitcosar-site get pod --field-selector=status.phase=Running -o name | wc -l)
echo "  çalışan pod: $PODLAR"
[ "$PODLAR" -ge 5 ] || HAZIR=0
if [ "$HAZIR" != 1 ]; then
  echo "HATA: yeni site hazır değil, trafik DEVRALINMADI." >&2
  exit 1
fi

echo
echo "=== 2) WordPress yedeği ==="
kctl -n bitcosar get ingress wordpress wp-engelli wp-giris -o yaml > "$YEDEK/wp-ingress.yaml" 2>/dev/null
# `kubectl get -o yaml` bir List döndürür: `kind: Ingress` satırları 2
# boşlukla GİRİNTİLİDİR. Baştaki `^` ile aramak 0 veriyordu (12 Kasım
# 2026'da yaşandı: yedek "boş" sanıldı, geri dönüş yine de çalıştı çünkü
# dosya aslında doluydu). Girintisiz arama kullanılıyor.
n=$(grep -c 'kind: Ingress' "$YEDEK/wp-ingress.yaml" 2>/dev/null | head -1 | tr -dc '0-9')
n=${n:-0}
echo "  ingress yedek: $n adet -> $YEDEK"
if [ "$n" -lt 1 ]; then
  echo "HATA: yedek alınamadı, DURDUM." >&2; exit 1
fi
kctl -n bitcosar get deploy wordpress -o yaml > "$YEDEK/wp-deploy.yaml" 2>/dev/null
kctl -n bitcosar get deploy wordpress -o jsonpath='{.spec.replicas}' > "$YEDEK/wp-replicas.txt" 2>/dev/null
echo "  eski replika sayısı: $(cat "$YEDEK/wp-replicas.txt" 2>/dev/null)"

echo
echo "=== 3) WordPress pod'u durduruluyor (SİLİNMİYOR) ==="
kctl -n bitcosar scale deploy/wordpress --replicas=0
sleep 3
kctl -n bitcosar get deploy wordpress wp-mysql 2>&1 | tail -2

echo
echo "=== 4) WordPress ingress'leri kaldırılıyor ==="
kctl -n bitcosar delete ingress wordpress wp-engelli wp-giris --ignore-not-found

# ÖNİZLEME INGRESS'İ ÖNCE SİLİNMELİ: `k8s.bitcosar.com` yolunu o tutuyor.
# Yeni ingress aynı host+yolu tanımladığı için nginx admission webhook
# isteği reddediyor ("host ... already defined"). 12 Kasım 2026'da bu sıra
# yüzünden uygulama reddedildi ve trafik boşta kaldı (404), sonra geri
# dönüldü.
echo "=== 4b) önizleme ingress'i kaldırılıyor ==="
kctl -n bitcosar-site delete ingress bitcosar-onizleme --ignore-not-found
sleep 3

echo
echo "=== 5) yeni ingress uygulanıyor ==="
kctl apply -f "$(wslpath -w ~/bitcosar/deploy/k8s-bitcosar/50-ingress.yaml)" || {
  echo "HATA: ingress uygulanamadı, WordPress geri açılıyor." >&2
  kctl -n bitcosar-site delete ingress bitcosar-site bitcosar-collab --ignore-not-found
  kctl apply -f "$(wslpath -w "$YEDEK/wp-ingress.yaml")"
  kctl -n bitcosar scale deploy/wordpress --replicas="$(cat "$YEDEK/wp-replicas.txt" 2>/dev/null || echo 1)"
  exit 1
}

echo
echo "=== 6) DOĞRULAMA (bitcosar.com artık yeni site) ==="
sleep 8
BASARILI=0
for i in $(seq 1 10); do
  kod=$(curl -s -o /dev/null -w '%{http_code}' --max-time 25 https://bitcosar.com/)
  echo "  deneme $i: / -> $kod"
  if [ "$kod" = "200" ]; then BASARILI=1; break; fi
  sleep 8
done

if [ "$BASARILI" != 1 ]; then
  echo
  echo "!!! DOĞRULAMA BAŞARISIZ — GERİ DÖNÜLÜYOR !!!" >&2
  kctl -n bitcosar-site delete ingress bitcosar-site --ignore-not-found
  kctl apply -f "$(wslpath -w "$YEDEK/wp-ingress.yaml")"
  kctl -n bitcosar scale deploy/wordpress --replicas="$(cat "$YEDEK/wp-replicas.txt" 2>/dev/null || echo 1)"
  echo "WordPress geri açıldı. Yedek: $YEDEK" >&2
  exit 1
fi

echo
echo "=== 7) OG ve içerik doğrulaması ==="
curl -s --max-time 25 https://bitcosar.com/ > /tmp/canli.html
echo "  ana sayfa: $(wc -c < /tmp/canli.html) bayt"
echo "  canonical: $(grep -oE 'rel="canonical" href="[^"]*"' /tmp/canli.html | head -1)"
echo "  og-varsayilan.png: $(curl -s -o /dev/null -w '%{http_code}' --max-time 20 https://bitcosar.com/og-varsayilan.png)"
echo "  robots.txt tuzak: $(curl -s --max-time 15 https://bitcosar.com/robots.txt | grep -c 'arsiv/dizin')"
echo
echo "CANLI: bitcosar.com -> bitcosar-site"
echo "Geri dönüş: bash deploy/scripts/bitcosar-canliya-geri.sh $YEDEK"
