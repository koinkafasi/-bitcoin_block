
function GZ_IKON(ad) {
  return '<svg class="ti" aria-hidden="true"><use href="/static/vendor/tabler-icons.svg?v=b89545a6#ti-' + ad + '"></use></svg>';
}

(function () {
  "use strict";

var flipEl = document.getElementById("gz-flip");
  if (flipEl) initReader();

  function initReader() {
    var pages = flipEl.querySelectorAll(".gz-flip-page");
    if (!pages.length) return;
    var info = document.getElementById("gz-pageinfo");
    var flip = null;

    if (window.StPageFlip && window.StPageFlip.PageFlip) {
      var pf = new window.StPageFlip.PageFlip(flipEl, {
        width: 420,
        height: 594,
        size: "stretch",
        minWidth: 240,
        maxWidth: 900,
        minHeight: 340,
        maxHeight: 1270,
        drawShadow: true,
        flippingTime: 600,
        usePortrait: true,
        autoSize: true,
        showCover: true,
        mobileScrollSupport: true
      });
      pf.loadFromHTML(pages);
      flip = pf;
      pf.on("flip", function (e) {
        if (info) info.textContent = "Sayfa " + (e.data + 1) + " / " + pages.length;
      });
    } else {
      // Fallback: yatay kaydırmalı basit görünüm
      flipEl.classList.add("gz-flip-fallback");
    }

    var prev = document.getElementById("gz-prev");
    var next = document.getElementById("gz-next");
    if (prev) prev.addEventListener("click", function () {
      if (flip) flip.flipPrev();
    });
    if (next) next.addEventListener("click", function () {
      if (flip) flip.flipNext();
    });

    var gridBtn = document.getElementById("gz-grid");
    var gridView = document.getElementById("gz-grid-view");
    if (gridBtn && gridView) {
      gridBtn.addEventListener("click", function () {
        var hidden = gridView.hasAttribute("hidden");
        if (hidden) {
          gridView.removeAttribute("hidden");
          flipEl.style.display = "none";
          gridBtn.innerHTML = GZ_IKON("book") + "Okumaya Dön";
        } else {
          gridView.setAttribute("hidden", "");
          flipEl.style.display = "";
          gridBtn.innerHTML = GZ_IKON("layout-grid") + "Tüm Sayfalar";
        }
      });
    }
  }
})();
