
(function () {
  'use strict';

  function init() {
    var dataInput = document.getElementById('reports-chart-data');
    if (!dataInput || dataInput.dataset.ready === '1') return;
    dataInput.dataset.ready = '1';

    var el = document.getElementById('mostReadChart');
    if (el && typeof window.Chart !== 'undefined') {
      var data;
      try { data = JSON.parse(dataInput.value || '[]'); } catch (e) { data = []; }
      if (data && data.length) {
        var labels = data.map(function (d) { return d.label.length > 40 ? d.label.slice(0, 40) + '…' : d.label; });
        var values = data.map(function (d) { return d.value; });
        new window.Chart(el, {
          type: 'bar',
          data: { labels: labels, datasets: [{ label: 'Okunma', data: values, backgroundColor: 'rgba(225,29,72,.75)', borderRadius: 6 }] },
          options: { plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true } } }
        });
      }
    }

    var pvEl = document.getElementById('rc-pv'),
      rpmEl = document.getElementById('rc-rpm'),
      sitesEl = document.getElementById('rc-sites'),
      outEl = document.getElementById('rc-out');
    if (pvEl && outEl) {
      function calc() {
        var pv = parseFloat(pvEl.value) || 0,
          rpm = parseFloat(rpmEl.value) || 0,
          n = Number.parseInt(sitesEl.value, 10) || 1;
        var month = pv * rpm / 1000 * n;
        outEl.textContent = 'Aylık: $' + Math.round(month).toLocaleString('en-US') +
          ' · Yıllık: $' + Math.round(month * 12).toLocaleString('en-US');
      }
      [pvEl, rpmEl, sitesEl].forEach(function (e) { e.addEventListener('input', calc); });
      calc();
    }
  }

  window.__tbRaporlarInit = init;
  if (document.readyState !== 'loading') init();
  else document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('htmx:afterSwap', function () {
    if (document.getElementById('reports-chart-data')) init();
  });
})();