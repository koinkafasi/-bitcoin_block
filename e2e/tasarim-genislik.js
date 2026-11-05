const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  for (const g of [1500, 1366, 1280, 1024, 820]) {
    await p.setViewport({ width: g, height: 1000 });
    await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' });
    await new Promise((x) => setTimeout(x, 900));

    const d = await p.evaluate(() => {
      const olc = (sec) => {
        const el = document.querySelector(sec);
        if (!el) return null;
        const r = el.getBoundingClientRect();
        return { h: Math.round(r.height), w: Math.round(r.width), x: Math.round(r.x), sag: Math.round(r.right) };
      };
      const kok = document.querySelector('.admin-body');
      const bloklar = [];
      Array.from(kok.children).forEach((el, i) => {
        const r = el.getBoundingClientRect();
        if (r.height < 2) return;
        bloklar.push({ i, id: el.id || el.className.toString().split(' ')[0], h: Math.round(r.height), y: Math.round(r.y) });
      });

      // plPanel icindeki alanlar tasiyor mu
      const panel = document.getElementById('plPanel');
      const tasan = [];
      if (panel) {
        const pr = panel.getBoundingClientRect();
        panel.querySelectorAll('.bld-field, .opt-switch, input, select, button').forEach((el) => {
          const r = el.getBoundingClientRect();
          if (r.width < 1 || r.height < 1) return;
          if (r.right > pr.right + 2 || r.bottom > pr.bottom + 2) {
            tasan.push({
              el: el.tagName + '.' + el.className.toString().split(' ')[0],
              id: el.id || '',
              sagFazla: Math.round(r.right - pr.right),
              altFazla: Math.round(r.bottom - pr.bottom),
            });
          }
        });
      }
      // ilk kart icindekiler
      const ilk = kok.children[0];
      const ilkCocuk = ilk ? Array.from(ilk.children).map((el) => {
        const r = el.getBoundingClientRect();
        return { el: el.tagName + '.' + el.className.toString().split(' ')[0], h: Math.round(r.height), w: Math.round(r.width) };
      }) : [];

      return {
        bldWrap: olc('.bld-wrap'),
        plPanel: olc('#plPanel'),
        tokens: olc('#bpTokens'),
        ilkKartH: ilk ? Math.round(ilk.getBoundingClientRect().height) : 0,
        ilkCocuk,
        bloklar,
        toplamUst: bloklar.filter((x) => x.y < (document.querySelector('.bld-wrap')?.getBoundingClientRect().y || 99999)).reduce((a, x) => a + x.h, 0),
        tasan: tasan.slice(0, 6),
        yatayTasma: document.documentElement.scrollWidth - window.innerWidth,
      };
    });

    console.log('\n### ' + g + 'px');
    console.log('  ilk kart h:', d.ilkKartH, '| çocuklar:', JSON.stringify(d.ilkCocuk));
    console.log('  bld-wrap:', JSON.stringify(d.bldWrap), '| plPanel:', JSON.stringify(d.plPanel), '| tokens:', JSON.stringify(d.tokens));
    console.log('  üst blok toplam yükseklik:', d.toplamUst);
    console.log('  bloklar:', JSON.stringify(d.bloklar));
    console.log('  plPanel taşan:', d.tasan.length ? JSON.stringify(d.tasan) : 'yok');
    console.log('  yatay taşma:', d.yatayTasma);
  }
  await b.close();
})();
