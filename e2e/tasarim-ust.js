const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const r = await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1400));

  // 1) Ust alan: admin-body'nin dogrudan cocuklari, yukseklikleri
  const d = await p.evaluate(() => {
    const out = [];
    const kok = document.querySelector('.admin-body');
    Array.from(kok.children).forEach((el, i) => {
      const rr = el.getBoundingClientRect();
      if (rr.height < 2) return;
      out.push({
        i,
        el: el.tagName + (el.id ? '#' + el.id : '') + '.' + el.className.toString().split(' ').slice(0, 2).join('.'),
        h: Math.round(rr.height),
        y: Math.round(rr.y),
        metin: (el.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 46),
      });
    });
    return out;
  });
  console.log('=== ÜST ALAN (admin-body çocukları) ===');
  d.forEach((x) => console.log('  ' + JSON.stringify(x)));
  console.log('  TOPLAM ilk bloklar:', d.reduce((a, x) => a + x.h, 0));

  // 2) Sabit renkli (tema-bagimsiz) metinler: koyu temada okunmuyor
  await p.evaluate(() => document.documentElement.setAttribute('data-theme', 'dark'));
  await new Promise((x) => setTimeout(x, 400));

  const sabit = await p.evaluate(() => {
    const out = [];
    const gorunur = (el) => {
      const s = getComputedStyle(el);
      return s.display !== 'none' && s.visibility !== 'hidden' && el.getClientRects().length > 0;
    };
    // Tema degiskenleri degisirken RENGI DEGISMEYEN ogeler = sabit kodlanmis.
    // Iki temada da ayni renk -> supheli (koyu temada koyu metin).
    const secilen = [];
    const sec = ['.bp-zone-h', '.bp-colctl', '.bp-chipbar', '.bp-chipbar .ty', '.bp-chipbar .ops button',
      '.bp-zone', '.bp-page', '.bp-ph', '.rt-editor', '.rt-toolbar', '.rt-toolbar button',
      '.vh-types', '.vh-types span', '.pl-hidden-tag', '.bld-history-item .vh-types',
      '.manset-cat', '.catview-tag', '.ab-badge', '.ct-demo', '.hs-demo', '.bld-field label', '.muted', '.hint'];
    for (const s of sec) {
      document.querySelectorAll(s).forEach((el) => {
        if (!gorunur(el)) return;
        const cs = getComputedStyle(el);
        secilen.push({ sec: s, renk: cs.color, zemin: cs.backgroundColor, metin: (el.textContent || '').trim().slice(0, 26) });
      });
    }
    return secilen.slice(0, 60);
  });

  console.log('\n=== KOYU TEMADA renkler (sabit kodlanmış olanlar) ===');
  const gorulen = new Set();
  sabit.forEach((x) => {
    const k = x.sec + '|' + x.renk + '|' + x.zemin;
    if (gorulen.has(k)) return;
    gorulen.add(k);
    console.log('  ' + JSON.stringify(x));
  });

  await b.close();
})();
