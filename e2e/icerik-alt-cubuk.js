const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Icerik formunda SAYFA SONUNDA boydan boya duran cubuk ne?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 900 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 3000));

  const d = await p.evaluate(() => {
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    const sayfaH = document.documentElement.scrollHeight;
    const out = [];

    // Tam genislige yakin, cok dugmeli ya da "harf/font" iceren ogeler
    document.querySelectorAll('body *').forEach((el) => {
      const s = getComputedStyle(el);
      if (s.display === 'none' || s.visibility === 'hidden') return;
      const r = el.getBoundingClientRect();
      if (r.width < vw * 0.85) return;
      if (r.height < 20) return;
      const dugme = el.querySelectorAll('button').length;
      const metin = (el.textContent || '').replace(/\s+/g, ' ').trim();
      const sinif = el.className.toString();
      // Harf/font ipucu
      const ipucu = /font|harf|letter|tipografi|karakter/i.test(sinif + ' ' + metin);
      if (dugme < 6 && !ipucu) return;
      out.push({
        el: el.tagName + (el.id ? '#' + el.id : '') + '.' + sinif.split(' ').slice(0, 2).join('.'),
        y: Math.round(r.y + window.scrollY),
        h: Math.round(r.height),
        w: Math.round(r.width),
        dugme,
        ipucu,
        metin: metin.slice(0, 70),
        konum: s.position,
        ust: s.bottom,
        zIndex: s.zIndex,
        sayfaSonu: Math.round(r.y + window.scrollY + r.height) >= sayfaH - 40,
      });
    });

    // Ayrica body'nin en son cocuklari
    const son = Array.from(document.body.children).slice(-6).map((el) => {
      const r = el.getBoundingClientRect();
      return {
        el: el.tagName + (el.id ? '#' + el.id : '') + '.' + el.className.toString().split(' ')[0],
        h: Math.round(r.height), y: Math.round(r.y + window.scrollY),
        metin: (el.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 60),
      };
    });

    return { vw, vh, sayfaH, bulunan: out.slice(0, 12), sonCocuklar: son };
  });

  console.log('sayfa yüksekliği:', d.sayfaH);
  console.log('--- tam genişlik + çok düğmeli öğeler ---');
  d.bulunan.forEach((x) => console.log('  ' + JSON.stringify(x)));
  console.log('--- body son çocukları ---');
  d.sonCocuklar.forEach((x) => console.log('  ' + JSON.stringify(x)));
  await p.screenshot({ path: '/tmp/icerik-alt.png', fullPage: false });
  await b.close();
})();
