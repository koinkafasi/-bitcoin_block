const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// A-7: GERCEK calisma zamani degerleri. Marka kiti satir ici CSS ile
// geliyor; kaynak dosyada --accent tanimli degil. Tahmin yerine olc.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const sayfalar = ['/admin', '/admin/icerikler', '/admin/members'];
  for (const yol of sayfalar) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 450));
    if (!r || r.status() !== 200) continue;

    const d = await p.evaluate(() => {
      const cs = getComputedStyle(document.documentElement);
      const t = (ad) => cs.getPropertyValue(ad).trim();
      const olc = (sec) => {
        const el = document.querySelector(sec);
        if (!el) return null;
        const s = getComputedStyle(el);
        return { sec, renk: s.color, zemin: s.backgroundColor, boyut: s.fontSize, kalinlik: s.fontWeight };
      };
      return {
        tema: document.documentElement.getAttribute('data-theme'),
        accent: t('--accent'),
        accent2: t('--accent-2'),
        ok: t('--ok'),
        pnSig: t('--pn-sig'),
        bg: t('--bg'),
        surface: t('--surface'),
        text: t('--text'),
        ornekler: [
          olc('.btn.ws-act'),
          olc('.badge'),
          olc('.pn-delta--up'),
          olc('.pn-link'),
          olc('.btn'),
        ].filter(Boolean),
      };
    });
    console.log('\n### ' + yol);
    console.log(JSON.stringify(d, null, 1));
  }
  await b.close();
})();
