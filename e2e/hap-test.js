const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

const oku = (p) => p.evaluate(() => {
  const bilgi = (ray) => {
    if (!ray) return null;
    let pill = null;
    for (const c of ray.children) if (c.classList.contains('tb-hap')) pill = c;
    if (!pill) return { var: false };
    const aktif = ray.querySelector('.on, .active, [aria-current="page"]');
    const pr = pill.getBoundingClientRect();
    const ar = aktif ? aktif.getBoundingClientRect() : null;
    // Kırpma penceresinin GERÇEK konumu: hapın kutucuğu tüm şeridi kaplar,
    // görünen alan clip-path'tir. Yalnız clip'i ölçmek gerekir.
    const m = /inset\(([-\d.]+)px ([-\d.]+)px ([-\d.]+)px ([-\d.]+)px/.exec(pill.style.clipPath || '');
    const kSol = m ? Math.round(pr.left + parseFloat(m[4])) : null;
    const kSag = m ? Math.round(pr.right - parseFloat(m[2])) : null;
    return {
      var: ray.classList.contains('tb-hap-var'),
      kopya: pill.children.length,
      clip: pill.style.clipPath,
      opak: pill.style.opacity,
      pencereSol: kSol, pencereSag: kSag,
      aktifAd: aktif ? aktif.textContent.trim() : null,
      aktifSol: ar ? Math.round(ar.left) : null,
      aktifSag: ar ? Math.round(ar.right) : null,
      hapBg: getComputedStyle(pill).backgroundColor,
      kopyaRenk: pill.firstElementChild ? getComputedStyle(pill.firstElementChild).color : null,
    };
  };
  return {
    ws: bilgi(document.querySelector('.ws-links')),
    tabs: bilgi(document.querySelector('.tabs')),
    wsAdlar: [...document.querySelectorAll('.ws-links > a')].map(a => a.textContent.trim()),
  };
});

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  p.on('console', (m) => { if (m.type() === 'error') hata.push('console: ' + m.text()); });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 800));
  const a = await oku(p);

  console.log('=== /admin/icerikler ===');
  console.log('ws sekmeleri :', a.wsAdlar.join(' · '));
  console.log('ws hap       :', JSON.stringify(a.ws));
  console.log('içerik hap   :', JSON.stringify(a.tabs));

  // Hap gerçekten aktif sekmeyle örtüşüyor mu? clip-path'i ayrıştır ve
  // pill kutusuyla karşılaştır.
  const kontrol = await p.evaluate(() => {
    const ray = document.querySelector('.ws-links');
    const pill = ray.querySelector('.tb-hap');
    const aktif = ray.querySelector('.on');
    const m = /inset\(([-\d.]+)px ([-\d.]+)px ([-\d.]+)px ([-\d.]+)px/.exec(pill.style.clipPath);
    if (!m) return 'clip yok';
    const pr = pill.getBoundingClientRect();
    const clipSol = pr.left + parseFloat(m[4]);
    const clipSag = pr.right - parseFloat(m[2]);
    const ar = aktif.getBoundingClientRect();
    return {
      clipSol: Math.round(clipSol), aktifSol: Math.round(ar.left),
      clipSag: Math.round(clipSag), aktifSag: Math.round(ar.right),
      sapmaSol: Math.round(clipSol - ar.left), sapmaSag: Math.round(clipSag - ar.right),
    };
  });
  console.log('hizalama     :', JSON.stringify(kontrol));

  // Sayfalar arası geçiş: Kategoriler'e tıkla, konum devri yazıldı mı?
  await p.click('.ws-links a[href="/admin/kategoriler"]');
  await p.waitForNavigation({ waitUntil: 'networkidle2', timeout: 30000 }).catch(() => {});
  await new Promise((r) => setTimeout(r, 120));
  const gecisAni = await p.evaluate(() => {
    const ray = document.querySelector('.ws-links');
    const pill = ray && ray.querySelector('.tb-hap');
    if (!pill) return 'hap yok';
    const m = /inset\(([-\d.]+)px ([-\d.]+)px ([-\d.]+)px ([-\d.]+)px/.exec(pill.style.clipPath);
    return m ? { sol: Math.round(parseFloat(m[4])), sag: Math.round(parseFloat(m[2])), opak: pill.style.opacity } : 'clip yok';
  });
  console.log('--- Kategoriler e geçiş (animasyon ortası) ---');
  console.log('erken kare   :', JSON.stringify(gecisAni));

  await new Promise((r) => setTimeout(r, 800));
  const bb = await oku(p);
  console.log('oturmuş      :', JSON.stringify(bb.ws));
  console.log('aktif sekme  :', bb.ws.aktifAd, '· pencere', bb.ws.pencereSol, '-', bb.ws.pencereSag, '· sekme', bb.ws.aktifSol, '-', bb.ws.aktifSag);

  // Ayarlar sekmeleri (JS ile sınıf değiştiren .tabs) hapı takip ediyor mu?
  await p.goto(BASE + '/admin/ayarlar', { waitUntil: 'networkidle2', timeout: 45000 });
  await new Promise((r) => setTimeout(r, 600));
  const ayar1 = (await oku(p)).tabs;
  await p.evaluate(() => {
    const t = document.getElementById('ayar-tabs');
    const hedef = t.querySelector('a[data-sek="gorsel"]');
    t.querySelectorAll('a').forEach(a => a.classList.toggle('active', a === hedef));
    hedef.dispatchEvent(new MouseEvent('click', { bubbles: true }));
  });
  await new Promise((r) => setTimeout(r, 700));
  const ayar2 = (await oku(p)).tabs;
  console.log('--- ayarlar sekmeleri (JS sınıf değişimi) ---');
  console.log('önce         :', ayar1 && ayar1.aktifAd, 'pencere', ayar1 && ayar1.pencereSol + '-' + ayar1.pencereSag, '· sekme', ayar1 && ayar1.aktifSol + '-' + ayar1.aktifSag);
  console.log('sonra        :', ayar2 && ayar2.aktifAd, 'pencere', ayar2 && ayar2.pencereSol + '-' + ayar2.pencereSag,
    '· sekme', ayar2 && ayar2.aktifSol + '-' + ayar2.aktifSag);

  // HTMX ile içerik sekmesi değişimi (hap yeniden kurulmalı)
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));
  const htmxOnce = (await oku(p)).tabs;
  const sek = await p.$('.tabs a:not(.active)');
  if (sek) { await sek.click(); await new Promise((r) => setTimeout(r, 1500)); }
  const htmxSonra = (await oku(p)).tabs;
  console.log('--- HTMX sekme değişimi ---');
  console.log('önce         :', htmxOnce && htmxOnce.aktifAd, 'pencere', htmxOnce && htmxOnce.pencereSol + '-' + htmxOnce.pencereSag, '· sekme', htmxOnce && htmxOnce.aktifSol + '-' + htmxOnce.aktifSag);
  console.log('sonra        :', htmxSonra && htmxSonra.aktifAd, 'pencere', htmxSonra && htmxSonra.pencereSol + '-' + htmxSonra.pencereSag, '· sekme', htmxSonra && htmxSonra.aktifSol + '-' + htmxSonra.aktifSag);

  // Renk: hap koyu, kopya krem
  console.log('--- renk ---');
  console.log('hap zemini   :', a.ws && a.ws.hapBg, '· kopya metni:', a.ws && a.ws.kopyaRenk);

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));
  await p.screenshot({ path: '/tmp/hap-icerikler.png' });

  console.log('--- JS hatası ---');
  console.log(hata.length ? [...new Set(hata)].slice(0, 4).join(' | ') : 'yok');
  await b.close();
})();
