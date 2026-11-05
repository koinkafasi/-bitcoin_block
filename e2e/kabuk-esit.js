const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

// Ölçüm: iki sayfa AYNI kabuğu kullanmalı + hareket katmanı çalışmalı.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  p.on('console', (m) => { if (m.type() === 'error') hata.push('console: ' + m.text()); });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const olc = async (yol) => {
    await p.goto(BASE + yol, { waitUntil: 'networkidle2', timeout: 60000 });
    await new Promise((r) => setTimeout(r, 700));
    return p.evaluate(() => {
      const g = (s) => {
        const e = document.querySelector(s);
        if (!e) return null;
        const r = e.getBoundingClientRect(), c = getComputedStyle(e);
        return { w: Math.round(r.width), h: Math.round(r.height), radius: c.borderRadius, bg: c.backgroundColor, pad: c.padding, gap: c.gap };
      };
      return {
        url: location.pathname,
        kabuk: {
          pnApp: !!document.querySelector('.pn-app'),
          adminWrap: !!document.querySelector('.admin-wrap'),
          side: !!document.querySelector('.admin-side'),
        },
        side: g('.admin-side'),
        top: g('.admin-top'),
        body: g('.admin-body'),
        pill: g('.search-pill'),
        crumb: !!document.querySelector('.admin-crumb'),
        font: getComputedStyle(document.body).fontFamily.split(',')[0].replace(/"/g, ''),
        h1: (document.querySelector('.admin-top h1') || {}).textContent,
        // hareket katmanı
        animKart: (() => {
          const c = document.querySelector('.admin-body > *, .pn-kpis > .pn-kpi');
          if (!c) return null;
          const s = getComputedStyle(c);
          return { ad: s.animationName, sure: s.animationDuration, gecikme: s.animationDelay };
        })(),
        progressVar: !!document.querySelector('.pn-progress') || typeof window.tbToast === 'function',
        toastVar: typeof window.tbToast,
        vtDestek: typeof document.startViewTransition === 'function',
        tasma: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
        kpi: document.querySelectorAll('.pn-kpi, .kpi').length,
      };
    });
  };

  const a = await olc('/admin');
  const b2 = await olc('/admin/icerikler');

  const ayni = (k) => JSON.stringify(a[k]) === JSON.stringify(b2[k]);
  console.log('=== KABUK AYNI MI ===');
  for (const k of ['side', 'top', 'pill']) {
    const ok = ayni(k);
    console.log(`  ${ok ? '✓' : '✗'} ${k.padEnd(5)} /admin: ${JSON.stringify(a[k])}`);
    if (!ok) console.log(`          /icerikler: ${JSON.stringify(b2[k])}`);
  }
  console.log(`  ${a.font === b2.font ? '✓' : '✗'} font : ${a.font} / ${b2.font}`);
  const bp = (o) => o.body ? o.body.pad + ' · gap ' + o.body.gap : 'yok';
  console.log(`  ${bp(a) === bp(b2) ? '✓' : '✗'} body : ${bp(a)} / ${bp(b2)}`);
  console.log('  pano kabuğu (.pn-app):', a.kabuk.pnApp, '· ortak kabuk:', a.kabuk.adminWrap);

  console.log('\n=== HAREKET KATMANI ===');
  console.log('  kart animasyonu /admin     :', JSON.stringify(a.animKart));
  console.log('  kart animasyonu /icerikler :', JSON.stringify(b2.animKart));
  console.log('  tbToast /admin     :', a.toastVar, '· /icerikler:', b2.toastVar);
  console.log('  view transitions   :', a.vtDestek);
  console.log('  taşma              :', a.tasma || b2.tasma ? 'VAR' : 'yok');
  console.log('  KPI /admin         :', a.kpi);

  // Hareket davranışları: kaydırma, toast, htmx çubuğu
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 400));
  await p.evaluate(() => window.scrollTo(0, 400));
  await new Promise((r) => setTimeout(r, 250));
  const stuck = await p.evaluate(() => document.querySelector('.admin-top').classList.contains('is-stuck'));
  console.log('  kaydırınca üst çubuk yükseldi:', stuck);

  const toast = await p.evaluate(async () => {
    window.tbToast('Deneme balonu', 'ok');
    await new Promise((r) => setTimeout(r, 120));
    const t = document.querySelector('.pn-toast');
    return t ? { metin: t.textContent, sinif: t.className } : null;
  });
  console.log('  toast              :', JSON.stringify(toast));

  // HTMX panel değişimi: sekme tıklaması
  await p.evaluate(() => window.scrollTo(0, 0));
  const htmxOnce = await p.evaluate(() => document.querySelectorAll('.table tbody tr').length);
  const sekme = await p.$('.tabs a:not(.active)');
  if (sekme) {
    await sekme.click();
    await new Promise((r) => setTimeout(r, 1400));
  }
  const htmxSonra = await p.evaluate(() => ({
    satir: document.querySelectorAll('.table tbody tr').length,
    url: location.pathname + location.search,
    panelVar: !!document.querySelector('#icerik-panel'),
    h1: (document.querySelector('.admin-top h1') || {}).textContent,
  }));
  console.log('  HTMX sekme         :', htmxOnce, '→', htmxSonra.satir, 'satır · panel', htmxSonra.panelVar, '·', htmxSonra.url.slice(0, 40));

  console.log('\nJS hatası:', hata.length ? [...new Set(hata)].slice(0, 4).join(' | ') : 'yok');
  await b.close();
})();
