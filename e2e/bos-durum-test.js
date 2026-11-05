const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Boş durum tutarlılığı: her ekranda tablo boş satırı tam genişliği kaplıyor
// mu ve biçim aynı mı (ikonlu kutu ya da sade metin).
const SAYFALAR = [
  ['/admin/icerikler?q=zzzzqqq', 'icerikler'],
  ['/admin/yorumlar?durum=hepsi&q=zzzzqqq', 'yorumlar'],
  ['/admin/members?q=zzzzqqq', 'uyeler'],
  ['/admin/dosyalar?q=zzzzqqq', 'medya'],
  ['/admin/etiketler', 'etiketler'],
  ['/admin/kategoriler', 'kategoriler'],
  ['/admin/aboneler', 'aboneler'],
  ['/admin/seo', 'seo'],
  ['/admin/trend', 'trend'],
  ['/admin/ayarlar', 'ayarlar'],
];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let toplam = 0, eksik = 0, elle = 0;
  for (const [yol, ad] of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 350));
    if (!r || r.status() !== 200) { console.log(ad.padEnd(12), 'atlandı', r ? r.status() : 'yok'); continue; }
    const d = await p.evaluate(() => {
      const out = [];
      document.querySelectorAll('table.table').forEach((t) => {
        const tr = t.querySelector('tr.tablo-bos');
        if (!tr) return;
        const td = tr.querySelector('td');
        const tg = t.getBoundingClientRect().width;
        const dg = td.getBoundingClientRect().width;
        const th = t.querySelectorAll('thead th').length;
        out.push({
          colspan: td.getAttribute('colspan'),
          th,
          eksikPx: Math.round(tg - dg),
          kutu: !!t.querySelector('.bos-durum'),
          metin: t.querySelector('.bos-durum__baslik')?.textContent || td.textContent.trim().slice(0, 30),
        });
      });
      return out;
    });
    for (const x of d) {
      toplam++;
      if (x.eksikPx > 2) { eksik++; console.log('  EKSİK', ad, JSON.stringify(x)); }
      if (!x.kutu) elle++;
    }
    console.log(ad.padEnd(12), r.status(), 'boş tablo:', d.length,
      d.length ? JSON.stringify(d.map((x) => ({ c: x.colspan, th: x.th, k: x.kutu }))) : '');
  }
  console.log('\nTOPLAM boş tablo:', toplam, '· genişlik eksiği:', eksik, '· ikonsuz:', elle);
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
