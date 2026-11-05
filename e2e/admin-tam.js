const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];
const ROTALAR = [
  '/admin', '/admin/merkez', '/admin/pano', '/admin/icerikler', '/admin/icerikler/yeni',
  '/admin/kategoriler', '/admin/dosyalar', '/admin/yorumlar', '/admin/geri-bildirim',
  '/admin/ayarlar', '/admin/gorunum', '/admin/marka', '/admin/menu', '/admin/header',
  '/admin/footer', '/admin/tasarim', '/admin/manset-editor', '/admin/resmi-ilanlar',
  '/admin/raporlar', '/admin/seo', '/admin/bot', '/admin/ai-kaynak', '/admin/aboneler',
  '/admin/reklamlar', '/admin/bulten', '/admin/gazete', '/admin/siteler', '/admin/iletisim',
  '/admin/etiketler', '/admin/mahrecler', '/admin/ceviriler', '/admin/uyeler',
];
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hatalar = [], kotu = [];
  p.on('pageerror', (e) => hatalar.push(String(e.message)));
  p.on('response', (r) => { if (r.status() >= 500) kotu.push(r.status() + ' ' + r.url()); });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let tasma = 0, hata = 0;
  for (const yol of ROTALAR) {
    const resp = await p.goto(BASE + yol, { waitUntil: 'domcontentloaded', timeout: 45000 }).catch(() => null);
    await new Promise((r) => setTimeout(r, 320));
    const st = resp ? resp.status() : 0;
    const o = await p.evaluate(() => ({
      tasma: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
      font: getComputedStyle(document.body).fontFamily.split(',')[0].replace(/"/g, ''),
      side: (() => { const s = document.querySelector('.admin-side'); return s ? Math.round(s.getBoundingClientRect().width) : 0; })(),
      kart: document.querySelectorAll('.card').length,
    }));
    if (o.tasma) tasma++;
    if (st >= 400) hata++;
    console.log(`${yol.replace('/admin', '').padEnd(18) || '/'} ${String(st).padEnd(4)} font ${o.font.padEnd(8)} menü ${String(o.side).padEnd(4)} kart ${String(o.kart).padStart(3)}${o.tasma ? ' ⚠TAŞMA' : ''}`);
  }

  // Karanlık tema
  await p.evaluate(() => localStorage.setItem('tb_theme', JSON.stringify('dark')));
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 400));
  const koyu = await p.evaluate(() => {
    const c = (s) => { const e = document.querySelector(s); return e ? getComputedStyle(e).backgroundColor : null; };
    const th = document.querySelector('.table th');
    return {
      tema: document.documentElement.getAttribute('data-theme'),
      govde: getComputedStyle(document.body).backgroundColor,
      side: c('.admin-side'), kart: c('.card'),
      formKart: c('.admin-body form:has(> table.table)'),
      input: c('.input'),
      metin: getComputedStyle(document.body).color,
      th: th ? getComputedStyle(th).color : null,
      tasma: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
    };
  });
  console.log('\n=== KARANLIK TEMA ===');
  console.log(JSON.stringify(koyu));
  await p.screenshot({ path: '/tmp/admin-skin/koyu.png', fullPage: false });
  // açık temaya dön
  await p.evaluate(() => localStorage.setItem('tb_theme', JSON.stringify('light')));

  console.log('\n500+ yanıt:', kotu.length ? kotu.join(', ') : 'yok');
  console.log('JS hatası:', hatalar.length ? [...new Set(hatalar)].slice(0, 4).join(' | ') : 'yok');
  console.log(`taşan sayfa: ${tasma} · 4xx/5xx: ${hata}`);
  await b.close();
})();
