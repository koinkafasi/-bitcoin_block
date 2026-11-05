// piyasa.js — Kur & Borsa şeridinde fareyle sürükleyerek kaydırma.
//
// Neden gerekli: dokunmatik cihazlarda parmakla kaydırma yerleşik
// (overflow-x: auto). Masaüstünde fare tekerleği dikey kaydırır; şeridi
// yatay gezmenin doğal yolu sürüklemektir. Üst manşet de aynı şerit
// davranışını kullanır; orada kartlar tıklanabilir olduğu için sürükleme
// eklenmemişti — burada ekliyoruz ki şerit "tutup çekilebilir" olsun.
//
// Sözleşme:
//   • Yalnız [data-pys-scroll] öğelerinde çalışır (başka şeride dokunmaz).
//   • 5 px'ten fazla sürüklendiyse tıklama bastırılır: kullanıcı şeridi
//     kaydırırken yanlışlıkla /borsa sayfasına gitmez.
//   • Dokunmatikte (pointerType= touch) devre dışı: tarayıcı kendi
//     ataletli kaydırmasını yapsın, çift davranış olmasın.
(function () {
  "use strict";

  var ESIK = 5; // px — bunun altı tıklama sayılır

  function bagla(el) {
    if (el.dataset.pysDrag === "1") return;
    el.dataset.pysDrag = "1";

    var baslangicX = 0;
    var baslangicScroll = 0;
    var surukledi = false;
    var aktif = false;

    el.addEventListener("pointerdown", function (e) {
      if (e.pointerType === "touch") return; // yerleşik kaydırma yeterli
      if (e.button !== 0) return; // yalnız sol tuş
      aktif = true;
      surukledi = false;
      baslangicX = e.clientX;
      baslangicScroll = el.scrollLeft;
    });

    el.addEventListener("pointermove", function (e) {
      if (!aktif) return;
      var fark = e.clientX - baslangicX;
      if (!surukledi && Math.abs(fark) > ESIK) {
        surukledi = true;
        el.classList.add("is-dragging");
        // Sürükleme başladı: pointer'ı yakala ki hızlı hareket kaybolmasın.
        if (el.setPointerCapture) {
          try { el.setPointerCapture(e.pointerId); } catch (_) {}
        }
      }
      if (surukledi) {
        el.scrollLeft = baslangicScroll - fark;
        e.preventDefault();
      }
    });

    function bitir() {
      if (!aktif) return;
      aktif = false;
      el.classList.remove("is-dragging");
    }

    el.addEventListener("pointerup", bitir);
    el.addEventListener("pointercancel", bitir);
    el.addEventListener("pointerleave", function () {
      if (aktif && !surukledi) bitir();
    });

    // Sürükleme sonrası tıklamayı yut (bağlantıya gitmesin).
    el.addEventListener(
      "click",
      function (e) {
        if (surukledi) {
          e.preventDefault();
          e.stopPropagation();
          surukledi = false;
        }
      },
      true
    );
  }

  function tara() {
    var list = document.querySelectorAll("[data-pys-scroll]");
    for (var i = 0; i < list.length; i++) bagla(list[i]);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", tara);
  } else {
    tara();
  }
  // HTMX / tasarımcı önizlemesi sonradan ekleyebilir.
  document.addEventListener("htmx:afterSwap", tara);
  document.addEventListener("htmx:afterSettle", tara);
})();
