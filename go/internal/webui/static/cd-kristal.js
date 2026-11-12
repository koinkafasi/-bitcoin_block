// cd-kristal.js — yazı sayfası görsellerinde imleci izleyen parlama.
//
// Desenin kendisi CSS'te (.cd-kristal). Buradaki tek iş imlecin
// konumunu `--cd-mx/--cd-my` değişkenlerine yazmak.
//
// PERFORMANS KARARLARI:
//  · Dinleyici KAPSAYICIYA değil, document'e tek seferlik bağlanıyor
//    (olay delegasyonu): 40 görselde 40 ayrı dinleyici yerine 1 tane.
//  · Yazma requestAnimationFrame ile birleştiriliyor. mousemove saniyede
//    ~120 kez tetiklenir; her birinde style yazmak düzen hesabını
//    gereksiz yere uyandırır. rAF ile kare başına en çok bir yazma olur.
//  · Yalnız imleçli cihazlarda çalışır; dokunmatikte CSS zaten kapatıyor.
//  · `passive: true` — kaydırmayı bloklamaz.
(function () {
  "use strict";

  if (!window.matchMedia || !window.matchMedia("(hover: hover)").matches) {
    return;
  }

  var bekleyen = null; // {el, x, y}
  var planli = false;

  function uygula() {
    planli = false;
    if (!bekleyen) return;
    var b = bekleyen;
    bekleyen = null;
    b.el.style.setProperty("--cd-mx", b.x + "%");
    b.el.style.setProperty("--cd-my", b.y + "%");
  }

  document.addEventListener(
    "mousemove",
    function (e) {
      var el = e.target && e.target.closest ? e.target.closest(".cd-kristal") : null;
      if (!el) return;

      var r = el.getBoundingClientRect();
      if (!r.width || !r.height) return;

      bekleyen = {
        el: el,
        x: Math.round(((e.clientX - r.left) / r.width) * 100),
        y: Math.round(((e.clientY - r.top) / r.height) * 100),
      };
      if (!planli) {
        planli = true;
        requestAnimationFrame(uygula);
      }
    },
    { passive: true }
  );

  // İmleç çıkınca değişkenleri temizle: kalan değer, bir sonraki girişte
  // parlamanın eski noktadan "sıçramasına" yol açıyordu.
  document.addEventListener(
    "mouseout",
    function (e) {
      var el = e.target && e.target.closest ? e.target.closest(".cd-kristal") : null;
      if (!el) return;
      if (e.relatedTarget && el.contains(e.relatedTarget)) return;
      el.style.removeProperty("--cd-mx");
      el.style.removeProperty("--cd-my");
    },
    { passive: true }
  );
})();
