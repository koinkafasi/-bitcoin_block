// Ana sayfa sekme kategorileri paneli (templ sürümü).
// Sürükle-bırak sıralama + ekle/kaydet davranışı; HTMX sonrası yeniden bağlanır.
(function () {
  function init() {
    var root = document.querySelector("[data-tabcats]");
    if (!root || root.getAttribute("data-ready") === "1") return;
    root.setAttribute("data-ready", "1");

    var list = root.querySelector("#tcItems");
    if (!list) return;
    var dragItem = null;

    function counts() {
      var n = list.querySelectorAll("li[data-slug]").length;
      var c = root.querySelector("#tcCount");
      if (c) c.textContent = String(n);
    }

    function msg(t, k) {
      var st = root.querySelector("#tcStatus");
      if (!st) return;
      // Sunucu iletisi HTML olarak yorumlanmaz.
      var d = document.createElement("div");
      d.className = "flash " + (k === "success" ? "success" : "error");
      d.style.margin = "8px 0 0";
      d.textContent = t;
      st.textContent = "";
      st.appendChild(d);
      setTimeout(function () { st.textContent = ""; }, 4000);
    }

    function makeLi(slug) {
      var li = document.createElement("li");
      li.setAttribute("data-slug", slug);
      li.setAttribute("draggable", "true");
      li.style.cssText = "display:flex;align-items:center;gap:8px;padding:8px 10px;border:1px solid var(--border);border-radius:10px;margin-bottom:8px;background:var(--surface,#fff)";
      li.innerHTML =
        '<span style="cursor:grab;color:#a1a1aa;font-size:18px;user-select:none" title="Sürükle"><svg class="ti" aria-hidden="true"><use href="/static/vendor/tabler-icons.svg?v=b89545a6#ti-grip-vertical"></use></svg></span>' +
        '<button type="button" class="btn ghost small tc-up" title="Yukarı">↑</button>' +
        '<button type="button" class="btn ghost small tc-down" title="Aşağı">↓</button>' +
        '<span class="tc-ad" style="flex:1;min-width:120px;font-weight:600">/' + slug + "</span>" +
        '<a class="btn danger small tc-del" href="#" title="Kaldır">✕</a>';
      return li;
    }

    list.addEventListener("click", function (e) {
      var li = e.target.closest("li[data-slug]");
      if (!li) return;
      if (e.target.closest(".tc-up")) {
        e.preventDefault();
        var p = li.previousElementSibling;
        if (p) list.insertBefore(li, p);
        counts();
      } else if (e.target.closest(".tc-down")) {
        e.preventDefault();
        var nx = li.nextElementSibling;
        if (nx) list.insertBefore(nx, li);
        counts();
      } else if (e.target.closest(".tc-del")) {
        e.preventDefault();
        if (li.parentNode) li.parentNode.removeChild(li);
        counts();
      }
    });

    list.addEventListener("dragstart", function (e) {
      dragItem = e.target.closest("li[data-slug]");
      if (dragItem) e.dataTransfer.effectAllowed = "move";
    });
    list.addEventListener("dragover", function (e) {
      e.preventDefault();
      var t = e.target.closest("li[data-slug]");
      if (dragItem && t && t !== dragItem) {
        var r = dragItem.compareDocumentPosition(t) & Node.DOCUMENT_POSITION_PRECEDING ? "afterend" : "beforeend";
        list.insertBefore(dragItem, r === "afterend" ? t.nextSibling : t);
      }
    });
    list.addEventListener("dragend", function () {
      dragItem = null;
      counts();
    });

    var addBtn = root.querySelector("#tcAddCats");
    if (addBtn) {
      addBtn.addEventListener("click", function () {
        Array.prototype.slice.call(root.querySelectorAll(".tc-cat:checked")).forEach(function (chk) {
          var slug = chk.value;
          var exists = list.querySelector('li[data-slug="' + slug + '"]');
          if (!exists) list.appendChild(makeLi(slug));
          chk.checked = false;
        });
        counts();
      });
    }

    var saveBtn = root.querySelector("#tcSave");
    if (saveBtn) {
      saveBtn.addEventListener("click", function () {
        var slugs = [];
        list.querySelectorAll("li[data-slug]").forEach(function (li) {
          slugs.push(li.getAttribute("data-slug"));
        });
        if (!slugs.length && !confirm("Hiç kategori seçmediniz. Kaydederseniz sekme bloğu boş kalır. Devam edilsin mi?")) return;
        if (window.htmx) {
          var hid = root.querySelector('#tabcatsHtmxForm input[name="slugs"]');
          if (hid) hid.value = JSON.stringify(slugs);
          htmx.trigger(root.querySelector("#tabcatsHtmxForm"), "submit");
          return;
        }
        fetch("/admin/tab-kategorileri/kaydet", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ slugs: slugs })
        })
          .then(function (r) { return r.json(); })
          .then(function (d) {
            if (d.ok) {
              msg("Sekmeler kaydedildi (" + d.adet + " kategori).", "success");
              setTimeout(function () { window.location.reload(); }, 900);
            } else {
              msg("Hata: " + (d.error || "bilinmeyen"), "error");
            }
          })
          .catch(function (e) { msg("Ağ hatası: " + e, "error"); });
      });
    }

    var resetBtn = root.querySelector("#tcReset");
    if (resetBtn) {
      resetBtn.addEventListener("click", function () {
        if (window.htmx) return; // hx-post + hx-confirm
        if (!confirm("Sekme listesini sıfırlayın mı? Blok menüdeki ilk 8 kategoriye döner.")) return;
        fetch("/admin/tab-kategorileri/sifirla", { method: "POST" }).then(function () {
          window.location.reload();
        });
      });
    }

    counts();
  }

  document.addEventListener("DOMContentLoaded", init);
  document.addEventListener("htmx:afterSwap", init);
  if (document.readyState !== "loading") init();
})();
