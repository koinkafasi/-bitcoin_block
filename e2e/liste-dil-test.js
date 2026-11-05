const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 192 — içerik listesi: tek satır + dil etiketleri.
//
// Ölçülenler:
//   1. Dil kolonu Tip'ten ÖNCE mi (başlık sırası)
//   2. Aynı makale kaç satır (aynı kökün dilleri tek satır olmalı)
//   3. Etiketler görünüyor mu; "çeviri yok" etiketi AI ucuna bağlı mı
//   4. Etikete tıklamak o dilin editörünü açıyor mu
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 120000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1100 });
  const jsHata = [];
  p.on('pageerror', (e) => jsHata.push(e.message));
  p.on('dialog', async (d) => { await d.accept().catch(() => {}); });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler?durum=yayinda', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 500));

  const tablo = await p.evaluate(() => {
    const basliklar = [...document.querySelectorAll('table.table thead th')].map((x) => (x.textContent || '').trim());
    const satirlar = [...document.querySelectorAll('table.table tbody tr')];
    const bilgi = satirlar.map((tr) => ({
      baslik: ((tr.querySelector('td a') || {}).textContent || '').trim().slice(0, 34),
      etiketler: [...tr.querySelectorAll('.dil-pill')].map((e) => ({
        metin: (e.textContent || '').trim(),
        tur: e.className.replace('dil-pill ', ''),
        href: e.getAttribute && e.getAttribute('href'),
        post: e.getAttribute && e.getAttribute('hx-post'),
      })),
    }));
    return { basliklar, satirSayisi: satirlar.length, ilk5: bilgi.slice(0, 5) };
  });

  console.log('=== BAŞLIK SIRASI ===');
  console.log(tablo.basliklar.join(' | '));
  const iDil = tablo.basliklar.indexOf('Dil');
  const iTip = tablo.basliklar.indexOf('Tip');
  console.log('Dil indeksi:', iDil, '· Tip indeksi:', iTip, '· Dil Tip\'ten önce:',
    iDil >= 0 && iTip >= 0 && iDil < iTip ? 'EVET' : 'HAYIR');

  console.log('\n=== SATIRLAR ===');
  console.log('satır sayısı:', tablo.satirSayisi);
  tablo.ilk5.forEach((r, i) => {
    const et = r.etiketler.map((e) => e.metin + (e.tur.includes('is-yok') ? '(yok)' : '')).join(' ');
    console.log(`  ${i + 1}. ${r.baslik}  →  ${et}`);
  });

  const cift = {};
  tablo.ilk5.forEach((r) => { cift[r.baslik] = (cift[r.baslik] || 0) + 1; });
  const tekrar = Object.entries(cift).filter(([, n]) => n > 1);
  console.log('tekrar eden başlık:', tekrar.length ? JSON.stringify(tekrar) : 'yok');

  // Etikete tıkla → o dilin editörü açılmalı.
  const hedef = await p.evaluate(() => {
    const a = document.querySelector('table.table tbody tr .dil-pill.is-var[href]');
    return a ? a.getAttribute('href') : null;
  });
  console.log('\n=== ETİKETE TIKLAMA ===');
  console.log('hedef:', hedef || '(dolu çeviri etiketi yok)');
  if (hedef) {
    await p.goto(BASE + hedef, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await new Promise((r) => setTimeout(r, 1200));
    const son = await p.evaluate(() => new URL(location.href).pathname);
    console.log('açılan:', son, '· eşleşti:', son === hedef ? 'EVET' : 'HAYIR');
  }

  console.log('\nJS hatası:', jsHata.length ? [...new Set(jsHata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
