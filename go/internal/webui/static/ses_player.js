/* Ses oynatıcısı — SoundCloud benzeri dalga formu.
 *
 * Özellikler:
 *  - Dalga formu çubukları (transkriptteki waveform verisinden; yoksa
 *    sözde dalga formu çizilir, oynatıcı yine çalışır).
 *  - Dalga formuna tıklayınca/sürükleyince o saniyeye atlar.
 *  - Klavye: sol/sağ 5 sn (Shift 30 sn, Alt 1 sn), boşluk oynat/duraklat,
 *    Home/End baş/son.
 *  - Hız seçici (0,75× – 2×), süre göstergesi, çalınan kısmın vurgusu.
 *  - Transkript segmentine tıklayınca ilgili saniyeye atlar (eski davranış).
 *  - Karaoke vurgu: kelime zamanı varsa oynarken geçerli kelime işaretlenir
 *    ve transkript kutusunun içinde görünür tutulur (Faz 127).
 *  - Duraklatılan yerden devam ederken 2 sn geri sarar; kullanıcı elle
 *    sardıysa sarmaz.
 *  - Media Session: medya tuşları ve kilit ekranı kontrolleri.
 *
 * Hiçbir şeyi otomatik kaydetmez; yalnız oynatma durumunu yönetir.
 */
(function () {
  "use strict";

  var SVG_PLAY = "M8 5v14l11-7z";
  var SVG_PAUSE = "M6 5h4v14H6zM14 5h4v14h-4z";
  var RATE_KEY = "tb_ses_hiz";

  function fmt(sec) {
    if (!isFinite(sec) || sec < 0) sec = 0;
    var s = Math.floor(sec % 60);
    var m = Math.floor(sec / 60) % 60;
    var h = Math.floor(sec / 3600);
    if (h > 0) return h + ":" + pad(m) + ":" + pad(s);
    return m + ":" + pad(s);
  }

  function pad(n) { return n < 10 ? "0" + n : String(n); }

  /* Sözde dalga formu: gerçek veri yoksa görsel olarak makul bir desen. */
  function syntheticWave(bars) {
    var out = [];
    for (var i = 0; i < bars; i++) {
      var t = i / bars;
      var base = 0.32
        + 0.28 * Math.abs(Math.sin(t * 21.3))
        + 0.18 * Math.abs(Math.sin(t * 7.7 + 1.1))
        + 0.10 * Math.abs(Math.sin(t * 51.9 + 2.3));
      // Hafif rastgelelik (deterministik: konuma bağlı)
      var jitter = 0.06 * Math.abs(Math.sin(t * 133.7 + 4.1));
      out.push(Math.min(1, base + jitter));
    }
    return out;
  }

  function buildBars(container, peaks) {
    container.innerHTML = "";
    var n = peaks.length;
    // Çok fazla çubuk daralmış görünür; ekran genişliğine göre seyrelt.
    var maxBars = Math.max(80, Math.min(240, Math.floor(container.clientWidth / 4) || 160));
    var data = peaks;
    if (n > maxBars) {
      data = [];
      for (var i = 0; i < maxBars; i++) {
        var start = Math.floor((i / maxBars) * n);
        var end = Math.floor(((i + 1) / maxBars) * n);
        var mx = 0;
        for (var j = start; j < end && j < n; j++) {
          if (peaks[j] > mx) mx = peaks[j];
        }
        data.push(mx);
      }
    }
    var frag = document.createDocumentFragment();
    for (var k = 0; k < data.length; k++) {
      var bar = document.createElement("i");
      bar.className = "ses-wave-bar";
      // 8%–100% arası yükseklik: tamamen sessiz çubuklar da görünür kalsın.
      var h = Math.max(8, Math.round(data[k] * 100));
      bar.style.height = h + "%";
      frag.appendChild(bar);
    }
    container.appendChild(frag);
  }

  function initPlayer(root) {
    var audio = root.querySelector("audio[data-audio-player]");
    var playBtn = root.querySelector("[data-ses-play]");
    var wave = root.querySelector("[data-ses-wave]");
    var barsEl = root.querySelector("[data-ses-bars]");
    var playedEl = root.querySelector("[data-ses-played]");
    var headEl = root.querySelector("[data-ses-head]");
    var curEl = root.querySelector("[data-ses-current]");
    var totEl = root.querySelector("[data-ses-total]");
    var rateSel = root.querySelector("[data-ses-rate]");
    if (!audio || !wave || !barsEl) return;

    // Dalga formu verisi
    var peaks = [];
    var raw = root.getAttribute("data-waveform") || "";
    if (raw) {
      try { peaks = JSON.parse(raw) || []; } catch (_) { peaks = []; }
    }
    if (!Array.isArray(peaks) || peaks.length === 0) {
      peaks = syntheticWave(160);
      root.classList.add("ses-player-synthetic");
    }
    buildBars(barsEl, peaks);

    var duration = parseFloat(root.getAttribute("data-duration")) || 0;

    function setProgress() {
      var d = audio.duration;
      if (!isFinite(d) || d <= 0) d = duration;
      var pct = d > 0 ? (audio.currentTime / d) * 100 : 0;
      if (pct < 0) pct = 0;
      if (pct > 100) pct = 100;
      if (playedEl) playedEl.style.width = pct + "%";
      if (headEl) headEl.style.left = pct + "%";
      if (curEl) curEl.textContent = fmt(audio.currentTime);
      wave.setAttribute("aria-valuenow", String(Math.round(pct)));
    }

    function setTotal() {
      var d = audio.duration;
      if (isFinite(d) && d > 0 && totEl) totEl.textContent = fmt(d);
    }

    function setPlaying(on) {
      root.classList.toggle("is-playing", !!on);
      if (playBtn) {
        playBtn.setAttribute("aria-label", on ? "Duraklat" : "Oynat");
        var p = playBtn.querySelector(".ses-play-icon");
        var q = playBtn.querySelector(".ses-pause-icon");
        if (p) p.style.display = on ? "none" : "";
        if (q) q.style.display = on ? "" : "none";
      }
    }

    /* Duraklatılan yerden devam ederken 2 sn geri sar. Transkript
       düzeltmenin temel kuralı: son duyulan sözcükler tekrar duyulsun.
       Kullanıcı duraklattıktan sonra elle bir yere sardıysa (dalga formu,
       segment veya kelime tıklaması) geri sarma yapılmaz — yoksa tam
       tıkladığı kelimeden uzaklaşırdı. */
    var RESUME_REWIND = 2;
    var pausedAt = -1;

    function toggle() {
      if (audio.paused) {
        var atPause = pausedAt >= 0 && Math.abs(audio.currentTime - pausedAt) < 0.05;
        if (atPause && audio.currentTime > RESUME_REWIND) {
          audio.currentTime = audio.currentTime - RESUME_REWIND;
        }
        pausedAt = -1;
        var pr = audio.play();
        if (pr && pr.catch) pr.catch(function () { setPlaying(false); });
      } else {
        audio.pause();
      }
    }

    /* ---------------- Karaoke vurgu ----------------
       Transkript, oynatıcının kardeşidir (aynı <article> içinde). Kelime
       zamanı varsa oynarken geçerli kelime işaretlenir. Kelime yoksa bu
       bölüm tamamen sessiz kalır ve oynatıcı eskisi gibi çalışır. */
    var scope = root.closest("article") || document;
    var transcript = scope.querySelector("[data-transcript]");
    var words = [];
    if (transcript) {
      [].forEach.call(transcript.querySelectorAll("[data-ses-word]"), function (el) {
        var s = Number.parseInt(el.getAttribute("data-ms") || "", 10);
        var e = Number.parseInt(el.getAttribute("data-end-ms") || "", 10);
        if (isNaN(s)) return;
        // Bitişi eksik/bozuk kelimeye kısa bir pay ver; vurgu yine çalışsın.
        if (isNaN(e) || e <= s) e = s + 300;
        words.push({ el: el, s: s, e: e });
      });
    }
    var activeIdx = -1;

    /* Kelimeler zamana göre sıralı geldiği için ikili arama. timeupdate
       saniyede ~4 kez tetiklenir; 6.000 kelimelik bir röportajda doğrusal
       tarama her seferinde gereksiz iş olurdu. */
    function findWord(ms) {
      var lo = 0, hi = words.length - 1, ans = -1;
      while (lo <= hi) {
        var mid = (lo + hi) >> 1;
        if (words[mid].s <= ms) { ans = mid; lo = mid + 1; }
        else { hi = mid - 1; }
      }
      // ans: başlangıcı ms'den küçük/eşit olan son kelime. Bitişi de geçtiyse
      // kelimeler arası boşluktayız — hiçbir kelime vurgulanmaz.
      if (ans >= 0 && ms <= words[ans].e) return ans;
      return -1;
    }

    /* Vurgulanan kelimeyi transkript kutusunun içinde görünür tut. Yalnız
       kutunun scrollTop'u değişir; sayfa kaydırması ele geçirilmez. */
    function keepVisible(el) {
      if (!transcript) return;
      var box = transcript.getBoundingClientRect();
      var it = el.getBoundingClientRect();
      if (it.top < box.top + 4) {
        transcript.scrollTop -= (box.top + 4 - it.top);
      } else if (it.bottom > box.bottom - 4) {
        transcript.scrollTop += (it.bottom - (box.bottom - 4));
      }
    }

    function syncWord() {
      if (words.length === 0) return;
      var idx = findWord(Math.round(audio.currentTime * 1000));
      if (idx === activeIdx) return;
      if (activeIdx >= 0) words[activeIdx].el.classList.remove("is-now");
      activeIdx = idx;
      if (idx < 0) return;
      words[idx].el.classList.add("is-now");
      // Duraklatılmışken kaydırma yapma: kullanıcı okuyor olabilir.
      if (!audio.paused) keepVisible(words[idx].el);
    }

    /* ---------------- Media Session ----------------
       Klavyenin medya tuşları, kilit ekranı ve işletim sistemi oynatma
       kontrolleri. Oturum tarayıcıda tektir; bu yüzden sahiplik "play"
       anında alınır — sayfada birden çok kayıt varsa son çalan kazanır. */
    function claimMediaSession() {
      if (!("mediaSession" in navigator)) return;
      try {
        if (window.MediaMetadata) {
          navigator.mediaSession.metadata = new window.MediaMetadata({
            title: root.getAttribute("data-title") || "Röportaj kaydı",
            artist: "theblock — röportaj transkripsiyonu",
          });
        }
      } catch (_) { /* yoksay */ }
      var acts = [
        ["play", function () { toggle(); }],
        ["pause", function () { audio.pause(); }],
        ["seekbackward", function (d) {
          var off = (d && d.seekOffset) || 5;
          audio.currentTime = Math.max(0, audio.currentTime - off);
        }],
        ["seekforward", function (d) {
          var off = (d && d.seekOffset) || 5;
          audio.currentTime = Math.min(audio.duration || duration, audio.currentTime + off);
        }],
        ["seekto", function (d) {
          if (d && typeof d.seekTime === "number") audio.currentTime = d.seekTime;
        }],
      ];
      for (var i = 0; i < acts.length; i++) {
        try { navigator.mediaSession.setActionHandler(acts[i][0], acts[i][1]); } catch (_) { /* yoksay */ }
      }
    }

    /* Tıklama/sürükleme → konum. */
    function seekFromEvent(ev) {
      var rect = wave.getBoundingClientRect();
      if (rect.width <= 0) return;
      var clientX = ev.touches && ev.touches[0] ? ev.touches[0].clientX : ev.clientX;
      var ratio = (clientX - rect.left) / rect.width;
      ratio = Math.max(0, Math.min(1, ratio));
      var d = audio.duration;
      if (!isFinite(d) || d <= 0) d = duration;
      if (d > 0) {
        audio.currentTime = ratio * d;
        setProgress();
      }
    }

    var dragging = false;
    wave.addEventListener("mousedown", function (ev) {
      dragging = true;
      seekFromEvent(ev);
      ev.preventDefault();
    });
    document.addEventListener("mousemove", function (ev) {
      if (!dragging) return;
      seekFromEvent(ev);
    });
    document.addEventListener("mouseup", function () { dragging = false; });

    wave.addEventListener("touchstart", function (ev) { seekFromEvent(ev); }, { passive: true });
    wave.addEventListener("touchmove", function (ev) { seekFromEvent(ev); }, { passive: true });

    wave.addEventListener("keydown", function (ev) {
      // Shift: 30 sn kaba atlama · Alt: 1 sn ince atlama · yalın: 5 sn.
      var step = ev.shiftKey ? 30 : (ev.altKey ? 1 : 5);
      switch (ev.key) {
        case "ArrowLeft": audio.currentTime = Math.max(0, audio.currentTime - step); ev.preventDefault(); break;
        case "ArrowRight": audio.currentTime = Math.min(audio.duration || duration, audio.currentTime + step); ev.preventDefault(); break;
        case "Home": audio.currentTime = 0; ev.preventDefault(); break;
        case "End": audio.currentTime = (audio.duration || duration) - 0.1; ev.preventDefault(); break;
        case " ":
        case "Enter": toggle(); ev.preventDefault(); break;
      }
      setProgress();
    });

    if (playBtn) playBtn.addEventListener("click", toggle);

    audio.addEventListener("play", function () { setPlaying(true); claimMediaSession(); });
    audio.addEventListener("pause", function () { setPlaying(false); pausedAt = audio.currentTime; });
    audio.addEventListener("ended", function () { setPlaying(false); setProgress(); });
    audio.addEventListener("timeupdate", setProgress);
    audio.addEventListener("timeupdate", syncWord);
    audio.addEventListener("seeked", function () { pausedAt = -1; syncWord(); });
    audio.addEventListener("loadedmetadata", function () { setTotal(); setProgress(); });
    audio.addEventListener("durationchange", setTotal);
    audio.addEventListener("error", function () {
      root.classList.add("ses-player-error");
      if (totEl) totEl.textContent = "yüklenemedi";
    });

    /* Hız: tercih hatırlanır. */
    if (rateSel) {
      try {
        var saved = localStorage.getItem(RATE_KEY);
        if (saved) {
          var ok = [].some.call(rateSel.options, function (o) { return o.value === saved; });
          if (ok) rateSel.value = saved;
        }
      } catch (_) { /* yoksay */ }
      audio.playbackRate = parseFloat(rateSel.value) || 1;
      rateSel.addEventListener("change", function () {
        var v = parseFloat(rateSel.value) || 1;
        audio.playbackRate = v;
        try { localStorage.setItem(RATE_KEY, String(v)); } catch (_) { /* yoksay */ }
      });
    }

    /* Pencere yeniden boyutlanınca çubuk sayısını yeniden hesapla. */
    var resizeTimer = null;
    window.addEventListener("resize", function () {
      clearTimeout(resizeTimer);
      resizeTimer = setTimeout(function () { buildBars(barsEl, peaks); setProgress(); }, 180);
    });

    setProgress();
    syncWord();
  }

  function initAll() {
    document.querySelectorAll("[data-ses-player]").forEach(function (root) {
      if (root.dataset.sesReady === "1") return;
      root.dataset.sesReady = "1";
      initPlayer(root);
    });
  }

  document.addEventListener("DOMContentLoaded", initAll);
  document.addEventListener("htmx:afterSwap", initAll);
  document.addEventListener("htmx:afterSettle", initAll);
  // İlk yüklemede DOMContentLoaded kaçırılmış olabilir.
  if (document.readyState !== "loading") initAll();
})();
