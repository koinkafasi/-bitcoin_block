
(function () {
  "use strict";

  var auth = document.querySelector("[data-auth]");
  if (!auth) return;

  var huddle = auth.querySelector("[data-huddle]");
  var shake = auth.querySelector("[data-shake]");
  var loginEl = document.getElementById("login");
  var passwordEl = document.getElementById("password");
  var peek = auth.querySelector("[data-peek]");
  var peekLabel = auth.querySelector("[data-peek-label]");
  var capsNote = auth.querySelector("[data-caps]");
  var cta = auth.querySelector("[data-cta]");
  var live = document.querySelector("[data-live]");
  var form = auth.querySelector("[data-form]");

  var lessMotion = window.matchMedia("(prefers-reduced-motion: reduce)");

  var state = {
    focus: null,
    revealed: false,
    busy: false,
    refused: auth.dataset.mood === "wrong"
  };

  function say(msg) {
    if (live) live.textContent = msg;
  }

  function moodOf() {
    if (state.busy) return "busy";
    if (state.refused) return "wrong";
    if (state.revealed && passwordEl && passwordEl.value) return "exposed";
    
    if (state.focus && state.focus.type === "password") return "shy";
    if (state.focus) return "nosy";
    return "idle";
  }

  function render() {
    var mood = moodOf();
    if (auth.dataset.mood !== mood) auth.dataset.mood = mood;
    schedulePeek(mood === "shy");
  }

  
  var peekTimer = null;
  var peekOff = null;

  function schedulePeek(active) {
    var ink = auth.querySelector('[data-mate="eth"]');
    if (!ink) return;
    if (!active || lessMotion.matches) {
      window.clearTimeout(peekTimer);
      window.clearTimeout(peekOff);
      peekTimer = peekOff = null;
      ink.classList.remove("is-peeking");
      return;
    }
    if (peekTimer) return;
    var cycle = function () {
      ink.classList.add("is-peeking");
      peekOff = window.setTimeout(function () {
        ink.classList.remove("is-peeking");
      }, 780);
      peekTimer = window.setTimeout(cycle, 3200);
    };
    peekTimer = window.setTimeout(cycle, 1400);
  }

  
  var box = null;
  var gaze = { x: 0, y: 0, tx: 0, ty: 0 };
  var frame = null;

  function measure() {
    if (huddle) box = huddle.getBoundingClientRect();
    matesOlc();
  }

  
  var mates = [];

  function matesOlc() {
    var list = auth.querySelectorAll("[data-mate]");
    if (!mates.length) {
      for (var j = 0; j < list.length; j++) {
        mates.push({ el: list[j], cx: 0, cy: 0, yaricap: 1, near: 0, spinAt: 0 });
      }
    }
    for (var i = 0; i < mates.length; i++) {
      var r = mates[i].el.getBoundingClientRect();
      if (!r.width) continue;
      mates[i].cx = r.left + r.width / 2;
      mates[i].cy = r.top + r.height / 2;
      // Etki yarıçapı kutudan biraz geniş: fare yaklaşırken tepki
      // kutunun kenarında birden başlamasın.
      mates[i].yaricap = Math.max(r.width, r.height) * 0.78;
    }
    olcumZamani = Date.now();
  }

  
  var olcumZamani = 0;

  function yakinlik(x, y) {
    if (Date.now() - olcumZamani > 500) matesOlc();
    for (var i = 0; i < mates.length; i++) {
      var m = mates[i];
      var dx = x - m.cx;
      var dy = y - m.cy;
      var n = clamp(1 - Math.sqrt(dx * dx + dy * dy) / m.yaricap, 0, 1);
      if (Math.abs(n - m.near) > 0.012) {
        m.near = n;
        m.el.style.setProperty("--near", n.toFixed(2));
      }
      if (n > 0.88) cevir(m);
    }
  }

  function cevir(m) {
    var now = Date.now();
    if (now - m.spinAt < 2800) return;
    m.spinAt = now;
    m.el.classList.add("is-spinning");
    window.setTimeout(function () {
      m.el.classList.remove("is-spinning");
    }, 960);
  }

  function yakinligiBosalt() {
    for (var i = 0; i < mates.length; i++) {
      mates[i].near = 0;
      mates[i].el.style.setProperty("--near", "0");
    }
  }

  function clamp(v, min, max) {
    return Math.min(max, Math.max(min, v));
  }

  function pump() {
    gaze.x += (gaze.tx - gaze.x) * 0.15;
    gaze.y += (gaze.ty - gaze.y) * 0.15;
    var settled = Math.abs(gaze.tx - gaze.x) + Math.abs(gaze.ty - gaze.y) < 0.003;
    if (settled) {
      gaze.x = gaze.tx;
      gaze.y = gaze.ty;
    }
    if (huddle) {
      huddle.style.setProperty("--px", gaze.x.toFixed(3));
      huddle.style.setProperty("--py", gaze.y.toFixed(3));
    }
    frame = settled ? null : window.requestAnimationFrame(pump);
  }

  function lookAt(clientX, clientY) {
    if (lessMotion.matches || !huddle) return;
    if (!box || !box.width) measure();
    if (!box || !box.width) return;
    var cx = box.left + box.width * 0.55;
    var cy = box.top + box.height * 0.4;
    gaze.tx = clamp((clientX - cx) / (box.width * 0.85), -1, 1);
    gaze.ty = clamp((clientY - cy) / (box.height * 0.85), -1, 1);
    if (!frame) frame = window.requestAnimationFrame(pump);
  }

  function lookAtEl(el, ratio) {
    if (!el) return;
    var r = el.getBoundingClientRect();
    lookAt(r.left + r.width * clamp(typeof ratio === "number" ? ratio : 0, 0, 1), r.top + r.height * 0.5);
  }

  if (!lessMotion.matches) {
    window.addEventListener("pointermove", function (e) {
      if (e.pointerType !== "mouse") return;
      lookAt(e.clientX, e.clientY);
      yakinlik(e.clientX, e.clientY);
    }, { passive: true });
    // Fare pencereden çıkınca kahramanlar normal boyutlarına döner.
    document.addEventListener("pointerleave", yakinligiBosalt);
    window.addEventListener("blur", yakinligiBosalt);
    window.addEventListener("resize", measure, { passive: true });
    window.addEventListener("scroll", measure, { passive: true });
    window.addEventListener("load", measure);
    measure();
  }

  
  function errBox(name) {
    return auth.querySelector('[data-err="' + name + '"]');
  }

  function fieldOf(el) {
    return el ? el.closest(".afield") : null;
  }

  function setError(el, msg) {
    if (!el) return;
    var box = errBox(el.name);
    var field = fieldOf(el);
    if (!box || !field) return;
    var text = box.querySelector("[data-err-text]");
    if (msg) {
      if (text) text.textContent = msg;
      box.setAttribute("data-shown", "");
      field.setAttribute("data-invalid", "");
      el.setAttribute("aria-invalid", "true");
    } else {
      box.removeAttribute("data-shown");
      field.removeAttribute("data-invalid");
      el.removeAttribute("aria-invalid");
    }
  }

  function clearErrors() {
    setError(loginEl, "");
    setError(passwordEl, "");
  }

  
  var pozTimer = null;

  function pozYaz(metin) {
    var etiketler = auth.querySelectorAll("[data-kur-poz]");
    for (var i = 0; i < etiketler.length; i++) etiketler[i].textContent = metin;
  }

  function pozisyon(tip) {
    window.clearTimeout(pozTimer);
    if (!tip) {
      auth.removeAttribute("data-pos");
      pozYaz("");
      return;
    }
    auth.dataset.pos = tip;
    pozYaz(tip === "long" ? "LONG" : "SHORT");
    if (tip === "short") {
      // Isınma bir an sürer; sonra kahramanlar sakin hâline döner.
      pozTimer = window.setTimeout(function () { pozisyon(null); }, 2800);
    }
  }

  function pozToparla() {
    if (auth.dataset.pos === "long") pozisyon("short");
  }

  function refuse(el, msg) {
    setError(el, msg);
    state.refused = true;
    pozisyon("long");
    render();
    say(msg);
    if (el) {
      el.focus({ preventScroll: true });
      if (el === passwordEl && el.value) el.select();
    }
  }

  
  auth.querySelectorAll(".afield input").forEach(function (el) {
    el.addEventListener("focus", function () {
      state.focus = el;
      if (el === loginEl) lookAtEl(el, 0.12);
      render();
    });
    el.addEventListener("blur", function () {
      if (state.focus === el) state.focus = null;
      render();
    });
    el.addEventListener("input", function () {
      if (state.refused) state.refused = false;
      // Düzeltmeye başlamak pozisyonu çevirir: buz çözülür, ısınırlar.
      pozToparla();
      if (errBox(el.name) && errBox(el.name).hasAttribute("data-shown")) setError(el, "");
      render();
    });
  });

  if (loginEl) {
    loginEl.addEventListener("input", function () {
      lookAtEl(loginEl, clamp(loginEl.value.length / 26, 0, 1) * 0.9 + 0.05);
    });
  }

  
  var capsOn = false;
  function capsCheck(e) {
    var on = typeof e.getModifierState === "function" && e.getModifierState("CapsLock");
    if (on === capsOn) return;
    capsOn = on;
    if (capsNote) capsNote.hidden = !on;
    if (on) say("Caps Lock açık.");
  }
  if (passwordEl) {
    passwordEl.addEventListener("keydown", capsCheck);
    passwordEl.addEventListener("keyup", capsCheck);
    passwordEl.addEventListener("blur", function () {
      if (capsNote) capsNote.hidden = true;
    });
    passwordEl.addEventListener("focus", function () {
      if (capsNote) capsNote.hidden = !capsOn;
    });
  }

  
  if (peek && passwordEl) {
    peek.addEventListener("click", function () {
      state.revealed = !state.revealed;
      var sel = { start: passwordEl.selectionStart, end: passwordEl.selectionEnd };
      passwordEl.type = state.revealed ? "text" : "password";
      try {
        passwordEl.setSelectionRange(sel.start, sel.end);
      } catch (e) {
        
      }
      peek.setAttribute("aria-pressed", String(state.revealed));
      if (peekLabel) peekLabel.textContent = state.revealed ? "Şifreyi gizle" : "Şifreyi göster";
      say(state.revealed ? "Şifre görünür." : "Şifre gizli.");
      passwordEl.focus({ preventScroll: true });
      render();
    });
  }

  
  if (form && form.getAttribute("data-form") === "giris") {
    form.addEventListener("submit", function (e) {
      if (state.busy) {
        e.preventDefault();
        return;
      }
      clearErrors();

      var login = loginEl ? loginEl.value.trim() : "";
      var pass = passwordEl ? passwordEl.value : "";

      if (!login) {
        e.preventDefault();
        refuse(loginEl, "E-posta veya kullanıcı adı gerekli.");
        return;
      }
      if (!pass) {
        e.preventDefault();
        refuse(passwordEl, "Şifre gerekli.");
        return;
      }

      state.busy = true;
      if (cta) cta.setAttribute("data-busy", "");
      render();
      lookAtEl(cta, 0.5);
      say("Giriş bilgileri kontrol ediliyor…");
    });
  }

  
  if (form) {
    form.addEventListener("submit", function (e) {
      if (e.defaultPrevented) return;
      if (cta) cta.setAttribute("data-busy", "");
      say("Gönderiliyor…");
    });
  }

  
  window.addEventListener("pageshow", function () {
    state.busy = false;
    if (cta) cta.removeAttribute("data-busy");
    render();
  });

  
  // Sunucudan hata ile gelindiyse karakterler "wrong" kalır (sarsıntı bir kez
  // oynar); kullanıcı alana yazmaya başlayınca sakinleşirler. Zamanlayıcıyla
  // 1,4 sn sonra gülümsemeye dönmek, hata kutusu ekranda dururken yanlış
  // mesaj veriyordu.
  if (auth.dataset.mood === "wrong") {
    // Sunucu reddiyle gelindi: kur rozetleri açılır, pozisyon LONG.
    // Rozetler gelişin (auth-arrive) bitmesini bekler, yoksa aşağıdan
    // süzülen kahramanların üstünde havada asılı kalıyorlar.
    window.setTimeout(function () { pozisyon("long"); }, 900);
  }
  if (loginEl && auth.dataset.mood === "wrong") {
    loginEl.focus({ preventScroll: true });
    if (loginEl.value) loginEl.select();
  }
  // `autofocus` bizim dinleyicilerimiz bağlanmadan önce çalışıyor; ilk
  // karede odaktaki alanı yok sayarsak karakterler e-posta yazılırken bile
  // "idle" kalıyordu (ölçüldü).
  var ilkOdak = document.activeElement;
  if (ilkOdak && ilkOdak.closest && ilkOdak.closest(".afield")) {
    state.focus = ilkOdak;
  }
  render();
})();
