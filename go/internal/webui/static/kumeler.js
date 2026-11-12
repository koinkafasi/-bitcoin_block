
(function () {
  'use strict';
  function init() {
    var add = document.getElementById('kume-ekle');
    if (!add || add.dataset.ready === '1') return;
    add.dataset.ready = '1';
    add.addEventListener('click', function () {
      var sel = document.getElementById('kume-pool');
      var cid = sel.value;
      if (!cid || cid.indexOf('-') === 0) return;
      var url = '/admin/kumeler/' + add.getAttribute('data-kus') + '/icerik/' + cid;
      if (window.htmx) {
        window.htmx.ajax('POST', url, { target: '#kumeler-panel', swap: 'outerHTML', indicator: '#kumeler-yukleniyor' });
        return;
      }
      var fake = document.getElementById('kume-ekle-fake');
      if (fake) { fake.setAttribute('action', url); fake.submit(); }
    });
  }
  if (document.readyState !== 'loading') init();
  else document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('htmx:afterSwap', function () {
    if (document.getElementById('kume-ekle')) init();
  });
})();