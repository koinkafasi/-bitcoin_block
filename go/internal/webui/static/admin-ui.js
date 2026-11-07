/* admin-ui.js — WiseCP tarzı admin redizayn etkileşimleri (Faz C–E)
 * Breadcrumb, katlanabilir/sabitlenebilir menü, tablo araçları (arama,
 * sıralama, sayfalama), onay modalı, kısayol yardımı. Bağımlılık yok. */
(function () {
  "use strict";

  /* ================= Tercihler: localStorage + sunucu (Faz 173, A-3) ====
   *
   * ÖNCEDEN: grup açık/kapalı durumu ve daralmış menü yalnız
   * localStorage'daydı. Aynı editör başka bir cihazda (ya da gizli sekmede)
   * her şeyi baştan açmak zorunda kalıyordu.
   *
   * ŞİMDİ: sunucu cihazlar arası doğruluk kaynağı, localStorage anlık
   * önbellek. İkisinin de gerekçesi var:
   *   - localStorage: ilk boyamada gecikme olmasın (FOUC) ve ağ yoksa da
   *     panel çalışsın.
   *   - sunucu: tercih cihazdan bağımsız olsun.
   *
   * Sayfa sunucudaki belgeyi <meta name="tb-prefs"> içinde taşır; burada
   * yalnız önbellekte OLMAYAN ya da sunucudakiyle ÇELİŞEN anahtarlar
   * yazılır. Çelişkide sunucu kazanır (son yazan zaten sunucuya yazmıştı).
   *
   * Yazma: localStorage hemen, sunucuya 400 ms geciktirilmiş (bir formda
   * art arda grup açıp kapatmak istek yağmuru üretmesin).
   */
  var SUNUCU_ANAHTARLARI = null;
  try {
    var meta = document.querySelector('meta[name="tb-prefs"]');
    if (meta && meta.content) SUNUCU_ANAHTARLARI = JSON.parse(meta.content);
  } catch (e) { SUNUCU_ANAHTARLARI = null; }

  // Hangi anahtarlar sunucuya yazılır? Tema, yazı boyu gibi GÖRÜNTÜ
  // tercihleri bilinçli olarak yerel kalıyor: masaüstünde koyu, telefonda
  // açık tema isteyen kullanıcı var. Burada yalnız "iş akışı" tercihleri.
  var SENKRON_ONEK = ['tb_form_grup:', 'pn:'];

  function senkronMu(k) {
    for (var i = 0; i < SENKRON_ONEK.length; i++) {
      if (k.indexOf(SENKRON_ONEK[i]) === 0) return true;
    }
    return false;
  }

  var _bekleyen = {}; // anahtar → değer (geciktirilmiş yazma)
  var _zamanlayici = null;

  function sunucuyaYaz(k, v) {
    _bekleyen[k] = v;
    if (_zamanlayici) return;
    _zamanlayici = setTimeout(function () {
      _zamanlayici = null;
      var govde = [];
      for (var anahtar in _bekleyen) {
        if (!Object.prototype.hasOwnProperty.call(_bekleyen, anahtar)) continue;
        govde.push({ k: anahtar, v: _bekleyen[anahtar] });
      }
      _bekleyen = {};
      govde.forEach(function (o) {
        try {
          fetch('/admin/tercih', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            // Tercih kaybı sessiz olmalı: kullanıcıya hata balonu
            // göstermek, bir menüyü daralttığı için rahatsız edici olur.
            keepalive: true,
            body: JSON.stringify(o)
          }).catch(function () {});
        } catch (e) {}
      });
    }, 400);
  }

  var LS = {
    get: function (k, d) {
      try {
        var v = localStorage.getItem(k);
        if (v !== null) return JSON.parse(v);
      } catch (e) {}
      // Yerelde yok: sunucudaki değere düş ve önbelleğe yaz.
      if (SUNUCU_ANAHTARLARI && Object.prototype.hasOwnProperty.call(SUNUCU_ANAHTARLARI, k)) {
        var sv = SUNUCU_ANAHTARLARI[k];
        try { localStorage.setItem(k, JSON.stringify(sv)); } catch (e) {}
        return sv;
      }
      return d;
    },
    set: function (k, v) {
      try { localStorage.setItem(k, JSON.stringify(v)); } catch (e) {}
      if (senkronMu(k)) sunucuyaYaz(k, v);
    }
  };

  // Sayfa açılışında sunucu değerleri önbelleğe tohumlanır. Çelişkide
  // sunucu kazanır; böylece başka cihazda yapılan değişiklik buraya gelir.
  if (SUNUCU_ANAHTARLARI) {
    Object.keys(SUNUCU_ANAHTARLARI).forEach(function (k) {
      if (!senkronMu(k)) return;
      try { localStorage.setItem(k, JSON.stringify(SUNUCU_ANAHTARLARI[k])); } catch (e) {}
    });
  }

  /* ================= Breadcrumb ================= */
  // İsimli fonksiyon: içerik AJAX ile değişince yeniden çalışması gerekiyor
  // (bkz. tbAdminInit). IIFE olsaydı yalnız ilk yüklemede kurulurdu.
  function tbCrumb() {
    var el = document.getElementById('adminCrumb');
    if (!el) return;
    var parts = location.pathname.split('/').filter(Boolean); // ['admin','icerikler','123']
    // URL yolundan gelen parçalar HTML olarak birleştirilmez: adres çubuğu
    // saldırgan denetiminde olabilir (yansıtılmış DOM XSS). Düğümler
    // textContent ile kurulur.
    el.textContent = '';
    var ilk = document.createElement('a');
    ilk.href = '/admin';
    ilk.textContent = 'Pano';
    el.appendChild(ilk);
    var acc = '';
    var names = {
      icerikler: 'İçerikler', yeni: 'Yeni', dosyalar: 'Medya', kategoriler: 'Kategoriler',
      etiketler: 'Etiketler', yorumlar: 'Yorumlar', 'geri-bildirim': 'Geri Bildirim',
      members: 'Üyelikler', anketler: 'Anketler', etkinlikler: 'Etkinlikler',
      bildirimler: 'Bildirimler', 'ai-sohbet': 'AI Sohbet', seo: 'SEO', backlink: 'İç Link Ağı',
      indexleme: 'İndeksleme', tasima: 'Taşıma', paylasim: 'Sosyal Paylaşım', bulten: 'Bülten',
      gazete: 'Gazete', gazeteler: 'Gazete Manşetleri', sinemalar: 'Sinemalar',
      'resmi-ilanlar': 'Resmi İlanlar', reklamlar: 'Reklamlar', adsense: 'AdSense / GA',
      raporlar: 'Raporlar', siteler: 'Siteler', bot: 'Bot', 'ai-kaynak': 'AI Kaynak',
      ayarlar: 'Ayarlar', gorunum: 'Görünüm', merkez: 'Site Merkezi', menu: 'Menü',
      iletisim: 'İletişim & Form',
      header: 'Header', footer: 'Footer', marka: 'Marka', mahrecler: 'Mahreçler',
      kumeler: 'Kümeler', servisler: 'Servisler', 'manset-editor': 'Manşet Editörü',
      tasarim: 'Sayfa Tasarımcısı', 'tab-kategorileri': 'Sekme Kategorileri',
      cekirdek: 'Çekirdek', sablonlar: 'Şablonlar', sayilar: 'Sayılar', duzenle: 'Düzenle',
      aboneler: 'Aboneler', kaydedilenler: 'Kaydedilenler', rozetler: 'Rozetler'
    };
    for (var i = 1; i < parts.length; i++) {
      acc += '/' + parts.slice(0, i + 1).join('/');
      var isLast = i === parts.length - 1;
      var label = names[parts[i]] || parts[i];
      var sep = document.createElement('span');
      sep.className = 'sep';
      sep.textContent = '/';
      el.appendChild(sep);
      if (isLast) {
        var b = document.createElement('b');
        b.textContent = label;
        el.appendChild(b);
      } else {
        var a = document.createElement('a');
        a.href = acc;
        a.textContent = label;
        el.appendChild(a);
      }
    }
  }

  /* ================= Sidebar: katlanabilir gruplar + pin ================= */
  (function sidebar() {
    var side = document.querySelector('.admin-side');
    if (!side) return;
    var menus = side.querySelectorAll('ul.menu');
    var pins = LS.get('tb_nav_pins', []);

    // Katlanabilir gruplar
    menus.forEach(function (ul, idx) {
      var head = ul.querySelector('.menu-h');
      if (!head) return;
      var key = 'tb_nav_grp_' + idx;
      if (LS.get(key, false)) {
        ul.classList.add('group-collapsed');
        head.classList.add('collapsed');
      }
      head.addEventListener('click', function () {
        var collapsed = ul.classList.toggle('group-collapsed');
        head.classList.toggle('collapsed', collapsed);
        LS.set(key, collapsed);
      });
    });

    // Pin: her bağlantıya düğme ekle
    var links = [];
    menus.forEach(function (ul) {
      if (ul.classList.contains('pinned-menu')) return;
      ul.querySelectorAll(':scope > li > a').forEach(function (a) {
        links.push(a);
        var pin = document.createElement('button');
        pin.type = 'button';
        pin.className = 'pin-btn';
        var href = a.getAttribute('href') || '';
        var isPinned = pins.indexOf(href) >= 0;

        // Emoji yerine yerel SVG sprite. Sebep: buton <a> içinde olduğu için
        // emoji metni bağlantının textContent'ine giriyordu — menüde her
        // öğenin adı "Pano" + emoji diye okunuyor, ekran okuyucu ve
        // kopyala-yapıştır bozuluyordu. Ayrıca emoji genişliği etiketi
        // 3px kaydırıyordu.
        function pinCiz() {
          pin.innerHTML = '<svg class="ti" aria-hidden="true"><use href="/static/vendor/tabler-icons.svg?v=b89545a6#ti-pin"/></svg>';
          pin.classList.toggle('pinned', isPinned);
          var etiket = isPinned ? 'Sabitlemeyi kaldır' : 'Sabitle';
          pin.title = etiket;
          pin.setAttribute('aria-label', etiket);
          pin.setAttribute('aria-pressed', isPinned ? 'true' : 'false');
        }
        pinCiz();

        pin.addEventListener('click', function (ev) {
          ev.preventDefault();
          ev.stopPropagation();
          var i = pins.indexOf(href);
          if (i >= 0) { pins.splice(i, 1); isPinned = false; }
          else { pins.push(href); isPinned = true; }
          LS.set('tb_nav_pins', pins);
          pinCiz();
          renderPins();
        });
        a.appendChild(pin);
      });
    });

    function renderPins() {
      var old = side.querySelector('.pinned-menu');
      if (old) old.remove();
      if (!pins.length) return;
      var ul = document.createElement('ul');
      ul.className = 'menu pinned-menu';
      var head = document.createElement('li');
      head.className = 'menu-h';
      // Emoji yerine yerel sprite (Faz 161). Başlık metni <ul> içinde
      // olduğu için emoji, "Sabitlenmişler" adıyla birlikte okunuyordu.
      head.textContent = 'Sabitlenmişler';
      head.insertAdjacentHTML('afterbegin',
        '<svg class="ti" aria-hidden="true"><use href="/static/vendor/tabler-icons.svg?v=b89545a6#ti-pin"></use></svg>');
      ul.appendChild(head);
      pins.forEach(function (href) {
        var src = links.find(function (a) { return a.getAttribute('href') === href; });
        if (!src) return;
        var li = document.createElement('li');
        var a = document.createElement('a');
        a.href = href;
        a.innerHTML = src.innerHTML;
        var p = a.querySelector('.pin-btn');
        if (p) p.remove();
        li.appendChild(a);
        ul.appendChild(li);
      });
      // Hedef: ilk menünün KARDEŞİ olarak ekle (aynı sarmalayıcı içinde).
      //
      // DİKKAT: `side.insertBefore(ul, ilkMenu)` YANLIŞ. Menüler doğrudan
      // <aside class="admin-side"> çocuğu değil, <nav class="side-scroll">
      // içinde. insertBefore, referans düğümün DOĞRUDAN çocuk olmasını
      // ister; aksi hâlde "Failed to execute 'insertBefore'" hatası atar ve
      // pin özelliği tamamen çalışmaz (localStorage yazılıyor, grup hiç
      // çizilmiyordu — Faz 210'da ölçüldü).
      var ilkMenu = side.querySelector('ul.menu');
      if (ilkMenu && ilkMenu.parentNode) {
        ilkMenu.parentNode.insertBefore(ul, ilkMenu);
      } else if (side.firstChild) {
        side.insertBefore(ul, side.firstChild);
      } else {
        side.appendChild(ul);
      }
    }
    renderPins();
  })();

  /* ================= Tablo araçları (arama / sıralama / sayfalama) ================= */
  // tabloAdi: araç çubuğundaki arama kutusuna ayırt edici ad vermek için.
  // Önce tablo ÖNCESİNDEKİ en yakın başlık (h2/h3 ve <summary>), yoksa sütun
  // başlıklarından türetilmiş kısa bir ad, yoksa "tablo".
  function tabloAdi(tbl) {
    var kapsam = tbl.parentElement;
    for (var d = 0; kapsam && d < 3; d++, kapsam = kapsam.parentElement) {
      var baslik = kapsam.querySelector('h1, h2, h3, summary');
      if (baslik) {
        var t = (baslik.textContent || '').replace(/\s+/g, ' ').trim();
        if (t) return t.slice(0, 48);
      }
    }
    var th = tbl.tHead && tbl.tHead.rows[0];
    if (th) {
      var sutunlar = Array.prototype.map.call(th.cells, function (c) {
        return (c.textContent || '').replace(/\s+/g, ' ').trim();
      }).filter(Boolean).slice(0, 2);
      if (sutunlar.length) return sutunlar.join(' / ');
    }
    return 'tablo';
  }

  function tbTablolar() {
    document.querySelectorAll('.admin-body table.table').forEach(function (tbl) {
      // İçerik AJAX ile değişince yeniden çağrılıyor: aynı tabloya ikinci
      // kez araç çubuğu eklemek iki arama kutusu demek olurdu.
      //
      // Yalnız JS işareti yetmiyor: geri/ileri tuşunda htmx önbellekten
      // DOM'u geri koyuyor ve özel JS özellikleri kayboluyor (ölçüm: geri
      // tuşundan sonra 2 araç çubuğu). Bu yüzden DOM'a da bakıyoruz.
      var os = tbl.previousElementSibling;
      if (tbl.__tbArac || (os && os.classList.contains('tbl-tools'))) return;
      tbl.__tbArac = true;
      var rows = Array.prototype.slice.call(tbl.tBodies.length ? tbl.tBodies[0].rows : []);
      if (!rows.length) return;
      // Boş durum satırlarını (colspan) işaretle
      rows.forEach(function (r) {
        if (r.cells.length === 1 && r.cells[0].colSpan > 1) r.cells[0].classList.add('tbl-empty');
      });
      rows = rows.filter(function (r) { return !(r.cells.length === 1 && r.cells[0].colSpan > 1); });
      if (!rows.length) return;
      var headRow = tbl.tHead && tbl.tHead.rows[0];
      var colCount = rows[0].cells.length;

      // Araç çubuğu
      var tools = document.createElement('div');
      tools.className = 'tbl-tools';
      var search = document.createElement('input');
      search.className = 'input';
      search.type = 'search';
      search.placeholder = 'Tabloda ara…';
      // Erişilebilirlik (Faz 179, A-9): kutu JS ile üretiliyor, bu yüzden
      // hiçbir etiketi yoktu — ekran okuyucu yalnızca "metin alanı" diyordu.
      // Aynı sayfada birden çok tablo olabildiği için (üyeler: 2, gazete: 2)
      // ad tabloya göre ayrışmalı.
      search.setAttribute('aria-label', 'Tabloda ara — ' + tabloAdi(tbl));
      tools.appendChild(search);
      var info = document.createElement('span');
      info.className = 'tbl-info';
      tools.appendChild(info);
      tbl.parentNode.insertBefore(tools, tbl);

      var PAGE = 12;
      var page = 0, query = '', sortCol = -1, sortAsc = true;

      function apply() {
        var visible = rows.filter(function (r) {
          if (!query) return true;
          return r.textContent.toLocaleLowerCase('tr').indexOf(query) >= 0;
        });
        if (sortCol >= 0) {
          visible.sort(function (a, b) {
            var av = (a.cells[sortCol] ? a.cells[sortCol].textContent : '').trim();
            var bv = (b.cells[sortCol] ? b.cells[sortCol].textContent : '').trim();
            var an = parseFloat(av.replace(',', '.')), bn = parseFloat(bv.replace(',', '.'));
            var cmp = (!isNaN(an) && !isNaN(bn)) ? an - bn : av.localeCompare(bv, 'tr');
            return sortAsc ? cmp : -cmp;
          });
        }
        var pages = Math.max(1, Math.ceil(visible.length / PAGE));
        if (page >= pages) page = pages - 1;
        rows.forEach(function (r) { r.style.display = 'none'; });
        visible.slice(page * PAGE, page * PAGE + PAGE).forEach(function (r) {
          r.style.display = '';
          tbl.tBodies[0].appendChild(r); // sıralamayı DOM'a yansıt
        });
        info.textContent = visible.length + ' kayıt' + (pages > 1 ? ' · sayfa ' + (page + 1) + '/' + pages : '');
        pager.style.display = pages > 1 ? '' : 'none';
        prev.disabled = page === 0;
        next.disabled = page >= pages - 1;
      }

      // Sıralama
      if (headRow) {
        Array.prototype.forEach.call(headRow.cells, function (th, i) {
          if (!th.textContent.trim()) return;
          th.classList.add('sortable');
          th.addEventListener('click', function () {
            if (sortCol === i) sortAsc = !sortAsc; else { sortCol = i; sortAsc = true; }
            headRow.querySelectorAll('th').forEach(function (x) { x.classList.remove('asc', 'desc'); });
            th.classList.add(sortAsc ? 'asc' : 'desc');
            apply();
          });
        });
      }

      // Sayfalama
      var pager = document.createElement('div');
      pager.className = 'tbl-pager';
      var prev = document.createElement('button');
      prev.type = 'button'; prev.textContent = '‹ Önceki';
      var next = document.createElement('button');
      next.type = 'button'; next.textContent = 'Sonraki ›';
      prev.addEventListener('click', function () { if (page > 0) { page--; apply(); } });
      next.addEventListener('click', function () { page++; apply(); });
      pager.appendChild(prev); pager.appendChild(next);
      tbl.parentNode.insertBefore(pager, tbl.nextSibling);

      search.addEventListener('input', function () {
        query = search.value.toLocaleLowerCase('tr');
        page = 0;
        apply();
      });
      apply();
    });
  }

  /* ================= Onay modalı (data-confirm) ================= */
  (function confirmModal() {
    var modal = document.createElement('div');
    modal.className = 'ui-modal';
    modal.innerHTML = '<div class="ui-box"><h3 id="uiConfirmTitle">Onay</h3>' +
      '<p id="uiConfirmText" class="muted"></p><div class="ui-actions">' +
      '<button class="btn ghost" data-ui-no type="button">Vazgeç</button>' +
      '<button class="btn" data-ui-yes type="button">Evet</button></div></div>';
    document.body.appendChild(modal);
    var pending = null;
    modal.querySelector('[data-ui-no]').addEventListener('click', function () {
      modal.classList.remove('open');
      if (pending) pending(false);
    });
    modal.querySelector('[data-ui-yes]').addEventListener('click', function () {
      modal.classList.remove('open');
      if (pending) pending(true);
    });
    window.uiConfirm = function (text) {
      modal.querySelector('#uiConfirmText').textContent = text || 'Emin misiniz?';
      modal.classList.add('open');
      return new Promise(function (resolve) { pending = resolve; });
    };
    // data-confirm formlarini native confirm yerine modala bagla.
    //
    // Olay devri (delegation) kullaniliyor: panel HTMX ile yenilendiginde
    // tek tek baglanan dinleyiciler kayboluyor ve yikici islemler ONAY
    // SORULMADAN gonderiliyordu. Belge duzeyinde dinleyince takastan sonra
    // gelen formlar da kapsaniyor.
    //
    // data-confirm iki yerde olabilir:
    //   - <form data-confirm="...">        → tum form icin onay
    //   - <button type=submit data-confirm="...">  → yalniz o dugme icin
    // Ikincisi gozden kaciyordu: "Varsayilana Don" ve "Tumunde degistir"
    // yikici dugmeleri onay sormadan gonderiliyordu.
    function tbOnayMetni(f, ev) {
      var m = f.getAttribute('data-confirm');
      if (m) return m;
      var g = ev.submitter;
      if (!g) {
        // submitter desteklemeyen tarayicilar / programatik tetikleme.
        var a = document.activeElement;
        if (a && a.form === f && a.hasAttribute('data-confirm')) g = a;
      }
      return (g && g.getAttribute('data-confirm')) || '';
    }

    document.addEventListener('submit', function (ev) {
      var f = ev.target;
      if (!f || f.tagName !== 'FORM') return;
      var metin = tbOnayMetni(f, ev);
      // Deger "/" ile basliyorsa bu bir URL'dir (form disi kural); onay
      // metni degildir. app.js o durumu kendi ustleniyor.
      if (!metin || metin.charAt(0) === '/') return;
      ev.preventDefault();
      // Ayni form icin ikinci kez sorma (htmx.trigger yeniden submit eder).
      if (f.__tbOnaylandi) return;
      window.uiConfirm(metin).then(function (ok) {
        if (!ok) return;
        f.__tbOnaylandi = true;
        // HTMX ile yonetilen formu htmx gondermeli; form.submit() olayi
        // atlatir ve HTMX devreye giremez.
        if (window.htmx && (f.hasAttribute('hx-post') || f.hasAttribute('hx-get'))) {
          window.htmx.trigger(f, 'submit');
        } else {
          f.submit();
        }
        // Bayragi birak: sayfa yenilenmezse ikinci gonderim yine onay sorsun.
        setTimeout(function () { f.__tbOnaylandi = false; }, 0);
      });
    }, true);
  })();

  /* ================= Kısayol yardımı (?) ================= */
  (function help() {
    var modal = document.createElement('div');
    modal.className = 'ui-modal';
    modal.innerHTML = '<div class="ui-box"><h3>Kısayollar</h3><div class="keys">' +
      '<kbd>Ctrl</kbd><b>K</b><span></span><span></span>' +
      '</div></div>';
    // Basit liste
    modal.querySelector('.keys').innerHTML =
      '<kbd>Ctrl+K</kbd><span>Komut paleti / arama</span>' +
      '<kbd>Ctrl+Z</kbd><span>Geri al (editörler)</span>' +
      '<kbd>Ctrl+Shift+Z</kbd><span>Yinele</span>' +
      '<kbd>↕</kbd><span>Tablo yoğunluğu (üst bar)</span>' +
      '<kbd><svg class="ti" aria-hidden="true"><use href="/static/vendor/tabler-icons.svg?v=b89545a6#ti-pin"/></svg></kbd><span>Menüde sabitleme</span>' +
      '<kbd>?</kbd><span>Bu pencere</span>' +
      '<kbd>Esc</kbd><span>Pencereyi kapat</span>';
    document.body.appendChild(modal);
    document.addEventListener('keydown', function (ev) {
      if (ev.target.matches('input,textarea,select')) return;
      if (ev.key === '?') { modal.classList.add('open'); }
      if (ev.key === 'Escape') { modal.classList.remove('open'); }
    });
  })();

  /* ================= Çalışma alanı sekme çubuğu (ADMIN-IA Faz A) =================
   * 11 alanlı sade menünün alt sayfaları; her admin sayfasına otomatik enjekte
   * edilir. URL'ler değişmez; derin linkler korunur. */
  function tbWsTabs() {
    var WS = [
      { id: 'pano', label: 'Pano', match: [/^\/admin$/], items: [['Pano', '/admin']] },
      { id: 'icerik', label: 'İçerik',
        match: [/^\/admin\/icerikler(\/|$)/, /^\/admin\/kategoriler(\/|$)/, /^\/admin\/etiketler(\/|$)/, /^\/admin\/ceviriler(\/|$)/, /^\/admin\/sinemalar(\/|$)/, /^\/admin\/resmi-ilanlar(\/|$)/, /^\/admin\/video-yayin(\/|$)/],
        items: [['İçerikler', '/admin/icerikler'], ['Yeni İçerik', '/admin/icerikler/yeni'], ['Video & Canlı', '/admin/video-yayin'], ['Kategoriler', '/admin/kategoriler'], ['Etiketler', '/admin/etiketler'], ['Çeviriler', '/admin/ceviriler'], ['Sinemalar', '/admin/sinemalar'], ['Resmi İlanlar', '/admin/resmi-ilanlar']],
        action: { label: '＋ Yeni İçerik', href: '/admin/icerikler/yeni', hide: ['/admin/icerikler/yeni'] } },
      { id: 'medya', label: 'Medya', match: [/^\/admin\/dosyalar(\/|$)/], items: [['Medya', '/admin/dosyalar']] },
      { id: 'tasarim', label: 'Site & Tasarım',
        match: [/^\/admin\/tasarim(\/|$)/, /^\/admin\/menu(\/|$)/, /^\/admin\/header(\/|$)/, /^\/admin\/footer(\/|$)/, /^\/admin\/tab-kategorileri(\/|$)/, /^\/admin\/marka(\/|$)/, /^\/admin\/manset-editor(\/|$)/, /^\/admin\/gorunum(\/|$)/],
        items: [['Sayfa Tasarımcısı', '/admin/tasarim'], ['Menü', '/admin/menu'], ['Header', '/admin/header'], ['Footer', '/admin/footer'], ['Sekmeler', '/admin/tab-kategorileri'], ['Marka', '/admin/marka'], ['Manşet Editörü', '/admin/manset-editor'], ['Görünüm', '/admin/gorunum']] },
      { id: 'gazete', label: 'Gazete',
        match: [/^\/admin\/gazete(\/|$)/, /^\/admin\/gazeteler(\/|$)/],
        items: [['E-Baskı', '/admin/gazete'], ['Şablon Galerisi', '/admin/gazete/sablonlar'], ['Manşetler', '/admin/gazeteler']],
        action: { label: 'Yeni Sayı', ikon: 'news', href: '/admin/gazete', hide: ['/admin/gazete'] } },
      { id: 'topluluk', label: 'Topluluk',
        match: [/^\/admin\/yorumlar(\/|$)/, /^\/admin\/geri-bildirim(\/|$)/, /^\/admin\/members(\/|$)/, /^\/admin\/yazarlar(\/|$)/, /^\/admin\/anketler(\/|$)/, /^\/admin\/etkinlikler(\/|$)/, /^\/admin\/bildirim(\/|$)/, /^\/admin\/ai-sohbet(\/|$)/, /^\/admin\/aboneler(\/|$)/],
        items: [['Yorumlar', '/admin/yorumlar'], ['Geri Bildirim', '/admin/geri-bildirim'], ['Aboneler', '/admin/aboneler'], ['Üyelikler', '/admin/members'], ['Yazarlar', '/admin/yazarlar'], ['Anketler', '/admin/anketler'], ['Etkinlikler', '/admin/etkinlikler'], ['Bildirimler', '/admin/bildirim'], ['AI Sohbet', '/admin/ai-sohbet']] },
      { id: 'seo', label: 'SEO & Dağıtım',
        match: [/^\/admin\/seo(\/|$)/, /^\/admin\/backlink(\/|$)/, /^\/admin\/indexleme(\/|$)/, /^\/admin\/tasima(\/|$)/, /^\/admin\/paylasim(\/|$)/, /^\/admin\/bulten(\/|$)/],
        items: [['SEO & İç Link', '/admin/seo'], ['İç Link Ağı', '/admin/backlink'], ['İndeksleme', '/admin/indexleme'], ['Taşıma & 301', '/admin/tasima'], ['Sosyal Paylaşım', '/admin/paylasim'], ['Bülten', '/admin/bulten']] },
      { id: 'gelir', label: 'Gelir',
        match: [/^\/admin\/reklamlar(\/|$)/, /^\/admin\/adsense(\/|$)/, /^\/admin\/raporlar(\/|$)/],
        items: [['Reklamlar', '/admin/reklamlar'], ['AdSense / GA', '/admin/adsense'], ['Raporlar', '/admin/raporlar']] },
      { id: 'otomasyon', label: 'Otomasyon & AI',
        match: [/^\/admin\/bot(\/|$)/, /^\/admin\/ai-kaynak(\/|$)/, /^\/admin\/servisler(\/|$)/, /^\/admin\/kumeler(\/|$)/, /^\/admin\/mahrecler(\/|$)/],
        items: [['Bot / Haber Akışı', '/admin/bot'], ['AI Kaynak', '/admin/ai-kaynak'], ['Servisler', '/admin/servisler'], ['Konu Kümeleri', '/admin/kumeler'], ['Mahreçler', '/admin/mahrecler']] },
      { id: 'sistem', label: 'Sistem',
        match: [/^\/admin\/ayarlar(\/|$)/, /^\/admin\/siteler(\/|$)/, /^\/admin\/bilesenler(\/|$)/, /^\/admin\/iletisim(\/|$)/],
        items: [['Ayarlar', '/admin/ayarlar'], ['İletişim & Form', '/admin/iletisim'], ['Siteler', '/admin/siteler'], ['Bileşenler', '/admin/bilesenler']] },
    ];
    /* Yön hesabı için dışarı açık: sekme sırası bilgisi başka modülde
       (içerik geçişi) gerekiyor. Kopyalamak yerine paylaşılıyor. */
    window.tbWsListe = WS;
    var path = location.pathname.replace(/\/+$/, '') || '/';
    var ws = null;
    for (var i = 0; i < WS.length; i++) {
      for (var j = 0; j < WS[i].match.length; j++) {
        if (WS[i].match[j].test(path)) { ws = WS[i]; break; }
      }
      if (ws) break;
    }
    // Sol menüde aktif alanı işaretle
    document.querySelectorAll('.ws-menu a[data-ws]').forEach(function (a) {
      a.classList.toggle('active', !!ws && a.getAttribute('data-ws') === ws.id);
    });
    // Sekme çubuğu (tek sayfalık alanlarda gizli)
    if (!ws || ws.items.length <= 1) return;
    var crumb = document.getElementById('adminCrumb');
    if (!crumb || document.getElementById('wsTabs')) return;
    // Birincil aksiyon (Faz C): alanın ana işi — her sayfada aynı yerde.
    // Gizleme yalnız tam sayfa eşleşmesinde (aksiyonun kendi formu/girişinde).
    var act = '';
    if (ws.action) {
      var hidden = (ws.action.hide || []).some(function (h) {
        return path === h;
      });
      if (!hidden) {
        act = '<a class="btn small ws-act" href="' + ws.action.href + '">' +
          (ws.action.ikon
            ? '<svg class="ti" aria-hidden="true"><use href="/static/vendor/tabler-icons.svg?v=b89545a6#ti-' + ws.action.ikon + '"></use></svg>'
            : '') + ws.action.label + '</a>';
      }
    }
    var nav = document.createElement('nav');
    nav.className = 'ws-tabs';
    nav.id = 'wsTabs';
    nav.setAttribute('aria-label', ws.label + ' sekmeleri');
    // Kabuk korumalı gezinme: yalnız #adminMain değişir, yan menü ve bu
    // çubuk yerinde kalır. Detay: tbAdminInit.
    nav.setAttribute('hx-boost', 'true');
    // Sınıf seçicisi bilinçli: 18 şablonda <main class="admin-main">
    // var, id eklemek 18 dosyaya dokunmak olurdu. Sınıf tek ve benzersiz.
    nav.setAttribute('hx-target', '.admin-main');
    nav.setAttribute('hx-select', '.admin-main');
    // OUTERHTML: INNERHTML iken yanıttaki .admin-main mevcut main'in
    // İÇİNE giriyor, her gezinmede iç içe ikinci bir <main> oluşuyordu.
    nav.setAttribute('hx-swap', 'outerHTML show:window:top');
    nav.setAttribute('hx-push-url', 'true');
    nav.innerHTML =
      '<span class="ws-links">' + (function () {
        // Tam eşleşme varsa YALNIZ o işaretlenir. Aksi hâlde hem
        // "/admin/icerikler" hem "/admin/icerikler/yeni" .on alıyor ve
        // hap yanlış sekmede kalıyordu.
        var tam = -1, k;
        for (k = 0; k < ws.items.length; k++) {
          if (path === ws.items[k][1]) { tam = k; break; }
        }
        // AĞIR SAYFALAR boost DIŞI kalır. Bu sayfaların betikleri
        // <head>'de ve boost yalnız #adminMain'i değiştirdiği için head
        // yenilenmez; boost'lanırsa editör hiç yüklenmez ve sayfa ölü gelir.
        var AGIR = {
          '/admin/icerikler/yeni': 1,
          '/admin/manset-editor': 1,
          '/admin/ses-ayarlari': 1,
          // Sayfa tasarımcısı: betiği <head>'de değil gövdenin sonunda ve
          // SortableJS'e bağlı. Boost'lanırsa tuval kurulmaz, sayfa boş gelir.
          '/admin/tasarim': 1
        };
        return ws.items.map(function (it, i) {
          var on = tam >= 0 ? i === tam : path.indexOf(it[1] + '/') === 0;
          var boost = AGIR[it[1]] ? ' hx-boost="false"' : '';
          return '<a href="' + it[1] + '"' + (on ? ' class="on"' : '') + boost + '>' + it[0] + '</a>';
        }).join('');
      })() + '</span>' +
      '<span class="ws-right">' + act + '</span>';
    crumb.parentNode.insertBefore(nav, crumb.nextSibling);
    // Dinamik eklenen öğeyi htmx'e tanıt: aksi hâlde hx-boost yok sayılır.
    if (window.htmx && window.htmx.process) window.htmx.process(nav);
  }

  /* ================= Kabuk korumalı gezinme (Faz 156) =================
   *
   * Ölçüm: her gezinmede 8 CSS + 12 JS (~262 KB) yeniden ayrıştırılıyor,
   * admin-ui.js (42 KB) baştan koşuyordu (breadcrumb, sekmeler, 4 tablo
   * aracı, hap). Sunucu 8–29 ms; yavaşlık yeniden YÜKLEMEDEN geliyordu.
   *
   * Çözüm: yan menü ve üst çubuk DOM'da kalır, yalnız #adminMain
   * htmx ile değişir. Böylece:
   *   - beyaz parlama yok (CSS zaten yüklü),
   *   - yan menü titremiyor, kaydırma konumu korunuyor,
   *   - sekme hapı eski yerinden yenisine akıyor (sessionStorage devri),
   *   - JS yeniden çalışmıyor; yalnız bu fonksiyon gerekenleri tazeliyor.
   *
   * Ağır sayfalar (içerik formu, manşet editörü, ses ayarları) boost dışı:
   * betikleri <head>'de ve boost head'i yenilemez.
   */
  /* ========== Yardım merkezi araması (Faz 165) ==========
   * Maddeler sunucudan `data-ara` ile gelir (küçük harfe çevrilmiş başlık +
   * gövde). Süzme istemcide: 40+ madde için sorgu atmak gereksiz.
   * Gizleme `hidden` özniteliğiyle — CSS `display:none` verir.
   */
  function tbYardim() {
    var kutu = document.getElementById('yardimAra');
    if (!kutu || kutu.__tbKuruldu) return;
    kutu.__tbKuruldu = true;

    var sayi = document.getElementById('yardimSayi');
    var bos = document.getElementById('yardimBos');
    var ogeler = [].slice.call(document.querySelectorAll('.yardim [data-ara]'));

    function norm(v) {
      return (v || '').toLocaleLowerCase('tr')
        .replace(/İ/g, 'i').replace(/I/g, 'ı');
    }

    function suz() {
      var q = norm(kutu.value.trim());
      var gorunen = 0;
      ogeler.forEach(function (el) {
        var esles = !q || (el.getAttribute('data-ara') || '').indexOf(q) >= 0;
        el.hidden = !esles;
        if (esles) gorunen++;
      });
      if (sayi) sayi.textContent = q ? (gorunen + ' sonuç') : '';
      if (bos) {
        var ipucuVar = [].slice.call(document.querySelectorAll('.yardim-izgara [data-ara]'))
          .some(function (el) { return !el.hidden; });
        bos.hidden = !q || ipucuVar;
      }
    }

    kutu.addEventListener('input', suz);
    kutu.addEventListener('keydown', function (e) {
      if (e.key === 'Escape') { kutu.value = ''; suz(); }
    });
  }

  /* ========== Genel "Yeni" menüsü (Faz 168) ==========
   * <details> kendi açılıp kapanmasını yönetir; eksik olan iki şey var:
   * dışarı tıklayınca ve Esc ile kapanmak. Ayrıca aynı anda açık kalmasın
   * diye başka bir details açılınca kapanır.
   */
  function tbYeniMenu() {
    var hepsi = document.querySelectorAll('.new-dd');
    if (!hepsi.length) return;
    hepsi.forEach(function (dd) {
      if (dd.__tbKuruldu) return;
      dd.__tbKuruldu = true;
      dd.addEventListener('toggle', function () {
        if (dd.open) {
          // Diğer açık menüleri kapat (bildirim çanı gibi).
          document.querySelectorAll('.new-dd[open]').forEach(function (d) {
            if (d !== dd) d.open = false;
          });
          document.querySelectorAll('details.notif-dd[open]').forEach(function (d) {
            d.open = false;
          });
        }
      });
    });

    if (document.__tbYeniMenuKuruldu) return;
    document.__tbYeniMenuKuruldu = true;

    document.addEventListener('click', function (e) {
      document.querySelectorAll('.new-dd[open]').forEach(function (dd) {
        if (!dd.contains(e.target)) dd.open = false;
      });
    });
    document.addEventListener('keydown', function (e) {
      if (e.key !== 'Escape') return;
      document.querySelectorAll('.new-dd[open]').forEach(function (dd) {
        dd.open = false;
        var ozet = dd.querySelector('summary');
        if (ozet) ozet.focus();
      });
    });
  }

  function tbAdminInit() {
    tbYardim();
    tbYeniMenu();
    // Geçmiş (geri/ileri) yalnız İÇERİĞİ geri yüklesin. htmx varsayılan
    // olarak <body>'yi önbelleğe alır ve geri tuşunda tüm gövdeyi değiştirir;
    // bu, yan menüyü de yeniden kurar (ölçüm: geri tuşunda işaret kayboldu).
    // hx-history-elt ile yalnız .admin-main saklanır.
    var main = document.querySelector('.admin-main');
    if (main && !main.hasAttribute('hx-history-elt')) {
      main.setAttribute('hx-history-elt', '');
      if (window.htmx && window.htmx.process) window.htmx.process(main);
    }
    tbCrumb();
    tbWsTabs();
    tbTablolar();
    if (window.tbHapTara) window.tbHapTara();
    // Yan menü hapı (aktif öğenin zemini) — Pacely'deki `.nav__pill`.
    if (window.tbSbHapKur) window.tbSbHapKur();
    // Kaydırma konumu korunduğu için üst çubuğun "yükselmiş" hâlini
    // yeniden uygula (yeni #adminMain ile sınıf kayboluyor).
    var top = document.querySelector('.admin-top');
    if (top) top.classList.toggle('is-stuck', window.scrollY > 6);
  }
  window.tbAdminInit = tbAdminInit;

  // Başlık: boost'ta <head> değişmediği için document.title elle güncellenir.
  function tbBaslik() {
    var h1 = document.querySelector('.admin-top h1');
    if (h1 && h1.textContent.trim()) {
      document.title = h1.textContent.trim() + ' — theblock Panel';
    }
  }

  // Geri/ileri tuşu: htmx önbellekten .admin-main'i geri koyar. Kurucular
  // yeniden çalışmalı, yoksa breadcrumb/sekme/tablo araçları boş kalır.
  document.body.addEventListener('htmx:historyRestore', function () {
    tbAdminInit();
    tbBaslik();
  });

  document.body.addEventListener('htmx:afterSettle', function (e) {
    var hedef = e.detail && e.detail.target;
    if (hedef && hedef.classList && hedef.classList.contains('admin-main')) {
      tbAdminInit();
      tbBaslik();
      return;
    }
    // Sayfa içi parça değişimi (hx-get + hx-target): tablo araçları ve hap
    // yeni DOM'da yeniden kurulmalı. Ölçüm: içerik sekmesi değişince tablo
    // araç çubuğu kayboluyordu (arama/sıralama/sayfalama gidiyordu).
    tbTablolar();
    if (window.tbHapTara) window.tbHapTara();
    if (window.tbSbHapKur) window.tbSbHapKur();
  });

  // Hata koruması: 4xx/5xx ya da giriş sayfasına yönlenme içeriğe
  // yazılırsa panel bozulur. Bu durumda tam yükleme yap.
  document.body.addEventListener('htmx:beforeSwap', function (e) {
    var x = e.detail && e.detail.xhr;
    if (!x) return;
    var yol = (x.responseURL || '').replace(location.origin, '');
    var giris = yol.indexOf('/admin/giris') === 0 || yol.indexOf('/giris') === 0;
    if (x.status >= 400 || x.status === 0 || giris) {
      e.detail.shouldSwap = false;
      var git = (e.detail.requestConfig && e.detail.requestConfig.path) || location.href;
      location.href = git;
    }
  });

  /* ========== İçerik geçişi: yatay süzülme ==========================
   *
   * Kullanıcı isteği: sekme hapı yana kayarken (İçerikler → Kategoriler…)
   * ALTTAKİ İÇERİK de aynı yönde "sürüklenerek" gelsin. Önce içerik
   * yalnız 3 px dikey kayıyordu (pn-gir); hap yatay akarken içerik
   * yerinde belirdiği için hareket kopuk duruyordu.
   *
   * Yön, ÇALIŞMA ALANI SEKME SIRASINDAN gelir (window.tbWsListe):
   *   • hedef sekme sağdaysa içerik SAĞDAN gelir  (+1)
   *   • hedef sekme soldaysa içerik SOLDAN gelir  (−1)
   *   • farklı çalışma alanına geçişte yan menü sırası kullanılır
   * Yön bilinmiyorsa (derin bağlantı, geri/ileri) +1 varsayılanı.
   *
   * Süre/eğri hap ile AYNI: 320 ms, --pn-ease = cubic-bezier(.22,1,.36,1).
   * Hareketi azalt açıksa hiç animasyon kurulmaz.
   *
   * Neden yalnız içerik çocukları: üst çubuk, kırıntı yolu ve sekme
   * çubuğu yerinde kalmalı — hap onların içinde akıyor. Yalnız
   * .admin-main'in içerik çocukları süzülür.
   */
  (function icerikSuzulme() {
    var AZALT = null;
    try { AZALT = window.matchMedia('(prefers-reduced-motion: reduce)'); } catch (e) {}
    var SURE = 320;
    var EGRI = 'cubic-bezier(.22,1,.36,1)'; // --pn-ease (hap ile aynı)
    var MESAFE = 28;
    var yon = 1;
    // Durum sekmesi gezinmesinde yön (0 = sekme gezinmesi değil).
    var sekmeYonu = 0;

    /* Çubuktaki sekme sırası. hedefUrl verilmezse SUNUCUNUN işaretlediği
     * (aria-current / .active / .on) sekmenin sırası döner. */
    function sekmeIndeksi(ray, hedefUrl) {
      var kardes = [].slice.call(ray.children).filter(function (c) {
        return c.matches('a, button') && !c.classList.contains('tb-hap');
      });
      var i, c, h;
      if (hedefUrl) {
        for (i = 0; i < kardes.length; i++) {
          h = kardes[i].getAttribute('href') || '';
          if (h === hedefUrl) return i;
        }
        // Sorgu dizesi farklı yazılmış olabilir: yol + arama karşılaştır.
        var aranan = hedefUrl.split('#')[0];
        for (i = 0; i < kardes.length; i++) {
          h = (kardes[i].getAttribute('href') || '').split('#')[0];
          if (h && h === aranan) return i;
        }
        return -1;
      }
      for (i = 0; i < kardes.length; i++) {
        c = kardes[i];
        if (c.getAttribute('aria-current') === 'page' ||
            c.classList.contains('active') || c.classList.contains('on')) return i;
      }
      return -1;
    }

    function konum(yol) {
      var L = window.tbWsListe || [];
      for (var i = 0; i < L.length; i++) {
        var it = L[i].items || [];
        for (var j = 0; j < it.length; j++) {
          var p = it[j][1];
          if (yol === p) return [i, j];
          // Alt sayfalar: /admin/icerikler/123 → İçerikler sekmesi
          if (p !== '/admin' && yol.indexOf(p + '/') === 0) return [i, j];
        }
      }
      return null;
    }

    document.body.addEventListener('htmx:beforeSwap', function (e) {
      yon = 1;
      sekmeYonu = 0;

      /* Durum sekmeleri (İçerikler: Yayında/Taslak/…) TÜM paneli takas
       * ediyor; hepsi AYNI yolu (`/admin/icerikler`) yalnız farklı `?durum=`
       * ile çağırıyor.
       *
       * `detail.elt` burada işe yaramıyor: htmx onu tıklanan <a> değil,
       * gösterge DIV'i olarak veriyor (ölçüldü: "DIV."). Bu yüzden yön,
       * İSTEK ADRESİ ile çubuktaki sekme sırası karşılaştırılarak bulunur. */
      var istek = (e.detail && e.detail.requestConfig && e.detail.requestConfig.path) || '';
      if (istek) {
        var ray = document.querySelector('#icerik-panel .tabs');
        if (ray && istek.split('?')[0] === location.pathname) {
          var yeniIdx = sekmeIndeksi(ray, istek);
          var eskiIdx = sekmeIndeksi(ray, null);
          if (yeniIdx >= 0 && eskiIdx >= 0 && yeniIdx !== eskiIdx) {
            sekmeYonu = yeniIdx < eskiIdx ? -1 : 1;
          }
          if (sekmeYonu !== 0) return;
        }
      }

      var hedef = istek;
      hedef = hedef.split('?')[0];
      var su = location.pathname;
      var a = konum(su);
      var b = konum(hedef);
      if (!a || !b) return;
      if (b[0] < a[0]) yon = -1;
      else if (b[0] === a[0] && b[1] < a[1]) yon = -1;
    });

    /* Bir kaptaki içerik çocuklarını yataktan süzülmeye sokar.
     * `hariç`: yerinde kalacak seçici (sekme çubuğu). */
    function suzul(kok, yonDeger, haric) {
      if (!kok || !kok.animate) return;
      if (AZALT && AZALT.matches) return;

      var secili = [];
      Array.prototype.forEach.call(kok.children, function (el) {
        if (!el || !el.matches) return;
        if (el.matches('.admin-top, .admin-crumb, .ws-tabs, script, style, link')) return;
        if (haric && el.matches(haric)) return;
        if (!el.offsetWidth && !el.offsetHeight) return;
        secili.push(el);
      });

      secili.forEach(function (el, i) {
        var a = el.animate(
          [
            // Opaklık 0'dan başlamıyor: tam saydam başlangıç, yavaş ağda
            // bir kare "boş panel" gösteriyor. 0.45, pn-yenile ile aynı
            // his, ama kayma yönü yatay.
            { transform: 'translateX(' + (yonDeger * MESAFE) + 'px)', opacity: 0.45 },
            { transform: 'translateX(0px)', opacity: 1 }
          ],
          {
            duration: SURE,
            easing: EGRI,
            delay: Math.min(i * 12, 48),
            fill: 'both'
          }
        );
        a.onfinish = function () {
          el.style.transform = '';
          el.style.opacity = '';
          try { a.cancel(); } catch (err) {}
        };
      });
    }

    document.body.addEventListener('htmx:afterSwap', function (e) {
      // Tetikleyen bağlantı, hedefi .admin-main olan bir gezinme kabuğunda mı?
      // Panel içi kısmi takaslar (tablo yenile, filtre) bu süzülmeyi
      // TETİKLEMEMELİ; yoksa her AJAX güncellemesinde sayfa kayardı.
      // htmx, boost'lu gezinmede `detail.elt` alanına TIKLANAN BAĞLANTIYI
      // değil, TAKAS HEDEFİNİ (.admin-main) koyuyor (ölçüldü). İki durumu
      // da kabul et: hedef .admin-main ise bu bir kabuk gezinmesidir;
      // panel içi kısmi takaslarda elt başka bir öğe olur ve süzülme olmaz.
      var tetik = e.detail && e.detail.elt;
      var kabukGezinmesi = false;
      if (tetik && tetik.classList) {
        kabukGezinmesi = tetik.classList.contains('admin-main') ||
          !!(tetik.closest && tetik.closest('[hx-target=".admin-main"]'));
      }

      /* Durum sekmesi gezinmesi (İçerikler, reklamlar, SEO…): panelin
       * kendisi takas edilir. Sekme çubuğu yerinde kalsın, altındaki
       * içerik hapın gittiği yönden süzülsün — üst menüdeki davranışın
       * aynısı. */
      if (!kabukGezinmesi && sekmeYonu !== 0) {
        var panel = document.querySelector('#icerik-panel') ||
          (e.detail && e.detail.target && e.detail.target.isConnected ? e.detail.target : null);
        if (panel) {
          panel.classList.add('tb-suzuluyor');
          suzul(panel, sekmeYonu, '.tabs');
          setTimeout(function () { panel.classList.remove('tb-suzuluyor'); }, SURE + 120);
        }
        sekmeYonu = 0;
        return;
      }
      if (!kabukGezinmesi) return;

      // DİKKAT: hx-swap="outerHTML" ile htmx `detail.target` alanında ESKİ
      // (artık DOM'dan çıkmış) öğeyi bırakıyor. Onu animasyonlarsak kimse
      // görmez — ölçüldü: animasyon nesnesi oluşuyor ama ekranda hareket yok.
      // Bu yüzden güncel öğe DOM'dan sorgulanıyor.
      var kok = document.querySelector('.admin-main');
      if (!kok || !kok.isConnected) return;
      if (AZALT && AZALT.matches) return;
      if (!kok.animate) return;

      // pn-yenile (kap belirmesi) ile çakışmasın: kendi animasyonumuz var.
      kok.classList.add('tb-suzuluyor');
      suzul(kok, yon, null);
      setTimeout(function () { kok.classList.remove('tb-suzuluyor'); }, SURE + 120);
    });
  })();

  /* İlk kurulum: yukarıdaki üç kurucu artık isimli fonksiyon. */
  tbAdminInit();
  tbBaslik();
  /* Yan menü giriş animasyonu (Pacely mount efekti): yalnız TAM sayfa
   * yüklemesinde; boost gezinmesinde menü DOM'da kaldığı için oynamaz. */
  if (window.tbSbGiris) window.tbSbGiris();

  /* ================= Tablo yoğunluğu (ADMIN-IA Faz D) ================= */
  (function density() {
    var btn = document.getElementById('tblDenseBtn');
    var apply = function () {
      document.body.classList.toggle('tbl-dense', LS.get('tb_tbl_dense', false));
      if (btn) btn.classList.toggle('on', document.body.classList.contains('tbl-dense'));
    };
    apply();
    if (btn) btn.addEventListener('click', function () {
      LS.set('tb_tbl_dense', !document.body.classList.contains('tbl-dense'));
      apply();
    });
  })();
  /* ========== Yayına uygunluk puanı: sonuç + gerekçe (spec 11 U3) ==========
   *
   * Puan bir AI yargısı değil, üstveri eksiksizliği sezgiselidir
   * (internal/scoring). Editöre 72 sayısını göstermek işe yaramıyordu —
   * 34.156 taslağın %88'i 60-79 bandında, yani sayı tek başına ayırt
   * etmiyor. Bu yüzden sunucu artık kalem kalem dökümü (`sinyaller`)
   * gönderiyor ve burada "neyi düzeltirsen artar" listesi basılıyor.
   *
   * Bu blok app.js'ten buraya taşındı: yalnız panelde kullanılıyor ve
   * app.js her ziyaretçiye iniyor (bkz. Faz 135, CSS'te aynı ayrım). */
  /* Sınıf adları düz yazılır, birleştirilmez: 'skor-' + durum biçiminde
   * üretilen ad hiçbir dosyada geçmediği için ne grep ne de CSS katman
   * denetimi onu görebiliyor (bkz. internal/webui/admincss_test.go). */
  var DURUM_SINIF = {
    iyi: 'skor-iyi', orta: 'skor-orta', eksik: 'skor-eksik', bilgi: 'skor-bilgi'
  };

  function tbSkorSonuc(d) {
    var input = document.getElementById('aiSkor');
    var note = document.getElementById('aiSkorNote');
    var list = document.getElementById('aiSkorGerekce');
    if (list) list.innerHTML = '';
    if (!d || !d.ok) {
      if (note) note.textContent = 'Hata: ' + ((d && d.hata) || 'puan hesaplanamadı');
      return;
    }
    if (input) input.value = d.skor;
    if (note) {
      note.textContent = 'Puan: ' + d.skor + ' — kapı: ' +
        (d.kapi_gecer ? 'geçti' : 'ENGELLENDİ (' + (d.kapi_nedeni || '') + ')');
    }
    if (!list || !d.sinyaller || !d.sinyaller.length) return;
    d.sinyaller.forEach(function (s) {
      var li = document.createElement('li');
      li.className = 'skor-kalem ' + (DURUM_SINIF[s.durum] || DURUM_SINIF.bilgi);
      var ad = document.createElement('span');
      ad.className = 'skor-ad';
      ad.textContent = s.ad;
      var puan = document.createElement('span');
      puan.className = 'skor-puan';
      puan.textContent = s.puan + ' / ' + s.enb;
      li.appendChild(ad);
      li.appendChild(puan);
      if (s.ipucu) {
        var ip = document.createElement('span');
        ip.className = 'skor-ipucu';
        ip.textContent = s.ipucu;
        li.appendChild(ip);
      }
      list.appendChild(li);
    });
  }
  /* Satır içi yedek betik (htmx yokken) de aynı çizimi kullansın. */
  window.tbSkorSonuc = tbSkorSonuc;
  document.body.addEventListener('aiScore', function (e) {
    tbSkorSonuc((e && e.detail) || {});
  });

  /* ========== Yan menü rayı (Faz 151) ==========
   *
   * Pacely'nin katlanabilir rayı. Tercih pano ile AYNI anahtarda tutulur
   * (`pn:rail`): pano ile diğer panel sayfaları arasında geçerken menünün
   * genişliği zıplamasın. Daraltma yalnız görsel — menü öğeleri DOM'da
   * kalır, erişilebilirlik ve klavye gezinmesi bozulmaz.
   *
   * İlk açılışta genişlikten türetilir (1280 px altı dar), sonra kullanıcı
   * kararı kalıcı olur. */
  /* ---- Ray geçişi: Pacely birebir (kullanıcı isteği) ----------------
   *
   * Kaynak: demo.codeandchill.store/pacely — `qt()` ve `Jt()` fonksiyonları
   * (paket: assets/index-BlFqaHp6.js). Oradaki GSAP çağrıları birebir
   * okunup burada Web Animations API'ye çevrildi; GSAP bağımlılığı yok.
   *
   * PACELY'DE OLAN (ve burada aynen uygulanan):
   *
   *  DARALTMA
   *    1. yalnız etiketler soldurulur:  opacity 1→0, x 0→-8,
   *       140 ms, power2.in
   *    2. menü sağdan perdeyle kapanır: clip-path
   *       inset(0 0 0 0) → inset(0 (genişlik-88)px 0 0),
   *       300 ms, power3.in
   *    3. perde bitince sınıf değişir (width ANLIK 88 px), clip-path ve
   *       etiket stilleri temizlenir, ana kolon FLIP ile yerine oturur.
   *
   *  GENİŞLETME
   *    1. sınıf hemen değişir (width anlık 252 px)
   *    2. perde soldan açılır: clip-path
   *       inset(0 (genişlik-88)px 0 0) → inset(0 0 0 0),
   *       500 ms, expo.out
   *    3. etiketler soldan sırayla gelir: opacity 0→1, x -10→0,
   *       550 ms, power3.out, gecikme 120 ms, kademe 10 ms
   *
   *  ANA KOLON (her iki yönde) — "Jt": menü genişliği değişince içerik
   *  sola/zıplar; bu zıplama ters yönde telafi edilip 600 ms expo.out ile
   *  0'a getirilir. İçerik yer değiştirmez, "kayar".
   *
   * Eğri karşılıkları (GSAP → cubic-bezier):
   *   power2.in  → cubic-bezier(.55,.085,.68,.53)
   *   power3.in  → cubic-bezier(.55,.055,.675,.19)
   *   power3.out → cubic-bezier(.215,.61,.355,1)
   *   expo.out   → cubic-bezier(.19,1,.22,1)
   *
   * "Hareketi azalt" açıksa (Ee() karşılığı) animasyon hiç kurulmaz:
   * sınıf anında değişir, tercih yine kaydedilir.
   */
  var RAY_GENISLIK = 88; // Pacely --rail-w ile aynı
  var RAY_EGRI = {
    power2in: 'cubic-bezier(.55,.085,.68,.53)',
    power3in: 'cubic-bezier(.55,.055,.675,.19)',
    power3out: 'cubic-bezier(.215,.61,.355,1)',
    expoOut: 'cubic-bezier(.19,1,.22,1)'
  };
  var RAY_HAREKET_AZALT = null;
  try { RAY_HAREKET_AZALT = window.matchMedia('(prefers-reduced-motion: reduce)'); } catch (e) {}

  /* Perde animasyonunu oynatır.
   *
   * `fill:'both'` şart: GSAP gibi bitiş değerini TUTMALI, aksi hâlde perde
   * bitmeden stil geri sıçrar (GSAP te "clearProps" o yüzden var).
   * Bitince animasyon `cancel()` edilir: fill kalırsa eleman DOM'da kalıcı
   * `clip-path` ile yaşar ve sonraki geçişte eski değer yarışır. */
  function rayPerde(el, den, ye, sure, egri, bitince) {
    if (!el.animate) { bitince && bitince(); return null; }
    var a = el.animate(
      [{ clipPath: den }, { clipPath: ye }],
      { duration: sure, easing: egri, fill: 'both' }
    );
    a.onfinish = function () {
      el.style.clipPath = '';
      try { a.cancel(); } catch (e) {}
      bitince && bitince();
    };
    return a;
  }

  /* Aynı mantık: biten animasyonu iptal et, satır içi stili temizle. */
  function rayBitir(a, el) {
    a.onfinish = function () {
      el.style.opacity = '';
      el.style.transform = '';
      try { a.cancel(); } catch (e) {}
    };
  }

  function tbRailGecis(dar, btn) {
    var kok = document.documentElement;
    if (kok.classList.contains('admin-rail') === dar) return;
    var yan = document.querySelector('.admin-side');
    var anamenu = document.querySelector('.admin-main');
    var oncekiSol = anamenu ? anamenu.getBoundingClientRect().left : 0;

    function sinifiUygula() {
      kok.classList.toggle('admin-rail', dar);
      LS.set('pn:rail', dar);
      if (btn) {
        btn.setAttribute('aria-expanded', dar ? 'false' : 'true');
        btn.setAttribute('aria-label', dar ? 'Menüyü genişlet' : 'Menüyü daralt');
        btn.setAttribute('title', dar ? 'Menüyü genişlet' : 'Menüyü daralt (kalıcı)');
      }
      // Pano sayfası kendi rayını kullanıyor; orada da aynı sınıf geçerli.
      var pnApp = document.getElementById('pnApp');
      if (pnApp) pnApp.classList.toggle('is-rail', dar);
    }

    // Ana kolon FLIP telafisi (Pacely `Jt`).
    function anamenuKaydir() {
      if (!anamenu || !anamenu.animate) return;
      var simdi = anamenu.getBoundingClientRect().left;
      var fark = oncekiSol - simdi;
      if (Math.abs(fark) < 0.5) return;
      var a = anamenu.animate(
        [{ transform: 'translateX(' + fark + 'px)' }, { transform: 'translateX(0px)' }],
        { duration: 600, easing: RAY_EGRI.expoOut, fill: 'both' }
      );
      a.onfinish = function () {
        anamenu.style.transform = '';
        try { a.cancel(); } catch (e) {}
      };
    }

    // Hareket azalt / destek yok: Pacely de aynen böyle yapıyor.
    if (!yan || !anamenu || (RAY_HAREKET_AZALT && RAY_HAREKET_AZALT.matches)) {
      sinifiUygula();
      return;
    }

    var etiketler = yan.querySelectorAll('.menu a .mn, .brand');

    if (dar) {
      // 1) etiketler soldurulur — ikonlar yerinde kalır.
      //
      // NOT: bu animasyon perde bitene kadar TUTULUR (pacely de öyle:
      // clearProps perdenin onComplete'inde). 140 ms'de temizlenseydi
      // etiketler perde kapanmadan geri gelip bir an görünürdü.
      var etiketAnim = [];
      if (etiketler.length && etiketler[0].animate) {
        Array.prototype.forEach.call(etiketler, function (el) {
          var a = el.animate(
            [{ opacity: 1, transform: 'translateX(0px)' },
             { opacity: 0, transform: 'translateX(-8px)' }],
            { duration: 140, easing: RAY_EGRI.power2in, fill: 'both' }
          );
          etiketAnim.push([a, el]);
        });
      }
      // 2) sağdan perde + 3) sınıf değişimi
      // Perde miktarı DARALTMA öncesi genişlikten: 252 - 88 = 164 px.
      var perdeDar = yan.offsetWidth - RAY_GENISLIK;
      rayPerde(yan, 'inset(0px 0px 0px 0px)', 'inset(0px ' + perdeDar + 'px 0px 0px)',
        300, RAY_EGRI.power3in, function () {
          sinifiUygula();
          etiketAnim.forEach(function (cift) {
            cift[1].style.opacity = '';
            cift[1].style.transform = '';
            try { cift[0].cancel(); } catch (e) {}
          });
          anamenuKaydir();
        });
    } else {
      // 1) sınıf hemen
      sinifiUygula();
      // 2) perde soldan açılır.
      // Miktar sınıf değiştikten SONRA ölçülür: Pacely de böyle yapıyor
      // (`o(); let e = n.offsetWidth - 88`). Öncesinde ölçülseydi menü hâlâ
      // 88 px olduğu için perde 0 px çıkar ve geçiş görünmezdi.
      var perdeGen = yan.offsetWidth - RAY_GENISLIK;
      rayPerde(yan, 'inset(0px ' + perdeGen + 'px 0px 0px)', 'inset(0px 0px 0px 0px)',
        500, RAY_EGRI.expoOut, null);
      // 3) etiketler sırayla gelir (opacity 0→1, x -10→0)
      if (etiketler.length && etiketler[0].animate) {
        Array.prototype.forEach.call(etiketler, function (el, i) {
          var a = el.animate(
            [{ opacity: 0, transform: 'translateX(-10px)' },
             { opacity: 1, transform: 'translateX(0px)' }],
            {
              duration: 550, easing: RAY_EGRI.power3out,
              delay: 120 + i * 10, fill: 'both'
            }
          );
          rayBitir(a, el);
        });
      }
      anamenuKaydir();
    }
  }
  window.tbRailGecis = tbRailGecis;

  (function yanMenuRay() {
    var btn = document.getElementById('adminRail');
    if (!btn) return;
    var kok = document.documentElement; // açılışta sınıfı koyan satır içi betik
    btn.setAttribute('aria-expanded', kok.classList.contains('admin-rail') ? 'false' : 'true');
    btn.addEventListener('click', function () {
      tbRailGecis(!kok.classList.contains('admin-rail'), btn);
    });
  })();

  /* ========== İçerik formu: katlanabilir gruplar (spec 11 U2) ==========
   *
   * İki davranış ekler, ikisi de olmadan katlama editörü yavaşlatırdı:
   *
   * 1) Durum hatırlanır. Hep "Künye"yi açan bir editör her kayıtta yeniden
   *    açmak zorunda kalmamalı. localStorage okunamazsa (gizli sekme,
   *    çerez engeli) sessizce varsayılana düşer.
   * 2) Gizli alandaki doğrulama hatası görünür kılınır. Kapalı bir
   *    <details> içindeki geçersiz alan için tarayıcı hata balonunu
   *    gösteremez ve kaydetme sessizce durur — editör neden kaydedilmediğini
   *    anlayamaz. Böyle bir durumda grup açılır, alana odaklanılır ve grup
   *    kısa süre vurgulanır. */
  (function formGruplari() {
    var gruplar = document.querySelectorAll('details.form-grup');
    if (!gruplar.length) return;

    gruplar.forEach(function (d) {
      // Anahtar ÖNCE sabit slug'dan (data-grup), yoksa görünen metinden.
      //
      // NEDEN: anahtar görünen başlıktan türetildiğinde ipucu metni
      // ("mahreç · yazar · editör · konu kümesi") de anahtara giriyordu.
      // Sonuç: başlık/ipucu düzenlendiğinde kullanıcının açık bıraktığı
      // grup sessizce kapanıyor, ayrıca "·" karakteri sunucudaki anahtar
      // doğrulamasına takılıyordu (Faz 173 ölçümü).
      var ozet = d.querySelector('summary') || {};
      var slug = (ozet.getAttribute && ozet.getAttribute('data-grup')) || '';
      var anahtar;
      if (slug) {
        anahtar = 'tb_form_grup:' + slug;
      } else {
        var ad = (ozet.textContent || '').trim().split('\n')[0].slice(0, 40);
        anahtar = 'tb_form_grup:' + ad;
      }
      var kayitli = LS.get(anahtar, null);
      if (kayitli === true) d.open = true;
      // Son bilinen durum: ilk uygulama sırasında tetiklenen toggle, az önce
      // okuduğumuz değeri sunucuya geri yazmasın (gereksiz istek).
      var son = d.open;
      d.addEventListener('toggle', function () {
        if (d.open === son) return;
        son = d.open;
        LS.set(anahtar, d.open);
      });
    });

    /* invalid olayı balonlanmaz; yakalama evresinde dinlemek gerekiyor. */
    document.addEventListener('invalid', function (e) {
      var alan = e.target;
      if (!alan || !alan.closest) return;
      var grup = alan.closest('details.form-grup');
      if (!grup || grup.open) return;
      grup.open = true;
      grup.classList.add('grup-vurgu');
      setTimeout(function () { grup.classList.remove('grup-vurgu'); }, 1400);
      try { alan.focus({ preventScroll: false }); } catch (err) { alan.focus(); }
    }, true);
  })();

  /* ========== Hareket ve geçiş katmanı (Faz 154) ==========
   *
   * Panelin HER sayfasında geçerli. Pano ile diğer sayfalar arasında
   * davranış farkı kalmaması için ortak dosyaya konuldu.
   *
   * Dört iş:
   *   1. Üst çubuk kaydırınca "yükselir" (.is-stuck) — okunabilirlik.
   *   2. HTMX isteğinde ince ilerleme çubuğu — "dondu mu?" sorusunu kapatır.
   *   3. window.tbToast: panelin her yerinden çağrılabilen tek balon sistemi.
   *   4. Görünüm geçişleri: HTMX'e karışmadan, yalnız tam sayfa
   *      gezinmelerde yumuşak çapraz geçiş.
   *
   * Hepsi savunmacı: biri hata verse bile diğerleri ve panelin kalanı
   * çalışmaya devam eder (her blok kendi try/catch'inde).
   */

  /* ---------- 1) Kaydırınca üst çubuk yükselir ---------- */
  (function ustCubuk() {
    var top = document.querySelector('.admin-top');
    if (!top) return;
    var bekliyor = false;
    function guncelle() {
      bekliyor = false;
      top.classList.toggle('is-stuck', window.scrollY > 6);
    }
    window.addEventListener('scroll', function () {
      // rAF ile: scroll olayı saniyede yüzlerce kez geliyor, her birinde
      // classList yazmak düzeni titretiyor.
      if (!bekliyor) { bekliyor = true; requestAnimationFrame(guncelle); }
    }, { passive: true });
    guncelle();
  })();

  /* ---------- 2) HTMX ilerleme çubuğu ---------- */
  (function ilerleme() {
    if (!window.htmx) return;
    var cubuk = document.querySelector('.pn-progress');
    if (!cubuk) {
      cubuk = document.createElement('div');
      cubuk.className = 'pn-progress';
      cubuk.setAttribute('aria-hidden', 'true');
      document.body.appendChild(cubuk);
    }
    var sayac = 0, zaman = null;
    function basla() {
      sayac++;
      if (sayac > 1) return;
      cubuk.classList.add('on');
      cubuk.style.width = '8%';
      // %85'e kadar yavaşça yaklaş: gerçek bitişi bilmiyoruz ama
      // "çalışıyor" hissi sürmeli. Sona varmış gibi göstermiyoruz.
      var hedef = 85;
      zaman = setInterval(function () {
        var w = parseFloat(cubuk.style.width) || 0;
        if (w < hedef) cubuk.style.width = (w + (hedef - w) * .12).toFixed(1) + '%';
      }, 220);
    }
    function bitir() {
      sayac = Math.max(0, sayac - 1);
      if (sayac > 0) return;
      if (zaman) { clearInterval(zaman); zaman = null; }
      cubuk.style.width = '100%';
      setTimeout(function () {
        cubuk.classList.remove('on');
        setTimeout(function () { cubuk.style.width = '0'; }, 260);
      }, 160);
    }
    document.body.addEventListener('htmx:beforeRequest', basla);
    document.body.addEventListener('htmx:afterRequest', bitir);
    document.body.addEventListener('htmx:responseError', bitir);
    document.body.addEventListener('htmx:sendError', bitir);
    document.body.addEventListener('htmx:timeout', bitir);
  })();

  /* ---------- 3) Ortak balon (toast) ---------- */
  (function balonlar() {
    var kutu = document.getElementById('tbToasts');
    if (!kutu) {
      kutu = document.createElement('div');
      kutu.id = 'tbToasts';
      kutu.className = 'pn-toasts';
      kutu.setAttribute('aria-live', 'polite');
      document.body.appendChild(kutu);
    }
    // window.tbToast("Kaydedildi")            → nötr
    // window.tbToast("Kaydedildi", "ok")      → yeşil
    // window.tbToast("Kaydedilemedi", "hata") → kırmızı
    window.tbToast = function (mesaj, tur) {
      if (!mesaj) return;
      var t = document.createElement('div');
      t.className = 'pn-toast' + (tur === 'hata' ? ' pn-toast--hata' : tur === 'ok' ? ' pn-toast--ok' : '');
      t.textContent = mesaj;
      kutu.appendChild(t);
      setTimeout(function () {
        t.style.transition = 'opacity .22s ease, transform .22s ease';
        t.style.opacity = '0';
        t.style.transform = 'translateY(6px)';
        setTimeout(function () { t.remove(); }, 240);
      }, 2800);
    };
  })();

  /* ---------- 4) Sayfa geçişi: kaldırıldı ----------
   *
   * Burada manuel `startViewTransition` vardı. Tam sayfa görünüm geçişi
   * sekme hapının animasyonuyla çakışıyordu: tarayıcı geçiş boyunca
   * DONDURULMUŞ görüntüyü gösterdiği için hap animasyonu görünmüyor,
   * geçiş bitince hap çoktan varmış oluyordu. Ölçüm: aynı geçişte hap
   * ya hiç kımıldamıyor ya da aniden zıplıyordu.
   *
   * Tek geçiş sistemi: hap (admin-ui.js kayganHap) + kartların kademeli
   * girişi (admin-pano.css pn-gir). Referans da SPA olduğu için tam
   * sayfa geçişi kullanmıyor.

  /* ---------- 5) Form gönderiminde düğme durumu ----------
   *
   * Çift gönderimi engeller ve "bir şey oldu" geri bildirimi verir. HTMX
   * formlarında HTMX zaten kendi göstergesini çalıştırıyor; orada
   * dokunmuyoruz. */
  (function formlar() {
    document.addEventListener('submit', function (e) {
      var f = e.target;
      if (!f || f.tagName !== 'FORM') return;
      if (f.hasAttribute('hx-post') || f.hasAttribute('hx-get')) return;
      if (f.target && f.target !== '_self') return;
      var btn = f.querySelector('button[type="submit"]:not([disabled])');
      if (!btn) return;
      // Emniyet supabı: sunucu yanıt vermezse düğme kilitli kalmasın.
      setTimeout(function () {
        btn.disabled = false;
        btn.classList.remove('is-bekliyor');
      }, 12000);
      btn.disabled = true;
      btn.classList.add('is-bekliyor');
    }, true);
  })();

  /* ========== Kayan "mürekkep hapı" (Faz 155) ==========
   *
   * Murmura'nın sekme geçişinin aynısı. Hap, İKİ KENARI BAĞIMSIZ hareket eden
   * kırpılmış bir katmandır; içinde etiketlerin zıt renkli kopyaları durur.
   * Kırpma penceresi etiketin üstüne geldiği anda renk tam kenarda değişir.
   *
   * Neden kopya: tek bir arka plan kutusunu büyütmek metni de ölçekler ve yazı
   * titrer. İki kenarı ayrı sürmek hem "esneme" hissi verir hem yazıyı hiç
   * ölçeklemez.
   *
   * Neden GSAP yok: referans power3 eğrilerini kullanıyor; küp eğri 3 satır.
   * 60 KB'lık bir animasyon paketi yalnız bunun için inmesin.
   *
   * Kenar zamanlaması (referanstan):
   *   sağa giderken  → sol (önde) 550ms inOut, sağ (arkada) 360ms out
   *   sola giderken  → sol (arkada) 360ms out, sağ (önde) 550ms inOut
   * Önde giden kenar yavaş başlar, arkada kalan hızlı kapanır: hap önce
   * uzar, sonra toplanır.
   */
  (function kayganHap() {
    var azHareket = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    var ANAHTAR = 'tb:hap-gecis';
    var sure = { onde: 550, arkada: 360 };

    // power3 eğrileri (GSAP'in karşılığı).
    function out3(t) { return 1 - Math.pow(1 - t, 3); }
    function inOut3(t) { return t < .5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2; }

    function yurusun(bas, son, ms, egri, her) {
      if (azHareket || ms <= 0) { her(son); return; }
      var t0 = performance.now();
      (function adim(t) {
        var p = Math.min(1, (t - t0) / ms);
        her(bas + (son - bas) * egri(p));
        if (p < 1) requestAnimationFrame(adim);
      })(t0);
    }

    /* Kopya: ölçüyü AÇIKÇA kopyalar. CSS'te ortak sınıf paylaşmak yerine
     * hesaplanmış değerleri yazıyoruz — sekme çubuğu 4 farklı yerde farklı
     * puntoyla kullanılıyor (ayarlar .tabs, içerik .tabs, ws-tabs…). Ortak
     * sınıf birinde kayarsa hepsinde kayar. */
    function kopyala(a) {
      var s = document.createElement('span');
      s.className = 'tb-hap-kopya';
      s.innerHTML = a.innerHTML;
      var c = getComputedStyle(a);
      s.style.display = 'inline-flex';
      s.style.alignItems = 'center';
      s.style.whiteSpace = 'nowrap';
      s.style.padding = c.padding;
      s.style.margin = c.margin;
      s.style.border = c.border;
      s.style.borderRadius = c.borderRadius;
      s.style.fontFamily = c.fontFamily;
      s.style.fontSize = c.fontSize;
      s.style.fontWeight = c.fontWeight;
      s.style.letterSpacing = c.letterSpacing;
      s.style.lineHeight = c.lineHeight;
      s.style.gap = c.gap;
      return s;
    }

    function aktifOge(ray, sec) {
      var adaylar = [];
      for (var i = 0; i < ray.children.length; i++) {
        var c = ray.children[i];
        if (c.classList.contains('tb-hap')) continue;
        if (c.matches(sec)) adaylar.push(c);
      }
      /* Sıra: ÖNCE sunucunun işaretlediği sekme (aria-current / .active / .on).
       *
       * Önceden "tam yol eşleşmesi her zaman kazanır" kuralı ilk sıradaydı ve
       * İçerikler'in durum sekmelerini bozuyordu: hepsi
       * `/admin/icerikler?durum=…` adresine gidiyor, sorgu atıldığında hepsi
       * `/admin/icerikler` oluyor ve o da location.pathname'e eşit. Sonuç:
       * hap hangi sekme seçili olursa olsun HEP İLK sekmeye ("Yayında")
       * oturuyordu (ölçüldü: clip 4 sekme boyunca hiç değişmedi).
       *
       * Tam yol eşleşmesi yalnız AYIRT EDİCİ olarak gerekli:
       * /admin/icerikler/yeni sayfasında hem "İçerikler" (önek) hem
       * "Yeni İçerik" (tam) işaretli oluyor; o durumda tam eşleşme kazanır. */
      var yol = location.pathname.replace(/\/+$/, '') || '/';
      var isaretli = [];
      var j, a, href;
      for (j = 0; j < adaylar.length; j++) {
        a = adaylar[j];
        if (a.getAttribute('aria-current') === 'page' ||
            a.classList.contains('on') ||
            a.classList.contains('active')) {
          isaretli.push(a);
        }
      }
      if (isaretli.length === 1) return isaretli[0];
      if (isaretli.length > 1) {
        for (j = 0; j < isaretli.length; j++) {
          a = isaretli[j];
          href = (a.getAttribute('href') || '').split('?')[0].replace(/\/+$/, '');
          if (href && href === yol) return a;
        }
        return isaretli[0];
      }
      // Hiç işaret yok: tam yol eşleşmesi, yoksa ilk sekme.
      for (j = 0; j < adaylar.length; j++) {
        a = adaylar[j];
        href = (a.getAttribute('href') || '').split('?')[0].replace(/\/+$/, '');
        if (href && href === yol) return a;
      }
      return adaylar[0] || null;
    }

    function oku(id) {
      try {
        var ham = sessionStorage.getItem(ANAHTAR);
        if (!ham) return null;
        var d = JSON.parse(ham);
        return d && d.id === id ? d : null;
      } catch (e) { return null; }
    }
    function yaz(id, veri) {
      try { sessionStorage.setItem(ANAHTAR, JSON.stringify(veri)); } catch (e) {}
    }
    function sil() {
      try { sessionStorage.removeItem(ANAHTAR); } catch (e) {}
    }

    /* Belirme animasyonunu kısa süre susturur; sonra sınıf kaldırılır ki
     * sonraki takaslarda normal davranış geri gelsin. */
    function rayonBelirmeSustur(ray, kap) {
      var hedefler = [ray];
      if (kap && kap !== ray) hedefler.push(kap);
      hedefler.forEach(function (el) { el.classList.add('tb-hap-geliyor'); });
      setTimeout(function () {
        hedefler.forEach(function (el) { el.classList.remove('tb-hap-geliyor'); });
      }, 700);
    }

    function hareket(pill, ray, hedef, canli) {
      if (!pill || !ray) return;
      if (!hedef || !hedef.offsetWidth) { pill.style.opacity = '0'; return; }

      var W = pill.offsetWidth, H = pill.offsetHeight;
      var sol = hedef.offsetLeft;
      var sag = W - (hedef.offsetLeft + hedef.offsetWidth);
      var ust = hedef.offsetTop;
      var alt = H - (hedef.offsetTop + hedef.offsetHeight);
      var yuvarlak = pill.dataset.radius || '999px';

      function yaz(l, r) {
        pill.style.clipPath =
          'inset(' + ust + 'px ' + r + 'px ' + alt + 'px ' + l + 'px round ' + yuvarlak + ')';
        pill.__lr = [l, r];   // görsel durum: art arda tıklamada sıçrama olmasın
      }

      // Aynı hedefe zaten gidiliyorsa yeniden başlatma: iki tween aynı
      // hapı yazınca değerler ileri geri zıplıyordu.
      var hedefImza = sol + ':' + sag;
      if (canli && pill.__anim && pill.__hedef === hedefImza) return;

      var onceki = pill.__lr;
      // İlk yerleşim, sayfa geçişi ya da hareket kısıtlı: anında otur.
      if (!canli || azHareket || !onceki) {
        yaz(sol, sag);
        pill.style.opacity = '1';
        return;
      }
      var sagaGidis = sol > onceki[0];
      var benim = (pill.__id = (pill.__id || 0) + 1);
      pill.__anim = true;
      pill.__hedef = hedefImza;
      var kalan = 2;
      var bitir = function () { if (--kalan === 0 && pill.__id === benim) pill.__anim = false; };

      yurusun(onceki[0], sol,
        sagaGidis ? sure.onde : sure.arkada,
        sagaGidis ? inOut3 : out3,
        function (v) { if (pill.__id === benim) { yaz(v, pill.__lr[1]); if (v === sol) bitir(); } });

      yurusun(onceki[1], sag,
        sagaGidis ? sure.arkada : sure.onde,
        sagaGidis ? out3 : inOut3,
        function (v) { if (pill.__id === benim) { yaz(pill.__lr[0], v); if (v === sag) bitir(); } });

      pill.style.opacity = '1';
    }

    /* Hap kimliği: aynı sekme çubuğu iki farklı sayfada AYNI kimliği almalı,
     * yoksa geçiş kaydı eşleşmez. Çalışma alanı çubuğu her sayfada tek;
     * diğer çubuklar kendi id'siyle ayrışır. */
    function kimlik(ray) {
      if (ray.classList.contains('ws-links')) return 'ws';
      // Açık kimlik: şablon `data-hap` verirse o kazanır.
      if (ray.dataset && ray.dataset.hap) return 'hap:' + ray.dataset.hap;
      if (ray.id) return 'ray:' + ray.id;
      /* id'siz çubuklar (İçerikler durum sekmeleri, reklamlar, SEO…) her
       * takasta YENİDEN doğuyor. Sabit bir 'adsiz' anahtarı bütün çubukları
       * aynı kayda bağlar; bu yüzden kimlik ilk bağlantının YOLUNDAN
       * türetiliyor: /admin/icerikler?durum=… → 'yol:/admin/icerikler'.
       * Aynı çubuk iki sayfada aynı kimliği alır, kayıt eşleşir. */
      var a = ray.querySelector('a[href]');
      var yol = '';
      if (a) {
        yol = (a.getAttribute('href') || '').split('?')[0].split('#')[0];
      }
      return 'yol:' + (yol || 'adsiz');
    }

    /* ray: sekme çubuğu, sec: öğe seçicisi, radius: köşe yarıçapı */
    function kur(ray, sec, radius) {
      if (!ray) return;
      sec = sec || 'a';
      radius = radius || '999px';

      if (getComputedStyle(ray).position === 'static') ray.style.position = 'relative';

      var pill = null;
      for (var i = 0; i < ray.children.length; i++) {
        if (ray.children[i].classList.contains('tb-hap')) { pill = ray.children[i]; break; }
      }
      if (!pill) {
        pill = document.createElement('span');
        pill.className = 'tb-hap';
        pill.setAttribute('aria-hidden', 'true');
        ray.appendChild(pill);
      }
      pill.dataset.radius = radius;

      // Kutu modeli ray ile birebir: kopyalar tam etiketlerin üstüne oturmalı.
      var rc = getComputedStyle(ray);
      pill.style.padding = rc.padding;
      pill.style.gap = rc.gap === 'normal' ? '0px' : rc.gap;
      pill.style.flexWrap = rc.flexWrap;
      pill.style.borderRadius = radius;

      var ogeler = [], c;
      for (var j = 0; j < ray.children.length; j++) {
        c = ray.children[j];
        if (c === pill) continue;
        if (c.matches(sec)) ogeler.push(c);
      }
      pill.textContent = '';
      ogeler.forEach(function (a) { pill.appendChild(kopyala(a)); });

      ray.classList.add('tb-hap-var');
      // Kimlik KARARLI olmalı: sayfalar arası geçiş kaydı bu anahtarla
      // eşleşiyor. Rastgele üretildiğinde her sayfa yüklemesinde yeni bir
      // kimlik doğuyor ve hap animasyonsuz, doğrudan hedefe oturuyordu.
      if (!ray.dataset.hapId) ray.dataset.hapId = kimlik(ray);

      var hedef = aktifOge(ray, sec);
      var gecis = oku(ray.dataset.hapId);
      var canli = false;
      if (gecis && hedef && Date.now() - gecis.t < 5000) {
        // Sayfalar arası: hap ESKİ sayfadaki yerinden yola çıkar. Menü iki
        // sayfada da aynı yerde olduğu için koordinatlar doğrudan geçerli.
        pill.__lr = [gecis.l, gecis.r];
        pill.style.clipPath =
          'inset(' + hedef.offsetTop + 'px ' + gecis.r + 'px ' +
          (pill.offsetHeight - (hedef.offsetTop + hedef.offsetHeight)) + 'px ' +
          gecis.l + 'px round ' + radius + ')';
        pill.style.opacity = '1';
        canli = true;
        sil();
        /* HTMX takasında kap `.htmx-added` sınıfıyla belirme animasyonu
         * oynatır (pn-yenile: opacity .35→1). Hap o kabın İÇİNDE olduğu
         * için kayarken sönük görünüyordu — ölçüldü: opacity 1 → 0 → 1.
         * Kap ve ray bu süre boyunca belirme animasyonundan muaf tutulur. */
        var kap = ray.closest('.htmx-added, .htmx-settling');
        rayonBelirmeSustur(ray, kap);
        // İlk kareyi bekle: yeni DOM'un yerleşimi otursun, ölçüm doğru olsun.
        requestAnimationFrame(function () {
          hareket(pill, ray, aktifOge(ray, sec), true);
        });
      } else {
        hareket(pill, ray, hedef, canli);
      }

      // Tıklama: hem anında tepki hem SONRAKİ sayfaya konum devri.
      if (!ray.__tbTik) {
        ray.__tbTik = true;
        ray.addEventListener('click', function (e) {
          var a = e.target.closest(sec);
          if (!a || a.parentNode !== ray) return;
          var href = a.getAttribute('href') || '';
          if (!href || href.charAt(0) === '#' || a.getAttribute('target') === '_blank') return;
          /* hx-get/hx-post sekmeleri de gezinmedir: panel takas edilir ve
           * yeni çubuk hapı ESKİ yerinden devralmalı. Önce burada
           * atlanıyordu; İçerikler'deki durum sekmeleri bu yüzden
           * kaymıyor, sadece solup beliriyordu (ölçüldü). */
          if (a.getAttribute('hx-target') && a.getAttribute('hx-target') === 'body') return;
          if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
          // Hapın O ANKİ konumu kaydedilir, tıklanan sekmenin değil:
          // yeni sayfada o sekme aktif olacağı için tıklanan konumu yazmak
          // "eski = yeni" demek olur ve hap hiç kımıldamaz.
          var anlik = pill.__lr;
          var W = pill.offsetWidth;
          yaz(ray.dataset.hapId, {
            id: ray.dataset.hapId,
            l: anlik ? anlik[0] : a.offsetLeft,
            r: anlik ? anlik[1] : W - (a.offsetLeft + a.offsetWidth),
            t: Date.now()
          });
        });
      }

      // Aktif öğe herhangi bir yolla değişebilir: HTMX değişimi, ayarlar
      // sekmelerindeki JS. Sınıf değişimini izlemek hepsini kapsar.
      if (!ray.__tbIzle) {
        var bekliyor = false;
        ray.__tbIzle = new MutationObserver(function () {
          if (bekliyor) return;
          bekliyor = true;
          requestAnimationFrame(function () {
            bekliyor = false;
            hareket(pill, ray, aktifOge(ray, sec), true);
          });
        });
        ray.__tbIzle.observe(ray, { attributes: true, attributeFilter: ['class'], subtree: true });
      }

      // Genişlik GERÇEKTEN değişince kırpma penceresini tazele.
      //
      // ResizeObserver observe() çağrısında bir kez hemen tetiklenir. Eski
      // kod bu ilk çağrıda "ani otur" yapıyor ve süren animasyonu kesiyordu:
      // hap hedefe zıplıyordu (sola gidişte belirgin). İlk değeri kaydedip
      // aynı genişlikte hiçbir şey yapmıyoruz.
      if (!ray.__tbBoyut) {
        var sonGenislik = ray.clientWidth;
        ray.__tbBoyut = new ResizeObserver(function () {
          var g = ray.clientWidth;
          if (g === sonGenislik) return;
          sonGenislik = g;
          hareket(pill, ray, aktifOge(ray, sec), false);
        });
        ray.__tbBoyut.observe(ray);
      }
      return pill;
    }

    window.tbHapKur = kur;

    /* Tüm sekme çubuklarını tara. ws-tabs JS ile sonradan eklendiği için
     * bu tarama betiğin sonunda çalışır; HTMX değişimlerinde tekrarlanır. */
    function tara() {
      document.querySelectorAll('.ws-links').forEach(function (ray) {
        if (ray.__tbKuruldu) return;
        ray.__tbKuruldu = true;
        kur(ray, 'a', '999px');
      });
      document.querySelectorAll('.tabs').forEach(function (ray) {
        if (ray.__tbKuruldu) return;
        ray.__tbKuruldu = true;
        kur(ray, 'a, button', '9px');
      });
    }

    window.tbHapTara = tara;
    tara();
    if (document.readyState !== 'complete') {
      window.addEventListener('load', tara);
    }
    // Yazı tipi sonradan yüklenirse kopyalar kayar: yeniden ölç.
    if (document.fonts && document.fonts.ready) {
      document.fonts.ready.then(function () {
        document.querySelectorAll('.ws-links, .tabs').forEach(function (ray) {
          if (!ray.__tbKuruldu) return;
          var pill = null;
          for (var i = 0; i < ray.children.length; i++) {
            if (ray.children[i].classList.contains('tb-hap')) { pill = ray.children[i]; break; }
          }
          // Yazı tipi sonradan geldiyse genişlik değişmiş olabilir; yumuşak
          // geçişle düzelt. Ama süren bir animasyon varsa KARIŞMA: iki tween
          // aynı hapı yazınca değerler ileri geri zıplıyordu.
          if (pill && !pill.__anim) {
            hareket(pill, ray, aktifOge(ray, ray.classList.contains('tabs') ? 'a, button' : 'a'), true);
          }
        });
      });
    }
    document.body.addEventListener('htmx:afterSettle', function () {
      document.querySelectorAll('.tabs').forEach(function (ray) {
        if (!ray.__tbKuruldu) {
          ray.__tbKuruldu = true;
          kur(ray, 'a, button', '9px');
        }
      });
    });

    /* ========== Yan menü hapı: dikey kayma (Pacely birebir) ==========
     *
     * Kullanıcı isteği: Pacely sol menüsündeki geçiş/animasyon farkları.
     *
     * Referans (pacely/assets/index-BlFqaHp6.js → `Le(pill, hedef, …)`,
     * paket yardımcısı `yo`):
     *
     *   • Hap `.nav__pill` tek bir span; `.side__scroll` içinde MUTLAK
     *     konumlu, `background: var(--side-tile)`, `border-radius: 12px`,
     *     `pointer-events: none`.
     *   • Aktif menü öğesinin KENDİ zemini yok
     *     (`.nav__item[aria-current=page]{background:0 0}`) — zemini hap
     *     veriyor. Bizde aktif öğe kendi kart yüzeyini çiziyordu, yani
     *     "kayan hap" değil "anında yanan kart" görünüyordu.
     *   • Geçişte ÖNCE BASKIN EKSEN bulunur:
     *       m = |Δt| > |Δl|   → dikey hareket
     *     Sonra üç tween paralel oynar:
     *       önde giden kenar   360 ms  power3.out
     *       arkada kalan kenar 550 ms  power3.inOut
     *       çapraz iki kenar   450 ms  power3.inOut
     *     Yani hap dikey inerken önce uzar, sonra toplanır; yatay
     *     sekmelerde de aynı kural (bizde iki kenar vardı, çapraz eksik).
     *
     * Kullanıcı isteği "birebir" olduğu için sayılar olduğu gibi alındı.
     */
    var SB_MS = { onde: 360, arkada: 550, capraz: 450 };
    var SB_YARICAP = 12;

    function sbCiz(pill, e) {
      pill.style.clipPath = 'inset(' + e.t + 'px ' + e.r + 'px ' + e.b + 'px ' + e.l + 'px round ' +
        SB_YARICAP + 'px)';
      pill.__e = { l: e.l, r: e.r, t: e.t, b: e.b };
    }

    /* Hapı hedefe yerleştirir. animate=false → anında (ilk yerleşim,
     * ray değişimi, yeniden boyutlandırma). */
    function sbYerlestir(pill, ray, hedef, animate) {
      if (!pill || !ray) return;
      if (!hedef || !hedef.offsetWidth) { pill.style.opacity = '0'; return; }

      var cs = getComputedStyle(ray);
      var gen = /auto|scroll/.test(cs.overflowX) && ray.scrollWidth > ray.clientWidth
        ? ray.scrollWidth : ray.offsetWidth;
      var yuk = /auto|scroll/.test(cs.overflowY) && ray.scrollHeight > ray.clientHeight
        ? ray.scrollHeight : ray.offsetHeight;

      var hedefE = {
        l: hedef.offsetLeft,
        r: gen - (hedef.offsetLeft + hedef.offsetWidth),
        t: hedef.offsetTop,
        b: yuk - (hedef.offsetTop + hedef.offsetHeight)
      };

      var eski = pill.__e;
      var ayni = eski &&
        Math.abs(eski.l - hedefE.l) < 1 &&
        Math.abs(eski.r - hedefE.r) < 1 &&
        Math.abs(eski.t - hedefE.t) < 1 &&
        Math.abs(eski.b - hedefE.b) < 1;

      // Animasyon sürerken aynı hedefe yeniden yerleşme: sıçrama olmasın.
      if (animate && ayni && pill.__anim) return;

      if (!animate || azHareket || !eski) {
        sbCiz(pill, hedefE);
        pill.style.opacity = '1';
        return;
      }

      var dikey = Math.abs(hedefE.t - eski.t) > Math.abs(hedefE.l - eski.l);
      var ileri = dikey ? hedefE.t > eski.t : hedefE.l > eski.l;

      // Önde/arkada kalan kenar ve çapraz kenarlar (bkz. `yo`).
      var on, arka, capraz;
      if (dikey) {
        on = ileri ? 'b' : 't';
        arka = ileri ? 't' : 'b';
        capraz = ['l', 'r'];
      } else {
        on = ileri ? 'r' : 'l';
        arka = ileri ? 'l' : 'r';
        capraz = ['t', 'b'];
      }

      var durum = { l: eski.l, r: eski.r, t: eski.t, b: eski.b };
      var benim = (pill.__id = (pill.__id || 0) + 1);
      pill.__anim = true;
      var kalan = 4;
      function bitir() { if (--kalan === 0 && pill.__id === benim) pill.__anim = false; }
      function yaz() { if (pill.__id === benim) sbCiz(pill, durum); }

      yurusun(durum[on], hedefE[on], SB_MS.onde, out3, function (v) {
        if (pill.__id !== benim) return;
        durum[on] = v; yaz(); if (v === hedefE[on]) bitir();
      });
      yurusun(durum[arka], hedefE[arka], SB_MS.arkada, inOut3, function (v) {
        if (pill.__id !== benim) return;
        durum[arka] = v; yaz(); if (v === hedefE[arka]) bitir();
      });
      capraz.forEach(function (k) {
        yurusun(durum[k], hedefE[k], SB_MS.capraz, inOut3, function (v) {
          if (pill.__id !== benim) return;
          durum[k] = v; yaz(); if (v === hedefE[k]) bitir();
        });
      });

      pill.style.opacity = '1';
    }

    /* Yan menü hapını kur: aktif öğeyi izle ve hapı kaydır. */
    function sbKur() {
      var ray = document.querySelector('.admin-side .side-scroll');
      if (!ray) return null;
      var pill = ray.querySelector('.tb-sb-hap');
      if (!pill) {
        pill = document.createElement('span');
        pill.className = 'tb-sb-hap';
        pill.setAttribute('aria-hidden', 'true');
        ray.insertBefore(pill, ray.firstChild);
        ray.classList.add('tb-sb-hap-var');
      }
      var hedefBul = function () {
        return ray.querySelector('.menu a.active, .menu a[aria-current="page"]');
      };
      if (!ray.__tbSbIzle) {
        ray.__tbSbIzle = true;
        var bekliyor = false;
        var mo = new MutationObserver(function () {
          if (bekliyor) return;
          bekliyor = true;
          requestAnimationFrame(function () {
            bekliyor = false;
            sbYerlestir(pill, ray, hedefBul(), true);
          });
        });
        // Yalnız sınıf/öznitelik değişimi: metin değişse yeniden yerleşir.
        mo.observe(ray, { subtree: true, attributes: true, attributeFilter: ['class', 'aria-current'] });
        if (window.ResizeObserver) {
          var ro = new ResizeObserver(function () {
            sbYerlestir(pill, ray, hedefBul(), false);
          });
          ro.observe(ray);
          ray.__tbSbBoyut = ro;
        }
        if (document.fonts && document.fonts.ready) {
          document.fonts.ready.then(function () {
            sbYerlestir(pill, ray, hedefBul(), false);
          });
        }
      }
      sbYerlestir(pill, ray, hedefBul(), false);
      return pill;
    }
    window.tbSbHapKur = sbKur;

    /* ========== Yan menünün girişi (Pacely birebir) ==================
     *
     * Referans (aynı dosya, uygulama bağlanma efekti):
     *   .side              x −28 → 0, opacity 0 → 1, 700 ms, expo.out
     *   .side__head / .side__group / .nav__item / .side__foot > *
     *                      opacity 0 → 1, x −12 → 0,
     *                      500 ms, power3.out, kademe 25 ms, gecikme 100 ms
     *   .nav__pill         scaleX .4 → 1, opacity 0 → 1,
     *                      600 ms, back.out(2), origin 0% 50%, gecikme 450 ms
     *
     * Bizde panel kabuğu SSR olduğu ve boost yalnız .admin-main'i
     * değiştirdiği için bu efekt yalnız TAM sayfa yüklemesinde bir kez
     * oynar — Pacely'nin SPA "mount" anının karşılığı budur.
     */
    function sbGiris() {
      if (azHareket) return;
      if (window.innerWidth <= 760) return; // Pacely de dar ekranda atlıyor
      var yan = document.querySelector('.admin-side');
      if (!yan || !yan.animate) return;
      if (yan.__tbGirdi) return;
      yan.__tbGirdi = true;

      var EXPO = 'cubic-bezier(.19,1,.22,1)';        // expo.out
      var P3OUT = 'cubic-bezier(.215,.61,.355,1)';   // power3.out
      // back.out(2): GSAP'in yay geri tepmesi; cubic-bezier yaklaşıklığı.
      var BACK = 'cubic-bezier(.34,1.56,.64,1)';

      var temizle = function (el, a) {
        a.onfinish = function () {
          el.style.transform = '';
          el.style.opacity = '';
          try { a.cancel(); } catch (e) {}
        };
      };

      var govde = yan.animate(
        [{ transform: 'translateX(-28px)', opacity: 0 }, { transform: 'translateX(0px)', opacity: 1 }],
        { duration: 700, easing: EXPO, fill: 'both' }
      );
      temizle(yan, govde);

      /* Pacely hedefleri: .side__head, .side__group, .nav__item,
       * .side__foot > *. Bizde grup başlığı yok; menü öğeleri <a>. */
      var ogeler = yan.querySelectorAll('.side-head, .menu a');
      Array.prototype.forEach.call(ogeler, function (el, i) {
        var a = el.animate(
          [{ transform: 'translateX(-12px)', opacity: 0 }, { transform: 'translateX(0px)', opacity: 1 }],
          { duration: 500, easing: P3OUT, delay: 100 + i * 25, fill: 'both' }
        );
        temizle(el, a);
      });

      var hap = yan.querySelector('.tb-sb-hap');
      if (hap) {
        hap.style.transformOrigin = '0% 50%';
        var a = hap.animate(
          [{ transform: 'scaleX(.4)', opacity: 0 }, { transform: 'scaleX(1)', opacity: 1 }],
          { duration: 600, easing: BACK, delay: 450, fill: 'both' }
        );
        /* DİKKAT: Pacely bu tweende `clearProps:'transform'` — yalnız
         * transform temizlenir, opacity SATIR İÇİ 1 KALIR (CSS'te
         * `.nav__pill{opacity:0}`). Bizde de aynısı olmalı: `temizle()`
         * opacity'yi de silseydi hap CSS'teki 0'a düşer ve menüde hiç
         * aktif zemin görünmezdi (ölçüldü: opacity 0). */
        a.onfinish = function () {
          hap.style.transform = '';
          hap.style.opacity = '1';
          try { a.cancel(); } catch (e) {}
        };
      }
    }
    window.tbSbGiris = sbGiris;
  })();

  /* ---------- 4) AI SEO önerisini uygula (Faz 188) ----------
   *
   * Öneri paneli formun içinde açılıyor. "İki alanı doldur" düğmesi SEO
   * başlığı ve meta açıklamayı TEK adımda yazar; değerler düğmenin
   * data-* özniteliklerinde taşınıyor, ek istek gerekmiyor.
   *
   * Neden `input` olayı da tetikleniyor: formda karakter sayacı, SERP
   * önizlemesi ve sosyal kart önizlemesi `input` dinliyor. Yalnız value
   * yazmak önizlemeyi eski değerde bırakıyordu.
   */
  (function seoOneri() {
    function alan(sec) { return document.querySelector(sec); }

    function yaz(el, deger) {
      if (!el || !deger) return;
      el.value = deger;
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
    }

    document.addEventListener('click', function (e) {
      var uygula = e.target.closest('[data-seo-uygula]');
      if (uygula) {
        var baslik = uygula.getAttribute('data-baslik') || '';
        var aciklama = uygula.getAttribute('data-aciklama') || '';
        yaz(alan('[name=seo_baslik]'), baslik);
        yaz(alan('[name=seo_aciklama]'), aciklama);
        var eksik = [];
        if (baslik && !alan('[name=seo_baslik]')) eksik.push('SEO başlığı');
        if (aciklama && !alan('[name=seo_aciklama]')) eksik.push('meta açıklama');
        if (window.tbToast) {
          if (eksik.length) {
            window.tbToast('Form alanı bulunamadı: ' + eksik.join(', '), 'hata');
          } else {
            window.tbToast('SEO alanları dolduruldu — kaydetmeyi unutmayın.', 'ok');
          }
        }
        return;
      }

      var kapat = e.target.closest('[data-seo-kapat]');
      if (kapat) {
        var panel = document.getElementById('seoOneri');
        if (panel) panel.innerHTML = '';
        return;
      }
    });
  })();

  /* ================= Yan menü: ilk kurulum =================
   *
   * NEDEN BURADA: `tbAdminInit()` dosyanın ortasında (satır ~952) çağrılıyor
   * ama `tbSbHapKur` / `tbSbGiris` çok sonra, `kayganHap` IIFE'sinin
   * içinde tanımlanıyor. İlk çağrıda `window.tbSbHapKur` henüz yoktu,
   * dolayısıyla hap sayfa yüklenirken HİÇ kurulmuyor; ancak bir gezinmeden
   * sonra (afterSettle → tbAdminInit) ortaya çıkıyordu (ölçüldü).
   * Kurulum dosyanın en sonunda, tanımlardan SONRA yapılır. */
  if (window.tbSbHapKur) window.tbSbHapKur();
  if (window.tbSbGiris) window.tbSbGiris();

})();
