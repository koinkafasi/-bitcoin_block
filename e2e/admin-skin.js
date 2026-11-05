// Panelin TAMAMINDA Pacely görünümünü ölçer: kabuk, kart, KPI, tablo, form.
// Görsel yargı yerine geometri + renk + JS hatası.
const puppeteer = require('puppeteer');
const fs = require('fs');

const BASE = process.argv[2] || 'http://127.0.0.1:8093';
const SID = process.argv[3] || '';
const OUT = '/tmp/admin-skin';

const SAYFALAR = [
  '/admin', '/admin/icerikler', '/admin/kategoriler', '/admin/dosyalar',
  '/admin/yorumlar', '/admin/ayarlar', '/admin/resmi-ilanlar', '/admin/raporlar',
  '/admin/seo', '/admin/bot', '/admin/aboneler', '/admin/reklamlar',
];

(async () => {
  fs.mkdirSync(OUT, { recursive: true });
  const browser = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage'] });
  const page = await browser.newPage();
  await page.setViewport({ width: 1500, height: 1000 });
  const hatalar = [];
  page.on('pageerror', (e) => hatalar.push(String(e.message)));

  if (SID) await page.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await page.goto(BASE + SAYFALAR[0], { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 700));

  const kabuk = await page.evaluate(() => {
    const g = (s) => {
      const e = document.querySelector(s);
      if (!e) return null;
      const r = e.getBoundingClientRect();
      const c = getComputedStyle(e);
      return { w: Math.round(r.width), h: Math.round(r.height), radius: c.borderRadius, bg: c.backgroundColor, shadow: c.boxShadow.slice(0, 40), pad: c.padding };
    };
    return {
      side: g('.admin-side'),
      top: g('.admin-top'),
      body: g('.admin-body'),
      arama: g('#cmdkBtn'),
      btn: g('.admin-top .icon-btn:not(#cmdkBtn)'),
      card: g('.card'),
      kpi: g('.kpi'),
      kpiRenk: (() => {
        const k = [...document.querySelectorAll('.kpi-grid > .kpi')].map((e) => getComputedStyle(e).backgroundColor);
        return k;
      })(),
      navAktif: (() => {
        const a = document.querySelector('.admin-side .menu a.active');
        if (!a) return null;
        const c = getComputedStyle(a);
        return { metin: a.textContent.trim().slice(0, 22), bg: c.backgroundColor, radius: c.borderRadius, weight: c.fontWeight };
      })(),
      tablo: g('.table'),
      th: (() => { const t = document.querySelector('.table th'); if (!t) return null; const c = getComputedStyle(t); return { size: c.fontSize, weight: c.fontWeight, transform: c.textTransform }; })(),
      input: g('.input, .select'),
      btnDugme: g('.btn'),
      font: getComputedStyle(document.body).fontFamily.split(',')[0],
    };
  });

  console.log('=== KABUK ===');
  console.log('yan menü  :', kabuk.side && `${kabuk.side.w}px · bg ${kabuk.side.bg}`);
  console.log('üst çubuk :', kabuk.top && `${kabuk.top.h}px · bg ${kabuk.top.bg}`);
  console.log('gövde pad :', kabuk.body && kabuk.body.pad);
  console.log('arama hapı:', kabuk.arama && `${kabuk.arama.w}x${kabuk.arama.h} · radius ${kabuk.arama.radius}`);
  console.log('ikon düğme:', kabuk.btn && `${kabuk.btn.w}x${kabuk.btn.h} · radius ${kabuk.btn.radius}`);
  console.log('kart      :', kabuk.card && `radius ${kabuk.card.radius} · ${kabuk.card.pad}`);
  console.log('nav aktif :', JSON.stringify(kabuk.navAktif));
  console.log('kpi renk  :', JSON.stringify(kabuk.kpiRenk));
  console.log('tablo th  :', JSON.stringify(kabuk.th));
  console.log('input     :', kabuk.input && `radius ${kabuk.input.radius}`);
  console.log('btn       :', kabuk.btnDugme && `radius ${kabuk.btnDugme.radius} · h ${kabuk.btnDugme.h}`);
  console.log('font      :', kabuk.font);

  console.log('\n=== SAYFALAR ===');
  for (const yol of SAYFALAR) {
    await page.goto(BASE + yol, { waitUntil: 'networkidle2', timeout: 45000 });
    await new Promise((r) => setTimeout(r, 350));
    const o = await page.evaluate(() => ({
      baslik: (document.querySelector('.admin-top h1') || {}).textContent || '',
      kart: document.querySelectorAll('.card').length,
      kpi: document.querySelectorAll('.kpi').length,
      tablo: document.querySelectorAll('.table').length,
      form: document.querySelectorAll('form').length,
      radiusKart: (() => { const c = document.querySelector('.card'); return c ? getComputedStyle(c).borderRadius : '-'; })(),
      yatay: document.documentElement.scrollWidth > document.documentElement.clientWidth,
      adim: document.querySelectorAll('.tabs a, .tabs button').length,
    }));
    const ad = yol.replace('/admin', '') || '/';
    console.log(`${ad.padEnd(20)} ${o.baslik.slice(0, 22).padEnd(24)} kart ${String(o.kart).padStart(2)} · kpi ${o.kpi} · tablo ${o.tablo} · form ${o.form} · radius ${o.radiusKart}${o.yatay ? ' · ⚠ YATAY TAŞMA' : ''}`);
    if (['/admin', '/admin/icerikler', '/admin/ayarlar'].includes(yol)) {
      await page.screenshot({ path: `${OUT}/${ad.replace(/\//g, '_') || 'pano'}.png`, fullPage: false });
    }
  }

  // Ray katlama
  await page.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 400));
  const once = await page.evaluate(() => document.querySelector('.admin-side').getBoundingClientRect().width);
  await page.click('#adminRail');
  await new Promise((r) => setTimeout(r, 420));
  const sonra = await page.evaluate(() => ({
    w: document.querySelector('.admin-side').getBoundingClientRect().width,
    dar: document.body.classList.contains('admin-rail'),
    kayit: localStorage.getItem('pn:rail'),
  }));
  console.log('\n=== RAY ===');
  console.log(`${Math.round(once)} → ${Math.round(sonra.w)}px · dar ${sonra.dar} · kayıt ${sonra.kayit}`);

  console.log('\nJS hatası:', hatalar.length ? hatalar.slice(0, 4).join(' | ') : 'yok');
  await browser.close();
})();
