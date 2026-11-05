'use strict';
/**
 * Sesli haber: duraklatma, kaldığı yerden devam ve baştan başlatma.
 * Gerçek ses yerine speechSynthesis sahte (mock) ile test edilir.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('sesli haber duraklat/devam/baştan', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);
    await page.evaluateOnNewDocument(() => {
      class FakeUtterance {
        constructor(text) { this.text = text; this.lang = ''; this.rate = 1; this.volume = 1; }
      }
      const fake = {
        speaking: false,
        _u: null,
        getVoices: () => [{ lang: 'tr-TR', name: 'Test' }],
        cancel() {
          this.speaking = false;
          if (this._u && this._u.onerror) this._u.onerror({ error: 'canceled' });
        },
        speak(u) {
          this._u = u;
          this.speaking = true;
          window.__ttsSpoken = window.__ttsSpoken || [];
          window.__ttsSpoken.push({ len: u.text.length, text: u.text });
          setTimeout(() => { if (u.onboundary) u.onboundary({ charIndex: 60 }); }, 20);
        },
      };
      Object.defineProperty(window, 'speechSynthesis', { configurable: true, value: fake });
      Object.defineProperty(window, 'SpeechSynthesisUtterance', { configurable: true, value: FakeUtterance });
    });

    await goto(page, '/');
    await page.evaluate(() => { try { localStorage.clear(); } catch (e) {} });
    const href = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(href, 'makale bağlantısı yok');
    await goto(page, href);

    const key = await page.evaluate(() =>
      'tb_tts_pos_' + document.querySelector('main[data-article-id]').getAttribute('data-article-id'));
    assert.equal(await page.evaluate((k) => localStorage.getItem(k), key), null, 'başlangıçta kayıt olmamalı');

    await page.click('#ttsPlay');
    await page.waitForFunction(() => document.getElementById('artAudio').classList.contains('playing'));
    await new Promise((r) => setTimeout(r, 120));

    const first = await page.evaluate(() => window.__ttsSpoken[0]);
    const firstSpoken = first.len;
    // Metin karşılaştırması boşluktan bağımsızdır: paragraf sonları innerText'te
    // \n iken seslendirme metninde tek boşluğa indirgenir. Uzunluk yerine
    // normalize edilmiş metin eşitliği doğrulanır.
    const bodyText = await page.$eval('.article .body', (el) =>
      el.innerText.replace(/\s+/g, ' ').trim());
    const spokenText = (first.text || '').replace(/\s+/g, ' ').trim();
    assert.equal(spokenText, bodyText, 'tarayıcı sesi tüm metni okumaya başlamalı');

    const wordHl = await page.evaluate(() => {
      if (!window.CSS || !CSS.highlights) return null;
      const h = CSS.highlights.get('tts-word');
      if (!h) return null;
      const r = [...h][0];
      return r ? r.toString() : null;
    });
    assert.ok(wordHl && wordHl.trim().length > 0, 'okunan kelime vurgulanmalı: ' + wordHl);

    // Duraklat
    await page.click('#ttsPlay');
    await page.waitForFunction(() => !document.getElementById('artAudio').classList.contains('playing'));
    const saved = await page.evaluate((k) => parseFloat(localStorage.getItem(k) || '0'), key);
    assert.ok(saved > 3, 'duraklatınca konum kaydedilmeli: ' + saved);
    const hlPaused = await page.evaluate(() => {
      const h = CSS.highlights ? CSS.highlights.get('tts-word') : null;
      return h && [...h][0] ? [...h][0].toString() : null;
    });
    assert.ok(hlPaused && hlPaused.trim().length > 0, 'duraklatınca vurgu korunmalı');

    // Sayfa yenilenince "kaldığınız yerden" notu ve konum korunmalı
    await page.reload({ waitUntil: 'networkidle2' });
    const note = await page.$eval('#ttsNote', (el) => el.textContent);
    assert.ok(note.includes('Kaldığınız'), 'yenilemede devam notu görünmeli: ' + note);
    const savedAfter = await page.evaluate((k) => parseFloat(localStorage.getItem(k) || '0'), key);
    assert.ok(savedAfter > 3, 'konum yenilemeden sonra korunmalı');

    // Devam: kalan metin, tam metinden kısa olmalı
    await page.evaluate(() => { window.__ttsSpoken = []; });
    await page.click('#ttsPlay');
    await page.waitForFunction(() => (window.__ttsSpoken || []).length > 0);
    const resumed = await page.evaluate(() => window.__ttsSpoken[0].len);
    assert.ok(resumed < firstSpoken, 'devam ederken baştan okumamalı');

    // Baştan başlat (duraklatıp konumu sıfırla)
    await page.click('#ttsPlay');
    await page.waitForFunction(() => !document.getElementById('artAudio').classList.contains('playing'));
    await page.click('#ttsRestart');
    const cleared = await page.evaluate((k) => localStorage.getItem(k), key);
    assert.equal(cleared, null, 'baştan başlatınca konum silinmeli');
    const noteAfter = await page.$eval('#ttsNote', (el) => el.textContent);
    assert.ok(noteAfter.includes('Baştan'), 'baştan notu görünmeli');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('mobil: sabit mini oynatıcı ile duraklat/devam', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 390, 844);
    await page.evaluateOnNewDocument(() => {
      class FakeUtterance { constructor(text) { this.text = text; } }
      const fake = {
        speaking: false, _u: null,
        getVoices: () => [{ lang: 'tr-TR' }],
        cancel() { this.speaking = false; if (this._u && this._u.onerror) this._u.onerror({ error: 'canceled' }); },
        speak(u) { this._u = u; this.speaking = true; setTimeout(() => { if (u.onboundary) u.onboundary({ charIndex: 60 }); }, 20); },
      };
      Object.defineProperty(window, 'speechSynthesis', { configurable: true, value: fake });
      Object.defineProperty(window, 'SpeechSynthesisUtterance', { configurable: true, value: FakeUtterance });
    });

    await goto(page, '/');
    const href = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(href, 'makale bağlantısı yok');
    await goto(page, href);

    await page.click('#ttsPlay');
    await page.waitForFunction(() => document.getElementById('artAudio').classList.contains('playing'));

    // Yazıda aşağı in: ana oynatıcı ekrandan çıkınca mini görünmeli.
    // Gerçek kullanıcı kaydırması (wheel) kullanılır: programatik
    // window.scrollTo, TTS'in "okunan yeri takip et" kaydırması tarafından
    // geri alınır ve mini oynatıcı hemen kapanır.
    await page.mouse.move(200, 400);
    await page.mouse.wheel({ deltaY: 1200 });
    await page.waitForFunction(() => document.getElementById('ttsMini').classList.contains('show'), { timeout: 4000 });
    await new Promise((r) => setTimeout(r, 350)); // açılış animasyonu bitsin
    assert.equal(await page.$eval('#ttsMini', (el) => el.getAttribute('aria-hidden')), 'false');

    // Mini ile duraklat
    await page.click('#ttsMiniPlay');
    await page.waitForFunction(() => !document.getElementById('artAudio').classList.contains('playing'));
    assert.equal(await page.$eval('#ttsMiniPlay', (el) => el.getAttribute('aria-label')), 'Oynat',
      'mini düğme duraklatınca Oynat olmalı');
    assert.ok(await page.$eval('#ttsMini', (el) => el.classList.contains('show')),
      'duraklatınca mini görünür kalmalı');

    // Mini ile devam
    await page.click('#ttsMiniPlay');
    await page.waitForFunction(() => document.getElementById('artAudio').classList.contains('playing'));

    // Kapat
    await page.click('#ttsMiniClose');
    await page.waitForFunction(() => !document.getElementById('ttsMini').classList.contains('show'));

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
