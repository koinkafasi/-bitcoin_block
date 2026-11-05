const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 186: medya galerisi kart gorunumu + surukleme + yukleme halkasi.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const r = await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' }).catch(() => null);
  await new Promise((x) => setTimeout(x, 900));
  if (!r || r.status() !== 200) { console.log('atlandı', r ? r.status() : ''); await b.close(); return; }

  for (const g of [1440, 1024, 820]) {
    await p.setViewport({ width: g, height: 1000 });
    await new Promise((x) => setTimeout(x, 500));

    const d = await p.evaluate(() => {
      const panel = document.getElementById('medya-panel');
      const grid = panel.querySelector('.media-grid');
      if (!grid) return { yok: true };
      const gs = getComputedStyle(grid);
      const kart = grid.querySelector('.media-item');
      const drop = grid.querySelector('.mgal-drop');
      const ks = kart ? getComputedStyle(kart) : null;
      const kartR = kart ? kart.getBoundingClientRect() : null;
      const dropR = drop ? drop.getBoundingClientRect() : null;
      const cap = grid.querySelector('.mi-cap');
      const capS = cap ? getComputedStyle(cap) : null;
      const sec = grid.querySelector('.mi-sec');
      const secS = sec ? getComputedStyle(sec) : null;
      return {
        sutun: gs.gridTemplateColumns.split(' ').length,
        autoRows: gs.gridAutoRows,
        colGap: gs.columnGap,
        kart: kartR ? { w: Math.round(kartR.width), h: Math.round(kartR.height), span: kart.style.gridRowEnd } : null,
        kRadius: ks ? ks.borderRadius : null,
        kShadow: ks ? ks.boxShadow.slice(0, 70) : null,
        drop: dropR ? { w: Math.round(dropR.width), h: Math.round(dropR.height), span: drop.style.gridRowEnd } : null,
        dropBtn: (() => {
          const btn = grid.querySelector('.mgal-drop__btn');
          if (!btn) return null;
          const s = getComputedStyle(btn);
          return { border: s.borderWidth + ' ' + s.borderStyle, radius: s.borderRadius, bg: s.backgroundColor };
        })(),
        ico: (() => {
          const el = grid.querySelector('.mgal-drop__ico');
          if (!el) return null;
          const s = getComputedStyle(el);
          const rr = el.getBoundingClientRect();
          return { w: Math.round(rr.width), h: Math.round(rr.height), radius: s.borderRadius, bg: s.backgroundColor, color: s.color };
        })(),
        cap: capS ? { bg: capS.backgroundImage.slice(0, 60), color: capS.color, pad: capS.padding } : null,
        secOpacity: secS ? secS.opacity : null,
        secSize: sec ? Math.round(sec.getBoundingClientRect().width) : null,
        bodyGorunur: (() => {
          const body = grid.querySelector('.media-item .mi-body');
          return body ? getComputedStyle(body).display !== 'none' : null;
        })(),
        kartSayisi: grid.querySelectorAll('.media-item').length,
      };
    });

    console.log('\n### ' + g + 'px');
    console.log(JSON.stringify(d, null, 1));
  }

  // Surukleme durumu
  const drag = await p.evaluate(() => {
    const panel = document.getElementById('medya-panel');
    const dt = new DataTransfer();
    dt.items.add(new File(['x'], 'test.jpg', { type: 'image/jpeg' }));
    panel.dispatchEvent(new DragEvent('dragenter', { bubbles: true, dataTransfer: dt }));
    const btn = panel.querySelector('.mgal-drop__btn');
    const s = getComputedStyle(btn);
    const sinif = panel.className;
    panel.dispatchEvent(new DragEvent('dragleave', { bubbles: true, dataTransfer: dt }));
    return { sinif, bg: s.backgroundColor, borderStyle: s.borderStyle, borderColor: s.borderColor, sonra: panel.className };
  });
  console.log('\n=== SURUKLEME (is-drag) ===');
  console.log(JSON.stringify(drag, null, 1));

  // Yukleme karti: JS fonksiyonunu dogrudan cagirmadan, sahte bir dosya ile
  // ilerleme kartini olusturup olcelim (XHR gercek istek atmadan once).
  const up = await p.evaluate(async () => {
    const panel = document.getElementById('medya-panel');
    const grid = panel.querySelector('.media-grid');
    if (!grid) return { yok: true };
    const CEVRE = 2 * Math.PI * 22;
    const el = document.createElement('figure');
    el.className = 'mgal-up';
    el.innerHTML =
      '<div class="mgal-up__card">' +
      '<svg class="mgal-up__ring" viewBox="0 0 52 52" aria-hidden="true">' +
      '<circle cx="26" cy="26" r="22" class="mgal-up__track"></circle>' +
      '<circle cx="26" cy="26" r="22" class="mgal-up__arc" stroke-dasharray="' + CEVRE + '" stroke-dashoffset="' + CEVRE + '"></circle>' +
      '</svg><b class="mgal-up__pct">0%</b><small class="mgal-up__name">deneme-fotograf.jpg</small></div>';
    const drop = grid.querySelector('.mgal-drop');
    grid.insertBefore(el, drop ? drop.nextSibling : grid.firstChild);
    // %45 goster
    const arc = el.querySelector('.mgal-up__arc');
    arc.style.strokeDashoffset = String(CEVRE * (1 - 0.45));
    el.querySelector('.mgal-up__pct').textContent = '45%';
    await new Promise((r) => setTimeout(r, 200));
    const ring = el.querySelector('.mgal-up__ring');
    const rs = getComputedStyle(ring);
    const rr = ring.getBoundingClientRect();
    return {
      ring: { w: Math.round(rr.width), h: Math.round(rr.height), transform: rs.transform },
      arcStroke: getComputedStyle(arc).stroke,
      arcWidth: getComputedStyle(arc).strokeWidth,
      dasharray: arc.getAttribute('stroke-dasharray'),
      dashoffset: arc.getAttribute('stroke-dashoffset'),
      pct: el.querySelector('.mgal-up__pct').textContent,
      ad: el.querySelector('.mgal-up__name').textContent,
      span: el.style.gridRowEnd,
    };
  });
  console.log('\n=== YUKLEME KARTI ===');
  console.log(JSON.stringify(up, null, 1));

  console.log('\nJS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
