
(function () {
  'use strict';

  function init() {
    var cfgInput = document.getElementById('header-config');
    if (!cfgInput || cfgInput.dataset.ready === '1') return;
    cfgInput.dataset.ready = '1';

    var cfg;
    try { cfg = JSON.parse(cfgInput.value || '{}'); } catch (e) { cfg = {}; }
    cfg.buttons = Array.isArray(cfg.buttons) ? cfg.buttons : [];

    var q = function (s, root) { return (root || document).querySelector(s); };
    var qa = function (s) { return [].slice.call(document.querySelectorAll(s)); };
    function esc(s) {
      return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
    }
    function buttonRow(b) {
      var row = document.createElement('div');
      row.className = 'header-btn-row';
      row.style.cssText = 'display:flex;gap:6px;align-items:center;margin-bottom:6px;flex-wrap:wrap';
      row.innerHTML = '<input class="input hb-label" placeholder="Buton adı" value="' + esc(b.label) + '" style="flex:1;min-width:120px">' +
        '<input class="input hb-url" placeholder="/sayfa/..." value="' + esc(b.url) + '" style="flex:2;min-width:150px">' +
        '<select class="input hb-style" style="max-width:120px"><option value="primary"' + (b.style === 'primary' ? ' selected' : '') + '>Dolu</option><option value="ghost"' + (b.style === 'ghost' ? ' selected' : '') + '>Çerçeveli</option></select>' +
        '<button type="button" class="btn danger small hb-del">✕</button>';
      return row;
    }
    function renderButtons() {
      var box = q('#hButtons');
      if (!box) return;
      box.innerHTML = '';
      cfg.buttons.forEach(function (b) { box.appendChild(buttonRow(b)); });
    }
    function sync() {
      cfg.model = (q('input[name="headerModel"]:checked') || {}).value || 'model1';
      var skin = q('#hSkin'), width = q('#hWidth');
      cfg.skin = skin ? skin.value : cfg.skin;
      cfg.width = width ? width.value : cfg.width;
      var sticky = q('#hSticky'), topbar = q('#hTopbar');
      cfg.sticky = sticky ? sticky.checked : cfg.sticky;
      cfg.topbar = topbar ? topbar.checked : cfg.topbar;
      var tt = q('#hTopbarText');
      cfg.topbar_text = tt ? tt.value : cfg.topbar_text;
      var tg = q('#hTagline'), hm = q('#hHome'), hs = q('#hSearch'), ht = q('#hTheme'), hmn = q('#hMenu'), hmb = q('#hMember');
      if (tg) cfg.show_tagline = tg.checked;
      if (hm) cfg.show_home = hm.checked;
      if (hs) cfg.show_search = hs.checked;
      if (ht) cfg.show_theme = ht.checked;
      if (hmn) cfg.show_menu = hmn.checked;
      if (hmb) cfg.hide_member = !hmb.checked;
      cfg.buttons = qa('#hButtons .header-btn-row').map(function (r) {
        return { label: q('.hb-label', r).value, url: q('.hb-url', r).value, style: q('.hb-style', r).value };
      });
      var wrap = q('#hTopbarWrap');
      if (wrap) wrap.style.display = cfg.topbar ? '' : 'none';
    }
    var topbar = q('#hTopbar');
    if (topbar) topbar.addEventListener('change', sync);
    var addBtn = q('#hButtonAdd');
    if (addBtn) addBtn.addEventListener('click', function () {
      cfg.buttons.push({ label: '', url: '', style: 'primary' });
      renderButtons();
    });
    document.addEventListener('click', function (e) {
      if (e.target.closest('.hb-del')) {
        sync();
        var r = e.target.closest('.header-btn-row');
        var i = qa('#hButtons .header-btn-row').indexOf(r);
        if (i >= 0) cfg.buttons.splice(i, 1);
        renderButtons();
      }
    });
    var save = q('#hSave');
    if (save) save.addEventListener('click', function () {
      sync();
      var form = q('#headerHtmxForm');
      if (!form) return;
      form.querySelector('[name=config]').value = JSON.stringify(cfg);
      if (window.htmx) {
        window.htmx.trigger(form, 'submit');
        return;
      }
      fetch('/admin/header/kaydet', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ config: cfg }) })
        .then(function (r) { return r.json(); })
        .then(function (j) {
          if (j.ok) location.reload();
          else alert(j.error || 'Hata');
        });
    });
    renderButtons();
  }

  window.__tbHeaderInit = init;
  if (document.readyState !== 'loading') init();
  else document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('htmx:afterSwap', function () {
    if (document.getElementById('header-config')) init();
  });
})();