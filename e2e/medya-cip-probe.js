const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// 1) Ilk yuklemede kac dilim isteniyor? 2) cip gecisini hangi kural eziyor?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const istekler = [];
  p.on('request', (r) => {
    const u = r.url();
    if (u.indexOf('/admin/dosyalar') !== -1) istekler.push(u.replace(BASE, ''));
  });

  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'domcontentloaded' });
  // Hemen olc: dilim istekleri geldi mi
  await new Promise((x) => setTimeout(x, 200));
  const erken = await p.evaluate(() => document.querySelectorAll('#medya-panel .media-item').length);
  const erkenIstek = istekler.slice();

  await new Promise((x) => setTimeout(x, 2500));
  const gec = await p.evaluate(() => document.querySelectorAll('#medya-panel .media-item').length);

  console.log('erken kart:', erken, '· gec kart:', gec);
  console.log('istekler:', JSON.stringify(istekler, null, 1));

  // Cip gecisini ezen kural
  const gecis = await p.evaluate(() => {
    const cip = document.querySelector('.media-cip');
    const out = { hesaplanan: getComputedStyle(cip).transition, eslesen: [] };
    const yuru = (list) => {
      for (const k of list) {
        if (k.cssRules) { yuru(k.cssRules); continue; }
        if (!k.selectorText || !k.style || !k.style.transition) continue;
        try {
          if (cip.matches(k.selectorText)) {
            out.eslesen.push(k.selectorText + ' → ' + k.style.transition.slice(0, 80));
          }
        } catch (e) {}
      }
    };
    for (const s of document.styleSheets) {
      try { yuru(s.cssRules); } catch (e) {}
    }
    out.hesaplananOzellik = getComputedStyle(cip).transitionProperty;
    out.hesaplananSure = getComputedStyle(cip).transitionDuration;
    out.zemin = getComputedStyle(cip).backgroundColor;
    out.yazi = getComputedStyle(cip).color;
    return out;
  });
  console.log('\n=== CIP GECISI ===');
  console.log(JSON.stringify(gecis, null, 1));

  // Secili cip
  const secili = await p.evaluate(() => {
    const c = document.querySelector('.media-cip.is-on');
    const s = getComputedStyle(c);
    return { zemin: s.backgroundColor, yazi: s.color, kenar: s.borderColor };
  });
  console.log('\n=== SECILI CIP ===', JSON.stringify(secili));

  await b.close();
})();
