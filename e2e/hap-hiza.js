const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

// Hapın yerleştiği tüm sekmeli sayfalarda hizalama bozulmuyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  for (const yol of ['/admin/icerikler', '/admin/yorumlar', '/admin/ceviriler',
                     '/admin/ayarlar', '/admin/aboneler', '/admin/kategoriler',
                     '/admin/etiketler', '/admin/resmi-ilanlar', '/admin/sinemalar']) {
    await p.goto(BASE + yol, { waitUntil: 'networkidle2', timeout: 45000 });
    await new Promise((r) => setTimeout(r, 500));
    const o = await p.evaluate(() => {
      const cikti = [];
      document.querySelectorAll('.ws-links, .tabs').forEach((ray) => {
        let pill = null;
        for (const c of ray.children) if (c.classList.contains('tb-hap')) pill = c;
        if (!pill) { cikti.push({ ray: ray.className.split(' ')[0], hap: false }); return; }
        const aktif = ray.querySelector('.on, .active, [aria-current="page"]');
        const pr = pill.getBoundingClientRect();
        const m = /inset\(([-\d.]+)px ([-\d.]+)px ([-\d.]+)px ([-\d.]+)px/.exec(pill.style.clipPath || '');
        const ar = aktif ? aktif.getBoundingClientRect() : null;
        cikti.push({
          ray: ray.classList.contains('ws-links') ? 'ws' : (ray.id || 'tabs'),
          hap: true,
          kopya: pill.children.length,
          oge: [...ray.children].filter(c => !c.classList.contains('tb-hap')).length,
          aktif: aktif ? aktif.textContent.trim().slice(0, 18) : null,
          sapmaSol: ar && m ? Math.round(pr.left + parseFloat(m[4]) - ar.left) : null,
          sapmaSag: ar && m ? Math.round(pr.right - parseFloat(m[2]) - ar.right) : null,
          opak: pill.style.opacity,
        });
      });
      return { cikti, tasma: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1 };
    });
    const satir = o.cikti.map((c) => c.hap
      ? `${c.ray} ${c.kopya}/${c.oge} "${c.aktif}" Δ${c.sapmaSol}/${c.sapmaSag}`
      : `${c.ray} HAP YOK`).join(' · ');
    console.log(`${yol.replace('/admin', '').padEnd(16)} ${satir}${o.tasma ? ' ⚠TAŞMA' : ''}`);
  }
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
