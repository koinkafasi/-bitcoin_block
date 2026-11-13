/* cd-entegrasyon.js — üçüncü taraf kimliklerini ÇALIŞMA ANINDA yükler.
 *
 * NEDEN AYRI DOSYA, SATIR İÇİ DEĞİL:
 *   · CSP: sayfa başındaki satır içi blokların sha256'sı derleme/çalışma
 *     anında hesaplanıp başlığa yazılıyor. Panelden değişen bir kimlik
 *     o hash'i bozar ve Google Analytics sessizce ölürdü.
 *   · Güvenlik: kendi dosyamız `script-src 'self'` ile geçer; panel yalnız
 *     doğrulanmış bir KİMLİK taşır, kod değil.
 *
 * Yüklenenler (hepsi isteğe bağlı, kimlik boşsa hiç çalışmaz):
 *   data-ga4         G-XXXXXXXXXX  → gtag.js
 *   data-gtm         GTM-XXXXXXX   → gtm.js (GA4 ile birlikte olabilir)
 *   data-reklam-*    zone kimliği  → Coinzilla display + sticky
 *   data-cointraffic wkey          → Cointraffic
 *
 * Tüm yüklemeler `async` ve hata toleranslı: bir ağ çökse sayfa etkilenmez.
 */
(function () {
  'use strict';

  var el = document.currentScript;
  if (!el) return;

  var ga4 = (el.getAttribute('data-ga4') || '').trim();
  var gtm = (el.getAttribute('data-gtm') || '').trim();
  var zone = (el.getAttribute('data-reklam-zone') || '').trim();
  var reklamAktif = el.getAttribute('data-reklam-aktif') === '1';
  var cotKey = (el.getAttribute('data-cointraffic') || '').trim();

  // --- yardımcı: betik ekle (tek kez) ------------------------------------
  function betik(src, async) {
    if (document.querySelector('script[src="' + src + '"]')) return null;
    var s = document.createElement('script');
    s.src = src;
    s.async = async !== false;
    document.head.appendChild(s);
    return s;
  }

  // --- GA4 ---------------------------------------------------------------
  // gtag.js bir kez yüklenir; GA4 ve GTM aynı dataLayer'ı paylaşır.
  if (ga4) {
    window.dataLayer = window.dataLayer || [];
    window.gtag = window.gtag || function () { window.dataLayer.push(arguments); };
    window.gtag('js', new Date());
    window.gtag('config', ga4);
    betik('https://www.googletagmanager.com/gtag/js?id=' + encodeURIComponent(ga4));
  }

  // --- Google Tag Manager ------------------------------------------------
  if (gtm) {
    window.dataLayer = window.dataLayer || [];
    window.dataLayer.push({ 'gtm.start': new Date().getTime(), event: 'gtm.js' });
    betik('https://www.googletagmanager.com/gtm.js?id=' + encodeURIComponent(gtm));
  }

  // --- Coinzilla reklam ---------------------------------------------------
  // Zone kimliği yalnız [a-z0-9] ise kullanılır; panelden gelen değer
  // httpapi'de doğrulanıyor, burada ikinci kez süzülür (savunma derinliği).
  if (reklamAktif && /^[a-z0-9]{8,64}$/i.test(zone)) {
    window.coinzilla_display = window.coinzilla_display || [];
    betik('https://coinzillatag.com/lib/display.js');

    // Reklam yuvaları: .cd-reklam[data-zone] — yuva yoksa istek de yok.
    var yuvalar = document.querySelectorAll('.cd-reklam[data-zone]');
    for (var i = 0; i < yuvalar.length; i++) {
      var y = yuvalar[i];
      var genislik = parseInt(y.getAttribute('data-genislik') || '728', 10);
      var yukseklik = parseInt(y.getAttribute('data-yukseklik') || '90', 10);
      window.coinzilla_display.push({
        zone: y.getAttribute('data-zone'),
        width: genislik,
        height: yukseklik,
        // Coinzilla kendi seçicisiyle doldurur; yuvaya sınıf veriyoruz.
        callback: (function (hedef) {
          return function () {
            hedef.setAttribute('data-reklam-yuklendi', '1');
          };
        })(y)
      });
    }

    // Sabit (sticky) alt bant: yalnız mobil/masaüstü genişliklerde.
    window.coinzilla_sticky = window.coinzilla_sticky || [];
    window.coinzilla_sticky.push([zone, {
      // Kapat düğmesi bizim stillerimizle; Coinzilla kendi kapatmasını da basar.
      close: true
    }]);
    betik('https://coinzillatag.com/lib/sticky.js');

    // BOŞ YUVAYI KAPAT — ölçülmüş bir görsel hata:
    // Reklam engelleyici kullanan ziyaretçide display.js hiç yüklenmez ve
    // yuva sonsuza kadar boş gri kutu olarak kalır (90 px, her sayfada).
    // Ayrıca Coinzilla'nın o bölge için dolgusu olmayabilir.
    // 5 sn sonra yuva hâlâ doldurulmadıysa gizle: kalıcı boş kutu, geç
    // gelen 90 px'lik yer değişiminden daha kötü. Yer AYNI karede
    // kapatıldığı için CLS etkisi tek seferlik ve küçük.
    setTimeout(function () {
      var yuvalar2 = document.querySelectorAll('.cd-reklam[data-zone]');
      for (var j = 0; j < yuvalar2.length; j++) {
        var y2 = yuvalar2[j];
        if (y2.getAttribute('data-reklam-yuklendi') === '1') continue;
        // Coinzilla iframe ya da img basmışsa dolu say.
        if (y2.querySelector('iframe, img, ins, div > *')) continue;
        y2.setAttribute('data-reklam-bos', '1');
      }
    }, 5000);
  }

  // --- Cointraffic --------------------------------------------------------
  if (cotKey && /^[A-Za-z0-9]{6,64}$/.test(cotKey)) {
    betik('https://appsha1.cointraffic.io/js/?wkey=' + encodeURIComponent(cotKey));
  }
})();
