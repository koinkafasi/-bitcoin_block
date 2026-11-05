const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];
const EMOJI = /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}\u{2300}-\u{23FF}\u{25A0}-\u{25FF}\u{2800}-\u{28FF}]/u;

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  for (const yol of ['/admin/servisler', '/admin/tasarim']) {
    await p.goto(BASE + yol, { waitUntil: 'networkidle2' });
    await new Promise((r) => setTimeout(r, 400));
    const d = await p.evaluate((src) => {
      const re = new RegExp(src, 'u');
      const y = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      const out = [];
      while (y.nextNode()) {
        const t = (y.currentNode.nodeValue || '').trim();
        const par = y.currentNode.parentElement;
        if (re.test(t) && par && par.getClientRects().length) out.push(t.slice(0, 40));
      }
      return out;
    }, EMOJI.source);
    console.log(yol, d.length, JSON.stringify(d));
  }
  await b.close();
})();
