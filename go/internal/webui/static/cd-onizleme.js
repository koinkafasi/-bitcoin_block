// cd-onizleme.js — video kartı üstüne gelince kısa hareketli önizleme.
//
// `data-onizleme` taşıyan <img>'in kaynağı hover'da değiştirilir, imleç
// çıkınca geri alınır. Yeni DOM elemanı yok, <video> yok.
//
// PERFORMANS KARARLARI:
//  · İstek YALNIZ hover'da. Liste sayfasının ilk yüklemesine sıfır yük.
//  · 140 ms gecikme: kullanıcı kartların üstünden geçerken her birini
//    indirmek bir ekranda 15+ gereksiz istek demekti. Niyet beklenir.
//  · Önizleme GİZLİ bir Image() ile ön-yüklenir; yalnız başarıyla
//    yüklenirse <img>'e geçilir. Doğrudan src atasaydık, 404 durumunda
//    kart bir an boş kalırdı.
//  · Tek belge dinleyicisi (delegasyon), kart başına dinleyici yok.
(function () {
  "use strict";

  if (!window.matchMedia || !window.matchMedia("(hover: hover)").matches) {
    return;
  }

  var GECIKME_MS = 140;
  var zamanlayici = null;
  var etkin = null; // o an önizleme gösteren <img>

  function geriAl(img) {
    if (!img) return;
    var asil = img.getAttribute("data-asil-src");
    if (asil) {
      img.src = asil;
      img.removeAttribute("data-asil-src");
    }
    // srcset geri gelmeli: önizlemeye geçerken boşaltılmıştı, yoksa
    // tarayıcı srcset'i tercih edip önizlemeyi hiç göstermezdi.
    var asilSet = img.getAttribute("data-asil-srcset");
    if (asilSet !== null) {
      img.srcset = asilSet;
      img.removeAttribute("data-asil-srcset");
    }
  }

  function basla(img) {
    var url = img.getAttribute("data-onizleme");
    if (!url || img.getAttribute("data-asil-src")) return;

    var deneme = new Image();
    deneme.onload = function () {
      // İmleç bu arada çıkmış olabilir.
      if (etkin !== img) return;
      img.setAttribute("data-asil-src", img.currentSrc || img.src);
      img.setAttribute("data-asil-srcset", img.srcset || "");
      img.srcset = "";
      img.src = url;
    };
    // onerror: sessiz. Önizleme YouTube'un belgelenmemiş ucundan gelir;
    // yoksa kart statik kalır, kullanıcı bir eksiklik görmez.
    deneme.src = url;
  }

  document.addEventListener(
    "mouseover",
    function (e) {
      var img = e.target && e.target.closest
        ? e.target.closest("img[data-onizleme]")
        : null;
      if (!img || img === etkin) return;

      if (etkin) geriAl(etkin);
      etkin = img;
      clearTimeout(zamanlayici);
      zamanlayici = setTimeout(function () {
        if (etkin === img) basla(img);
      }, GECIKME_MS);
    },
    { passive: true }
  );

  document.addEventListener(
    "mouseout",
    function (e) {
      var img = e.target && e.target.closest
        ? e.target.closest("img[data-onizleme]")
        : null;
      if (!img || img !== etkin) return;
      if (e.relatedTarget && img.contains(e.relatedTarget)) return;
      clearTimeout(zamanlayici);
      geriAl(img);
      etkin = null;
    },
    { passive: true }
  );
})();
