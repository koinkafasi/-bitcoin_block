/* Kur Çevirici (kcr-).
 *
 * Kurlar <option data-rate> olarak sunucudan gelir (1 birim = X TRY);
 * hesap istemcide yapılır, ağ isteği yok. Sonuç sunucuda da hesaplanmış
 * olarak basılır — JS kapalıysa/gecikirse kutu boş kalmaz.
 *
 * Türkçe sayı biçimi: 1.234,5678 (binlik nokta, ondalık virgül).
 */
(function () {
  'use strict';

  var reduce = false;
  try {
    reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  } catch (e) { /* eski tarayıcı */ }

  function sayiOku(s) {
    s = String(s || '').trim();
    if (!s) return 0;
    // "1.234,56" → 1234.56 ; "1234.56" → 1234.56
    if (s.indexOf(',') !== -1) {
      s = s.replace(/\./g, '').replace(',', '.');
    }
    var v = parseFloat(s.replace(/[^\d.\-]/g, ''));
    return isFinite(v) ? v : 0;
  }

  function bicimle(v, hane) {
    if (!isFinite(v)) return '—';
    hane = hane === undefined ? 2 : hane;
    try {
      return new Intl.NumberFormat('tr-TR', {
        minimumFractionDigits: hane,
        maximumFractionDigits: hane
      }).format(v);
    } catch (e) {
      return v.toFixed(hane).replace('.', ',');
    }
  }

  /* Fiyat ölçeğine göre ondalık: çok küçük değerlerde 4-8 hane göster. */
  function haneSec(v) {
    if (!isFinite(v) || v === 0) return 2;
    var a = Math.abs(v);
    if (a < 0.01) return 8;
    if (a < 0.1) return 6;
    if (a < 1) return 4;
    if (a < 1000) return 2;
    return 2;
  }

  function kur(sel) {
    var o = sel.options[sel.selectedIndex];
    return o ? parseFloat(o.getAttribute('data-rate')) : NaN;
  }

  function hesapla(kok, canli) {
    var amountEl = kok.querySelector('[data-kcr-amount]');
    var fromEl = kok.querySelector('[data-kcr-from]');
    var toEl = kok.querySelector('[data-kcr-to]');
    var out = kok.querySelector('[data-kcr-result]');
    if (!amountEl || !fromEl || !toEl || !out) return;

    var miktar = sayiOku(amountEl.value);
    var f = kur(fromEl), t = kur(toEl);
    if (!isFinite(f) || !isFinite(t) || f <= 0 || t <= 0) {
      out.textContent = '—';
      out.removeAttribute('data-fin-v');
      return;
    }
    var sonuc = miktar * f / t;
    var eski = parseFloat(out.getAttribute('data-fin-v'));
    // Birim adları sunucu üretimi <option> değerleridir; yine de HTML olarak
    // birleştirilmez, metin düğümü olarak yazılır.
    out.textContent = '';
    out.appendChild(document.createTextNode(bicimle(miktar, 2) + ' ' + fromEl.value + ' = '));    var b = document.createElement('b');
    b.textContent = bicimle(sonuc, haneSec(sonuc)) + ' ' + toEl.value;
    out.appendChild(b);
    out.setAttribute('data-fin-v', String(sonuc));

    // Canlı kur güncellemesinde sonuç değiştiyse kısa vurgu; kullanıcı
    // yazarken/seçerken vurgu yok (sadece fin:kur olayında).
    if (canli && isFinite(eski) && Math.abs(eski - sonuc) > 1e-12) {
      var sinif = reduce
        ? (sonuc >= eski ? 'fin-tint-up' : 'fin-tint-down')
        : (sonuc >= eski ? 'fin-flash-up' : 'fin-flash-down');
      out.classList.remove('fin-flash-up', 'fin-flash-down', 'fin-tint-up', 'fin-tint-down');
      void out.offsetWidth;
      out.classList.add(sinif);
      window.setTimeout(function () { out.classList.remove(sinif); }, 1200);
    }
  }

  function basla() {
    var list = document.querySelectorAll('[data-kcr]');
    for (var i = 0; i < list.length; i++) {
      (function (kok) {
        var amount = kok.querySelector('[data-kcr-amount]');
        var from = kok.querySelector('[data-kcr-from]');
        var to = kok.querySelector('[data-kcr-to]');
        var go = kok.querySelector('[data-kcr-go]');
        var swap = kok.querySelector('[data-kcr-swap]');
        if (!amount || !from || !to) return;

        // Canlı hesap: yazarken/seçerken güncelle (DÖNÜŞTÜR yine çalışır).
        amount.addEventListener('input', function () { hesapla(kok); });
        from.addEventListener('change', function () { hesapla(kok); });
        to.addEventListener('change', function () { hesapla(kok); });
        if (go) go.addEventListener('click', function () { hesapla(kok); });
        if (swap) swap.addEventListener('click', function () {
          var a = from.value;
          from.value = to.value;
          to.value = a;
          hesapla(kok);
        });
        amount.addEventListener('keydown', function (e) {
          if (e.key === 'Enter') { e.preventDefault(); hesapla(kok); }
        });

        // data-fin-v tabanını kur: sunucu sonucu zaten basılı olduğu için
        // ilk canlı güncellemede "eski değer" bilinmeli; yoksa vurgu atlanır.
        hesapla(kok);
      })(list[i]);
    }
  }

  // Canlı finans akışı kurları tazeledikten sonra sonucu yeniden hesapla.
  document.addEventListener('fin:kur', function () {
    var list = document.querySelectorAll('[data-kcr]');
    for (var i = 0; i < list.length; i++) hesapla(list[i], true);
  });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', basla);
  } else {
    basla();
  }
})();
