const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

// Kategori kartı <details> içine alındı: TÜM alanlar hâlâ gönderiliyor mu?
// Kapalı <details> içeriği DOM'da kalır ve form normal gönderilir — bu
// varsayım test edilmeli, yoksa kaydetme sessizce veri kaybeder.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/kategoriler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));

  const d = await p.evaluate(() => {
    const form = document.querySelector('form.card:has(input[name="sira"])');
    const det = form && form.querySelector('details.kat-gelismis');
    const alanlar = form ? [...form.elements].map((e) => e.name).filter(Boolean) : [];
    return {
      detailsVar: !!det,
      kapali: det ? !det.open : null,
      formAlani: alanlar.length,
      alanlar: [...new Set(alanlar)],
      // FormData ile gerçekten gönderiliyor mu?
      formDataAnahtar: form ? [...new FormData(form).keys()].length : 0,
      ozetMetni: det ? det.querySelector('summary').textContent.trim().slice(0, 40) : null,
      kartSayisi: document.querySelectorAll('form.card:has(input[name="sira"])').length,
    };
  });
  console.log('details       :', d.detailsVar, '· kapalı:', d.kapali);
  console.log('özet          :', d.ozetMetni);
  console.log('form alanı    :', d.formAlani, '→', d.alanlar.join(', '));
  console.log('FormData      :', d.formDataAnahtar, 'anahtar (kapalı details içindekiler dahil)');
  console.log('kart sayısı   :', d.kartSayisi);

  // Gerçekten kaydet: bir kategorinin sırasını değiştir, kaydet, geri oku.
  const once = await p.evaluate(() => {
    const form = document.querySelector('form.card:has(input[name="sira"])');
    return { ad: form.querySelector('input[name="ad"]').value, sira: form.querySelector('input[name="sira"]').value };
  });
  const sonuc = await p.evaluate(async () => {
    const form = document.querySelector('form.card:has(input[name="sira"])');
    const sira = form.querySelector('input[name="sira"]');
    sira.value = String(Number(sira.value) + 1);
    const fd = new FormData(form);
    const r = await fetch(form.getAttribute('action'), {
      method: 'POST', body: fd, credentials: 'same-origin',
      headers: { 'HX-Request': 'true' },
    });
    return { kod: r.status, boyut: (await r.text()).length };
  });
  console.log('\nkaydet (kapalı details ile):', JSON.stringify(sonuc));

  await p.reload({ waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 400));
  const sonra = await p.evaluate(() => {
    const form = document.querySelector('form.card:has(input[name="sira"])');
    return { ad: form.querySelector('input[name="ad"]').value, sira: form.querySelector('input[name="sira"]').value };
  });
  console.log('önce :', JSON.stringify(once));
  console.log('sonra:', JSON.stringify(sonra));
  console.log('sıra değişti mi:', once.sira !== sonra.sira ? '✓ (kapalı details alanı gönderildi)' : '✗ GÖNDERİLMEDİ');

  // Geri al
  await p.evaluate(async (s) => {
    const form = document.querySelector('form.card:has(input[name="sira"])');
    form.querySelector('input[name="sira"]').value = s;
    await fetch(form.getAttribute('action'), { method: 'POST', body: new FormData(form), credentials: 'same-origin', headers: { 'HX-Request': 'true' } });
  }, once.sira);

  console.log('\nJS hatası:', hata.length ? hata.slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
