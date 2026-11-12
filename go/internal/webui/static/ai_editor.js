
(function () {
  "use strict";
  var drawer = document.getElementById("aiEditor");
  if (!drawer) return;
  var menu = document.getElementById("aiSelectMenu");
  var body = document.getElementById("aiEditorBody");
  var loading = document.getElementById("aiEditorLoading");
  var contextBox = document.getElementById("aiEditorContext");
  var instruction = document.getElementById("aiInstruction");
  var angle = document.getElementById("aiAngle");
  var detail = document.querySelector('[name="detay"]');
  function tiptap() { return window.__contentEditor || null; }
  var contentID = drawer.getAttribute("data-content-id") || "0";
  var state = { action: "", selection: "", start: 0, end: 0, title: "" };
  var lastMouse = { x: 0, y: 0 };
  var pollGen = 0;
  var pollTimer = null;

  function stopPolling() {
    pollGen++;
    if (pollTimer) { clearTimeout(pollTimer); pollTimer = null; }
  }
  function renderResultHTML(html, autoPoll) {
    body.innerHTML = html;
    var result = body.querySelector(".ai-result");
    if (result && result.getAttribute("data-job-id")) {
      drawer.setAttribute("data-job-id", result.getAttribute("data-job-id"));
    }
    var pending = body.querySelector("[data-ai-pending]");
    if (pending && autoPoll !== false) startPolling(pending.getAttribute("data-ai-poll-url"));
  }
  function startPolling(url) {
    if (!url) return;
    stopPolling();
    var gen = pollGen;
    function tick() {
      fetch(url, { credentials: "same-origin", headers: { "HX-Request": "true" } })
        .then(function (r) { return r.text(); })
        .then(function (html) {
          if (gen !== pollGen || drawer.hidden) return;
          renderResultHTML(html, false);
          if (body.querySelector("[data-ai-pending]")) {
            pollTimer = setTimeout(tick, 1500);
          }
        })
        .catch(function () {
          if (gen !== pollGen || drawer.hidden) return;
          pollTimer = setTimeout(tick, 4000);
        });
    }
    pollTimer = setTimeout(tick, 700);
  }

  function isNew() { return !contentID || contentID === "0"; }
  function esc(s) { return String(s == null ? "" : s); }
  function hideMenu() { if (menu) menu.hidden = true; }
  function showMenu(x, y, text, start, end, allowEmpty) {
    if (!menu) return;
    if ((!text || text.trim().length < 3) && !allowEmpty) return;
    state.selection = text || "";
    state.start = start || 0;
    state.end = end || 0;
    var count = document.getElementById("aiSelCount");
    if (count) count.textContent = String((text || "").trim().length);
    menu.hidden = false;
    var w = menu.offsetWidth || 320;
    var h = menu.offsetHeight || 420;
    var left = Math.max(8, Math.min(x, window.innerWidth - w - 8));
    var top = Math.max(8, Math.min(y, window.innerHeight - h - 8));
    menu.style.left = left + "px";
    menu.style.top = top + "px";
  }
  function selectedText() {
    var ed = tiptap();
    if (ed) {
      var sel = ed.state.selection;
      if (!sel || sel.empty) return "";
      return ed.state.doc.textBetween(sel.from, sel.to, "\n");
    }
    if (!detail) return "";
    var a = detail.selectionStart || 0, b = detail.selectionEnd || 0;
    return a < b ? detail.value.slice(a, b) : "";
  }
  function openDrawer(title, reset) {
    var wasHidden = drawer.hidden;
    drawer.hidden = false;
    drawer.setAttribute("aria-hidden", "false");
    document.body.classList.add("ai-drawer-open");
    var t = document.getElementById("aiEditorTitle");
    if (t) t.textContent = title || "AI işlemi";
    if (reset !== false && wasHidden) {
      if (instruction) instruction.value = "";
      if (angle) angle.value = "";
      body.innerHTML = '<p class="muted small">Metin seçip menüden bir işlem seçin. Sonuç önce burada gösterilir; siz uygulamadan habere yazılmaz.</p>';
    }
    var close = drawer.querySelector("[data-ai-close]");
    if (close && wasHidden) close.focus();
  }
  function closeDrawer() {
    stopPolling();
    drawer.hidden = true;
    drawer.setAttribute("aria-hidden", "true");
    document.body.classList.remove("ai-drawer-open");
    var ed = tiptap();
    if (ed) ed.commands.focus();
    else if (detail) detail.focus();
  }
  function setContext() {
    if (!contextBox) return;
    if (isNew()) {
      contextBox.innerHTML = '<div class="ai-warn">AI işlemi için önce haberi kaydedin.</div>';
      return;
    }
    var sel = state.selection || selectedText();
    if (sel) {
      contextBox.innerHTML = '<div class="small muted">Seçili metin (' + sel.trim().length + ' karakter)</div><div class="ai-context-text">' + escapeHTML(sel.slice(0, 180)) + (sel.length > 180 ? "…" : "") + "</div>";
    } else {
      contextBox.innerHTML = '<div class="small muted">Metin seçilmedi — işlem tüm haber gövdesi üzerinde çalışacak.</div>';
    }
  }
  function escapeHTML(s) {
    return esc(s).replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
  }
  function showLoading(on) {
    if (loading) loading.hidden = !on;
    if (body) body.setAttribute("aria-busy", on ? "true" : "false");
  }
  function runAction(action) {
    if (isNew()) { setContext(); return; }
    if (!action) return;
    state.action = action;
    stopPolling();
    if (menu) hideMenu();
    openDrawer(state.title || action);
    setContext();
    showLoading(true);
    body.innerHTML = "";
    var params = new URLSearchParams();
    params.set("action", action);
    params.set("selection", state.selection || selectedText());
    params.set("instruction", instruction ? instruction.value : "");
    params.set("angle", angle ? angle.value : "");
    fetch("/admin/icerikler/" + encodeURIComponent(contentID) + "/ai/eylem", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8", "HX-Request": "true" },
      body: params.toString()
    }).then(function (r) { return r.text(); }).then(function (html) {
      showLoading(false);
      renderResultHTML(html);
    }).catch(function () {
      showLoading(false);
      stopPolling();
      body.innerHTML = '<div class="ai-result-error"><strong>Bağlantı hatası.</strong><p>AI isteği gönderilemedi. İnternet/sunucu bağlantısını kontrol edin.</p></div>';
    });
  }
  function aiBlocks(value) {
    var parts = String(value || "").split(/\n{2,}/).map(function (s) { return s.trim(); }).filter(Boolean);
    return parts.map(function (text) {
      return { type: "paragraph", content: [{ type: "text", text: text }] };
    });
  }
  function applyValue(el, value, mode) {
    if (!el || value == null) return false;
    value = String(value);
    var ed = tiptap();
    if (ed && (el === detail || el.id === "f_detail" || el.closest && el.closest("#blockEditor"))) {
      var blocks = aiBlocks(value);
      if (!blocks.length) return false;
      if (mode === "replace-selection" && !ed.state.selection.empty) {
        ed.chain().focus().insertContent(blocks).run();
      } else {
        var end = ed.state.doc.content.size;
        ed.chain().focus().insertContentAt(end, blocks).run();
      }
      if (window.__contentEditorSync) window.__contentEditorSync();
      return true;
    }
    if (el.tagName === "TEXTAREA" && el === detail) {
      var start = state.start, end = state.end;
      if (mode === "replace-selection" && start < end && end <= el.value.length && el.value.slice(start, end) === state.selection) {
        el.focus();
        el.setRangeText(value, start, end, "end");
      } else {
        el.value = el.value.replace(/\s+$/, "") + (el.value.trim() ? "\n\n" : "") + value;
      }
    } else {
      el.value = value;
    }
    el.dispatchEvent(new Event("input", { bubbles: true }));
    el.dispatchEvent(new Event("change", { bubbles: true }));
    return true;
  }
  function notice(msg) {
    var n = drawer.querySelector(".ai-apply-notice");
    if (!n) {
      n = document.createElement("div");
      n.className = "ai-apply-notice";
      drawer.querySelector(".ai-drawer-panel").appendChild(n);
    }
    n.textContent = msg;
    n.classList.add("show");
    clearTimeout(n._t);
    n._t = setTimeout(function () { n.classList.remove("show"); }, 3500);
  }
  function sourceValue(btn) {
    var sel = btn.getAttribute("data-ai-apply-source");
    if (sel) {
      var el = drawer.querySelector(sel);
      if (el) return el.value;
    }
    return btn.getAttribute("data-ai-apply-value") || "";
  }
  function copyResult() {
    var parts = [];
    var fields = body.querySelectorAll("#aiFieldTitle,#aiFieldSpot,#aiFieldIntro,#aiFieldBody,#aiFieldSeoTitle,#aiFieldMeta,#aiFieldSlug");
    fields.forEach(function (el) {
      var label = el.previousElementSibling ? el.previousElementSibling.textContent : "";
      if (el.value.trim()) parts.push((label ? label + ": " : "") + el.value.trim());
    });
    if (!parts.length) {
      body.querySelectorAll(".ai-sec-text").forEach(function (el) { if (el.value.trim()) parts.push(el.value.trim()); });
    }
    body.querySelectorAll(".ai-claim").forEach(function (el) { parts.push(el.innerText.trim()); });
    var text = parts.join("\n\n");
    if (!text) { notice("Kopyalanacak metin yok."); return; }
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(function () { notice("Kopyalandı."); });
    } else {
      var ta = document.createElement("textarea");
      ta.value = text; document.body.appendChild(ta); ta.select();
      try { document.execCommand("copy"); notice("Kopyalandı."); } catch (e) { notice("Kopyalanamadı."); }
      ta.remove();
    }
  }
  function newDraft(btn) {
    var job = btn.getAttribute("data-job-id") || drawer.getAttribute("data-job-id");
    if (!job || isNew()) return;
    btn.disabled = true;
    var params = new URLSearchParams();
    params.set("job_id", job);
    fetch("/admin/icerikler/" + encodeURIComponent(contentID) + "/ai/yeni-taslak", {
      method: "POST", credentials: "same-origin",
      headers: { "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8", "HX-Request": "true" },
      body: params.toString()
    }).then(function (r) {
      var redirect = r.headers.get("HX-Redirect");
      if (redirect) { window.location.href = redirect; return; }
      return r.text().then(function (t) { body.innerHTML = t; });
    }).catch(function () { notice("Yeni taslak oluşturulamadı."); })
      .finally(function () { btn.disabled = false; });
  }
  function loadHistory(btn) {
    if (isNew()) return;
    stopPolling();
    openDrawer("AI işlem geçmişi");
    showLoading(true);
    fetch("/admin/icerikler/" + encodeURIComponent(contentID) + "/ai/gecmis", {
      credentials: "same-origin", headers: { "HX-Request": "true" }
    }).then(function (r) { return r.text(); }).then(function (html) {
      showLoading(false); body.innerHTML = html;
    }).catch(function () {
      showLoading(false); body.innerHTML = '<div class="ai-result-error">Geçmiş yüklenemedi.</div>';
    });
  }

  if (menu) {
    menu.addEventListener("mousedown", function (e) { e.preventDefault(); });
    menu.addEventListener("click", function (e) {
      var b = e.target.closest("[data-ai-open]");
      if (!b) return;
      state.title = b.querySelector("b") ? b.querySelector("b").textContent : "";
      runAction(b.getAttribute("data-ai-open"));
    });
  }

  if (detail) {
    detail.addEventListener("mouseup", function (e) {
      lastMouse.x = e.clientX; lastMouse.y = e.clientY;
      setTimeout(function () {
        var text = selectedText();
        if (text.trim().length >= 3) showMenu(e.clientX + 8, e.clientY + 8, text, detail.selectionStart, detail.selectionEnd);
        else hideMenu();
      }, 0);
    });
    detail.addEventListener("keyup", function (e) {
      if (e.key === "Escape") { hideMenu(); return; }
      if (!(e.shiftKey || e.ctrlKey || e.metaKey)) return;
      setTimeout(function () {
        var text = selectedText();
        if (text.trim().length >= 3) {
          var r = detail.getBoundingClientRect();
          showMenu(r.left + 24, r.top + 36, text, detail.selectionStart, detail.selectionEnd);
        }
      }, 0);
    });
  }
  document.addEventListener("mouseup", function (e) {
    if (!tiptap() || !e.target.closest || !e.target.closest(".ProseMirror")) return;
    setTimeout(function () {
      var text = selectedText();
      if (text.trim().length >= 3) showMenu(e.clientX + 8, e.clientY + 8, text, 0, 0);
      else hideMenu();
    }, 0);
  });
  document.addEventListener("keyup", function (e) {
    if (!tiptap() || !e.target.closest || !e.target.closest(".ProseMirror")) return;
    if (e.key === "Escape") { hideMenu(); return; }
    if (!(e.shiftKey || e.ctrlKey || e.metaKey)) return;
    setTimeout(function () {
      var text = selectedText();
      if (text.trim().length >= 3) {
        var r = e.target.getBoundingClientRect();
        showMenu(r.left + 24, r.top + 36, text, 0, 0);
      }
    }, 0);
  });

  
  document.addEventListener("contextmenu", function (e) {
    var host = e.target && e.target.closest ? e.target.closest("[data-ai-ctx]") : null;
    if (!host || !menu) return;
    e.preventDefault();
    var text = selectedText();
    showMenu(e.clientX, e.clientY, text, detail ? detail.selectionStart : 0,
      detail ? detail.selectionEnd : 0, true);
  });
  document.addEventListener("mousedown", function (e) {
    if (menu && !menu.hidden && !e.target.closest("#aiSelectMenu")) hideMenu();
  });
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") { if (menu) hideMenu(); if (!drawer.hidden) closeDrawer(); }
  });
  window.addEventListener("scroll", hideMenu, { passive: true });
  window.addEventListener("resize", hideMenu);

  
  document.addEventListener("click", function (e) {
    var legacy = e.target.closest("[data-ai]");
    if (!legacy) return;
    if (e.target.closest("#aiSelectMenu")) return;
    e.preventDefault();
    e.stopPropagation();
    if (e.stopImmediatePropagation) e.stopImmediatePropagation();
    var map = {
      title: "headlines", spot: "intro", summarize: "summary",
      rewrite: "rewrite_angle", seo_title: "seo", seo_description: "seo"
    };
    state.selection = selectedText();
    state.start = detail ? detail.selectionStart : 0;
    state.end = detail ? detail.selectionEnd : 0;
    state.title = legacy.textContent.trim();
    runAction(map[legacy.getAttribute("data-ai")] || "news");
  }, true);

  drawer.addEventListener("click", function (e) {
    if (e.target.closest("[data-ai-close]")) { closeDrawer(); return; }
    var alt = e.target.closest("[data-ai-headline]");
    if (alt) {
      var titleEl = document.getElementById("f_title");
      if (titleEl) {
        titleEl.value = alt.getAttribute("data-ai-headline");
        titleEl.dispatchEvent(new Event("input", { bubbles: true }));
        notice("Başlık alanına uygulandı; kaydetmeyi unutmayın.");
      }
      return;
    }
    var apply = e.target.closest("[data-ai-apply-source]");
    if (apply) {
      var target = document.querySelector(apply.getAttribute("data-ai-apply-target"));
      if (applyValue(target, sourceValue(apply), apply.getAttribute("data-ai-apply-mode"))) {
        notice("Sonuç alana yazıldı; haberi kaydetmeyi unutmayın.");
      }
      return;
    }
    if (e.target.closest("[data-ai-copy]")) { copyResult(); return; }
    if (e.target.closest("[data-ai-new-draft]")) { newDraft(e.target.closest("[data-ai-new-draft]")); return; }
    if (e.target.closest("[data-ai-history]")) { loadHistory(e.target.closest("[data-ai-history]")); return; }
    if (e.target.closest("[data-ai-retry]")) { runAction(state.action); return; }
    var cancel = e.target.closest("[data-ai-cancel]");
    if (cancel) {
      var curl = cancel.getAttribute("data-ai-cancel-url");
      cancel.disabled = true;
      fetch(curl, { method: "POST", credentials: "same-origin", headers: { "HX-Request": "true" } })
        .then(function () {
          var p = body.querySelector("[data-ai-pending]");
          if (p) startPolling(p.getAttribute("data-ai-poll-url"));
        })
        .catch(function () { notice("İş iptal edilemedi."); cancel.disabled = false; });
      return;
    }
    var job = e.target.closest("[data-ai-job-url]");
    if (job) {
      stopPolling();
      showLoading(true);
      fetch(job.getAttribute("data-ai-job-url"), { credentials: "same-origin", headers: { "HX-Request": "true" } })
        .then(function (r) { return r.text(); })
        .then(function (html) { showLoading(false); renderResultHTML(html); })
        .catch(function () { showLoading(false); notice("Sonuç yüklenemedi."); });
    }
  });
  
  document.addEventListener("click", function (e) {
    var btn = e.target.closest && e.target.closest("[data-quote-seek]");
    if (!btn) return;
    e.preventDefault();
    // Doğrulanmamış alıntının saniyesi yoktur; yanlış yere sarmayalım.
    if (btn.getAttribute("data-quote-bad") === "1") {
      notice("Bu alıntı kaynak transkriptte bulunamadı; dinlenecek bir saniye yok.");
      return;
    }
    var ms = Number.parseInt(btn.getAttribute("data-quote-seek") || "0", 10);
    var media = btn.getAttribute("data-quote-media") || "";
    if (isNaN(ms)) return;
    var player = media
      ? document.querySelector('audio[data-audio-player="' + media + '"]')
      : null;
    if (!player) player = document.querySelector("audio[data-audio-player]");
    if (!player) {
      notice("Ses kaydı bu sayfada bulunamadı. Ses panelini açıp tekrar deneyin.");
      return;
    }
    try {
      player.currentTime = Math.max(0, ms / 1000);
      var p = player.play();
      if (p && p.catch) p.catch(function () {});
      if (player.scrollIntoView) {
        player.scrollIntoView({ behavior: "smooth", block: "center" });
      }
    } catch (err) {
      notice("Kayıt oynatılamadı.");
    }
  });

  
  window.__aiEditor = {
    open: function (title) {
      stopPolling();
      openDrawer(title || "AI işlemi", false);
      body.innerHTML = "";
    },
    loading: showLoading,
    setHTML: function (html) { showLoading(false); renderResultHTML(html); },
    notice: notice
  };
})();
