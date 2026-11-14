/* cd-cerez.js — çerez rızası (KVKK açık rıza / GDPR consent).
 *
 * NE YAPAR:
 *   · İlk ziyarette alt köşede rıza kartı gösterir: "Yalnızca zorunlu",
 *     "Tercihleri yönet", "Tümünü kabul et". Ret, kabul kadar kolaydır
 *     (aynı boyutta düğme, tek tık) — rıza özgür iradeyle verilmiş
 *     sayılabilsin diye.
 *   · Kararı `tb_riza` çerezinde saklar: "<sürüm>:<a0|a1><r0|r1>:<ISO tarih>".
 *     a = analitik, r = reklam. Sürüm/tarih ispat içindir.
 *   · Kararı `window.cdRiza` ile diğer betiklere açar ve değiştiğinde
 *     `cd-riza` olayını yayar. Üçüncü taraf betikleri (GA4, GTM, reklam
 *     ağları) cd-entegrasyon.js YALNIZ ilgili rıza varsa yükler.
 *   · Altbilgideki "Çerez tercihleri" bağlantısı ([data-cerez-ayar]) kartı
 *     yeniden açar: rızayı geri almak, vermek kadar kolay olmalı.
 *
 * ZORUNLU ÇEREZLER rızaya bağlı değildir (oturum, güvenlik, bu tercih).
 *
 * CSP UYUMLU: satır içi stil/olay yok; arayüz DOM API ile kurulur, stiller
 * cd-cerez.css içinde. Bu dosya cd-entegrasyon.js'ten ÖNCE yüklenmelidir.
 */
(function () {
  'use strict';

  var CEREZ = 'tb_riza';
  // Çerez politikası ya da kullanılan üçüncü taraflar değişirse artır:
  // eski kararlar geçersizleşir ve rıza yeniden sorulur.
  var SURUM = '1';
  var GUN = 180;

  var en = (document.documentElement.lang || 'tr').toLowerCase().indexOf('en') === 0;
  var M = en ? {
    baslik: 'Your cookie choices',
    metin: 'We use necessary cookies to run the site. With your consent we also use analytics cookies to measure visits and advertising cookies to show and measure ads; these involve third parties located outside your country. You can change your choice at any time.',
    politika: 'Cookie Policy',
    reddet: 'Necessary only',
    yonet: 'Manage preferences',
    kabul: 'Accept all',
    kaydet: 'Save my choices',
    zorunlu: 'Necessary',
    zorunluAcik: 'Session, security and your cookie choice. Always on.',
    analitik: 'Analytics',
    analitikAcik: 'Google Analytics / Tag Manager: page views and traffic sources.',
    reklam: 'Advertising',
    reklamAcik: 'Coinzilla and Cointraffic: showing ads and measuring their performance.',
    hep: 'Always on',
    etiket: 'Cookie preferences'
  } : {
    baslik: 'Çerez tercihleriniz',
    metin: 'Sitenin çalışması için zorunlu çerezleri kullanıyoruz. Onay verirseniz ziyaretleri ölçmek için analitik, reklam göstermek ve ölçmek için reklam çerezlerini de kullanırız; bu çerezler yurt dışındaki üçüncü taraflara veri aktarımı içerir. Tercihinizi dilediğiniz zaman değiştirebilirsiniz.',
    politika: 'Çerez Politikası',
    reddet: 'Yalnızca zorunlu',
    yonet: 'Tercihleri yönet',
    kabul: 'Tümünü kabul et',
    kaydet: 'Seçimimi kaydet',
    zorunlu: 'Zorunlu',
    zorunluAcik: 'Oturum, güvenlik ve çerez tercihiniz. Her zaman açık.',
    analitik: 'Analitik',
    analitikAcik: 'Google Analytics / Tag Manager: sayfa görüntüleme ve trafik kaynağı ölçümü.',
    reklam: 'Reklam',
    reklamAcik: 'Coinzilla ve Cointraffic: reklam gösterimi ve reklam performansı ölçümü.',
    hep: 'Her zaman açık',
    etiket: 'Çerez tercihleri'
  };

  function cerezOku(ad) {
    var parca = ('; ' + document.cookie).split('; ' + ad + '=');
    return parca.length === 2 ? decodeURIComponent(parca.pop().split(';')[0]) : '';
  }

  function kararOku() {
    var p = cerezOku(CEREZ).split(':');
    if (p[0] !== SURUM || !/^a[01]r[01]$/.test(p[1] || '')) return null;
    return { analitik: p[1].charAt(1) === '1', reklam: p[1].charAt(3) === '1' };
  }

  function kararYaz(k) {
    var d = new Date();
    var deger = SURUM + ':a' + (k.analitik ? 1 : 0) + 'r' + (k.reklam ? 1 : 0) + ':' + d.toISOString();
    d.setTime(d.getTime() + GUN * 864e5);
    document.cookie = CEREZ + '=' + encodeURIComponent(deger) + '; expires=' + d.toUTCString() +
      '; path=/; SameSite=Lax' + (location.protocol === 'https:' ? '; Secure' : '');
  }

  // Rıza geri alındığında bu tarayıcıda kalan Google Analytics çerezlerini
  // sil. Daha önce iletilmiş veri geri alınamaz; yalnız tanımlayıcılar silinir.
  function analitikCerezleriSil() {
    var hepsi = document.cookie.split(';');
    for (var i = 0; i < hepsi.length; i++) {
      var ad = hepsi[i].split('=')[0].trim();
      if (!/^_ga|^_gid|^_gat/.test(ad)) continue;
      var kokler = ['', location.hostname, '.' + location.hostname.replace(/^www\./, '')];
      for (var k = 0; k < kokler.length; k++) {
        document.cookie = ad + '=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/' +
          (kokler[k] ? '; domain=' + kokler[k] : '');
      }
    }
  }

  var karar = kararOku();
  // Diğer betikler için tek doğruluk kaynağı. Karar yokken ikisi de kapalı.
  window.cdRiza = { analitik: !!(karar && karar.analitik), reklam: !!(karar && karar.reklam), verildi: !!karar };

  function uygula(k) {
    var onceki = window.cdRiza;
    kararYaz(k);
    window.cdRiza = { analitik: k.analitik, reklam: k.reklam, verildi: true };
    if (!k.analitik) analitikCerezleriSil();
    kapat();
    // Bir izin GERİ ALINDIYSA yüklenmiş üçüncü taraf betikleri bellekten
    // atılamaz; temiz durum için sayfa yenilenir. İzin verildiyse yenileme
    // gerekmez: cd-entegrasyon.js olayı dinleyip eksik betikleri yükler.
    if ((onceki.analitik && !k.analitik) || (onceki.reklam && !k.reklam)) {
      location.reload();
      return;
    }
    document.dispatchEvent(new CustomEvent('cd-riza', { detail: window.cdRiza }));
  }

  // --- Arayüz -------------------------------------------------------------

  var kart = null;
  var oncekiOdak = null;

  function el(ad, sinif, metin) {
    var e = document.createElement(ad);
    if (sinif) e.className = sinif;
    if (metin) e.textContent = metin;
    return e;
  }

  function dugme(sinif, metin, is) {
    var b = el('button', 'cd-cerez__dugme ' + sinif, metin);
    b.type = 'button';
    b.addEventListener('click', is);
    return b;
  }

  function satir(anahtar, secili, kilitli) {
    var s = el('label', 'cd-cerez__satir');
    var metin = el('span', 'cd-cerez__satir-metin');
    metin.appendChild(el('b', '', M[anahtar]));
    metin.appendChild(el('small', '', M[anahtar + 'Acik']));
    s.appendChild(metin);
    if (kilitli) {
      s.appendChild(el('span', 'cd-cerez__hep', M.hep));
    } else {
      var kutu = el('input', 'cd-cerez__anahtar');
      kutu.type = 'checkbox';
      kutu.checked = secili;
      kutu.setAttribute('data-riza', anahtar);
      s.appendChild(kutu);
    }
    return s;
  }

  function kapat() {
    if (!kart) return;
    kart.remove();
    kart = null;
    if (oncekiOdak && oncekiOdak.focus) oncekiOdak.focus();
  }

  function ac(ayrintili) {
    if (kart) kapat();
    oncekiOdak = document.activeElement;
    var mevcut = window.cdRiza;

    kart = el('section', 'cd-cerez');
    kart.setAttribute('role', 'dialog');
    kart.setAttribute('aria-label', M.etiket);

    kart.appendChild(el('h2', 'cd-cerez__baslik', M.baslik));
    var p = el('p', 'cd-cerez__metin', M.metin + ' ');
    var bag = el('a', '', M.politika);
    bag.href = '/cerez-politikasi';
    p.appendChild(bag);
    kart.appendChild(p);

    var liste = el('div', 'cd-cerez__liste');
    liste.hidden = !ayrintili;
    liste.appendChild(satir('zorunlu', true, true));
    liste.appendChild(satir('analitik', mevcut.analitik, false));
    liste.appendChild(satir('reklam', mevcut.reklam, false));
    kart.appendChild(liste);

    var dugmeler = el('div', 'cd-cerez__dugmeler');
    var reddet = dugme('cd-cerez__dugme--ikincil', M.reddet, function () {
      uygula({ analitik: false, reklam: false });
    });
    var yonet = dugme('cd-cerez__dugme--metin', M.yonet, function () {
      liste.hidden = false;
      yonet.hidden = true;
      kaydet.hidden = false;
    });
    var kaydet = dugme('cd-cerez__dugme--ikincil cd-cerez__dugme--tam', M.kaydet, function () {
      uygula({
        analitik: kart.querySelector('[data-riza="analitik"]').checked,
        reklam: kart.querySelector('[data-riza="reklam"]').checked
      });
    });
    var kabul = dugme('cd-cerez__dugme--ana', M.kabul, function () {
      uygula({ analitik: true, reklam: true });
    });
    yonet.hidden = !!ayrintili;
    kaydet.hidden = !ayrintili;
    dugmeler.appendChild(reddet);
    dugmeler.appendChild(yonet);
    dugmeler.appendChild(kaydet);
    dugmeler.appendChild(kabul);
    kart.appendChild(dugmeler);

    document.body.appendChild(kart);
  }

  function basla() {
    // Altbilgideki "Çerez tercihleri" bağlantısı kartı ayrıntılı açar.
    document.addEventListener('click', function (e) {
      var t = e.target.closest ? e.target.closest('[data-cerez-ayar]') : null;
      if (!t) return;
      e.preventDefault();
      ac(true);
    });
    if (!karar) ac(false);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', basla);
  } else {
    basla();
  }
})();
