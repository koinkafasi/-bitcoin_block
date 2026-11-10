#!/usr/bin/env python3
"""Tabler SVG sprite üretici (vendor aracı — çalışma anında dış bağımlılık yok).
NEDEN: sprite elle seçilmiş 41 ikondan oluşuyordu ve kodda kullanılan 6 ikon
sprite'ta yoktu (ti-bell, ti-layout-grid, ti-layout-sidebar-left-collapse…).
Olmayan ikon SESSİZCE boş kutu olarak basılıyor — ne hata ne uyarı. Bu araç
kütüphaneyi koddan türetir: kaynak kodda geçen adlar + küratörlü geniş liste.
Kullanım:
    python3 deploy/scripts/ikon-uret.py --kaynak /tmp/tabler-src/package \\
        --cikti go/internal/webui/static/vendor
Lisans: Tabler MIT (bkz. vendor/IKON-LISANS.txt). Üretilen sprite da MIT.
"""
import argparse
import glob
import io
import os
import re
import sys
# Küratörlü liste: panel + haber sitesi + finans/hava/sosyal için makul
# kapsama. Kodda geçen adlar bunlara EKLENİR; eksik kalırsa test yakalar.
KURATORLU = """
activity adjustments alert-circle alert-triangle api-app archive arrow-back arrow-down
arrow-left arrow-narrow-down arrow-narrow-left arrow-narrow-right arrow-narrow-up
arrow-right arrow-up arrows-exchange arrows-sort arrows-up-down article aspect-ratio
at atom award baby-carriage backpack badge backspace ban bandage barcode battery
bed bell bell-off bell-ringing bike binary currency-bitcoin blockquote bold book bookmark
bookmark-off books border-all box box-multiple braces brackets brand-adobe
brand-android brand-apple brand-facebook brand-github brand-google brand-instagram
brand-linkedin brand-telegram brand-tiktok brand-whatsapp brand-x brand-youtube
brightness brightness-down brightness-up browser browser-check brush bucket
bug building building-arch building-bank building-factory building-store
buildings bulb bulb-off calendar calendar-event calendar-month calendar-time
calendar-week camera camera-off camera-plus cancel shopping-cart
cash chart-area chart-arcs chart-bar chart-candle chart-donut chart-dots
chart-histogram chart-infographic chart-line chart-pie chart-treemap check
 checkbox checklist chevron-down chevron-left chevron-right
chevron-up circle circle-check circle-dashed circle-dot circle-minus circle-plus
circle-x clipboard clipboard-check clipboard-copy clipboard-list clipboard-text
clock clock-edit clock-hour-4 clock-hour-9 cloud cloud-download cloud-off
cloud-rain cloud-snow cloud-storm cloud-upload code color-swatch columns
command compass components confetti container contrast cookie copy copyright
cpu credit-card crop crosshair crown current-location dashboard database
database-export temperature device-desktop device-mobile device-tablet diamond
dice direction discount divide dna dog door download drag-drop droplet edit
egg emergency-bed mood-happy mood-neutral  engine
 eraser error-404 exchange eye eye-off face-mask feather file file-alert
file-analytics file-certificate file-chart file-check file-code file-description
file-download file-euro file-export file-import file-info file-invoice
file-music file-pencil file-plus file-report file-search file-settings
file-shredder file-text file-upload file-x files filter filter-off fingerprint
 flame  flip-horizontal flip-vertical focus folder
folder-open folder-plus folders forbid git-fork forms frame friends gauge
ghost gift git-branch git-commit git-merge git-pull-request glass glass-full
globe globe-off gps graph grid-dots grip-vertical growth hand-stop hash hdr
heading headphones heart heart-broken heart-off help help-circle hexagon highlight
history home home-2 hourglass icons icons id id ice-cream image-in-picture
inbox inbox-off indent-decrease indent-increase infinity info-square inner-shadow-top
italic json key keyboard language lasso square-toggle layout layout-align-bottom
layout-align-center layout-align-left layout-align-middle layout-align-right
layout-align-top layout-board layout-board-split layout-bottombar layout-cards
layout-collage layout-columns layout-dashboard layout-distribute-horizontal
layout-distribute-vertical layout-grid layout-grid-add layout-kanban
layout-list layout-navbar layout-off layout-rows layout-sidebar layout-sidebar
layout-sidebar-left-collapse layout-sidebar-left-expand layout-sidebar-right
layout-sidebar-right-collapse leaf letter-case letter-spacing library
license lifebuoy line link link-off list list-check list-details list-numbers
list-search live-photo live-view location lock lock-open lock-open-off
login logout macro magnet mail mail-forward mail-opened mail-fast mail-x
man menu menu-2 message message-2 message-2-off message-chatbot message-circle
 message-dots message-off message-plus messages meteor
microphone microphone-off microscope  minus minus-vertical device-mobile
moneybag mood-crazy-happy mood-smile moon moon-stars apps mouse movie
music music-off navigation north-star notebook notes notification numbers
octagon bolt-off omega package package-export package-import paint palette
paper-bag paperclip parking parking password align-left parentheses pencil pencil
pennant percentage percentage perspective phone phone-call phone-off photo
photo-off photo-plus picture-in-picture pig-money pill pin pin-end pin-invoke
planet plant player-play play-card playlist plus point pointer plug plug-connected
point-off pokeball polaroid polygon power pray progress progress-alert
puzzle qrcode question-mark quote radio  receipt recharging refresh
refresh-alert refresh-dot registered repeat repeat-off replace resize rocket
rotate rotate-clockwise route router rss ruler-2 run sandbox scale scan
school scissors screenshot script search search-off selector section send
seo server server-2 settings settings-2 shadow shape share share-2 share-3
shield shield-check shield-lock shield-x shirt shoe shopping-bag shopping-cart
shopping-cart-off arrows-shuffle  sign-left signal-h signal-4g signature sitemap
skull slash slice device-mobile cloud-fog sort-ascending sort-descending sos speakerphone
 play-basketball square square-check square-dot square-plus
square-x stack stack-2 stack-3 star star-off stars layout-bottombar steering-wheel
stethoscope sticker stopwatch storm news ruler-measure strikethrough subscript
sum sun sun-high sun-low sunrise sunset superscript switch-2 switch table
table-alias table-export table-import table-off tag tags target tax telescope
template tent terminal test-pipe text-caption text-direction-ltr text-plus
text-recognition text-size text-wrap texture thermometer thumb-down thumb-up
ticket tie tilde clock-hour-4 timeline toggle-left toggle-right tool trash
trash-off trending-down trending-up trending-up triangle triangle-inverted trophy
truck  typography underline unlink upload urgent user user-check
user-circle user-exclamation user-heart user-minus user-off user-plus
users users-group vaccine variable vector video video-off viewport-narrow
viewport-wide virus volume volume-2 volume-3 volume-off wallet wallet
wallpaper wand wave-sine webhook weight wheel wheelchair wifi wifi-off wind
window window-maximize window-minimize world world-off writing zoom-in zoom-out
zoom-pan zoom-reset zoom-question brand-google-analytics brand-vercel
coin currency-bitcoin currency-dollar currency-euro currency-lira currency-pound
currency-ruble currency-yen currency-taka exchange building-hospital
stethoscope dog paw map-pin map-pins map route road traffic-lights
elevator escalator parking-circle charging-pile
"""
def topla_kod_adlari(kok):
    """Kaynak kodda geçen ikon adları: statik `#ti-x` + dinamik `Icon: "x"`.
    NOT: desen `#ti-` ile sınırlı. Daha gevşek bir `ti-` deseni normal
    metinlerde yanlış eşleşiyordu (ör. "benzerlik-testi-yayinda-int" içindeki
    `testi-yayinda-int` bir ikon sanılıyordu).
    """
    adlar = set()
    desenler = [
        r'Icon:\s*"([a-z0-9-]+)"',
        r'Ikon:\s*"([a-z0-9-]+)"',
        r'PanoIkon\("([a-z0-9-]+)"\)',
        r'#ti-([a-z0-9-]+)',
        r'#tif-([a-z0-9-]+)',
        # Editör araç çubuğu: etiket olarak "'@simge'" yazılıyor
        # (editor-src/src/editor.js → static/editor.js). Bundle taranmaz,
        # KAYNAK taranır: bundle minify'lı ve ad orada da geçse okunamaz.
        r"'@([a-z0-9-]+)'",
    ]
    kokler = [os.path.join(kok, 'go/internal/**/*'),
              os.path.join(kok, 'editor-src/src/**/*')]
    for p in [x for k in kokler for x in glob.glob(k, recursive=True)]:
        if not p.endswith(('.go', '.templ', '.html', '.js')):
            continue
        if '/node_modules/' in p.replace(os.sep, '/'):
            continue
        try:
            s = io.open(p, encoding='utf-8', errors='ignore').read()
        except OSError:
            continue
        for d in desenler:
            adlar |= set(re.findall(d, s))
    adlar.discard('')
    return adlar
def svg_oku(yol):
    """Tabler SVG dosyasından içeriği çıkarır (dış <svg> atılır)."""
    s = io.open(yol, encoding='utf-8').read()
    i = s.find('>', s.find('<svg'))
    j = s.rfind('</svg>')
    if i < 0 or j < 0:
        return None
    ic = s[i + 1:j]
    # Yorum ve gereksiz boşlukları temizle
    ic = re.sub(r'<!--.*?-->', '', ic, flags=re.S)
    ic = re.sub(r'\s+', ' ', ic).strip()
    return ic
def uret(kaynak, cikti, dolu_dahil=True):
    outline_dir = os.path.join(kaynak, 'icons', 'outline')
    filled_dir = os.path.join(kaynak, 'icons', 'filled')
    mevcut = {os.path.splitext(os.path.basename(p))[0]
              for p in glob.glob(os.path.join(outline_dir, '*.svg'))}
    mevcut_dolu = {os.path.splitext(os.path.basename(p))[0]
                   for p in glob.glob(os.path.join(filled_dir, '*.svg'))}
    istenen = set(KURATORLU.split()) | topla_kod_adlari('/home/alper/theblock')
    bulunan = sorted(istenen & mevcut)
    eksik = sorted(istenen - mevcut)
    if eksik:
        print('UYARI: tabler\'de bulunmayan adlar atlandı: %s' % ' '.join(eksik),
              file=sys.stderr)
    parcalar = ['<svg xmlns="http://www.w3.org/2000/svg" style="display:none" '
                'aria-hidden="true">']
    parcalar.append('<!-- theblock ikon sprite\'i — Tabler Icons (MIT). '
                    'Üretici: deploy/scripts/ikon-uret.py. ELLE DÜZENLEMEYİN. -->')
    n = 0
    for ad in bulunan:
        ic = svg_oku(os.path.join(outline_dir, ad + '.svg'))
        if not ic:
            continue
        parcalar.append(
            '<symbol id="ti-%s" viewBox="0 0 24 24" fill="none" stroke="currentColor" '
            'stroke-width="2" stroke-linecap="round" stroke-linejoin="round">%s</symbol>'
            % (ad, ic))
        n += 1
    # Dolgulu varyantlar: marka/rozet gibi alanlarda gerekiyor (tif- öneki).
    if dolu_dahil:
        for ad in sorted(istenen & mevcut_dolu):
            ic = svg_oku(os.path.join(filled_dir, ad + '.svg'))
            if not ic:
                continue
            parcalar.append(
                '<symbol id="tif-%s" viewBox="0 0 24 24" fill="currentColor">%s</symbol>'
                % (ad, ic))
            n += 1
    parcalar.append('</svg>')
    hedef = os.path.join(cikti, 'tabler-icons.svg')
    io.open(hedef, 'w', encoding='utf-8').write('\n'.join(parcalar) + '\n')
    boyut = os.path.getsize(hedef)
    print('yazıldı: %s' % hedef)
    print('sembol: %d  boyut: %.1f KB' % (n, boyut / 1024))
    # Manifest: hangi ikonların bulunduğunu listeleyen düz metin (galeri ve
    # testler bunu okur; sprite'ı ayrıştırmaya gerek kalmaz).
    io.open(os.path.join(cikti, 'ikonlar.txt'), 'w', encoding='utf-8').write(
        '\n'.join(bulunan) + '\n')
    print('manifest: ikonlar.txt (%d ad)' % len(bulunan))
if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument('--kaynak', required=True, help='/yol/package (tabler npm)')
    ap.add_argument('--cikti', required=True, help='vendor dizini')
    ap.add_argument('--sadece-outline', action='store_true')
    a = ap.parse_args()
    uret(a.kaynak, a.cikti, dolu_dahil=not a.sadece_outline)
