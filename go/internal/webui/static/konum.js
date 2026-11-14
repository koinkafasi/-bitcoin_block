/* Ziyaretçi konumu — izin penceresi ve panel kapısı.
 *
 * NEDEN AYRI DOSYA: cd.css/cd.js bütçeleri var; bu katman her sayfada
 * gerekiyor ama ağır değil. Sunucu tarafı ÜLKE kaydı ayrı yerde
 * (konum_middleware.go) — bu betik yalnız GERÇEK koordinat içindir.
 *
 * AKIŞ (14 Kasım 2026'dan beri):
 *   1) Sayfa yüklenir. Çerez `tb_konum` "1" ise hiçbir şey yapılmaz.
 *   2) NORMAL SAYFA: bilgilendirme kartı AÇILMAZ; tarayıcının kendi
 *      konum izni doğrudan istenir (site sahibinin kararı). İzin
 *      verilirse koordinat POST /api/konum ile gönderilir.
 *   3) KAPI (üye paneli): kart/sayfa kartı açılır, kapatılamaz;
 *      "Konumu etkinleştir" → tarayıcı izni → POST /api/konum.
 *   4) Reddedilirse yalnız ülke bilgisi tazelenir.
 *
 * TARAYICI SINIRI: izin kararını tarayıcı verdirir; site kabul ettiremez.
 * Kullanıcı "Engelle" derse Chrome bir daha sormaz (yalnız site
 * ayarlarından açılır) — o durumda boşuna çağrı yapılmaz. Balonu yalnız
 * kapatanlara sonraki sayfa açılışında yeniden sorulur.
 *
 * CSP UYUMLU: satır içi olay yok, delege dinleyici yok; doğrudan
 * addEventListener. ('unsafe-inline' kapalı.)
 */
(function () {
  'use strict';

  var kart = document.querySelector('[data-konum]');
  if (!kart) return;

  var kapiModu = kart.getAttribute('data-kapi') === '1';
  var durumEl = kart.querySelector('[data-konum-durum]');
  var verDugme = kart.querySelector('[data-konum-ver]');

  function cerez(ad) {
    var parca = ('; ' + document.cookie).split('; ' + ad + '=');
    return parca.length === 2 ? parca.pop().split(';')[0] : '';
  }

  // Konum zaten verilmişse kartı hiç açma.
  if (cerez('tb_konum') === '1' && !kapiModu) return;
  if (cerez('tb_konum') === '1' && kapiModu) return;

  // Okuma engellenmesin diye kartı açık bırak ama gövde kaydırmasını kilitle.
  function ac() {
    kart.hidden = false;
    document.documentElement.style.overflow = 'hidden';
    if (verDugme) verDugme.focus();
  }
  function kapat() {
    kart.hidden = true;
    document.documentElement.style.overflow = '';
  }

  function durum(metin, tur) {
    if (!durumEl) return;
    durumEl.textContent = metin || '';
    durumEl.className = 'cd-konum__durum' + (tur ? ' cd-konum__durum--' + tur : '');
  }

  // Sunucuya gönder: koordinat veya red.
  function gonder(govde, bitti) {
    var xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/konum', true);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function () {
      bitti(xhr.status >= 200 && xhr.status < 300);
    };
    xhr.onerror = function () { bitti(false); };
    xhr.send(JSON.stringify(govde));
  }

  function reddet() {
    gonder({ red: true }, function () {
      if (kapiModu) {
        durum('Konum olmadan panel açılamıyor. Daha sonra tekrar deneyebilirsiniz.', 'hata');
        return;
      }
      kapat();
    });
  }

  function izinIste() {
    if (!navigator.geolocation) {
      durum('Tarayıcınız konum servisini desteklemiyor.', 'hata');
      return;
    }
    durum('Tarayıcı izni bekleniyor…');
    if (verDugme) verDugme.disabled = true;

    navigator.geolocation.getCurrentPosition(function (konum) {
      gonder({
        enlem: konum.coords.latitude,
        boylam: konum.coords.longitude,
        dogruluk_m: Math.round(konum.coords.accuracy || 0)
      }, function (ok) {
        if (verDugme) verDugme.disabled = false;
        if (!ok) {
          durum(kart.getAttribute('data-hata'), 'hata');
          return;
        }
        durum('Konum kaydedildi. Teşekkürler!', 'ok');
        setTimeout(function () {
          if (kapiModu) {
            // Panel kapısı: çerez yazıldı, sunucu artık izin görüyor.
            window.location.reload();
          } else {
            kapat();
          }
        }, 900);
      });
    }, function (hata) {
      if (verDugme) verDugme.disabled = false;
      // PERMISSION_DENIED (1): kullanıcı reddetti → sunucuya bildir.
      // Konum alınamıyor (2/3): geçici; kart açık kalır, tekrar denenebilir.
      if (hata && hata.code === 1) {
        reddet();
        durum('Tarayıcı izni verilmedi. Ayarlardan açabilirsiniz.', 'hata');
        return;
      }
      durum(kart.getAttribute('data-hata'), 'hata');
    }, { enableHighAccuracy: false, timeout: 12000, maximumAge: 300000 });
  }

  // DOĞRUDAN İZİN: kart açmadan tarayıcının konum iznini ister.
  // Sonuç sessizce sunucuya bildirilir; sayfada hiçbir şey gösterilmez.
  function dogrudanIste() {
    navigator.geolocation.getCurrentPosition(function (konum) {
      gonder({
        enlem: konum.coords.latitude,
        boylam: konum.coords.longitude,
        dogruluk_m: Math.round(konum.coords.accuracy || 0)
      }, function () {});
    }, function (hata) {
      // Red sunucuya BİR KEZ bildirilir (çerez "0" olur); her sayfada
      // yinelenen istek atılmaz.
      if (hata && hata.code === 1 && cerez('tb_konum') !== '0') {
        gonder({ red: true }, function () {});
      }
    }, { enableHighAccuracy: false, timeout: 12000, maximumAge: 300000 });
  }

  function dogrudanBasla() {
    if (!navigator.geolocation) return;
    // Tarayıcı kalıcı olarak engellediyse çağrı anında hata verir ve
    // kullanıcıya hiçbir şey sorulmaz; önce izin durumuna bak.
    if (navigator.permissions && navigator.permissions.query) {
      navigator.permissions.query({ name: 'geolocation' }).then(function (izin) {
        if (izin.state !== 'denied') {
          dogrudanIste();
        } else if (cerez('tb_konum') !== '0') {
          gonder({ red: true }, function () {});
        }
      }, dogrudanIste);
      return;
    }
    dogrudanIste();
  }

  // Kapı SAYFASI (tam sayfa kart) ile modal AYNI ANDA gösterilmez.
  //
  // ÖLÇÜLEN KUSUR (gerçek tarayıcı, 14 Kasım 2026): /hesabim kapısında
  // sayfanın kendi kartı basılıyor, üstüne modal açılıyordu → arkada
  // soluk ikinci kart ve başlık görünüyordu (çift içerik).
  // Sayfa kartı yeterli ve daha erişilebilir (modal değil, kaydırılabilir),
  // bu yüzden kapı sayfasında modal hiç açılmıyor.
  var sayfaKarti = !!document.querySelector('.cd-kapi__kart');
  if (sayfaKarti) {
    kart.hidden = true;
  }

  // Kapı modunda "Şimdi değil" yok: kart kapanamaz.
  if (!kapiModu) {
    kart.querySelectorAll('[data-konum-kapat]').forEach(function (el) {
      el.addEventListener('click', reddet);
    });
    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && !kart.hidden) reddet();
    });
  }

  // Kapı sayfasındaki düğme de aynı akışı başlatır (modalın dışında).
  document.querySelectorAll('[data-konum-ver]').forEach(function (el) {
    el.addEventListener('click', izinIste);
  });

  // Sayfa yüklendikten sonra başla: ilk boyama (LCP) engellenmesin.
  // Kapı modunda kart, normal sayfada doğrudan tarayıcı izni.
  if (!sayfaKarti) window.setTimeout(kapiModu ? ac : dogrudanBasla, 600);
})();
