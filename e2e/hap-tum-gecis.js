const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

// Tüm sekme çubuklarında sayfa geçişi: hizalama + animasyon + flaş kontrolü.
const SEKMELER = [
  ['/admin/icerikler', '/admin/icerikler/yeni', 'İçerikler→Yeni'],
  ['/admin/icerikler/yeni', '/admin/icerikler', 'Yeni→İçerikler'],
  ['/admin/icerikler', '/admin/kategoriler', 'İçerikler→Kategoriler'],
  ['/admin/kategoriler', '/admin/etiketler', 'Kategoriler→Etiketler'],
  ['/admin/etiketler', '/admin/ceviriler', 'Etiketler→Çeviriler'],
  ['/admin/ceviriler', '/admin/sinemalar', 'Çeviriler→Sinemalar'],
  ['/admin/sinemalar', '/admin/resmi-ilanlar', 'Sinemalar→Resmi İlanlar'],
  ['/admin/resmi-ilanlar', '/admin/icerikler', 'Resmi İlanlar→İçerikler'],
];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let kotu = 0;
  for (const [kaynak, hedef, ad] of SEKMELER) {
    await p.goto(BASE + kaynak, { waitUntil: 'networkidle2', timeout: 45000 });
    await new Promise((r) => setTimeout(r, 450));
    const bas = await p.evaluate(() => {
      const ray = document.querySelector('.ws-links');
      const pill = ray && ray.querySelector('.tb-hap');
      const m = pill ? /inset\([-\d.]+px [-\d.]+px [-\d.]+px ([-\d.]+)px/.exec(pill.style.clipPath || '') : null;
      return { sol: m ? Math.round(parseFloat(m[1])) : null, var: !!pill };
    });
    if (bas.sol === null) { console.log(ad.padEnd(26), 'HAP YOK'); kotu++; continue; }

    p.click(`.ws-links a[href="${hedef}"]`).catch(() => {});
    const kareler = [];
    for (let i = 0; i < 60; i++) {
      await new Promise((r) => setTimeout(r, 25));
      const o = await p.evaluate(() => {
        const pill = document.querySelector('.ws-links .tb-hap');
        if (!pill) return null;
        const m = /inset\([-\d.]+px [-\d.]+px [-\d.]+px ([-\d.]+)px/.exec(pill.style.clipPath || '');
        return m ? Math.round(parseFloat(m[1])) : null;
      }).catch(() => null);
      if (o !== null) kareler.push(o);
    }
    const tekil = kareler.filter((v, i) => i === 0 || v !== kareler[i - 1]);
    const son = kareler.length ? kareler[kareler.length - 1] : null;
    const yon = (son !== null && bas.sol !== null) ? Math.sign(son - bas.sol) : 0;
    // Flaş: hareket yönünün TERSİNE en az 3 px sıçrama. Sola giderken
    // azalma normaldir; ters yönde artış flaştır.
    const geri = yon === 0 ? false
      : tekil.some((v, i) => i > 0 && (v - tekil[i - 1]) * yon < -3);

    // Hedef sekmenin konumu ile son kare örtüşüyor mu?
    const hiza = await p.evaluate(() => {
      const ray = document.querySelector('.ws-links');
      const pill = ray && ray.querySelector('.tb-hap');
      const aktif = ray && ray.querySelector('.on');
      if (!pill || !aktif) return null;
      const pr = pill.getBoundingClientRect(), ar = aktif.getBoundingClientRect();
      const m = /inset\([-\d.]+px ([-\d.]+)px [-\d.]+px ([-\d.]+)px/.exec(pill.style.clipPath || '');
      return m ? {
        sapmaSol: Math.round(pr.left + parseFloat(m[2]) - ar.left),
        sapmaSag: Math.round(pr.right - parseFloat(m[1]) - ar.right),
        aktif: aktif.textContent.trim().slice(0, 16),
      } : null;
    });

    const ok = tekil.length > 1 && !geri && hiza && hiza.sapmaSol === 0 && Math.abs(hiza.sapmaSag) <= 1;
    if (!ok) kotu++;
    console.log(`${ad.padEnd(26)} ${ok ? '✓' : '✗'} kare ${String(tekil.length).padStart(2)} · ${bas.sol}→${son} · "${hiza ? hiza.aktif : '?'}" Δ${hiza ? hiza.sapmaSol : '?'}/${hiza ? hiza.sapmaSag : '?'}${geri ? ' ⚠GERİ' : ''}`);
  }
  console.log(`\nsonuç: ${kotu === 0 ? 'TÜM GEÇİŞLER TEMİZ' : kotu + ' SORUN'}`);
  await b.close();
})();
