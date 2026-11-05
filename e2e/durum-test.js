const puppeteer = require('puppeteer');

// Faz 163–164 görsel/ölçüm doğrulaması.
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000, deviceScaleFactor: 2 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  // 1) Boş durum: filtreyi hiç sonuç vermeyecek şekilde daralt.
  await p.goto(BASE + '/admin/icerikler?q=zzzzzzzz', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));
  const bos = await p.evaluate(() => {
    const el = document.querySelector('.bos-durum');
    if (!el) return null;
    const r = el.getBoundingClientRect();
    const c = getComputedStyle(el);
    return {
      baslik: (el.querySelector('.bos-durum__baslik') || {}).textContent,
      aciklama: (el.querySelector('.bos-durum__aciklama') || {}).textContent,
      cta: (el.querySelector('.btn') || {}).textContent,
      ikon: !!el.querySelector('svg.ti'),
      rol: el.getAttribute('role'),
      h: Math.round(r.height),
      display: c.display,
    };
  });
  console.log('boş durum:', JSON.stringify(bos));
  await p.screenshot({ path: '/tmp/bos-durum.png' });

  // 2) Yapışkan başlık: sayfayı kaydır, thead hâlâ görünür mü?
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));
  const sticky = await p.evaluate(async () => {
    const wrap = document.querySelector('.table-scroll');
    const th = document.querySelector('.table thead th');
    if (!wrap || !th) return { yok: true };
    const once = th.getBoundingClientRect().top;
    wrap.scrollTop = 400;
    await new Promise((r) => setTimeout(r, 250));
    const sonra = th.getBoundingClientRect().top;
    const c = getComputedStyle(th);
    return {
      wrapYuk: Math.round(wrap.getBoundingClientRect().height),
      kaydirilabilir: wrap.scrollHeight > wrap.clientHeight,
      once: Math.round(once), sonra: Math.round(sonra),
      sticky: c.position, zemin: c.backgroundColor, z: c.zIndex,
    };
  });
  console.log('yapışkan başlık:', JSON.stringify(sticky));

  // 3) Yükleniyor göstergesi: halka var mı, htmx sınıfı duruyor mu?
  const yuk = await p.evaluate(() => {
    const el = document.querySelector('.yukleniyor');
    if (!el) return null;
    return {
      kimlik: el.id,
      htmx: el.classList.contains('htmx-indicator'),
      halka: Math.round((el.querySelector('.yukleniyor__halka') || { getBoundingClientRect: () => ({ width: 0 }) }).getBoundingClientRect().width),
      metin: el.textContent.trim(),
      rol: el.getAttribute('role'),
    };
  });
  console.log('yükleniyor:', JSON.stringify(yuk));

  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
