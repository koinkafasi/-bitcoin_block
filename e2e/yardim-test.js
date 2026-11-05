const puppeteer = require('puppeteer');

// Faz 165: yardım merkezi.
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000, deviceScaleFactor: 2 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const r = await p.goto(BASE + '/admin/yardim', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 700));
  console.log('HTTP:', r.status());

  const icerik = await p.evaluate(() => {
    const q = (s) => document.querySelectorAll(s).length;
    return {
      kisayol: q('.yardim-kisayol'),
      ipucu: q('.yardim-madde'),
      sss: q('.yardim-kutu'),
      baglanti: q('.yardim-bag__oge'),
      simgesiz: q('.yardim-blok .yardim-baslik svg') ,
      emoji: /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}]/u.test(document.querySelector('.admin-body').innerText),
      tasma: document.documentElement.scrollWidth - window.innerWidth,
    };
  });
  console.log(JSON.stringify(icerik));

  // Arama: Türkçe büyük harf testi ("GAZETE" -> "gazete")
  await p.click('#yardimAra');
  await p.type('#yardimAra', 'GAZETE', { delay: 40 });
  await new Promise((x) => setTimeout(x, 350));
  const suz = await p.evaluate(() => ({
    sayi: (document.getElementById('yardimSayi') || {}).textContent,
    gorunenMadde: [...document.querySelectorAll('.yardim-madde')].filter((e) => !e.hidden).length,
    gorunenSss: [...document.querySelectorAll('.yardim-kutu')].filter((e) => !e.hidden).length,
  }));
  console.log('arama GAZETE:', JSON.stringify(suz));

  // Sonuçsuz arama
  await p.evaluate(() => { const i = document.getElementById('yardimAra'); i.value = 'zzzzqq'; i.dispatchEvent(new Event('input')); });
  await new Promise((x) => setTimeout(x, 300));
  const bos = await p.evaluate(() => ({
    sayi: (document.getElementById('yardimSayi') || {}).textContent,
    ipucuBos: !document.getElementById('yardimBos').hidden,
  }));
  console.log('sonuçsuz:', JSON.stringify(bos));

  // Temizle
  await p.keyboard.press('Escape');
  await new Promise((x) => setTimeout(x, 250));
  const temiz = await p.evaluate(() => [...document.querySelectorAll('.yardim-madde')].filter((e) => !e.hidden).length);
  console.log('Esc sonrası görünen ipucu:', temiz);

  // Paletten açılıyor mu (Faz 162 bağlantısı)
  await p.keyboard.down('Control'); await p.keyboard.press('KeyK'); await p.keyboard.up('Control');
  await new Promise((x) => setTimeout(x, 300));
  await p.type('#cmdkInput', 'yardim', { delay: 60 });
  await new Promise((x) => setTimeout(x, 600));
  const palet = await p.evaluate(() => {
    const li = [...document.querySelectorAll('.cmdk-oge')].find((x) => (x.dataset.href || '').includes('/yardim'));
    return li ? { href: li.dataset.href, ad: li.querySelector('.cmdk-oge__ad').textContent } : null;
  });
  console.log('palette yardım:', JSON.stringify(palet));

  await p.keyboard.press('Escape');
  await p.screenshot({ path: '/tmp/yardim.png', fullPage: true });
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
