/* csp-delegated.js — CSP uyumlu olay delegasyonu.
 *
 * NEDEN: Zorunlu (enforce) CSP, satır içi olay özniteliklerini
 * (`onsubmit="return confirm(...)"`, `onchange="this.form.submit()"`)
 * ÇALIŞTIRMAZ — 'unsafe-inline' yok. Bu yüzden bu davranışlar şablonlardan
 * kaldırıldı ve burada, tek bir delege dinleyiciyle karşılandı.
 *
 * Desteklenen öznitelikler:
 *   data-confirm="metin"    → window.confirm; false dönerse işlem iptal.
 *                             htmx varsa (hx-post/hx-get) onay sorulmaz,
 *                             çünkü htmx isteği bizim submit'imizle değil
 *                             kendi içinde gönderir.
 *   data-confirm-her-zaman  → htmx olsa da onay sor.
 *   data-autosubmit         → change olayında formu gönder (htmx varsa
 *                             form.submit() çağrılmaz, htmx kendi gönderir).
 *   data-hata-gizle         → error olayında öğeyi kaldır (bozuk görsel).
 *   data-sec                → click/focus'ta içeriği seç.
 *   data-kopyala            → data-copy değerini panoya kopyala, etiketi
 *                             "Kopyalandı" yap, 1,5 sn sonra geri al.
 *   data-set-alan="a=1;b=2" → submit öncesi form alanlarını doldur.
 *
 * Tümü yakalama (capture) fazında dinlenir: htmx kendi işleyicisini
 * bağlamış olsa da sıra bizde kalır.
 */
(function () {
  'use strict';

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
})();
