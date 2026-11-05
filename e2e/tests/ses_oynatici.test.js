'use strict';
/**
 * Ses oynatıcısı — dalga formu, tıklayarak/sürükleyerek sarma, klavye, hız.
 *
 * Gerçek waveform verisi olan bir içerik arar; bulamazsa atlar (waveform
 * transkripsiyon tamamlandığında üretilir). Oynatıcı veri yokken de çalışır
 * (sözde dalga formu) ama bu test gerçek veriyi hedefler.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

// Testler arasında bulunan içerik kimliğini paylaş (taramayı tekrarlamayalım).
let cachedID = null;

/** Dalga formu verisi olan içeriği bulur. Liste filtresi (?ses=var) yalnız
 *  transkripti olan içerikleri getirir; böylece tarama kısa sürer. */
async function contentWithWaveform(page, limit = 4) {
  if (cachedID) {
    // Önbellekteki içeriğe her testte yeniden git (sayfa yeni açılıyor).
    await goto(page, '/admin/icerikler/' + cachedID);
    return cachedID;
  }
  await goto(page, '/admin/icerikler?ses=var');
  const ids = await page.evaluate(() => {
    const out = [];
    for (const a of document.querySelectorAll('a[href*="/admin/icerikler/"]')) {
      const m = (a.getAttribute('href') || '').match(/^\/admin\/icerikler\/(\d+)$/);
      if (m && !out.includes(m[1])) out.push(m[1]);
    }
    return out;
  });
  for (const id of ids.slice(0, limit)) {
    await goto(page, '/admin/icerikler/' + id);
    const ok = await page.waitForSelector('[data-ses-player]', { timeout: 8000 })
      .then(() => true).catch(() => false);
    if (!ok) continue;
    const meta = await page.evaluate(() => {
      const root = document.querySelector('[data-ses-player]');
      return {
        wave: !!root && (root.getAttribute('data-waveform') || '').length > 50,
        synth: !!root && root.classList.contains('ses-player-synthetic'),
      };
    });
    if (meta.wave && !meta.synth) {
      cachedID = id;
      return id;
    }
  }
  return '';
}

async function waitMeta(page) {
  await page.waitForFunction(() => {
    const a = document.querySelector('audio[data-audio-player]');
    return a && isFinite(a.duration) && a.duration > 1;
  }, { timeout: 25000 });
}

test('ses oynatıcı: dalga formu çubukları ve zaman etiketleri', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 1000);
    const id = await contentWithWaveform(page);
    if (!id) return t.skip('dalga formu olan ses kaydı yok');

    await page.waitForSelector('[data-ses-player]', { timeout: 15000 });
    await waitMeta(page);

    const info = await page.evaluate(() => {
      const root = document.querySelector('[data-ses-player]');
      const bars = [...root.querySelectorAll('.ses-wave-bar')];
      const heights = bars.map((b) => parseInt(b.style.height, 10));
      const audio = root.querySelector('audio[data-audio-player]');
      return {
        barCount: bars.length,
        varied: new Set(heights).size > 3,
        synthetic: root.classList.contains('ses-player-synthetic'),
        total: (root.querySelector('[data-ses-total]') || {}).textContent,
        current: (root.querySelector('[data-ses-current]') || {}).textContent,
        role: (root.querySelector('[data-ses-wave]') || {}).getAttribute('role'),
        hasRate: !!root.querySelector('[data-ses-rate]'),
        duration: audio ? Math.round(audio.duration) : 0,
        stream: audio ? audio.getAttribute('src') : '',
      };
    });

    assert.ok(info.barCount >= 40, 'dalga formu çubukları basılmalı: ' + info.barCount);
    assert.ok(info.varied, 'çubuk yükseklikleri gerçek sesten gelmeli (tekdüze değil)');
    assert.equal(info.synthetic, false, 'gerçek waveform verisi kullanılmalı');
    assert.ok(/^\d+:\d{2}$/.test(info.total), 'toplam süre mm:ss olmalı: ' + info.total);
    assert.equal(info.current, '0:00', 'başlangıç konumu 0:00 olmalı');
    assert.equal(info.role, 'slider', 'dalga formu erişilebilir kaydırıcı olmalı');
    assert.ok(info.hasRate, 'hız seçici olmalı');
    assert.ok(info.stream.includes('/admin/ses/'), 'private akış ucu kullanılmalı');
  });
});

test('ses oynatıcı: tıklama ve sürükleme kaydı sarar', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 1000);
    const id = await contentWithWaveform(page);
    if (!id) return t.skip('dalga formu olan ses kaydı yok');
    await page.waitForSelector('[data-ses-player]', { timeout: 15000 });
    await waitMeta(page);

    // Görünüm alanına kaydır (tıklama koordinatları görünür olmalı).
    await page.evaluate(() => {
      document.querySelector('[data-ses-player]').scrollIntoView({ block: 'center' });
    });
    await new Promise((r) => setTimeout(r, 300));

    const box = await page.evaluate(() => {
      const r = document.querySelector('.ses-wave').getBoundingClientRect();
      return { x: r.x, y: r.y, w: r.width, h: r.height };
    });
    const dur = await page.evaluate(() =>
      document.querySelector('audio[data-audio-player]').duration);

    async function clickAt(ratio) {
      await page.mouse.click(box.x + box.w * ratio, box.y + box.h / 2);
      await new Promise((r) => setTimeout(r, 420));
      return page.evaluate(() => {
        const a = document.querySelector('audio[data-audio-player]');
        return {
          t: a.currentTime,
          label: (document.querySelector('[data-ses-current]') || {}).textContent,
          played: (document.querySelector('[data-ses-played]') || {}).style.width,
          aria: (document.querySelector('[data-ses-wave]') || {}).getAttribute('aria-valuenow'),
        };
      });
    }

    const q1 = await clickAt(0.25);
    assert.ok(Math.abs(q1.t - dur * 0.25) < dur * 0.04,
      '%25 tıklama yarıçapa yakın saniyeye sarmalı (' + q1.t.toFixed(1) + ' / ' + (dur * 0.25).toFixed(1) + ')');
    assert.ok(/^\d+:\d{2}$/.test(q1.label), 'konum etiketi güncellenmeli: ' + q1.label);
    assert.ok(parseFloat(q1.played) > 20, 'çalınan kısım vurgusu ilerlemeli: ' + q1.played);

    const q3 = await clickAt(0.75);
    assert.ok(Math.abs(q3.t - dur * 0.75) < dur * 0.04,
      '%75 tıklama doğru saniyeye sarmalı (' + q3.t.toFixed(1) + ')');

    // Sürükleme: %10 → %40
    await page.mouse.move(box.x + box.w * 0.10, box.y + box.h / 2);
    await page.mouse.down();
    await page.mouse.move(box.x + box.w * 0.40, box.y + box.h / 2, { steps: 8 });
    await page.mouse.up();
    await new Promise((r) => setTimeout(r, 400));
    const drag = await page.evaluate(() => {
      const a = document.querySelector('audio[data-audio-player]');
      return a.currentTime;
    });
    assert.ok(Math.abs(drag - dur * 0.40) < dur * 0.05,
      'sürükleme son konuma sarmalı (' + drag.toFixed(1) + ')');
  });
});

test('ses oynatıcı: oynat/duraklat, hız ve klavye', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 1000);
    const id = await contentWithWaveform(page);
    if (!id) return t.skip('dalga formu olan ses kaydı yok');
    await page.waitForSelector('[data-ses-player]', { timeout: 15000 });
    await waitMeta(page);
    await page.evaluate(() => {
      document.querySelector('[data-ses-player]').scrollIntoView({ block: 'center' });
    });

    // Oynat
    await page.click('[data-ses-play]');
    await new Promise((r) => setTimeout(r, 600));
    const playing = await page.evaluate(() => {
      const root = document.querySelector('[data-ses-player]');
      const a = root.querySelector('audio');
      return { cls: root.classList.contains('is-playing'), paused: a.paused, t: a.currentTime };
    });
    assert.ok(!playing.paused, 'oynatma başlamalı');
    assert.ok(playing.cls, 'oynatırken is-playing sınıfı olmalı');
    assert.ok(playing.t > 0, 'konum ilerlemeli');

    // Duraklat
    await page.click('[data-ses-play]');
    await new Promise((r) => setTimeout(r, 300));
    const paused = await page.evaluate(() => {
      const root = document.querySelector('[data-ses-player]');
      return { cls: root.classList.contains('is-playing'), paused: root.querySelector('audio').paused };
    });
    assert.ok(paused.paused, 'duraklatma çalışmalı');
    assert.equal(paused.cls, false, 'duraklatınca is-playing kalkmalı');

    // Hız
    await page.select('[data-ses-rate]', '1.5');
    await new Promise((r) => setTimeout(r, 200));
    const rate = await page.evaluate(() => {
      const a = document.querySelector('audio[data-audio-player]');
      return { r: a.playbackRate, stored: localStorage.getItem('tb_ses_hiz') };
    });
    assert.equal(rate.r, 1.5, 'oynatma hızı uygulanmalı');
    assert.equal(rate.stored, '1.5', 'hız tercihi hatırlanmalı');

    // Klavye: Home → başa
    await page.focus('.ses-wave');
    await page.keyboard.press('Home');
    await new Promise((r) => setTimeout(r, 300));
    const home = await page.evaluate(() =>
      document.querySelector('audio[data-audio-player]').currentTime);
    assert.ok(home < 1.0, 'Home kaydı başa almalı (t=' + home.toFixed(2) + ')');
  });
});
