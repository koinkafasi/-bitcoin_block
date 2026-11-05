/* =====================================================================
   Üye girişi / kayıt — huddle davranışı (Faz 198)
   =====================================================================
   Tek kaynak: `.auth[data-mood]`. Bu betik yalnız o özniteliği ve form
   durumlarını yönetir; çizim tamamen CSS'te.

   Durumlar: idle → nosy (e-posta) → shy (şifre) → exposed (şifre açık)
   + busy (gönderiliyor) + wrong (sunucu reddi). "win" yok: giriş başarılı
   olunca sayfa sunucudan /hesabim'e yönlenir.

   Fare: yalnız gerçek fare (pointerType === "mouse") izlenir; dokunmatikte
   bakış sabit kalır. Hareket azaltma açıksa izleme ve nabız kapalıdır.
   ===================================================================== */
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
    /* Kayıt formunda `#login` yok; karşılaştırmayı alanın kendisi üzerinden
       yapmak `null === null` yüzünden kayıt sayfasını sürekli "nosy"
       bırakıyordu (ölçüldü). */
    if (state.focus && state.focus.type === "password") return "shy";
    if (state.focus) return "nosy";
    return "idle";
  }

  function render() {
    var mood = moodOf();
    if (auth.dataset.mood !== mood) auth.dataset.mood = mood;
    schedulePeek(mood === "shy");
  }

  /* --- utangaçken bir tanesi dayanamaz, bakar ------------------------ */
  var peekTimer = null;
  var peekOff = null;

  function schedulePeek(active) {
    var ink = auth.querySelector('[data-mate="ink"]');
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

  /* --- fareyi izleyen bakış ------------------------------------------ */
  var box = null;
  var gaze = { x: 0, y: 0, tx: 0, ty: 0 };
  var frame = null;

  function measure() {
    if (huddle) box = huddle.getBoundingClientRect();
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
      if (e.pointerType === "mouse") lookAt(e.clientX, e.clientY);
    }, { passive: true });
    window.addEventListener("resize", measure, { passive: true });
    window.addEventListener("scroll", measure, { passive: true });
    window.addEventListener("load", measure);
    measure();
  }

  /* --- alan hataları -------------------------------------------------- */
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

  function refuse(el, msg) {
    setError(el, msg);
    state.refused = true;
    render();
    say(msg);
    if (el) {
      el.focus({ preventScroll: true });
      if (el === passwordEl && el.value) el.select();
    }
  }

  /* Form alanları: odak durumu karakterlerin tek girdisi. Kayıt formunda
     ad/e-posta da grubu meraklandırır (yalnız giriş alanı değil). */
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
      if (errBox(el.name) && errBox(el.name).hasAttribute("data-shown")) setError(el, "");
      render();
    });
  });

  if (loginEl) {
    loginEl.addEventListener("input", function () {
      lookAtEl(loginEl, clamp(loginEl.value.length / 26, 0, 1) * 0.9 + 0.05);
    });
  }

  /* --- Caps Lock ------------------------------------------------------ */
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

  /* --- şifreyi göster ------------------------------------------------- */
  if (peek && passwordEl) {
    peek.addEventListener("click", function () {
      state.revealed = !state.revealed;
      var sel = { start: passwordEl.selectionStart, end: passwordEl.selectionEnd };
      passwordEl.type = state.revealed ? "text" : "password";
      try {
        passwordEl.setSelectionRange(sel.start, sel.end);
      } catch (e) {
        /* tür değişimi seçimi sıfırlayabilir; zararsız */
      }
      peek.setAttribute("aria-pressed", String(state.revealed));
      if (peekLabel) peekLabel.textContent = state.revealed ? "Şifreyi gizle" : "Şifreyi göster";
      say(state.revealed ? "Şifre görünür." : "Şifre gizli.");
      passwordEl.focus({ preventScroll: true });
      render();
    });
  }

  /* --- gönderim (yalnız giriş formu) ---------------------------------- */
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

  /* Kayıt formu: tarayıcı doğrulaması geçtiyse düğme "gönderiliyor" olur. */
  if (form) {
    form.addEventListener("submit", function (e) {
      if (e.defaultPrevented) return;
      if (cta) cta.setAttribute("data-busy", "");
      say("Gönderiliyor…");
    });
  }

  /* Geri dönüşte (bfcache) düğme "gönderiliyor" durumunda kalmasın. */
  window.addEventListener("pageshow", function () {
    state.busy = false;
    if (cta) cta.removeAttribute("data-busy");
    render();
  });

  /* --- açılış --------------------------------------------------------- */
  // Sunucudan hata ile gelindiyse karakterler "wrong" kalır (sarsıntı bir kez
  // oynar); kullanıcı alana yazmaya başlayınca sakinleşirler. Zamanlayıcıyla
  // 1,4 sn sonra gülümsemeye dönmek, hata kutusu ekranda dururken yanlış
  // mesaj veriyordu.
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
