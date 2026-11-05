const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  for (const yol of ['/admin', '/admin/icerikler', '/admin/ayarlar', '/admin/tasarim']) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 600));
    if (!r || r.status() !== 200) continue;

    for (const tema of ['light', 'dark']) {
      await p.evaluate((t) => document.documentElement.setAttribute('data-theme', t), tema);
      await new Promise((x) => setTimeout(x, 300));
      const d = await p.evaluate(() => {
        const cs = getComputedStyle(document.documentElement);
        const body = getComputedStyle(document.body);
        // Ornek bir metin: h1 ya da ilk baslik
        const h = document.querySelector('h1, h2, .admin-crumb, main p') || document.body;
        return {
          tokenText: cs.getPropertyValue('--text').trim(),
          tokenBg: cs.getPropertyValue('--bg').trim(),
          bodyRenk: body.color,
          ornek: (h.textContent || '').trim().slice(0, 24),
          ornekRenk: getComputedStyle(h).color,
        };
      });
      console.log(yol.padEnd(18), tema.padEnd(6), JSON.stringify(d));
    }
  }
  await b.close();
})();
