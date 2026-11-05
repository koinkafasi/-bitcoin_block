const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Cip rengini hangi kural veriyor? (koyu temada 142,142,142 cikti)
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 900));

  for (const t of ['light', 'dark']) {
    await p.evaluate((tema) => document.documentElement.setAttribute('data-theme', tema), t);
    await new Promise((x) => setTimeout(x, 500));
    const d = await p.evaluate(() => {
      const cip = document.querySelector('.media-cip:not(.is-on)');
      const s = getComputedStyle(cip);
      const eslesen = [];
      const yuru = (list) => {
        for (const k of list) {
          if (k.cssRules) { yuru(k.cssRules); continue; }
          if (!k.selectorText || !k.style || !k.style.color) continue;
          try {
            if (cip.matches(k.selectorText)) eslesen.push(k.selectorText + ' → ' + k.style.color);
          } catch (e) {}
        }
      };
      for (const sh of document.styleSheets) { try { yuru(sh.cssRules); } catch (e) {} }
      return {
        renk: s.color,
        textToken: getComputedStyle(document.documentElement).getPropertyValue('--text').trim(),
        mutedToken: getComputedStyle(document.documentElement).getPropertyValue('--muted-fg').trim(),
        eslesen: eslesen.slice(0, 8),
      };
    });
    console.log('=== ' + t + ' ===');
    console.log(JSON.stringify(d, null, 1));
  }
  await b.close();
})();
