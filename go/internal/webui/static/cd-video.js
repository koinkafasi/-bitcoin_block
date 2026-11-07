/* cd-video.js — YouTube oynatıcıyı TIKLAMAYLA yerleştirir.
 *
 * NEDEN: video sayfalarında iframe baştan basılsa her sayfa görüntülemesi
 * YouTube'un ~1 MB'lık oynatıcısını indirir (LCP ve üçüncü taraf maliyeti).
 * Kapak görseli + oynat düğmesi basılıyor; kullanıcı tıklayınca iframe
 * yerleştiriliyor ve otomatik oynuyor.
 *
 * CSP: bu dosya `script-src 'self'` kapsamında harici betik; satır içi kod
 * YOK (hash/nonce gerekmez). Çerçeve adresi youtube-nocookie.com; politika
 * `frame-src` ile zaten izinli (bkz. httpapi/csp.go).
 */
(function () {
  'use strict';

  function oynat(kapak) {
    var kimlik = kapak.getAttribute('data-cd-video');
    if (!kimlik) return;
    var baslik = kapak.getAttribute('data-cd-baslik') || 'Video';
    var kutu = kapak.parentElement;
    if (!kutu) return;
    /* Çift tıklama / hızlı ikinci tıklama: ikinci iframe eklenmesin. */
    if (kutu.querySelector('iframe')) return;
    var f = document.createElement('iframe');
    f.className = 'cd-vplayer__iframe';
    f.src = 'https://www.youtube-nocookie.com/embed/' + encodeURIComponent(kimlik) +
      '?autoplay=1&rel=0&modestbranding=1';
    f.title = baslik;
    f.loading = 'lazy';
    f.setAttribute('allow',
      'accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share');
    f.setAttribute('allowfullscreen', '');
    kutu.innerHTML = '';
    kutu.appendChild(f);
    /* Klavye odağı yeni oynatıcıya geçsin (erişilebilirlik). */
    f.focus();
  }

  document.addEventListener('click', function (ev) {
    var kapak = ev.target && ev.target.closest ? ev.target.closest('[data-cd-video]') : null;
    if (!kapak) return;
    ev.preventDefault();
    oynat(kapak);
  });
})();
