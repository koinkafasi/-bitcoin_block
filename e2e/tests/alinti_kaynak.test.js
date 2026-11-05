'use strict';
/**
 * Alıntı kaynakları — Faz 124 (istemci tarafı).
 *
 * Sunucunun ürettiği işaretleme Go testinde doğrulanıyor
 * (`quote_sources_template_test.go`). Burada asıl riskli kısım test edilir:
 * saniyeye tıklayınca ses kaydının o noktaya sarması ve doğrulanmamış
 * alıntının sarmaması.
 *
 * İşaretleme, sunucunun ürettiğiyle birebir aynı yapıdadır ve AI çekmecesine
 * enjekte edilir; böylece test ASR/AI sağlayıcısına bağımlı olmaz.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

// Sunucunun ai_editor_result şablonundaki alıntı kaynakları bloğunun aynısı.
function quoteSourcesHTML(mediaID) {
  return `
  <div class="ai-result" data-content-id="1" data-job-id="1">
    <div class="ai-quote-sources">
      <div class="ai-sec-title">Alıntı kaynakları</div>
      <button type="button" class="ai-quote-src"
              data-quote-seek="3000" data-quote-media="${mediaID}"
              title="Doğrulandı">
        <span class="ai-quote-time">0:03</span>
        <span class="ai-quote-speaker">A</span>
        <span class="ai-quote-text">“Yatırım planı 2027 yılında başlayacak”</span>
        <span class="ai-quote-flag">✓</span>
      </button>
      <button type="button" class="ai-quote-src ai-quote-src-bad"
              data-quote-seek="0" data-quote-media="${mediaID}" data-quote-bad="1"
              title="Kaynakta bulunamadı">
        <span class="ai-quote-time">—</span>
        <span class="ai-quote-speaker">kaynak yok</span>
        <span class="ai-quote-text">“Bakan yeni bir vergi duyurdu”</span>
        <span class="ai-quote-flag">✕</span>
      </button>
    </div>
  </div>`;
}

/** Ses oynatıcısı yoksa, JS sözleşmesini sınamak için sahte oynatıcı kurar. */
async function ensurePlayer(page) {
  return page.evaluate(() => {
    const existing = document.querySelector('audio[data-audio-player]');
    if (existing) {
      return { media: existing.getAttribute('data-audio-player'), real: true };
    }
    const media = 'e2e-sahte';
    const audio = document.createElement('audio');
    audio.setAttribute('data-audio-player', media);
    audio.dataset.fake = '1';
    let t = 0;
    let paused = true;
    Object.defineProperty(audio, 'currentTime', {
      get() { return t; },
      set(v) { t = v; window.__seekCalls = (window.__seekCalls || []).concat([v]); },
    });
    Object.defineProperty(audio, 'paused', { get() { return paused; } });
    audio.play = function () { paused = false; window.__playCalls = (window.__playCalls || 0) + 1; return Promise.resolve(); };
    audio.pause = function () { paused = true; t = 0; };
    document.getElementById('sesPanel').appendChild(audio);
    return { media, real: false };
  });
}

async function firstContentID(page) {
  await goto(page, '/admin/icerikler');
  return page.evaluate(() => {
    const ids = [...document.querySelectorAll('a[href*="/admin/icerikler/"]')]
      .map((el) => el.getAttribute('href') || '')
      .map((h) => (h.match(/^\/admin\/icerikler\/(\d+)$/) || [])[1])
      .filter(Boolean);
    return ids[0] || '';
  });
}

test('alıntı kaynağı: doğrulanmış saniye kaydı sarar, doğrulanmamış sarmaz', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 950);
    const id = await firstContentID(page);
    if (!id) return t.skip('içerik yok');

    await goto(page, '/admin/icerikler/' + id);
    await page.waitForSelector('#sesPanel', { timeout: 20000 });

    // Sayfada ses oynatıcısı varsa onu kullan; yoksa JS sözleşmesi için
    // sahte oynatıcı kur (gerçek entegrasyon elle doğrulandı).
    const player = await ensurePlayer(page);
    const mediaID = player.media;

    // Sonucu (aynı işaretlemeyle) AI çekmecesine enjekte et.
    await page.evaluate((html) => {
      if (!window.__aiEditor) throw new Error('__aiEditor yok');
      window.__aiEditor.open('Ses kaydından taslak');
      window.__aiEditor.setHTML(html);
    }, quoteSourcesHTML(mediaID));

    await page.waitForSelector('.ai-quote-src', { timeout: 8000 });

    const info = await page.evaluate((media) => {
      const btns = [...document.querySelectorAll('.ai-quote-src')];
      const player = document.querySelector('audio[data-audio-player="' + media + '"]');
      return {
        count: btns.length,
        badMarked: btns.every((b) => {
          const bad = b.classList.contains('ai-quote-src-bad');
          return bad ? b.getAttribute('data-quote-bad') === '1' : !b.getAttribute('data-quote-bad');
        }),
        hasPlayer: !!player,
        firstSeek: btns[0].getAttribute('data-quote-seek'),
        firstMedia: btns[0].getAttribute('data-quote-media'),
      };
    }, mediaID);

    assert.equal(info.count, 2, 'iki alıntı kaynağı olmalı');
    assert.ok(info.hasPlayer, 'eşleşen ses oynatıcısı bulunmalı');
    assert.equal(info.firstMedia, mediaID, 'alıntı medya kimliği taşımalı');
    assert.equal(info.firstSeek, '3000', 'alıntı saniyesi taşınmalı');
    assert.ok(info.badMarked, 'doğrulanmamış alıntı data-quote-bad ile işaretli olmalı');

    // 1) Doğrulanmamış alıntı: sarma YOK.
    const before = await page.evaluate((media) => {
      const p = document.querySelector('audio[data-audio-player="' + media + '"]');
      if (p) { p.pause(); p.currentTime = 0; }
      return p ? p.currentTime : -1;
    }, mediaID);

    await page.evaluate(() => document.querySelector('.ai-quote-src-bad').click());
    await new Promise((r) => setTimeout(r, 300));

    const afterBad = await page.evaluate((media) => {
      const p = document.querySelector('audio[data-audio-player="' + media + '"]');
      return { t: p ? p.currentTime : -1, paused: p ? p.paused : null };
    }, mediaID);
    assert.ok(Math.abs(afterBad.t - before) < 0.6,
      'doğrulanmamış alıntı kaydı sarmamalı (' + before + ' -> ' + afterBad.t + ')');
    assert.equal(afterBad.paused, true, 'doğrulanmamış alıntı oynatmayı başlatmamalı');

    // 2) Doğrulanmış alıntı: 3. saniyeye sar + oynat.
    await page.evaluate(() => {
      document.querySelector('.ai-quote-src:not(.ai-quote-src-bad)').click();
    });
    await new Promise((r) => setTimeout(r, 400));

    const afterOk = await page.evaluate((media) => {
      const p = document.querySelector('audio[data-audio-player="' + media + '"]');
      return { t: p ? p.currentTime : -1, paused: p ? p.paused : null };
    }, mediaID);
    assert.ok(afterOk.t >= 2.9, 'kayıt 3. saniyeye sarmalı (t=' + afterOk.t + ')');
    assert.equal(afterOk.paused, false, 'sarma sonrası oynatma başlamalı');

    assert.deepEqual(errors, [], 'sayfa hatası olmamalı: ' + errors.join(' | '));
  });
});
