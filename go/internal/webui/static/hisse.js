/* Hisseler modülü + "Tüm Hisseler" sayfası.
 *
 * 1) Şerit hareketi — iki model (blok ayarından):
 *      data-model="surekli" : Habertürk modeli, kesintisiz sabit hız (px/sn)
 *      data-model="adim"    : aralıklı — her turda yumuşak kayma + bekleme
 *
 *    CSS'te de bir yedek animasyon var (JS yoksa çalışır). JS varsa
 *    .hss-js sınıfı eklenir ve CSS animasyonu kapanır; hareket rAF ile
 *    piksel bazında sürülür. Nedeni: yüzde tabanlı CSS animasyonu içerik
 *    genişliği değiştiğinde (yazı tipi/görsel yüklenmesi) hızı değiştiriyor,
 *    "bir an yavaşlıyor sonra hızlanıyor" gibi görünüyordu. Piksel/sn sabit
 *    hız bu sorunu tamamen kaldırır.
 *
 * 2) Tüm Hisseler sayfasında canlı arama: satırlar zaten DOM'da olduğu için
 *    ağ isteği yok; Türkçe karakter duyarsız (ışık → isik) filtre uygulanır.
 */
(function () {
  'use strict';

  var reduce = false;
  try {
    reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  } catch (e) { /* eski tarayıcı */ }

  /* ---------------- 1) Şerit ---------------- */

  function initSerit(el) {
    // Hisseler (hss-) ve Kripto (krp-) aynı hareket motorunu kullanır.
    var track = el.querySelector('.hss-track, .krp-track');
    var frame = el.querySelector('.hss-frame, .krp-frame');
    var group = el.querySelector('.hss-group, .krp-group');
    if (!track || !frame || !group) return;

    // mq-js: CSS yedek animasyonunu kapat (ortak sınıf).
    el.classList.add('mq-js');
    if (el.classList.contains('hss-block')) el.classList.add('hss-js');

    var model = el.getAttribute('data-model') || 'surekli';
    var speed = parseFloat(el.getAttribute('data-speed')) || 60;   // px/sn
    var hold = parseFloat(el.getAttribute('data-hold')) || 4;      // sn
    var glide = parseFloat(el.getAttribute('data-glide')) || 700;  // ms

    var x = 0;           // mevcut kaydırma (negatif)
    var width = 0;       // bir grubun genişliği (dikişsiz döngü noktası)
    var paused = false;
    var last = 0;
    var rafId = 0;

    function measure() {
      width = group.getBoundingClientRect().width;
      if (width < 1) width = 1;
      // Sıfırla: içerik yeniden akıtıldıysa (yazı tipi yüklendi) konum taşmasın.
      if (-x >= width) x = x % width;
    }

    function apply() {
      track.style.transform = 'translate3d(' + x.toFixed(2) + 'px,0,0)';
    }

    /* sürekli: her karede hız*dt kadar sola kay */
    function tickSurekli(now) {
      if (!last) last = now;
      var dt = (now - last) / 1000;
      last = now;
      if (!paused) {
        x -= speed * dt;
        if (-x >= width) x += width;
        apply();
      }
      rafId = window.requestAnimationFrame(tickSurekli);
    }

    /* adım: bekle → yumuşak kaydır (bir görünür genişlik) → tekrar */
    function tickAdim() {
      if (paused) {
        window.setTimeout(tickAdim, 400);
        return;
      }
      var page = Math.max(120, frame.getBoundingClientRect().width * 0.85);
      if (-x + page >= width) page = width + x; // sınırı aşma
      if (page <= 1) { x = 0; apply(); window.setTimeout(tickAdim, hold * 1000); return; }

      var start = x;
      var target = x - page;
      var t0 = 0;
      function step(now) {
        if (!t0) t0 = now;
        if (paused) { window.requestAnimationFrame(step); return; }
        var p = Math.min(1, (now - t0) / glide);
        // easeInOutCubic — "hafif" giriş/çıkış
        var e = p < 0.5 ? 4 * p * p * p : 1 - Math.pow(-2 * p + 2, 3) / 2;
        x = start + (target - start) * e;
        apply();
        if (p < 1) { window.requestAnimationFrame(step); return; }
        if (-x >= width) x += width;
        window.setTimeout(tickAdim, hold * 1000);
      }
      window.requestAnimationFrame(step);
    }

    function start() {
      iptal();
      measure();
      apply();
      if (reduce) {
        // Hareket azaltma: animasyon yok, şerit elle kaydırılabilir.
        frame.style.overflowX = 'auto';
        return;
      }
      if (model === 'adim') {
        window.setTimeout(tickAdim, hold * 1000);
      } else {
        rafId = window.requestAnimationFrame(tickSurekli);
      }
    }
    function iptal() { if (rafId) { window.cancelAnimationFrame(rafId); rafId = 0; } }

    // Üzerine gelince / odaklanınca dur (okumak için).
    frame.addEventListener('mouseenter', function () { paused = true; });
    frame.addEventListener('mouseleave', function () { paused = false; last = 0; });
    frame.addEventListener('focusin', function () { paused = true; });
    frame.addEventListener('focusout', function () { paused = false; last = 0; });

    // Sekme arkada kalınca rAF zaten durur; geri gelince sıçramasın.
    document.addEventListener('visibilitychange', function () {
      last = 0;
    });

    // Genişlik değişince (döndürme, font yüklemesi) yeniden ölç.
    if (window.ResizeObserver) {
      var ro = new ResizeObserver(function () { measure(); });
      ro.observe(el);
    } else {
      window.addEventListener('resize', measure);
    }
    if (document.fonts && document.fonts.ready) {
      document.fonts.ready.then(measure).catch(function () {});
    }

    start();
  }

  function initSeritler() {
    var list = document.querySelectorAll('[data-hss], [data-marquee]');
    for (var i = 0; i < list.length; i++) initSerit(list[i]);
  }

  /* ---------------- 2) Tüm Hisseler araması ---------------- */

  function trNorm(s) {
    s = String(s || '');
    var map = {
      'ı': 'i', 'İ': 'i', 'I': 'i', 'ş': 's', 'Ş': 's', 'ğ': 'g', 'Ğ': 'g',
      'ü': 'u', 'Ü': 'u', 'ö': 'o', 'Ö': 'o', 'ç': 'c', 'Ç': 'c', 'â': 'a',
      'î': 'i', 'û': 'u'
    };
    var out = '';
    for (var i = 0; i < s.length; i++) {
      var ch = s[i];
      out += (map[ch] !== undefined ? map[ch] : ch);
    }
    return out.toLowerCase().replace(/\s+/g, ' ').trim();
  }

  function initArama(kok) {
    var input = kok.querySelector('[data-hsp-q]');
    var body = kok.querySelector('[data-hsp-body]');
    var empty = kok.querySelector('[data-hsp-empty]');
    var count = kok.querySelector('[data-hsp-count]');
    if (!input || !body) return;

    var rows = Array.prototype.slice.call(body.querySelectorAll('tr'));

    function filtre() {
      var q = trNorm(input.value);
      var n = 0;
      for (var i = 0; i < rows.length; i++) {
        var r = rows[i];
        var hay = r.getAttribute('data-arama') || trNorm(r.textContent);
        var ok = !q || hay.indexOf(q) !== -1;
        r.hidden = !ok;
        if (ok) n++;
      }
      if (count) count.textContent = String(n);
      if (empty) empty.hidden = n !== 0;
    }

    var t = 0;
    input.addEventListener('input', function () {
      window.clearTimeout(t);
      t = window.setTimeout(filtre, 90);
    });
    input.addEventListener('keydown', function (e) {
      if (e.key === 'Escape') { input.value = ''; filtre(); }
    });

    // Sayfa ?q= ile geldiyse ilk durum zaten sunucuda süzülmüş; yine de
    // istemci tarafı filtreyi tutarlı kılmak için bir kez uygula.
    if (input.value) filtre();
  }

  /* Kripto sayfası: tablo/kart/kompakt — hepsinde satır/kart öğeleri
     data-arama taşır. Ortak filtre mantığı. */
  function initKriptoArama(kok) {
    var input = kok.querySelector('[data-kps-q]');
    var body = kok.querySelector('[data-kps-body]');
    var empty = kok.querySelector('[data-kps-empty]');
    var count = kok.querySelector('[data-kps-count]');
    if (!input || !body) return;

    var items = Array.prototype.slice.call(body.querySelectorAll('[data-arama]'));

    function filtre() {
      var q = trNorm(input.value);
      var n = 0;
      for (var i = 0; i < items.length; i++) {
        var r = items[i];
        var hay = r.getAttribute('data-arama') || trNorm(r.textContent);
        var ok = !q || hay.indexOf(q) !== -1;
        r.hidden = !ok;
        if (ok) n++;
      }
      if (count) count.textContent = String(n);
      if (empty) empty.hidden = n !== 0;
    }

    var t = 0;
    input.addEventListener('input', function () {
      window.clearTimeout(t);
      t = window.setTimeout(filtre, 90);
    });
    input.addEventListener('keydown', function (e) {
      if (e.key === 'Escape') { input.value = ''; filtre(); }
    });
    if (input.value) filtre();
  }

  /* ---------------- 3) Canlı finans akışı ----------------
   *
   * Kur & Borsa, Hisseler, Kripto Para, servis bandı ve Kur Çevirici aynı
   * /api/piyasa/canli ucundan beslenir. Davranış:
   *   • Yalnız görünür sekmede, 30 sn'de bir istek (ETag → çoğu zaman 304).
   *   • İlk yüklemede sessiz: sunucudan gelen değerlerle aynıysa parlamaz.
   *   • Değer değiştiyse fiyat hücresi 1,2 sn yeşil/kırmızı parlar; yön
   *     oku ve yüzde aynı anda güncellenir.
   *   • "Canlı" rozeti ve "Son güncelleme" satırı tazelenir.
   *   • Hareket azaltma açıksa animasyon yerine statik renk vurgusu.
   *   • Hata sessizce yutulur; 30→60→120 sn geri çekilme ile yeniden denenir.
   *     Eski değerler asla sıfırlanmaz/uydurulmaz.
   */

  var FIN_POLL = 30000;
  var FIN_MAX = 120000;
  var FIN_FLASH = 1200;
  var finTimer = 0;
  var finBekle = FIN_POLL;
  var finBusy = false;
  var finEtag = '';
  var finSon = 0;
  var finIlk = true;
  var finDurum = null;

  function finSayi(v) {
    var n = parseFloat(v);
    return isFinite(n) ? n : NaN;
  }

  function finTR(v, hane) {
    if (!isFinite(v)) return '—';
    try {
      return new Intl.NumberFormat('tr-TR', {
        minimumFractionDigits: hane,
        maximumFractionDigits: hane
      }).format(v);
    } catch (e) {
      return v.toFixed(hane).replace('.', ',');
    }
  }

  function finIsaretli(v) {
    var s = finTR(Math.abs(v), 2);
    if (v > 0) return '+' + s;
    if (v < 0) return '-' + s;
    return s;
  }

  /* paraKisa (Go) ile aynı kısaltmalar: bin → b, milyon → mn, milyar → mr. */
  function finKisa(v) {
    if (!isFinite(v) || v <= 0) return '—';
    if (v >= 1e12) return finTR(v / 1e12, 2) + ' tn';
    if (v >= 1e9) return finTR(v / 1e9, 2) + ' mr';
    if (v >= 1e6) return finTR(v / 1e6, 1) + ' mn';
    if (v >= 1e3) return finTR(v / 1e3, 1) + ' b';
    return finTR(v, 2);
  }

  function finToggle(el, sinif, acik) {
    if (!el || !el.classList) return;
    if (acik) el.classList.add(sinif);
    else el.classList.remove(sinif);
  }

  function finUst(el, secici) {
    while (el && el !== document) {
      if (el.matches && el.matches(secici)) return el;
      el = el.parentNode;
    }
    return null;
  }

  function finParlat(el, yukari) {
    if (!el) return;
    var sinif = reduce
      ? (yukari ? 'fin-tint-up' : 'fin-tint-down')
      : (yukari ? 'fin-flash-up' : 'fin-flash-down');
    el.classList.remove('fin-flash-up', 'fin-flash-down', 'fin-tint-up', 'fin-tint-down');
    // Aynı sınıf art arda eklenirse animasyon yeniden başlamaz.
    void el.offsetWidth;
    el.classList.add(sinif);
    window.setTimeout(function () { el.classList.remove(sinif); }, FIN_FLASH);
  }

  function finAlanMetni(tur, fld, kayit, el) {
    var v = kayit[fld];
    if (typeof v !== 'number' || !isFinite(v)) return null;
    if (tur === 'hisse') {
      if (fld === 'f') return finTR(v, 2);
      if (fld === 'd') return '%' + finIsaretli(v);
    }
    if (tur === 'kripto') {
      if (fld === 'f') return finTR(v, 2);
      if (fld === 'd' || fld === 'd7') return '%' + finIsaretli(v);
      if (fld === 'h' || fld === 'p') return finKisa(v);
    }
    if (tur === 'piyasa') {
      if (fld === 'f') {
        var hane = parseInt(el.getAttribute('data-fin-dec') || (kayit.o || 2), 10);
        if (!isFinite(hane) || hane < 0) hane = 2;
        var birim = el.getAttribute('data-fin-unit') || kayit.b || '';
        return birim + finTR(v, hane);
      }
      if (fld === 'd') return '%' + finIsaretli(v);
    }
    if (tur === 'servis') {
      if (fld === 'f') return v.toFixed(2);
      if (fld === 'd') return v.toFixed(2) + '%';
    }
    return null;
  }

  function finAlanYon(el, tur, fld, kayit) {
    var v = kayit[fld];
    if (typeof v !== 'number' || !isFinite(v)) return;
    var yon = v > 0 ? 1 : v < 0 ? -1 : 0;

    // Tüm Hisseler tablosu: yüzde hücresi + ▲/▼ metni.
    if (el.classList.contains('hsp-chg')) {
      el.classList.remove('hss-up', 'hss-down', 'hss-flat');
      el.classList.add('hss-' + (yon > 0 ? 'up' : yon < 0 ? 'down' : 'flat'));
      var ok = el.querySelector('[data-fin-arrow-text]');
      if (ok) ok.textContent = yon > 0 ? '▲' : yon < 0 ? '▼' : '–';
    }

    // Kripto tablosu: 24s ve 7g hücreleri kendi rengini taşır.
    if (el.classList.contains('kps-chg')) {
      el.classList.remove('up', 'down', 'flat');
      el.classList.add(yon > 0 ? 'up' : yon < 0 ? 'down' : 'flat');
    }

    // Servis bandı: yeşil/kırmızı rozet + ok yönü.
    if (el.classList.contains('tk-chg')) {
      el.classList.remove('up', 'down');
      if (yon > 0) el.classList.add('up');
      else if (yon < 0) el.classList.add('down');
      var p = el.querySelector('path');
      if (p) p.setAttribute('d', yon < 0 ? 'M6 9l6 6 6-6' : 'M18 15l-6-6-6 6');
    }
  }

  function finAlanGuncelle(el, tur, fld, kayit, sessiz) {
    var metin = finAlanMetni(tur, fld, kayit, el);
    if (metin === null) return false;
    var yeni = kayit[fld];
    var eski = finSayi(el.getAttribute('data-fin-v'));
    var degisti = !isFinite(eski) || Math.abs(eski - yeni) > 1e-9;

    // Yüzde hücresinde iç span (ör. hsp-pct / tk-chg pct) varsa koru.
    var pct = (fld === 'd' || fld === 'd7') ? el.querySelector('[data-fin-pct]') : null;
    if (pct) pct.textContent = metin;
    else el.textContent = metin;

    el.setAttribute('data-fin-v', String(yeni));
    if (degisti && !sessiz) finParlat(el, isFinite(eski) ? yeni >= eski : true);
    return degisti;
  }

  var finOklar = {
    ucgen: { '-1': 'M12 19l7-12H5z', '1': 'M12 5l7 12H5z', '0': 'M5 11h14v2H5z' },
    ok: { '-1': 'M6 9l6 6 6-6', '1': 'M18 15l-6-6-6 6', '0': 'M5 12h14' },
    uzun: { '-1': 'M6 6l12 12M18 8v10H8', '1': 'M6 18L18 6M8 6h10v10', '0': 'M5 12h14' }
  };

  function finOk(kok, onek, yon) {
    var ikon = kok.querySelector('.' + onek + '-ico');
    if (!ikon) return;
    var blok = finUst(kok, '.' + onek + '-strip, .' + onek + '-block');
    var stil = 'noktali';
    if (blok) {
      var m = String(blok.className || '').match(new RegExp(onek + '-arrow-([a-z]+)'));
      if (m) stil = m[1];
    }
    if (stil === 'yok') return;

    // Noktalı referans okunda yön yalnız döndürme/sınıf ile değişir.
    if (stil === 'noktali') {
      finToggle(ikon, onek + '-ico-down', yon < 0);
      finToggle(ikon, onek + '-ico-flat', yon === 0);
      return;
    }
    var yol = finOklar[stil] && finOklar[stil][String(yon)];
    if (!yol) return;
    var p = ikon.querySelector('path');
    if (p) p.setAttribute('d', yol);
    finToggle(ikon, onek + '-ico-flat', yon === 0);
  }

  // Noktalı okun kesikli kuyruğu: her gerçek güncellemede kısa akış.
  // Yön değişmese de "veri geldi" hissi verir; reduced-motion'da çalışmaz.
  function finOkAkis(kok, onek) {
    if (reduce) return;
    var ikon = kok.querySelector('.' + onek + '-ico-noktali');
    if (!ikon) return;
    ikon.classList.remove('fin-ok-akis');
    // Aynı sınıf art arda eklenirse animasyon yeniden başlamaz.
    void ikon.offsetWidth;
    ikon.classList.add('fin-ok-akis');
    if (ikon._finAkisTimer) window.clearTimeout(ikon._finAkisTimer);
    ikon._finAkisTimer = window.setTimeout(function () {
      ikon.classList.remove('fin-ok-akis');
      ikon._finAkisTimer = 0;
    }, 900);
  }

  function finYonGuncelle(kok, tur, kayit) {
    var yon;
    if (tur === 'piyasa') {
      yon = typeof kayit.y === 'number' ? kayit.y : (kayit.d > 0 ? 1 : kayit.d < 0 ? -1 : 0);
    } else {
      yon = kayit.d > 0 ? 1 : kayit.d < 0 ? -1 : 0;
    }
    if (tur === 'hisse') {
      kok.classList.remove('hss-up', 'hss-down', 'hss-flat');
      kok.classList.add('hss-' + (yon > 0 ? 'up' : yon < 0 ? 'down' : 'flat'));
      finOk(kok, 'hss', yon);
    } else if (tur === 'kripto') {
      kok.classList.remove('krp-up', 'krp-down', 'krp-flat');
      kok.classList.add('krp-' + (yon > 0 ? 'up' : yon < 0 ? 'down' : 'flat'));
    } else if (tur === 'piyasa') {
      kok.classList.remove('pys-up', 'pys-down');
      if (yon > 0) kok.classList.add('pys-up');
      else if (yon < 0) kok.classList.add('pys-down');
      finOk(kok, 'pys', yon);
    }
  }

  function finKurGuncelle(kurlar) {
    if (!kurlar) return;
    var list = document.querySelectorAll('option[data-fin-kur]');
    var degisti = false;
    for (var i = 0; i < list.length; i++) {
      var kod = list[i].getAttribute('data-fin-kur');
      var v = kurlar[kod];
      if (typeof v !== 'number' || !isFinite(v) || v <= 0) continue;
      var eski = finSayi(list[i].getAttribute('data-rate'));
      if (!isFinite(eski) || Math.abs(eski - v) > 1e-12) degisti = true;
      list[i].setAttribute('data-rate', String(v));
    }
    if (degisti) {
      try { document.dispatchEvent(new Event('fin:kur')); } catch (e) { /* eski tarayıcı */ }
    }
  }

  function finZamanGuncelle(veri) {
    var list = document.querySelectorAll('[data-fin-time]');
    for (var i = 0; i < list.length; i++) {
      var el = list[i];
      var tur = el.getAttribute('data-fin-time');
      var z = veri && veri.guncel && veri.guncel[tur];
      if (!z) continue;
      var dt = el.getAttribute('data-fin-time-format') === 'dt';
      el.textContent = 'Son güncelleme: ' + (dt ? z.dt : z.hm);
      el.hidden = false;
    }

    var canli = document.querySelectorAll('[data-fin-live]');
    var g = veri && veri.guncel;
    if (!g) return;
    for (var k = 0; k < canli.length; k++) {
      // Rozet kendi türünün saatini göstersin (hisse 6 dk, kripto 2 dk).
      var turAd = canli[k].getAttribute('data-fin-live');
      var t = (turAd && g[turAd]) || g.hisse || g.kripto || g.piyasa || g.servis;
      if (!t) { canli[k].hidden = true; continue; }
      canli[k].hidden = false;
      canli[k].title = 'Otomatik güncellenir · Son güncelleme: ' + t.hm;
    }
  }

  function finDuyur(veri) {
    if (!finDurum) {
      finDurum = document.createElement('span');
      finDurum.className = 'sr-only';
      finDurum.setAttribute('role', 'status');
      finDurum.setAttribute('aria-live', 'polite');
      document.body.appendChild(finDurum);
    }
    var g = veri && veri.guncel;
    var t = g && (g.hisse || g.kripto || g.piyasa || g.servis);
    finDurum.textContent = 'Piyasa verileri güncellendi' + (t ? ', ' + t.hm : '');
  }

  function finUygula(veri, sessiz) {
    var list = document.querySelectorAll('[data-fin]');
    var degisen = false;
    for (var i = 0; i < list.length; i++) {
      var kok = list[i];
      var tur = kok.getAttribute('data-fin');
      var sembol = kok.getAttribute('data-fin-sym');
      if (!sembol) continue; // çevirici gibi işaretleyiciler
      var kayit = veri && veri[tur] && veri[tur][sembol];
      if (!kayit) continue;

      var kokDegisti = false;
      var alanlar = kok.querySelectorAll('[data-fin-fld]');
      for (var j = 0; j < alanlar.length; j++) {
        var el = alanlar[j];
        var fld = el.getAttribute('data-fin-fld');
        if (finAlanGuncelle(el, tur, fld, kayit, sessiz)) {
          degisen = true;
          kokDegisti = true;
        }
        if (fld === 'd' || fld === 'd7') finAlanYon(el, tur, fld, kayit);
      }
      finYonGuncelle(kok, tur, kayit);
      if (kokDegisti && !sessiz && tur === 'hisse') finOkAkis(kok, 'hss');
    }
    finKurGuncelle(veri && veri.kur);
    finZamanGuncelle(veri);
    if (degisen && !sessiz) finDuyur(veri);
  }

  function finTurleri() {
    var set = {};
    var list = document.querySelectorAll('[data-fin]');
    for (var i = 0; i < list.length; i++) {
      var t = list[i].getAttribute('data-fin');
      if (t) set[t] = 1;
    }
    var out = [];
    for (var k in set) {
      if (Object.prototype.hasOwnProperty.call(set, k)) out.push(k);
    }
    return out;
  }

  function finPlan(ms) {
    window.clearTimeout(finTimer);
    if (document.visibilityState && document.visibilityState !== 'visible') return;
    finTimer = window.setTimeout(finIste, ms === undefined ? finBekle : ms);
  }

  function finIste() {
    if (finBusy) return;
    if (document.visibilityState && document.visibilityState !== 'visible') return;
    var turler = finTurleri();
    if (!turler.length) return;

    finBusy = true;
    var url = '/api/piyasa/canli?tur=' + encodeURIComponent(turler.join(','));
    var basliklar = {};
    if (finEtag) basliklar['If-None-Match'] = finEtag;
    var ctrl = null;
    var zaman = 0;
    if (window.AbortController) {
      ctrl = new AbortController();
      zaman = window.setTimeout(function () {
        try { ctrl.abort(); } catch (e) { /* yoksay */ }
      }, 12000);
    }
    var secenek = { headers: basliklar, credentials: 'same-origin', cache: 'no-store' };
    if (ctrl) secenek.signal = ctrl.signal;

    window.fetch(url, secenek).then(function (r) {
      if (zaman) window.clearTimeout(zaman);
      if (r.status === 304) return null;
      if (!r.ok) throw new Error('HTTP ' + r.status);
      var etag = r.headers.get('ETag');
      if (etag) finEtag = etag;
      return r.json();
    }).then(function (veri) {
      finBusy = false;
      finBekle = FIN_POLL;
      finSon = Date.now();
      if (veri) {
        finUygula(veri, finIlk);
        finIlk = false;
      }
      finPlan();
    }).catch(function () {
      if (zaman) window.clearTimeout(zaman);
      finBusy = false;
      // İlk istek başarısız olduysa sonraki başarı sessiz sayılmasın: değer
      // bu arada değişmiş olabilir, kullanıcı farkı görmeli.
      finIlk = false;
      // Sessiz geri çekilme: 30 → 60 → 120 sn. Kullanıcıya hata kutusu yok.
      finBekle = Math.min(finBekle * 2, FIN_MAX);
      finPlan();
    });
  }

  function finBasla() {
    if (window.__finansCanli) return;
    window.__finansCanli = true;
    if (!window.fetch) return;

    // Sayfa yerleşsin; ilk istek hemen değil, kısa bir gecikmeyle.
    if (document.querySelector('[data-fin]')) finPlan(1200);

    document.addEventListener('visibilitychange', function () {
      if (document.visibilityState === 'visible') {
        if (Date.now() - finSon >= FIN_POLL) finPlan(0);
        else finPlan();
      } else {
        window.clearTimeout(finTimer);
      }
    });
    window.addEventListener('pagehide', function () {
      window.clearTimeout(finTimer);
    });
    document.addEventListener('htmx:afterSwap', function () {
      if (document.querySelector('[data-fin]')) finPlan(0);
    });
  }

  function basla() {
    initSeritler();
    var sayfa = document.querySelector('[data-hisse-page]');
    if (sayfa) initArama(sayfa);
    var kripto = document.querySelector('[data-kripto-page]');
    if (kripto) initKriptoArama(kripto);
    finBasla();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', basla);
  } else {
    basla();
  }
})();
