
(function () {
  "use strict";

  var hero = document.querySelector(".cd-giris");
  if (!hero || !window.requestAnimationFrame) return;

  var tuval = document.createElement("canvas");
  tuval.className = "cd-giris__prizma";
  tuval.setAttribute("aria-hidden", "true");
  var ctx = tuval.getContext && tuval.getContext("2d");
  if (!ctx) return;
  hero.insertBefore(tuval, hero.firstChild);
  hero.classList.add("cd-giris--prizma");

  
  var RENKLER = [
    [120, 200, 255],
    [255, 255, 255],
    [165, 120, 255],
    [52, 230, 200],
    [255, 105, 180],
    [90, 140, 255],
    [150, 255, 235]
  ];

  
  var MX = 0.68;
  var MY = 0.48;
  var YASSILIK = 0.62;
  var IZ_SAYISI = 240;

  var dpr = 1;
  var izler = kur(IZ_SAYISI);
  var calisiyor = false;
  var gorunur = true;
  var sonKare = 0;

  var azHareket = window.matchMedia("(prefers-reduced-motion: reduce)");
  var darEkran = window.matchMedia("(max-width: 900px)");

  function animasyonluMu() {
    return !azHareket.matches && !darEkran.matches;
  }

  function kur(adet) {
    var liste = [];
    for (var i = 0; i < adet; i++) {
      liste.push({
        aci: Math.random() * Math.PI * 2,
        // Sürüklenme yönü ve hızı ize özel: topluca dönen bir disk
        // yerine birbirinin içinden geçen izler istiyoruz.
        hiz: (0.00002 + Math.random() * 0.00006) * (Math.random() < 0.5 ? -1 : 1),
        ic: 0.04 + Math.random() * 0.12,
        dis: 0.42 + Math.random() * 0.8,
        kalinlik: 0.6 + Math.random() * 2.3,
        parlaklik: 0.22 + Math.random() * 0.78,
        renk: RENKLER[(Math.random() * RENKLER.length) | 0],
        faz: Math.random() * Math.PI * 2,
        nabiz: 0.3 + Math.random() * 1.2
      });
    }
    return liste;
  }

  
  function guc() {
    var v = parseFloat(getComputedStyle(hero).getPropertyValue("--cd-giris-isin"));
    return isFinite(v) ? v : 0.6;
  }

  
  function koyuZemin() {
    var bg = getComputedStyle(document.body).backgroundColor || "";
    var m = bg.match(/\d+(\.\d+)?/g);
    if (!m || m.length < 3) return true;
    var luma = 0.2126 * +m[0] + 0.7152 * +m[1] + 0.0722 * +m[2];
    return luma < 128;
  }

  function olcuAyarla() {
    var r = hero.getBoundingClientRect();
    if (r.width < 1 || r.height < 1) return false;
    dpr = Math.min(window.devicePixelRatio || 1, 1.5);
    var w = Math.round(r.width * dpr);
    var h = Math.round(r.height * dpr);
    if (tuval.width !== w || tuval.height !== h) {
      tuval.width = w;
      tuval.height = h;
    }
    return true;
  }

  function ciz(zaman) {
    var W = tuval.width;
    var H = tuval.height;
    var mx = W * MX;
    var my = H * MY;
    var R = Math.sqrt(W * W + H * H) * 0.62;
    var g = guc();
    var hareket = animasyonluMu();
    var koyu = koyuZemin();

    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, W, H);

    // Dikey sıkıştırma merkez etrafında: izler yatay uzansın.
    ctx.setTransform(1, 0, 0, YASSILIK, 0, my * (1 - YASSILIK));
    ctx.globalCompositeOperation = koyu ? "lighter" : "source-over";
    ctx.lineCap = "round";

    for (var i = 0; i < izler.length; i++) {
      var s = izler[i];
      var a = s.aci + (hareket ? zaman * s.hiz : 0);
      var cos = Math.cos(a);
      var sin = Math.sin(a);
      var x0 = mx + cos * R * s.ic;
      var y0 = my + sin * R * s.ic;
      var x1 = mx + cos * R * s.dis;
      var y1 = my + sin * R * s.dis;
      var nabiz = hareket ? 0.72 + 0.28 * Math.sin(zaman * 0.0006 * s.nabiz + s.faz) : 1;
      var alfa = s.parlaklik * g * nabiz;
      if (alfa <= 0.004) continue;

      var grd = ctx.createLinearGradient(x0, y0, x1, y1);
      var c = s.renk;
      // Krem zeminde parlak tonlar (özellikle beyaz iz) görünmez:
      // renk koyultulur, böylece iz zemine karşı okunur.
      if (!koyu) c = [(c[0] * 0.45) | 0, (c[1] * 0.42) | 0, (c[2] * 0.5) | 0];
      var kok = "rgba(" + c[0] + "," + c[1] + "," + c[2] + ",";
      grd.addColorStop(0, kok + "0)");
      grd.addColorStop(0.16, kok + alfa.toFixed(3) + ")");
      grd.addColorStop(0.5, kok + (alfa * 0.7).toFixed(3) + ")");
      grd.addColorStop(1, kok + "0)");
      ctx.strokeStyle = grd;
      ctx.lineWidth = s.kalinlik * dpr;
      ctx.beginPath();
      ctx.moveTo(x0, y0);
      ctx.lineTo(x1, y1);
      ctx.stroke();
    }

    // Sönümler alfadan düşülür (`destination-out`), üste gri bir
    // örtü sermekten farkı: hero'nun zemini olduğu gibi kalıyor.
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.globalCompositeOperation = "destination-out";

    // 1) Metin sütunu: soldan sağa açılan sönüm.
    var sol = ctx.createLinearGradient(0, 0, W * 0.66, 0);
    sol.addColorStop(0, "rgba(0,0,0,0.94)");
    sol.addColorStop(0.5, "rgba(0,0,0,0.5)");
    sol.addColorStop(1, "rgba(0,0,0,0)");
    ctx.fillStyle = sol;
    ctx.fillRect(0, 0, W, H);

    // 2) Kenarlar: izler hero'nun sınırında kesilmiş görünmesin.
    var kenar = ctx.createRadialGradient(mx, my, R * 0.1, mx, my, R * 0.95);
    kenar.addColorStop(0, "rgba(0,0,0,0)");
    kenar.addColorStop(0.68, "rgba(0,0,0,0)");
    kenar.addColorStop(1, "rgba(0,0,0,0.92)");
    ctx.fillStyle = kenar;
    ctx.fillRect(0, 0, W, H);

    ctx.globalCompositeOperation = "source-over";
  }

  function kare(zaman) {
    if (!calisiyor) return;
    // ~30 fps: göz bu yavaşlıktaki sürüklenmede farkı görmüyor,
    // iş yükü yarıya iniyor.
    if (zaman - sonKare >= 32) {
      sonKare = zaman;
      ciz(zaman);
    }
    requestAnimationFrame(kare);
  }

  function basla() {
    if (calisiyor || !gorunur || !animasyonluMu()) return;
    calisiyor = true;
    sonKare = 0;
    requestAnimationFrame(kare);
  }

  function dur() {
    calisiyor = false;
  }

  function yenile() {
    if (!olcuAyarla()) return;
    ciz(performance.now());
    if (animasyonluMu()) basla();
    else dur();
  }

  yenile();

  
  var kaydirmaTimer = null;
  window.addEventListener("scroll", function () {
    if (calisiyor) dur();
    window.clearTimeout(kaydirmaTimer);
    kaydirmaTimer = window.setTimeout(basla, 180);
  }, { passive: true });

  // Hero ekrandan çıkınca çizim durur.
  if (window.IntersectionObserver) {
    new IntersectionObserver(function (girisler) {
      gorunur = girisler[0].isIntersecting;
      if (gorunur) basla();
      else dur();
    }, { threshold: 0 }).observe(hero);
  }

  if (window.ResizeObserver) {
    var zaman = 0;
    new ResizeObserver(function () {
      clearTimeout(zaman);
      zaman = setTimeout(yenile, 120);
    }).observe(hero);
  } else {
    window.addEventListener("resize", yenile);
  }

  // Tema anahtarı `--cd-giris-isin` değerini değiştiriyor; duran
  // kare (dar ekran / az hareket) bunu kendiliğinden görmez.
  if (window.MutationObserver) {
    new MutationObserver(function () {
      if (!calisiyor) ciz(performance.now());
    }).observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
  }

  function mqDinle(mq, fn) {
    if (mq.addEventListener) mq.addEventListener("change", fn);
    else if (mq.addListener) mq.addListener(fn); // Safari 13 ve öncesi
  }
  mqDinle(azHareket, yenile);
  mqDinle(darEkran, yenile);
})();
