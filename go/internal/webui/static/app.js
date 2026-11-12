
(function () {
  "use strict";
  
  window.tbI18n = (function () {
    var sozluk = {};
    try {
      var raw = document.body.getAttribute("data-i18n");
      if (raw) sozluk = JSON.parse(raw) || {};
    } catch (e) { sozluk = {}; }
    // Sunucu sözlüğü (<head> içinde basılır) taban; data-i18n varsa üzerine.
    if (window.__tbI18nSeed && typeof window.__tbI18nSeed === "object") {
      for (var k in window.__tbI18nSeed) {
        if (Object.prototype.hasOwnProperty.call(window.__tbI18nSeed, k)) {
          sozluk[k] = window.__tbI18nSeed[k];
        }
      }
    }
    return function (anahtar, yedek) {
      var v = sozluk[anahtar];
      return (typeof v === "string" && v) ? v : yedek;
    };
  })();

  
  const THEME_KEY = "tb_theme";
  function applyTheme(t, persist) {
    const root = document.documentElement;
    if (t === "dark" || t === "light") {
      root.setAttribute("data-theme", t);
      if (persist) {
        try { localStorage.setItem(THEME_KEY, t); } catch (e) {}
      }
    } else {
      root.removeAttribute("data-theme"); // sistem tercihine bırak
      if (persist) {
        try { localStorage.removeItem(THEME_KEY); } catch (e) {}
      }
    }
    updateThemeIcon();
  }
  function currentTheme() {
    try {
      const saved = localStorage.getItem(THEME_KEY);
      if (saved === "dark" || saved === "light") return saved;
    } catch (e) {}
    return document.documentElement.getAttribute("data-theme") === "dark" ? "dark" : "light";
  }
  function updateThemeIcon() {
    // İkon artık CSS ile yönetiliyor (.icon-sun/.icon-moon) — JS'e gerek yok.
  }
  document.addEventListener("DOMContentLoaded", function () {
    // sunucu ilk yüklemede data-theme yazmış olabilir; kullanıcı tercihi varsa üzerine yaz
    const saved = (function () { try { return localStorage.getItem(THEME_KEY); } catch (e) { return null; } })();
    if (saved === "dark" || saved === "light") {
      document.documentElement.setAttribute("data-theme", saved);
    } else if (!document.documentElement.hasAttribute("data-theme")) {
      const sys = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
      document.documentElement.setAttribute("data-theme", sys);
    }
    updateThemeIcon();
  });
  
  const HAREKET_AZALT = (function () {
    try { return window.matchMedia("(prefers-reduced-motion: reduce)"); } catch (e) { return null; }
  })();
  const TEMA_GECIS_MS = 450;
  const TEMA_GECIS_EGRI = "cubic-bezier(.65,0,.35,1)";

  function temaGecis(next, btn) {
    const root = document.documentElement;
    const uygula = function () { applyTheme(next, true); };
    // Zaten o temadaysa geçiş kurmaya gerek yok.
    if (root.getAttribute("data-theme") === next) { uygula(); return; }
    const vt = document.startViewTransition;
    if (typeof vt !== "function" || (HAREKET_AZALT && HAREKET_AZALT.matches)) {
      uygula();
      return;
    }
    const r = btn && btn.getBoundingClientRect ? btn.getBoundingClientRect() : null;
    const cx = r ? r.left + r.width / 2 : window.innerWidth - 40;
    const cy = r ? r.top + r.height / 2 : 40;
    const yariCap = Math.hypot(
      Math.max(cx, window.innerWidth - cx),
      Math.max(cy, window.innerHeight - cy)
    );
    let gecis;
    try {
      gecis = vt.call(document, uygula);
    } catch (e) { uygula(); return; }
    if (!gecis || !gecis.ready) return;
    gecis.ready.then(function () {
      try {
        root.animate(
          {
            clipPath: [
              "circle(0px at " + cx + "px " + cy + "px)",
              "circle(" + yariCap + "px at " + cx + "px " + cy + "px)"
            ]
          },
          {
            duration: TEMA_GECIS_MS,
            easing: TEMA_GECIS_EGRI,
            pseudoElement: "::view-transition-new(root)"
          }
        );
      } catch (e) {  }
    }).catch(function () {});
  }

  document.addEventListener("click", function (e) {
    const btn = e.target.closest("[data-theme-btn]");
    if (btn) {
      const next = currentTheme() === "dark" ? "light" : "dark";
      temaGecis(next, btn);
    }
  });

  
  const FS_KEY = "tb_font_scale";
  function applyFontScale(v, save) {
    const root = document.documentElement;
    if (v === "1") root.style.fontSize = "";
    else root.style.fontSize = parseFloat(v) * 100 + "%";
    if (save) { try { localStorage.setItem(FS_KEY, v); } catch (e) {} }
  }
  function currentFontScale() {
    const saved = (function () { try { return localStorage.getItem(FS_KEY) || "1"; } catch (e) { return "1"; } })();
    const px = parseFloat(getComputedStyle(document.documentElement).fontSize) || 16;
    if (px > 19) return "1.5";
    if (px > 17.5) return "1.25";
    return saved;
  }
  document.addEventListener("DOMContentLoaded", function () {
    applyFontScale(currentFontScale(), false);
    const sels = document.querySelectorAll("[data-font-scale]");
    sels.forEach(function (s) {
      s.value = currentFontScale();
      s.addEventListener("change", function () { applyFontScale(s.value, true); });
    });
  });

  
  document.addEventListener("change", function (e) {
    const sel = e.target.closest("[data-site-select]");
    if (sel) { location.href = "/admin/site-sec?site=" + sel.value; return; }
    const sel2 = e.target.closest("[data-lang-select]");
    if (sel2) {
      const lang = sel2.value;
      // Geçerli sayfa korunur (Faz 189).
      //
      // Önceki davranış her zaman dilin ANA SAYFASINA gidiyordu: makale
      // okurken dil değiştiren kullanıcı haberi kaybediyordu. Artık yol
      // korunur, yalnız dil öneki değişir:
      //   /haber-42        → /en/haber-42
      //   /en/haber-42     → /de/haber-42
      //   /en/kategori/spor→ /kategori/spor  (tr öneksiz)
      const dilKodu = (window.__tbLangCodes || []).join(",") || "en,de,fr,es,it,pt,nl,ru,uk,pl,ro,el,ar,fa,zh,ja,ko,hi,bn,ur,az,kk,uz";
      const set = new Set(dilKodu.split(",").filter(Boolean));
      let yol = location.pathname || "/";
      const parcalar = yol.split("/").filter(Boolean);
      if (parcalar.length && set.has(parcalar[0])) {
        parcalar.shift();
      }
      const kalan = parcalar.join("/");
      let hedef;
      if (!kalan) {
        hedef = lang === "tr" ? "/" : "/" + lang + "/";
      } else if (lang === "tr") {
        hedef = "/" + kalan;
      } else {
        hedef = "/" + lang + "/" + kalan;
      }
      // Sorgu ve çapa korunur (sayfalama/arama kaybolmasın).
      if (location.search) hedef += location.search;
      if (location.hash) hedef += location.hash;
      window.location.href = hedef;
    }
  });

  
  document.addEventListener("click", function (e) {
    const t = e.target.closest("[data-ai]");
    if (!t) return;
    // Yeni AI editörü varsa eski doğrudan-yazan akışı devre dışıdır; tıklama
    // ai_editor.js tarafından onaylı sonuç paneline yönlendirilir.
    if (document.getElementById("aiEditor")) return;
    // HTMX varsa hx-post + aiResult olayı devrede; fetch fallback'ini atla.
    if (t.hasAttribute("hx-post") && window.htmx) return;
    e.preventDefault();
    const action = t.getAttribute("data-ai");
    const targetSel = t.getAttribute("data-target");
    const fromSel = t.getAttribute("data-from");
    const id = t.getAttribute("data-id");
    const btn = t;
    const old = btn.textContent;
    btn.disabled = true; btn.textContent = "…";

    const payload = {
      action: action,
      content_id: id ? Number.parseInt(id, 10) : 0,
      input: fromSel ? (document.querySelector(fromSel) ? document.querySelector(fromSel).value : "") : "",
    };
    fetch("/admin/icerikler/ai", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    })
      .then(function (r) { return r.json(); })
      .then(function (j) {
        btn.disabled = false; btn.textContent = old;
        if (!j.ok) { alert(j.error || "Hata"); return; }
        const out = j.result || "";
        const target = document.querySelector(targetSel);
        if (target) target.value = out;
        if (action === "title" && document.querySelector('[data-slug-from="title"]')) {
          const seo = document.querySelector("[data-seo-preview]");
          if (seo) updateSeoPreview();
        }
      })
      .catch(function () { btn.disabled = false; btn.textContent = old; alert("AI isteği başarısız"); });
  });

  
  document.body.addEventListener("aiResult", function (e) {
    var d = (e && e.detail) || {};
    var target = d.target ? document.querySelector(d.target) : null;
    if (target) {
      target.value = d.result || "";
      target.dispatchEvent(new Event("input", { bubbles: true }));
    }
    if (d.action === "title" && window.updateSeoPreview) window.updateSeoPreview();
  });

  
  document.addEventListener("click", function (e) {
    var b = e.target.closest("[data-trend-tag]");
    if (!b) return;
    var input = document.querySelector('input[name="etiketler"]');
    if (!input) return;
    var tag = (b.getAttribute("data-trend-tag") || "").trim();
    if (!tag) return;
    var parts = input.value.split(",").map(function (s) { return s.trim(); }).filter(Boolean);
    var exists = parts.some(function (s) { return s.toLocaleLowerCase("tr") === tag.toLocaleLowerCase("tr"); });
    if (!exists) parts.push(tag);
    input.value = parts.join(", ");
    input.dispatchEvent(new Event("input", { bubbles: true }));
    b.classList.add("active");
    b.textContent = "✓ " + tag;
  });

  
  document.addEventListener("click", function (e) {
    const b = e.target.closest("[data-voice]");
    if (!b) return;
    const target = document.querySelector(b.getAttribute("data-target"));
    if (!("webkitSpeechRecognition" in window) && !("SpeechRecognition" in window)) {
      alert("Tarayıcınız sesli giriş desteklemiyor (Chrome/Edge önerilir).");
      return;
    }
    const SR = window.SpeechRecognition || window.webkitSpeechRecognition;
    const rec = new SR();
    rec.lang = b.getAttribute("data-lang") || "tr-TR";
    rec.continuous = false;
    rec.interimResults = false;
    // DİKKAT: textContent ATANMAZ. Faz 161'de bu düğmenin içeriği yerel
    // SVG oldu; metin atamak SVG'yi siliyor ve düğme boşalıyordu (tema
    // düğmesinde de aynı tuzak yaşanmıştı). Durum sınıfla bildirilir,
    // "Dinliyor…" etiketi CSS ::after ile çizilir.
    const eskiEtiket = b.getAttribute("aria-label") || "Sesle yaz";
    b.classList.add("recording");
    b.setAttribute("aria-label", "Dinliyor…");
    rec.onresult = function (ev) {
      const text = ev.results[0][0].transcript;
      if (target) {
        target.value = (target.value ? target.value + " " : "") + text;
        target.dispatchEvent(new Event("input"));
      }
    };
    const bitir = function () {
      b.classList.remove("recording");
      b.setAttribute("aria-label", eskiEtiket);
    };
    rec.onend = bitir;
    rec.onerror = bitir;
    rec.start();
  });

  
  document.addEventListener("click", function (e) {
    const t = e.target.closest("[data-toggle]");
    if (!t) return;
    // HTMX varsa hx-post'u HTMX yürütür; JS fallback yalnız HTMX yokken çalışır.
    if (t.hasAttribute("hx-post") && window.htmx) return;
    e.preventDefault();
    fetch(t.getAttribute("data-toggle"), { method: "POST", headers: { "X-Requested-With": "XMLHttpRequest" } })
      .then(function (r) { return r.json(); })
      .then(function (j) { if (j.href) window.location.href = j.href; else window.location.reload(); })
      .catch(function () { window.location.reload(); });
  });

  
  document.addEventListener("click", function (e) {
    const t = e.target.closest("[data-confirm]");
    if (!t) return;
    const hedef = t.getAttribute("data-confirm") || "";
    // Deger "/" ile baslamiyorsa onay METNIDIR; o kural admin-ui.js'in.
    if (!hedef.startsWith("/")) return;
    // Formun kendisindeki deger de onay metnidir.
    if (t.tagName === "FORM") return;
    
    const tip = (t.getAttribute("type") || "").toLowerCase();
    const gonderici = (t.tagName === "BUTTON" || t.tagName === "INPUT") &&
      (tip === "" || tip === "submit") && t.form;
    if (gonderici) return;
    // HTMX varsa hx-confirm + hx-post devrede; native form fallback'i atla.
    if (t.hasAttribute("hx-post") && window.htmx) return;
    e.preventDefault();
    if (!window.confirm("Bu kaydı silmek istediğinize emin misiniz? Bu işlem geri alınamaz.")) return;
    const f = document.createElement("form");
    f.method = "POST";
    f.action = hedef;
    f.style.display = "none";
    document.body.appendChild(f);
    f.submit();
  });

  
  window.updateSeoPreview = function () {
    const box = document.querySelector("[data-seo-preview]");
    if (!box) return;
    // Kaynak seçimi: SEO alanı DOLU ise önizlemede o gösterilir.
    //
    // Google <title>'ı değil seo_title'ı, meta açıklama olarak da
    // seo_description'ı kullanıyor. Önizleme yalnız ana başlığa bakınca
    // editör "SEO başlığım 68 karakter, kesilir mi?" sorusunu
    // cevaplayamıyordu (canlı ölçümde önizleme hiç güncellenmiyordu).
    const alanDeger = function (sec) {
      const el = document.querySelector(sec);
      return el && el.value ? el.value.trim() : "";
    };
    const title = alanDeger("[data-src=seo_title]") || alanDeger("[data-src=title]");
    const desc = alanDeger("[data-src=seo_desc]") || alanDeger("[data-src=spot]");
    const t = title;
    const s = desc;
    const setBind = function (key, val) {
      const el = box.querySelector("[data-bind=" + key + "]");
      if (el) el.textContent = val;
    };
    // Google başlığı ~60, açıklamayı ~160 karakterde keser. Önizleme
    // tam metni gösterirse yazar kesilmeyi GÖREMİYOR — sitede 2.359
    // başlık bu yüzden kesik yayımlanmıştı (12 Kasım 2026 ölçümü).
    // Önizleme artık aramada görüneni gösteriyor.
    const KES_BASLIK = 60, KES_ACIKLAMA = 160;
    const kes = function (metin, sinir) {
      const d = Array.from(String(metin));
      return d.length > sinir ? d.slice(0, sinir).join("").trim() + "…" : metin;
    };
    setBind("title", kes(t, KES_BASLIK) || "Başlık burada görünür");
    setBind("desc", kes(s, KES_ACIKLAMA) || "Açıklama burada görünür…");
    // Sayaç: kaç karakter ve kesiliyor mu.
    const sayac = function (sec, uzunluk, sinir, ad) {
      const el = document.querySelector(sec);
      if (!el) return;
      const kesik = uzunluk > sinir;
      el.textContent = uzunluk + " karakter" +
        (uzunluk === 0 ? "" : kesik ? " — aramada kesilecek (sınır " + sinir + ")" : " — tam görünür");
      el.classList.toggle("uyari", kesik);
    };
    sayac("#seoBaslikSayac", Array.from(t).length, KES_BASLIK, "başlık");
    sayac("#seoAciklamaSayac", Array.from(s).length, KES_ACIKLAMA, "açıklama");
    setBind("url", window.location.origin + "/haber-" + slugify(t) + "-0");
    const social = document.querySelector("[data-social-preview]");
    if (social) {
      const st = social.querySelector('[data-bind="social-title"]');
      const sd = social.querySelector('[data-bind="social-desc"]');
      if (st) st.textContent = t || "Başlık burada görünür";
      if (sd) sd.textContent = s || "Açıklama burada görünür…";
    }
    let pts = 0;
    const add = function (on, w) { if (on) pts += w; };
    // Eşik 60: Google'ın kestiği yer. 70'e kadar ödüllendirmek yazarı
    // kesilen başlık yazmaya yönlendiriyordu.
    add(t.length >= 40 && t.length <= 60, 20);
    add(s.length >= 120 && s.length <= 160, 15);
    add(t.length > 0, 25); // yer tutucu + başlık varlığı
    add(s.length > 0, 15);
    const g = box.querySelector(".gauge");
    if (g) { g.style.setProperty("--v", pts); g.setAttribute("data-score", pts); }
  };
  document.addEventListener("input", function (e) {
    if (e.target.matches("[data-src]")) window.updateSeoPreview && window.updateSeoPreview();
  });

  function slugify(str) {
    return String(str).toLowerCase().trim()
      .replace(/[çÇ]/g, "c").replace(/[ğĞ]/g, "g").replace(/[ıIİ]/g, "i")
      .replace(/[öÖ]/g, "o").replace(/[şŞ]/g, "s").replace(/[üÜ]/g, "u")
      .replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "") || "icerik";
  }

  
  function calcEval(expr, vars) {
    expr = String(expr).replace(/\[([a-zA-Z0-9_]+)\]/g, function (m, name) { return "(" + (isFinite(vars[name]) ? vars[name] : 0) + ")"; });
    var toks = expr.match(/[0-9.]+|[+*/().%-]|[a-zA-Z_][a-zA-Z0-9_]*/g) || [];
    var pos = 0;
    function peek() { return toks[pos]; }
    function parseExpr() {
      var left = parseTerm();
      while (peek() === "+" || peek() === "-") { var op = peek(); pos++; var r = parseTerm(); left = op === "+" ? left + r : left - r; }
      return left;
    }
    function parseTerm() {
      var left = parseF();
      while (peek() === "*" || peek() === "/" || peek() === "%") { var op = peek(); pos++; var r = parseF(); left = op === "*" ? left * r : op === "/" ? (r ? left / r : NaN) : left % r; }
      return left;
    }
    function parseF() {
      var t = toks[pos++];
      if (t === "(") { var v = parseExpr(); pos++; return v; }
      if (t === "-") { return -parseF(); }
      return parseFloat(t) || 0;
    }
    return parseExpr();
  }

  function buildCalc() {
    document.querySelectorAll("script[data-calc]").forEach(function (s) {
      var schema;
      try { schema = JSON.parse(s.textContent); } catch (e) { return; }
      if (!schema || !schema.fields || !schema.outputs) return;
      var root = document.getElementById("calc-root");
      if (!root) root = s.parentNode;
      root.innerHTML = "";
      var inputs = {};
      schema.fields.forEach(function (f) {
        if (!f.id) return;
        var field = document.createElement("div");
        field.className = "field";
        var label = document.createElement("label"); label.textContent = f.label || f.id;
        var inp = document.createElement("input");
        inp.className = "input";
        inp.type = f.type === "number" ? "number" : "text";
        inp.value = f.default !== undefined ? f.default : "";
        inp.step = f.step || "any";
        inputs[f.id] = inp;
        field.appendChild(label); field.appendChild(inp);
        root.appendChild(field);
        inp.addEventListener("input", update);
      });
      var outs = document.createElement("div"); outs.id = "calc-outs"; root.appendChild(outs);
      outs.style.cssText = "border-top:1px solid var(--border);margin-top:10px;padding-top:10px";
      function update() {
        var vars = {};
        Object.keys(inputs).forEach(function (k) { vars[k] = parseFloat(inputs[k].value) || 0; });
        outs.innerHTML = "";
        schema.outputs.forEach(function (o) {
          var v = calcEval(o.expr || "0", vars);
          var row = document.createElement("div");
          row.style.cssText = "display:flex;justify-content:space-between;padding:4px 0";
          // o.label sunucudan gelen bileşen yapılandırmasıdır; textContent
          // ile yazılır (etiket içinde HTML yorumlanmasın).
          var ad = document.createElement("span");
          ad.textContent = o.label || "";
          var deger = document.createElement("strong");
          deger.textContent = isFinite(v) ? String(Math.round(v * 100) / 100) : "—";
          row.appendChild(ad);
          row.appendChild(deger);
          outs.appendChild(row);
        });
      }
      update();
    });
  }
  document.addEventListener("DOMContentLoaded", buildCalc);
})();

function tbUrlB64ToUint8Array(base64) {
  const padding = '='.repeat((4 - base64.length % 4) % 4);
  const b64 = (base64 + padding).replace(/-/g, '+').replace(/_/g, '/');
  const raw = atob(b64);
  const out = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}
window.tbPush = async function () {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    alert('Tarayıcınız web bildirimlerini desteklemiyor.');
    return;
  }
  try {
    const perm = await Notification.requestPermission();
    if (perm !== 'granted') { alert('Bildirim izni verilmedi.'); return; }
    const reg = await navigator.serviceWorker.register('/sw.js');
    const r = await fetch('/api/push/vapid', { headers: { 'Accept': 'application/json' } });
    const j = await r.json();
    if (!j.publicKey) { alert('Sunucu VAPID anahtarı vermedi.'); return; }
    const sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: tbUrlB64ToUint8Array(j.publicKey)
    });
    const res = await fetch('/api/push/abone', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(sub)
    });
    if (res.ok) { alert('Bildirimler açıldı. Teşekkürler!'); }
    else { alert('Abonelik kaydedilemedi.'); }
  } catch (e) {
    console.error(e); alert('Bildirim aboneliği başarısız.');
  }
};
(function () {
  var el = document.querySelector('[data-article-id]');
  if (!el) return;
  var sent = false;
  function check() {
    if (sent) return;
    var h = document.documentElement;
    var pct = (h.scrollTop + window.innerHeight) / (h.scrollHeight || 1);
    if (pct >= 0.5) {
      sent = true;
      try {
        fetch('/api/scroll/' + el.getAttribute('data-article-id'), {
          method: 'POST', credentials: 'same-origin', keepalive: true
        });
      } catch (e) {}
    }
  }
  window.addEventListener('scroll', check, { passive: true });
  window.addEventListener('load', check);
})();
(function () {
  var bar = document.getElementById('readingBar');
  var top = document.getElementById('toTop');
  function onScroll() {
    var h = document.documentElement;
    var max = h.scrollHeight - h.clientHeight;
    if (bar) bar.style.width = (max > 0 ? Math.min(100, (h.scrollTop / max) * 100) : 0) + '%';
    if (top) top.classList.toggle('show', h.scrollTop > 600);
  }
  window.addEventListener('scroll', onScroll, { passive: true });
  window.addEventListener('resize', onScroll, { passive: true });
  onScroll();
  if (top) top.addEventListener('click', function () { window.scrollTo({ top: 0, behavior: 'smooth' }); });

  function syncThemeColor() {
    var t = document.documentElement.getAttribute('data-theme') || 'light';
    var m = document.querySelector('meta[name=theme-color]');
    if (m) m.setAttribute('content', '#18181b');
  }
  syncThemeColor();
  try { new MutationObserver(syncThemeColor).observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] }); } catch (e) {}
})();
(function () {
  var box = document.getElementById('cmdk');
  var input = document.getElementById('cmdkInput');
  var list = document.getElementById('cmdkList');
  var btn = document.getElementById('cmdkBtn');
  if (!box || !input || !list) return;

  var RECENT = 'tb_cmdk_recent';
  var SVG_NS = 'http://www.w3.org/2000/svg';

  // Rotalar: menü kısayolları. `ikon` yerel sprite adı.
  var ROTALAR = [
    ['Pano', '/admin', 'layout-dashboard'],
    ['İçerikler', '/admin/icerikler', 'file-text'],
    ['Yeni içerik', '/admin/icerikler/yeni', 'file-plus'],
    ['Medya', '/admin/dosyalar', 'photo'],
    ['Kategoriler', '/admin/kategoriler', 'folder'],
    ['Etiketler', '/admin/etiketler', 'tag'],
    ['Mahreçler', '/admin/mahrecler', 'building'],
    ['Konu kümeleri', '/admin/kumeler', 'stack'],
    ['Manşet editörü', '/admin/manset-editor', 'layout-cards'],
    ['Servisler', '/admin/servisler', 'plug'],
    ['Gelen kutusu', '/admin/gelen-kutusu', 'inbox'],
    ['Yorumlar', '/admin/yorumlar', 'message-circle'],
    ['Geri bildirim', '/admin/geri-bildirim', 'message-dots'],
    ['Üyelikler', '/admin/members', 'users'],
    ['Anketler', '/admin/anketler', 'list-check'],
    ['Etkinlikler', '/admin/etkinlikler', 'calendar-event'],
    ['Bildirimler', '/admin/bildirim', 'bell'],
    ['AI sohbet', '/admin/ai-sohbet', 'message-chatbot'],
    ['SEO ve iç link', '/admin/seo', 'search'],
    ['İç link ağı', '/admin/backlink', 'link'],
    ['İndeksleme', '/admin/indexleme', 'world-search'],
    ['Taşıma ve 301', '/admin/tasima', 'arrows-exchange'],
    ['Sosyal paylaşım', '/admin/paylasim', 'share-3'],
    ['Bülten', '/admin/bulten', 'mail'],
    ['Gazete (e-baskı)', '/admin/gazete', 'news'],
    ['Gazete manşetleri', '/admin/gazeteler', 'news'],
    ['Şablon galerisi', '/admin/gazete/sablonlar', 'template'],
    ['Sinemalar', '/admin/sinemalar', 'movie'],
    ['Resmi ilanlar', '/admin/resmi-ilanlar', 'license'],
    ['Reklamlar', '/admin/reklamlar', 'speakerphone'],
    ['AdSense ve GA', '/admin/adsense', 'chart-bar'],
    ['Raporlar', '/admin/raporlar', 'chart-line'],
    ['Siteler', '/admin/siteler', 'world'],
    ['Bot', '/admin/bot', 'robot'],
    ['Tasarım', '/admin/tasarim', 'palette'],
    ['Marka', '/admin/marka', 'brush'],
    ['Menü', '/admin/menu', 'menu-2'],
    ['Ayarlar', '/admin/ayarlar', 'settings'],
    ['Görünüm', '/admin/gorunum', 'adjustments'],
    ['Çeviriler', '/admin/ceviriler', 'language'],
    ['Ses ayarları', '/admin/ses-ayarlari', 'microphone'],
    ['Yardım', '/admin/yardim', 'help-circle']
  ];

  // Hızlı eylemler: rota değil, niyet. Sorguyla eşleşince en üstte çıkar.
  var EYLEMLER = [
    ['Yeni haber oluştur', '/admin/icerikler/yeni', 'file-plus'],
    ['Gazete sayısı üret', '/admin/gazete', 'news'],
    ['Bülten gönder', '/admin/bulten', 'send'],
    ['Marka renklerini ayarla', '/admin/gorunum', 'brush'],
    ['Yorumları onayla', '/admin/yorumlar', 'message-circle'],
    ['Medya yükle', '/admin/dosyalar', 'upload']
  ];

  var sec = 0;
  var duz = []; // klavye için düz gezinme listesi: {href, ad}
  var istekNo = 0;
  var zamanlayici = null;

  // Türkçe arama katlaması.
  //
  // NEDEN: palet "Yardım" kaydını ararken kullanıcı "yardim" yazıyor
  // (noktasız ı klavyede pratik değil). toLocaleLowerCase('tr') "Yardım"ı
  // "yardım" yapar, sorgu "yardim" kalır — eşleşmez. Aynı sorun ş/ğ/ü/ö/ç
  // için de var. Eşleştirmede aksan ve nokta farkı yok sayılır; GÖRÜNTÜLENEN
  // metin değişmez, yalnız karşılaştırma katlanır.
  var KATLA = {
    'ı': 'i', 'İ': 'i', 'I': 'i', 'ş': 's', 'Ş': 's', 'ğ': 'g', 'Ğ': 'g',
    'ü': 'u', 'Ü': 'u', 'ö': 'o', 'Ö': 'o', 'ç': 'c', 'Ç': 'c', 'â': 'a', 'î': 'i', 'û': 'u'
  };
  function norm(v) {
    return (v || '').replace(/[ıİIşŞğĞüÜöÖçÇâîû]/g, function (c) {
      return KATLA[c] || c;
    }).toLowerCase();
  }

  function sonKullanilanlar() {
    try { return JSON.parse(localStorage.getItem(RECENT) || '[]') || []; } catch (e) { return []; }
  }
  function sonaEkle(href, ad) {
    var v = sonKullanilanlar().filter(function (x) { return x.href !== href; });
    v.unshift({ href: href, ad: ad });
    try { localStorage.setItem(RECENT, JSON.stringify(v.slice(0, 6))); } catch (e) {}
  }

  // Basit sıralama: baştan eşleşme > içinde eşleşme; kısa etiket önce.
  function skor(ad, q) {
    var a = norm(ad), s = norm(q);
    if (!s) return 0;
    var i = a.indexOf(s);
    if (i === 0) return 1000 - a.length;
    if (i > 0) return 500 - i - a.length;
    return 0;
  }

  function ikon(ad) {
    var svg = document.createElementNS(SVG_NS, 'svg');
    svg.setAttribute('class', 'ti');
    svg.setAttribute('aria-hidden', 'true');
    var use = document.createElementNS(SVG_NS, 'use');
    use.setAttribute('href', '/static/vendor/tabler-icons.svg?v=b89545a6#ti-' + (ad || 'arrow-right'));
    svg.appendChild(use);
    return svg;
  }

  // Eşleşen parçayı <mark> ile vurgular; metin düğüm olarak eklenir.
  function vurgula(hedef, metin, q) {
    var s = norm(q);
    var i = s ? norm(metin).indexOf(s) : -1;
    if (i < 0) { hedef.textContent = metin; return; }
    hedef.appendChild(document.createTextNode(metin.slice(0, i)));
    var m = document.createElement('mark');
    m.textContent = metin.slice(i, i + s.length);
    hedef.appendChild(m);
    hedef.appendChild(document.createTextNode(metin.slice(i + s.length)));
  }

  function ogeKur(o, q) {
    var li = document.createElement('li');
    li.className = 'cmdk-oge';
    li.setAttribute('role', 'option');
    li.dataset.href = o.href;
    li.dataset.ad = o.ad;

    var ik = document.createElement('span');
    ik.className = 'cmdk-oge__ikon';
    ik.appendChild(ikon(o.ikon));
    li.appendChild(ik);

    var g = document.createElement('span');
    g.className = 'cmdk-oge__govde';
    var b = document.createElement('span');
    b.className = 'cmdk-oge__ad';
    vurgula(b, o.ad, q);
    g.appendChild(b);
    if (o.alt) {
      var alt = document.createElement('span');
      alt.className = 'cmdk-oge__alt';
      alt.textContent = o.alt;
      g.appendChild(alt);
    }
    li.appendChild(g);

    if (o.rozet) {
      var r = document.createElement('span');
      r.className = 'badge cmdk-oge__rozet';
      r.textContent = o.rozet;
      li.appendChild(r);
    }
    var ok = document.createElement('svg');
    ok.setAttribute('class', 'ti cmdk-oge__ok');
    ok.setAttribute('aria-hidden', 'true');
    var u = document.createElementNS(SVG_NS, 'use');
    u.setAttribute('href', '/static/vendor/tabler-icons.svg?v=b89545a6#ti-corner-down-left');
    ok.appendChild(u);
    li.appendChild(ok);
    return li;
  }

  function grupBasligi(ad) {
    var h = document.createElement('li');
    h.className = 'cmdk-grup';
    h.setAttribute('role', 'presentation');
    h.textContent = ad;
    return h;
  }

  function bosDurum(baslik, aciklama) {
    var li = document.createElement('li');
    li.className = 'cmdk-bos';
    li.setAttribute('role', 'presentation');
    var svg = ikon('search-off');
    li.appendChild(svg);
    var b = document.createElement('strong');
    b.textContent = baslik;
    li.appendChild(b);
    var p = document.createElement('span');
    p.textContent = aciklama;
    li.appendChild(p);
    return li;
  }

  function ciz(basliklar, q) {
    list.textContent = '';
    duz = [];
    sec = 0;
    basliklar.forEach(function (gr) {
      if (!gr.oge.length) return;
      list.appendChild(grupBasligi(gr.ad));
      gr.oge.forEach(function (o) { list.appendChild(ogeKur(o, q)); duz.push(o); });
    });
    if (!duz.length) {
      list.appendChild(bosDurum('Sonuç yok', '"' + q + '" için içerik, kategori, medya ya da sayfa bulunamadı.'));
    }
    isaretle(0);
  }

  function isaretle(n) {
    var ogeler = list.querySelectorAll('.cmdk-oge');
    if (!ogeler.length) return;
    sec = (n + ogeler.length) % ogeler.length;
    for (var i = 0; i < ogeler.length; i++) {
      ogeler[i].classList.toggle('on', i === sec);
      ogeler[i].setAttribute('aria-selected', i === sec ? 'true' : 'false');
    }
    if (ogeler[sec].scrollIntoView) ogeler[sec].scrollIntoView({ block: 'nearest' });
  }

  function yerelGruplar(q) {
    var gruplar = [];
    var eylem = EYLEMLER
      .map(function (e) { return { ad: e[0], href: e[1], ikon: e[2], alt: 'Eylem', p: skor(e[0], q) }; })
      .filter(function (e) { return !q || e.p > 0; })
      .sort(function (a, b) { return b.p - a.p; });
    if (eylem.length) gruplar.push({ ad: 'Eylem', oge: q ? eylem.slice(0, 4) : [] });

    var sayfa = ROTALAR
      .map(function (r) { return { ad: r[0], href: r[1], ikon: r[2], alt: 'Sayfa', p: skor(r[0], q) }; })
      .filter(function (r) { return !q || r.p > 0; })
      .sort(function (a, b) { return b.p - a.p; });
    if (sayfa.length) gruplar.push({ ad: 'Sayfa', oge: sayfa.slice(0, q ? 6 : 40) });
    return gruplar;
  }

  function tazele(q) {
    if (!q) {
      var gruplar = yerelGruplar('');
      var son = sonKullanilanlar().map(function (x) {
        return { ad: x.ad, href: x.href, ikon: 'history', alt: 'Son kullanılan' };
      });
      if (son.length) gruplar.unshift({ ad: 'Son kullanılan', oge: son });
      ciz(gruplar, '');
      return;
    }
    ciz(yerelGruplar(q), q);

    // Sunucu araması: yarış koşuluna karşı sıra numarası.
    var no = ++istekNo;
    fetch('/admin/ara?q=' + encodeURIComponent(q), { credentials: 'same-origin' })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (d) {
        if (!d || no !== istekNo) return;
        var varlik = (d.gruplar || []).map(function (g) {
          return {
            ad: g.ad,
            oge: (g.oge || []).map(function (o) {
              return { ad: o.baslik, href: o.href, ikon: g.ikon, alt: o.alt, rozet: o.rozet };
            })
          };
        });
        // Varlık sonuçları rota sonuçlarından ÖNCE gelir: kullanıcı bir şey
        // arıyorsa çoğunlukla veri arıyordur, sayfa değil.
        var yerel = yerelGruplar(q);
        var eylem = yerel.filter(function (g) { return g.ad === 'Eylem'; });
        var sayfa = yerel.filter(function (g) { return g.ad === 'Sayfa'; });
        ciz(eylem.concat(varlik, sayfa), q);
      })
      .catch(function () {});
  }

  function ac() {
    box.classList.add('open');
    box.removeAttribute('aria-hidden');
    input.value = '';
    tazele('');
    input.focus();
  }
  function kapat() {
    box.classList.remove('open');
    box.setAttribute('aria-hidden', 'true');
    if (btn) btn.focus();
  }
  function git(href, ad) {
    if (!href) return;
    if (ad) sonaEkle(href, ad);
    kapat();
    window.location.href = href;
  }

  if (btn) btn.addEventListener('click', ac);
  box.setAttribute('aria-hidden', 'true');

  document.addEventListener('keydown', function (e) {
    if ((e.ctrlKey || e.metaKey) && (e.key === 'k' || e.key === 'K')) {
      e.preventDefault();
      box.classList.contains('open') ? kapat() : ac();
      return;
    }
    if (!box.classList.contains('open')) return;
    if (e.key === 'Escape') { e.preventDefault(); kapat(); }
    else if (e.key === 'ArrowDown') { e.preventDefault(); isaretle(sec + 1); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); isaretle(sec - 1); }
    else if (e.key === 'Enter') {
      e.preventDefault();
      var o = duz[sec];
      if (o) git(o.href, o.ad);
    }
  });

  input.addEventListener('input', function () {
    var q = input.value.trim();
    clearTimeout(zamanlayici);
    // 120 ms: her tuşta istek atmamak için; yazma hızında fark edilmez.
    zamanlayici = setTimeout(function () { tazele(q); }, 120);
  });

  list.addEventListener('click', function (e) {
    var li = e.target.closest('.cmdk-oge');
    if (li) git(li.dataset.href, li.dataset.ad);
  });
  list.addEventListener('mousemove', function (e) {
    var li = e.target.closest('.cmdk-oge');
    if (!li) return;
    var ogeler = [].slice.call(list.querySelectorAll('.cmdk-oge'));
    var i = ogeler.indexOf(li);
    if (i >= 0 && i !== sec) isaretle(i);
  });
  box.addEventListener('click', function (e) { if (e.target === box) kapat(); });
})();

(function () {
  document.querySelectorAll('.hero-slider').forEach(function (slider) {
    var allSlides = Array.prototype.slice.call(slider.querySelectorAll('.hero-slide'));
    if (!allSlides.length) return;
    var allDots = Array.prototype.slice.call(slider.querySelectorAll('.hero-dot'));
    var idx = 0, timer = null, lim = allSlides.length;
    var autoplay = slider.getAttribute('data-autoplay') === '1';
    var interval = (Number.parseInt(slider.getAttribute('data-interval'), 10) || 6) * 1000;
    function limitFor() {
      // Slider kolonu 2/3 genişlikte olabildiği için slider.clientWidth
      // kullanmak masaüstünde yanlışlıkla 8'e düşürüyordu. Ekran genişliği
      // esas alınır; mobile/tablet kırılımları tüm stiller için geçerlidir.
      var w = window.innerWidth || document.documentElement.clientWidth;
      if (w < 560) return Math.min(allSlides.length, 5);
      if (w < 900) return Math.min(allSlides.length, 8);
      return allSlides.length;
    }
    function active() { return Math.max(1, Math.min(lim, allSlides.length)); }
    function show(i) {
      var n = active();
      idx = ((i % n) + n) % n;
      allSlides.forEach(function (s, j) { s.classList.toggle('on', j === idx); });
      allDots.forEach(function (d, j) { d.classList.toggle('on', j === idx); });
    }
    function applyLimit() {
      lim = limitFor();
      allSlides.forEach(function (s, j) { s.classList.toggle('hero-off', j >= lim); });
      allDots.forEach(function (d, j) { d.classList.toggle('hero-off', j >= lim); });
      if (idx >= lim) idx = 0;
      show(idx);
    }
    function next() { show(idx + 1); }
    function stop() { if (timer) { clearInterval(timer); timer = null; } }
    function start() { if (autoplay && active() > 1) { stop(); timer = setInterval(next, interval); } }
    var prev = slider.querySelector('.hero-prev'), nx = slider.querySelector('.hero-next');
    if (prev) prev.addEventListener('click', function (e) { e.preventDefault(); show(idx - 1); start(); });
    if (nx) nx.addEventListener('click', function (e) { e.preventDefault(); show(idx + 1); start(); });
    allDots.forEach(function (d) {
      var go = function (e) {
        if (e) e.preventDefault();
        show(Number.parseInt(d.getAttribute('data-idx'), 10) || 0);
        start();
      };
      d.addEventListener('click', go);
      // Manşet II'deki gibi sayı üzerine gelince akıcı geçiş.
      d.addEventListener('mouseenter', function () { go(); });
    });
    slider.addEventListener('mouseenter', stop);
    slider.addEventListener('mouseleave', start);
    var x0 = null;
    slider.addEventListener('touchstart', function (e) { x0 = e.touches[0].clientX; }, { passive: true });
    slider.addEventListener('touchend', function (e) {
      if (x0 === null) return;
      var dx = e.changedTouches[0].clientX - x0;
      if (Math.abs(dx) > 40) { show(idx + (dx < 0 ? 1 : -1)); start(); }
      x0 = null;
    }, { passive: true });
    document.addEventListener('visibilitychange', function () { if (document.hidden) stop(); else start(); });
    var rt = null;
    window.addEventListener('resize', function () {
      clearTimeout(rt);
      rt = setTimeout(function () { applyLimit(); start(); }, 150);
    });
    applyLimit();
    show(0);
    start();
  });

  // 6'lı manşet: daralan ekranda slayt sayısını azalt (6 → 5 → 4) ki pager ve
  // başlık bozulmasın; tabletten mobile geçişte de düzgün kalır. Masaüstü
  // üst sınırı "20 üzeri" izniyle 48'e çıkar; dar ekranlar her zaman sıkı kalır.
  document.querySelectorAll('.m6').forEach(function (root) {
    var slides = Array.prototype.slice.call(root.querySelectorAll('.m6-slide'));
    var nums = Array.prototype.slice.call(root.querySelectorAll('.m6-num'));
    if (!slides.length) return;
    var hasAll = !!root.querySelector('.m6-slide-all');
    var allowOver = root.getAttribute('data-allow-over') === '1';
    var customSize = Number.parseInt(root.getAttribute('data-pager-size'), 10) || 0;
    function limitFor() {
      var w = window.innerWidth || document.documentElement.clientWidth;
      var base = w < 560 ? 4 : (w < 900 ? 5 : (allowOver ? 48 : 20));
      return Math.min(slides.length, hasAll ? base + 1 : base);
    }
    var idx = 0, timer = null, lim = slides.length;
    var autoplay = root.getAttribute('data-autoplay') === '1';
    var interval = (Number.parseInt(root.getAttribute('data-interval'), 10) || 6) * 1000;
    function active() { return Math.min(lim, slides.length); }
    function show(i) {
      var n = active(); if (n < 1) return;
      idx = ((i % n) + n) % n;
      slides.forEach(function (s, j) { s.classList.toggle('on', j === idx); });
      nums.forEach(function (d, j) { d.classList.toggle('on', j === idx); });
    }
    function applyLimit() {
      lim = limitFor();
      slides.forEach(function (s, j) { s.classList.toggle('m6-off', j >= lim); });
      nums.forEach(function (d, j) { d.classList.toggle('m6-off', j >= lim); });
      // Kullanıcı sayı büyüklüğü seçtiyse onu koru; seçmediyse dikey
      // pager'ı sığdırmak için otomatik küçült.
      if (customSize > 0) {
        root.style.setProperty('--m6-num-size', Math.max(14, Math.min(48, customSize)) + 'px');
        return;
      }
      if (window.matchMedia('(max-width: 900px)').matches) { root.style.removeProperty('--m6-num-size'); return; }
      var avail = (root.clientHeight || 0) - 68;
      var size = Math.floor(avail / Math.max(1, lim));
      if (size > 30) size = 30;
      if (size < 18) size = 18;
      root.style.setProperty('--m6-num-size', size + 'px');
    }
    function stop() { if (timer) { clearInterval(timer); timer = null; } }
    function start() { if (autoplay && active() > 1) { stop(); timer = setInterval(function () { show(idx + 1); }, interval); } }
    var prev = root.querySelector('.m6-prev'), nx = root.querySelector('.m6-next');
    if (prev) prev.addEventListener('click', function (e) { e.preventDefault(); show(idx - 1); start(); });
    if (nx) nx.addEventListener('click', function (e) { e.preventDefault(); show(idx + 1); start(); });
    nums.forEach(function (d) { d.addEventListener('click', function (e) { e.preventDefault(); show(Number.parseInt(d.getAttribute('data-idx'), 10) || 0); start(); }); });
    root.addEventListener('mouseenter', stop);
    root.addEventListener('mouseleave', start);
    var x0 = null;
    root.addEventListener('touchstart', function (e) { x0 = e.touches[0].clientX; }, { passive: true });
    root.addEventListener('touchend', function (e) {
      if (x0 === null) return;
      var dx = e.changedTouches[0].clientX - x0;
      if (Math.abs(dx) > 40) { show(idx + (dx < 0 ? 1 : -1)); start(); }
      x0 = null;
    }, { passive: true });
    document.addEventListener('visibilitychange', function () { if (document.hidden) stop(); else start(); });
    var rt = null;
    window.addEventListener('resize', function () {
      clearTimeout(rt);
      rt = setTimeout(function () {
        applyLimit();
        var n = active(); if (idx >= n) { idx = n - 1; }
        show(idx); start();
      }, 150);
    });
    applyLimit();
    show(0);
    start();
  });

  // Manşet II: sayı üzerine gelince geçiş, oklar ve dokunmatik kaydırma.
  document.querySelectorAll('.m2').forEach(function (root) {
    var slides = Array.prototype.slice.call(root.querySelectorAll('.m2-slide'));
    var nums = Array.prototype.slice.call(root.querySelectorAll('.m2-num'));
    if (!slides.length) return;
    var idx = 0;
    function show(i) {
      idx = ((i % slides.length) + slides.length) % slides.length;
      slides.forEach(function (s, j) { s.classList.toggle('on', j === idx); });
      nums.forEach(function (n, j) {
        n.classList.toggle('on', j === idx);
        n.setAttribute('aria-selected', j === idx ? 'true' : 'false');
      });
    }
    nums.forEach(function (n, i) {
      n.addEventListener('click', function () { show(i); });
      n.addEventListener('mouseenter', function () { show(i); });
    });
    var prev = root.querySelector('.m2-prev'), next = root.querySelector('.m2-next');
    if (prev) prev.addEventListener('click', function () { show(idx - 1); });
    if (next) next.addEventListener('click', function () { show(idx + 1); });
    var x0 = null;
    root.addEventListener('touchstart', function (e) { x0 = e.touches[0].clientX; }, { passive: true });
    root.addEventListener('touchend', function (e) {
      if (x0 === null) return;
      var dx = e.changedTouches[0].clientX - x0;
      if (Math.abs(dx) > 40) show(idx + (dx < 0 ? 1 : -1));
      x0 = null;
    }, { passive: true });
    show(0);
  });

  document.querySelectorAll('.kat-sekmeli').forEach(function (sec) {
    var tabs = sec.querySelectorAll('.kat-tab');
    var panels = sec.querySelectorAll('.kat-panel');
    function activate(t) {
      var slug = t.getAttribute('data-tab');
      tabs.forEach(function (x) { x.classList.toggle('on', x === t); });
      panels.forEach(function (p) { p.classList.toggle('on', p.getAttribute('data-panel') === slug); });
    }
    tabs.forEach(function (t) {
      t.addEventListener('click', function () { activate(t); });
      // haberpi ile birebir: sekmenin üzerine gelince otomatik geçiş
      t.addEventListener('mouseenter', function () { activate(t); });
    });

    // Sekme içi sayfalama: sağ/sol oklar ve noktalar 5'erli (blok "count")
    // gruplar arasında gezinir; son sayfadan sonra ilk sayfaya döner.
    panels.forEach(function (panel) {
      var pages = [].slice.call(panel.querySelectorAll('.kat-page'));
      if (pages.length < 2) return;
      var dots = [].slice.call(panel.querySelectorAll('.kat-dot'));
      var idx = 0;
      function show(n) {
        idx = ((n % pages.length) + pages.length) % pages.length;
        pages.forEach(function (pg, i) {
          pg.hidden = i !== idx;
          pg.classList.toggle('on', i === idx);
        });
        dots.forEach(function (d, i) {
          d.classList.toggle('on', i === idx);
          if (i === idx) { d.setAttribute('aria-current', 'true'); }
          else { d.removeAttribute('aria-current'); }
        });
      }
      panel.querySelectorAll('[data-knav]').forEach(function (b) {
        b.addEventListener('click', function (e) {
          e.preventDefault();
          show(idx + Number.parseInt(b.getAttribute('data-knav'), 10));
        });
      });
      dots.forEach(function (d, i) {
        d.addEventListener('click', function (e) {
          e.preventDefault();
          show(i);
        });
      });
      show(0);
    });
  });

  // Makale: yazı boyutu A− / A / A+
  var tools = document.querySelectorAll('[data-font]');
  if (tools.length) {
    var body = document.querySelector('.article .body');
    if (body) {
      var size = 1.05;
      var apply = function () { body.style.fontSize = size.toFixed(2) + 'rem'; };
      tools.forEach(function (b) {
        b.addEventListener('click', function () {
          var v = b.getAttribute('data-font');
          if (v === '+') size = Math.min(1.65, size + 0.1);
          else if (v === '-') size = Math.max(0.85, size - 0.1);
          else size = 1.05;
          apply();
        });
      });
    }
  }
})();

(function () {
  var dd = document.getElementById('navDiger');
  if (dd) {
    var ddBtn = dd.querySelector('.nav-dd-btn');
    function setOpen(open) { dd.classList.toggle('open', open); if (ddBtn) ddBtn.setAttribute('aria-expanded', open ? 'true' : 'false'); }
    if (ddBtn) ddBtn.addEventListener('click', function (e) { e.stopPropagation(); setOpen(!dd.classList.contains('open')); });
    document.addEventListener('click', function (e) { if (dd && !dd.contains(e.target)) setOpen(false); });
    // dokunmatik dışa dokunma
    document.addEventListener('touchstart', function (e) { if (dd && !dd.contains(e.target)) setOpen(false); }, { passive: true });
  }

  var search = document.getElementById('hpSearch');
  function openSearch() { if (!search) return; search.classList.add('open'); var i = search.querySelector('input'); if (i) setTimeout(function () { i.focus(); }, 30); }
  function closeSearch() { if (search) search.classList.remove('open'); }
  document.addEventListener('click', function (e) {
    if (e.target.closest('[data-search-btn]')) { e.preventDefault(); openSearch(); }
    else if (search && e.target === search) closeSearch();
  });

  var drawer = document.getElementById('hpDrawer');

  
  function drawerIsaretle() {
    if (!drawer) return;
    var yol = location.pathname.replace(/\/$/, '') || '/';
    var baglar = drawer.querySelectorAll('a[href]');
    Array.prototype.forEach.call(baglar, function (a) {
      var hedef;
      try { hedef = new URL(a.getAttribute('href'), location.origin).pathname.replace(/\/$/, '') || '/'; }
      catch (e) { return; }
      if (hedef === yol) a.setAttribute('aria-current', 'page');
      else a.removeAttribute('aria-current');
    });
  }

  function openDrawer() {
    if (drawer) { drawer.classList.add('open'); drawerIsaretle(); }
    document.body.style.overflow = 'hidden';
  }
  function closeDrawer() {
    if (drawer) drawer.classList.remove('open');
    document.body.style.overflow = '';
  }
  document.addEventListener('click', function (e) {
    if (e.target.closest('[data-menu-btn]')) { e.preventDefault(); openDrawer(); }
    else if (e.target.closest('[data-drawer-close]')) closeDrawer();
  });
  document.addEventListener('keydown', function (e) { if (e.key === 'Escape') { closeSearch(); closeDrawer(); } });

  
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', drawerIsaretle);
  } else {
    drawerIsaretle();
  }
})();

(function () {
  "use strict";
  var items = Array.prototype.slice.call(document.querySelectorAll('.nav-item.has-mega'));
  if (!items.length) return;
  var hoverTimer = null, closeTimer = null;

  function cancelHoverTimers() {
    if (hoverTimer) { clearTimeout(hoverTimer); hoverTimer = null; }
    if (closeTimer) { clearTimeout(closeTimer); closeTimer = null; }
  }

  function showMega(it) {
    var btn = it.querySelector('[data-mega-btn]');
    var panel = it.querySelector('[data-mega-panel]');
    if (!btn || !panel || it.classList.contains('mega-open')) return;
    closeAll(it);
    it.classList.add('mega-open');
    btn.setAttribute('aria-expanded', 'true');
    panel.hidden = false;
    syncSeritSpace();
  }

  function scheduleHoverOpen(it) {
    cancelHoverTimers();
    if (it.classList.contains('mega-open')) return;
    hoverTimer = setTimeout(function () { showMega(it); }, 80);
  }

  function scheduleHoverClose(it) {
    cancelHoverTimers();
    closeTimer = setTimeout(function () {
      var panel = it.querySelector('[data-mega-panel]');
      if (it.matches(':hover')) return;
      if (panel && panel.matches(':hover')) return;
      closeAll();
    }, 220);
  }

  function closeAll(except) {
    cancelHoverTimers();
    items.forEach(function (it) {
      if (it === except) return;
      it.classList.remove('mega-open');
      var b = it.querySelector('[data-mega-btn]');
      var p = it.querySelector('[data-mega-panel]');
      if (b) b.setAttribute('aria-expanded', 'false');
      if (p) p.hidden = true;
    });
    syncSeritSpace();
  }

  
  function syncSeritSpace() {
    var open = document.querySelector('.nav-mega-serit:not([hidden])');
    if (open) {
      // Şeridin dış yüksekliği kullanılmalı: üst kenarlık (3px) dahil
      // edilmezse hikâye/ilk modül şeridin altında 3px kalıyordu.
      var h = Math.ceil(open.getBoundingClientRect().height) || 46;
      document.body.style.setProperty('--mega-serit-h', h + 'px');
      document.body.classList.add('mega-serit-open');
      seritHizala(open);
      watchSerit(open);
    } else {
      document.body.classList.remove('mega-serit-open');
      document.body.style.removeProperty('--mega-serit-h');
    }
  }

  
  function seritHizala(open) {
    var inner = open.querySelector('.nav-mega-inner');
    if (!inner) return;
    if (inner.style.width) inner.style.width = '';
  }

  
  var seritObserver = null;
  function watchSerit(el) {
    if (typeof ResizeObserver === 'undefined' || !el) return;
    if (seritObserver && seritObserver.__el === el) return;
    if (seritObserver) seritObserver.disconnect();
    seritObserver = new ResizeObserver(function () {
      var open = document.querySelector('.nav-mega-serit:not([hidden])');
      if (!open) return;
      var h = Math.ceil(open.getBoundingClientRect().height) || 46;
      document.body.style.setProperty('--mega-serit-h', h + 'px');
      seritHizala(open);
    });
    seritObserver.__el = el;
    seritObserver.observe(el);
  }
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(function () { syncSeritSpace(); });
  }
  window.addEventListener('load', function () { syncSeritSpace(); });

  items.forEach(function (it) {
    var btn = it.querySelector('[data-mega-btn]');
    var panel = it.querySelector('[data-mega-panel]');
    if (!btn || !panel) return;
    btn.addEventListener('click', function (e) {
      // Dar ekranda "Diğer" menüsüne taşınmışsa normal bağlantı gibi çalışsın.
      if (it.closest('.nav-dd-menu')) return;
      e.preventDefault();
      var open = it.classList.contains('mega-open');
      closeAll(it);
      if (open) {
        it.classList.remove('mega-open');
        btn.setAttribute('aria-expanded', 'false');
        panel.hidden = true;
      } else {
        it.classList.add('mega-open');
        btn.setAttribute('aria-expanded', 'true');
        panel.hidden = false;
      }
      syncSeritSpace();
    });
    it.addEventListener('mouseenter', function () { scheduleHoverOpen(it); });
    it.addEventListener('mouseleave', function () { scheduleHoverClose(it); });
    panel.addEventListener('mouseenter', cancelHoverTimers);
    panel.addEventListener('mouseleave', function () { scheduleHoverClose(it); });
    panel.addEventListener('click', function (e) {
      if (e.target.closest('a') && !panel.classList.contains('nav-mega-serit')) closeAll();
    });
    var closeBtn = panel.querySelector('[data-mega-close]');
    if (closeBtn) closeBtn.addEventListener('click', function () { closeAll(); });
  });

  document.addEventListener('click', function (e) {
    // Sabit şerit ikinci menü gibi davranır; dışarı tıklama onu kapatmaz.
    if (document.querySelector('.nav-mega-serit:not([hidden])')) return;
    if (!e.target.closest('.nav-item.has-mega')) closeAll();
  });
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') closeAll();
  });
  window.addEventListener('resize', syncSeritSpace);
})();

(function () {
  "use strict";
  var drawer = document.getElementById('hpDrawer');
  if (!drawer) return;
  var panel = drawer.querySelector('.hp-drawer-panel');
  if (!panel) return;
  var opens = Array.prototype.slice.call(drawer.querySelectorAll('[data-subopen]'));
  var views = Array.prototype.slice.call(drawer.querySelectorAll('[data-subview]'));
  if (!opens.length || !views.length) return;

  function hideAll() {
    views.forEach(function (v) { v.hidden = true; });
    opens.forEach(function (b) { b.setAttribute('aria-expanded', 'false'); });
    panel.classList.remove('sub-open');
  }

  function open(slug, btn) {
    hideAll();
    var view = document.getElementById('hpSubView-' + slug);
    if (!view) return;
    view.hidden = false;
    panel.classList.add('sub-open');
    if (btn) btn.setAttribute('aria-expanded', 'true');
    if (typeof panel.scrollTo === 'function') panel.scrollTo(0, 0);
  }

  opens.forEach(function (btn) {
    btn.addEventListener('click', function (e) {
      e.preventDefault();
      e.stopPropagation();
      open(btn.getAttribute('data-subopen'), btn);
    });
  });

  drawer.querySelectorAll('[data-subback]').forEach(function (btn) {
    btn.addEventListener('click', function (e) {
      e.preventDefault();
      hideAll();
    });
  });

  drawer.querySelectorAll('[data-drawer-close]').forEach(function (btn) {
    btn.addEventListener('click', hideAll);
  });
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') hideAll();
  });

  // Açık sayfa bir alt kategoriyse ilgili alt menü panelini aç.
  var path = location.pathname.replace(/\/+$/, '');
  views.forEach(function (view) {
    var hit = Array.prototype.slice.call(view.querySelectorAll('a')).some(function (a) {
      return a.getAttribute('href') === path;
    });
    if (hit) {
      var slug = view.id.replace('hpSubView-', '');
      var btn = drawer.querySelector('[data-subopen="' + slug + '"]');
      open(slug, btn);
    }
  });
})();

(function () {
  var lists = document.querySelectorAll('[data-autoscroll]');
  if (!lists.length) return;
  var reduced = window.matchMedia &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  lists.forEach(function (list) {
    if (reduced) return;
    if (list.children.length < 4) return;
    var paused = false;
    var wrap = list.parentElement;   // .trend-scroll — sabit kayan pencere
    if (wrap) {
      wrap.addEventListener('mouseenter', function () { paused = true; });
      wrap.addEventListener('mouseleave', function () { paused = false; });
      wrap.addEventListener('touchstart', function () { paused = true; }, { passive: true });
    }
    list.addEventListener('focusin', function () { paused = true; });
    list.addEventListener('focusout', function () { paused = false; });
    var hidden = false;
    document.addEventListener('visibilitychange', function () { hidden = document.hidden; });

    var DUR = 600;      // kayma suresi (ms)
    var DELAY = 3200;   // iki kayma arasi (ms)
    list.style.willChange = 'transform';

    function step() {
      var first = list.firstElementChild;
      if (!first) { setTimeout(loop, DELAY); return; }
      var cs = getComputedStyle(list);
      var gap = parseFloat(cs.rowGap || cs.gap) || 0;
      var h = first.getBoundingClientRect().height + gap;

      // Alt boşluk oluşmasın: ilk öğenin klonunu sona ekle (kayma boyunca dolu kalsın)
      var clone = first.cloneNode(true);
      clone.setAttribute('aria-hidden', 'true');
      list.appendChild(clone);

      var done = false;
      function finish() {
        if (done) return;
        done = true;
        list.removeEventListener('transitionend', onEnd);
        list.style.transition = 'none';
        list.style.transform = 'translateY(0)';
        if (first.parentNode === list) list.removeChild(first); // ekran dışına çıkan ORIJINALI kaldır
        var ns = list.querySelectorAll('.st-n');
        for (var i = 0; i < ns.length; i++) { ns[i].textContent = String(i + 1); }
        void list.offsetHeight;       // reflow — sıçrama olmasın
        list.style.transition = '';
        setTimeout(loop, DELAY);
      }
      function onEnd(e) {
        if (e.target !== list) return; // yalnızca listenin kendi geçişi
        finish();
      }
      list.style.transition = 'transform ' + DUR + 'ms cubic-bezier(0.4, 0, 0.2, 1)';
      requestAnimationFrame(function () {
        list.style.transform = 'translateY(-' + h + 'px)';
      });
      list.addEventListener('transitionend', onEnd);
      setTimeout(finish, DUR + 150); // güvenlik ağı
    }
    function loop() {
      if (paused || hidden || !document.hasFocus()) { setTimeout(loop, DELAY); return; }
      step();
    }
    setTimeout(loop, DELAY);
  });
})();

var __tbHikayeInit = false;
function initHikaye() {
  if (__tbHikayeInit) return;
  var storiesEl = document.getElementById('stories');
  if (!storiesEl || typeof window.hikayeData === 'undefined' || !window.hikayeData.length) return;
  if (typeof Zuck === 'undefined') return;
  __tbHikayeInit = true;
  try {
    new Zuck(storiesEl, {
      skin: 'Snapgram',
      avatars: true,
      list: false,
      openEffect: true,
      cubeEffect: false, // yüzde tabanlı slider ile çalışmaz (kaynak ile aynı)
      autoFullScreen: false,
      backButton: true,
      previousTap: true,
      localStorage: false,
      language: {
        unmute: 'Sesi aç',
        keyboardTip: 'Sonraki için boşluk tuşuna bas',
        visitLink: 'Habere git',
        time: {
          ago: 'önce', hour: 'saat', hours: 'saat',
          minute: 'dakika', minutes: 'dakika', fromnow: 'sonra',
          seconds: 'saniye', yesterday: 'dün', tomorrow: 'yarın', days: 'gün'
        }
      },
      stories: window.hikayeData.map(function (h) {
        var baslik = h.baslik
          ? h.baslik.charAt(0).toLocaleUpperCase('tr-TR') + h.baslik.slice(1).toLocaleLowerCase('tr-TR')
          : h.baslik;
        return {
          id: h.id, photo: h.photo, name: h.name, link: h.link,
          lastUpdated: h.lastUpdated,
          items: [{
            id: h.itemId, type: 'photo', length: 5,
            
            src: h.photoBuyuk || h.photo, preview: h.photo,
            link: h.link, linkText: baslik,
            seen: false, time: h.time
          }]
        };
      })
    });
  } catch (e) {
    console.warn('Zuck hikaye yüklenemedi:', e);
  }

  // Modal slaytlarına bulanık arka plan ekle (haberpi styles.css görünümü)
  setTimeout(function () {
    var modal = document.getElementById('zuck-modal');
    if (!modal) return;

    function bulanikZemin() {
      modal.querySelectorAll('.item').forEach(function (item) {
        if (item.querySelector('.zuck-blur-bg')) return;
        var img = item.querySelector('img.media');
        if (!img || !img.src) return;
        var bg = document.createElement('div');
        bg.className = 'zuck-blur-bg';
        bg.style.backgroundImage = 'url("' + img.src.replace(/"/g, '%22') + '")';
        item.insertBefore(bg, item.firstChild);
      });
    }

    new MutationObserver(bulanikZemin).observe(modal, { childList: true, subtree: true });
    bulanikZemin();
  }, 300);
}

function __tbHikayeBaslat() { initHikaye(); }
document.addEventListener('DOMContentLoaded', __tbHikayeBaslat);
window.addEventListener('load', __tbHikayeBaslat);

(function () {
  "use strict";
  var timer = null;

  function dengele() {
    var nav = document.querySelector('.site-nav');
    var diger = document.getElementById('navDiger');
    if (!nav || !diger) return;
    if (nav.offsetWidth === 0) return; // mobilde menü gizliyken atla
    var menu = diger.querySelector('.nav-dd-menu');
    if (!menu) return;

    var wrap0 = nav.style.flexWrap;
    nav.style.flexWrap = 'nowrap';

    // 1) Diğer'de toplananları sırayla ana menüye geri koy (sıra korunur)
    Array.prototype.slice.call(menu.children).forEach(function (it) {
      nav.insertBefore(it, diger);
    });

    // 2) ana menüdeki tüm kategori öğeleri (doğrudan çocuk)
    var all = Array.prototype.filter.call(nav.children, function (c) {
      return c.classList && c.classList.contains('nav-item');
    });

    // 3) "data-tb-pin" ile işaretli kategoriler (ör. İlçeler, Sanat)
    //    HER ZAMAN "Diğer" alt menüsünde kalır — ekrana sığsa bile.
    var pinned = [], others = [];
    all.forEach(function (it) {
      (it.hasAttribute('data-tb-pin') ? pinned : others).push(it);
    });
    Array.prototype.slice.call(pinned).forEach(function (it) {
      menu.appendChild(it);
    });

    // 4) Diğer butonu: pin varsa açık kalır (boş değil), yoksa ölçüme göre
    diger.classList.toggle('tb-sakli', pinned.length === 0);

    // 5) geri kalan kategorilerden ekrana sığmayanları menüye aktar
    var k = others.length;
    while (k > 0 && nav.scrollWidth > nav.clientWidth) {
      k--;
      menu.insertBefore(others[k], menu.firstChild);
    }
    if (menu.childElementCount === 0) diger.classList.add('tb-sakli');
    nav.style.flexWrap = wrap0;
  }

  function dengeleYumusak() {
    if (timer) clearTimeout(timer);
    timer = setTimeout(dengele, 120);
  }

  document.addEventListener('DOMContentLoaded', dengeleYumusak);
  window.addEventListener('load', dengeleYumusak);
  window.addEventListener('resize', dengeleYumusak);
  window.addEventListener('orientationchange', dengeleYumusak);
  // Fontlar (Poppins) yüklendikten sonra genişlikler değişebilir — birkaç kez dene
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(dengeleYumusak);
  }
  setTimeout(dengeleYumusak, 400);
  setTimeout(dengeleYumusak, 1200);
})();

(function () {
  "use strict";

  function havaIkon(ac) {
    var s = String(ac || '').toLocaleLowerCase('tr-TR');
    var id;
    if (s.indexOf('kar') > -1) {
      id = 'snowing';
    } else if (s.indexOf('yağmur') > -1 || s.indexOf('saganak') > -1 || s.indexOf('çisenti') > -1 || s.indexOf('çis') > -1) {
      id = 'rainy';
    } else if (s.indexOf('fırtına') > -1 || s.indexOf('gök gürültüsü') > -1 || s.indexOf('şimşek') > -1) {
      id = 'thunderstorm';
    } else if (s.indexOf('sis') > -1 || s.indexOf('pus') > -1) {
      id = 'foggy';
    } else if (s.indexOf('bulut') > -1) {
      id = 'cloud';
    } else {
      id = 'sunny';
    }
    var paths = {
      sunny: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
      cloud: '<path d="M7 18h10a4 4 0 0 0 .6-7.96A6 6 0 0 0 6.2 9.2 4.5 4.5 0 0 0 7 18z"/>',
      rainy: '<path d="M7 16h10a4 4 0 0 0 .6-7.96A6 6 0 0 0 6.2 7.2 4.5 4.5 0 0 0 7 16z"/><path d="M8 19l-1 2M12 19l-1 2M16 19l-1 2"/>',
      snowing: '<path d="M12 3v18M4.2 7.5l15.6 9M19.8 7.5l-15.6 9"/><path d="M9 5l3 2 3-2M9 19l3-2 3 2"/>',
      thunderstorm: '<path d="M7 15h10a4 4 0 0 0 .6-7.96A6 6 0 0 0 6.2 6.2 4.5 4.5 0 0 0 7 15z"/><path d="M13 15l-2 4h3l-1 3"/>',
      foggy: '<path d="M4 9h16M4 13h16M6 17h12"/>'
    };
    return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' + (paths[id] || paths.sunny) + '</svg>';
  }

  function initHavaBari() {
    var sel = document.getElementById('hava-il-select');
    var ikon = document.getElementById('hava-ikon');
    var aciklama = document.getElementById('hava-aciklama');
    var sicaklik = document.getElementById('hava-sicaklik');
    var bilgi = document.getElementById('hava-bilgi');
    var loader = document.getElementById('hava-loader');
    if (!sel) return;
    if (ikon) ikon.innerHTML = havaIkon(aciklama ? aciklama.textContent : '');

    sel.addEventListener('change', function () {
      var slug = this.value;
      if (loader) loader.hidden = false;
      fetch('/api/hava/' + encodeURIComponent(slug))
        .then(function (r) { return r.json(); })
        .then(function (res) {
          if (loader) loader.hidden = true;
          if (!res || !res.ok) return;
          if (aciklama) aciklama.textContent = res.aciklama || '';
          if (sicaklik) sicaklik.textContent = Math.round(res.sicaklik) + '°C';
          if (ikon) ikon.innerHTML = havaIkon(res.aciklama || '');
          if (bilgi && res.slug) bilgi.setAttribute('href', '/hava-durumu');
        })
        .catch(function () { if (loader) loader.hidden = true; });
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initHavaBari);
  } else {
    initHavaBari();
  }
})();

(function () {
  function initVideoSure() {
    var tip = document.querySelector('select[name="tip"]');
    var alan = document.getElementById('vidSureField');
    if (!tip || !alan) return;
    function uyarla() {
      alan.hidden = tip.value !== 'video';
    }
    tip.addEventListener('change', uyarla);
    uyarla();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initVideoSure);
  } else {
    initVideoSure();
  }
})();

(function () {
  function initToplu() {
    var bar = document.getElementById('topluBar');
    if (!bar || bar.dataset.bound === '1') return;
    bar.dataset.bound = '1';
    var sayi = document.getElementById('topluSayi');
    var hepsi = document.getElementById('tumunuSec');
    var satirlar = document.querySelectorAll('.satirSec');
    var silBtn = bar.querySelector('[data-toplu-sil]');
    function say() {
      var n = 0;
      satirlar.forEach(function (c) { if (c.checked) n++; });
      if (sayi) sayi.textContent = n + ' içerik seçili';
      bar.hidden = n === 0;
      if (hepsi) hepsi.checked = n > 0 && n === satirlar.length;
    }
    satirlar.forEach(function (c) { c.addEventListener('change', say); });
    if (hepsi) hepsi.addEventListener('change', function () {
      satirlar.forEach(function (c) { c.checked = hepsi.checked; });
      say();
    });
    if (silBtn) silBtn.addEventListener('click', function (e) {
      if (!window.confirm('Seçili içerikler silinsin mi? Bu işlem geri alınamaz.')) e.preventDefault();
    });
    say();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initToplu);
  } else {
    initToplu();
  }
  document.addEventListener('htmx:afterSwap', initToplu);
  document.addEventListener('htmx:afterSettle', initToplu);
})();

(function () {
  function initMedyaToplu() {
    var bar = document.getElementById('medyaBar');
    if (!bar || bar.dataset.bound === '1') return;
    bar.dataset.bound = '1';
    var sayi = document.getElementById('medyaSayi');
    var hepsi = document.getElementById('medyaTumunuSec');
    var temizle = document.getElementById('medyaSecimTemizle');
    var kutular = Array.prototype.slice.call(document.querySelectorAll('.medyaSec'));
    var silBtn = bar.querySelector('[data-toplu-sil]');
    var lastIndex = -1;
    function syncSelected() {
      kutular.forEach(function (c) {
        var item = c.closest('.media-item');
        if (item) item.classList.toggle('selected', c.checked);
      });
    }
    function say() {
      var n = 0;
      kutular.forEach(function (c) { if (c.checked) n++; });
      if (sayi) sayi.textContent = n + ' medya seçili';
      bar.hidden = n === 0;
      if (hepsi) hepsi.checked = n > 0 && n === kutular.length;
      syncSelected();
    }
    kutular.forEach(function (c, index) {
      c.addEventListener('click', function (e) {
        if (e.shiftKey && lastIndex >= 0 && lastIndex !== index) {
          var start = Math.min(lastIndex, index);
          var end = Math.max(lastIndex, index);
          for (var i = start; i <= end; i += 1) kutular[i].checked = c.checked;
        }
        lastIndex = index;
        say();
      });
      c.addEventListener('change', say);
    });
    if (hepsi) hepsi.addEventListener('change', function () {
      kutular.forEach(function (c) { c.checked = hepsi.checked; });
      lastIndex = -1;
      say();
    });
    if (temizle) temizle.addEventListener('click', function () {
      kutular.forEach(function (c) { c.checked = false; });
      if (hepsi) hepsi.checked = false;
      lastIndex = -1;
      say();
    });
    if (silBtn) silBtn.addEventListener('click', function (e) {
      if (!window.confirm('Seçili medyayı silmek istediğinize emin misiniz? (kullanımdakiler atlanır — geri alınamaz)')) e.preventDefault();
    });
    say();
  }

  function initMedya() {
    initMedyaToplu();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initMedya);
  } else {
    initMedya();
  }
  document.addEventListener('htmx:afterSwap', initMedya);
  document.addEventListener('htmx:afterSettle', initMedya);
})();

(function () {
  function initMediaSelectPreviews(root) {
    (root || document).querySelectorAll('select.media-slot-select').forEach(function (select) {
      if (select.dataset.previewBound === '1') return;
      select.dataset.previewBound = '1';
      var preview = select.parentElement.querySelector('.media-slot-preview');
      if (!preview) return;
      function sync() {
        var opt = select.selectedOptions && select.selectedOptions[0];
        var src = opt && opt.value ? (opt.dataset.thumb || '') : '';
        preview.src = src;
        preview.hidden = !src;
      }
      select.addEventListener('change', sync);
      sync();
    });
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () { initMediaSelectPreviews(document); });
  } else {
    initMediaSelectPreviews(document);
  }
  document.addEventListener('htmx:afterSwap', function (e) { initMediaSelectPreviews(e.target || document); });
  document.addEventListener('htmx:afterSettle', function (e) { initMediaSelectPreviews(e.target || document); });
})();

(function () {
  function initYorumToplu() {
    var bar = document.getElementById('yorumBar');
    if (!bar || bar.dataset.bound === '1') return;
    bar.dataset.bound = '1';
    var sayi = document.getElementById('yorumSayi');
    var hepsi = document.getElementById('yorumTumunuSec');
    var kutular = document.querySelectorAll('.yorumSec');
    var silBtn = bar.querySelector('[data-toplu-yorum-sil]');
    function say() {
      var n = 0;
      kutular.forEach(function (c) { if (c.checked) n++; });
      if (sayi) sayi.textContent = n + ' yorum seçili';
      bar.hidden = n === 0;
      if (hepsi) hepsi.checked = n > 0 && n === kutular.length;
    }
    kutular.forEach(function (c) { c.addEventListener('change', say); });
    if (hepsi) hepsi.addEventListener('change', function () {
      kutular.forEach(function (c) { c.checked = hepsi.checked; });
      say();
    });
    if (silBtn) silBtn.addEventListener('click', function (e) {
      if (!window.confirm('Seçili yorumlar silinsin mi? Bu işlem geri alınamaz.')) e.preventDefault();
    });
    say();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initYorumToplu);
  } else {
    initYorumToplu();
  }
  document.addEventListener('htmx:afterSwap', initYorumToplu);
  document.addEventListener('htmx:afterSettle', initYorumToplu);
})();

(function () {
  function initAyarTabs() {
    var tabs = document.getElementById('ayar-tabs');
    if (!tabs || tabs.dataset.bound === '1') return;
    tabs.dataset.bound = '1';
    function show(sek) {
      document.querySelectorAll('section[data-sek]').forEach(function (s) {
        s.style.display = s.getAttribute('data-sek') === sek ? '' : 'none';
      });
      tabs.querySelectorAll('a').forEach(function (a) {
        a.classList.toggle('active', a.getAttribute('data-sek') === sek);
      });
      var input = document.querySelector('input[name=tab]');
      if (input) input.value = sek;
    }
    tabs.addEventListener('click', function (e) {
      var a = e.target.closest('a[data-sek]');
      if (a) { e.preventDefault(); show(a.getAttribute('data-sek')); }
    });
    show(tabs.getAttribute('data-aktif') || 'genel');
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initAyarTabs);
  } else {
    initAyarTabs();
  }
  document.addEventListener('htmx:afterSwap', initAyarTabs);
  document.addEventListener('htmx:afterSettle', initAyarTabs);
})();

(function () {
  document.addEventListener('click', function (e) {
    var t = e.target.closest('[data-test-post]');
    if (!t) return;
    if (t.hasAttribute('hx-post') && window.htmx) return;
    e.preventDefault();
    var f = document.createElement('form');
    f.method = 'POST';
    f.action = t.getAttribute('data-test-post');
    var i = document.createElement('input');
    i.type = 'hidden';
    i.name = 'kanal';
    i.value = t.getAttribute('data-kanal');
    f.style.display = 'none';
    f.appendChild(i);
    document.body.appendChild(f);
    f.submit();
  });
})();

(function () {
  "use strict";

  
  function tbAutoHideFlash(f) {
    if (!f || f.dataset.tbAutoHide) return;
    f.dataset.tbAutoHide = "1";
    var hideDelay = 6000;
    try { if (window.tbFlashHideMs > 0) hideDelay = window.tbFlashHideMs; } catch (e) {}
    setTimeout(function () {
      if (!f.parentNode) return;
      f.style.transition = "opacity .4s ease";
      f.style.opacity = "0";
      setTimeout(function () {
        if (f.parentNode) f.parentNode.removeChild(f);
      }, 420);
    }, hideDelay);
  }

  function tbAutoHideAll() {
    document.querySelectorAll(".admin-main .flash, main .flash, .admin-body .flash")
      .forEach(tbAutoHideFlash);
  }

  
  function tbToast(text) {
    var el = document.createElement("div");
    el.className = "flash error";
    el.textContent = text;
    el.style.cssText = "position:fixed;right:18px;bottom:18px;z-index:9999;max-width:360px";
    document.body.appendChild(el);
    tbAutoHideFlash(el);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", tbAutoHideAll);
  } else {
    tbAutoHideAll();
  }

  document.addEventListener("htmx:afterSwap", tbAutoHideAll);
  document.addEventListener("htmx:afterSettle", tbAutoHideAll);

  document.addEventListener("htmx:sendError", function () {
    tbToast("Ağ hatası — bağlantı kurulamadı.");
  });
  document.addEventListener("htmx:responseError", function (e) {
    var status = e.detail && e.detail.xhr ? e.detail.xhr.status : "";
    tbToast("Sunucu hatası" + (status ? " (" + status + ")" : "") + " — işlem tamamlanamadı.");
  });
})();

(function () {
  "use strict";
  var fs = null;

  document.addEventListener("htmx:beforeRequest", function () {
    var a = document.activeElement;
    if (!a || !a.id) return;
    fs = { id: a.id, tag: a.tagName, name: a.getAttribute("name") || "" };
  });

  document.addEventListener("htmx:afterSwap", function () {
    if (!fs) return;
    var t = document.getElementById(fs.id);
    if (t) {
      try { t.focus({ preventScroll: true }); } catch (e) { t.focus(); }
      if (t.tagName === "INPUT" || t.tagName === "TEXTAREA") {
        try { t.setSelectionRange(t.value.length, t.value.length); } catch (e) {}
      }
    }
    fs = null;
  });
})();

(function () {
  "use strict";
  var FAZLA = "tb_react_fazla", AZ = "tb_react_az";
  function liste(k) {
    try {
      var v = JSON.parse(localStorage.getItem(k) || "[]");
      return Array.isArray(v) ? v : [];
    } catch (e) { return []; }
  }
  function yaz(k, a) {
    try { localStorage.setItem(k, JSON.stringify(a)); } catch (e) {}
  }
  function idAl(a) {
    var href = a.getAttribute("href") || "";
    var m = href.match(/-(\d+)(?:[?#]|$)/);
    return m ? m[1] : "";
  }
  function svg() {
    return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M7 22H4a1 1 0 0 1-1-1v-8a1 1 0 0 1 1-1h3m0 10V12l4-8a2 2 0 0 1 2 2v4h5.5a2 2 0 0 1 2 2.3l-1.2 7a2 2 0 0 1-2 1.7H7z"/></svg>';
  }
  var toastTimer = null;
  function toast(text) {
    var t = document.getElementById("crToast");
    if (!t) {
      t = document.createElement("div");
      t.id = "crToast";
      t.className = "card-react-toast";
      t.setAttribute("role", "status");
      t.setAttribute("aria-live", "polite");
      document.body.appendChild(t);
    }
    t.textContent = text;
    t.classList.add("show");
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { t.classList.remove("show"); }, 2600);
  }
  function isaretle(shell, id) {
    var f = liste(FAZLA).indexOf(id) >= 0;
    var a = liste(AZ).indexOf(id) >= 0;
    shell.querySelectorAll(".cr-btn").forEach(function (b) {
      var on = (b.getAttribute("data-yon") === "fazla" && f) ||
               (b.getAttribute("data-yon") === "az" && a);
      b.classList.toggle("on", on);
      b.setAttribute("aria-pressed", on ? "true" : "false");
    });
  }
  function gonder(id, yon) {
    return fetch("/api/kart-tepki", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id: Number(id), yon: yon })
    });
  }
  function kur(a) {
    if (!a || a.closest(".card-react-shell")) return;
    var id = idAl(a);
    if (!id) return;
    var shell = document.createElement("div");
    shell.className = "card-react-shell";
    shell.classList.toggle("has-cover", !!a.querySelector("img"));
    a.parentNode.insertBefore(shell, a);
    shell.appendChild(a);
    var bar = document.createElement("div");
    bar.className = "card-react";
    bar.setAttribute("role", "group");
    bar.setAttribute("aria-label", "Bu haberle ilgili geri bildirim");
    // Etiketler çeviri sözlüğünden gelir ve panelden düzenlenebilir: HTML
    // olarak birleştirilmez, textContent ile yazılır (DOM XSS değil).
    function crBtn(yon, anahtar, varsayilan) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "cr-btn";
      b.setAttribute("data-yon", yon);
      b.setAttribute("aria-pressed", "false");
      var etiket = window.tbI18n(anahtar, varsayilan);
      b.title = etiket;
      b.innerHTML = svg(); // statik SVG (sabit dize)
      var sp = document.createElement("span");
      sp.textContent = etiket;
      b.appendChild(sp);
      return b;
    }
    bar.appendChild(crBtn("fazla", "related_more", "Buna benzer daha fazla"));
    bar.appendChild(crBtn("az", "related_less", "Buna benzer daha az"));
    shell.appendChild(bar);
    var yorum = document.createElement("a");
    yorum.className = "card-comment";
    yorum.href = (a.getAttribute("href") || "") + "#yorum-yaz";
    yorum.setAttribute("aria-label", "Yorumunuzu ekleyin");
    yorum.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7A8.38 8.38 0 0 1 4 11.5a8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/></svg><span>Yorumunuzu ekleyin</span>';
    shell.appendChild(yorum);
    isaretle(shell, id);
    bar.querySelectorAll(".cr-btn").forEach(function (btn) {
      btn.addEventListener("click", function (e) {
        e.preventDefault();
        e.stopPropagation();
        var yon = btn.getAttribute("data-yon");
        var fazla = liste(FAZLA), az = liste(AZ);
        var zaten = (yon === "fazla" && fazla.indexOf(id) >= 0) ||
                    (yon === "az" && az.indexOf(id) >= 0);
        fazla = fazla.filter(function (x) { return x !== id; });
        az = az.filter(function (x) { return x !== id; });
        var istek = zaten ? "iptal" : yon;
        if (!zaten) {
          if (yon === "fazla") fazla.push(id); else az.push(id);
        }
        yaz(FAZLA, fazla); yaz(AZ, az);
        isaretle(shell, id);
        toast(zaten ? "Geri bildirim kaldırıldı." :
          (yon === "fazla" ? "Buna benzer haberler daha fazla gösterilecek." : "Buna benzer haberler daha az gösterilecek."));
        gonder(id, istek).catch(function () {
          toast("Geri bildirim kaydedilemedi; bağlantınızı kontrol edin.");
        });
      });
    });
  }
  function tara(root) {
    (root || document).querySelectorAll("a.card[href]").forEach(kur);
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () { tara(document); });
  } else {
    tara(document);
  }
  var zaman = null;
  var goz = new MutationObserver(function () {
    if (zaman) clearTimeout(zaman);
    zaman = setTimeout(function () { tara(document); }, 120);
  });
  goz.observe(document.body, { childList: true, subtree: true });
})();

(function () {
  "use strict";

  function kurFlow(flow) {
    if (!flow || flow.dataset.akReady) return;
    flow.dataset.akReady = "1";

    var sinir = Number.parseInt(flow.getAttribute("data-ak-sinir"), 10) || 0;
    var oto = flow.getAttribute("data-ak-oto") === "1";
    var wrap = flow.parentNode ? flow.parentNode.querySelector(".ak-more-wrap") : null;
    var btn = wrap ? wrap.querySelector("[data-ak-more]") : null;
    var sentinel = flow.parentNode ? flow.parentNode.querySelector("[data-ak-sentinel]") : null;
    var endNote = flow.parentNode ? flow.parentNode.querySelector("[data-ak-end]") : null;
    var bas = Number.parseInt(flow.getAttribute("data-ak-bas"), 10) || 0;
    var url = flow.getAttribute("data-ak-url") || "";
    var yukleniyor = false;
    var bitti = false;

    function bitir() {
      bitti = true;
      if (btn) { btn.classList.add("is-done"); btn.disabled = true; }
      if (wrap) wrap.style.display = "none";
      if (sentinel) sentinel.style.display = "none";
      if (endNote) endNote.hidden = false;
    }

    function yukle() {
      if (bitti || yukleniyor || !url) return;
      if (sinir && bas >= sinir) { bitir(); return; }
      yukleniyor = true;
      if (btn) btn.classList.add("is-loading");
      var q = url + (url.indexOf("?") >= 0 ? "&" : "?") + "parca=1&bas=" + encodeURIComponent(bas);
      fetch(q, { headers: { "X-Requested-With": "XMLHttpRequest" }, credentials: "same-origin" })
        .then(function (r) {
          if (!r.ok) throw new Error("http " + r.status);
          var nextBas = Number.parseInt(r.headers.get("X-Akis-Bas"), 10);
          var son = r.headers.get("X-Akis-Son") === "1";
          return r.text().then(function (html) { return { html: html, nextBas: nextBas, son: son }; });
        })
        .then(function (res) {
          if (res.html) {
            var tmp = document.createElement("div");
            tmp.innerHTML = res.html;
            var eklenen = 0;
            while (tmp.firstElementChild) {
              flow.appendChild(tmp.firstElementChild);
              eklenen++;
            }
            if (!eklenen) res.son = true;
          }
          if (!isNaN(res.nextBas)) bas = res.nextBas;
          // İmleci DOM'a da yaz: durum denetlenebilir kalsın (E2E/QA).
          flow.setAttribute("data-ak-bas", String(bas));
          yukleniyor = false;
          if (btn) btn.classList.remove("is-loading");
          if (res.son || (sinir && bas >= sinir)) bitir();
        })
        .catch(function () {
          yukleniyor = false;
          if (btn) btn.classList.remove("is-loading");
        });
    }

    if (btn) {
      btn.addEventListener("click", function (e) { e.preventDefault(); yukle(); });
    }
    if (oto && "IntersectionObserver" in window && sentinel) {
      var io = new IntersectionObserver(function (entries) {
        entries.forEach(function (en) { if (en.isIntersecting) yukle(); });
      }, { rootMargin: "400px 0px" });
      io.observe(sentinel);
    }
    if (sinir && bas >= sinir) bitir();
  }

  function tara() {
    document.querySelectorAll("[data-ak-flow]").forEach(kurFlow);
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", tara);
  } else {
    tara();
  }
})();

