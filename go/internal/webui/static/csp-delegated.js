
(function () {
  'use strict';

  // data-yedek: <img> yüklenemezse yedek adaya geç. Değer VİRGÜLLE
  // ayrılmış ZİNCİR olabilir ("a.jpg,b.jpg,c.jpg"); her hata bir sonrakine
  // geçer, zincir bitince öğe gizlenir (kırık görsel simgesi kalmasın).
  //
  // NEDEN ZİNCİR (ölçüldü, 12 Kasım 2026): YouTube'un dikey kırpımı
  // `oar2.jpg` HER short'ta yok (bazı kimliklerde 404). Tek kademeli yedek
  // o short'ları 480x360 `hqdefault`'a düşürüyordu → dikey kartta 1.4x
  // büyütme, bulanık kapak. Artık sıra: oar2 → maxres → hqdefault.
  //
  // Satır içi `onerror` CSP `script-src 'self'` altında ÇALIŞMAZ; bu delege
  // dinleyici aynı işi yapar.
  document.addEventListener('error', function (e) {
    var el = e.target;
    if (!el || el.tagName !== 'IMG') return;
    var zincir = el.getAttribute('data-yedek');
    if (!zincir) return;
    var adaylar = zincir.split(',');
    var i = parseInt(el.dataset.yedekSirasi || '0', 10);
    if (!(i >= 0)) i = 0;
    if (i >= adaylar.length) {
      el.style.visibility = 'hidden';
      return;
    }
    el.dataset.yedekSirasi = String(i + 1);
    // srcset varsa kaldır: yoksa tarayıcı onu yeniden seçip 404'te ısrar eder.
    el.removeAttribute('srcset');
    el.src = adaylar[i];
  }, true); // capture: error olayı kabarmaz

  function htmxVar() {
    return typeof window.htmx !== 'undefined' && window.htmx !== null;
  }

  // --- data-set-alan: submit'ten ÖNCE alanları doldur ---
  document.addEventListener('submit', function (e) {
    var f = e.target;
    if (!f || !f.getAttribute) return;
    var spec = f.getAttribute('data-set-alan');
    if (!spec) return;
    spec.split(';').forEach(function (cift) {
      var i = cift.indexOf('=');
      if (i < 0) return;
      var ad = cift.slice(0, i).trim();
      var deger = cift.slice(i + 1);
      var alan = f.querySelector('[name="' + ad + '"]');
      if (alan) alan.value = deger;
    });
  }, true);

  // --- data-confirm: onay ---
  function onayGerekli(el) {
    if (!el || !el.getAttribute) return true;
    if (el.hasAttribute('data-confirm-her-zaman')) return true;
    // htmx isteği kendi gönderir; submit'i biz iptal edemeyiz.
    if (htmxVar() && (el.hasAttribute('hx-post') || el.hasAttribute('hx-get') ||
        el.hasAttribute('hx-delete') || el.hasAttribute('hx-put'))) {
      return false;
    }
    return true;
  }

  document.addEventListener('submit', function (e) {
    var f = e.target;
    if (!f || !f.getAttribute) return;
    var mesaj = f.getAttribute('data-confirm');
    if (!mesaj || !onayGerekli(f)) return;
    if (!window.confirm(mesaj)) {
      e.preventDefault();
      e.stopPropagation();
    }
  }, true);

  document.addEventListener('click', function (e) {
    var el = e.target.closest ? e.target.closest('[data-confirm]') : null;
    if (!el) return;
    // Form submit'i yukarıdaki submit dinleyicisi ele alır; burada yalnız
    // form dışı öğeler (buton/bağlantı) için onay sorulur.
    if (el.tagName === 'FORM') return;
    var mesaj = el.getAttribute('data-confirm');
    if (!mesaj || !onayGerekli(el)) return;
    if (!window.confirm(mesaj)) {
      e.preventDefault();
      e.stopPropagation();
    }
  }, true);

  // --- data-autosubmit ---
  document.addEventListener('change', function (e) {
    var el = e.target;
    if (!el || !el.getAttribute || !el.hasAttribute('data-autosubmit')) return;
    var f = el.form;
    if (!f) return;
    if (htmxVar() && (f.hasAttribute('hx-post') || f.hasAttribute('hx-get'))) {
      return; // htmx kendi tetikler
    }
    f.submit();
  }, true);

  // --- data-hata-gizle: bozuk görseli kaldır (error yakalama fazında) ---
  document.addEventListener('error', function (e) {
    var el = e.target;
    if (el && el.hasAttribute && el.hasAttribute('data-hata-gizle')) {
      el.remove();
    }
  }, true);

  // --- data-sec: içeriği seç ---
  document.addEventListener('click', function (e) {
    var el = e.target.closest ? e.target.closest('[data-sec]') : null;
    if (el && el.select) el.select();
  }, true);

  // --- data-kopyala ---
  document.addEventListener('click', function (e) {
    var el = e.target.closest ? e.target.closest('[data-kopyala]') : null;
    if (!el) return;
    var deger = el.getAttribute('data-copy') || '';
    var eski = el.textContent;
    function geri() { el.textContent = eski; }
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(deger).then(function () {
        el.textContent = 'Kopyalandı';
        setTimeout(geri, 1500);
      }).catch(geri);
    } else {
      // Güvenli bağlam yoksa (http) sessizce başarısız ol.
      geri();
    }
  }, true);

  // --- data-git: <select> değeri bir adres; değişince oraya git ---
  //
  // Dil seçici için. Satır içi `onchange` CSP tarafından ÇALIŞTIRILMAZ
  // ('unsafe-inline' yok) — bu yüzden delege dinleyici şart.
  document.addEventListener('change', function (e) {
    var el = e.target;
    if (!el || !el.hasAttribute || !el.hasAttribute('data-git')) return;
    var hedef = el.value;
    // Yalnız aynı-site yolları: '//evil.com' ve 'http://…' reddedilir
    // (açık yönlendirme koruması; aynı kural sunucuda da var).
    if (hedef && hedef.charAt(0) === '/' && hedef.charAt(1) !== '/') {
      window.location.href = hedef;
    }
  }, true);

  // --- data-tema: açık/koyu tema anahtarı ---
  //
  // Başlangıç sınıfı cd_head.templ'deki BLOKLAYICI init betiğinde yazılır
  // (FOUC'u önlemek için); burada yalnız kullanıcı tıklaması işlenir.
  // Tercih localStorage'da saklanır, böylece işletim sistemi ayarı
  // değişse bile kullanıcının seçimi korunur (CoinDesk deseni).
  function temaDugmeleri() {
    var koyu = document.documentElement.classList.contains('dark');
    var dugmeler = document.querySelectorAll('[data-tema]');
    for (var i = 0; i < dugmeler.length; i++) {
      dugmeler[i].setAttribute('aria-pressed', koyu ? 'true' : 'false');
      // Etiketler CoinDesk'in kendi metinleri (ölçüm 7 Kasım 2026):
      // açık temada "Karanlık moda geç", koyuda "Aydınlık moda geç".
      dugmeler[i].setAttribute('aria-label',
        koyu ? 'Aydınlık moda geç' : 'Karanlık moda geç');
    }
    // MOBİL MENÜDEKİ İKİ SEÇENEKLİ SEGMENT — CoinDesk "Theme" satırı.
    // `role="radiogroup"` + `aria-checked` (CoinDesk deseni); üst çubuktaki
    // tek düğme `aria-pressed` kullanmaya devam eder.
    var secenekler = document.querySelectorAll('[data-tema-deger]');
    for (var j = 0; j < secenekler.length; j++) {
      var secili = (secenekler[j].getAttribute('data-tema-deger') === 'dark') === koyu;
      secenekler[j].setAttribute('aria-checked', secili ? 'true' : 'false');
    }
  }

  function temaAyarla(koyu) {
    var kok = document.documentElement;
    kok.classList.remove('light', 'dark');
    kok.classList.add(koyu ? 'dark' : 'light');
    try { localStorage.setItem('tema', koyu ? 'dark' : 'light'); } catch (err) {  }
    temaDugmeleri();
  }

  document.addEventListener('click', function (e) {
    var el = e.target;
    if (!el || !el.closest) return;
    var secenek = el.closest('[data-tema-deger]');
    if (secenek) {
      temaAyarla(secenek.getAttribute('data-tema-deger') === 'dark');
      return;
    }
    var btn = el.closest('[data-tema]');
    if (!btn) return;
    temaAyarla(!document.documentElement.classList.contains('dark'));
  }, true);

  temaDugmeleri();

  // --- cd-kaydi: fiyat şeridinin kaydırma durumu ---
  //
  // Kullanıcı isteği: sayfa başında şerit SADE (iki satır, yüzde gizli);
  // sticky menü aşağı inince GENİŞ görünüm (tek satır + yüzde). Eşik
  // 80 px: menü ~73 px, şerit 67 px — ilk ekranın üstünde kalıyor.
  // Sınıf <html> üzerinde: CSS `html.cd-kaydi .cd-ticker__item` ile
  // bütün şeridi tek kuralla çeviriyor.
  //
  // MOBİLDE KAPALI: CoinDesk mobilde şerit yapışkan değil, sayfayla
  // kayıp gidiyor; tek satırlık "kaydırılmış" görünüm orada yok (ölçüldü
  // 9 Kasım 2026). Sınıf yalnız ≥1024 px'te eklenir; ekran daralınca
  // değişiklik olayı sınıfı temizler.
  var kaydiSorgu = window.matchMedia('(min-width: 1024px)');
  function kaydiGuncelle() {
    var kaydi = kaydiSorgu.matches && (window.scrollY || window.pageYOffset || 0) > 80;
    document.documentElement.classList.toggle('cd-kaydi', kaydi);
  }
  kaydiGuncelle();
  window.addEventListener('scroll', kaydiGuncelle, { passive: true });
  if (kaydiSorgu.addEventListener) {
    kaydiSorgu.addEventListener('change', kaydiGuncelle);
  }

  // --- data-arama-ac: arama paneli (CoinDesk komut paleti) ---
  //
  // CoinDesk'te arama düğmesi sayfa değiştirmez; üstte 560 px'lik bir
  // panel açar. Panel `hidden` gelir, form doğrudan /arama?q=... adresine
  // gider (JS kapalıysa da çalışır). Açma/kapama, çip süzgeci ve
  // "/" · Ctrl+K kısayolları burada — CSP 'unsafe-inline' yok.
  (function () {
    var panel = document.getElementById('cd-arama');
    if (!panel) return;
    var acDugmeler = document.querySelectorAll('[data-arama-ac]');
    var sonAc = null;
    var girdi = panel.querySelector('.cd-arama__girdi');
    var kok = document.documentElement;
    var mobilMenu = document.querySelector('[data-mobil-menu]');

    // Kaydırma çubuğu telafisi.
    //
    // `scrollbar-gutter: stable` modern tarayıcılarda yeri zaten ayırır.
    // Desteklemeyen tarayıcıda çubuk kaybolur ve görünüm 15 px genişleyerek
    // sayfayı titretir (ölçüldü, 12 Kasım 2026).
    //
    // ÖLÇÜT `clientWidth` DEĞİL: gutter ayrıldığında bile clientWidth tam
    // görünür (1440) döner, çünkü gutter clientWidth'e sayılmaz. İlk sürüm
    // bu yüzden gereksiz yere 15 px daha ekleyip içeriği 1410'a düşürdü.
    // Gerçek ölçü, sayfa gövdesinin genişliği.
    var yedekPad = '';
    function govdeGenislik() {
      return Math.round(document.body.getBoundingClientRect().width);
    }
    function kilitKur() {
      var once = govdeGenislik();
      kok.classList.add('cd-arama-acik');
      var fark = govdeGenislik() - once;
      if (fark > 0) {
        yedekPad = kok.style.paddingRight;
        kok.style.paddingRight = fark + 'px';
      }
    }
    function kilitCoz() {
      kok.classList.remove('cd-arama-acik');
      if (yedekPad !== '') {
        kok.style.paddingRight = yedekPad;
        yedekPad = '';
      }
    }

    function ac() {
      panel.hidden = false;
      kilitKur();
      // Mobil menü açıksa kapanır: panel onun üstünü örtüyor.
      if (mobilMenu && mobilMenu.open) mobilMenu.open = false;
      for (var i = 0; i < acDugmeler.length; i++) {
        acDugmeler[i].setAttribute('aria-expanded', 'true');
      }
      if (girdi) girdi.focus();
    }
    function kapat() {
      panel.hidden = true;
      kilitCoz();
      for (var i = 0; i < acDugmeler.length; i++) {
        acDugmeler[i].setAttribute('aria-expanded', 'false');
      }
      var geri = sonAc || acDugmeler[0];
      if (geri) geri.focus();
    }

    document.addEventListener('click', function (e) {
      var el = e.target;
      if (!el || !el.closest) return;
      var acEl = el.closest('[data-arama-ac]');
      if (acEl) { e.preventDefault(); sonAc = acEl; ac(); return; }
      if (el.closest('[data-arama-kapat]')) { e.preventDefault(); kapat(); return; }
      var cip = el.closest('[data-arama-cip]');
      if (!cip) return;
      e.preventDefault();
      var tur = cip.getAttribute('data-arama-cip');
      var cipler = panel.querySelectorAll('[data-arama-cip]');
      for (var i = 0; i < cipler.length; i++) {
        var aktif = cipler[i] === cip;
        cipler[i].classList.toggle('cd-arama__cip--aktif', aktif);
        cipler[i].setAttribute('aria-pressed', aktif ? 'true' : 'false');
      }
      var gruplar = panel.querySelectorAll('[data-arama-grup]');
      for (var j = 0; j < gruplar.length; j++) {
        gruplar[j].hidden = tur !== 'hepsi' &&
          gruplar[j].getAttribute('data-arama-grup') !== tur;
      }
    }, true);

    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && !panel.hidden) { kapat(); return; }
      var ae = document.activeElement;
      var yaziyor = ae && (/^(INPUT|TEXTAREA|SELECT)$/.test(ae.tagName) || ae.isContentEditable);
      var kisayol = (e.key === '/' && !yaziyor) ||
        ((e.ctrlKey || e.metaKey) && (e.key === 'k' || e.key === 'K'));
      if (kisayol) { e.preventDefault(); ac(); }
    });
  })();

  // --- data-hesap: üst çubuk hesap menüsü ---
  //
  // Açılış/kapama <details>'in KENDİ davranışında (JS'siz de çalışır;
  // klavye ve dokunmatik bedava). Burada yalnız JS'in ekleyebileceği iki
  // kapanış var: menünün DIŞINA tıklama ve Escape. İkisi de yoksa menü
  // özet düğmesine tekrar basılana kadar açık kalır — kullanıcıyı
  // kilitlemez.
  (function () {
    // `data-giris-menu`: ANONİM mobil giriş paneli (ayrı semantik —
    // oturum yokken "hesap menüsü" basılmaz kuralını bozmaz).
    var kutular = document.querySelectorAll('[data-hesap],[data-giris-menu]');
    if (!kutular.length) return;

    document.addEventListener('click', function (e) {
      var el = e.target;
      if (!el || !el.closest) return;
      for (var i = 0; i < kutular.length; i++) {
        var k = kutular[i];
        if (k.open && !k.contains(el)) k.open = false;
      }
    });

    document.addEventListener('keydown', function (e) {
      if (e.key !== 'Escape') return;
      for (var i = 0; i < kutular.length; i++) {
        var k = kutular[i];
        if (!k.open) continue;
        k.open = false;
        var ozet = k.querySelector('summary');
        if (ozet) ozet.focus();
      }
    });
  })();

  // --- data-mobil-menu: hamburger paneli ---
  //
  // Açılış/kapama <details>'in KENDİ davranışında (JS'siz de çalışır).
  // Burada yalnız JS'in ekleyebileceği iki kapanış var: panelin DIŞINA
  // tıklama ve Escape — hesap menüsüyle aynı desen (cd_hesap.go notu).
  (function () {
    var menu = document.querySelector('[data-mobil-menu]');
    if (!menu) return;

    document.addEventListener('click', function (e) {
      if (!menu.open) return;
      var el = e.target;
      if (el && el.closest && el.closest('[data-mobil-menu]')) return;
      menu.open = false;
    });

    document.addEventListener('keydown', function (e) {
      if (e.key !== 'Escape' || !menu.open) return;
      menu.open = false;
      var ozet = menu.querySelector('summary');
      if (ozet) ozet.focus();
    });
  })();

  // --- data-ticker-akis: kesintisiz fiyat şeridi -----------------------
  //
  // Kalemler JS ile GEREKTİĞİ KADAR çoğaltılır ve kayma TAM BİR TAKIM
  // boyu yapılır. Sabit iki kopya yetmiyordu: kalemler ekrandan dar
  // olabiliyor (ölçüldü: tek takım 913 px, görünür alan 1193 px), o
  // yüzden animasyon bitince sağda boşluk kalıyordu.
  //
  // Kopyalar `aria-hidden` taşır — ekran okuyucu aynı fiyatı N kez okumaz.
  (function () {
    var akis = document.querySelector('[data-ticker-akis]');
    if (!akis) return;
    var serit = akis.querySelector('[data-ticker-serit]');
    var grup = akis.querySelector('[data-ticker-grup]');
    if (!serit || !grup) return;

    var HIZ = 25; // piksel/saniye — sabit görsel hız, kalem sayısından bağımsız
    var zaman = null;

    function kur() {
      // Önce eski kopyaları temizle (yeniden boyutlandırmada tekrar kurulur).
      var eski = serit.querySelectorAll('[data-ticker-grup][aria-hidden="true"]');
      for (var i = 0; i < eski.length; i++) eski[i].parentNode.removeChild(eski[i]);
      serit.classList.remove('is-akis');

      var tek = grup.getBoundingClientRect().width;
      var alan = akis.getBoundingClientRect().width;
      if (tek <= 0 || alan <= 0) return;

      // Alanı dolduracak kadar kopya: en az 1 tur + ekran genişliği kadar
      // içerik her an görünür olsun diye +1 pay.
      var adet = Math.ceil(alan / tek) + 1;
      for (var k = 1; k < adet; k++) {
        var kopya = grup.cloneNode(true);
        kopya.setAttribute('aria-hidden', 'true');
        kopya.removeAttribute('data-ticker-grup');
        serit.appendChild(kopya);
      }

      serit.style.setProperty('--cd-ticker-adim', tek + 'px');
      serit.style.setProperty('--cd-ticker-sure', (tek / HIZ).toFixed(1) + 's');
      serit.classList.add('is-akis');
    }

    function yeniden() {
      if (zaman) clearTimeout(zaman);
      zaman = setTimeout(kur, 150);
    }

    // Görseller/fontlar yüklendikten sonra genişlik değişebilir.
    kur();
    window.addEventListener('load', yeniden);
    window.addEventListener('resize', yeniden);
    if (document.fonts && document.fonts.ready) document.fonts.ready.then(yeniden);
  })();
})();
