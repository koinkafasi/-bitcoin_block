
(function () {
  'use strict';

  var LS = {
    get: function (k, d) { try { var v = localStorage.getItem(k); return v === null ? d : JSON.parse(v); } catch (e) { return d; } },
    set: function (k, v) { try { localStorage.setItem(k, JSON.stringify(v)); } catch (e) {} }
  };

  // Pano içeriği ortak kabuğun içinde yaşıyor; kapı .pn-view.
  var app = document.querySelector('.pn-view');
  if (!app) return;

  
  (function sekmeler() {
    var seg = document.getElementById('pnSeg');
    var thumb = document.getElementById('pnSegThumb');
    if (!seg || !thumb) return;
    var btnlar = [].slice.call(seg.querySelectorAll('button'));

    function konumla(btn) {
      if (!btn) return;
      thumb.style.width = btn.offsetWidth + 'px';
      thumb.style.transform = 'translateX(' + (btn.offsetLeft - 3) + 'px)';
    }
    function sec(btn) {
      btnlar.forEach(function (b) { b.setAttribute('aria-pressed', b === btn ? 'true' : 'false'); });
      konumla(btn);
      var key = btn.getAttribute('data-pn-seri');
      var wraps = document.querySelectorAll('.pn-seriwrap');
      [].forEach.call(wraps, function (w) {
        w.classList.toggle('on', w.getAttribute('data-pn-w') === key);
      });
      LS.set('pn:aralik', key);
    }

    var kayitli = LS.get('pn:aralik', null);
    var aktif = null;
    if (kayitli) {
      aktif = btnlar.filter(function (b) { return b.getAttribute('data-pn-seri') === kayitli; })[0] || null;
    }
    if (!aktif) {
      aktif = btnlar.filter(function (b) { return b.getAttribute('aria-pressed') === 'true'; })[0] || btnlar[0];
    }
    btnlar.forEach(function (b) {
      b.addEventListener('click', function () { sec(b); });
    });
    // İlk yerleşim: fontlar yüklenmeden ölçmek yanlış genişlik verir.
    requestAnimationFrame(function () { sec(aktif); });
    window.addEventListener('load', function () { konumla(seg.querySelector('button[aria-pressed="true"]')); });
    window.addEventListener('resize', function () { konumla(seg.querySelector('button[aria-pressed="true"]')); });
    if (document.fonts && document.fonts.ready) {
      document.fonts.ready.then(function () { konumla(seg.querySelector('button[aria-pressed="true"]')); });
    }
  })();

  
  (function balon() {
    var box = document.getElementById('pnChartBox');
    var tip = document.getElementById('pnBalon');
    if (!box || !tip) return;

    function goster(el) {
      var v = el.getAttribute('data-v');
      if (v === null) return;
      var l = el.getAttribute('data-l') || '';
      tip.textContent = (l ? l + ' · ' : '') + v;
      var br = el.getBoundingClientRect();
      var ar = box.getBoundingClientRect();
      tip.hidden = false;
      tip.style.left = (br.left + br.width / 2 - ar.left) + 'px';
      tip.style.top = (br.top - ar.top) + 'px';
    }
    function gizle() { tip.hidden = true; }

    box.addEventListener('mouseover', function (e) {
      var el = e.target.closest('.pn-nokta, .pn-cubuk');
      if (el) goster(el); else gizle();
    });
    box.addEventListener('mouseleave', gizle);
    // Klavye: odaklanılabilir öğe yok, ama dokunmatikte ilk dokunuş bilgi verir.
    box.addEventListener('touchstart', function (e) {
      var el = e.target.closest('.pn-nokta, .pn-cubuk');
      if (el) goster(el);
    }, { passive: true });
  })();

  
  (function serit() {
    var ray = document.getElementById('pnPlanRay');
    if (!ray) return;
    var url = ray.getAttribute('data-url');
    var suruklenen = null;
    var eskiSol = '';

    function toast(mesaj, hata) {
      // Ortak balon sistemi varsa onu kullan (tek görünüm, tek davranış).
      if (window.tbToast) { window.tbToast(mesaj, hata ? 'hata' : 'ok'); return; }
      var kutu = document.getElementById('pnToasts');
      if (!kutu) return;
      var t = document.createElement('div');
      t.className = 'pn-toast' + (hata ? ' pn-toast--hata' : '');
      t.textContent = mesaj;
      kutu.appendChild(t);
      setTimeout(function () {
        t.style.opacity = '0';
        t.style.transition = 'opacity .25s ease';
        setTimeout(function () { t.remove(); }, 260);
      }, 2600);
    }

    // Sürükleme konumundan yüzde → saat. Pencere 36 saat: -6s … +30s.
    function yuzdeSaat(pct) {
      var now = new Date();
      var bas = new Date(now.getTime() - 6 * 3600 * 1000);
      var toplam = 36 * 3600 * 1000;
      var t = new Date(bas.getTime() + (pct / 100) * toplam);
      // 15 dakikaya yuvarla: sürükleme hassasiyeti sahte bir kesinlik
      // üretmesin, yayın saatleri çeyrek saatlik olsun.
      t.setMinutes(Math.round(t.getMinutes() / 15) * 15, 0, 0);
      return t;
    }

    function saatMetni(d) {
      var p = function (n) { return (n < 10 ? '0' : '') + n; };
      return p(d.getDate()) + '.' + p(d.getMonth() + 1) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes());
    }

    ray.addEventListener('dragstart', function (e) {
      var el = e.target.closest('.pn-visit');
      if (!el) return;
      suruklenen = el;
      eskiSol = el.style.left;
      el.classList.add('is-drag');
      try { e.dataTransfer.setData('text/plain', el.getAttribute('data-id')); } catch (err) {}
      e.dataTransfer.effectAllowed = 'move';
    });

    ray.addEventListener('dragend', function () {
      if (suruklenen) suruklenen.classList.remove('is-drag');
      suruklenen = null;
    });

    ray.addEventListener('dragover', function (e) {
      if (!suruklenen) return;
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
    });

    ray.addEventListener('drop', function (e) {
      if (!suruklenen) return;
      e.preventDefault();
      var r = ray.getBoundingClientRect();
      var pct = Math.max(0, Math.min(96, ((e.clientX - r.left) / r.width) * 100));
      var el = suruklenen;
      var id = el.getAttribute('data-id');
      var yeniSaat = yuzdeSaat(pct);

      // Optimistic: kart hemen taşınır, sunucu reddederse geri alınır.
      el.style.left = pct.toFixed(2) + '%';
      var kucuk = el.querySelector('small');
      if (kucuk) kucuk.textContent = saatMetni(yeniSaat);

      var govde = new URLSearchParams();
      govde.set('id', id);
      govde.set('saat', yeniSaat.toISOString());

      fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
        body: govde.toString(),
        credentials: 'same-origin'
      }).then(function (r) { return r.json().catch(function () { return {}; }); })
        .then(function (j) {
          if (!j || !j.ok) throw new Error((j && j.hata) || 'kaydedilemedi');
          el.classList.add('is-yeni');
          setTimeout(function () { el.classList.remove('is-yeni'); }, 500);
          // Etiketi SUNUCUdan gelen biçimle yaz: istemci saati ile sunucu
          // saati farklı dilimdeyse kartta ve balonda iki farklı saat
          // görünürdü. Tek doğru kaynak kaydın kendisi.
          if (kucuk && j.etiket) kucuk.textContent = j.etiket;
          toast('Yayın saati güncellendi · ' + (j.etiket || saatMetni(yeniSaat)));
        })
        .catch(function (err) {
          el.style.left = eskiSol;
          if (kucuk) kucuk.textContent = (el.getAttribute('title') || '').split(' · ')[0];
          toast('Taşınamadı: ' + err.message, true);
        });
    });

    // Klavye ile taşıma: odaklan, ok tuşlarıyla 1 saat, Shift ile 15 dk.
    ray.addEventListener('keydown', function (e) {
      var el = e.target.closest('.pn-visit');
      if (!el) return;
      var ok = { ArrowLeft: -1, ArrowRight: 1 }[e.key];
      if (!ok) return;
      e.preventDefault();
      var pct = parseFloat(el.style.left) || 0;
      var adim = (e.shiftKey ? 15 / 60 : 1) * (100 / 36); // 36 saatlik pencere
      var yeni = Math.max(0, Math.min(96, pct + ok * adim));
      el.style.left = yeni.toFixed(2) + '%';
      el.dispatchEvent(new CustomEvent('pn:klavye-tasi', { bubbles: false }));
      // Enter'a basınca kaydet (kazara taşımayı önlemek için).
      el.dataset.bekleyen = yeniSaatIcin(yeni).toISOString();
    });

    ray.addEventListener('keyup', function (e) {
      var el = e.target.closest('.pn-visit');
      if (!el || e.key !== 'Enter' || !el.dataset.bekleyen) return;
      var id = el.getAttribute('data-id');
      var saat = el.dataset.bekleyen;
      delete el.dataset.bekleyen;
      fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
        body: 'id=' + encodeURIComponent(id) + '&saat=' + encodeURIComponent(saat),
        credentials: 'same-origin'
      }).then(function (r) { return r.json(); }).then(function (j) {
        if (j && j.ok) {
          var k = el.querySelector('small');
          if (k) k.textContent = j.etiket;
          toast('Yayın saati güncellendi · ' + j.etiket);
        } else { toast('Taşınamadı', true); }
      }).catch(function () { toast('Taşınamadı', true); });
    });

    function yuzdeSaatIcin(pct) { return yuzdeSaat(pct); }
  })();

  
  (function palet() {
    var btn = document.getElementById('pnAra');
    if (!btn) return;
    btn.addEventListener('click', function () {
      // Panelin mevcut komut paleti varsa onu aç; yoksa içerik aramaya git.
      var cmdk = document.getElementById('cmdk');
      if (cmdk) {
        if (window.tbCmdkOpen) { window.tbCmdkOpen(); return; }
        cmdk.hidden = false;
        var inp = document.getElementById('cmdkInput');
        if (inp) inp.focus();
        return;
      }
      location.href = '/admin/icerikler';
    });
  })();

  

  
  (function magnetik() {
    var el = document.querySelector('.pn-magnetic');
    if (!el || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    if (window.matchMedia('(hover: none)').matches) return;
    var sinir = 60;
    el.addEventListener('mousemove', function (e) {
      var r = el.getBoundingClientRect();
      var dx = (e.clientX - (r.left + r.width / 2)) / r.width;
      var dy = (e.clientY - (r.top + r.height / 2)) / r.height;
      el.style.transform = 'translate(' + (dx * 4).toFixed(2) + 'px,' + (dy * 3 - 1).toFixed(2) + 'px)';
    });
    el.addEventListener('mouseleave', function () { el.style.transform = ''; });
    if (!sinir) return; // referans: sabiti okunur tutmak için
  })();
})();
