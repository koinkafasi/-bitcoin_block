#!/bin/bash
# theblock — duman testi (smoke): canlı/izole sunucu sağlık kontrolü
# Kullanım: bash deploy/smoke.sh [taban-url]
#
# İki katman vardır (bkz. specs/09-eksikler-ve-yapilacaklar.md E-4):
#
#   1) ALTYAPI  — her geçerli kurulumda, **boş veritabanıyla bile** doğru
#      olması gereken iddialar: durum kodları, yönlendirmeler, yapısal
#      işaretler, güvenlik kapıları. Bunlar CI kapısıdır; çıkış kodunu
#      belirler ve kırmızıysa gerçekten bir şey bozuktur.
#
#   2) İÇERİK   — yayın verisine veya editörün açtığı modüllere bağlı
#      iddialar (hikaye şeridi, slider, makale varlığı...). Bunlar
#      bilgilendiricidir, UYARI üretir, **CI'yı kırmaz**. Editör bir modülü
#      kapattığında veya veritabanı boş olduğunda derleme kırmızıya
#      dönmemelidir.
set -u
BASE="${1:-https://theblock.com.tr}"
FAIL=0
WARN=0

# --- katman 1: altyapı (CI kapısı) ---
chk() { # chk <yol> <beklenen-kod> <etiket>
  code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 20 -H 'X-Tb-Smoke: 1' "$BASE$1")
  if [ "$code" = "$2" ]; then printf "  ok   %-38s %s  %s\n" "$1" "$code" "$3"
  else printf "  FAIL %-38s %s (beklenen %s) %s\n" "$1" "$code" "$2" "$3"; FAIL=$((FAIL+1)); fi
}
contains() { # contains <yol> <desen> <etiket>
  if curl -s --max-time 20 -H 'X-Tb-Smoke: 1' "$BASE$1" | grep -q "$2"; then printf "  ok   %-38s içerik: %s\n" "$1" "$2"
  else printf "  FAIL %-38s içerik yok: %s\n" "$1" "$2"; FAIL=$((FAIL+1)); fi
}

# --- katman 2: içerik (bilgilendirici, CI'yı kırmaz) ---
chk_opt() { # chk_opt <yol> <beklenen-kod> <etiket>
  code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 20 -H 'X-Tb-Smoke: 1' "$BASE$1")
  if [ "$code" = "$2" ]; then printf "  ok   %-38s %s  %s\n" "$1" "$code" "$3"
  else printf "  UYARI %-37s %s (beklenen %s) %s\n" "$1" "$code" "$2" "$3"; WARN=$((WARN+1)); fi
}
contains_opt() { # contains_opt <yol> <desen> <etiket>
  if curl -s --max-time 20 -H 'X-Tb-Smoke: 1' "$BASE$1" | grep -q "$2"; then printf "  ok   %-38s içerik: %s\n" "$1" "$2"
  else printf "  UYARI %-37s içerik yok: %s  (%s)\n" "$1" "$2" "$3"; WARN=$((WARN+1)); fi
}

echo "=== theblock smoke :: $BASE ==="
echo "[public]"
chk "/" 200 "ana sayfa"
chk "/arama?q=haber" 200 "arama"
chk "/arsiv" 200 "arşiv"
chk "/sitemap.xml" 200 "sitemap"
chk "/sitemap-news.xml" 200 "news sitemap"
chk "/robots.txt" 200 "robots"
chk "/rss.xml" 200 "rss"
chk "/hava-durumu" 200 "hava"
chk "/namaz-vakitleri" 200 "namaz"
chk "/imsakiye" 200 "imsakiye"
chk "/borsa" 200 "borsa"
chk "/api/borsa" 200 "borsa API"
chk "/fikstur" 200 "fikstür"
chk "/astroloji" 200 "astroloji"
chk "/etkinlikler" 200 "etkinlikler"
chk "/anketler" 200 "anketler"
chk "/gazete-mansetleri" 200 "gazete manşetleri"
chk "/sinemalar" 200 "sinemalar"
chk "/resmi-ilanlar" 200 "resmi ilanlar"
# Kurumsal sayfalar KÖK adreste servis edilir (`/gizlilik-politikasi`),
# `/sayfa/...` ise kalıcı yönlendirme bırakır — bkz. internal/httpapi/footer.go
# ve contact_test.go. Bu betik eskiden `/sayfa/...` yollarını 200 bekliyordu
# ve sayfalar çalıştığı hâlde 3 HATA üretiyordu (ölçüm: 5 Kasım 2026).
#
# İki iddia da değerli: kanonik adres 200 vermeli, ESKİ adres de 301
# vermeye devam etmeli — dışarıdan gelen eski bağlantılar ve arama
# sonuçları o yönlendirmeye bağlı.
chk "/gizlilik-politikasi" 200 "gizlilik & KVKK"
chk "/kullanim-kosullari" 200 "kullanım koşulları"
chk "/cerez-politikasi" 200 "çerez politikası"
chk "/sayfa/gizlilik-politikasi" 301 "eski yol → kanonik (301 korunmalı)"
chk "/sayfa/kullanim-kosullari" 301 "eski yol → kanonik (301 korunmalı)"
chk "/sayfa/cerez-politikasi" 301 "eski yol → kanonik (301 korunmalı)"
chk "/olmayan-sayfa-404-testi" 404 "404 sayfası (çökmemeli)"
chk "/arsiv/dizin" 410 "bot tuzağı (honeypot)"
chk "/akis" 200 "akış modu"
chk "/akis?mod=manset" 200 "akış manşet modu"
chk "/?kategori=teknoloji" 200 "anasayfa kategori filtresi"
chk "/api/akis" 200 "akış API"
chk "/nobetci-eczaneler" 200 "eczane"
chk "/bulten" 200 "bülten"
chk "/ping" 200 "günlük özet arşivi"
chk "/asistan" 200 "AI asistan"
# /hesabim/* yolları requireMember ile korunur: oturumsuz istek 302 verir.
# Beklentinin 200 olması hatalıydı; 302 beklemek üye kapısını da doğrular.
chk "/hesabim/rozetler" 302 "başarımlar (üye kapısı → girişe yönlendirir)"
chk "/static/manifest.webmanifest" 200 "PWA manifest"
chk "/api/trends" 200 "trends API"
chk "/api/digest" 200 "digest API"
chk "/api/hava/ankara" 200 "hava API"

echo "[yapı]"
contains "/" "ld+json" "ana sayfa JSON-LD"
contains "/sitemap.xml" "xhtml:link" "sitemap hreflang"
contains "/robots.txt" "Disallow: /arsiv/dizin" "robots bot tuzağı"

echo "[pwa]"
for f in icon-192.png icon-512.png icon-maskable-192.png icon-maskable-512.png; do
  chk "/static/$f" 200 "ikon $f"
done

echo "[güvenlik]"
code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 -X POST "$BASE/api/bookmarks/1")
if [ "$code" = "401" ]; then echo "  ok   oturumsuz bookmark → 401"; else echo "  FAIL bookmark 401 beklenirken $code"; FAIL=$((FAIL+1)); fi
code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$BASE/admin/giris")
if [ "$code" = "200" ] || [ "$code" = "302" ]; then echo "  ok   panel giriş sayfası → $code"; else echo "  FAIL panel $code"; FAIL=$((FAIL+1)); fi
code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$BASE/admin/tasarim")
if [ "$code" = "302" ] || [ "$code" = "200" ]; then echo "  ok   sayfa tasarımcısı (auth) → $code"; else echo "  FAIL /admin/tasarim $code"; FAIL=$((FAIL+1)); fi

# --------------------------------------------------------------------------
# Buradan sonrası bilgilendiricidir: yayın verisine ve editörün açtığı
# modüllere bağlıdır. Boş veritabanlı taze kurulumda uyarı vermesi normaldir.
# --------------------------------------------------------------------------
echo "[içerik — bilgilendirici, CI'yı kırmaz]"
contains_opt "/" "hero-slider" "modül kapalı olabilir"
contains_opt "/" "ust-manset" "modül kapalı olabilir"
contains_opt "/" "home-grid" "yan sütun kapalı olabilir"
contains_opt "/" "hikayeData" "hikaye şeridi boş olabilir"

ART=$(curl -s --max-time 20 "$BASE/" | grep -o 'href="/[a-z0-9-]\+-[0-9]\+"' | head -1 | sed 's/href="//; s/"//')
if [ -n "$ART" ]; then
  chk_opt "$ART" 200 "makale: $ART"
  contains_opt "$ART" "NewsArticle" "makale JSON-LD"
  contains_opt "$ART" "breadcrumb" "makale breadcrumb"
  contains_opt "$ART" "dk okuma" "makale okuma süresi"
else
  echo "  UYARI ana sayfada makale linki yok (veritabanı boş olabilir)"; WARN=$((WARN+1))
fi

echo "=== SONUÇ: $([ $FAIL -eq 0 ] && echo 'TÜMÜ GEÇTİ ✅' || echo "$FAIL HATA ❌")$([ $WARN -gt 0 ] && echo " · $WARN uyarı (bilgilendirici)") ==="
exit $FAIL
