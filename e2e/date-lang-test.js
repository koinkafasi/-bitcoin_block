const puppeteer = require('puppeteer');

// Soru: Chrome'un yerel tarih alanı, sayfa lang="tr" iken Türk biçimini
// (gg.aa.yyyy) mi gösteriyor, yoksa tarayıcı arayüz dilini (en-US -> mm/dd/yyyy)
// mi? Bu, panelin gerçek kullanıcıya nasıl göründüğünü belirler.
(async () => {
  const b = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--lang=en-US'],
  });
  const p = await b.newPage();
  await p.setExtraHTTPHeaders({ 'Accept-Language': 'en-US,en;q=0.9' });
  await p.setViewport({ width: 720, height: 420, deviceScaleFactor: 2 });

  await p.setContent(`<!doctype html>
<html lang="tr"><head><meta charset="utf-8"><style>
  body{font-family:system-ui;background:#fff;color:#111;padding:20px;margin:0}
  .satir{margin:0 0 18px}
  label{display:block;font:600 13px system-ui;margin-bottom:4px}
  input{font:400 16px system-ui;padding:8px;border:1px solid #bbb;border-radius:8px}
</style></head><body>
  <div class="satir"><label>1) lang YOK (tarayıcı dili en-US)</label>
    <input type="date" value="2026-10-31"></div>
  <div class="satir"><label>2) lang="tr"</label>
    <input type="date" lang="tr" value="2026-10-31"></div>
  <div class="satir"><label>3) lang="tr" + bos deger (ipucu metni)</label>
    <input type="date" lang="tr"></div>
</body></html>`);

  const d = await p.evaluate(() => ({
    tarayiciDili: navigator.language,
    belgeDili: document.documentElement.lang,
    alanSayisi: document.querySelectorAll('input[type=date]').length,
  }));
  console.log(JSON.stringify(d));
  await p.screenshot({ path: '/tmp/date-lang.png' });
  await b.close();
})();
