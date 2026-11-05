const puppeteer = require('puppeteer');
const BASE = process.argv[2] || 'http://127.0.0.1:8091';

// /en sayfalarinda kalan Turkce metinleri olcer.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });

  const TR = /[çğıöşüÇĞİÖŞÜ]|\b(ve|için|haber|haberler|son|dakika|kategori|arama|yorum|giriş|abonelik|hakkında|iletişim|gündem|spor|ekonomi|siyaset|dünya|yaşam|yazar|galeri|video|foto|borsa|piyasa|hava|namaz|astroloji|arşiv|bülten|daha|fazla|tümü|tüm)\b/i;

  const sayfalar = ['/en', '/en/arama?q=istanbul', '/en/arsiv', '/en/akis'];
  for (const yol of sayfalar) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2', timeout: 60000 }).catch(() => null);
    if (!r || r.status() !== 200) { console.log(yol, '->', r && r.status()); continue; }
    await new Promise((x) => setTimeout(x, 600));
    const d = await p.evaluate((desen) => {
      const re = new RegExp(desen, 'i');
      const bulunan = new Map();
      const yuru = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      let n;
      while ((n = yuru.nextNode())) {
        const t = (n.textContent || '').trim();
        if (t.length < 3 || t.length > 70) continue;
        if (!re.test(t)) continue;
        // script/style atla
        const el = n.parentElement;
        if (!el || ['SCRIPT', 'STYLE', 'NOSCRIPT'].includes(el.tagName)) continue;
        bulunan.set(t, (bulunan.get(t) || 0) + 1);
      }
      return [...bulunan.entries()].sort((a, b) => b[1] - a[1]).slice(0, 12);
    }, TR.source);
    console.log('\n' + yol + ' · kalan Turkce metin: ' + d.length);
    d.forEach(([t, c]) => console.log('   ' + c + '× ' + t.slice(0, 60)));
  }
  await b.close();
})();
