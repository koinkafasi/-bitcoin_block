// Pano 2.0 arayüz doğrulaması: yerleşim ölçüleri + etkileşimler.
//
// Görsel yargı yerine ÖLÇÜM: ray genişliği, KPI ızgarası, grafik boyutu,
// sekme thumb konumu, sürükle-bırak sonrası toast ve DB'ye yazılan saat.
const puppeteer = require('puppeteer');

const PORT = process.argv[2] || '8094';
const SID = process.argv[3] || '';
const BASE = `http://127.0.0.1:${PORT}`;

(async () => {
  const browser = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage'] });
  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 1000 });
  const errors = [];
  const kotuIstek = [];
  page.on('pageerror', (e) => errors.push(String(e.message)));
  page.on('console', (m) => { if (m.type() === 'error') errors.push('console: ' + m.text()); });
  page.on('response', (r) => { if (r.status() >= 400) kotuIstek.push(r.status() + ' ' + r.url()); });

  if (SID) await page.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await page.goto(BASE + '/admin/pano', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 900));

  const olcum = await page.evaluate(() => {
    const box = (sel) => {
      const el = document.querySelector(sel);
      if (!el) return null;
      const r = el.getBoundingClientRect();
      const s = getComputedStyle(el);
      return { w: Math.round(r.width), h: Math.round(r.height), x: Math.round(r.x), y: Math.round(r.y), radius: s.borderRadius, bg: s.backgroundColor };
    };
    const seg = document.getElementById('pnSeg');
    const thumb = document.getElementById('pnSegThumb');
    const aktif = seg && seg.querySelector('button[aria-pressed="true"]');
    return {
      baslik: (document.querySelector('.pn-head h1') || {}).textContent,
      app: box('.pn-app'),
      side: box('.pn-side'),
      kpiSayisi: document.querySelectorAll('.pn-kpi').length,
      kpi: box('.pn-kpi'),
      kpiGrid: getComputedStyle(document.querySelector('.pn-kpis')).gridTemplateColumns,
      deger: (document.querySelector('.pn-kpi__value') || {}).textContent,
      delta: (document.querySelector('.pn-delta') || {}).textContent,
      spark: box('.pn-spark'),
      alan: box('#pnChartBox .pn-seriwrap.on svg'),
      alanPathUzunluk: (document.querySelector('#pnChartBox .pn-seriwrap.on .pn-alan') || {}).getAttribute
        ? (document.querySelector('#pnChartBox .pn-seriwrap.on .pn-alan').getAttribute('d') || '').length : 0,
      noktaSayisi: document.querySelectorAll('#pnChartBox .pn-seriwrap.on .pn-nokta').length,
      halka: box('.pn-halka'),
      dilimSayisi: document.querySelectorAll('.pn-dilim').length,
      lejant: [...document.querySelectorAll('.pn-lejant li')].map((li) => li.textContent.replace(/\s+/g, ' ').trim()).slice(0, 6),
      sutun: document.querySelectorAll('.pn-cubuk').length,
      yatay: document.querySelectorAll('.pn-yatay__satir').length,
      serit: box('.pn-plan__ray'),
      ziyaretSayisi: document.querySelectorAll('.pn-visit').length,
      ziyaretler: [...document.querySelectorAll('.pn-visit')].slice(0, 6).map((v) => ({
        left: v.style.left, w: Math.round(v.getBoundingClientRect().width), text: v.textContent.replace(/\s+/g, ' ').trim().slice(0, 40),
      })),
      simdi: box('.pn-plan__imdi'),
      segThumb: thumb ? { w: Math.round(thumb.getBoundingClientRect().width), x: Math.round(thumb.getBoundingClientRect().x) } : null,
      segAktif: aktif ? aktif.textContent : null,
      segAktifX: aktif ? Math.round(aktif.getBoundingClientRect().x) : null,
      gorunurSeri: [...document.querySelectorAll('.pn-seriwrap')].map((w) => w.classList.contains('on')),
      railDar: document.getElementById('pnApp').classList.contains('is-rail'),
    };
  });

  console.log('--- yerleşim ---');
  console.log('başlık      :', olcum.baslik);
  console.log('ray         :', olcum.side && olcum.side.w + 'px (dar=' + olcum.railDar + ')');
  console.log('kpi         :', olcum.kpiSayisi, 'adet ·', olcum.kpi && olcum.kpi.w + 'x' + olcum.kpi.h, '· radius', olcum.kpi && olcum.kpi.radius);
  console.log('kpi ızgara  :', olcum.kpiGrid);
  console.log('kpi değer   :', olcum.deger, '· delta:', olcum.delta);
  console.log('spark       :', olcum.spark && olcum.spark.w + 'x' + olcum.spark.h);
  console.log('alan grafik :', olcum.alan && olcum.alan.w + 'x' + olcum.alan.h, '· nokta', olcum.noktaSayisi, '· path', olcum.alanPathUzunluk, 'krk');
  console.log('halka       :', olcum.halka && olcum.halka.w + 'px ·', olcum.dilimSayisi, 'dilim');
  console.log('lejant      :', olcum.lejant.join(' | '));
  console.log('sütun/yatay :', olcum.sutun, '/', olcum.yatay);
  console.log('şerit       :', olcum.serit && olcum.serit.w + 'x' + olcum.serit.h, '· kart', olcum.ziyaretSayisi);
  olcum.ziyaretler.forEach((z) => console.log('   kart', z.left, z.w + 'px', '|', z.text));
  console.log('şimdi çizgi :', olcum.simdi && olcum.simdi.x);
  console.log('sekme thumb :', olcum.segThumb, '· aktif', olcum.segAktif, '@', olcum.segAktifX);
  console.log('görünür seri:', olcum.gorunurSeri.join(','));

  // --- etkileşim 1: sekmeye tıkla ---
  await page.click('#pnSeg button[data-pn-seri="30"]');
  await new Promise((r) => setTimeout(r, 500));
  const segSonra = await page.evaluate(() => ({
    gorunur: [...document.querySelectorAll('.pn-seriwrap')].map((w) => w.classList.contains('on')),
    nokta: document.querySelectorAll('#pnChartBox .pn-seriwrap.on .pn-nokta').length,
    aktif: document.querySelector('#pnSeg button[aria-pressed="true"]').textContent,
  }));
  console.log('--- etkileşim: sekme ---');
  console.log('30 gün seçildi:', segSonra.aktif, '· görünür seri', segSonra.gorunur.join(','), '· nokta', segSonra.nokta);

  // --- etkileşim 2: grafik balonu ---
  const balon = await page.evaluate(() => {
    const nokta = document.querySelector('#pnChartBox .pn-seriwrap.on .pn-nokta');
    if (!nokta) return 'nokta yok';
    const r = nokta.getBoundingClientRect();
    nokta.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, clientX: r.x + 1, clientY: r.y + 1 }));
    const t = document.getElementById('pnBalon');
    return t.hidden ? 'gizli' : t.textContent;
  });
  console.log('balon       :', balon);

  // --- etkileşim 3: sürükle-bırak ---
  const drag = await page.evaluate(async () => {
    const ray = document.getElementById('pnPlanRay');
    const el = ray.querySelector('.pn-visit');
    if (!el) return 'kart yok';
    const oncekiLeft = el.style.left;
    const r = ray.getBoundingClientRect();
    const dt = new DataTransfer();
    el.dispatchEvent(new DragEvent('dragstart', { bubbles: true, dataTransfer: dt }));
    const hedefX = r.left + r.width * 0.62;
    ray.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true, dataTransfer: dt, clientX: hedefX }));
    ray.dispatchEvent(new DragEvent('drop', { bubbles: true, cancelable: true, dataTransfer: dt, clientX: hedefX }));
    await new Promise((res) => setTimeout(res, 900));
    return {
      oncekiLeft,
      sonrakiLeft: el.style.left,
      saat: (el.querySelector('small') || {}).textContent,
      toast: (document.querySelector('.pn-toast') || {}).textContent || null,
      kartVar: !!el,
    };
  });
  console.log('--- etkileşim: sürükle-bırak ---');
  console.log('kart:', JSON.stringify(drag));

  // --- etkileşim 4: ray katlama ---
  const railOnce = await page.evaluate(() => document.querySelector('.pn-side').getBoundingClientRect().width);
  await page.click('#pnRail');
  await new Promise((r) => setTimeout(r, 420));
  const railSonra = await page.evaluate(() => ({
    w: document.querySelector('.pn-side').getBoundingClientRect().width,
    dar: document.getElementById('pnApp').classList.contains('is-rail'),
    kayit: localStorage.getItem('pn:rail'),
  }));
  console.log('--- etkileşim: ray ---');
  console.log('ray', Math.round(railOnce), '→', Math.round(railSonra.w), '· dar', railSonra.dar, '· kayıt', railSonra.kayit);

  // --- sayfa yenilenince tercih korunuyor mu ---
  await page.reload({ waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));
  const kalici = await page.evaluate(() => document.getElementById('pnApp').classList.contains('is-rail'));
  console.log('yenileme sonrası dar ray:', kalici);

  // --- duyarlı: 900px'de yan menü gizlenir ---
  await page.setViewport({ width: 880, height: 900 });
  await new Promise((r) => setTimeout(r, 350));
  const mobil = await page.evaluate(() => {
    const side = document.querySelector('.pn-side');
    const kpi = getComputedStyle(document.querySelector('.pn-kpis')).gridTemplateColumns;
    return { sideGorunur: side ? getComputedStyle(side).display : 'yok', kpiKolon: kpi.split(' ').length };
  });
  console.log('--- duyarlı (880px) ---');
  console.log('yan menü:', mobil.sideGorunur, '· kpi kolon:', mobil.kpiKolon);

  await page.setViewport({ width: 1440, height: 1000 });
  await page.screenshot({ path: '/tmp/pano-verify/shot.png', fullPage: false });
  console.log('--- hatalı istekler ---');
  console.log(kotuIstek.length ? kotuIstek.join('\n') : 'yok');
  console.log('--- JS hataları ---');
  console.log(errors.length ? errors.slice(0, 5).join('\n') : 'yok');
  await browser.close();
})();
