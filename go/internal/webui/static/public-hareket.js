/* ===========================================================================
   Public yüzey hareket katmanı + üst manşet şeridi denetleyicisi.
   - GSAP yereldir (static/vendor/gsap.min.js), CDN yok, eval yok: CSP temiz.
   - JS çalışmazsa veya GSAP yüklenmezse içerik olduğu gibi görünür kalır;
     gizleme yalnız <html class="hareket-hazir"> işaretlendikten sonra geçerli.
   - prefers-reduced-motion'a uyar: hareket tamamen kapanır, işlev kalır.
   =========================================================================== */
(function () {
  'use strict';

  var azHareket = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  var gsap = window.gsap || null;

  /* ---- 1) Üst manşet şeridi ---------------------------------------------
     Şerit overflow-x:auto ama scrollbar gizli ve göstergesi yoktu. Oklar,
     ilerleme çubuğu ve durum sınıfları gizli kartları keşfedilebilir yapar. */
  function seritKur(sarmal) {
    if (sarmal.dataset.seritKuruldu === '1') return;
    var serit = sarmal.querySelector('.ust-manset');
    if (!serit) return;
    sarmal.dataset.seritKuruldu = '1';

    var onceki = sarmal.querySelector('.um-ok-onceki');
    var sonraki = sarmal.querySelector('.um-ok-sonraki');
    var dolgu = sarmal.querySelector('.um-ilerleme-dolgu');

    function kaydirmaAdimi() {
      var kart = serit.querySelector('.um-card');
      if (!kart) return serit.clientWidth * 0.8;
      var bosluk = parseFloat(getComputedStyle(serit).columnGap || getComputedStyle(serit).gap || '16') || 16;
      // Görünen kart sayısı kadar kaydır; en az bir kart.
      var kartW = kart.getBoundingClientRect().width + bosluk;
      var adet = Math.max(1, Math.floor(serit.clientWidth / kartW));
      return kartW * adet;
    }

    function durumuTazele() {
      var kaydirilabilir = serit.scrollWidth - serit.clientWidth;
      var tam = kaydirilabilir <= 2;
      sarmal.classList.toggle('um-tam', tam);

      if (onceki) onceki.hidden = tam || serit.scrollLeft <= 2;
      if (sonraki) sonraki.hidden = tam || serit.scrollLeft >= kaydirilabilir - 2;

      if (dolgu && !tam) {
        var oran = serit.clientWidth / serit.scrollWidth;
        var ilerleme = serit.scrollLeft / kaydirilabilir;
        dolgu.style.width = Math.max(12, oran * 100) + '%';
        dolgu.style.transform = 'translateX(' + (ilerleme * (100 / Math.max(oran, 0.0001) - 100)) + '%)';
      }
    }

    function kaydir(yon) {
      serit.scrollBy({ left: yon * kaydirmaAdimi(), behavior: azHareket ? 'auto' : 'smooth' });
    }

    if (onceki) onceki.addEventListener('click', function () { kaydir(-1); });
    if (sonraki) sonraki.addEventListener('click', function () { kaydir(1); });

    // Klavye: şerit odaklanabilir olmalı ki ok tuşları çalışsın.
    if (!serit.hasAttribute('tabindex')) serit.setAttribute('tabindex', '0');
    if (!serit.hasAttribute('role')) serit.setAttribute('role', 'region');
    if (!serit.hasAttribute('aria-label')) serit.setAttribute('aria-label', 'Üst manşet kartları');
    serit.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowRight') { e.preventDefault(); kaydir(1); }
      else if (e.key === 'ArrowLeft') { e.preventDefault(); kaydir(-1); }
    });

    serit.addEventListener('scroll', function () {
      if (serit._tik) return;
      serit._tik = requestAnimationFrame(function () { serit._tik = 0; durumuTazele(); });
    }, { passive: true });

    if (window.ResizeObserver) new ResizeObserver(durumuTazele).observe(serit);
    window.addEventListener('load', durumuTazele);
    durumuTazele();
  }

  /* ---- 2) Giriş animasyonları --------------------------------------------
     Haber sitesi: abartısız. 14px yükselme + opaklık, kademeli.
     ScrollTrigger yerine IntersectionObserver — ek eklenti yüklemeye gerek yok. */
  var GIRIS_SECICI = [
    '.ust-manset > .um-card',
    '.kari-izgara > *',
    '.galeri-blok',
    '.ec-item',
    '.st-item'
  ].join(',');

  function girisKur(kok) {
    if (azHareket || !gsap) return;

    var hedefler = [].slice.call((kok || document).querySelectorAll(GIRIS_SECICI))
      .filter(function (el) { return !el.dataset.hrktKuruldu; });
    if (!hedefler.length) return;

    hedefler.forEach(function (el) {
      el.dataset.hrktKuruldu = '1';
      el.classList.add('hrkt-gir');
    });

    var gozcu = new IntersectionObserver(function (girdiler) {
      var gorunenler = girdiler.filter(function (g) { return g.isIntersecting; });
      if (!gorunenler.length) return;

      gorunenler.forEach(function (g) { gozcu.unobserve(g.target); });

      gsap.to(gorunenler.map(function (g) { return g.target; }), {
        opacity: 1,
        y: 0,
        duration: 0.5,
        ease: 'power2.out',
        stagger: 0.055,
        clearProps: 'transform,willChange'
      });
    }, { rootMargin: '0px 0px -8% 0px', threshold: 0.05 });

    gsap.set(hedefler, { y: 14 });
    hedefler.forEach(function (el) { gozcu.observe(el); });
  }

  /* ---- 3) Hero başlığı ----------------------------------------------------
     Sayfanın odak noktası; yüklenince tek seferlik, kısa bir yükselme. */
  function heroKur() {
    if (azHareket || !gsap) return;
    var baslik = document.querySelector('.m6-title-ana');
    if (!baslik || baslik.dataset.hrktKuruldu) return;
    baslik.dataset.hrktKuruldu = '1';
    gsap.from(baslik, { opacity: 0, y: 18, duration: 0.6, ease: 'power2.out', clearProps: 'all' });
  }

  function kur(kok) {
    [].slice.call((kok || document).querySelectorAll('.um-sarmal')).forEach(seritKur);
    girisKur(kok);
  }

  function basla() {
    if (gsap && !azHareket) document.documentElement.classList.add('hareket-hazir');
    kur(document);
    heroKur();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', basla);
  } else {
    basla();
  }

  /* htmx sayfanın bir parçasını değiştirdiğinde yeni gelen düğümler için
     denetleyici ve animasyonlar yeniden bağlanmalı. */
  document.body.addEventListener('htmx:afterSwap', function (e) {
    kur(e.target || document);
  });
})();
