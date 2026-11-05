const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];
const ROTALAR = [
  '/admin', '/admin/merkez', '/admin/icerikler', '/admin/icerikler/yeni',
  '/admin/kategoriler', '/admin/dosyalar', '/admin/yorumlar', '/admin/geri-bildirim',
  '/admin/ayarlar', '/admin/gorunum', '/admin/marka', '/admin/menu', '/admin/header',
  '/admin/footer', '/admin/tasarim', '/admin/manset-editor', '/admin/resmi-ilanlar',
  '/admin/raporlar', '/admin/seo', '/admin/bot', '/admin/ai-kaynak', '/admin/aboneler',
  '/admin/reklamlar', '/admin/bulten', '/admin/gazete', '/admin/siteler', '/admin/iletisim',
  '/admin/etiketler', '/admin/mahrecler', '/admin/ceviriler', '/admin/ikonlar',
];
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [], kotu = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  p.on('console', (m) => { if (m.type() === 'error') hata.push('console: ' + m.text()); });
  p.on('response', (r) => { if (r.status() >= 500) kotu.push(r.status() + ' ' + r.url()); });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let tasma = 0, kabukYok = 0, hataSayfa = 0, animYok = 0;
  for (const yol of ROTALAR) {
    const resp = await p.goto(BASE + yol, { waitUntil: 'domcontentloaded', timeout: 45000 }).catch(() => null);
    await new Promise((r) => setTimeout(r, 320));
    const st = resp ? resp.status() : 0;
    const o = await p.evaluate(() => ({
      tasma: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
      kabuk: !!document.querySelector('.admin-side') && !!document.querySelector('.admin-top'),
      anim: (() => {
        const c = document.querySelector('.admin-body > *, .pn-kpis > .pn-kpi');
        return c ? getComputedStyle(c).animationName : 'yok';
      })(),
      side: (() => { const s = document.querySelector('.admin-side'); return s ? Math.round(s.getBoundingClientRect().width) : 0; })(),
    }));
    if (o.tasma) tasma++;
    if (!o.kabuk) kabukYok++;
    if (o.anim === 'none' || o.anim === 'yok') animYok++;
    if (st >= 400) hataSayfa++;
    console.log(`${(yol.replace('/admin', '') || '/').padEnd(18)} ${String(st).padEnd(4)} menü ${String(o.side).padEnd(4)} anim ${o.anim.padEnd(9)}${o.tasma ? ' ⚠TAŞMA' : ''}`);
  }
  console.log('\nözet: taşan', tasma, '· kabuksuz', kabukYok, '· animasyonsuz', animYok, '· 4xx/5xx', hataSayfa);
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 5).join(' | ') : 'yok');
  console.log('500+:', kotu.length ? kotu.slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
