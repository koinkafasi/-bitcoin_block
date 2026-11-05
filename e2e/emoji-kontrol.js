const puppeteer = require('puppeteer');

// Faz 161: emoji yerine SVG. Bu betik:
//  1) iki sayfada kalan emoji metni arar,
//  2) her .ti simgesinin gerçekten boyutlandığını (0x0 = sprite yok) ölçer,
//  3) düğmede simge+etiket hizasını ölçer.
const BASE = process.argv[2];
const SID = process.argv[3];

const EMOJI = /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}\u{2190}-\u{21FF}\u{2300}-\u{23FF}\u{25A0}-\u{25FF}]/u;

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1100 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const sayfalar = [
    ['/admin/icerikler/yeni', 'icerikform'],
    ['/admin/gazete', 'gazete'],
  ];

  for (const [yol, ad] of sayfalar) {
    await p.goto(BASE + yol, { waitUntil: 'networkidle2' });
    await new Promise((r) => setTimeout(r, 900));

    const rapor = await p.evaluate((emojiSrc) => {
      const re = new RegExp(emojiSrc, 'u');
      const govde = document.querySelector('.admin-body') || document.body;

      // Görünür metin düğümlerinde kalan emoji
      const kalan = [];
      const yuruyen = document.createTreeWalker(govde, NodeFilter.SHOW_TEXT);
      while (yuruyen.nextNode()) {
        const t = yuruyen.currentNode.nodeValue || '';
        if (re.test(t)) {
          const par = yuruyen.currentNode.parentElement;
          if (par && par.offsetParent !== null) {
            kalan.push({ metin: t.trim().slice(0, 40), ebeveyn: par.tagName + '.' + par.className });
          }
        }
      }

      // Simgeler: yalnız GORUNUR olanlar. Dar ekranda araç çubuğunun
      // ikincil grubu (.be-secondary-group) gizleniyor ve "Daha fazla"
      // menüsüne taşınıyor; gizli öğe 0x0 olcuyor, bu hata degil.
      // NOT: offsetParent SVG öğelerinde TANIMSIZ (HTMLElement özelliği);
      // `offsetParent !== null` her zaman doğru dönüyordu. getClientRects
      // display:none alt ağacında boş dizi verir — doğru ölçüt bu.
      const ikonlar = [...govde.querySelectorAll('svg.ti')]
        .filter((s) => s.getClientRects().length > 0)
        .map((s) => {
        const r = s.getBoundingClientRect();
        return {
          ad: (s.querySelector('use') || {}).getAttribute
            ? (s.querySelector('use').getAttribute('href') || '').split('#')[1]
            : '?',
          kutu: Math.round(r.width) + 'x' + Math.round(r.height),
        };
      });
      const bos = ikonlar.filter((i) => i.kutu === '0x0');

      return {
        emojiSayisi: kalan.length,
        emojiOrnek: kalan.slice(0, 6),
        ikonSayisi: ikonlar.length,
        bosIkon: bos.slice(0, 8),
        farkliIkon: [...new Set(ikonlar.map((i) => i.ad))].sort(),
      };
    }, EMOJI.source);

    console.log('=== ' + ad + ' ===');
    console.log('  kalan emoji:', rapor.emojiSayisi, rapor.emojiSayisi ? JSON.stringify(rapor.emojiOrnek) : '');
    console.log('  svg.ti:', rapor.ikonSayisi, '· boyutsuz:', rapor.bosIkon.length,
      rapor.bosIkon.length ? JSON.stringify(rapor.bosIkon) : '');
    console.log('  simgeler:', rapor.farkliIkon.join(' '));
    await p.screenshot({ path: `/tmp/emoji-${ad}.png`, fullPage: ad === 'gazete' });
  }

  // Düğme hizası: ilk .btn.small içinde simge+etiket ölçüsü
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));
  const hiza = await p.evaluate(() => {
    const btn = [...document.querySelectorAll('.btn.small')].find((b) => b.querySelector('svg.ti'));
    if (!btn) return null;
    const s = btn.querySelector('svg.ti').getBoundingClientRect();
    const c = getComputedStyle(btn);
    return {
      metin: btn.textContent.trim().slice(0, 24),
      display: c.display, gap: c.gap,
      ikon: Math.round(s.width) + 'x' + Math.round(s.height),
      dugmeYuk: Math.round(btn.getBoundingClientRect().height),
    };
  });
  console.log('düğme hizası:', JSON.stringify(hiza));
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
