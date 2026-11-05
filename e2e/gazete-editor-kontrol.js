const puppeteer = require('puppeteer');

// Gazete editörü: araç çubuğu + blok paleti simgeleri.
const BASE = process.argv[2];
const SID = process.argv[3];
const SAYI = process.argv[4] || '78';

const EMOJI = /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}\u{2190}-\u{21FF}\u{2300}-\u{23FF}\u{25A0}-\u{25FF}]/u;

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1600, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const url = BASE + '/admin/gazete/sayilar/' + SAYI + '/duzenle';
  const r = await p.goto(url, { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1400));

  const rapor = await p.evaluate((emojiSrc) => {
    const re = new RegExp(emojiSrc, 'u');
    const kalan = [];
    const y = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    while (y.nextNode()) {
      const t = y.currentNode.nodeValue || '';
      const par = y.currentNode.parentElement;
      if (re.test(t) && par && par.offsetParent !== null) {
        kalan.push({ metin: t.trim().slice(0, 30), ebeveyn: par.tagName + '.' + par.className });
      }
    }
    const araclar = [...document.querySelectorAll('.be-btn')].map((x) => {
      const s = x.querySelector('svg.ti');
      const rr = s ? s.getBoundingClientRect() : null;
      return {
        cmd: x.dataset.cmd,
        metin: x.textContent.trim(),
        svg: s ? (s.querySelector('use').getAttribute('href') || '').split('#')[1] : null,
        kutu: rr ? Math.round(rr.width) + 'x' + Math.round(rr.height) : null,
        gorunur: x.offsetParent !== null,
      };
    });
    const palet = [...document.querySelectorAll('.gz-add')].map((x) => ({
      metin: x.textContent.trim().slice(0, 22),
      svg: x.querySelector('svg.ti') ? (x.querySelector('use').getAttribute('href') || '').split('#')[1] : null,
    }));
    return {
      kalan,
      aracSayisi: araclar.length,
      svgsizArac: araclar.filter((a) => a.gorunur && !a.svg && a.metin.length > 3).map((a) => a.cmd + ':' + a.metin),
      bosArac: araclar.filter((a) => a.svg && a.kutu === '0x0' && a.gorunur).map((a) => a.cmd + ':' + a.svg),
      aracBaslik: araclar.filter((a) => a.cmd === 'undo' || a.cmd === 'table' || a.cmd === 'comments'),
      palet,
    };
  }, EMOJI.source);

  console.log('HTTP:', r.status());
  console.log('kalan emoji:', rapor.kalan.length, rapor.kalan.length ? JSON.stringify(rapor.kalan.slice(0, 8)) : '');
  console.log('araç sayısı:', rapor.aracSayisi);
  console.log('SVG olmayan (görünür):', JSON.stringify(rapor.svgsizArac));
  console.log('boyutsuz (sembol yok):', JSON.stringify(rapor.bosArac));
  console.log('örnek araçlar:', JSON.stringify(rapor.aracBaslik));
  console.log('palet:', JSON.stringify(rapor.palet));
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');

  await p.screenshot({ path: '/tmp/gz-editor.png' });
  const tb = await p.$('.gz-toolbar');
  if (tb) await tb.screenshot({ path: '/tmp/gz-toolbar.png' });
  const pl = await p.$('.gz-palette');
  if (pl) await pl.screenshot({ path: '/tmp/gz-palet.png' });
  await b.close();
})();
