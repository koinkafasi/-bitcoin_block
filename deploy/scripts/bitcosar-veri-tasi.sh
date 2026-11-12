#!/usr/bin/env bash
# bitcosar-veri-tasi.sh — yerel geliştirme veritabanını kümeye taşır.
#
# NEDEN GEREKLİ: kümedeki `theblock` veritabanı BAŞKA bir siteye ait
# (theblock.com.tr, genel haber portalı — ölçüldü 12 Kasım 2026:
# 58.579 içerik, "Kılıçdaroğlu…", "Ardahan'da kış lastiği…").
# bitcosar.com'un kripto içeriği YEREL veritabanında (5.568 içerik,
# WordPress'ten taşınmış + RSS'ten gelen). Bu iki veri karışmamalı.
#
# NE YAPAR:
#   1) yerel veritabanının tam yedeğini alır
#   2) yedeği doğrular (tablo ve satır sayısı)
#   3) kümedeki bitcosar-site/postgres'e yükler
#
# NE YAPMAZ: kümedeki `theblock` namespace'ine DOKUNMAZ.
#
# Kullanım:
#   bash deploy/scripts/bitcosar-veri-tasi.sh --yedek-al     # 1. adım
#   bash deploy/scripts/bitcosar-veri-tasi.sh --yukle DOSYA  # 2. adım
set -uo pipefail

NS="${NAMESPACE:-bitcosar-site}"
YEDEK_DIZIN="${YEDEK_DIZIN:-$HOME/bitcosar-gecis}"
KUBECTL="${KUBECTL:-kubectl.exe}"

kullanim() {
  sed -n '2,22p' "${BASH_SOURCE[0]}" | sed 's|^# \?||'
  exit 1
}

yedek_al() {
  set -a
  # shellcheck disable=SC1090
  . "$HOME/.config/bitcosar/app.env"
  set +a
  local dsn="${DATABASE_URL:?DATABASE_URL yok}"

  mkdir -p "$YEDEK_DIZIN"
  local damga dosya
  damga=$(date +%Y%m%d-%H%M)
  dosya="$YEDEK_DIZIN/bitcosar-$damga.dump"

  echo "==> yerel veritabanı yedekleniyor"
  # --format=custom: seçmeli geri yükleme ve paralel restore mümkün.
  # --no-owner/--no-acl: kümedeki rol adları farklı; sahiplik taşınmaz.
  pg_dump "$dsn" --format=custom --no-owner --no-acl --file="$dosya" || {
    echo "HATA: yedek alınamadı" >&2; exit 1; }

  echo "    dosya : $dosya"
  echo "    boyut : $(du -h "$dosya" | cut -f1)"

  echo
  echo "==> yedek doğrulanıyor (içindekiler okunabiliyor mu)"
  local tablo
  tablo=$(pg_restore --list "$dosya" 2>/dev/null | grep -c "TABLE DATA" || echo 0)
  echo "    tablo verisi: $tablo"
  [ "$tablo" -gt 20 ] || { echo "HATA: yedek eksik görünüyor" >&2; exit 1; }

  echo
  echo "==> kaynak veritabanı özeti (karşılaştırma için)"
  psql "$dsn" -tAF'|' -c "
    select (select count(*) from contents)  as icerik,
           (select count(*) from contents where status='yayinda') as yayinda,
           (select count(*) from feeds)     as kaynak,
           (select count(*) from users)     as kullanici,
           (select count(*) from schema_migrations) as migration"

  echo
  echo "Sonraki adım:"
  echo "  bash deploy/scripts/bitcosar-veri-tasi.sh --yukle $dosya"
}

yukle() {
  local dosya="$1"
  [ -f "$dosya" ] || { echo "HATA: $dosya yok" >&2; exit 1; }

  echo "==> hedef küme kontrol ediliyor: namespace $NS"
  local pod
  pod=$("$KUBECTL" -n "$NS" get pod -l app=bitcosar-postgres -o name 2>/dev/null | head -1)
  [ -n "$pod" ] || {
    echo "HATA: $NS içinde postgres pod'u yok. Önce manifestleri sevk edin:" >&2
    echo "  NAMESPACE=$NS bash deploy/scripts/k8s-sevk.sh sha-XXXXXXX" >&2
    exit 1; }
  echo "    pod: $pod"

  # GÜVENLİK KAPISI: yanlış namespace'e yükleme veri kaybıdır.
  if [ "$NS" = "theblock" ]; then
    echo "HATA: hedef 'theblock' — orada BAŞKA bir sitenin verisi var." >&2
    echo "      NAMESPACE=bitcosar-site olmalı." >&2
    exit 1
  fi

  echo
  echo "==> hedefteki mevcut veri"
  "$KUBECTL" -n "$NS" exec "$pod" -- psql -U theblock -d theblock -tAc \
    "select coalesce((select count(*) from contents),0)" 2>/dev/null || echo "    (boş/yeni)"

  echo
  read -r -p "Yükleme hedefteki veriyi DEĞİŞTİRİR. Devam? (evet/hayir) " onay
  [ "$onay" = "evet" ] || { echo "iptal edildi"; exit 0; }

  echo
  echo "==> yükleniyor"
  "$KUBECTL" -n "$NS" cp "$dosya" "${pod#pod/}:/tmp/gecis.dump" || exit 1
  "$KUBECTL" -n "$NS" exec "$pod" -- pg_restore \
    --username=theblock --dbname=theblock \
    --clean --if-exists --no-owner --no-acl --exit-on-error \
    /tmp/gecis.dump || {
      echo "HATA: geri yükleme başarısız. Hedef veritabanı yarım kalmış olabilir." >&2
      exit 1; }
  "$KUBECTL" -n "$NS" exec "$pod" -- rm -f /tmp/gecis.dump

  echo
  echo "==> hedef özeti (kaynakla karşılaştırın)"
  "$KUBECTL" -n "$NS" exec "$pod" -- psql -U theblock -d theblock -tAF'|' -c "
    select (select count(*) from contents)  as icerik,
           (select count(*) from contents where status='yayinda') as yayinda,
           (select count(*) from feeds)     as kaynak,
           (select count(*) from users)     as kullanici,
           (select count(*) from schema_migrations) as migration"
}

case "${1:-}" in
  --yedek-al) yedek_al ;;
  --yukle)    [ $# -ge 2 ] || kullanim; yukle "$2" ;;
  *)          kullanim ;;
esac
