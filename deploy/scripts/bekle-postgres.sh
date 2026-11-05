#!/usr/bin/env bash
# Faz 209 — açılış yarışı kapısı.
#
# ÖLÇÜM (4 Kasım 2026, makine açılışı):
#   theblock-site.service 10:48:45'te başladı, Postgres konteyneri henüz
#   "the database system is not yet accepting connections" durumundaydı.
#   Site 23 saniye boyunca 11 kez açılıp öldü (migration bağlantısı
#   kurulamadı), ancak 10:49:08'de ayakta kalabildi. O pencerede
#   theblock.com.tr Cloudflare üzerinden hata verdi.
#
# ÇÖZÜM: uygulama başlamadan önce Postgres'in YALNIZ portu değil, sorgu
# kabul etmesi beklenir. `pg_isready` tam olarak bunu ölçer; TCP connect
# yeterli değildir (container açık, veritabanı hâlâ recovery'de olabilir).
#
# Kullanım: bekle-postgres.sh [saniye]   (varsayılan 180)
# Çıkış 0 = hazır, 1 = süre doldu (systemd yeniden dener).
set -u

LIMIT="${1:-180}"
KONTEYNER="${THEBLOCK_PG_KONTEYNER:-theblock-postgres-1}"
KULLANICI="${THEBLOCK_PG_KULLANICI:-theblock}"
VERITABANI="${THEBLOCK_PG_VERITABANI:-theblock}"

basla=$(date +%s)
deneme=0
while :; do
  deneme=$((deneme + 1))

  # 1) Konteyner ayakta mı?
  if docker inspect -f '{{.State.Running}}' "$KONTEYNER" 2>/dev/null | grep -q true; then
    # 2) Postgres bağlantı kabul ediyor mu? (port açık olsa da recovery olabilir)
    if docker exec "$KONTEYNER" pg_isready -U "$KULLANICI" -d "$VERITABANI" >/dev/null 2>&1; then
      # 3) Gerçekten sorgu çalışıyor mu? (son kapı: migration bunu yapacak)
      if docker exec "$KONTEYNER" psql -U "$KULLANICI" -d "$VERITABANI" -Atq -c 'select 1' >/dev/null 2>&1; then
        echo "bekle-postgres: hazır (${deneme}. deneme, $(( $(date +%s) - basla )) sn)"
        exit 0
      fi
    fi
  fi

  gecen=$(( $(date +%s) - basla ))
  if [ "$gecen" -ge "$LIMIT" ]; then
    echo "bekle-postgres: ${LIMIT} sn içinde hazır olmadı (konteyner=$KONTEYNER)" >&2
    docker ps -a --filter "name=$KONTEYNER" --format '  {{.Names}} {{.Status}}' >&2 || true
    exit 1
  fi
  sleep 2
done
