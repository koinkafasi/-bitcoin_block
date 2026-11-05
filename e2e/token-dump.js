const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 900 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 700));

  for (const tema of ['light', 'dark']) {
    await p.evaluate((t) => document.documentElement.setAttribute('data-theme', t), tema);
    await new Promise((x) => setTimeout(x, 300));
    const d = await p.evaluate(() => {
      const cs = getComputedStyle(document.documentElement);
      const t = (n) => cs.getPropertyValue(n).trim();
      return {
        accent: t('--accent'),
        accentText: t('--accent-text'),
        accentTextDark: t('--accent-text-dark'),
        accentFill: t('--accent-fill'),
        textKullanici: t('--text-kullanici'),
        textDark: t('--text-dark'),
        text: t('--text'),
        okFill: t('--ok-fill'),
        dangerFill: t('--danger-fill'),
      };
    });
    console.log(tema.toUpperCase(), JSON.stringify(d, null, 0));
  }
  await b.close();
})();
