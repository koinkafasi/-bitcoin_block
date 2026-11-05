/* footer.js — Footer Yönetimi paneli: şerit/sütun link satırı düzenleme,
 * sıralama, JSON kaydetme. HTMX swap sonrası yeniden bağlanır. */
(function () {
  'use strict';

  function init() {
    var root = document.getElementById('footer-panel');
    if (!root || root.dataset.ready === '1') return;
    root.dataset.ready = '1';

    function q(s, r) { return (r || document).querySelector(s); }
    function qa(s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); }
    function esc(s) {
      return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
    }
    function rowHTML(slug, label, url) {
      var r = document.createElement('li');
      r.className = 'f-row';
      if (slug) r.setAttribute('data-slug', slug);
      r.style.cssText = 'display:flex;gap:6px;align-items:center;margin-bottom:6px';
      r.innerHTML = '<button type="button" class="btn ghost small r-up">↑</button>' +
        '<button type="button" class="btn ghost small r-down">↓</button>' +
        '<input class="input r-label" value="' + esc(label) + '" placeholder="Etiket" style="flex:1;min-width:100px">' +
        '<input class="input r-url" value="' + esc(url) + '" placeholder="/sayfa/..." style="flex:2;min-width:140px">' +
        '<button type="button" class="btn danger small r-del">✕</button>';
      return r;
    }
    var stripAdd = q('#stripAdd');
    if (stripAdd) stripAdd.addEventListener('click', function () {
      q('#stripItems').appendChild(rowHTML('', '', ''));
    });
    var colAdd = q('#colAdd');
    if (colAdd) colAdd.addEventListener('click', function () {
      var div = document.createElement('div');
      div.className = 'fc card card-pad';
      div.style.cssText = 'margin-bottom:12px';
      div.innerHTML = '<div style="display:flex;gap:6px;align-items:center;margin-bottom:8px">' +
        '<button type="button" class="btn ghost small col-up">↑</button>' +
        '<button type="button" class="btn ghost small col-down">↓</button>' +
        '<input class="input fc-title" placeholder="Sütun Başlığı" style="flex:1">' +
        '<button type="button" class="btn danger small col-del">Sütunu Sil</button></div>' +
        '<ul class="fc-links" style="list-style:none;margin:0;padding:0"></ul>' +
        '<div style="display:flex;gap:6px;align-items:center;margin-top:8px;flex-wrap:wrap">' +
        '<input class="input al-label" placeholder="Yeni link etiketi" style="flex:1;min-width:110px">' +
        '<input class="input al-url" placeholder="URL (/sayfa/...)" style="flex:1;min-width:140px">' +
        '<button type="button" class="btn ghost small al-add">Link Ekle</button></div>';
      q('#footCols').appendChild(div);
    });
    document.addEventListener('click', function (e) {
      var r = e.target.closest('.r-up, .r-down, .r-del');
      if (r) {
        var li = r.closest('.f-row');
        if (!li) return;
        if (r.classList.contains('r-del')) { li.parentNode.removeChild(li); return; }
        if (r.classList.contains('r-up')) { var p = li.previousElementSibling; if (p) li.parentNode.insertBefore(li, p); }
        if (r.classList.contains('r-down')) { var n = li.nextElementSibling; if (n) li.parentNode.insertBefore(n, li); }
      }
      var al = e.target.closest('.al-add');
      if (al) {
        var fc = al.closest('.fc');
        var lab = q('.al-label', fc), url = q('.al-url', fc);
        q('.fc-links', fc).appendChild(rowHTML('', lab.value, url.value));
        lab.value = '';
        url.value = '';
      }
      var sel = e.target.closest('.add-cat');
      if (sel && sel.value) {
        var fc2 = sel.closest('.fc');
        q('.fc-links', fc2).appendChild(rowHTML(sel.value, sel.options[sel.selectedIndex].text, ''));
        sel.value = '';
      }
      var cu = e.target.closest('.col-up, .col-down, .col-del');
      if (cu) {
        var fc3 = cu.closest('.fc');
        if (!fc3) return;
        if (cu.classList.contains('col-del')) { fc3.parentNode.removeChild(fc3); return; }
        if (cu.classList.contains('col-up')) { var pp = fc3.previousElementSibling; if (pp) fc3.parentNode.insertBefore(fc3, pp); }
        if (cu.classList.contains('col-down')) { var nn = fc3.nextElementSibling; if (nn) fc3.parentNode.insertBefore(nn, fc3); }
      }
    });
    var save = q('#fSave');
    if (save) save.addEventListener('click', function () {
      var logo = 'right';
      qa('input[name="footLogo"]').forEach(function (x) { if (x.checked) logo = x.value; });
      var model = q('#footModel').value;
      var strip = qa('#stripItems .f-row').map(function (li) {
        return { label: q('.r-label', li).value, url: q('.r-url', li).value, slug: li.getAttribute('data-slug') || '' };
      });
      var cols = qa('#footCols .fc').map(function (fc) {
        return {
          title: q('.fc-title', fc).value,
          links: qa('.fc-links .f-row', fc).map(function (li) {
            return { label: q('.r-label', li).value, url: q('.r-url', li).value, slug: li.getAttribute('data-slug') || '' };
          })
        };
      });
      var cfg = {
        logo: logo, model: model,
        skin: (q('#footSkin') ? q('#footSkin').value : 'auto'),
        tagline: (q('#footTagline') ? q('#footTagline').value : ''),
        bottom_text: (q('#footBottomText') ? q('#footBottomText').value : ''),
        show_social: !!(q('#footShowSocial') && q('#footShowSocial').checked),
        show_brand: !!(q('#footShowBrand') && q('#footShowBrand').checked),
        show_tagline: !!(q('#footShowTagline') && q('#footShowTagline').checked),
        show_bottom: !!(q('#footShowBottom') && q('#footShowBottom').checked),
        configured: true,
        strip: strip,
        cols: cols
      };
      if (window.htmx) {
        var hidden = q('#footerHtmxForm input[name=config]');
        if (hidden) {
          hidden.value = JSON.stringify(cfg);
          window.htmx.trigger(document.getElementById('footerHtmxForm'), 'submit');
          return;
        }
      }
      var st = q('#footerStatus');
      function msg(t, k) {
        // Sunucudan gelen metin (hata iletisi) HTML olarak yorumlanmamalı.
        if (st) st.innerHTML = '<div class="flash ' + (k === 'success' ? 'success' : 'error') +
          '" style="margin:8px 0 0">' + esc(t) + '</div>';
      }
      fetch('/admin/footer/kaydet', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ config: cfg }) })
        .then(function (x) { return x.json(); })
        .then(function (d) {
          if (d.ok) {
            msg('Footer kaydedildi.', 'success');
            setTimeout(function () { window.location.reload(); }, 700);
          } else {
            msg('Hata: ' + (d.error || ''), 'error');
          }
        })
        .catch(function () { msg('Ağ hatası', 'error'); });
    });
  }

  window.__tbFooterInit = init;
  if (document.readyState !== 'loading') init();
  else document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('htmx:afterSwap', function () {
    if (document.getElementById('footer-panel')) init();
  });
})();