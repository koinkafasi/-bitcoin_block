/* gazete-editor.js — Gazete sayfa düzenleyici (yalnız panel)
 *
 * Tuval, baskının kendi sınıflarını (.gazete-sayfa / .gb) gerçek sayfa
 * ölçüsünde kullanır ve transform ile küçültür; blok gövdeleri sunucudan
 * gelir. Böylece editörde görünen ile basılan aynı HTML+CSS'ten gelir.
 * Bkz. specs/13-gazete-sayfa-duzenleyici.md K1.
 *
 * Okuyucu (çevirerek okuma) ayrı dosyada: gazete.js. Genel site editörü
 * indirmez.
 */
(function () {
  "use strict";

  var bootEl = document.getElementById("gz-boot");
  if (!bootEl) return;

  function IKON(ad) {
    return '<svg class="ti" aria-hidden="true"><use href="/static/vendor/tabler-icons.svg?v=b89545a6#ti-' + ad + '"></use></svg>';
  }

  var boot;
  try {
    boot = JSON.parse(bootEl.textContent);
  } catch (e) {
    boot = {};
  }

  /* ===================== Durum ===================== */

  var state = {
    issueId: boot.issueId || 0,
    sayfalar: (boot.sayfalar || []).map(function (p) {
      return {
        no: p.no,
        ad: p.ad || "",
        bloklar: Array.isArray(p.bloklar) ? p.bloklar : [],
        blokHtml: p.blokHtml || {}
      };
    }),
    aktif: 0,
    undo: [],
    redo: [],
    secililer: [],   // çoklu seçim; ilki birincil
    pano: null,      // kopyalanan bloklar
    zoom: 1,         // kullanıcı yakınlaştırması (sığdırmanın üstüne)
    surum: boot.surum || "",  // iyimser kilit damgası
    kirli: false
  };
  if (!state.sayfalar.length) {
    state.sayfalar = [{ no: 1, ad: "Kapak", bloklar: [], blokHtml: {} }];
  }

  var canvas = document.getElementById("gz-canvas");
  var olcekSarmal = document.getElementById("gz-canvas-olcek");
  var layersEl = document.getElementById("gz-layers");
  var propsEl = document.getElementById("gz-props-form");
  var pagesEl = document.getElementById("gz-pages");

  /* ===================== Yardımcılar ===================== */

  function esc(s) {
    return String(s == null ? "" : s).replace(/[&<>"]/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c];
    });
  }
  function clamp(v, min, max) { return Math.max(min, Math.min(max, v)); }
  function sayfa() { return state.sayfalar[state.aktif]; }
  function blokBul(id) {
    return sayfa().bloklar.find(function (b) { return b.id === id; });
  }
  function seciliBloklar() {
    return state.secililer.map(blokBul).filter(Boolean);
  }
  function birincil() { return blokBul(state.secililer[0]); }

  /* ---- mm → yüzde: ızgara şablonda mm cinsinden bildirilir ---- */
  function mmSayi(s) { return parseFloat(String(s || "").replace("mm", "")) || 0; }
  var sayfaWmm = mmSayi(boot.sayfaW) || 210;
  var sayfaHmm = mmSayi(boot.sayfaH) || 297;
  var izgara = boot.izgara || { sutun: 12, oluk: 4, kenar: 12 };

  // Sütun kenarlarının yüzde konumları. Blok koordinatları sayfanın
  // tamamına göre yüzde olduğu için ızgara da öyle hesaplanır.
  function sutunKenarlari() {
    var n = Math.max(1, izgara.sutun || 12);
    var kenar = (mmSayi(izgara.kenar + "mm") || izgara.kenar || 0) / sayfaWmm * 100;
    var oluk = (izgara.oluk || 0) / sayfaWmm * 100;
    var kullanilir = 100 - 2 * kenar;
    var sutunW = (kullanilir - (n - 1) * oluk) / n;
    if (sutunW <= 0) return [];
    var out = [];
    for (var i = 0; i < n; i++) {
      var sol = kenar + i * (sutunW + oluk);
      out.push(sol, sol + sutunW);
    }
    return out;
  }
  var SUTUNLAR = sutunKenarlari();
  var YAKALAMA = 0.8; // yüzde; A4'te ~1.7mm

  /* ===================== Geçmiş ===================== */

  function anlik() {
    state.undo.push(JSON.stringify(state.sayfalar));
    if (state.undo.length > 60) state.undo.shift();
    state.redo.length = 0;
    state.kirli = true;
  }
  function geriAl() {
    if (!state.undo.length) return;
    state.redo.push(JSON.stringify(state.sayfalar));
    state.sayfalar = JSON.parse(state.undo.pop());
    state.secililer = [];
    hepsiniCiz();
  }
  function yinele() {
    if (!state.redo.length) return;
    state.undo.push(JSON.stringify(state.sayfalar));
    state.sayfalar = JSON.parse(state.redo.pop());
    state.secililer = [];
    hepsiniCiz();
  }

  /* ===================== Blok fabrikası ===================== */

  function uid(tip) { return tip + "-" + Math.random().toString(36).slice(2, 8); }

  function enUstZ() {
    return sayfa().bloklar.reduce(function (m, b) {
      return Math.max(m, b.z || 0);
    }, 0);
  }

  function yeniBlok(tip) {
    var b = { id: uid(tip), tip: tip, x: 5, y: 5, w: 40, h: 20, z: enUstZ() + 1 };
    if (tip === "baslik") { b.baslik = "BÖLÜM"; b.h = 6; b.w = 96; b.x = 2; }
    if (tip === "metin") b.metin = "Metin…";
    if (tip === "liste") { b.baslik = "SON HABERLER"; b.kaynak = { tip: "son", limit: 5 }; b.h = 35; }
    if (tip === "manset") { b.kaynak = { tip: "manset", limit: 1 }; b.h = 45; b.w = 90; }
    if (tip === "kart") { b.kaynak = { tip: "son", limit: 2 }; b.h = 25; b.w = 44; }
    if (tip === "reklam") { b.h = 15; b.w = 30; }
    if (tip === "kunye") { b.h = 3; b.w = 96; b.x = 2; b.y = 96; }
    if (tip === "cizgi") { b.h = 1; b.w = 96; b.x = 2; }
    return b;
  }

  /* ===================== Tuval ===================== */

  function sirali(bloklar) {
    return bloklar.slice().sort(function (a, b) {
      return (a.z || 0) - (b.z || 0);
    });
  }

  function tuvaliCiz() {
    canvas.innerHTML = "";
    izgarayiCiz();
    var s = sayfa();
    sirali(s.bloklar).forEach(function (b) {
      var el = document.createElement("div");
      // Sarmalayıcı sınıfı iç gövde sınıflarıyla çarpışmamalı (bkz.
      // gazeteBlockHTML): çarpışma metnin yana akıp kırpılmasına yol açıyordu.
      var g = s.blokHtml[b.id] || {};
      // Stil sinifi ve olcek sunucuda hesaplanir (beyaz liste orada);
      // editor onlari oldugu gibi uygular.
      el.className = "gb gb-t-" + b.tip + (g.sinif ? " " + g.sinif : "") +
        (state.secililer.indexOf(b.id) >= 0 ? " gz-secili" : "") +
        (state.secililer[0] === b.id ? " gz-birincil" : "");
      el.style.cssText = "left:" + b.x + "%;top:" + b.y + "%;width:" + b.w +
        "%;height:" + b.h + "%;z-index:" + (b.z || 0) + ";" + (g.stil || "");
      el.dataset.id = b.id;
      el.innerHTML = g.html || "";
      if (state.secililer.indexOf(b.id) >= 0) {
        ["nw", "n", "ne", "e", "se", "s", "sw", "w"].forEach(function (yon) {
          var h = document.createElement("div");
          h.className = "gz-handle gz-h-" + yon;
          h.dataset.yon = yon;
          el.appendChild(h);
        });
      }
      canvas.appendChild(el);
    });
    olcekle();
    tasmaIsaretle();
    eksikHTMLGetir();
  }

  /* ---- Sütun ızgarası ---- */
  /* ---- Kesim / tasma payi / guvenli alan kilavuzlari (Faz 6) ----
   * Bunlar yalniz EDITORDE gorunur. Dijital nusha tam kesim olcusunde
   * basilir; tasma payi ve kesim isaretleri matbaa dosyasina aittir
   * (SPEC K8), oraya Faz 7'de ekleniyor. */
  function kilavuzKatmani() {
    var k = document.createElement("div");
    k.className = "gz-baski-kilavuz";
    k.setAttribute("aria-hidden", "true");
    ["kesim", "guvenli"].forEach(function (ad) {
      var d = document.createElement("div");
      d.className = "gz-kl gz-kl-" + ad;
      k.appendChild(d);
    });
    canvas.appendChild(k);
  }

  function izgarayiCiz() {
    kilavuzKatmani();
    if (!SUTUNLAR.length) return;
    var g = document.createElement("div");
    g.className = "gz-izgara";
    g.setAttribute("aria-hidden", "true");
    for (var i = 0; i < SUTUNLAR.length; i += 2) {
      var c = document.createElement("div");
      c.className = "gz-izgara-sutun";
      c.style.left = SUTUNLAR[i] + "%";
      c.style.width = (SUTUNLAR[i + 1] - SUTUNLAR[i]) + "%";
      g.appendChild(c);
    }
    canvas.appendChild(g);
  }

  /* ---- Ölçek: tuval gerçek mm ölçüsünde, ekrana sığsın diye küçültülür ---- */
  function sigdirOlcegi() {
    var kap = olcekSarmal && olcekSarmal.parentElement;
    if (!kap || !canvas.offsetWidth) return 1;
    return Math.min(1, Math.max(120, kap.clientWidth - 30) / canvas.offsetWidth);
  }
  function olcekle() {
    if (!olcekSarmal || !canvas || !canvas.offsetWidth) return;
    var k = sigdirOlcegi() * state.zoom;
    canvas.style.transform = "scale(" + k + ")";
    // Tutamak ve çerçeve kalınlığı ekranda sabit kalsın.
    canvas.style.setProperty("--gz-ters", String(1 / k));
    olcekSarmal.style.width = canvas.offsetWidth * k + "px";
    olcekSarmal.style.height = canvas.offsetHeight * k + "px";
    var et = document.getElementById("gz-zoom-etiket");
    if (et) et.textContent = Math.round(k * 100) + "%";
  }
  window.addEventListener("resize", olcekle);

  /* ===================== Taşma işareti ===================== */

  // Ön uyarıdır, otorite değil: editör kullanıcının tarayıcısında, baskı
  // sunucudaki Chromium'da çalışır. Kesin cevap "Sığdır Denetimi"nde.
  function tasmaIsaretle() {
    var sayac = 0;
    canvas.querySelectorAll(".gb").forEach(function (el) {
      var d = el.scrollHeight - el.clientHeight;
      var y = el.scrollWidth - el.clientWidth;
      var tasiyor = d > 1 || y > 1;
      el.classList.toggle("gz-tasma", tasiyor);
      if (tasiyor) sayac++;
    });
    var rozet = document.getElementById("gz-tasma-sayac");
    if (rozet) {
      rozet.textContent = sayac ? sayac + " blok sığmıyor" : "";
      if (sayac) rozet.removeAttribute("hidden");
      else rozet.setAttribute("hidden", "");
    }
    canvas.querySelectorAll("img:not([data-olcum])").forEach(function (im) {
      im.setAttribute("data-olcum", "1");
      im.addEventListener("load", tasmaIsaretle);
    });
  }

  /* ===================== Sunucu gövdeleri ===================== */

  function postForm(url, data) {
    var fd = new FormData();
    Object.keys(data).forEach(function (k) { fd.append(k, data[k]); });
    return fetch(url, { method: "POST", body: fd, credentials: "same-origin" });
  }

  var htmlBekleyen = false;
  function eksikHTMLGetir() {
    if (htmlBekleyen) return;
    var s = sayfa();
    var eksik = s.bloklar.filter(function (b) {
      return b.id && s.blokHtml[b.id] === undefined;
    });
    if (!eksik.length) return;
    // İstenenleri şimdiden işaretle: yanıt gelmezse sonsuz istek olmasın.
    eksik.forEach(function (b) { s.blokHtml[b.id] = ""; });
    htmlBekleyen = true;
    postForm("/admin/gazete/sayilar/" + state.issueId + "/blok-html", {
      bloklar: JSON.stringify(eksik)
    })
      .then(function (r) { return r.json(); })
      .then(function (j) {
        Object.keys(j || {}).forEach(function (id) {
          s.blokHtml[id] = {
            html: j[id].html || "", sinif: j[id].sinif || "", stil: j[id].stil || ""
          };
          if (j[id].icerik) {
            var b = blokBul(id);
            if (b) b.icerik = j[id].icerik;
          }
        });
        htmlBekleyen = false;
        tuvaliCiz();
      })
      .catch(function () { htmlBekleyen = false; });
  }

  function blokYenile(b) {
    if (!b || !b.id) return;
    delete sayfa().blokHtml[b.id];
    eksikHTMLGetir();
  }

  /* ===================== Yakalama ve kılavuzlar ===================== */

  // Hizalama hedefleri: sütun kenarları, sayfa kenarları ve diğer blokların
  // kenarları/merkezleri. Gazete sayfası sütun ürünüdür; keyfi adımla
  // hizalı sayfa üretilemez (SPEC K4).
  function hedefler(hariç) {
    var x = SUTUNLAR.slice();
    var y = [];
    x.push(0, 100);
    y.push(0, 100);
    sayfa().bloklar.forEach(function (b) {
      if (hariç.indexOf(b.id) >= 0) return;
      x.push(b.x, b.x + b.w, b.x + b.w / 2);
      y.push(b.y, b.y + b.h, b.y + b.h / 2);
    });
    return { x: x, y: y };
  }

  function enYakin(deger, liste) {
    var best = null, fark = YAKALAMA;
    liste.forEach(function (t) {
      var d = Math.abs(deger - t);
      if (d < fark) { fark = d; best = t; }
    });
    return best;
  }

  function kilavuzlariCiz(dikey, yatay) {
    canvas.querySelectorAll(".gz-kilavuz").forEach(function (e) { e.remove(); });
    dikey.forEach(function (x) {
      var d = document.createElement("div");
      d.className = "gz-kilavuz gz-kilavuz-d";
      d.style.left = x + "%";
      canvas.appendChild(d);
    });
    yatay.forEach(function (y) {
      var d = document.createElement("div");
      d.className = "gz-kilavuz gz-kilavuz-y";
      d.style.top = y + "%";
      canvas.appendChild(d);
    });
  }
  function kilavuzlariSil() {
    canvas.querySelectorAll(".gz-kilavuz").forEach(function (e) { e.remove(); });
  }

  /* ===================== Sürükleme / boyutlandırma ===================== */

  var surukle = null;

  canvas.addEventListener("pointerdown", function (ev) {
    var el = ev.target.closest(".gb");
    if (!el) {
      state.secililer = [];
      hepsiniCiz();
      return;
    }
    var b = blokBul(el.dataset.id);
    if (!b) return;

    var cokul = ev.shiftKey || ev.ctrlKey || ev.metaKey;
    if (cokul) {
      var i = state.secililer.indexOf(b.id);
      if (i >= 0) state.secililer.splice(i, 1);
      else state.secililer.unshift(b.id);
    } else if (state.secililer.indexOf(b.id) < 0) {
      state.secililer = [b.id];
    } else {
      // Zaten seçili: birincil yap, çoklu seçimi bozma.
      state.secililer = [b.id].concat(state.secililer.filter(function (x) { return x !== b.id; }));
    }

    anlik();
    var rect = canvas.getBoundingClientRect();
    var yon = ev.target.classList.contains("gz-handle") ? ev.target.dataset.yon : "";
    surukle = {
      yon: yon,
      startX: ev.clientX,
      startY: ev.clientY,
      rect: rect,
      ilk: seciliBloklar().map(function (x) {
        return { b: x, x: x.x, y: x.y, w: x.w, h: x.h };
      })
    };
    try { canvas.setPointerCapture(ev.pointerId); } catch (e) { /* yoksay */ }
    hepsiniCiz();
  });

  canvas.addEventListener("pointermove", function (ev) {
    if (!surukle || !surukle.ilk.length) return;
    var dx = ((ev.clientX - surukle.startX) / surukle.rect.width) * 100;
    var dy = ((ev.clientY - surukle.startY) / surukle.rect.height) * 100;
    var serbest = ev.altKey;
    var kilitOran = ev.shiftKey;
    var kimlikler = surukle.ilk.map(function (o) { return o.b.id; });
    var hed = hedefler(kimlikler);
    var kd = [], ky = [];

    surukle.ilk.forEach(function (o) {
      var b = o.b;
      if (surukle.yon) {
        boyutlandir(b, o, surukle.yon, dx, dy, kilitOran);
      } else {
        b.x = clamp(o.x + dx, 0, 100 - o.w);
        b.y = clamp(o.y + dy, 0, 100 - o.h);
      }
      if (!serbest) {
        // Sol/orta/sağ ve üst/orta/alt kenarlardan hangisi yakalarsa.
        var sx = enYakin(b.x, hed.x);
        if (sx === null) {
          var sr = enYakin(b.x + b.w, hed.x);
          if (sr !== null) { b.x = clamp(sr - b.w, 0, 100 - b.w); kd.push(sr); }
        } else { b.x = clamp(sx, 0, 100 - b.w); kd.push(sx); }

        var sy = enYakin(b.y, hed.y);
        if (sy === null) {
          var sa = enYakin(b.y + b.h, hed.y);
          if (sa !== null) { b.y = clamp(sa - b.h, 0, 100 - b.h); ky.push(sa); }
        } else { b.y = clamp(sy, 0, 100 - b.h); ky.push(sy); }
      }
    });

    tuvaliCiz();
    if (kd.length || ky.length) kilavuzlariCiz(kd, ky);
  });

  function boyutlandir(b, o, yon, dx, dy, kilitOran) {
    var sag = o.x + o.w, alt = o.y + o.h;
    if (yon.indexOf("e") >= 0) b.w = clamp(o.w + dx, 2, 100 - o.x);
    if (yon.indexOf("s") >= 0) b.h = clamp(o.h + dy, 2, 100 - o.y);
    if (yon.indexOf("w") >= 0) {
      b.x = clamp(o.x + dx, 0, sag - 2);
      b.w = sag - b.x;
    }
    if (yon.indexOf("n") >= 0) {
      b.y = clamp(o.y + dy, 0, alt - 2);
      b.h = alt - b.y;
    }
    if (kilitOran && o.w > 0 && o.h > 0) {
      // Shift: en-boy oranını koru (köşe tutamaklarında anlamlı).
      var oran = o.h / o.w;
      b.h = clamp(b.w * oran, 2, 100 - b.y);
    }
  }

  function surukleBitti() {
    if (!surukle) return;
    surukle = null;
    kilavuzlariSil();
    hepsiniCiz();
  }
  canvas.addEventListener("pointerup", surukleBitti);
  canvas.addEventListener("pointercancel", surukleBitti);

  /* ===================== Katmanlar ===================== */

  function katmanlariCiz() {
    layersEl.innerHTML = "";
    // Panel sırası boyama sırasıyla aynı olmalı: üstteki blok listede üstte.
    sirali(sayfa().bloklar).reverse().forEach(function (b) {
      var row = document.createElement("div");
      row.className = "gz-layer" +
        (state.secililer.indexOf(b.id) >= 0 ? " secili" : "");
      row.draggable = true;
      row.dataset.id = b.id;
      row.innerHTML = "<span>" + esc(b.tip) +
        (b.baslik ? " · " + esc(b.baslik) : "") + "</span>";

      var ust = document.createElement("button");
      ust.type = "button";
      ust.className = "gz-layer-btn";
      ust.title = "Öne getir";
      ust.innerHTML = IKON("arrow-up");
      ust.addEventListener("click", function (ev) {
        ev.stopPropagation();
        anlik();
        b.z = enUstZ() + 1;
        hepsiniCiz();
      });

      var alt = document.createElement("button");
      alt.type = "button";
      alt.className = "gz-layer-btn";
      alt.title = "Arkaya gönder";
      alt.innerHTML = IKON("arrow-down");
      alt.addEventListener("click", function (ev) {
        ev.stopPropagation();
        anlik();
        var enAlt = sayfa().bloklar.reduce(function (m, x) {
          return Math.min(m, x.z || 0);
        }, 0);
        b.z = enAlt - 1;
        hepsiniCiz();
      });

      var sil = document.createElement("button");
      sil.type = "button";
      sil.className = "gz-layer-btn gz-layer-del";
      sil.title = "Sil";
      sil.innerHTML = IKON("x");
      sil.addEventListener("click", function (ev) {
        ev.stopPropagation();
        blokSil([b.id]);
      });

      row.addEventListener("click", function (ev) {
        if (ev.shiftKey) {
          if (state.secililer.indexOf(b.id) < 0) state.secililer.unshift(b.id);
        } else {
          state.secililer = [b.id];
        }
        hepsiniCiz();
      });

      // Sürükleyerek sıralama: bırakılan satırın z'si hedefe göre ayarlanır.
      row.addEventListener("dragstart", function (ev) {
        ev.dataTransfer.setData("text/plain", b.id);
      });
      row.addEventListener("dragover", function (ev) { ev.preventDefault(); });
      row.addEventListener("drop", function (ev) {
        ev.preventDefault();
        var kaynakId = ev.dataTransfer.getData("text/plain");
        if (!kaynakId || kaynakId === b.id) return;
        var kaynak = blokBul(kaynakId);
        if (!kaynak) return;
        anlik();
        kaynak.z = (b.z || 0) + 1;
        zNormalle();
        hepsiniCiz();
      });

      row.appendChild(ust);
      row.appendChild(alt);
      row.appendChild(sil);
      layersEl.appendChild(row);
    });
  }

  // z değerlerini 0..n-1 aralığına çeker; sıralama korunur.
  function zNormalle() {
    sirali(sayfa().bloklar).forEach(function (b, i) { b.z = i; });
  }

  /* ===================== Özellikler ===================== */

  var STIL_SECENEK = {
    zemin: ["yok", "lacivert", "turuncu", "koyu", "acik", "beyaz"],
    metinRenk: ["koyu", "acik"],
    hiza: ["sol", "orta", "sag"],
    bindirme: ["yok", "alt", "ust", "tam"],
    kenarlik: ["yok", "ince", "kalin"]
  };
  var STIL_ETIKET = {
    zemin: "Zemin", metinRenk: "Metin rengi", hiza: "Hiza",
    bindirme: "Görsel bindirme", kenarlik: "Çerçeve"
  };

  function secimKur(ad, anahtar, deger) {
    var h = '<label class="small">' + ad + '<select class="input gz-stil" data-f="' + anahtar + '">';
    STIL_SECENEK[anahtar].forEach(function (v) {
      h += '<option value="' + v + '"' + (deger === v ? " selected" : "") + ">" + v + "</option>";
    });
    return h + "</select></label>";
  }

  function ozellikleriCiz() {
    var b = birincil();
    if (!b) {
      propsEl.innerHTML = '<p class="hint small">Düzenlemek için tuvalden bir blok seçin.</p>';
      return;
    }
    if (state.secililer.length > 1) {
      propsEl.innerHTML = '<p class="hint small">' + state.secililer.length +
        ' blok seçili. Hizalama ve taşıma çalışır; özellikler birincil bloğa uygulanır.</p>';
    } else {
      propsEl.innerHTML = "";
    }
    var stil = b.stil || {};
    var h =
      '<label class="small">X (%)<input class="input gz-num" data-f="x" type="number" step="0.5" value="' + b.x + '"></label>' +
      '<label class="small">Y (%)<input class="input gz-num" data-f="y" type="number" step="0.5" value="' + b.y + '"></label>' +
      '<label class="small">Genişlik<input class="input gz-num" data-f="w" type="number" step="0.5" value="' + b.w + '"></label>' +
      '<label class="small">Yükseklik<input class="input gz-num" data-f="h" type="number" step="0.5" value="' + b.h + '"></label>' +
      '<label class="small">Başlık<input class="input gz-txt" data-f="baslik" value="' + esc(b.baslik || "") + '"></label>' +
      (b.tip === "metin"
        ? '<label class="small">Metin<textarea class="input gz-txt" data-f="metin" rows="4">' + esc(b.metin || "") + "</textarea></label>"
        : "") +
      '<label class="small">Haber kaynağı<select class="input gz-src" data-f="kaynak.tip">' +
      ["son", "manset", "kategori", "sabit"].map(function (v) {
        var ad = { son: "Son haberler", manset: "Manşet", kategori: "Kategori", sabit: "Sabit içerik" }[v];
        return '<option value="' + v + '"' + (kaynakTip(b) === v ? " selected" : "") + ">" + ad + "</option>";
      }).join("") +
      "</select></label>" +
      '<label class="small">Kategori / içerik ID<input class="input gz-num" data-f="kaynak.id" type="number" value="' + ((b.kaynak && b.kaynak.id) || "") + '"></label>' +
      '<label class="small">Adet<input class="input gz-num" data-f="kaynak.limit" type="number" min="1" max="20" value="' + ((b.kaynak && b.kaynak.limit) || 5) + '"></label>' +
      '<div class="gz-props-ayrac">Görünüm</div>';

    Object.keys(STIL_SECENEK).forEach(function (k) {
      h += secimKur(STIL_ETIKET[k], k, stil[k] || "yok");
    });
    h += '<label class="small">Metin kolonu<input class="input gz-stil-num" data-f="sutun" type="number" min="1" max="4" value="' + (stil.sutun || 1) + '"></label>' +
      '<label class="small">Tipografi ölçeği<input class="input gz-stil-num" data-f="olcek" type="number" min="0.6" max="2" step="0.05" value="' + (stil.olcek || 1) + '"></label>';

    propsEl.insertAdjacentHTML("beforeend", h);

    propsEl.querySelectorAll("input,textarea,select").forEach(function (inp) {
      inp.addEventListener("change", function () {
        anlik();
        var f = inp.dataset.f;
        var v = inp.value;
        if (inp.classList.contains("gz-stil") || inp.classList.contains("gz-stil-num")) {
          b.stil = b.stil || {};
          if (inp.classList.contains("gz-stil-num")) b.stil[f] = Number(v) || 1;
          else if (v === "yok") delete b.stil[f];
          else b.stil[f] = v;
          blokYenile(b);
        } else if (f.indexOf("kaynak.") === 0) {
          b.kaynak = b.kaynak || {};
          var kf = f.slice(7);
          b.kaynak[kf] = kf === "tip" ? v : Number(v) || 0;
          blokYenile(b);
        } else if (f === "baslik" || f === "metin") {
          b[f] = v;
          blokYenile(b);
        } else {
          b[f] = Number(v) || 0;
          if (f === "w" && b.x + b.w > 100) b.w = 100 - b.x;
          if (f === "h" && b.y + b.h > 100) b.h = 100 - b.y;
        }
        hepsiniCiz();
      });
    });
  }
  function kaynakTip(b) { return (b.kaynak && b.kaynak.tip) || "son"; }

  /* ===================== Sayfa şeridi ===================== */

  function sayfalariCiz() {
    pagesEl.innerHTML = "";
    state.sayfalar.forEach(function (p, i) {
      var satir = document.createElement("div");
      satir.className = "gz-page-row" + (i === state.aktif ? " secili" : "");
      satir.draggable = true;
      satir.dataset.i = String(i);

      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "gz-page-btn";
      btn.textContent = p.no + (p.ad ? " · " + p.ad : "");
      btn.addEventListener("click", function () {
        state.aktif = i;
        state.secililer = [];
        hepsiniCiz();
      });
      // Çift tık: adlandır.
      btn.addEventListener("dblclick", function () { sayfaAdlandir(i); });
      satir.appendChild(btn);

      var sil = document.createElement("button");
      sil.type = "button";
      sil.className = "gz-layer-btn gz-layer-del";
      sil.title = "Sayfayı sil";
      sil.innerHTML = IKON("x");
      sil.addEventListener("click", function (ev) {
        ev.stopPropagation();
        sayfaSil(i);
      });
      satir.appendChild(sil);

      // Sürükleyerek sıralama.
      satir.addEventListener("dragstart", function (ev) {
        ev.dataTransfer.setData("text/plain", String(i));
      });
      satir.addEventListener("dragover", function (ev) { ev.preventDefault(); });
      satir.addEventListener("drop", function (ev) {
        ev.preventDefault();
        var kaynak = Number.parseInt(ev.dataTransfer.getData("text/plain"), 10);
        if (isNaN(kaynak) || kaynak === i) return;
        sayfaTasi(kaynak, i);
      });

      pagesEl.appendChild(satir);
    });
  }

  /* ---- Sayfa yönetimi ----
   * Sayfa numaraları her işlemden sonra 1..n olarak yeniden verilir:
   * baskıdaki "Sayfa N" etiketi ve okuyucu sırası buna dayanıyor. */
  function numaralariDuzelt() {
    state.sayfalar.forEach(function (p, i) { p.no = i + 1; });
  }

  function sayfaEkle() {
    anlik();
    state.sayfalar.splice(state.aktif + 1, 0,
      { no: 0, ad: "Yeni sayfa", bloklar: [], blokHtml: {} });
    numaralariDuzelt();
    state.aktif = state.aktif + 1;
    state.secililer = [];
    hepsiniCiz();
    bildir("Sayfa eklendi. Kaydetmeyi unutmayın.", "info");
  }

  function sayfaSil(i) {
    if (state.sayfalar.length <= 1) {
      bildir("Son sayfa silinemez.", "error");
      return;
    }
    var p = state.sayfalar[i];
    if (p.bloklar.length &&
        !window.confirm("Sayfa " + p.no + " ve içindeki " + p.bloklar.length +
                        " blok silinsin mi?")) return;
    anlik();
    state.sayfalar.splice(i, 1);
    numaralariDuzelt();
    state.aktif = clamp(state.aktif, 0, state.sayfalar.length - 1);
    state.secililer = [];
    hepsiniCiz();
  }

  function sayfaTasi(kaynak, hedef) {
    anlik();
    var p = state.sayfalar.splice(kaynak, 1)[0];
    state.sayfalar.splice(hedef, 0, p);
    numaralariDuzelt();
    state.aktif = hedef;
    hepsiniCiz();
  }

  function sayfaAdlandir(i) {
    var ad = window.prompt("Sayfa adı:", state.sayfalar[i].ad || "");
    if (ad === null) return;
    anlik();
    state.sayfalar[i].ad = ad.trim();
    hepsiniCiz();
  }

  function hepsiniCiz() {
    tuvaliCiz();
    katmanlariCiz();
    ozellikleriCiz();
    sayfalariCiz();
  }

  /* ===================== Blok işlemleri ===================== */

  function blokSil(idler) {
    if (!idler.length) return;
    anlik();
    var s = sayfa();
    s.bloklar = s.bloklar.filter(function (b) { return idler.indexOf(b.id) < 0; });
    state.secililer = [];
    hepsiniCiz();
  }

  function cogalt() {
    var secili = seciliBloklar();
    if (!secili.length) return;
    anlik();
    var s = sayfa();
    var yeniler = secili.map(function (b) {
      var k = JSON.parse(JSON.stringify(b));
      k.id = uid(b.tip);
      k.x = clamp(b.x + 2, 0, 100 - b.w);
      k.y = clamp(b.y + 2, 0, 100 - b.h);
      k.z = enUstZ() + 1;
      s.bloklar.push(k);
      // İçerik birebir kopyalandı; gövdeyi sunucudan yeniden istemeye
      // gerek yok, kaynağınkini devral.
      if (s.blokHtml[b.id] !== undefined) s.blokHtml[k.id] = s.blokHtml[b.id];
      return k;
    });
    state.secililer = yeniler.map(function (k) { return k.id; });
    hepsiniCiz();
  }

  function kopyala() {
    var secili = seciliBloklar();
    if (!secili.length) return;
    state.pano = JSON.parse(JSON.stringify(secili));
  }
  function yapistir() {
    if (!state.pano || !state.pano.length) return;
    anlik();
    var s = sayfa();
    var yeniler = state.pano.map(function (b) {
      var k = JSON.parse(JSON.stringify(b));
      k.id = uid(b.tip);
      k.x = clamp(b.x + 3, 0, 100 - b.w);
      k.y = clamp(b.y + 3, 0, 100 - b.h);
      k.z = enUstZ() + 1;
      return k;
    });
    yeniler.forEach(function (k) { s.bloklar.push(k); });
    state.secililer = yeniler.map(function (k) { return k.id; });
    hepsiniCiz();
  }

  /* ===================== Klavye ===================== */

  document.addEventListener("keydown", function (ev) {
    // Hedef her zaman bir element olmayabilir (document'e gonderilen olay);
    // matches yoksa cagirmak tum kisayollari dusuruyordu.
    var t = ev.target;
    if (t && typeof t.matches === "function" && t.matches("input,textarea,select")) return;
    var mod = ev.ctrlKey || ev.metaKey;

    if (mod && ev.key.toLowerCase() === "z") {
      ev.preventDefault();
      if (ev.shiftKey) yinele(); else geriAl();
      return;
    }
    if (mod && ev.key.toLowerCase() === "y") { ev.preventDefault(); yinele(); return; }
    if (mod && ev.key.toLowerCase() === "d") { ev.preventDefault(); cogalt(); return; }
    if (mod && ev.key.toLowerCase() === "c") { kopyala(); return; }
    if (mod && ev.key.toLowerCase() === "v") { ev.preventDefault(); yapistir(); return; }
    if (mod && ev.key.toLowerCase() === "a") {
      ev.preventDefault();
      state.secililer = sayfa().bloklar.map(function (b) { return b.id; });
      hepsiniCiz();
      return;
    }
    if (mod && (ev.key === "0")) { ev.preventDefault(); state.zoom = 1; olcekle(); return; }
    if (mod && (ev.key === "+" || ev.key === "=")) { ev.preventDefault(); zoomla(0.25); return; }
    if (mod && ev.key === "-") { ev.preventDefault(); zoomla(-0.25); return; }

    if (!state.secililer.length) return;
    if (ev.key === "Delete" || ev.key === "Backspace") {
      ev.preventDefault();
      blokSil(state.secililer.slice());
      return;
    }
    if (ev.key.indexOf("Arrow") === 0) {
      ev.preventDefault();
      anlik();
      var adim = ev.shiftKey ? 2 : 0.5;
      seciliBloklar().forEach(function (b) {
        if (ev.key === "ArrowLeft") b.x = clamp(b.x - adim, 0, 100 - b.w);
        if (ev.key === "ArrowRight") b.x = clamp(b.x + adim, 0, 100 - b.w);
        if (ev.key === "ArrowUp") b.y = clamp(b.y - adim, 0, 100 - b.h);
        if (ev.key === "ArrowDown") b.y = clamp(b.y + adim, 0, 100 - b.h);
      });
      hepsiniCiz();
    }
  });

  function zoomla(d) {
    state.zoom = clamp(state.zoom + d, 0.25, 3);
    olcekle();
  }

  /* ===================== Araç çubuğu ===================== */

  function bagla(id, fn) {
    var el = document.getElementById(id);
    if (el) el.addEventListener("click", fn);
  }
  bagla("gz-undo", geriAl);
  bagla("gz-redo", yinele);
  bagla("gz-zoom-in", function () { zoomla(0.25); });
  bagla("gz-zoom-out", function () { zoomla(-0.25); });
  bagla("gz-zoom-sifirla", function () { state.zoom = 1; olcekle(); });
  bagla("gz-sayfa-ekle", sayfaEkle);

  document.querySelectorAll(".gz-add").forEach(function (btn) {
    if (!btn.dataset.tip) return;
    btn.addEventListener("click", function () {
      anlik();
      var s = sayfa();
      var b = yeniBlok(btn.dataset.tip);
      b.y = Math.min(90, 5 + s.bloklar.length * 3);
      s.bloklar.push(b);
      state.secililer = [b.id];
      hepsiniCiz();
    });
  });

  /* ---- Kaydet ---- */
  var saveBtn = document.getElementById("gz-kaydet");
  if (saveBtn) {
    saveBtn.addEventListener("click", function () { kaydet(saveBtn); });
  }

  function kaydet(btn) {
    if (btn) { btn.disabled = true; btn.textContent = "Kaydediliyor…"; }
    return postForm("/admin/gazete/sayilar/" + state.issueId + "/sayfalar", {
      sayfalar: JSON.stringify(state.sayfalar.map(function (p) {
        return { no: p.no, ad: p.ad, bloklar: p.bloklar };
      })),
      surum: state.surum
    })
      .then(function (r) {
        if (r.status === 409) {
          // Baska bir yerde kaydedilmis: sessizce ezme, kullaniciya soyle.
          return r.json().then(function (j) { throw new Error(j.mesaj || "Cakisma"); });
        }
        if (!r.ok) return r.text().then(function (t) { throw new Error(t || r.status); });
        return r.json();
      })
      .then(function (j) {
        if (j && j.surum) state.surum = j.surum;
        state.kirli = false;
        if (btn) {
          btn.innerHTML = IKON("check") + "Kaydedildi";
          setTimeout(function () {
            btn.disabled = false;
            btn.innerHTML = IKON("device-floppy") + "Kaydet";
          }, 1500);
        }
      })
      .catch(function (e) {
        if (btn) {
          btn.innerHTML = IKON("x") + "Hata";
          btn.disabled = false;
        }
        bildir("Kaydedilemedi: " + (e && e.message ? e.message : "bilinmeyen hata"), "error");
      });
  }

  // Kaydedilmemiş değişiklikle sekmeyi kapatmak sessiz veri kaybıdır.
  window.addEventListener("beforeunload", function (ev) {
    if (!state.kirli) return;
    ev.preventDefault();
    ev.returnValue = "";
  });

  /* ---- Bildirim (panel içi; alert yerine) ---- */
  // bildirHTML: icinde baglanti olan bildirim. Metin sunucudan gelmez,
  // burada kuruluyor; kullanici verisi enjekte edilmiyor.
  function bildirHTML(html, tur) {
    var kap = document.getElementById("gz-bildirim");
    if (!kap) return;
    kap.className = "gz-bildirim " + (tur || "info");
    kap.innerHTML = html;
    kap.removeAttribute("hidden");
    clearTimeout(bildir._t);
    bildir._t = setTimeout(function () { kap.setAttribute("hidden", ""); }, 12000);
  }

  function bildir(mesaj, tur) {
    var kap = document.getElementById("gz-bildirim");
    if (!kap) return;
    kap.className = "gz-bildirim " + (tur || "info");
    kap.textContent = mesaj;
    kap.removeAttribute("hidden");
    clearTimeout(bildir._t);
    bildir._t = setTimeout(function () { kap.setAttribute("hidden", ""); }, 8000);
  }

  /* ---- Şablon ---- */
  var sablonSel = document.getElementById("gz-sablonlar");
  if (sablonSel && boot.sablonlar) {
    boot.sablonlar.forEach(function (t) {
      var o = document.createElement("option");
      o.value = t.id;
      o.textContent = t.ad;
      sablonSel.appendChild(o);
    });
  }
  bagla("gz-sablon-uygula", function () {
    var id = sablonSel && sablonSel.value;
    if (!id) { bildir("Önce bir şablon seçin.", "error"); return; }
    if (!window.confirm("Seçili şablon uygulansın mı? Kaydedilmemiş değişiklikler sayfaya yazılır.")) return;
    fetch("/admin/gazete/sablonlar/" + id, { credentials: "same-origin" })
      .then(function (r) { return r.json(); })
      .then(function (j) {
        if (!j.sayfalar || !j.sayfalar.length) { bildir("Şablon boş.", "error"); return; }
        anlik();
        state.sayfalar = j.sayfalar.map(function (p) {
          return {
            no: p.no, ad: p.ad || "",
            bloklar: Array.isArray(p.bloklar) ? p.bloklar : [],
            blokHtml: {}
          };
        });
        state.aktif = 0;
        state.secililer = [];
        hepsiniCiz();
        bildir("Şablon uygulandı: " + (j.ad || ""), "info");
      })
      .catch(function () { bildir("Şablon yüklenemedi.", "error"); });
  });

  bagla("gz-sablon", function () {
    var ad = window.prompt("Şablon adı:");
    if (!ad) return;
    postForm("/admin/gazete/sayilar/" + state.issueId + "/sablona-kaydet", {
      ad: ad, aciklama: "", varsayilan: "0"
    }).then(function () { bildir("Şablon kaydedildi: " + ad, "info"); });
  });

  /* ---- Doldur / doğrula / PDF / önizleme ---- */
  bagla("gz-doldur", function () {
    if (!window.confirm("Sayfalar otomatik doldurulsun mu? Elle düzenlediğiniz sayfalar korunur.")) return;
    postForm("/admin/gazete/sayilar/" + state.issueId + "/doldur", {}).then(function () {
      location.reload();
    });
  });

  var dogrulaBtn = document.getElementById("gz-dogrula");
  if (dogrulaBtn) {
    dogrulaBtn.addEventListener("click", function () {
      dogrulaBtn.disabled = true;
      var eskiIc = dogrulaBtn.innerHTML;
      dogrulaBtn.textContent = "Ölçülüyor…";
      postForm("/admin/gazete/sayilar/" + state.issueId + "/dogrula", {})
        .then(function (r) { return r.json(); })
        .then(function (j) {
          dogrulaBtn.disabled = false;
          dogrulaBtn.innerHTML = eskiIc;
          if (!j.olculdu) { bildir("Ölçüm yapılamadı: " + (j.hata || ""), "error"); return; }
          var t = j.tasmalar || [];
          if (!t.length) { bildir("Tüm bloklar sayfaya sığıyor. Baskıya hazır.", "ok"); return; }
          bildir(j.ozet || (t.length + " blok sığmıyor."), "error");
        })
        .catch(function () {
          dogrulaBtn.disabled = false;
          dogrulaBtn.innerHTML = eskiIc;
          bildir("Ölçüm isteği başarısız.", "error");
        });
    });
  }

  var pdfBtn = document.getElementById("gz-pdf");
  if (pdfBtn) {
    pdfBtn.addEventListener("click", function () {
      pdfBtn.disabled = true;
      pdfBtn.textContent = "Üretiliyor…";
      // Taşma geçidi sunucuda; burada yalnız sonucu bildiriyoruz.
      postForm("/admin/gazete/sayilar/" + state.issueId + "/pdf", {})
        .then(function (r) {
          pdfBtn.disabled = false;
          pdfBtn.innerHTML = IKON("file-type-pdf") + "PDF Üret";
          if (!r.ok) { bildir("PDF üretilemedi.", "error"); return; }
          var indir = "/gazete/" + (boot.issueDate || "") + "/pdf";
          bildirHTML('PDF üretildi. <a href="' + indir +
            '" target="_blank" rel="noopener">İndir</a>', "ok");
        })
        .catch(function () {
          pdfBtn.disabled = false;
          pdfBtn.innerHTML = IKON("file-type-pdf") + "PDF Üret";
          bildir("PDF üretilemedi.", "error");
        });
    });
  }

  /* ---- Matbaa dosyasi (SPEC K8 / Faz 7) ----
   * Dijital nushaya dokunmaz; ayri bir dosya uretir ve preflight raporunu
   * gosterir. Rapor temiz degilse dosya matbaaya gonderilmemeli. */
  var matbaaBtn = document.getElementById("gz-matbaa");
  if (matbaaBtn) {
    matbaaBtn.addEventListener("click", function () {
      matbaaBtn.disabled = true;
      var eskiIc = matbaaBtn.innerHTML;
      matbaaBtn.textContent = "Üretiliyor…";
      postForm("/admin/gazete/sayilar/" + state.issueId + "/matbaa", {})
        .then(function (r) { return r.json(); })
        .then(function (j) {
          matbaaBtn.disabled = false;
          matbaaBtn.innerHTML = eskiIc;
          if (!j.ok) { bildir("Matbaa dosyası üretilemedi: " + (j.hata || ""), "error"); return; }
          var rp = j.rapor || {};
          var ozet = "Matbaa dosyası üretildi — " + (rp.pdfx || "") +
            ", " + (rp.sayfa || 0) + " sayfa, " + (rp.sayfaBoyu || "");
          if (j.hazir) {
            bildir(ozet + ". Preflight temiz.", "ok");
          } else {
            bildir(ozet + ". Preflight uyarıları: " +
              (rp.uyarilar || []).join(" · "), "error");
          }
        })
        .catch(function () {
          matbaaBtn.disabled = false;
          matbaaBtn.innerHTML = eskiIc;
          bildir("Matbaa isteği başarısız.", "error");
        });
    });
  }

  bagla("gz-onizleme", function () {
    var acik = document.body.classList.toggle("gz-preview");
    // Onizlemede secim ve kilavuzlar kapanir: sayfa basilacagi gibi gorunsun.
    if (acik) state.secililer = [];
    canvas.classList.toggle("gz-kilavuz-gizli", acik);
    hepsiniCiz();
  });
  bagla("gz-izgara-ac", function () {
    canvas.classList.toggle("gz-izgara-gizli");
  });

  hepsiniCiz();
})();
