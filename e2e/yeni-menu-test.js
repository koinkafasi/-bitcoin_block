const puppeteer = require('puppeteer');

// Faz 168: genel "Yeni" menüsü + yorum araması.
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000, deviceScaleFactor: 2 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));

  const menu = await p.evaluate(() => {
    const dd = document.querySelector('.new-dd');
    if (!dd) return null;
    const ozet = dd.querySelector('summary');
    const kutu = ozet.getBoundingClientRect();
    return {
      baslik: ozet.textContent.trim(),
      acikBaslangic: dd.open,
      btnKutu: [Math.round(kutu.width), Math.round(kutu.height)],
      ogeSayisi: dd.querySelectorAll('.new-dd__menu a').length,
      ogeler: [...dd.querySelectorAll('.new-dd__menu b')].map((x) => x.textContent),
      ikonlar: [...dd.querySelectorAll('.new-dd__menu svg.ti use')].map((u) => (u.getAttribute('href') || '').split('#')[1]),
    };
  });
  console.log('menü:', JSON.stringify(menu));

  // Aç
  await p.click('.new-dd > summary');
  await new Promise((r) => setTimeout(r, 350));
  const acik = await p.evaluate(() => {
    const dd = document.querySelector('.new-dd');
    const m = dd.querySelector('.new-dd__menu').getBoundingClientRect();
    return { open: dd.open, kutu: [Math.round(m.width), Math.round(m.height)], x: Math.round(m.x), sag: Math.round(m.right) };
  });
  console.log('açık:', JSON.stringify(acik));
  await p.screenshot({ path: '/tmp/yeni-menu.png' });

  // Dışarı tıklama kapatıyor mu
  await p.mouse.click(300, 700);
  await new Promise((r) => setTimeout(r, 300));
  console.log('dış tıklama sonrası açık:', await p.evaluate(() => document.querySelector('.new-dd').open));

  // Esc kapatıyor mu
  await p.click('.new-dd > summary');
  await new Promise((r) => setTimeout(r, 250));
  await p.keyboard.press('Escape');
  await new Promise((r) => setTimeout(r, 250));
  console.log('Esc sonrası açık:', await p.evaluate(() => document.querySelector('.new-dd').open));

  // Yorum araması
  await p.goto(BASE + '/admin/yorumlar', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));
  const yorum = await p.evaluate(() => {
    const f = document.querySelector('#yorum-panel form.filtre-grid');
    return {
      varMi: !!f,
      alan: (f && f.querySelector('input[name=q]')) ? f.querySelector('input[name=q]').getAttribute('aria-label') : null,
      durumGizli: (f && f.querySelector('input[name=durum]')) ? f.querySelector('input[name=durum]').value : null,
      bosDurum: (() => { const e = document.querySelector('.bos-durum__baslik'); return e ? e.textContent : null; })(),
    };
  });
  console.log('yorum filtresi:', JSON.stringify(yorum));

  // Arama çalışıyor mu (HTMX)
  await p.type('#yorum-panel input[name=q]', 'zzzqqq', { delay: 40 });
  await p.click('#yorum-panel form.filtre-grid button[type=submit]');
  await new Promise((r) => setTimeout(r, 900));
  const sonuc = await p.evaluate(() => ({
    url: location.search,
    bos: (document.querySelector('.bos-durum__baslik') || {}).textContent,
    aciklama: (document.querySelector('.bos-durum__aciklama') || {}).textContent,
    temizle: !!document.querySelector('#yorum-panel form.filtre-grid a.btn.ghost'),
  }));
  console.log('arama sonrası:', JSON.stringify(sonuc));


  // Üye araması (N-11)
  await p.goto(BASE + '/admin/members', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));
  const uyeFiltre = await p.evaluate(() => {
    const f = document.querySelector('#members-panel form.filtre-grid');
    return {
      varMi: !!f,
      alan: (f && f.querySelector('input[name=q]')) ? f.querySelector('input[name=q]').getAttribute('aria-label') : null,
    };
  });
  console.log('üye filtresi:', JSON.stringify(uyeFiltre));

  await p.type('#members-panel input[name=q]', 'zzzqqq', { delay: 40 });
  await p.click('#members-panel form.filtre-grid button[type=submit]');
  await new Promise((r) => setTimeout(r, 900));
  const uyeSonuc = await p.evaluate(() => ({
    url: location.search,
    // Sayfada birden çok boş durum var (ödeme tablosu da boş); üye
    // tablosununki "Sonuç yok" olmalı.
    bosBasliklar: [...document.querySelectorAll('.bos-durum__baslik')].map((e) => e.textContent),
    sonucYok: [...document.querySelectorAll('.bos-durum__baslik')].some((e) => e.textContent === 'Sonuç yok'),
    temizle: !!document.querySelector('#members-panel form.filtre-grid a.btn.ghost'),
  }));
  console.log('üye araması sonrası:', JSON.stringify(uyeSonuc));

  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
