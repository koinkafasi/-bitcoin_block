// Manset Editoru motoru (templ surumu) — inline script'ten tasindi.
// Panel HTMX ile yeniden basildiginda boot() tekrar baglanir.
(function () {
  "use strict";
  function boot() {
    var CV = document.getElementById("me-canvas");
    if (!CV || CV.getAttribute("data-ready") === "1") return;
    CV.setAttribute("data-ready", "1");
    var ctx = CV.getContext("2d");
  var CW = 1200, CH = 675;
  var FONT = 'system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif';
  var clamp = function (v, a, b) { return v < a ? a : (v > b ? b : v); };
  var state = JSON.parse(JSON.stringify(window.INIT_DESIGN || {}));
  var PRESETS = Array.isArray(window.MANSET_PRESETS) ? window.MANSET_PRESETS : [];
  state.layers = state.layers || [];
  state.background = state.background || {};
  state.background.overlay = state.background.overlay || {};
  state.badge = state.badge || {};
  var bgImg = null, curId = 0, selected = (state.layers[0] || {}).id || "";
  var zoom = "fit", snap = true, showGrid = false, selObj = null; // selObj: {kind:'layer'|'badge', ref}

  // ---- history (undo/redo) ----
  var hist = [], hp = -1;
  function snapshot() { hist = hist.slice(0, hp + 1); hist.push(JSON.stringify(state)); hp = hist.length - 1; if (hist.length > 80) { hist.shift(); hp--; } }
  function restore(s) { state = JSON.parse(s); state.layers = state.layers || []; selected = (state.layers[0] || {}).id || ""; selObj = null; ui(); }
  function undo() { if (hp > 0) { hp--; restore(hist[hp]); } }
  function redo() { if (hp < hist.length - 1) { hp++; restore(hist[hp]); } }
  function mutate(fn, snap) { if (snap !== false) snapshot(); fn(); schedule(); }

  // ---- render ----
  var raf = 0;
  function schedule() { if (raf) return; raf = requestAnimationFrame(function () { raf = 0; render(); }); }
  function fontStr(l) { return (l.weight >= 600 ? "bold " : "") + (l.fontSize || 30) + "px " + FONT; }
  function measure(fs, weight, s) { ctx.font = (weight >= 600 ? "bold " : "") + fs + "px " + FONT; return ctx.measureText(s).width; }
  function wrapped(l) {
    var fs = l.fontSize || 30, maxW = l.maxWidth > 0 ? l.maxWidth : (CW - (l.x || 0) - 40);
    var text = l.uppercase ? (l.text || "").toUpperCase() : (l.text || "");
    var out = [], lines = text.split("\n");
    ctx.font = fontStr(l);
    for (var i = 0; i < lines.length; i++) {
      var words = lines[i].split(/\s+/).filter(Boolean);
      if (!words.length) { out.push(""); continue; }
      var ln = words[0];
      for (var j = 1; j < words.length; j++) {
        if (ctx.measureText(ln + " " + words[j]).width <= maxW || !ln) ln += " " + words[j];
        else { out.push(ln); ln = words[j]; }
      }
      out.push(ln);
    }
    return out;
  }
  function lineW(l, line) { ctx.font = fontStr(l); return ctx.measureText(line).width; }
  function blockBox(l) {
    var ls = wrapped(l), fs = l.fontSize || 30, lh = (l.lineHeight || 1.15) * fs;
    var w = 0; ls.forEach(function (s) { var x = lineW(l, s); if (x > w) w = x; });
    return { lines: ls, w: w, h: Math.max(fs, ls.length * lh), lh: lh, fs: fs };
  }
  function layerTopLeft(l, b) {
    var x = l.x || 0;
    if (l.align === "center") x -= b.w / 2; else if (l.align === "right") x -= b.w;
    return { x: x, y: (l.y || 0) - b.fs * 0.8 };
  }
  function badgeBox() {
    var b = state.badge; if (!b.active || !b.text) return null;
    var fs = b.fontSize || 24, pad = (b.padding || 12);
    ctx.font = "bold " + fs + "px " + FONT;
    var w = ctx.measureText(b.uppercase ? b.text.toUpperCase() : b.text).width;
    return { x: b.x || 60, y: b.y || 60, w: w + pad * 2, h: fs + pad * 0.8, fs: fs, pad: pad };
  }

  function render() {
    var bg = state.background || {};
    ctx.save(); ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, CW, CH);
    ctx.fillStyle = bg.color || "#0b0e14"; ctx.fillRect(0, 0, CW, CH);
    if (bgImg) ctx.drawImage(bgImg, 0, 0, CW, CH);
    var ov = bg.overlay || {};
    if (ov.active !== false) {
      var maxA = clamp((ov.opacity == null ? 85 : ov.opacity) / 100, 0, 1), sp = clamp(ov.spread || 70, 5, 100) / 100;
      var c = ov.color || "#000000";
      if (ov.mode === "solid") { ctx.globalAlpha = maxA; ctx.fillStyle = c; ctx.fillRect(0, 0, CW, CH); ctx.globalAlpha = 1; }
      else if (ov.mode === "vignette") {
        var outer = Math.hypot(CW / 2, CH / 2), inner = outer * (1 - sp);
        var rg = ctx.createRadialGradient(CW / 2, CH / 2, inner, CW / 2, CH / 2, outer);
        rg.addColorStop(0, "rgba(0,0,0,0)"); rg.addColorStop(1, hexA(c, maxA));
        ctx.fillStyle = rg; ctx.fillRect(0, 0, CW, CH);
      } else {
        var g = gradFor(ov.direction || "left", c, maxA, sp); ctx.fillStyle = g; ctx.fillRect(0, 0, CW, CH);
      }
    }
    drawBadge();
    state.layers.forEach(function (l) { if (l.visible !== false) drawLayer(l); });
    drawSelection();
    ctx.restore();
  }
  function hexA(hex, a) { var h = (hex || "#000").replace("#", ""); if (h.length !== 6) h = "000000"; var r = Number.parseInt(h.substr(0, 2), 16), g = Number.parseInt(h.substr(2, 2), 16), b = Number.parseInt(h.substr(4, 2), 16); return "rgba(" + r + "," + g + "," + b + "," + a + ")"; }
  function gradFor(dir, c, maxA, sp) {
    var x0 = 0, y0 = 0, x1 = 0, y1 = 0;
    switch (dir) {
      case "right": x0 = CW; x1 = 0; break;
      case "top": y1 = CH; break;
      case "bottom": y0 = CH; break;
      case "tl": x1 = CW; y1 = CH; break;
      case "tr": x0 = CW; y1 = CH; break;
      case "bl": y0 = CH; x1 = CW; break;
      case "br": x0 = CW; y0 = CH; break;
      default: x1 = CW; break;
    }
    var g = ctx.createLinearGradient(x0, y0, x1, y1);
    g.addColorStop(0, hexA(c, maxA)); g.addColorStop(sp, "rgba(0,0,0,0)");
    return g;
  }
  function drawLayer(l) {
    var b = blockBox(l), tl = layerTopLeft(l, b);
    ctx.font = fontStr(l); ctx.textBaseline = "alphabetic";
    if (l.shadow) { ctx.save(); ctx.fillStyle = l.shadowColor || "#000000"; }
    b.lines.forEach(function (line, i) {
      var y = (l.y || 0) + i * b.lh;
      var w = lineW(l, line), x = (l.x || 0);
      if (l.align === "center") x -= w / 2; else if (l.align === "right") x -= w;
      if (l.shadow) ctx.fillText(line, x + 2, y + 2);
    });
    if (l.shadow) ctx.restore();
    ctx.fillStyle = l.color || "#fff";
    b.lines.forEach(function (line, i) {
      var y = (l.y || 0) + i * b.lh, w = lineW(l, line), x = (l.x || 0);
      if (l.align === "center") x -= w / 2; else if (l.align === "right") x -= w;
      ctx.fillText(line, x, y);
    });
  }
  function drawBadge() {
    var bb = badgeBox(); if (!bb) return;
    var b = state.badge, r = b.radius == null ? 8 : b.radius;
    ctx.fillStyle = b.color || "#e11d48";
    rr(bb.x, bb.y, bb.w, bb.h, r); ctx.fill();
    ctx.font = "bold " + bb.fs + "px " + FONT; ctx.fillStyle = b.textColor || "#ffffff";
    ctx.textBaseline = "middle";
    ctx.fillText(b.uppercase ? b.text.toUpperCase() : b.text, bb.x + bb.pad, bb.y + bb.h / 2 + bb.fs * 0.04);
    ctx.textBaseline = "alphabetic";
  }
  function rr(x, y, w, h, r) { ctx.beginPath(); ctx.moveTo(x + r, y); ctx.arcTo(x + w, y, x + w, y + h, r); ctx.arcTo(x + w, y + h, x, y + h, r); ctx.arcTo(x, y + h, x, y, r); ctx.arcTo(x, y, x + w, y, r); ctx.closePath(); }

  function drawSelection() {
    if (!selObj) return;
    ctx.save(); ctx.strokeStyle = "#22d3ee"; ctx.lineWidth = 2; ctx.setLineDash([6, 4]);
    var bx;
    if (selObj.kind === "badge") { var bb = badgeBox(); if (bb) bx = bb; }
    else { var l = selObj.ref, b = blockBox(l), tl = layerTopLeft(l, b); bx = { x: tl.x - 6, y: tl.y - 4, w: b.w + 12, h: b.h + 12 }; }
    if (bx) ctx.strokeRect(bx.x, bx.y, bx.w, bx.h);
    ctx.restore();
  }
  function guides(lines) {
    if (!lines || !lines.length) return;
    ctx.save(); ctx.strokeStyle = "#f472b6"; ctx.lineWidth = 1;
    lines.forEach(function (g) {
      ctx.beginPath();
      if (g.v != null) { ctx.moveTo(g.v, 0); ctx.lineTo(g.v, CH); } else { ctx.moveTo(0, g.h); ctx.lineTo(CW, g.h); }
      ctx.stroke();
    });
    ctx.restore();
  }

  // ---- coordinate + hit test ----
  function toCanvas(cx, cy) { var r = CV.getBoundingClientRect(); return { x: (cx - r.left) * (CW / r.width), y: (cy - r.top) * (CH / r.height) }; }
  function hit(px, py) {
    var bb = badgeBox();
    if (bb && px >= bb.x && px <= bb.x + bb.w && py >= bb.y && py <= bb.y + bb.h) return { kind: "badge", ref: state.badge };
    for (var i = state.layers.length - 1; i >= 0; i--) {
      var l = state.layers[i]; if (l.visible === false || l.locked) continue;
      var b = blockBox(l), tl = layerTopLeft(l, b);
      if (px >= tl.x - 8 && px <= tl.x + b.w + 8 && py >= tl.y - 6 && py <= tl.y + b.h + 6) return { kind: "layer", ref: l };
    }
    return null;
  }

  // ---- drag with snapping ----
  var drag = null, curGuides = [];
  CV.addEventListener("pointermove", function (e) {
    var p = toCanvas(e.clientX, e.clientY);
    if (!drag) { CV.style.cursor = hit(p.x, p.y) ? "move" : "default"; return; }
    var nx = p.x - drag.ox, ny = p.y - drag.oy;
    curGuides = [];
    if (snap) {
      var b = drag.kind === "badge" ? { w: drag.w, h: drag.h } : blockBox(drag.ref);
      var cand = [];
      // dikey hedefler
      [0, CW / 2, CW].forEach(function (v) { cand.push({ v: v, off: [0, -b.w / 2, -b.w] }); });
      state.layers.forEach(function (o) { if (o === drag.ref || o.visible === false) return; var ob = blockBox(o), otl = layerTopLeft(o, ob); [otl.x, otl.x + ob.w / 2, otl.x + ob.w].forEach(function (v) { cand.push({ v: v, off: [0, -b.w / 2, -b.w] }); }); });
      var T = 8;
      outer: for (var i = 0; i < cand.length; i++) for (var k = 0; k < 3; k++) {
        var target = cand[i].v + cand[i].off[k];
        if (Math.abs(nx - target) <= T) { nx = target; curGuides.push({ v: cand[i].v }); break outer; }
      }
      var candH = [];
      [0, CH / 2, CH].forEach(function (v) { candH.push(v); });
      state.layers.forEach(function (o) { if (o === drag.ref || o.visible === false) return; var ob = blockBox(o); candH.push((o.y || 0)); });
      outerH: for (var j = 0; j < candH.length; j++) { if (Math.abs(ny - candH[j]) <= T) { ny = candH[j]; curGuides.push({ h: candH[j] }); break outerH; } }
    }
    drag.ref.x = Math.round(nx); drag.ref.y = Math.round(ny);
    schedule();
    scheduleGuides();
  });
  var gRaf = 0;
  function scheduleGuides() { if (gRaf) return; gRaf = requestAnimationFrame(function () { gRaf = 0; render(); guides(curGuides); }); }
  CV.addEventListener("pointerdown", function (e) {
    CV.setPointerCapture(e.pointerId);
    var p = toCanvas(e.clientX, e.clientY);
    var h = hit(p.x, p.y);
    if (!h) { selObj = null; selected = ""; uiSel(); schedule(); return; }
    selObj = h;
    if (h.kind === "layer") { selected = h.ref.id; uiSel(); }
    var bb = h.kind === "badge" ? badgeBox() : blockBox(h.ref);
    var ax = h.kind === "badge" ? (h.ref.x || 60) : (h.ref.x || 0);
    var ay = h.kind === "badge" ? (h.ref.y || 60) : (h.ref.y || 0);
    drag = { kind: h.kind, ref: h.ref, ox: p.x - ax, oy: p.y - ay, w: bb.w, h: bb.h };
    schedule();
  });
  CV.addEventListener("pointerup", function () { if (drag) { snapshot(); } drag = null; curGuides = []; schedule(); });
  CV.addEventListener("pointercancel", function () { drag = null; curGuides = []; schedule(); });

  // ---- keyboard ----
  document.addEventListener("keydown", function (e) {
    var tag = (e.target.tagName || "").toLowerCase();
    if (tag === "input" || tag === "textarea" || tag === "select") return;
    var mod = e.ctrlKey || e.metaKey;
    if (mod && (e.key === "z" || e.key === "Z")) { e.preventDefault(); undo(); return; }
    if (mod && (e.key === "y" || e.key === "Y")) { e.preventDefault(); redo(); return; }
    if (mod && (e.key === "s" || e.key === "S")) { e.preventDefault(); save(); return; }
    if (mod && (e.key === "d" || e.key === "D")) { e.preventDefault(); dup(); return; }
    if (!selObj || selObj.kind !== "layer") return;
    var step = e.shiftKey ? 10 : 1, l = selObj.ref, moved = false;
    if (e.key === "ArrowLeft") { l.x -= step; moved = true; } else if (e.key === "ArrowRight") { l.x += step; moved = true; }
    else if (e.key === "ArrowUp") { l.y -= step; moved = true; } else if (e.key === "ArrowDown") { l.y += step; moved = true; }
    else if (e.key === "Delete" || e.key === "Backspace") { e.preventDefault(); del(); return; }
    if (moved) { e.preventDefault(); l.x = Math.round(l.x); l.y = Math.round(l.y); schedule(); uiSel(); }
  });

  // ---- UI wiring ----
  function $(s) { return document.querySelector(s); }
  // sekmeler
  document.querySelectorAll(".me-tabs button").forEach(function (b) {
    b.addEventListener("click", function () {
      document.querySelectorAll(".me-tabs button").forEach(function (x) { x.classList.toggle("on", x === b); });
      document.querySelectorAll(".me-pane").forEach(function (p) { p.classList.toggle("on", p.getAttribute("data-pane") === b.getAttribute("data-tab")); });
    });
  });
  // yön butonları
  var DIRS = [["tl", "↖"], ["top", "↑"], ["tr", "↗"], ["left", "←"], ["all", "◆"], ["right", "→"], ["bl", "↙"], ["bottom", "↓"], ["br", "↘"]];
  var dirBox = $("#me-dirs");
  DIRS.forEach(function (d) {
    var b = document.createElement("button"); b.className = "btn ghost small"; b.type = "button"; b.textContent = d[1]; b.title = d[0];
    b.setAttribute("data-dir", d[0]);
    if (d[0] === "all") b.title = "Tümü (vinyet benzeri düz)";
    b.addEventListener("click", function () { mutate(function () { state.background.overlay.direction = d[0]; if (d[0] === "all") state.background.overlay.mode = "solid"; }); uiBg(); });
    dirBox.appendChild(b);
  });
  function setInput(id, v) { var el = $(id); if (el) el.value = v; }
  function uiBg() {
    var bg = state.background, ov = bg.overlay || (bg.overlay = {});
    setInput("#me-bg-color", bg.color || "#0b0e14");
    setInput("#me-overlay-color", ov.color || "#000000");
    $("#me-overlay-on").checked = ov.active !== false;
    setInput("#me-overlay-mode", ov.mode || "gradient");
    setInput("#me-overlay-spread", ov.spread || 70);
    setInput("#me-overlay-opacity", ov.opacity == null ? 85 : ov.opacity);
    $("#me-spread-v").textContent = "%" + (ov.spread || 70);
    $("#me-opacity-v").textContent = "%" + (ov.opacity == null ? 85 : ov.opacity);
    if (bg.media_id) { var s = $("#me-bg-media"); if (s) s.value = String(bg.media_id); }
    document.querySelectorAll("#me-dirs [data-dir]").forEach(function (b) { b.style.outline = (b.getAttribute("data-dir") === (ov.direction || "left")) ? "2px solid var(--accent)" : "none"; });
  }
  function uiBadge() {
    var b = state.badge;
    $("#me-badge-on").checked = b.active !== false;
    setInput("#me-badge-text", b.text || "");
    setInput("#me-badge-color", b.color || "#e11d48");
    setInput("#me-badge-textcolor", b.textColor || "#ffffff");
    setInput("#me-badge-size", b.fontSize || 24);
    setInput("#me-badge-radius", b.radius == null ? 8 : b.radius);
    setInput("#me-badge-pad", b.padding == null ? 12 : b.padding);
    $("#me-badge-upper").checked = !!b.uppercase;
  }
  function uiSel() {
    document.querySelectorAll(".me-layer").forEach(function (el) { el.classList.toggle("on", el.getAttribute("data-id") === selected); });
    var l = curLayer();
    var box = $("#me-layerform");
    if (!l) { box.innerHTML = '<p class="muted small">Katman seçili değil.</p>'; return; }
    box.innerHTML =
      '<div class="field"><label for="m-t">Metin</label><textarea class="textarea" id="m-t" style="min-height:56px">' + esc(l.text || "") + '</textarea></div>' +
      '<div class="input-row" style="grid-template-columns:1fr 1fr"><div class="field"><label for="m-s">Boyut</label><input class="input" type="number" id="m-s" min="10" max="180" value="' + (l.fontSize || 30) + '"></div>' +
      '<div class="field"><label for="m-w">Kalınlık</label><input class="input" type="number" id="m-w" min="100" max="900" step="100" value="' + (l.weight || 400) + '"></div></div>' +
      '<div class="input-row" style="grid-template-columns:1fr 1fr"><div class="field"><label for="m-c">Renk</label><input type="color" class="input" id="m-c" value="' + (l.color || "#ffffff") + '"></div>' +
      '<div class="field"><label for="m-a">Hizalama</label><select class="input" id="m-a"><option value="left"' + (l.align === "left" ? " selected" : "") + '>Sol</option><option value="center"' + (l.align === "center" ? " selected" : "") + '>Orta</option><option value="right"' + (l.align === "right" ? " selected" : "") + '>Sağ</option></select></div></div>' +
      '<div class="field"><label for="m-lh">Satır aralığı</label><input type="range" id="m-lh" min="0.8" max="1.8" step="0.05" value="' + (l.lineHeight || 1.15) + '"></div>' +
      '<div class="field"><label for="m-mw">Maks. genişlik (kaydırma) <span class="muted small">' + (l.maxWidth > 0 ? l.maxWidth + "px" : "otomatik") + '</span></label><input type="range" id="m-mw" min="120" max="1120" step="20" value="' + (l.maxWidth > 0 ? l.maxWidth : 780) + '"></div>' +
      '<div class="input-row" style="grid-template-columns:1fr 1fr"><label class="small muted"><input type="checkbox" id="m-uc"' + (l.uppercase ? " checked" : "") + '> BÜYÜK</label>' +
      '<label class="small muted"><input type="checkbox" id="m-sh"' + (l.shadow ? " checked" : "") + '> Gölge</label></div>' +
      '<div class="field"><label for="m-shc">Gölge rengi</label><input type="color" class="input" id="m-shc" value="' + (l.shadowColor || "#000000") + '"></div>' +
      '<label class="small muted"><input type="checkbox" id="m-lock"' + (l.locked ? " checked" : "") + '> Kilitli</label>';
    bind("#m-t", function (v) { l.text = v; }, true);
    bind("#m-s", function (v) { l.fontSize = Number.parseInt(v, 10) || 30; }, true);
    bind("#m-w", function (v) { l.weight = Number.parseInt(v, 10) || 400; }, true);
    bind("#m-c", function (v) { l.color = v; }, true);
    bind("#m-a", function (v) { l.align = v; }, true);
    bind("#m-lh", function (v) { l.lineHeight = Number.parseFloat(v) || 1.15; }, true);
    bind("#m-mw", function (v) { l.maxWidth = Number.parseInt(v, 10); }, true);
    chk("#m-uc", function (v) { l.uppercase = v; });
    chk("#m-sh", function (v) { l.shadow = v; });
    bind("#m-shc", function (v) { l.shadowColor = v; }, true);
    chk("#m-lock", function (v) { l.locked = v; });
  }
  function esc(s) {
    return String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
  function bind(sel, cb, snap) { var el = $(sel); if (!el) return; var h = function () { mutate(function () { cb(el.value); }, snap !== false); }; el.addEventListener("input", h); el.addEventListener("change", h); }
  function chk(sel, cb) { var el = $(sel); if (!el) return; el.addEventListener("change", function () { mutate(function () { cb(el.checked); }); if (sel === "#m-lock") uiSel(); }); }
  function curLayer() { for (var i = 0; i < state.layers.length; i++) if (state.layers[i].id === selected) return state.layers[i]; return null; }
  function uiLayers() {
    var box = $("#me-layers"); box.innerHTML = "";
    state.layers.forEach(function (l, i) {
      var row = document.createElement("div"); row.className = "me-layer" + (l.id === selected ? " on" : ""); row.setAttribute("data-id", l.id);
      row.innerHTML = '<span class="nm">' + esc(l.role || "") + " · " + esc((l.text || "").slice(0, 24)) + "</span>";
      var up = mk("↑", "Yukarı", function (e) { e.stopPropagation(); move(i, -1); });
      var dn = mk("↓", "Aşağı", function (e) { e.stopPropagation(); move(i, 1); });
      var vis = mk(l.visible === false ? "🙈" : "👁", "Görünürlük", function (e) { e.stopPropagation(); mutate(function () { l.visible = l.visible === false; }); uiLayers(); });
      var cp = mk("⧉", "Çoğalt", function (e) { e.stopPropagation(); dup(); });
      var del = mk("🗑", "Sil", function (e) { e.stopPropagation(); del(); });
      row.appendChild(up); row.appendChild(dn); row.appendChild(vis); row.appendChild(cp); row.appendChild(del);
      row.addEventListener("click", function () { selObj = { kind: "layer", ref: l }; selected = l.id; uiSel(); schedule(); });
      box.appendChild(row);
    });
  }
  function mk(t, title, fn) { var b = document.createElement("button"); b.className = "me-icobtn"; b.type = "button"; b.textContent = t; b.title = title; b.addEventListener("click", fn); return b; }
  function move(i, d) { var j = i + d; if (j < 0 || j >= state.layers.length) return; mutate(function () { var a = state.layers; var t = a[i]; a[i] = a[j]; a[j] = t; }); uiLayers(); }
  function dup() { var l = curLayer(); if (!l) return; mutate(function () { var c = JSON.parse(JSON.stringify(l)); c.id = "k" + Date.now(); c.x += 16; c.y += 16; state.layers.push(c); selected = c.id; selObj = { kind: "layer", ref: c }; }); ui(); }
  function del() { var idx = -1; state.layers.forEach(function (l, i) { if (l.id === selected) idx = i; }); if (idx < 0) return; mutate(function () { state.layers.splice(idx, 1); selected = (state.layers[0] || {}).id || ""; selObj = null; }); ui(); }
  function addLayer() { mutate(function () { var c = { id: "k" + Date.now(), role: "katman", text: "Yeni katman", x: 80, y: 300, fontSize: 40, weight: 700, color: "#ffffff", align: "left", lineHeight: 1.15, maxWidth: 0, uppercase: false, shadow: false, shadowColor: "#000000", visible: true, locked: false }; state.layers.push(c); selected = c.id; selObj = { kind: "layer", ref: c }; }); ui(); }
  function align(mode) {
    var l = curLayer(); if (!l) return;
    var b = blockBox(l);
    mutate(function () {
      switch (mode) {
        case "left": l.align = "left"; l.x = 60; break;
        case "right": l.align = "right"; l.x = CW - 60; break;
        case "hcenter": l.align = "center"; l.x = CW / 2; break;
        case "top": l.y = 60; break;
        case "bottom": l.y = CH - 40; break;
        case "vcenter": l.y = CH / 2; break;
      }
    }); schedule(); uiSel();
  }
  function reorderTo(front) { var idx = -1; state.layers.forEach(function (l, i) { if (l.id === selected) idx = i; }); if (idx < 0) return; mutate(function () { var l = state.layers.splice(idx, 1)[0]; if (front) state.layers.push(l); else state.layers.unshift(l); }); uiLayers(); }

  function ui() { uiBg(); uiBadge(); uiLayers(); uiSel(); schedule(); }
  function applyZoom() {
    var wrap = document.querySelector(".me-canvaswrap");
    var target = zoom === "fit" ? (document.getElementById("me-stage").clientWidth - 40) : CW * Number.parseFloat(zoom);
    var z = clamp(target / CW, 0.15, 2);
    CV.style.width = Math.round(CW * z) + "px"; CV.style.height = Math.round(CH * z) + "px";
    if (wrap) { wrap.style.width = CV.style.width; wrap.style.height = CV.style.height; }
    var g = document.getElementById("me-grid-overlay"); if (g) { g.style.backgroundSize = (60 * z) + "px " + (60 * z) + "px"; g.classList.toggle("on", showGrid); }
  }
  window.addEventListener("resize", applyZoom);

  // ---- bg controls ----
  $("#me-bg-color").addEventListener("input", function (e) { mutate(function () { state.background.color = e.target.value; }, true); });
  $("#me-overlay-color").addEventListener("input", function (e) { mutate(function () { state.background.overlay.color = e.target.value; }, true); });
  $("#me-overlay-on").addEventListener("change", function (e) { mutate(function () { state.background.overlay.active = e.target.checked; }); });
  $("#me-overlay-mode").addEventListener("change", function (e) { mutate(function () { state.background.overlay.mode = e.target.value; }); });
  $("#me-overlay-spread").addEventListener("input", function (e) { mutate(function () { state.background.overlay.spread = Number.parseInt(e.target.value, 10); }, true); $("#me-spread-v").textContent = "%" + e.target.value; });
  $("#me-overlay-opacity").addEventListener("input", function (e) { mutate(function () { state.background.overlay.opacity = Number.parseInt(e.target.value, 10); }, true); $("#me-opacity-v").textContent = "%" + e.target.value; });
  $("#me-bg-media").addEventListener("change", function (e) {
    var id = e.target.value;
    mutate(function () { state.background.media_id = id ? Number.parseInt(id, 10) : null; });
    if (id) fetch("/admin/manset-editor/media/" + id).then(function (r) { return r.json(); }).then(function (j) { if (j.ok) { var im = new Image(); im.onload = function () { bgImg = im; schedule(); }; im.src = j.url; } }).catch(function () {  });
    else { bgImg = null; schedule(); }
  });
  document.querySelectorAll("[data-preset]").forEach(function (b) {
    b.addEventListener("click", function () {
      var p = b.getAttribute("data-preset");
      mutate(function () {
        var ov = state.background.overlay;
        if (p === "none") { ov.active = false; }
        else if (p === "soft") { ov.active = true; ov.mode = "gradient"; ov.direction = "left"; ov.spread = 45; ov.opacity = 55; }
        else if (p === "strong") { ov.active = true; ov.mode = "gradient"; ov.direction = "left"; ov.spread = 90; ov.opacity = 92; }
        else if (p === "bottom") { ov.active = true; ov.mode = "gradient"; ov.direction = "bottom"; ov.spread = 70; ov.opacity = 88; }
      }); uiBg();
    });
  });

  // ---- badge controls ----
  $("#me-badge-on").addEventListener("change", function (e) { mutate(function () { state.badge.active = e.target.checked; }); });
  $("#me-badge-text").addEventListener("input", function (e) { mutate(function () { state.badge.text = e.target.value; }, true); if (selObj && selObj.kind === "badge") {} });
  $("#me-badge-color").addEventListener("input", function (e) { mutate(function () { state.badge.color = e.target.value; }, true); });
  $("#me-badge-textcolor").addEventListener("input", function (e) { mutate(function () { state.badge.textColor = e.target.value; }, true); });
  $("#me-badge-size").addEventListener("input", function (e) { mutate(function () { state.badge.fontSize = Number.parseInt(e.target.value, 10) || 24; }, true); });
  $("#me-badge-radius").addEventListener("input", function (e) { mutate(function () { state.badge.radius = Number.parseInt(e.target.value, 10) || 0; }, true); });
  $("#me-badge-pad").addEventListener("input", function (e) { mutate(function () { state.badge.padding = Number.parseInt(e.target.value, 10) || 12; }, true); });
  $("#me-badge-upper").addEventListener("change", function (e) { mutate(function () { state.badge.uppercase = e.target.checked; }); });

  // ---- toolbar ----
  $("#me-addlayer").addEventListener("click", addLayer);
  $("#me-undo").addEventListener("click", undo);
  $("#me-redo").addEventListener("click", redo);
  $("#me-front").addEventListener("click", function () { reorderTo(true); });
  $("#me-back").addEventListener("click", function () { reorderTo(false); });
  $("#me-dup").addEventListener("click", dup);
  $("#me-del").addEventListener("click", del);
  document.querySelectorAll("[data-align]").forEach(function (b) { b.addEventListener("click", function () { align(b.getAttribute("data-align")); }); });
  $("#me-zoom").addEventListener("change", function (e) { zoom = e.target.value; applyZoom(); });
  $("#me-snap").addEventListener("change", function (e) { snap = e.target.checked; });
  $("#me-grid").addEventListener("change", function (e) { showGrid = e.target.checked; applyZoom(); });

  // ---- save / templates / export ----
  function save() {
    var name = $("#me-name").value;
    var payload = { id: curId, name: name, is_default: $("#me-default").checked, design: state };
    if (window.htmx) {
      var f = document.getElementById("mansetHtmxForm");
      f.querySelector('[name=id]').value = curId;
      f.querySelector('[name=name]').value = name;
      f.querySelector('[name=is_default]').value = $("#me-default").checked ? "1" : "";
      f.querySelector('[name=design]').value = JSON.stringify(state);
      htmx.trigger(f, "submit");
      return;
    }
    fetch("/admin/manset-editor/kaydet", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) })
      .then(function (r) { return r.json(); }).then(function (j) { if (j.ok) { curId = j.id; location.reload(); } else { alert(j.error || "Hata"); } })
      .catch(function () { alert("Bağlantı hatası: kayıt gönderilemedi."); });
  }
  $("#me-save").addEventListener("click", save);
  document.querySelectorAll("[data-tplid]").forEach(function (b) {
    b.addEventListener("click", function () {
      fetch("/admin/manset-editor/template/" + b.getAttribute("data-tplid")).then(function (r) { return r.json(); }).then(function (j) {
        if (j.ok) { curId = j.id; state = j.design; state.layers = state.layers || []; state.background = state.background || {}; state.background.overlay = state.background.overlay || {}; state.badge = state.badge || {}; selected = (state.layers[0] || {}).id || ""; selObj = null; $("#me-default").checked = !!j.is_default; snapshot(); ui(); }
      }).catch(function () { alert("Şablon yüklenemedi."); });
    });
  });
  document.querySelectorAll("[data-tplid-del]").forEach(function (b) {
    b.addEventListener("click", function () {
      if (window.htmx) return; // hx-post + hx-confirm
      if (!confirm("Tasarım silinsin mi?")) return;
      fetch("/admin/manset-editor/sil", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id: Number.parseInt(b.getAttribute("data-tplid-del"), 10) }) }).then(function (r) { if (!r.ok) throw new Error("HTTP " + r.status); location.reload(); }).catch(function () { alert("Şablon silinemedi."); });
    });
  });
  $("#me-export").addEventListener("click", function () {
    var btn = this; btn.disabled = true; btn.textContent = "…";
    fetch("/admin/manset-editor/export", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ design: state, fmt: $("#me-fmt").value }) })
      .then(function (r) { return r.json(); }).then(function (j) {
        btn.disabled = false; btn.textContent = "⬇ Dışa Aktar";
        var out = $("#me-out");
        if (j.ok) {
          // Görsel ve metin DOM ile kurulur: sunucudan gelen adres/metin
          // HTML olarak birleştirilmez (attribüte kaçış riski).
          out.textContent = "";
          var img = document.createElement("img");
          img.src = j.url || "";
          img.alt = "";
          img.style.cssText = "max-width:100%;border-radius:8px;border:1px solid var(--border)";
          out.appendChild(img);
          var bilgi = document.createElement("div");
          bilgi.className = "small muted";
          bilgi.style.marginTop = "6px";
          bilgi.textContent = "Render edildi ve Medya'ya kaydedildi (" + (j.format || "jpeg") + ", media_id=" + j.media_id + ").";
          out.appendChild(bilgi);
        } else {
          out.textContent = "";
          var hata = document.createElement("div");
          hata.className = "flash error";
          hata.textContent = j.error || "Hata";
          out.appendChild(hata);
        }
      })
      // Ağ hatasında buton "…" (devre dışı) kalıyordu — kullanıcı bir daha
      // deneyemiyordu. `.catch` hem durumu geri alır hem sebebi söyler.
      .catch(function () {
        btn.disabled = false; btn.textContent = "⬇ Dışa Aktar";
        var out = $("#me-out");
        out.textContent = "";
        var hata = document.createElement("div");
        hata.className = "flash error";
        hata.textContent = "Bağlantı hatası: dışa aktarma başarısız.";
        out.appendChild(hata);
      });
  });

  // ---- AI ----
  function ai(kind) {
    fetch("/admin/icerikler/ai", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ action: "title", input: $("#me-ai-konu").value || "son dakika", content_id: 0 }) })
      .then(function (r) { return r.json(); }).then(function (j) {
        if (!j.ok || !j.result) return;
        var t = j.result;
        mutate(function () {
          if (kind === "badge") { state.badge.active = true; state.badge.text = t.slice(0, 34).toUpperCase(); state.badge.uppercase = true; }
          else { var L = state.layers.filter(function (l) { return /ana_baslik/.test(l.role); }); var l = kind === "spot" ? (state.layers.filter(function (x) { return /alt_baslik/.test(x.role); })[0] || state.layers[2] || state.layers[0]) : (L[0] || state.layers[1] || state.layers[0]); if (l) l.text = t; }
        }); ui();
      })
      .catch(function () { alert("AI isteği başarısız: bağlantı hatası."); });
  }
  $("#me-ai-title").addEventListener("click", function () { ai("title"); });
  $("#me-ai-badge").addEventListener("click", function () { ai("badge"); });
  $("#me-ai-spot").addEventListener("click", function () { ai("spot"); });

  // ---- hazır şablonlar ----
  function applyPreset(p) {
    var keepMedia = state.background && state.background.media_id ? state.background.media_id : null;
    var next = JSON.parse(JSON.stringify(p.design || p));
    next.background = next.background || {};
    next.background.media_id = keepMedia;
    next.background.overlay = next.background.overlay || {};
    next.badge = next.badge || {};
    next.layers = next.layers || [];
    state = next;
    curId = 0;
    selected = (state.layers[0] || {}).id || "";
    selObj = null;
    $("#me-default").checked = false;
    snapshot();
    ui();
  }
  function renderPresets() {
    var box = $("#me-presets");
    if (!box) return;
    box.innerHTML = "";
    PRESETS.forEach(function (p) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "btn ghost small";
      b.textContent = p.name;
      b.addEventListener("click", function () { applyPreset(p); });
      box.appendChild(b);
    });
  }

  // ---- init ----
  renderPresets();
  snapshot();
  if (window.BGURL && window.BGURL.indexOf("/media/none") !== 0) { var im = new Image(); im.onload = function () { bgImg = im; schedule(); }; im.src = window.BGURL; }
  ui(); applyZoom();
  }
  document.addEventListener("DOMContentLoaded", boot);
  document.addEventListener("htmx:afterSwap", boot);
  if (document.readyState !== "loading") boot();
})();
