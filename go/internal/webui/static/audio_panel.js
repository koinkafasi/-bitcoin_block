
(function () {
  "use strict";
  var pollTimer = null;
  var pollGen = 0;

  function panelBody() {
    return document.getElementById("sesPanelBody");
  }

  function stopPolling() {
    pollGen++;
    if (pollTimer) {
      clearTimeout(pollTimer);
      pollTimer = null;
    }
  }

  function startPolling(url) {
    if (!url) return;
    stopPolling();
    var gen = pollGen;
    function tick() {
      fetch(url, { credentials: "same-origin", headers: { "HX-Request": "true" } })
        .then(function (r) { return r.text(); })
        .then(function (html) {
          if (gen !== pollGen) return;
          var host = panelBody();
          if (!host) return;
          host.innerHTML = html;
          bindPolling();
        })
        .catch(function () {
          if (gen !== pollGen) return;
          pollTimer = setTimeout(tick, 5000);
        });
    }
    pollTimer = setTimeout(tick, 1200);
  }

  function bindPolling() {
    var host = panelBody();
    if (!host) {
      stopPolling();
      return;
    }
    var marker = host.querySelector("[data-audio-poll-url]");
    if (marker) startPolling(marker.getAttribute("data-audio-poll-url"));
    else stopPolling();
    bindUpload(host);
  }

  
  function fmtBytes(n) {
    if (!isFinite(n) || n <= 0) return "0 B";
    var units = ["B", "KB", "MB", "GB"];
    var i = 0;
    while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
    return (i === 0 ? n.toFixed(0) : n.toFixed(1)) + " " + units[i];
  }

  function fmtSeconds(s) {
    if (!isFinite(s) || s < 0) return "";
    if (s < 60) return Math.round(s) + " sn";
    var m = Math.floor(s / 60);
    var r = Math.round(s % 60);
    return m + " dk " + r + " sn";
  }

  function bindUpload(host) {
    var form = host.querySelector("form.ses-upload");
    if (!form || form.dataset.uploadBound === "1") return;
    form.dataset.uploadBound = "1";

    form.addEventListener("submit", function (ev) {
      // HTMX'in kendi gönderimini durdur; XHR ile biz gönderiyoruz.
      ev.preventDefault();
      ev.stopPropagation();

      var fileInput = form.querySelector('input[type="file"][name="ses"]');
      var file = fileInput && fileInput.files && fileInput.files[0];
      if (!file) {
        showUploadState(host, "error", "Ses dosyası seçilmedi.");
        return;
      }
      var consent = form.querySelector('input[name="onay"]');
      if (consent && !consent.checked) {
        showUploadState(host, "error", "Kayıt için kişilerin onayını teyit etmelisiniz.");
        return;
      }

      var maxBytes = 500 * 1024 * 1024;
      if (file.size > maxBytes) {
        showUploadState(host, "error",
          "Dosya çok büyük (" + fmtBytes(file.size) + "). En fazla 500 MB yükleyebilirsiniz.");
        return;
      }

      var xhr = new XMLHttpRequest();
      var action = form.getAttribute("action") || "/admin/icerikler";
      var started = Date.now();
      var lastTick = 0;

      xhr.open("POST", action, true);
      xhr.setRequestHeader("HX-Request", "true");
      xhr.responseType = "text";

      xhr.upload.onprogress = function (e) {
        if (!e.lengthComputable) return;
        var pct = Math.min(100, Math.round((e.loaded / e.total) * 100));
        var now = Date.now();
        // Her %1'de veya 250 ms'de bir güncelle (gereksiz DOM yazımını önle).
        if (pct === lastTick && now - (xhr._lastAt || 0) < 250) return;
        lastTick = pct;
        xhr._lastAt = now;
        var elapsed = (now - started) / 1000;
        var speed = elapsed > 0.4 ? e.loaded / elapsed : 0;
        var eta = speed > 0 ? (e.total - e.loaded) / speed : NaN;
        setUploadProgress(host, pct, fmtBytes(e.loaded) + " / " + fmtBytes(e.total),
          speed > 0 ? fmtBytes(speed) + "/sn" : "", eta);
      };

      xhr.upload.onload = function () {
        // Baytlar gönderildi; sunucu tarafı (kayıt + kuyruk) sürüyor.
        setUploadPhase(host, "processing", "Yükleme tamamlandı, sunucu işliyor…");
      };

      xhr.onload = function () {
        var ok = xhr.status >= 200 && xhr.status < 400;
        if (ok && xhr.responseText) {
          // Sunucunun döndürdüğü panel gövdesini yerleştir.
          host.innerHTML = xhr.responseText;
          bindPolling();
          return;
        }
        var msg = "Yükleme başarısız";
        if (xhr.status === 413) msg = "Dosya sunucu sınırını aştı (413).";
        else if (xhr.status === 0) msg = "Bağlantı kesildi.";
        else if (xhr.status) msg = msg + " (HTTP " + xhr.status + ").";
        showUploadState(host, "error", msg + " Lütfen tekrar deneyin.");
      };

      xhr.onerror = function () {
        showUploadState(host, "error", "Ağ hatası: yükleme tamamlanamadı.");
      };

      xhr.ontimeout = function () {
        showUploadState(host, "error", "Yükleme zaman aşımına uğradı.");
      };

      setUploadPhase(host, "uploading", "Yükleniyor…");
      setUploadProgress(host, 0, "", "", NaN);
      disableForm(form, true);
      xhr.send(new FormData(form));
    });
  }

  function disableForm(form, on) {
    Array.prototype.forEach.call(form.querySelectorAll("button, input, select"), function (el) {
      if (on) { el.dataset.wasDisabled = el.disabled ? "1" : "0"; el.disabled = true; }
      else if (el.dataset.wasDisabled !== "1") el.disabled = false;
    });
  }

  function uploadBox(host) {
    var box = host.querySelector("[data-ses-upload-state]");
    if (box) return box;
    box = document.createElement("div");
    box.className = "ses-upload-state";
    box.setAttribute("data-ses-upload-state", "");
    box.setAttribute("role", "status");
    box.setAttribute("aria-live", "polite");
    box.innerHTML =
      '<div class="ses-up-row">' +
      '<span class="ses-up-spinner" aria-hidden="true"></span>' +
      '<span class="ses-up-label">Yükleniyor…</span>' +
      '<span class="ses-up-pct"></span>' +
      '</div>' +
      '<div class="ses-up-bar"><i></i></div>' +
      '<div class="ses-up-meta"><span class="ses-up-bytes"></span>' +
      '<span class="ses-up-speed"></span><span class="ses-up-eta"></span></div>';
    var form = host.querySelector("form.ses-upload");
    if (form && form.parentNode) form.parentNode.insertBefore(box, form.nextSibling);
    else host.insertBefore(box, host.firstChild);
    return box;
  }

  function setUploadPhase(host, phase, label) {
    var box = uploadBox(host);
    box.dataset.phase = phase;
    box.className = "ses-upload-state ses-up-" + phase;
    var l = box.querySelector(".ses-up-label");
    if (l && label) l.textContent = label;
  }

  function setUploadProgress(host, pct, bytes, speed, eta) {
    var box = uploadBox(host);
    var bar = box.querySelector(".ses-up-bar i");
    if (bar) bar.style.width = pct + "%";
    var p = box.querySelector(".ses-up-pct");
    if (p) p.textContent = pct + "%";
    var b = box.querySelector(".ses-up-bytes");
    if (b) b.textContent = bytes || "";
    var s = box.querySelector(".ses-up-speed");
    if (s) s.textContent = speed || "";
    var e = box.querySelector(".ses-up-eta");
    if (e) e.textContent = isFinite(eta) && eta > 0 ? "· kalan ~" + fmtSeconds(eta) : "";
  }

  function showUploadState(host, kind, msg) {
    var box = uploadBox(host);
    box.className = "ses-upload-state ses-up-" + kind;
    var l = box.querySelector(".ses-up-label");
    if (l) l.textContent = msg;
    var bar = box.querySelector(".ses-up-bar");
    if (bar) bar.style.display = "none";
    var meta = box.querySelector(".ses-up-meta");
    if (meta) meta.style.display = "none";
    var p = box.querySelector(".ses-up-pct");
    if (p) p.textContent = "";
    var form = host.querySelector("form.ses-upload");
    if (form) disableForm(form, false);
  }

  
  function seek(btn, word) {
    var mediaID = btn.getAttribute("data-audio-seek");
    var attr = word ? word.getAttribute("data-ms") : btn.getAttribute("data-start-ms");
    var ms = Number.parseInt(attr || "0", 10);
    if (!mediaID || isNaN(ms)) return;
    var player = document.querySelector('audio[data-audio-player="' + mediaID + '"]');
    if (!player) return;
    try {
      player.currentTime = Math.max(0, ms / 1000);
      var p = player.play();
      if (p && p.catch) p.catch(function () {});
    } catch (err) {
      
    }
    if (player.scrollIntoView) {
      player.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
  }

  function openDraft(btn) {
    var url = btn.getAttribute("data-audio-draft");
    if (!url || !window.__aiEditor) return;
    window.__aiEditor.open("Ses kaydından taslak");
    window.__aiEditor.loading(true);
    fetch(url, {
      method: "POST",
      credentials: "same-origin",
      headers: {
        "HX-Request": "true",
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8"
      }
    })
      .then(function (r) { return r.text(); })
      .then(function (html) {
        window.__aiEditor.loading(false);
        window.__aiEditor.setHTML(html);
      })
      .catch(function () {
        window.__aiEditor.loading(false);
        window.__aiEditor.setHTML(
          '<div class="ai-result-error"><strong>Bağlantı hatası.</strong>' +
          '<p>Ses taslağı isteği gönderilemedi. Sunucu bağlantısını kontrol edin.</p></div>'
        );
      });
  }

  document.addEventListener("DOMContentLoaded", bindPolling);
  document.addEventListener("htmx:afterSwap", bindPolling);
  // DOMContentLoaded kaçırılmış olabilir (defer + htmx): hemen de dene.
  if (document.readyState !== "loading") bindPolling();

  document.addEventListener("click", function (e) {
    var seekBtn = e.target.closest("[data-audio-seek]");
    if (seekBtn) {
      e.preventDefault();
      seek(seekBtn, e.target.closest("[data-ses-word]"));
      return;
    }
    var draftBtn = e.target.closest("[data-audio-draft]");
    if (draftBtn) {
      e.preventDefault();
      openDraft(draftBtn);
    }
  });
})();
