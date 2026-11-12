
(function () {
  "use strict";

  var kap = document.querySelector("[data-tepki]");
  if (!kap || !window.fetch) return;

  var id = kap.getAttribute("data-tepki");
  if (!id) return;

  var ANAHTAR = "cd-tepki-" + id;
  var secili = null;
  try {
    secili = window.localStorage.getItem(ANAHTAR);
  } catch (e) {
    // Gizli sekmede localStorage erişimi hata verebilir; tepki yine
    // gönderilir, yalnız "hatırlama" çalışmaz.
  }

  kap.hidden = false;

  var dugmeler = kap.querySelectorAll("[data-yon]");
  var tesekkur = kap.querySelector("[data-tepki-tesekkur]");

  function isaretle() {
    for (var i = 0; i < dugmeler.length; i++) {
      var on = dugmeler[i].getAttribute("data-yon") === secili;
      dugmeler[i].classList.toggle("is-secili", on);
      dugmeler[i].setAttribute("aria-pressed", on ? "true" : "false");
    }
    if (tesekkur) tesekkur.hidden = !secili;
  }

  function gonder(yon) {
    return fetch("/api/kart-tepki", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id: Number(id), yon: yon })
    });
  }

  for (var i = 0; i < dugmeler.length; i++) {
    dugmeler[i].addEventListener("click", function (e) {
      var yon = e.currentTarget.getAttribute("data-yon");
      // Aynı düğmeye ikinci kez basmak seçimi geri alır.
      var yeni = secili === yon ? null : yon;
      secili = yeni;
      isaretle();
      try {
        if (yeni) window.localStorage.setItem(ANAHTAR, yeni);
        else window.localStorage.removeItem(ANAHTAR);
      } catch (err) {  }
      gonder(yeni || "iptal")["catch"](function () {
        // Ağ hatası kullanıcıya gösterilmiyor: tepki kritik bir işlem
        // değil ve hata kutusu makale sonunda gürültü olurdu.
      });
    });
  }

  isaretle();
})();
