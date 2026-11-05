/* Gazete Manşetleri — şerit + yakınlaştırma görüntüleyici (Faz 197c).
 *
 * İKİ YÜZEY, TEK MOTOR:
 *   1) Ana sayfa modülü (.gzm)   → dar bölgede tek slayt, geniş bölgede şerit
 *   2) Galeri sayfası (/gazete-mansetleri) → ızgara
 * Kapağa tıklama HER İKİ yüzeyde de aynı büyütücüyü açar. Kaynağa (CDN'e)
 * gidilmez, yeni sekme açılmaz.
 *
 * YAKINLAŞTIRMA ÖZELLİKLERİ:
 *   • Sığdır / %100 / kademeli yakınlaştırma (düğmeler)
 *   • Tekerlek ile yakınlaştırma — imlecin altındaki nokta sabit kalır
 *   • Sürükleyerek gezinme (görüntü kenarı dışına taşmaz)
 *   • Çift tıklama: sığdır ↔ %200
 *   • Dokunmatik: iki parmakla yakınlaştırma + tek parmakla kaydırma
 *   • Klavye: ← → gazete, + − yakınlaştır, 0 sığdır, Esc kapat
 *   • Galeride ← → ile gazeteler arasında gezinme (sayaç: 3 / 30)
 *
 * Şerit JS olmadan da çalışır (CSS scroll-snap). Bu dosya konfor katmanıdır.
 */
(function () {
  'use strict';

  var AZAMI_OLCEK = 6;      /* %600 */
  var KADEMELER = [1, 1.5, 2, 3, 4, 6];
  var IPUCU_GIZLE_MS = 4200;

  function azaltHareket() {
    return !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
  }

  function sinirla(v, alt, ust) {
    return v < alt ? alt : (v > ust ? ust : v);
  }

  function svg(yol, ek) {
    return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" ' +
      'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"' + (ek || '') + '>' +
      yol + '</svg>';
  }

  var IKON = {
    once: svg('<path d="m15 18-6-6 6-6"/>'),
    sonra: svg('<path d="m9 18 6-6-6-6"/>'),
    arti: svg('<path d="M12 5v14M5 12h14"/>'),
    eksi: svg('<path d="M5 12h14"/>'),
    sigdir: svg('<path d="M8 3H5a2 2 0 0 0-2 2v3M16 3h3a2 2 0 0 1 2 2v3M8 21H5a2 2 0 0 1-2-2v-3M16 21h3a2 2 0 0 0 2-2v-3"/>'),
    kapat: svg('<path d="M18 6 6 18M6 6l12 12"/>'),
    buyut: svg('<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5M11 8v6M8 11h6"/>')
  };

  /* =====================================================================
     YAKINLAŞTIRMA GÖRÜNTÜLEYİCİ — sayfa başına tek örnek
     ===================================================================== */
  var G = null;

  function gorselOrnek() {
    if (G) return G;

    var d = document.createElement('dialog');
    d.className = 'gzz';
    d.setAttribute('aria-label', 'Gazete manşeti — büyütülmüş görünüm');
    d.innerHTML =
      '<div class="gzz-ic">' +
        '<div class="gzz-bar">' +
          '<div class="gzz-bilgi">' +
            '<span class="gzz-ad" data-gzz-ad></span>' +
            '<span class="gzz-tarih" data-gzz-tarih></span>' +
          '</div>' +
          '<div class="gzz-araclar">' +
            '<button type="button" class="gzz-btn" data-gzz-once aria-label="Önceki gazete" title="Önceki (←)">' + IKON.once + '</button>' +
            '<span class="gzz-sayac" data-gzz-sayac aria-live="polite"></span>' +
            '<button type="button" class="gzz-btn" data-gzz-sonra aria-label="Sonraki gazete" title="Sonraki (→)">' + IKON.sonra + '</button>' +
            '<span class="gzz-ayrac" aria-hidden="true"></span>' +
            '<button type="button" class="gzz-btn" data-gzz-eksi aria-label="Uzaklaştır" title="Uzaklaştır (−)">' + IKON.eksi + '</button>' +
            '<span class="gzz-yuzde" data-gzz-yuzde aria-live="polite">%100</span>' +
            '<button type="button" class="gzz-btn" data-gzz-arti aria-label="Yakınlaştır" title="Yakınlaştır (+)">' + IKON.arti + '</button>' +
            '<button type="button" class="gzz-btn" data-gzz-sigdir aria-label="Ekrana sığdır" title="Ekrana sığdır (0)">' + IKON.sigdir + '</button>' +
            '<button type="button" class="gzz-btn gzz-kapat" data-gzz-kapat aria-label="Kapat" title="Kapat (Esc)">' + IKON.kapat + '</button>' +
          '</div>' +
        '</div>' +
        '<div class="gzz-sahne" data-gzz-sahne>' +
          '<img class="gzz-gorsel" data-gzz-gorsel alt="" decoding="async" draggable="false">' +
        '</div>' +
        '<p class="gzz-ipucu" data-gzz-ipucu>Tekerlek ile yakınlaştır · sürükleyerek gezin · çift tıkla: sığdır ↔ %200</p>' +
      '</div>';

    document.body.appendChild(d);

    var g = {
      dialog: d,
      sahne: d.querySelector('[data-gzz-sahne]'),
      img: d.querySelector('[data-gzz-gorsel]'),
      adEl: d.querySelector('[data-gzz-ad]'),
      tarihEl: d.querySelector('[data-gzz-tarih]'),
      sayacEl: d.querySelector('[data-gzz-sayac]'),
      yuzdeEl: d.querySelector('[data-gzz-yuzde]'),
      ipucuEl: d.querySelector('[data-gzz-ipucu]'),
      onceBtn: d.querySelector('[data-gzz-once]'),
      sonraBtn: d.querySelector('[data-gzz-sonra]'),
      ogeler: [],
      i: 0,
      olcek: 1,     /* 1 = görüntünün gerçek boyutu */
      taban: 1,     /* "sığdır" ölçeği */
      x: 0,
      y: 0,
      dogalW: 0,
      dogalH: 0,
      odakGeri: null,
      ipucuZaman: 0,
      surukleme: null,
      sonTasindi: false,
      kivilcim: []
    };
    G = g;

    /* --- kapama ---
       ÖLÇÜM (Chromium 131, gerçek tarayıcı testi): dinamik oluşturulmuş bir
       <dialog> üzerinde d.close() çağrısı `open` durumunu false yapıyor ama
       `close` OLAYI 250 ms içinde gelmiyor. Odağı ve "kapandı"
       bildirimini bu olaya bırakmak, kapatınca odağın iletişim kutusunun
       içinde kalmasına ve şeridin sonsuza kadar duraklamasına yol açıyordu.
       Artık ikisi de kapatma anında yapılır; `close` olayı yalnız yedek
       (çift bildirim kapalıBildirildi ile engellenir). */
    var kapaliBildirildi = false;

    /* Odağı tetikleyici karta geri verir. Kapatma SONRASI tarayıcı kendi
       odak geri yüklemesini gecikmeli yapabildiği için tek deneme yetmiyor
       (ölçüm: gerçek galeri sayfasında ilk deneme kaybediliyordu). Bu yüzden
       birkaç kez, kısa aralıklarla denenir; başarılı olunca bırakılır. */
    function odakGeriVer() {
      var geri = g.odakGeri;
      if (!geri) return;
      if (document.activeElement === geri) { g.odakGeri = null; return; }
      if (typeof geri.focus === 'function') geri.focus();
      if (document.activeElement === geri) g.odakGeri = null;
    }

    function kapatBildir() {
      if (kapaliBildirildi) return;
      kapaliBildirildi = true;
      document.dispatchEvent(new CustomEvent('gm:zoom-kapat'));
      odakGeriVer();
      window.requestAnimationFrame(odakGeriVer);
      window.setTimeout(odakGeriVer, 70);
      window.setTimeout(odakGeriVer, 220);
    }

    function kapat() {
      if (typeof d.close === 'function' && d.open) d.close();
      else d.removeAttribute('open');
      kapatBildir();
    }
    d.querySelector('[data-gzz-kapat]').addEventListener('click', kapat);

    /* Arka plana tıklama: yalnız sahnenin BOŞ alanına denk gelirse kapat.
       Görüntüye tıklamak kapatmaz (yakınlaştırmak için tıklanır).
       Sürükleme sonrası gelen tıklama da kapatmaz. */
    g.sahne.addEventListener('click', function (e) {
      if (g.sonTasindi) { g.sonTasindi = false; return; }
      if (e.target === g.sahne) kapat();
    });

    d.addEventListener('close', function () {
      odakGeriVer();
      kapatBildir();
    });
    d.addEventListener('cancel', function (e) {
      /* Esc: tarayıcı varsayılanı dialogu kapatır; ek iş yok. */
      if (e && e.preventDefault) e.preventDefault();
      kapat();
    });

    /* --- ölçekleme --- */
    function uygula(animasyon) {
      var w = g.dogalW * g.olcek;
      var h = g.dogalH * g.olcek;
      g.img.style.width = w ? w + 'px' : '';
      g.img.style.height = h ? h + 'px' : '';
      g.img.style.transform = 'translate(' + g.x + 'px,' + g.y + 'px)';
      g.img.style.transition = (animasyon && !azaltHareket()) ? 'transform .16s ease' : 'none';
      g.yuzdeEl.textContent = '%' + Math.round(g.olcek * 100);
      var tek = g.ogeler.length <= 1;
      g.onceBtn.disabled = tek;
      g.sonraBtn.disabled = tek;
      g.sahne.classList.toggle('gzz-kaydir', g.olcek > g.taban + 0.001);
    }

    function sinirlaKonum() {
      var sw = g.sahne.clientWidth;
      var sh = g.sahne.clientHeight;
      var w = g.dogalW * g.olcek;
      var h = g.dogalH * g.olcek;
      var maxX = Math.max(0, (w - sw) / 2);
      var maxY = Math.max(0, (h - sh) / 2);
      g.x = sinirla(g.x, -maxX, maxX);
      g.y = sinirla(g.y, -maxY, maxY);
    }

    /* Ölçeği, sahne merkezine göre (cx, cy) noktası sabit kalacak biçimde
       değiştirir. cx/cy sahne merkezinden uzaklıktır (px). */
    function olcekle(yeni, cx, cy, animasyon) {
      var eski = g.olcek;
      var hedef = sinirla(yeni, g.taban, AZAMI_OLCEK);
      if (Math.abs(hedef - eski) < 0.0005) return;
      if (cx === undefined) { cx = 0; cy = 0; }
      /* İmlecin altındaki görüntü noktası: (cx - x) / eski */
      var ix = (cx - g.x) / eski;
      var iy = (cy - g.y) / eski;
      g.olcek = hedef;
      g.x = cx - ix * hedef;
      g.y = cy - iy * hedef;
      sinirlaKonum();
      uygula(animasyon);
    }

    function sigdir(animasyon) {
      g.olcek = g.taban;
      g.x = 0;
      g.y = 0;
      uygula(animasyon);
    }

    /* Sığdır ölçeği: görüntü sahneye tam sığar; %100'ün üstüne çıkmaz. */
    function tabanHesapla() {
      var sw = g.sahne.clientWidth || 1;
      var sh = g.sahne.clientHeight || 1;
      if (!g.dogalW || !g.dogalH) return 1;
      return Math.min(1, Math.min(sw / g.dogalW, sh / g.dogalH));
    }

    function kademeli(yon) {
      var mevcut = g.olcek;
      var hedef = mevcut;
      if (yon > 0) {
        for (var i = 0; i < KADEMELER.length; i++) {
          if (KADEMELER[i] > mevcut + 0.001) { hedef = KADEMELER[i]; break; }
        }
        if (hedef <= mevcut) hedef = AZAMI_OLCEK;
      } else {
        for (var j = KADEMELER.length - 1; j >= 0; j--) {
          if (KADEMELER[j] < mevcut - 0.001) { hedef = KADEMELER[j]; break; }
        }
        if (hedef >= mevcut) hedef = g.taban;
      }
      olcekle(hedef, 0, 0, true);
    }

    d.querySelector('[data-gzz-arti]').addEventListener('click', function () { kademeli(1); });
    d.querySelector('[data-gzz-eksi]').addEventListener('click', function () { kademeli(-1); });
    d.querySelector('[data-gzz-sigdir]').addEventListener('click', function () { sigdir(true); });

    /* --- gezinme --- */
    function git(n) {
      if (g.ogeler.length === 0) return;
      g.i = ((n % g.ogeler.length) + g.ogeler.length) % g.ogeler.length;
      yukle(g.ogeler[g.i]);
    }
    g.onceBtn.addEventListener('click', function () { git(g.i - 1); });
    g.sonraBtn.addEventListener('click', function () { git(g.i + 1); });

    /* --- tekerlek: imlecin altındaki nokta sabit --- */
    g.sahne.addEventListener('wheel', function (e) {
      if (!g.dogalW) return;
      e.preventDefault();
      var r = g.sahne.getBoundingClientRect();
      var cx = e.clientX - r.left - r.width / 2;
      var cy = e.clientY - r.top - r.height / 2;
      var k = Math.exp(-e.deltaY * (e.deltaMode === 1 ? 0.05 : 0.0016));
      olcekle(g.olcek * k, cx, cy, false);
      ipucuGizle();
    }, { passive: false });

    /* --- çift tıklama --- */
    g.sahne.addEventListener('dblclick', function (e) {
      if (!g.dogalW) return;
      e.preventDefault();
      if (g.olcek <= g.taban + 0.02) {
        var r = g.sahne.getBoundingClientRect();
        olcekle(2, e.clientX - r.left - r.width / 2, e.clientY - r.top - r.height / 2, true);
      } else {
        sigdir(true);
      }
    });

    /* --- sürükleme (fare + tek parmak) --- */
    g.img.style.touchAction = 'none';
    g.sahne.addEventListener('pointerdown', function (e) {
      if (e.pointerType === 'mouse' && e.button !== 0) return;
      g.kivilcim = g.kivilcim.filter(function (k) { return k.id !== e.pointerId; });
      g.kivilcim.push({ id: e.pointerId, x: e.clientX, y: e.clientY });
      if (g.kivilcim.length > 2) g.kivilcim.shift();

      if (g.kivilcim.length === 1) {
        g.surukleme = {
          x0: e.clientX, y0: e.clientY, gx: g.x, gy: g.y, tasindi: false
        };
      } else if (g.kivilcim.length === 2) {
        var a = g.kivilcim[0];
        var b = g.kivilcim[1];
        g.kivilcimMesafe = Math.hypot(a.x - b.x, a.y - b.y) || 1;
        g.kivilcimOlcek = g.olcek;
        g.kivilcimMerkez = { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
        g.surukleme = null;
      }
    });

    g.sahne.addEventListener('pointermove', function (e) {
      var kayit = null;
      for (var i = 0; i < g.kivilcim.length; i++) {
        if (g.kivilcim[i].id === e.pointerId) { kayit = g.kivilcim[i]; break; }
      }
      if (!kayit) return;
      kayit.x = e.clientX;
      kayit.y = e.clientY;

      if (g.kivilcim.length >= 2 && g.kivilcimMerkez) {
        /* İki parmak: mesafe oranı kadar yakınlaştır, merkez sabit. */
        var a = g.kivilcim[0];
        var b = g.kivilcim[1];
        var mesafe = Math.hypot(a.x - b.x, a.y - b.y) || 1;
        var r = g.sahne.getBoundingClientRect();
        var cx = g.kivilcimMerkez.x - r.left - r.width / 2;
        var cy = g.kivilcimMerkez.y - r.top - r.height / 2;
        olcekle(g.kivilcimOlcek * (mesafe / g.kivilcimMesafe), cx, cy, false);
        ipucuGizle();
        return;
      }

      if (!g.surukleme) return;
      var dx = e.clientX - g.surukleme.x0;
      var dy = e.clientY - g.surukleme.y0;
      if (Math.abs(dx) + Math.abs(dy) > 3) g.surukleme.tasindi = true;
      /* Sığdır durumunda kaydırma yok: görüntü zaten tam görünüyor. */
      if (g.olcek <= g.taban + 0.001) return;
      if (e.pointerType !== 'mouse' && !g.surukleme.tasindi) return;
      e.preventDefault();
      g.x = g.surukleme.gx + dx;
      g.y = g.surukleme.gy + dy;
      sinirlaKonum();
      uygula(false);
    });

    function birak(e) {
      if (g.surukleme) g.sonTasindi = g.surukleme.tasindi;
      g.kivilcim = g.kivilcim.filter(function (k) { return k.id !== e.pointerId; });
      if (g.kivilcim.length < 2) {
        g.kivilcimMerkez = null;
        g.kivilcimMesafe = 0;
      }
      if (g.kivilcim.length === 0) g.surukleme = null;
    }
    g.sahne.addEventListener('pointerup', birak);
    g.sahne.addEventListener('pointercancel', birak);
    g.sahne.addEventListener('pointerleave', function (e) {
      if (e.pointerType === 'mouse' && !g.surukleme) birak(e);
    });

    /* --- klavye --- */
    d.addEventListener('keydown', function (e) {
      switch (e.key) {
        case 'ArrowLeft': e.preventDefault(); git(g.i - 1); break;
        case 'ArrowRight': e.preventDefault(); git(g.i + 1); break;
        case '+': case '=': e.preventDefault(); kademeli(1); break;
        case '-': case '_': e.preventDefault(); kademeli(-1); break;
        case '0': e.preventDefault(); sigdir(true); break;
        default: break;
      }
    });

    /* --- ipucu --- */
    function ipucuGizle() {
      window.clearTimeout(g.ipucuZaman);
      g.ipucuEl.classList.add('gzz-ipucu-gizli');
    }
    g.ipucuZaman = window.setTimeout(function () {
      g.ipucuEl.classList.add('gzz-ipucu-gizli');
    }, IPUCU_GIZLE_MS);

    /* --- görüntü yükleme --- */
    function yukle(oge) {
      var ad = oge.getAttribute('data-gm-ad') || '';
      var tarih = oge.getAttribute('data-gm-tarih') || '';
      var buyuk = oge.getAttribute('data-gm-buyuk') || '';
      if (!buyuk) {
        var ic = oge.querySelector('img');
        buyuk = ic ? ic.getAttribute('src') : '';
      }
      g.adEl.textContent = ad;
      g.tarihEl.textContent = tarih;
      g.sayacEl.textContent = g.ogeler.length > 1 ? (g.i + 1) + ' / ' + g.ogeler.length : '';
      g.img.alt = (ad ? ad + ' — ' : '') + (tarih ? tarih + ' ' : '') + 'manşeti';
      d.classList.remove('gzz-hazir');
      d.classList.add('gzz-yukleniyor');
      g.dogalW = 0;
      g.dogalH = 0;

      function hazir() {
        g.dogalW = g.img.naturalWidth || 900;
        g.dogalH = g.img.naturalHeight || 1310;
        g.taban = tabanHesapla();
        sigdir(false);
        d.classList.remove('gzz-yukleniyor');
        d.classList.add('gzz-hazir');
        /* Sığdır sırasında sahne ölçüsü değişmiş olabilir; bir kez daha. */
        window.requestAnimationFrame(function () {
          var yeni = tabanHesapla();
          if (Math.abs(yeni - g.taban) > 0.001) {
            g.taban = yeni;
            sigdir(false);
          }
        });
      }

      if (g.img.getAttribute('src') === buyuk && g.img.complete) {
        hazir();
      } else {
        g.img.onload = hazir;
        g.img.onerror = function () {
          d.classList.remove('gzz-yukleniyor');
          d.classList.add('gzz-hata');
        };
        g.img.src = buyuk;
      }
      window.clearTimeout(g.ipucuZaman);
      g.ipucuEl.classList.remove('gzz-ipucu-gizli');
      g.ipucuZaman = window.setTimeout(function () {
        g.ipucuEl.classList.add('gzz-ipucu-gizli');
      }, IPUCU_GIZLE_MS);
    }

    /* --- açılış --- */
    function ac(ogeler, i, tetik) {
      g.ogeler = ogeler;
      g.i = i;
      g.odakGeri = tetik || null;
      kapaliBildirildi = false;
      d.classList.remove('gzz-hata');
      if (typeof d.showModal === 'function') {
        if (!d.open) d.showModal();
      } else {
        d.setAttribute('open', 'open');
      }
      yukle(ogeler[i]);
      document.dispatchEvent(new CustomEvent('gm:zoom-ac'));
      window.setTimeout(function () {
        var k = d.querySelector('[data-gzz-kapat]');
        if (k) k.focus();
      }, 30);
    }

    /* Pencere yeniden boyutlanınca sığdır ölçeğini tazele. */
    window.addEventListener('resize', function () {
      if (!d.open || !g.dogalW) return;
      var yeni = tabanHesapla();
      if (Math.abs(yeni - g.taban) > 0.001) {
        g.taban = yeni;
        if (g.olcek <= yeni + 0.001) sigdir(false);
        else olcekle(Math.max(g.olcek, yeni), 0, 0, false);
      }
    });

    g.ac = ac;
    return g;
  }

  /* =====================================================================
     TIKLAMA — tek delege: modül ve galeri aynı yolu kullanır
     ===================================================================== */
  function zoomAc(tetik) {
    var grup = tetik.closest('[data-gm-grup]') || document;
    var ogeler = Array.prototype.slice.call(grup.querySelectorAll('[data-gm-zoom]'));
    if (ogeler.length === 0) ogeler = [tetik];
    var i = ogeler.indexOf(tetik);
    if (i < 0) i = 0;
    gorselOrnek().ac(ogeler, i, tetik);
  }

  document.addEventListener('click', function (e) {
    var el = e.target.closest ? e.target.closest('[data-gm-zoom]') : null;
    if (!el) return;
    /* Kaynağa gitme / yeni sekme yok: tıklama yalnız büyütür. */
    e.preventDefault();
    zoomAc(el);
  });

  /* =====================================================================
     ŞERİT (ana sayfa modülü) — JS olmadan CSS scroll-snap ile çalışır
     ===================================================================== */
  function adetOku(track) {
    var v = parseInt(getComputedStyle(track).getPropertyValue('--gzm-adet'), 10);
    return v > 0 ? v : 1;
  }

  function adim(track, slayt) {
    if (!slayt) return track.clientWidth;
    var bosluk = parseFloat(getComputedStyle(track).columnGap || '0') || 0;
    return slayt.getBoundingClientRect().width + bosluk;
  }

  function seritKur(kok) {
    var track = kok.querySelector('[data-gm-track]');
    if (!track) return;
    var slaytlar = Array.prototype.slice.call(track.querySelectorAll('[data-gm-slide]'));
    if (slaytlar.length === 0) return;

    var onceki = kok.querySelector('[data-gm-prev]');
    var sonraki = kok.querySelector('[data-gm-next]');
    var noktaKabi = kok.querySelector('[data-gm-dots]');
    var aralik = parseInt(kok.getAttribute('data-gm-aralik') || '6', 10);
    var sayfa = 0;
    var zamanlayici = null;
    var duraklat = false;
    var zoomAcik = false;

    kok.classList.add('gzm-js');

    var noktalar = [];
    if (noktaKabi) {
      noktaKabi.textContent = '';
      slaytlar.forEach(function (s, i) {
        var b = document.createElement('button');
        b.type = 'button';
        b.className = 'gzm-dot';
        b.setAttribute('role', 'tab');
        b.setAttribute('aria-label', (i + 1) + '. gazete: ' + (s.getAttribute('data-gm-ad') || ''));
        b.setAttribute('aria-selected', i === 0 ? 'true' : 'false');
        b.addEventListener('click', function () { git(i); sifirla(); });
        noktaKabi.appendChild(b);
        noktalar.push(b);
      });
    }

    function seciliGuncelle() {
      if (noktalar.length === 0) return;
      var n = sinirla(sayfa, 0, noktalar.length - 1);
      noktalar.forEach(function (b, i) {
        b.setAttribute('aria-selected', i === n ? 'true' : 'false');
      });
    }

    function sayfaSayisi() {
      return Math.max(1, Math.ceil(slaytlar.length / adetOku(track)));
    }

    function konumdanSayfa() {
      var adet = adetOku(track);
      var ad = adim(track, slaytlar[0]) || 1;
      return Math.round(Math.round(track.scrollLeft / ad) / adet);
    }

    function git(n) {
      var adet = adetOku(track);
      var toplam = sayfaSayisi();
      if (toplam <= 1) { sayfa = 0; segGuncelle(); return; }
      n = ((n % toplam) + toplam) % toplam;
      sayfa = n;
      track.scrollTo({
        left: Math.round(n * adet * adim(track, slaytlar[0])),
        behavior: azaltHareket() ? 'auto' : 'smooth'
      });
      seciliGuncelle();
      segGuncelle();
    }

    function segGuncelle() {
      var tek = sayfaSayisi() <= 1;
      if (onceki) onceki.disabled = tek;
      if (sonraki) sonraki.disabled = tek;
      if (noktaKabi) noktaKabi.hidden = tek;
    }

    function ileri() { git(sayfa + 1); }
    function geri() { git(sayfa - 1); }

    if (sonraki) sonraki.addEventListener('click', function () { ileri(); sifirla(); });
    if (onceki) onceki.addEventListener('click', function () { geri(); sifirla(); });

    track.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowRight') { e.preventDefault(); ileri(); sifirla(); }
      if (e.key === 'ArrowLeft') { e.preventDefault(); geri(); sifirla(); }
    });

    var kaydirmaZamani = 0;
    track.addEventListener('scroll', function () {
      window.clearTimeout(kaydirmaZamani);
      kaydirmaZamani = window.setTimeout(function () {
        sayfa = konumdanSayfa();
        seciliGuncelle();
      }, 90);
    }, { passive: true });

    function dur() {
      if (zamanlayici) { window.clearInterval(zamanlayici); zamanlayici = null; }
    }

    function basla() {
      dur();
      if (aralik <= 0 || sayfaSayisi() <= 1) return;
      if (azaltHareket() || duraklat || zoomAcik || document.hidden) return;
      zamanlayici = window.setInterval(function () {
        if (duraklat || zoomAcik || document.hidden) return;
        ileri();
      }, Math.max(2, aralik) * 1000);
    }

    function sifirla() { basla(); }

    kok.addEventListener('mouseenter', function () { duraklat = true; basla(); });
    kok.addEventListener('mouseleave', function () { duraklat = false; basla(); });
    kok.addEventListener('focusin', function () { duraklat = true; basla(); });
    kok.addEventListener('focusout', function (e) {
      if (!kok.contains(e.relatedTarget)) { duraklat = false; basla(); }
    });
    document.addEventListener('visibilitychange', function () { basla(); });

    /* Büyütücü açıkken şerit dönmesin (okuyucu görüntüye bakıyor). */
    document.addEventListener('gm:zoom-ac', function () { zoomAcik = true; basla(); });
    document.addEventListener('gm:zoom-kapat', function () { zoomAcik = false; basla(); });

    var boyutZamani = 0;
    function boyutDegisti() {
      window.clearTimeout(boyutZamani);
      boyutZamani = window.setTimeout(function () {
        var toplam = sayfaSayisi();
        sayfa = sinirla(sayfa, 0, toplam - 1);
        track.scrollLeft = Math.round(sayfa * adetOku(track) * adim(track, slaytlar[0]));
        segGuncelle();
        basla();
      }, 120);
    }
    if (window.ResizeObserver) {
      new ResizeObserver(boyutDegisti).observe(track);
    } else {
      window.addEventListener('resize', boyutDegisti);
    }

    segGuncelle();
    basla();
  }

  function baslat() {
    var kokler = document.querySelectorAll('[data-gm]');
    for (var i = 0; i < kokler.length; i++) seritKur(kokler[i]);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', baslat);
  } else {
    baslat();
  }
})();
