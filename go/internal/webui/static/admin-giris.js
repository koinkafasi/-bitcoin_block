
(function () {
  'use strict';

  var $ = function (id) { return document.getElementById(id); };

  var cat = $('cat');
  var card = $('card');
  var form = $('form');
  var email = $('email');
  var pw = $('pw');
  var peekBtn = $('peek');
  var statusEl = $('status');
  var bubble = $('bubble');
  var bubbleText = $('bubbleText');
  var joy = $('joy');
  var rail = $('rail');
  var sheen = $('sheen');
  var caps = $('caps');
  var petBtn = $('pet');
  var submitBtn = $('submit');
  var srvFlash = $('srvFlash');

  if (!cat || !form) return;

  var RM = matchMedia('(prefers-reduced-motion: reduce)');
  var SVG_NS = 'http://www.w3.org/2000/svg';

  var EYE = { l: { x: 124, y: 96 }, r: { x: 176, y: 96 } };
  var SHOULDER = { l: { x: 104, y: 178 }, r: { x: 196, y: 178 } };
  var ARM_LEN = 78;
  var NECK = { x: 150, y: 152 };
  var TAIL_BASE = { x: 204, y: 244 };
  var MOUTH = {
    neutral: 'M150,131 q-9,12 -18,2 M150,131 q9,12 18,2',
    small: 'M150,132 q-7,9 -14,1 M150,132 q7,9 14,1',
    sad: 'M133,142 q8,-11 17,-2 q9,9 17,-2',
    happy: 'M129,129 q21,25 42,0'
  };

  
  function aimArm(from, tx, ty, side) {
    var dx = tx - from.x, dy = ty - from.y;
    var len = Math.hypot(dx, dy);
    var a = Math.atan2(-dx, dy) * 180 / Math.PI;
    if (side < 0 && a > 0) a -= 360;
    if (side > 0 && a < 0) a += 360;
    return { a: a, s: len / ARM_LEN };
  }

  var HIDE_L = aimArm(SHOULDER.l, EYE.l.x, EYE.l.y, -1);
  var HIDE_R = aimArm(SHOULDER.r, EYE.r.x, EYE.r.y, 1);
  var PEEK_R = aimArm(SHOULDER.r, 214, 124, 1);
  var JOY_L = { a: HIDE_L.a - 43, s: 1 };
  var JOY_R = { a: HIDE_R.a + 43, s: 1 };
  var REST_L = { a: -8, s: 1 };
  var REST_R = { a: 8, s: 1 };
  var FLINCH_L = { a: 22, s: 1 };
  var FLINCH_R = { a: -22, s: 1 };

  
  function Yay(v, k, d) {
    this.v = v; this.t = v; this.vel = 0; this.k = k || 150; this.d = d || 20;
  }
  Yay.prototype.set = function (v) {
    this.t = v;
    if (RM.matches) { this.v = v; this.vel = 0; }
  };
  Yay.prototype.kick = function (v) { if (!RM.matches) this.vel += v; };
  Yay.prototype.step = function (dt) {
    if (RM.matches) { this.v = this.t; return this.v; }
    var n = Math.max(1, Math.ceil(dt / 0.008));
    var h = dt / n;
    for (var i = 0; i < n; i++) {
      this.vel += (this.k * (this.t - this.v) - this.d * this.vel) * h;
      this.v += this.vel * h;
    }
    return this.v;
  };

  var sp = {
    armLA: new Yay(REST_L.a, 118, 17), armLS: new Yay(REST_L.s, 150, 21),
    armRA: new Yay(REST_R.a, 118, 17), armRS: new Yay(REST_R.s, 150, 21),
    tilt: new Yay(0, 130, 18), headY: new Yay(0, 150, 17),
    bodyY: new Yay(0, 140, 15), gazeX: new Yay(0, 190, 24),
    gazeY: new Yay(0, 190, 24), earL: new Yay(0, 200, 15),
    earR: new Yay(0, 200, 15), happy: new Yay(0, 130, 22),
    blush: new Yay(0, 90, 18), glow: new Yay(0, 70, 16),
    tailD: new Yay(0, 90, 16), tiltX: new Yay(0, 80, 15),
    tiltY: new Yay(0, 80, 15), shX: new Yay(150, 60, 13),
    shY: new Yay(150, 60, 13)
  };

  var mood = 'idle';
  var revealed = false;
  var locked = false;
  var busy = false;
  var shakeT = 0;
  var shakeHeadT = 0;
  var HEAD_SHAKE = 1;

  var BUBBLE = {
    watch: { say: 'selam!', hold: 1500 },
    hide: { say: 'bakmıyorum!' },
    peek: { say: '…tamam, tek göz' },
    sad: { say: 'olmaz ki!', tone: 'is-bad' },
    happy: { say: 'mırnav!', tone: 'is-good' },
    wait: { say: 'bir saniye…' }
  };
  var bubbleTimer = null;

  function setMood(m) {
    if (mood === m) return;
    mood = m;
    document.documentElement.dataset.mood = m;
    var b = BUBBLE[m];
    showBubble(b && b.say, b && b.hold, b && b.tone);
    applyPose();
  }

  function showBubble(text, hold, tone) {
    clearTimeout(bubbleTimer);
    bubble.classList.remove('is-bad', 'is-good');
    if (!text) { bubble.classList.remove('is-on'); return; }
    bubbleText.textContent = text;
    if (tone) bubble.classList.add(tone);
    bubble.classList.add('is-on');
    sizeTail();
    if (hold) {
      bubbleTimer = setTimeout(function () {
        var b = BUBBLE[mood];
        if (b && !b.hold) showBubble(b.say, null, b.tone);
        else bubble.classList.remove('is-on');
      }, hold);
    }
  }

  function applyPose() {
    var m = mood;
    var hiding = m === 'hide' || m === 'peek';

    sp.armLA.set(hiding ? HIDE_L.a : m === 'sad' ? FLINCH_L.a : m === 'happy' ? JOY_L.a : REST_L.a);
    sp.armLS.set(hiding ? HIDE_L.s : m === 'sad' ? FLINCH_L.s : m === 'happy' ? JOY_L.s : REST_L.s);
    sp.armRA.set(m === 'peek' ? PEEK_R.a : m === 'hide' ? HIDE_R.a : m === 'sad' ? FLINCH_R.a : m === 'happy' ? JOY_R.a : REST_R.a);
    sp.armRS.set(m === 'peek' ? PEEK_R.s : m === 'hide' ? HIDE_R.s : m === 'sad' ? FLINCH_R.s : m === 'happy' ? JOY_R.s : REST_R.s);

    moveRail();
    sp.happy.set(m === 'happy' ? 1 : 0);
    sp.blush.set(m === 'happy' ? 1 : hiding ? 0.9 : m === 'sad' ? 0.5 : 0.34);
    sp.glow.set(m === 'happy' ? 1 : 0);
    sp.tailD.set(m === 'sad' ? 1 : 0);

    $('mouth').setAttribute('d', m === 'happy' ? MOUTH.happy : m === 'sad' ? MOUTH.sad : hiding ? MOUTH.small : MOUTH.neutral);
    $('tongue').style.opacity = m === 'happy' ? '1' : '0';

    if (m === 'sad') {
      sp.earL.set(38); sp.earR.set(-38); sp.headY.set(-2); sp.tilt.set(0);
    } else if (m === 'happy') {
      sp.earL.set(-8); sp.earR.set(8); sp.headY.set(0); sp.tilt.set(0);
    } else {
      sp.earL.set(0); sp.earR.set(0); sp.headY.set(0);
    }

    if (!hiding && m !== 'watch') { sp.gazeX.set(0); sp.gazeY.set(0); sp.tilt.set(0); }
    if (m === 'hide') { sp.gazeX.set(0); sp.gazeY.set(0); sp.tilt.set(0); }
    if (m === 'peek') { sp.gazeX.set(4.6); sp.gazeY.set(3.8); sp.tilt.set(3); }
  }

  
  function moveRail() {
    if (!rail) return;
    var t = document.activeElement === email ? email
      : (document.activeElement === pw || document.activeElement === peekBtn) ? pw : null;
    if (!t) { rail.style.opacity = '0'; return; }
    var f = form.getBoundingClientRect(), r = t.getBoundingClientRect();
    rail.style.opacity = '1';
    rail.style.transform = 'translate3d(0,' + (r.top - f.top + (r.height - 46) / 2).toFixed(1) + 'px,0)';
  }

  
  function sizeTail() {
    var b = bubble.getBoundingClientRect(), c = cat.getBoundingClientRect();
    if (!b.height || !c.height) return;
    var headTop = c.top + 16 / 300 * c.height;
    bubble.style.setProperty('--tail-h', Math.max(22, Math.round(headTop - b.bottom + 10)) + 'px');
  }
  addEventListener('resize', sizeTail);
  if (document.fonts) document.fonts.ready.then(sizeTail);

  addEventListener('resize', moveRail);
  form.addEventListener('focusin', moveRail);
  form.addEventListener('focusout', function () { setTimeout(moveRail, 0); });

  function checkCaps(e) {
    var on = typeof e.getModifierState === 'function' && e.getModifierState('CapsLock');
    var s = on ? 'Caps Lock açık' : '';
    if (caps.textContent !== s) caps.textContent = s;
  }
  pw.addEventListener('keydown', checkCaps);
  pw.addEventListener('keyup', checkCaps);
  pw.addEventListener('blur', function () { caps.textContent = ''; });

  
  var mirror = document.createElement('span');
  mirror.setAttribute('aria-hidden', 'true');
  mirror.style.cssText = 'position:absolute;left:-9999px;top:0;white-space:pre;visibility:hidden;';
  document.body.appendChild(mirror);

  function caretFraction(el) {
    var cs = getComputedStyle(el);
    mirror.style.font = cs.font;
    mirror.style.letterSpacing = cs.letterSpacing;
    var pos = el.value.length;
    try { if (el.selectionStart != null) pos = el.selectionStart; } catch (e) {}
    var masked = el.type === 'password' && !revealed;
    mirror.textContent = masked ? new Array(pos + 1).join('\u2022') : el.value.slice(0, pos);
    var inner = el.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight)
      - parseFloat(cs.borderLeftWidth) - parseFloat(cs.borderRightWidth);
    return Math.max(0, Math.min(1, mirror.getBoundingClientRect().width / Math.max(1, inner)));
  }

  function trackCaret() {
    if (mood !== 'watch') return;
    var f = caretFraction(email);
    sp.gazeX.set(1.6 + f * 4.4);
    sp.gazeY.set(-0.4);
    sp.tilt.set(2.2 + f * 5.2);
  }

  
  var blinkAt = 1.2, blinkPhase = 0, blinkQueued = 0;
  function blinkEnvelope(clock, dt) {
    if (RM.matches) return 1;
    if (blinkPhase > 0) {
      blinkPhase -= dt;
      var p = 1 - Math.max(0, blinkPhase) / 0.14;
      var tri = Math.abs(p * 2 - 1);
      if (blinkPhase <= 0 && blinkQueued > 0) { blinkQueued--; blinkPhase = 0.14; }
      return 0.06 + 0.94 * tri;
    }
    if (clock > blinkAt) {
      blinkPhase = 0.14;
      blinkQueued = Math.random() < 0.34 ? 1 : 0;
      blinkAt = clock + 2.4 + Math.random() * 3.4;
    }
    return 1;
  }

  
  var STAR = 'M0,-9 L2.6,-2.6 L9,0 L2.6,2.6 L0,9 L-2.6,2.6 L-9,0 L-2.6,-2.6 Z';
  var HEART = 'M0,7 C-9.5,0 -7.4,-8.4 -2.7,-8.4 C-0.9,-8.4 0,-6.9 0,-6.9 C0,-6.9 0.9,-8.4 2.7,-8.4 C7.4,-8.4 9.5,0 0,7 Z';

  function burst(n, power, clear) {
    if (RM.matches) return;
    if (clear) joy.textContent = '';
    for (var i = 0; i < n; i++) {
      var heart = i % 3 === 2;
      var el = document.createElementNS(SVG_NS, 'path');
      el.setAttribute('d', heart ? HEART : STAR);
      el.setAttribute('fill', heart ? '#FF7E92' : i % 2 ? '#FFC24A' : '#FF9A4D');
      var ang = (-160 + 320 * i / Math.max(1, n - 1) + (Math.random() * 18 - 9)) * Math.PI / 180;
      var dist = (78 + Math.random() * 46) * power;
      var x0 = 150 + Math.cos(ang) * 46, y0 = 92 + Math.sin(ang) * 40;
      var x1 = 150 + Math.cos(ang) * dist, y1 = 92 + Math.sin(ang) * dist - 26;
      var sc = (heart ? 0.62 + Math.random() * 0.4 : 0.42 + Math.random() * 0.55) * power;
      el.setAttribute('transform', 'translate(' + x0 + ',' + y0 + ') scale(0)');
      joy.appendChild(el);
      var anim = el.animate([
        { transform: 'translate(' + x0 + 'px,' + y0 + 'px) scale(0) rotate(0deg)', opacity: 0 },
        { transform: 'translate(' + ((x0 + x1) / 2) + 'px,' + ((y0 + y1) / 2 - 8) + 'px) scale(' + sc + ') rotate(' + (heart ? 8 : 90) + 'deg)', opacity: 1, offset: 0.32 },
        { transform: 'translate(' + x1 + 'px,' + y1 + 'px) scale(' + (sc * 0.82) + ') rotate(' + (heart ? 16 : 220) + 'deg)', opacity: 0 }
      ], {
        duration: 1050 + Math.random() * 520,
        delay: i * 26,
        easing: 'cubic-bezier(.22,.9,.3,1)',
        fill: 'forwards'
      });
      if (!clear) anim.onfinish = (function (node) { return function () { node.remove(); }; })(el);
    }
  }
  function celebrate() { burst(14, 1, true); }

  
  var last = performance.now() / 1000, clock = 0;

  function tick(now) {
    var t = now / 1000;
    var dt = Math.min(0.05, t - last);
    last = t;
    clock += dt;
    var a = RM.matches ? 0 : 1;

    var breathRate = mood === 'happy' ? 3.1 : (mood === 'hide' || mood === 'peek') ? 1.5 : 2;
    var squash = Math.sin(clock * breathRate) * 0.014 * a;
    var bodyY = sp.bodyY.step(dt);
    $('bodyG').setAttribute('transform',
      'translate(0,' + bodyY.toFixed(2) + ') translate(150,256) scale(' +
      (1 + squash).toFixed(4) + ',' + (1 - squash * 0.65).toFixed(4) + ') translate(-150,-256)');

    
    var hsX = 0, hsY = 0;
    if (shakeHeadT > 0 && !RM.matches) {
      shakeHeadT = Math.max(0, shakeHeadT - dt);
      var dec = Math.pow(shakeHeadT / HEAD_SHAKE, 2);
      hsX = Math.sin(shakeHeadT * 29) * 17 * dec;
      hsY = Math.sin(shakeHeadT * 29) * 5.5 * dec;
    } else if (shakeHeadT > 0) {
      shakeHeadT = 0;
    }

    var tilt = sp.tilt.step(dt);
    var headY = sp.headY.step(dt) + bodyY * 0.9 + Math.sin(clock * breathRate + 0.5) * 1.3 * a;
    $('headG').setAttribute('transform',
      'translate(' + hsY.toFixed(2) + ',' + headY.toFixed(2) + ') translate(' + NECK.x + ',' + NECK.y +
      ') rotate(' + (tilt + hsX).toFixed(2) + ') translate(' + (-NECK.x) + ',' + (-NECK.y) + ')');

    var earSwayL = Math.sin(clock * 1.1) * 2.4 * a;
    var earSwayR = Math.sin(clock * 1.1 + 1.7) * 2.4 * a;
    $('earL').setAttribute('transform', 'translate(112,66) rotate(' + (sp.earL.step(dt) + earSwayL - hsX * 0.42).toFixed(2) + ') translate(-112,-66)');
    $('earR').setAttribute('transform', 'translate(188,66) rotate(' + (sp.earR.step(dt) + earSwayR - hsX * 0.42).toFixed(2) + ') translate(-188,-66)');

    var bellRot = hsX * 1.25 + Math.sin(clock * 1.9) * 1.7 * a;
    $('bell').setAttribute('transform', 'translate(150,' + (166 + bodyY).toFixed(2) + ') rotate(' + bellRot.toFixed(2) + ')');

    var gazeX = sp.gazeX.step(dt), gazeY = sp.gazeY.step(dt);
    var happy = sp.happy.step(dt);
    var open = blinkEnvelope(clock, dt) * (1 - happy) + happy;

    [['l', 'L'], ['r', 'R']].forEach(function (pair) {
      var e = EYE[pair[0]];
      var o = pair[0] === 'r' && mood === 'peek' ? 1 : open;
      $('eye' + pair[1]).setAttribute('transform',
        'translate(' + e.x + ',' + (e.y + headY).toFixed(2) + ') scale(1,' + o.toFixed(3) + ')');
      $('pupil' + pair[1]).setAttribute('transform', 'translate(' + gazeX.toFixed(2) + ',' + gazeY.toFixed(2) + ')');
      $('eye' + pair[1] + 'open').style.opacity = String(1 - happy);
      $('eye' + pair[1] + 'happy').style.opacity = String(happy);
    });

    var aL = sp.armLA.step(dt), sL = sp.armLS.step(dt);
    var aR = sp.armRA.step(dt), sR = sp.armRS.step(dt);
    $('armL').setAttribute('transform', 'translate(' + SHOULDER.l.x + ',' + (SHOULDER.l.y + bodyY).toFixed(2) + ') rotate(' + aL.toFixed(2) + ') scale(1,' + sL.toFixed(3) + ')');
    $('armR').setAttribute('transform', 'translate(' + SHOULDER.r.x + ',' + (SHOULDER.r.y + bodyY).toFixed(2) + ') rotate(' + aR.toFixed(2) + ') scale(1,' + sR.toFixed(3) + ')');

    var tailD = sp.tailD.step(dt);
    var tailSpeed = mood === 'happy' ? 9.5 : mood === 'watch' ? 2.4 : mood === 'sad' ? 0.9 : 1.5;
    var tailAmp = (mood === 'happy' ? 17 : mood === 'watch' ? 10 : mood === 'sad' ? 3 : 7) * a;
    var tailSwing = Math.sin(clock * tailSpeed) * tailAmp;
    var tailY = 182 + tailD * 62 + Math.cos(clock * tailSpeed) * 5 * a;
    $('tail').setAttribute('d',
      'M' + TAIL_BASE.x + ',' + (TAIL_BASE.y + bodyY).toFixed(1) +
      ' Q' + (252 + tailSwing * 0.5).toFixed(1) + ',' + (246 + bodyY).toFixed(1) +
      ' ' + (262 + tailSwing * 1.5).toFixed(1) + ',' + tailY.toFixed(1));

    var blush = sp.blush.step(dt);
    $('blushL').style.opacity = String(blush * 0.85);
    $('blushR').style.opacity = String(blush * 0.85);
    $('glow').setAttribute('opacity', sp.glow.step(dt).toFixed(3));

    
    var shakeX = 0;
    if (shakeT > 0) {
      shakeT = Math.max(0, shakeT - dt);
      var k = shakeT / 0.5;
      shakeX = Math.sin(shakeT * 62) * 13 * k * k;
    }
    var rx = sp.tiltX.step(dt), ry = sp.tiltY.step(dt);
    card.style.transform = (shakeX || rx || ry)
      ? 'translate3d(' + shakeX.toFixed(2) + 'px,0,0) perspective(1500px) rotateX(' + rx.toFixed(2) + 'deg) rotateY(' + ry.toFixed(2) + 'deg)'
      : '';
    sheen.style.transform = 'translate3d(' + sp.shX.step(dt).toFixed(1) + 'px,' + sp.shY.step(dt).toFixed(1) + 'px,0)';

    requestAnimationFrame(tick);
  }
  requestAnimationFrame(tick);

  
  var FINE = matchMedia('(hover: hover) and (pointer: fine)');
  var petPanel = document.querySelector('.pet');
  card.addEventListener('pointermove', function (e) {
    if (!FINE.matches || RM.matches) return;
    var c = card.getBoundingClientRect();
    sp.tiltY.set(((e.clientX - c.left) / c.width - 0.5) * 4.2);
    sp.tiltX.set(-((e.clientY - c.top) / c.height - 0.5) * 3);
    var p = petPanel.getBoundingClientRect();
    sp.shX.set(e.clientX - p.left);
    sp.shY.set(e.clientY - p.top);
  });
  card.addEventListener('pointerleave', function () { sp.tiltX.set(0); sp.tiltY.set(0); });

  
  var PATS = ['mrrp!', 'purrr…', 'yine mi?', 'hihi'];
  var pats = 0;
  petBtn.addEventListener('click', function () {
    if (locked) return;
    sp.headY.kick(-140);
    sp.earL.kick(-240);
    sp.earR.kick(240);
    sp.blush.kick(4);
    sp.bodyY.kick(-90);
    showBubble(PATS[pats++ % PATS.length], 1200);
    burst(5, 0.6, false);
  });

  
  email.addEventListener('focus', function () { if (!locked) { setMood('watch'); trackCaret(); } });
  email.addEventListener('input', function () { sp.earL.kick(-160); sp.earR.kick(160); trackCaret(); });
  email.addEventListener('keyup', trackCaret);
  email.addEventListener('click', trackCaret);

  pw.addEventListener('focus', function () { if (!locked) setMood(revealed ? 'peek' : 'hide'); });
  pw.addEventListener('input', function () {
    sp.earL.kick(revealed ? -120 : -260);
    sp.earR.kick(revealed ? 120 : 260);
  });

  [email, pw].forEach(function (el) {
    el.addEventListener('blur', function () {
      setTimeout(function () {
        if (locked || mood === 'sad' || mood === 'happy' || busy) return;
        if (!form.contains(document.activeElement)) setMood('idle');
      }, 0);
    });
  });

  peekBtn.addEventListener('click', function () {
    revealed = !revealed;
    pw.type = revealed ? 'text' : 'password';
    peekBtn.setAttribute('aria-pressed', String(revealed));
    peekBtn.setAttribute('aria-label', revealed ? 'Şifreyi gizle' : 'Şifreyi göster');
    if (!locked && (mood === 'hide' || mood === 'peek')) setMood(revealed ? 'peek' : 'hide');
  });

  function say(text, tone) {
    statusEl.textContent = text;
    statusEl.className = 'status is-on' + (tone ? ' is-' + tone : '');
  }

  
  form.addEventListener('submit', function (e) {
    e.preventDefault();
    if (locked || busy) return;

    if (!email.value.trim() || !pw.value) {
      say('İkisi de gerekli.', 'bad');
      shakeT = 0.5;
      card.classList.add('is-bad');
      setMood('sad');
      return;
    }

    busy = true;
    submitBtn.disabled = true;
    setMood('wait');

    var body = new URLSearchParams();
    body.append('email', email.value.trim());
    body.append('sifre', pw.value);

    fetch('/admin/giris' + location.search, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body.toString()
    }).then(function (res) {
      var url = res.url || '';
      var path = url ? new URL(url, location.origin).pathname : '/admin/giris';

      if (path.indexOf('/admin/giris/2fa') === 0) {
        location.href = url;
        return null;
      }

      if (path.indexOf('/admin/giris') === 0) {
        return res.text().then(function (html) {
          var metin = 'E-posta veya şifre hatalı.';
          try {
            var doc = new DOMParser().parseFromString(html, 'text/html');
            var fl = doc.querySelector('.flash');
            if (fl && fl.textContent.trim()) metin = fl.textContent.trim();
          } catch (err) {}
          basarisiz(metin);
          return null;
        });
      }

      basarili(url);
      return null;
    }).catch(function () {
      basarisiz('Sunucuya ulaşılamadı. Bağlantınızı kontrol edin.');
    });
  });

  function basarisiz(msg) {
    busy = false;
    submitBtn.disabled = false;
    card.classList.add('is-bad');
    say(msg, 'bad');
    shakeT = 0.5;
    shakeHeadT = HEAD_SHAKE;
    setMood('sad');
    sp.headY.kick(-150);
    
    setTimeout(function () {
      if (locked || mood !== 'sad' || busy) return;
      var yaziyor = pw.value || form.contains(document.activeElement);
      setMood(yaziyor ? (revealed ? 'peek' : 'hide') : 'idle');
    }, 1100);
  }

  function basarili(nextURL) {
    busy = false;
    locked = true;
    card.classList.remove('is-bad');
    card.classList.add('is-good');
    say('Hoş geldiniz — kapı açık.', 'good');
    setMood('happy');
    sp.bodyY.kick(-260);
    celebrate();
    pw.blur();
    
    setTimeout(function () {
      location.href = nextURL || '/admin';
    }, 1050);
  }

  RM.addEventListener('change', applyPose);
  applyPose();
  document.documentElement.dataset.mood = mood;
  document.documentElement.dataset.ready = '1';

  
  if (srvFlash && srvFlash.textContent.trim()) {
    var metin = srvFlash.textContent.trim();
    var hata = !srvFlash.classList.contains('info') && !srvFlash.classList.contains('success');
    srvFlash.hidden = true;
    say(metin, hata ? 'bad' : undefined);
    if (hata) {
      card.classList.add('is-bad');
      shakeT = 0.5;
      shakeHeadT = HEAD_SHAKE;
      setMood('sad');
      sp.headY.kick(-150);
    }
  }
})();
