/* menu.js — Menü Yapısı paneli: sürükle-bırak sıralama, kategori/özel link
 * ekleme, ana/alt geçişi, JSON kaydetme. HTMX swap sonrası yeniden bağlanır. */
(function () {
  'use strict';

  function init() {
    var list = document.getElementById('menuItems');
    if (!list || list.dataset.ready === '1') return;
    list.dataset.ready = '1';
    var dragItem = null;

    function counts() {
      var main = 0, sub = 0, total = 0;
      list.querySelectorAll('li[data-slug],li[data-url]').forEach(function (li) {
        total++;
        if ((li.getAttribute('data-sub') || '0') === '1') { sub++; } else { main++; }
      });
      var c = document.getElementById('menuCount'),
        m = document.getElementById('menuMainCount'),
        s = document.getElementById('menuSubCount');
      if (c) c.textContent = total;
      if (m) m.textContent = main;
      if (s) s.textContent = sub;
    }

    function makeLi(slug, url, name, label) {
      var li = document.createElement('li');
      li.setAttribute('data-slug', slug || '');
      li.setAttribute('data-url', url || '');
      li.setAttribute('data-sub', '0');
      li.setAttribute('draggable', 'true');
      li.style.cssText = 'display:flex;align-items:center;gap:8px;padding:8px 10px;border:1px solid var(--border);border-radius:10px;margin-bottom:8px;background:var(--surface,#fff)';
      li.innerHTML = '<span class="m-drag" style="cursor:grab;color:#a1a1aa;font-size:18px;user-select:none" title="Sürükle"><svg class="ti" aria-hidden="true"><use href="/static/vendor/tabler-icons.svg?v=b89545a6#ti-grip-vertical"></use></svg></span>' +
        '<button type="button" class="btn ghost small m-up" title="Yukarı">↑</button>' +
        '<button type="button" class="btn ghost small m-down" title="Aşağı">↓</button>' +
        (url ? '<span class="badge" style="margin-right:2px;flex:0 0 auto">🔗</span>' : '') +
        '<input class="input m-label" value="' + (label || '') + '" placeholder="' + (name || 'Kategori') + '" style="flex:1;min-width:120px" title="Menüde görünecek ad">' +
        '<span class="shape small" style="' + (url ? 'border-color:var(--warn);color:var(--warn)' : 'color:var(--muted-fg)') + '">' + (url ? 'Özel Link' : '/' + slug) + '</span>' +
        '<button type="button" class="btn ghost small m-sub" title="Ana / Alt geçişi">Ana menü</button>' +
        '<a class="btn danger small m-del" href="#" title="Kaldır">✕</a>';
      return li;
    }

    list.addEventListener('click', function (e) {
      var li = e.target.closest('li');
      if (!li) return;
      if (e.target.closest('.m-up')) {
        e.preventDefault();
        var p = li.previousElementSibling;
        if (p) list.insertBefore(li, p);
        counts();
      } else if (e.target.closest('.m-down')) {
        e.preventDefault();
        var nx = li.nextElementSibling;
        if (nx) list.insertBefore(nx, li);
        counts();
      } else if (e.target.closest('.m-del')) {
        e.preventDefault();
        li.parentNode.removeChild(li);
        counts();
      } else if (e.target.closest('.m-sub')) {
        e.preventDefault();
        var sub = (li.getAttribute('data-sub') || '0') === '1';
        li.setAttribute('data-sub', sub ? '0' : '1');
        var btn = e.target.closest('.m-sub');
        btn.textContent = sub ? 'Ana menü' : 'Alt menü (Diğer)';
        btn.style.background = sub ? '' : 'var(--tema,transparent)';
        btn.style.color = sub ? '' : '#fff';
        counts();
      }
    });

    list.addEventListener('dragstart', function (e) {
      dragItem = e.target.closest('li');
      if (dragItem) e.dataTransfer.effectAllowed = 'move';
    });
    list.addEventListener('dragover', function (e) {
      e.preventDefault();
      var t = e.target.closest('li');
      if (dragItem && t && t !== dragItem) {
        var r = dragItem.compareDocumentPosition(t) & Node.DOCUMENT_POSITION_PRECEDING ? 'afterend' : 'beforeend';
        list.insertBefore(dragItem, r === 'afterend' ? t.nextSibling : t);
      }
    });
    list.addEventListener('dragend', function () {
      dragItem = null;
      counts();
    });

    var addCats = document.getElementById('menuAddCats');
    if (addCats) addCats.addEventListener('click', function () {
      Array.prototype.slice.call(document.querySelectorAll('.add-cat:checked')).forEach(function (chk) {
        var label = chk.closest('[data-add]');
        if (!label) return;
        var slug = label.getAttribute('data-add'),
          name = label.getAttribute('data-name');
        if (list.querySelector('li[data-slug="' + slug + '"]')) { chk.checked = false; return; }
        list.appendChild(makeLi(slug, '', name, ''));
        chk.checked = false;
      });
      counts();
    });

    var addLink = document.getElementById('menuAddLink');
    if (addLink) addLink.addEventListener('click', function () {
      var n = document.getElementById('linkName'),
        u = document.getElementById('linkUrl');
      var name = (n.value || '').trim(),
        url = (u.value || '').trim();
      if (!name || !url) { alert('Ad ve URL girin.'); return; }
      list.appendChild(makeLi('', url, name, name));
      n.value = '';
      u.value = '';
      counts();
    });

    var save = document.getElementById('menuSave');
    if (save) save.addEventListener('click', function () {
      var items = [];
      list.querySelectorAll('li[data-slug],li[data-url]').forEach(function (li) {
        var slug = li.getAttribute('data-slug') || '';
        var url = li.getAttribute('data-url') || '';
        var lab = li.querySelector('.m-label');
        items.push({ slug: slug, url: url, label: (lab ? lab.value.trim() : ''), sub: (li.getAttribute('data-sub') || '0') === '1' });
      });
      var st = document.getElementById('menuStatus');
      if (window.htmx) {
        var hidden = document.querySelector('#menuHtmxForm input[name=items]');
        if (hidden) {
          hidden.value = JSON.stringify({ items: items });
          window.htmx.trigger(document.getElementById('menuHtmxForm'), 'submit');
          return;
        }
      }
      function msg(t, k) {
        if (st) {
          // Sunucu iletisi HTML olarak yorumlanmaz.
          var d = document.createElement('div');
          d.className = 'flash ' + (k === 'success' ? 'success' : 'error');
          d.style.margin = '8px 0 0';
          d.textContent = t;
          st.textContent = '';
          st.appendChild(d);
          setTimeout(function () { st.textContent = ''; }, 4000);
        }
      }
      fetch('/admin/menu/kaydet', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ items: items }) })
        .then(function (r) { return r.json(); })
        .then(function (d) {
          if (d.ok) {
            msg('Menü kaydedildi (' + d.adet + ' öğe).', 'success');
            setTimeout(function () { window.location.reload(); }, 900);
          } else {
            msg('Hata: ' + (d.error || 'bilinmeyen'), 'error');
          }
        })
        .catch(function (e) { msg('Ağ hatası: ' + e, 'error'); });
    });

    var reset = document.getElementById('menuReset');
    if (reset) reset.addEventListener('click', function () {
      if (window.htmx) return; // hx-post + hx-confirm
      if (!confirm('Menüyü varsayılana (tüm kategoriler) sıfırlasın mı?')) return;
      fetch('/admin/menu/sifirla', { method: 'POST' }).then(function () { window.location.reload(); });
    });

    counts();
  }

  window.__tbMenuInit = init;
  if (document.readyState !== 'loading') init();
  else document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('htmx:afterSwap', function () {
    if (document.getElementById('menuItems')) init();
  });
})();