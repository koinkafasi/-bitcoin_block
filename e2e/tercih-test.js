const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// A-3 uçtan uca: tercih sunucuya yazılıyor mu, TEMİZ bir profilde
// (localStorage boş) sunucudan geliyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });

  async function oturum() {
    // Her çağrı AYRI tarayıcı bağlamı: localStorage paylaşılmaz.
    const ctx = await b.createBrowserContext();
    const p = await ctx.newPage();
    await p.setViewport({ width: 1500, height: 1000 });
    if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
    return { ctx, p };
  }

  // ---- 1. cihaz: grubu aç, menüyü daralt ----
  const A = await oturum();
  const hata = [];
  A.p.on('pageerror', (e) => hata.push(e.message));

  await A.p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 800));

  const metaVar = await A.p.evaluate(() => !!document.querySelector('meta[name="tb-prefs"]'));
  console.log('meta tb-prefs var:', metaVar);

  const gruplar = await A.p.evaluate(() => [...document.querySelectorAll('details.form-grup')].map((d) => ({
    ad: (d.querySelector('summary') || {}).textContent.trim().split('\n')[0],
    acik: d.open,
  })));
  console.log('gruplar (başlangıç):', JSON.stringify(gruplar));

  // "Künye" grubunu aç.
  await A.p.evaluate(() => {
    const d = [...document.querySelectorAll('details.form-grup')]
      .find((x) => (x.querySelector('summary') || {}).getAttribute('data-grup') === 'kunye');
    if (d) { d.open = true; }
  });
  // Menüyü daralt.
  await A.p.click('#adminRail');
  await new Promise((r) => setTimeout(r, 1200)); // 400 ms debounce + istek

  const giden = [];
  A.p.on('request', (r) => { if (r.method() === 'POST') giden.push(r.url()); });
  await new Promise((r) => setTimeout(r, 300));

  const lsA = await A.p.evaluate(() => ({
    grup: localStorage.getItem('tb_form_grup:kunye'),
    rail: localStorage.getItem('pn:rail'),
  }));
  console.log('1. cihaz localStorage:', JSON.stringify(lsA));

  // ---- 2. cihaz: TAZE profil, localStorage boş ----
  const B = await oturum();
  await B.p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 900));

  const metaB = await B.p.evaluate(() => {
    const m = document.querySelector('meta[name="tb-prefs"]');
    try { return m ? JSON.parse(m.content) : null; } catch (e) { return 'PARSE HATASI'; }
  });
  console.log('2. cihaz meta:', JSON.stringify(metaB));

  const durumB = await B.p.evaluate(() => {
    const d = [...document.querySelectorAll('details.form-grup')]
      .find((x) => (x.querySelector('summary') || {}).getAttribute('data-grup') === 'kunye');
    return {
      kunyeAcik: d ? d.open : null,
      menuDar: document.documentElement.classList.contains('admin-rail'),
      ls: localStorage.getItem('tb_form_grup:kunye'),
    };
  });
  console.log('2. cihaz durum (localStorage BOŞ olmalı, tercih sunucudan):', JSON.stringify(durumB));

  // Açık karşılaştırma: iki cihazın durumu aynı olmalı.
  const A2 = await A.p.evaluate(() => ({
    kunye: (() => {
      const d = [...document.querySelectorAll('details.form-grup')]
        .find((x) => (x.querySelector('summary') || {}).getAttribute('data-grup') === 'kunye');
      return d ? d.open : null;
    })(),
    dar: document.documentElement.classList.contains('admin-rail'),
  }));
  const eslesti = A2.kunye === durumB.kunyeAcik && A2.dar === durumB.menuDar;
  console.log('1. cihaz:', JSON.stringify(A2));
  console.log('EŞLEŞTİ:', eslesti, eslesti ? '✓' : '✗ — tercih taşınmadı');

  // Meta her panel sayfasında olmalı (pano, ayarlar).
  for (const yol of ['/admin', '/admin/ayarlar', '/admin/icerikler']) {
    await B.p.goto(BASE + yol, { waitUntil: 'domcontentloaded' });
    const varMi = await B.p.evaluate(() => !!document.querySelector('meta[name="tb-prefs"]'));
    console.log('  meta', yol, varMi ? 'var' : 'YOK');
  }

  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await A.ctx.close(); await B.ctx.close();
  await b.close();
})();
