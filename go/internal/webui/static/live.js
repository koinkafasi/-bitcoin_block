(function () {
  'use strict';

  var main = document.querySelector('main[data-article-id]');
  if (!main) return;
  var id = Number.parseInt(main.getAttribute('data-article-id') || '0', 10);
  if (!id) return;

  var currentVersion = Number.parseInt(main.getAttribute('data-content-version') || '0', 10) || 0;
  var refreshing = false;
  var es = null;
  var errorCount = 0;
  var pollTimer = null;
  var removedShown = false;

  function toast(text, sticky) {
    var el = document.getElementById('liveToast');
    if (!el) {
      el = document.createElement('div');
      el.id = 'liveToast';
      el.setAttribute('role', 'status');
      el.style.cssText = 'position:fixed;left:50%;bottom:24px;transform:translateX(-50%);' +
        'background:#111;color:#fff;padding:10px 16px;border-radius:10px;z-index:9999;' +
        'font-size:.9rem;box-shadow:0 8px 24px rgba(0,0,0,.25);max-width:90vw;text-align:center';
      document.body.appendChild(el);
    }
    el.textContent = text;
    el.style.display = 'block';
    if (sticky) return;
    clearTimeout(el._liveTimer);
    el._liveTimer = setTimeout(function () { el.style.display = 'none'; }, 5000);
  }

  function showRemoved() {
    if (removedShown) return;
    removedShown = true;
    main.setAttribute('data-content-status', 'removed');
    toast('Bu içerik yayından kaldırıldı.', true);
    if (es) {
      es.close();
      es = null;
    }
  }

  function replaceDocumentTitle(oldTitle, newTitle) {
    if (!newTitle) return;
    if (oldTitle && document.title.indexOf(oldTitle) !== -1) {
      document.title = document.title.replace(oldTitle, newTitle);
    }
  }

  function applySnapshot(data) {
    if (!data || !data.ok) return;
    if (data.status && data.status !== 'yayinda') {
      showRemoved();
      return;
    }
    if (data.version && data.version <= currentVersion) return;

    var titleEl = document.getElementById('article-title');
    var spotEl = document.getElementById('article-spot');
    var bodyEl = document.getElementById('article-body');
    var oldTitle = titleEl ? titleEl.textContent : '';

    if (titleEl && data.title) {
      titleEl.textContent = data.title;
      replaceDocumentTitle(oldTitle, data.title);
    }
    if (spotEl) {
      spotEl.textContent = data.spot || '';
      spotEl.hidden = !data.spot;
    }
    if (bodyEl && !data.locked && typeof data.html === 'string') {
      bodyEl.innerHTML = data.html;
    }
    if (data.cover_url) {
      var cover = document.getElementById('article-cover');
      if (cover) {
        cover.src = data.cover_url;
        if (data.title) cover.alt = data.title;
      }
    }
    if (data.title) {
      var ogTitle = document.querySelector('meta[property="og:title"]');
      if (ogTitle) ogTitle.setAttribute('content', data.title);
    }
    if (data.spot) {
      var desc = document.querySelector('meta[name="description"]');
      if (desc) desc.setAttribute('content', data.spot);
      var ogDesc = document.querySelector('meta[property="og:description"]');
      if (ogDesc) ogDesc.setAttribute('content', data.spot);
    }

    currentVersion = data.version || currentVersion;
    main.setAttribute('data-content-version', String(currentVersion));
    toast('İçerik güncellendi.');
  }

  function refresh() {
    if (refreshing) return;
    refreshing = true;
    fetch('/api/icerik/' + id + '/canli', {
      cache: 'no-store',
      credentials: 'same-origin',
      headers: { 'Accept': 'application/json' }
    }).then(function (res) {
      if (res.status === 404 || res.status === 410) {
        showRemoved();
        return null;
      }
      if (!res.ok) return null;
      return res.json();
    }).then(function (data) {
      if (data) applySnapshot(data);
      refreshing = false;
    }).catch(function () {
      refreshing = false;
    });
  }

  function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(function () {
      if (document.visibilityState === 'visible') refresh();
    }, 30000);
  }

  function connect() {
    if (!window.EventSource) {
      startPolling();
      return;
    }
    try {
      es = new EventSource('/canli/icerik/' + id);
    } catch (_) {
      startPolling();
      return;
    }
    es.addEventListener('snapshot', function (e) {
      errorCount = 0;
      try {
        var data = JSON.parse(e.data);
        if (data.kind === 'removed' || (data.status && data.status !== 'yayinda')) {
          showRemoved();
          return;
        }
        if (data.version && data.version > currentVersion) refresh();
      } catch (_) {  }
    });
    es.addEventListener('content', function (e) {
      errorCount = 0;
      try {
        var data = JSON.parse(e.data);
        if (data.kind === 'removed' || (data.status && data.status !== 'yayinda')) {
          showRemoved();
          return;
        }
        if (!data.version || data.version > currentVersion) refresh();
      } catch (_) {  }
    });
    es.onopen = function () {
      errorCount = 0;
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
    };
    es.onerror = function () {
      errorCount++;
      // EventSource kendi yeniden bağlanır; üst üste hata olursa yedek poll.
      if (errorCount >= 3) startPolling();
    };
  }

  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'visible') refresh();
  });
  window.addEventListener('pageshow', function () { refresh(); });
  window.addEventListener('pagehide', function () {
    if (es) {
      es.close();
      es = null;
    }
  });
  // Test ve gömülü önizlemeler için bağlantıyı elle kapatma.
  window.__liveClose = function () {
    if (es) {
      es.close();
      es = null;
    }
  };

  connect();
})();
