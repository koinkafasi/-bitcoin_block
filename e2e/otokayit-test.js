const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];
const IDLER = process.argv.slice(4);

// Faz 191 ÖLÇÜM: sayfaya hiç dokunulmadan "değişiklik var" sayılıyor mu?
//
// Neden 3 içerik: hatanın tek bir makaleye mi (collab odası bozuk) yoksa
// tüm editöre mi özgü olduğunu ayırmak için. Her biri taze sayfada.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  const diyalog = [];
  p.on('dialog', async (d) => { diyalog.push(d.type()); await d.accept().catch(() => {}); });

  for (const id of IDLER) {
    await p.goto(BASE + '/admin/icerikler/' + id, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await new Promise((r) => setTimeout(r, 2500));
    const durum = await p.evaluate(() => {
      const tas = document.querySelector('[name="detay"]');
      let pm = 0, pmDugum = 0;
      document.querySelectorAll('.ProseMirror').forEach((x) => { pm++; pmDugum += x.childNodes.length; });
      let jsonDugum = -1, jsonBayt = (tas.value || '').length;
      try {
        const o = JSON.parse(tas.value);
        jsonDugum = (Array.isArray(o) ? o : (o.content || [])).length;
      } catch (e) {}
      return {
        durum: ((document.getElementById('asDurumMetin') || {}).textContent || '').trim(),
        collab: !!window.__collabActive,
        synced: !!window.__collabSynced,
        pm: pm,
        pmDugum: pmDugum,
        jsonDugum: jsonDugum,
        jsonBayt: jsonBayt,
      };
    }).catch(() => ({ durum: '?', collab: null, synced: null, pm: -1, pmDugum: -1, jsonDugum: -1, jsonBayt: -1 }));

    console.log('içerik ' + id + ' · ' + JSON.stringify(durum));
  }

  // Gezinme: diyalog çıkıyor mu?
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'domcontentloaded', timeout: 60000 })
    .catch(() => console.log('  gezinme hatası'));
  console.log('diyaloglar:', diyalog.length ? diyalog.join(', ') : 'yok');
  await b.close();
})();
