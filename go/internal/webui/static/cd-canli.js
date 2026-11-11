/* cd-canli.js — CANLI piyasa verisi (sunucudan İTME, SSE).
 *
 * NEDEN (kullanıcı isteği, 11 Kasım 2026): "tüm veriler değişim yaşanınca
 * ANLIK değişmesi ve doğru olması gerekli".
 *
 * ÖNCEKİ SÜRÜM 20 sn'de bir yokluyordu: fiyat değişse bile ekran 20 sn'ye
 * kadar eski kalıyordu ve her ziyaretçi ayrı istek üretiyordu.
 *
 * ŞİMDİ: sunucu Binance'i 5 sn'de bir tazeler ve DEĞİŞTİĞİ ANDA yazar.
 * Tarayıcı `/canli/piyasa` akışını dinler (EventSource). Fiyat katmanı her
 * 5 sn, tam katman (grafik/göstergeler/kur) 30 sn'de bir gelir.
 *
 * GÜNCELLENEN YÜZEYLER (kanca yoksa sessizce atlanır):
 *   • üst şerit            [data-cg-sym="BTC"]
 *   • ana sayfa hero       .cd-giris
 *   • kur / piyasa modülü  [data-cg-piy="usd"]
 *   • fiyat tablosu        [data-cg-coin="BTC"]
 *   • coin sayfası         [data-cg-coin-fiyat]
 *
 * DOĞRULUK KURALLARI:
 *   • null/boş değer YAZILMAZ — uydurma veri yok.
 *   • Bağlantı koparsa "CANLI" yazısı yaşa döner ("12 DK ÖNCE"), yeşil
 *     nokta söner. Olmayan canlılık iddia edilmez.
 *   • EventSource yoksa / koparsa 20 sn yoklamaya düşer (yine de tazelenir).
 *   • Sekme gizliyken akış KAPATILIR (pil/trafik); görünür olunca yeniden
 *     açılır ve anında tazeleme istenir.
 */
(function () {
  'use strict';

  var kok = document.querySelector('.cd-giris');
  var serit = document.querySelector('.cd-ticker');
  var piyKalem = document.querySelector('[data-cg-piy], [data-cg-coin]');
  var coinFiyat = document.querySelector('[data-cg-coin-fiyat]');
  if (!kok && !serit && !piyKalem && !coinFiyat) return; // canlı yüzey yok

  var ES = null;               // EventSource
  var yedekZaman = null;       // yoklama zamanlayıcısı
  var yasZaman = null;
  var saatFarki = 0;           // sunucuSaati - istemciSaati (ms)
  var sonCekildi = 0;          // son verinin sunucu zaman damgası (sn)
  var bagli = false;
  var gecikme = 20000;

  function $(s, el) { return (el || document).querySelector(s); }
  function $$(s, el) { return Array.prototype.slice.call((el || document).querySelectorAll(s)); }

  function yaz(el, metin) {
    if (!el || metin === undefined || metin === null || metin === '') return;
    if (el.textContent !== metin) el.textContent = metin;
  }

  function yon(el, up) {
    if (!el) return;
    el.classList.toggle('cd-yukari', !!up);
    el.classList.toggle('cd-asagi', !up);
  }

  function para(el, metin, up) {
    var d = $(metin, el);
    if (!d) return;
    yon(d, up);
  }

  // --- yarım daire yay geometrisi (sunucudaki cd_giris.go ile AYNI) ---
  var YAY_R = 84;
  var YAY_UZUNLUK = Math.PI * YAY_R;
  function kirp(p) { p = Number(p); if (!isFinite(p)) return 0; return p < 0 ? 0 : (p > 100 ? 100 : p); }
  function yayDeger(p) { return (YAY_UZUNLUK * kirp(p) / 100).toFixed(1) + ' 264'; }
  function noktaXY(p) {
    var t = Math.PI * (1 - kirp(p) / 100);
    return [100 + YAY_R * Math.cos(t), 100 - YAY_R * Math.sin(t)];
  }

  // --- yaş göstergesi ---
  function yasYaz() {
    if (!kok) return;
    var el = $('[data-cg="yas"]', kok);
    if (!el) return;
    if (!sonCekildi) { yaz(el, ''); return; }
    var simdi = (Date.now() + saatFarki) / 1000;
    var fark = Math.max(0, simdi - sonCekildi);
    var metin;
    if (fark < 60) metin = 'CANLI';
    else if (fark < 3600) metin = Math.floor(fark / 60) + ' DK ÖNCE';
    else if (fark < 86400) metin = Math.floor(fark / 3600) + ' SA ÖNCE';
    else metin = Math.floor(fark / 86400) + ' GÜN ÖNCE';
    yaz(el, metin);
    // Yeşil nokta yalnız gerçekten canlıyken yanıp sönsün.
    var nokta = $('.cd-giris__taze-nokta', kok);
    if (nokta) nokta.style.opacity = fark < 60 ? '' : '0.35';
    var rozet = $('.cd-giris__taze', kok);
    if (rozet) rozet.classList.toggle('cd-giris__taze--bayat', !(fark < 60));
  }

  // --- alan güncelleyicileri ---

  function fiyatGuncelle(ad, veri) {
    if (!kok || !veri || !veri.var) return;
    yaz($('[data-cg="' + ad + '-fiyat"]', kok), veri.fiyat);
    var d = $('[data-cg="' + ad + '-degisim"]', kok);
    yaz(d, veri.degisim);
    yon(d, veri.up);
  }

  function gostergeGuncelle(ad, veri) {
    if (!kok || !veri || !veri.var) return;
    yaz($('[data-cg="' + ad + '-deger"]', kok), String(veri.deger));
    var etiket = $('[data-cg="' + ad + '-etiket"]', kok);
    yaz(etiket, veri.etiket);
    if (etiket) yon(etiket, veri.up);

    var yay = $('[data-cg="' + ad + '-yay"]', kok);
    if (yay) yay.setAttribute('stroke-dasharray', yayDeger(veri.deger));

    var nokta = $('[data-cg="' + ad + '-nokta"]', kok);
    if (nokta) {
      var xy = noktaXY(veri.deger);
      nokta.setAttribute('cx', xy[0].toFixed(1));
      nokta.setAttribute('cy', xy[1].toFixed(1));
    }
    var svg = yay && yay.closest('svg');
    if (svg) {
      svg.setAttribute('aria-label',
        ad === 'fng' ? 'Korku ve açgözlülük endeksi: ' + veri.etiket
                     : 'Bitcoin 14 günlük RSI: ' + veri.etiket);
    }
    if (ad === 'fng') {
      if (veri.dun) yaz($('[data-cg="fng-dun"]', kok), String(veri.dun));
      if (veri.hafta) yaz($('[data-cg="fng-hafta"]', kok), String(veri.hafta));
    }
  }

  function seritGuncelle(liste) {
    if (!serit || !liste) return;
    for (var i = 0; i < liste.length; i++) {
      var it = liste[i];
      var k = serit.querySelector('[data-cg-sym="' + it.sym + '"]');
      if (!k) continue;
      yaz($('.cd-ticker__px', k), it.fiyat);
      var ch = $('.cd-up, .cd-down', k);
      if (ch) {
        yaz(ch, it.degisim);
        ch.classList.toggle('cd-up', !!it.up);
        ch.classList.toggle('cd-down', !it.up);
      }
    }
  }

  // Fiyat tablosu + coin sayfaları: [data-cg-coin] / [data-cg-coin-fiyat].
  function coinGuncelle(harita) {
    if (!harita) return;
    $$('[data-cg-coin]').forEach(function (el) {
      var v = harita[el.getAttribute('data-cg-coin')];
      if (!v || !v.var) return;
      yaz($('[data-cg-coin-fiyat]', el), v.fiyat);
      var d = $('[data-cg-coin-degisim]', el);
      if (d) { yaz(d, v.degisim); yon(d, v.up); }
    });
    // Tek varlık sayfası (fiyat doğrudan sayfada).
    var tek = $('[data-cg-coin-fiyat]');
    if (tek && !tek.closest('[data-cg-coin]')) {
      var sym = tek.getAttribute('data-cg-coin-fiyat');
      var vv = harita[sym];
      if (vv && vv.var) {
        yaz(tek, vv.fiyat);
        var dd = $('[data-cg-coin-degisim]');
        if (dd) { yaz(dd, vv.degisim); yon(dd, vv.up); }
      }
    }
  }

  // Kur / BIST kalemleri (PiyasaOf).
  function piyasaGuncelle(harita) {
    if (!harita) return;
    $$('[data-cg-piy]').forEach(function (el) {
      var k = harita[el.getAttribute('data-cg-piy')];
      if (!k) return;
      var deger = el.querySelector('[data-cg-piy-deger]');
      if (deger) {
        var o = typeof k.o === 'number' ? k.o : 2;
        yaz(deger, sayiBicim(k.f, o));
      }
      var d = el.querySelector('[data-cg-piy-degisim]');
      if (d) {
        // Yon: 1 yukarı, -1 aşağı, 0 değişmedi.
        if (typeof k.d === 'number' && k.d !== 0) {
          yaz(d, (k.d > 0 ? '+' : '') + k.d.toFixed(2) + '%');
          yon(d, k.d > 0);
        }
      }
    });
  }

  function sayiBicim(f, o) {
    if (typeof f !== 'number' || !isFinite(f)) return '';
    var p = f.toFixed(o);
    var parca = p.split('.');
    parca[0] = parca[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',');
    return parca.join('.');
  }

  // --- mesaj uygulama ---

  function uygula(v) {
    if (!v) return;
    if (v.sa) saatFarki = v.sa * 1000 - Date.now();
    if (v.cekildi) sonCekildi = v.cekildi;
    yasYaz();

    // Her iki mesaj tipinde de gelen fiyat katmanı.
    fiyatGuncelle('btc', v.btc);
    fiyatGuncelle('eth', v.eth);
    fiyatGuncelle('sol', v.sol);
    // Şerit: ayrı liste + coin haritası birleştirilir. Sunucu delta
    // gönderdiği için "değişen 8. coin" serit listesine sığmayabilir; coin
    // haritasındaki her sembol şeritte de aranır.
    var seritListe = (v.serit || []).slice();
    if (v.coin) {
      for (var ck in v.coin) {
        if (!Object.prototype.hasOwnProperty.call(v.coin, ck)) continue;
        var cv = v.coin[ck];
        seritListe.push({ sym: ck, fiyat: cv.fiyat, degisim: cv.degisim, up: cv.up });
      }
    }
    seritGuncelle(seritListe);
    seritGuncelle(v.serit_bist);
    coinGuncelle(v.coin);

    if (v.t === 't' && v.hazir && kok) {
      var cizgi = $('[data-cg="cizgi"]', kok);
      if (cizgi && v.yol) cizgi.setAttribute('d', v.yol);
      var alan = $('[data-cg="alan"]', kok);
      if (alan && v.alan) alan.setAttribute('d', v.alan);
      if (v.hacim) yaz($('[data-cg="hacim"]', kok), v.hacim);
      if (v.dominans) yaz($('[data-cg="dominans"]', kok), v.dominans);
      if (v.endeks) {
        var e = $('[data-cg="endeks"]', kok);
        yaz(e, v.endeks);
        yon(e, v.endeks_up);
      }
      gostergeGuncelle('fng', v.fng);
      gostergeGuncelle('rsi', v.rsi);
    }

    if (v.t === 't') piyasaGuncelle(v.piyasa);
  }

  // --- bağlantı yönetimi ---

  function baglantiDurumu(acik) {
    bagli = acik;
    if (!kok) return;
    var rozet = $('.cd-giris__taze', kok);
    if (rozet) rozet.classList.toggle('cd-giris__taze--kesik', !acik);
  }

  function akisA() {
    if (typeof window.EventSource === 'undefined') { yedekBaslat(); return; }
    if (ES) return;
    ES = new EventSource('/canli/piyasa');

    ES.onopen = function () {
      baglantiDurumu(true);
      gecikme = 20000;
      yedekDurdur();
    };
    ES.onmessage = function (ev) {
      baglantiDurumu(true);
      try { uygula(JSON.parse(ev.data)); } catch (e) { /* bozuk mesaj: yok say */ }
    };
    ES.onerror = function () {
      // EventSource kendi yeniden bağlanır; yine de yedek yoklamayı aç ki
      // uzun bir kesintide veri tamamen donmasın.
      baglantiDurumu(false);
      yedekBaslat();
    };
  }

  function akisKapat() {
    if (ES) { ES.close(); ES = null; }
    baglantiDurumu(false);
  }

  function yedekBaslat() {
    if (yedekZaman) return;
    function tur() {
      if (document.hidden) { yedekZaman = null; return; }
      fetch('/canli/giris?_=' + Date.now(), {
        cache: 'no-store', credentials: 'same-origin',
        headers: { Accept: 'application/json' }
      }).then(function (r) {
        if (!r.ok) throw new Error('HTTP ' + r.status);
        return r.json();
      }).then(function (v) {
        uygula(v);
        gecikme = 20000;
        yedekZaman = setTimeout(tur, gecikme);
      }).catch(function () {
        gecikme = Math.min(gecikme * 2, 300000);
        yedekZaman = setTimeout(tur, gecikme);
      });
    }
    tur();
  }

  function yedekDurdur() {
    if (yedekZaman) { clearTimeout(yedekZaman); yedekZaman = null; }
  }

  function baslat() {
    if (document.hidden) return;
    akisA();
  }

  document.addEventListener('visibilitychange', function () {
    if (document.hidden) {
      akisKapat();
      yedekDurdur();
    } else {
      baslat();
    }
  });

  // Yaş yazısı ağ gerektirmez: 10 sn'de bir yerel saatle tazelenir.
  yasZaman = setInterval(yasYaz, 10000);

  baslat();
})();
