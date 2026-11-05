const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// "Kontrast 1" çıkan öğeler gerçekten görünmez mi? Renkleri ve arka planı ölç.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const hedefler = [
    ['/admin/icerikler/yeni', [['asDurumMetin', '#asDurumMetin'], ['be-status', '.be-status'], ['vid-count', '.vid-count'], ['be-drag', '.be-drag']]],
    ['/admin/members', [['badge', '.badge']]],
    ['/admin', [['pn-delta', '.pn-delta--up'], ['pn-link', '.pn-link'], ['brand-b', '.brand b']]],
  ];

  for (const [yol, secimler] of hedefler) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 600));
    if (!r || r.status() !== 200) { console.log('atlandı', yol); continue; }
    console.log('\n###', yol);
    for (const [ad, sec] of secimler) {
      const el = await p.$(sec);
      if (!el) { console.log(' ', ad, 'yok'); continue; }
      const d = await p.evaluate((sel) => {
        const e = document.querySelector(sel);
        const s = getComputedStyle(e);
        const zincir = [];
        let c = e;
        while (c && zincir.length < 5) {
          const cs = getComputedStyle(c);
          zincir.push((c.tagName + '.' + c.className.toString().split(' ')[0]).slice(0, 26) + ' bg=' + cs.backgroundColor + ' op=' + cs.opacity);
          c = c.parentElement;
        }
        const r = e.getBoundingClientRect();
        return {
          metin: (e.textContent || '').trim().slice(0, 30),
          renk: s.color, bg: s.backgroundColor, boyut: s.fontSize, kalinlik: s.fontWeight,
          opaklik: s.opacity, gorunurluk: s.visibility, rect: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)],
          zincir,
        };
      }, sec);
      console.log(' ', ad, JSON.stringify(d));
      try {
        await el.screenshot({ path: `/tmp/probe-${ad}.png` });
      } catch (e) {}
    }
  }
  await b.close();
})();
