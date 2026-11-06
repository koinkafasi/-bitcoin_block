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
      dugmeler[i].setAttribute('aria-label',
        koyu ? 'Açık temaya geç' : 'Koyu temaya geç');
    }
  }

  document.addEventListener('click', function (e) {
    var el = e.target;
    if (!el || !el.closest) return;
    var btn = el.closest('[data-tema]');
    if (!btn) return;
    var kok = document.documentElement;
    var koyu = !kok.classList.contains('dark');
    kok.classList.remove('light', 'dark');
    kok.classList.add(koyu ? 'dark' : 'light');
    try { localStorage.setItem('tema', koyu ? 'dark' : 'light'); } catch (err) { /* yok say */ }
    temaDugmeleri();
  }, true);

  temaDugmeleri();

  // --- cd-kaydi: fiyat şeridinin kaydırma durumu ---
  //
  // Kullanıcı isteği: sayfa başında şerit SADE (iki satır, yüzde gizli);
  // sticky menü aşağı inince GENİŞ görünüm (tek satır + yüzde). Eşik
  // 80 px: menü ~73 px, şerit 67 px — ilk ekranın üstünde kalıyor.
  // Sınıf <html> üzerinde: CSS `html.cd-kaydi .cd-ticker__item` ile
  // bütün şeridi tek kuralla çeviriyor.
  function kaydiGuncelle() {
    var kaydi = (window.scrollY || window.pageYOffset || 0) > 80;
    document.documentElement.classList.toggle('cd-kaydi', kaydi);
  }
  kaydiGuncelle();
  window.addEventListener('scroll', kaydiGuncelle, { passive: true });
})();
