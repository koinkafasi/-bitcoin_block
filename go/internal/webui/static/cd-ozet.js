(function () {
  "use strict";

  var ozet = document.querySelector("[data-ozet]");
  if (!ozet) return;

  var govde = ozet.querySelector(".cd-ozet__govde");
  var perde = ozet.querySelector("[data-ozet-kapat]");
  var mobil = window.matchMedia("(max-width: 767px)");

  var kok = document.documentElement;
  var yedekPad = "";

  function kilitAc() {
    if (mobil.matches && ozet.open) {
      // Kaydırma çubuğu kaybı görünümü genişletip sayfayı titretmesin.
      // Ölçüt gövde genişliği: `clientWidth` gutter ayrıldığında da tam
      // döner, o yüzden yanıltıcı (bkz. csp-delegated.js aynı not).
      var once = Math.round(document.body.getBoundingClientRect().width);
      kok.classList.add("cd-ozet-kilit");
      var fark = Math.round(document.body.getBoundingClientRect().width) - once;
      if (fark > 0) {
        yedekPad = kok.style.paddingRight;
        kok.style.paddingRight = fark + "px";
      }
      if (govde) govde.scrollTop = 0;
    } else {
      kok.classList.remove("cd-ozet-kilit");
      if (yedekPad !== "") {
        kok.style.paddingRight = yedekPad;
        yedekPad = "";
      }
    }
  }

  function kapat() {
    ozet.open = false;
    kilitAc();
  }

  ozet.addEventListener("toggle", kilitAc);

  var kapatBtn = ozet.querySelectorAll("[data-ozet-kapat]");
  for (var i = 0; i < kapatBtn.length; i++) {
    kapatBtn[i].addEventListener("click", function (e) {
      e.preventDefault();
      e.stopPropagation();
      kapat();
    });
  }

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" && ozet.open) kapat();
  });

  // Masaüstüne geçerken kilit kalmasın.
  var degisim = function () { kilitAc(); };
  if (mobil.addEventListener) mobil.addEventListener("change", degisim);
  else if (mobil.addListener) mobil.addListener(degisim);

  kilitAc();
})();
