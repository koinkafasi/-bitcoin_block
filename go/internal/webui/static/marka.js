/* marka.js — Marka Merkezi etkileşimleri: logo'dan renk çıkarma + canlı önizleme.
 * HTMX swap sonrası otomatik yeniden bağlanır. */
(function () {
  'use strict';

  function init() {
    var input = document.getElementById('logoInput');
    if (!input || input.dataset.ready === '1') return;
    input.dataset.ready = '1';

    input.addEventListener('change', function () {
      var f = input.files && input.files[0];
      if (!f) return;
      var url = URL.createObjectURL(f);
      var prev = document.getElementById('logoPrev');
      if (prev) {
        // blob: adresi gerçi tarayıcı üretimi ama DOM ile kurmak daha güvenli.
        prev.textContent = '';
        var onizleme = document.createElement('img');
        onizleme.src = url;
        onizleme.alt = 'logo';
        prev.appendChild(onizleme);
      }
      var img = new Image();
      img.onload = function () {
        try {
          var c = document.createElement('canvas');
          c.width = 32; c.height = 32;
          var ctx = c.getContext('2d');
          ctx.drawImage(img, 0, 0, 32, 32);
          var d = ctx.getImageData(0, 0, 32, 32).data;
          var r = 0, g = 0, b = 0, n = 0;
          for (var i = 0; i < d.length; i += 4) {
            if (d[i + 3] < 128) continue;
            if (d[i] > 245 && d[i + 1] > 245 && d[i + 2] > 245) continue;
            r += d[i]; g += d[i + 1]; b += d[i + 2]; n++;
          }
          if (n > 0) {
            var hex = '#' + [r / n, g / n, b / n].map(function (v) {
              return ('0' + Math.round(v).toString(16)).slice(-2);
            }).join('');
            var pi = document.getElementById('primaryInput');
            if (pi) { pi.value = hex; pi.dispatchEvent(new Event('input')); }
          }
        } catch (e) { /* canvas güvenlik hatası — sessiz */ }
        URL.revokeObjectURL(url);
      };
      img.src = url;
    });

    var form = document.getElementById('gorunumForm');
    if (form) {
      form.addEventListener('input', function () {
        var pv = document.getElementById('gorunumPreview');
        if (!pv) return;
        var p = form.primary.value, s = form.secondary.value, t = form.text.value, f = form.font.value;
        pv.style.setProperty('--accent', p);
        pv.style.setProperty('--accent-2', p);
        pv.style.setProperty('--brand', s);
        pv.style.setProperty('--text', t);
        pv.style.fontFamily = f ? ("'" + f + "', sans-serif") : '';
      });
      var fontSel = form.font;
      if (fontSel) fontSel.addEventListener('change', function () {
        var f = fontSel.value;
        if (!f) return;
        var l = document.createElement('link');
        l.rel = 'stylesheet';
        l.href = 'https://fonts.googleapis.com/css2?family=' + encodeURIComponent(f) + ':wght@400;600;700&display=swap';
        document.head.appendChild(l);
      });
    }
  }

  window.__tbMarkaInit = init;
  if (document.readyState !== 'loading') init();
  else document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('htmx:afterSwap', function () {
    if (document.getElementById('logoInput') || document.getElementById('gorunumForm')) init();
  });
})();