const puppeteer = require('puppeteer');
const BASE = process.argv[2] || 'http://127.0.0.1:8091';
// Sunucuyu ayakta tutar (ceviri arka plan isi calisabilsin diye).
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.goto(BASE + '/', { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => {});
  console.log('sunucu ayakta');
  await new Promise((r) => setTimeout(r, 2000));
  await b.close();
})();
