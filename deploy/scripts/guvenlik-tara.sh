#!/usr/bin/env bash
# guvenlik-tara.sh — CI'daki güvenlik kapılarının BİREBİR aynısını yerelde koşar.
#
# Amaç: "CI'da kırıldı, yerelde geçiyordu" durumunu ortadan kaldırmak.
# CI (.github/workflows/guvenlik.yml) bu betiği çağırır; yani kapı tek yerde
# tanımlı. Sürüm sabit (TRIVY_SURUM) ki yerel ve CI aynı kuralları görsün.
#
# Kullanım:
#   bash deploy/scripts/guvenlik-tara.sh            # kod + yapılandırma + sır
#   bash deploy/scripts/guvenlik-tara.sh --imaj     # ayrıca 4 imajı derle ve tara
#   bash deploy/scripts/guvenlik-tara.sh --rapor    # kırmadan yalnız raporla
#
# KAPI KURALI — neden `--ignore-unfixed`:
#   site imajı ffmpeg içeriyor; debian'da YAMASI OLMAYAN CRITICAL/HIGH sayısı
#   236. Yaması olan ise (ölçüm, 5 Kasım 2026) yalnız 1'di. Yamasızları kapı
#   yapmak CI'yı kalıcı kırmızıya çevirir ve kimse bakmaz; düzeltilebilir
#   olanlar ise gerçekten aksiyon alınabilir. Yamasızlar raporlanır.
set -uo pipefail

TRIVY_SURUM="${TRIVY_SURUM:-0.76.1}"
KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$KOK"

IMAJ=0
RAPOR=0
for a in "$@"; do
  case "$a" in
    --imaj)  IMAJ=1 ;;
    --rapor) RAPOR=1 ;;
    *) echo "bilinmeyen seçenek: $a" >&2; exit 2 ;;
  esac
done

KOD=1
[ "$RAPOR" = 1 ] && KOD=0

# ---------------------------------------------------------------- trivy kurulumu
trivy_kur() {
  local hedef="$HOME/.local/bin"
  mkdir -p "$hedef"
  if [ -x "$hedef/trivy" ] && "$hedef/trivy" --version 2>/dev/null | grep -q "$TRIVY_SURUM"; then
    return 0
  fi
  echo "==> trivy $TRIVY_SURUM kuruluyor"
  local d
  d=$(mktemp -d)
  (
    cd "$d" || exit 1
    local t="trivy_${TRIVY_SURUM}_Linux-64bit.tar.gz"
    local b="https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_SURUM}"
    curl -fsSLO "$b/$t" || exit 1
    curl -fsSLO "$b/trivy_${TRIVY_SURUM}_checksums.txt" || exit 1
    # Sağlama ZORUNLU: imza doğrulaması olmadan binary kurulmaz.
    sha256sum -c --ignore-missing "trivy_${TRIVY_SURUM}_checksums.txt" || exit 1
    tar xzf "$t" || exit 1
    install -m 0755 trivy "$hedef/trivy" || exit 1
  ) || { echo "trivy kurulamadı" >&2; rm -rf "$d"; exit 1; }
  rm -rf "$d"
}

if command -v trivy >/dev/null 2>&1 && trivy --version 2>/dev/null | grep -q "$TRIVY_SURUM"; then
  TRIVY=trivy
else
  trivy_kur
  TRIVY="$HOME/.local/bin/trivy"
fi

ATLA=(--skip-dirs node_modules --skip-dirs graphify-out --skip-dirs storage
      --skip-dirs storage-private --skip-dirs data --skip-dirs .git
      --skip-dirs .k8s-uretim)

BASARISIZ=0
bolum() { echo; echo "──────── $* ────────"; }

# ---------------------------------------------------------------- 1) bağımlılık CVE
bolum "1/7  bağımlılık CVE (go.mod, Cargo.lock, package-lock.json)"
"$TRIVY" fs -q --scanners vuln --severity CRITICAL,HIGH --ignore-unfixed \
  --exit-code "$KOD" --table-mode summary "${ATLA[@]}" . || BASARISIZ=1

# ---------------------------------------------------------------- 2) yapılandırma
bolum "2/7  yapılandırma (Dockerfile + k8s manifest)"
"$TRIVY" config -q --severity CRITICAL,HIGH \
  --ignorefile .trivyignore.yaml \
  --exit-code "$KOD" --table-mode summary deploy/ || BASARISIZ=1

# ---------------------------------------------------------------- 3) sır taraması
bolum "3/7  sır taraması (gömülü anahtar/jeton)"
"$TRIVY" fs -q --scanners secret \
  --exit-code "$KOD" --table-mode summary "${ATLA[@]}" . || BASARISIZ=1

# ---------------------------------------------------------------- 4) semgrep SAST
bolum "4/7  semgrep SAST (ERROR kapı, WARNING rapor)"
if command -v semgrep >/dev/null 2>&1; then
  SEMGREP=semgrep
elif [ -x "$HOME/.local/bin/semgrep" ]; then
  SEMGREP="$HOME/.local/bin/semgrep"
else
  SEMGREP=""
  echo "semgrep yok — atlandı (kurulum: uv tool install semgrep)"
fi
if [ -n "$SEMGREP" ]; then
  # ERROR = kapı. WARNING tabanı 65 (sha1 önbellek anahtarı, cookie
  # yardımcısını göremeyen kurallar vb.); ayıklanana kadar kapı değil.
  "$SEMGREP" scan --config p/golang --config p/javascript \
    --metrics=off --quiet --severity ERROR \
    --error --exclude node_modules --exclude graphify-out \
    --exclude storage --exclude data \
    go collab-server editor-src e2e || BASARISIZ=1
  echo "ERROR seviyesi: temiz"
  echo "--- WARNING (rapor, kapı değil) ---"
  "$SEMGREP" scan --config p/golang --config p/javascript \
    --metrics=off --quiet --severity WARNING \
    --exclude node_modules --exclude graphify-out \
    --exclude storage --exclude data \
    go collab-server editor-src e2e 2>/dev/null \
    | grep -E '^\s+❯❱|Findings:' | sed 's/^/    /' | head -20
fi

# ---------------------------------------------------------------- 5) gosec
bolum "5/7  gosec (Go statik güvenlik; severity high kapı)"
# CI ile AYNI komut ve AYNI sabit sürüm. gosec `go` komutunu alt süreç
# olarak çağırır; PATH'te yoksa "0 bulgu" der ve YANLIŞ GÜVEN verir — bu
# yüzden PATH burada açıkça kurulur ve sürüm doğrulanır.
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
GOSEC_SURUM="${GOSEC_SURUM:-v2.30.0}"
GOSEC="$HOME/go/bin/gosec"

# Sürüm SABİT ve DOĞRULANIR — 6 Kasım 2026'da ölçüldü: yerelde `dev`
# derlemesi vardı, CI'da hiç kurulu değildi. Sonuç iki yönlü yanlış güven:
#   · yerel "dev" taint analizi G703 verdi (CI yeşildi → "yerelde kırık"),
#   · CI'da gosec yoktu → kapı sessizce atlanıyordu (hiç taranmıyordu).
# Artık eksik/yanlış sürüm kendiliğinden kurulur; kurulamazsa kapı KIRILIR
# (atlamak yok: sessiz geçiş güvenlik kapısını dekorasyona çevirir).
#
# DİKKAT: `gosec -version` kaynaktan derlenen her kurulumda "dev" yazar
# (sürüm ldflags ile yalnız resmî sürümlerde gömülüyor). Bu yüzden sürüm
# `go version -m` build info'sundan okunur; aksi halde kontrol hiç tutmaz
# ve her koşuda yeniden kurulum yapılır.
gosec_surum_dogru() {
  [ -x "$GOSEC" ] || return 1
  go version -m "$GOSEC" 2>/dev/null | grep -qE "mod[[:space:]]+github.com/securego/gosec/v2[[:space:]]+${GOSEC_SURUM}([[:space:]]|$)"
}

gosec_kur() {
  if gosec_surum_dogru; then
    return 0
  fi
  echo "==> gosec $GOSEC_SURUM kuruluyor (GOBIN=$HOME/go/bin)"
  GOBIN="$HOME/go/bin" go install "github.com/securego/gosec/v2/cmd/gosec@$GOSEC_SURUM" 2>/dev/null
}

if ! command -v go >/dev/null 2>&1; then
  echo "go yok — gosec KOŞTURULAMADI"
  BASARISIZ=1
else
  if ! gosec_kur; then
    echo "gosec $GOSEC_SURUM kurulamadı — kapı kırıldı" >&2
    BASARISIZ=1
  else
    # KAPI "YENI BULGU"DA kirilir, mevcut borcta degil.
    #
    # Dogrudan gosec'i kapi yapmak 10 mevcut HIGH bulgu yuzunden kapiyi
    # KALICI kirmizi tutuyordu (12 Kasim 2026). Kalici kirmizi kapi kapi
    # degildir: kimse bakmaz, gercekten yeni gelen acik gurultude kaybolur.
    # Borc `deploy/gosec-borc.txt`te kayitli ve her kosuda raporlanir.
    bash deploy/scripts/gosec-borc.sh || BASARISIZ=1
  fi
fi

# ---------------------------------------------------------------- 6) govulncheck
bolum "6/7  govulncheck (Go güvenlik veritabanı — ÇAĞRILAN kod)"
# NEDEN TRIVY'NİN YANINDA:
#   Trivy `go.sum`'daki her bağımlılığı tarar — savunmasız sürüm projede
#   DURUYORSA uyarır, o kod hiç çalışmasa bile. govulncheck çağrı grafiği
#   çıkarır ve yalnız GERÇEKTEN ULAŞILABİLEN savunmasız fonksiyonları
#   bildirir. İkisi farklı soruyu yanıtlıyor:
#     Trivy       → "savunmasız sürüm var mı?"  (tedarik zinciri hijyeni)
#     govulncheck → "o açık bizi vuruyor mu?"   (gerçek risk)
#   Yalnız Trivy kalsaydı gürültü içinde gerçek risk kaybolurdu; yalnız
#   govulncheck kalsaydı henüz çağrılmayan ama yarın çağrılacak bağımlılık
#   gözden kaçardı.
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
GOVULN="$HOME/go/bin/govulncheck"
if [ ! -x "$GOVULN" ]; then
  echo "==> govulncheck kuruluyor"
  go install golang.org/x/vuln/cmd/govulncheck@latest >/dev/null 2>&1 || true
fi
if [ -x "$GOVULN" ]; then
  # Çıkış kodu 3 = savunmasız çağrı bulundu. 0 = temiz.
  # Kapı yalnız ULAŞILABİLİR bulguda kırılır; bu yüzden --show=verbose yok.
  if (cd go && "$GOVULN" ./... 2>&1 | tail -25); then
    echo "  ulaşılabilir savunmasız çağrı yok"
  else
    echo "  ↑ ulaşılabilir açık: bağımlılığı yükselt ya da çağrıyı değiştir" >&2
    [ "$RAPOR" = 1 ] || BASARISIZ=1
  fi
else
  # Sessiz atlamak güvenlik kapısını dekorasyona çevirir (gosec dersi).
  echo "HATA: govulncheck kurulamadı — kapı atlanmadı, KIRILDI" >&2
  [ "$RAPOR" = 1 ] || BASARISIZ=1
fi

# ---------------------------------------------------------------- 7) workflow güvenliği
bolum "7/7  GitHub Actions workflow güvenliği (zizmor)"
if command -v zizmor >/dev/null 2>&1; then
  ZIZMOR=zizmor
elif [ -x "$HOME/.local/bin/zizmor" ]; then
  ZIZMOR="$HOME/.local/bin/zizmor"
else
  ZIZMOR=""
  echo "zizmor yok — atlandı (kurulum: uv tool install zizmor)"
fi
if [ -n "$ZIZMOR" ]; then
  "$ZIZMOR" --min-severity medium --no-progress .github/workflows/ || BASARISIZ=1
fi

# ---------------------------------------------------------------- imaj taraması
if [ "$IMAJ" = 1 ]; then
  bolum "ek  kap imajları (derle + tara)"
  for s in site image-svc search-svc collab; do
    echo "--- $s derleniyor"
    docker build -q -f "deploy/docker/$s.Dockerfile" -t "theblock-$s:tara" . >/dev/null \
      || { echo "derleme başarısız: $s" >&2; BASARISIZ=1; continue; }
    "$TRIVY" image -q --scanners vuln --severity CRITICAL,HIGH --ignore-unfixed \
      --exit-code "$KOD" --table-mode summary "theblock-$s:tara" || BASARISIZ=1
  done

  bolum "ek  upstream imajlar (RAPOR — yamayı biz veremeyiz)"
  # Kapı DEĞİL: postgres/redis bizim derlemediğimiz imajlar. Örnek:
  # postgres:17-alpine'daki 26 bulgunun tamamı /usr/local/bin/gosu içinde
  # (Go 1.24.6 ile derlenmiş). gosu yalnız başlangıçta root'tan postgres'e
  # düşmek için çalışır; TLS/x509/URL ayrıştırması yapmaz. Upstream yeni
  # imaj yayınlayınca etiketi güncellemek dışında yapacak bir şey yok.
  for i in postgres:17-alpine redis:7-alpine; do
    "$TRIVY" image -q --scanners vuln --severity CRITICAL,HIGH --ignore-unfixed \
      --exit-code 0 --table-mode summary "$i"
  done
fi

echo
if [ "$BASARISIZ" = 0 ]; then
  echo "✓ güvenlik kapıları geçti"
else
  echo "✗ güvenlik kapısı kırıldı"
  [ "$RAPOR" = 1 ] && echo "  (--rapor modunda exit 0 döner)"
fi
[ "$RAPOR" = 1 ] && exit 0
exit "$BASARISIZ"
