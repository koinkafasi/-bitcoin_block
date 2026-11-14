/* ── Trafik güvenliği: istemci sinyalleri ────────────────────────────────
   14 Kasım 2026.

   NE GÖNDERİLİR: navigator.webdriver, WebGL çizim birimi (yazılım mı),
   saat dilimi, birincil dil, WebRTC'nin açığa verdiği aday IP'ler ve
   sayfadaki etkileşim sayısı. Gönderim TEK istek, sayfa başına BİR kez.

   NEDEN SUNUCU HESAPLAMIYOR: bu değerler yalnız tarayıcıda var. Sunucu
   WebRTC'nin ne sızdırdığını göremez.

   GÜVENLİK: sunucu bu veriye GÜVENMEZ. Puanı yalnız YÜKSELTİR; "ben
   temizim" diyen bir bot puanını düşüremez (bkz. guvenlik_api.go).

   GİZLİLİK: WebRTC ölçümü bir STUN sunucusuna (Google) ulaşır; gizlilik
   maliyeti panelde yazılı. Yönetici WebRTC katmanını kapatırsa sunucu bu
   IP'leri hiç işlemez.

   ÇEREZ YOKSA GÖNDERİLMEZ: anonim beacon sunucuda yok sayılır; boşuna
   istek atmayalım.
   ─────────────────────────────────────────────────────────────────────── */
(function () {
  'use strict';

  var YOL = '/api/guvenlik';
  var OTURUM_ANAHTAR = 'tb_guvenlik_beacon';
  var BEKLEME_MS = 2500;   // bağlantı ve etkileşim için kısa gözlem
  var ETKILESIM_MS = 6000; // "etkileşim yok" kararı bundan sonra

  function cerezVar(ad) {
    var parcalar = document.cookie ? document.cookie.split(';') : [];
    for (var i = 0; i < parcalar.length; i++) {
      var p = parcalar[i].trim();
      if (p.indexOf(ad + '=') === 0) return true;
    }
    return false;
  }

  // Aynı sekmede tekrar göndermemek için oturum işareti. Sayfa başına
  // DEĞİL oturum başına: aksi hâlde her makale görüntülemesi bir istek
  // daha atardı (gereksiz yük).
  try {
    if (sessionStorage.getItem(OTURUM_ANAHTAR)) return;
    sessionStorage.setItem(OTURUM_ANAHTAR, '1');
  } catch (e) {
    /* gizli modda sessionStorage atabilir; yine de devam */
  }

  if (!cerezVar('tb_vid')) return;

  // ---- WebGL çizim birimi: yazılım rasterleştirici headless işaretidir.
  function gpuBilgisi() {
    try {
      var c = document.createElement('canvas');
      var gl = c.getContext('webgl') || c.getContext('experimental-webgl');
      if (!gl) return '';
      var uzanti = gl.getExtension('WEBGL_debug_renderer_info');
      if (!uzanti) return '';
      var r = gl.getParameter(uzanti.UNMASKED_RENDERER_WEBGL);
      return typeof r === 'string' ? r : '';
    } catch (e) {
      return '';
    }
  }

  // ---- WebRTC: aday IP'ler. Yalnız GENEL adresler sunucuda anlamlı;
  // yerel/mDNS adresleri sunucu tarafında elenir.
  function webrtcIPler(cb) {
    var cikti = [];
    var bitti = false;
    function bitir() {
      if (bitti) return;
      bitti = true;
      cb(cikti);
    }
    try {
      var PC = window.RTCPeerConnection || window.webkitRTCPeerConnection;
      if (!PC) return bitir();
      var pc = new PC({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });
      pc.createDataChannel('x');
      pc.onicecandidate = function (ev) {
        if (!ev || !ev.candidate || !ev.candidate.candidate) return bitir();
        var aday = ev.candidate.candidate;
        // "candidate:... <ip> <port> typ <tür>"
        var m = aday.match(/([0-9]{1,3}(\.[0-9]{1,3}){3}|[a-f0-9]{1,4}(:[a-f0-9]{0,4}){2,7})/i);
        if (m && cikti.indexOf(m[1]) < 0) cikti.push(m[1]);
      };
      pc.createOffer().then(function (t) { return pc.setLocalDescription(t); }).catch(bitir);
      // Sınırlı bekleme: toplama bitmezse elimizdekiyle dön.
      setTimeout(function () {
        try { pc.close(); } catch (e) {}
        bitir();
      }, BEKLEME_MS);
    } catch (e) {
      bitir();
    }
  }

  // ---- etkileşim sayacı
  var etkilesim = 0;
  function say() { etkilesim++; }
  ['pointerdown', 'keydown', 'wheel', 'touchstart', 'scroll'].forEach(function (olay) {
    window.addEventListener(olay, say, { passive: true, once: false });
  });

  function gonder(ipListesi) {
    var govde = {
      webdriver: navigator.webdriver === true,
      gpu: gpuBilgisi(),
      tz: '',
      dil: '',
      webrtc: ipListesi,
      etkilesim: etkilesim
    };
    try {
      govde.tz = Intl.DateTimeFormat().resolvedOptions().timeZone || '';
    } catch (e) {}
    govde.dil = navigator.language || '';

    try {
      var veri = JSON.stringify(govde);
      // 4 KB sınırı sunucuda; burada da kırp (aday listesi şişmesin).
      if (veri.length > 3500) {
        govde.webrtc = ipListesi.slice(0, 4);
        veri = JSON.stringify(govde);
      }
      if (navigator.sendBeacon) {
        var blob = new Blob([veri], { type: 'application/json' });
        if (navigator.sendBeacon(YOL, blob)) return;
      }
      fetch(YOL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: veri,
        credentials: 'same-origin',
        keepalive: true
      }).catch(function () {});
    } catch (e) {
      /* ölçüm başarısız: sunucu tarafı katmanlar yeterli */
    }
  }

  function basla() {
    webrtcIPler(function (ipListesi) {
      // Etkileşim penceresini bekle: "0 etkileşim" ancak gerçekten kimse
      // dokunmadıysa anlamlı.
      setTimeout(function () { gonder(ipListesi); }, ETKILESIM_MS);
    });
  }

  if (document.readyState === 'complete') {
    basla();
  } else {
    window.addEventListener('load', basla, { once: true });
  }
})();
