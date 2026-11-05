// AI Sohbet paneli davranışı (templ sürümü).
// Panel HTMX ile yeniden basıldığında da bağlanır (data-ready + htmx:afterSwap).
(function () {
  function init() {
    var panel = document.querySelector("[data-ai-chat]");
    if (!panel || panel.getAttribute("data-ready") === "1") return;
    panel.setAttribute("data-ready", "1");

    var form = panel.querySelector("#ai-form");
    var input = panel.querySelector("#ai-input");
    var msgs = panel.querySelector("#ai-msgs");
    var status = panel.querySelector("#ai-status");
    var send = panel.querySelector("#ai-send");
    if (!form || !input || !msgs || !status) return;
    msgs.scrollTop = msgs.scrollHeight;

    function append(role, text) {
      var d = document.createElement("div");
      d.style.cssText = "margin:10px 0;max-width:85%";
      var h = document.createElement("div");
      h.className = "small muted";
      h.textContent = role === "user" ? "🧑‍💻 Siz" : "🤖 AI";
      var b = document.createElement("div");
      b.style.whiteSpace = "pre-wrap";
      b.style.padding = "9px 12px";
      b.style.borderRadius = "10px";
      b.style.marginTop = "3px";
      if (role === "user") {
        b.style.background = "var(--accent)";
        b.style.color = "#fff";
      } else {
        b.style.background = "var(--surface-2)";
      }
      b.textContent = text;
      d.appendChild(h);
      d.appendChild(b);
      msgs.appendChild(d);
      msgs.scrollTop = msgs.scrollHeight;
    }

    function sendMsg(text) {
      if (window.htmx) {
        input.value = text;
        htmx.trigger(form, "submit");
        return;
      }
      var conv = form.getAttribute("data-conv");
      append("user", text);
      input.value = "";
      status.textContent = "Yanıt bekleniyor…";
      if (send) send.disabled = true;
      fetch("/admin/ai-sohbet/" + conv + "/mesaj", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ metin: text })
      })
        .then(function (r) { return r.json(); })
        .then(function (j) {
          if (send) send.disabled = false;
          if (j.ok) {
            append("assistant", j.result);
            status.textContent = "";
            if (j.conv_id && String(j.conv_id) !== conv) {
              form.setAttribute("data-conv", j.conv_id);
              if (history.replaceState) history.replaceState(null, "", "/admin/ai-sohbet?konu=" + j.conv_id);
            }
          } else {
            status.textContent = "Hata: " + (j.error || "bilinmeyen");
          }
        })
        .catch(function () {
          if (send) send.disabled = false;
          status.textContent = "Ağ hatası";
        });
    }

    form.addEventListener("submit", function (e) {
      if (window.htmx) return;
      e.preventDefault();
      var text = input.value.trim();
      if (!text) return;
      sendMsg(text);
    });

    input.addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        form.dispatchEvent(new Event("submit"));
      }
    });

    panel.querySelectorAll("[data-tpl]").forEach(function (b) {
      b.addEventListener("click", function () {
        sendMsg(b.getAttribute("data-tpl"));
      });
    });
  }

  document.addEventListener("DOMContentLoaded", init);
  document.addEventListener("htmx:afterSwap", init);
  if (document.readyState !== "loading") init();
})();
