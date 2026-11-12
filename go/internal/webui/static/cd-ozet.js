(function () {
  "use strict";

  var ozet = document.querySelector("[data-ozet]");
  if (!ozet) return;

  var govde = ozet.querySelector(".cd-ozet__govde");
  var perde = ozet.querySelector("[data-ozet-kapat]");
  var mobil = window.matchMedia("(max-width: 767px)");

  function kilitAc() {
    if (mobil.matches && ozet.open) {
      document.documentElement.classList.add("cd-ozet-kilit");
      if (govde) govde.scrollTop = 0;
    } else {
      document.documentElement.classList.remove("cd-ozet-kilit");
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
