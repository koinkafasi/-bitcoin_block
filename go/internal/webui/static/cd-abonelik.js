(function () {
  "use strict";

  var cycle = "monthly";

  function uygula() {
    var aylik = document.querySelectorAll("[data-monthly]");
    var yillik = document.querySelectorAll("[data-annual]");
    var i;
    for (i = 0; i < aylik.length; i++) aylik[i].hidden = cycle !== "monthly";
    for (i = 0; i < yillik.length; i++) yillik[i].hidden = cycle !== "annual";

    var btn = document.querySelectorAll("[data-cycle]");
    for (i = 0; i < btn.length; i++) {
      var secili = btn[i].getAttribute("data-cycle") === cycle;
      btn[i].classList.toggle("is-secili", secili);
      btn[i].setAttribute("aria-pressed", secili ? "true" : "false");
    }
    if (seciliBtn) ozetGuncelle();
  }

  var butonlar = document.querySelectorAll("[data-cycle]");
  for (var b = 0; b < butonlar.length; b++) {
    butonlar[b].addEventListener("click", function (e) {
      cycle = e.currentTarget.getAttribute("data-cycle");
      uygula();
    });
  }

  var form = document.getElementById("abonForm");
  var ozet = document.getElementById("abonSummary");
  var seciliBtn = null;

  function ozetGuncelle() {
    if (!form || !seciliBtn) return;
    var fiyat = cycle === "annual"
      ? seciliBtn.getAttribute("data-annual")
      : seciliBtn.getAttribute("data-monthly");
    if (ozet) {
      ozet.textContent = seciliBtn.getAttribute("data-tier-name") + " — " + fiyat +
        " ₺ (" + (cycle === "annual" ? "yıllık" : "aylık") + ")";
    }
    var alan = document.getElementById("abonCycle");
    if (alan) alan.value = cycle;
  }

  function kapat() {
    if (form) form.hidden = true;
  }

  var kartlar = document.querySelectorAll("[data-tier]");
  for (var k = 0; k < kartlar.length; k++) {
    kartlar[k].addEventListener("click", function (e) {
      if (!form) return;
      seciliBtn = e.currentTarget;
      var alan = document.getElementById("abonTier");
      if (alan) alan.value = seciliBtn.getAttribute("data-tier");
      ozetGuncelle();
      form.hidden = false;
      var ilk = form.querySelector('input[name="tckn"]');
      if (ilk) ilk.focus();
    });
  }

  var kapatBtn = document.querySelectorAll("[data-abon-close]");
  for (var c = 0; c < kapatBtn.length; c++) {
    kapatBtn[c].addEventListener("click", kapat);
  }
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") kapat();
  });

  uygula();
})();
