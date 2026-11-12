
(function () {
  'use strict';

  var YT_KOKLER = ['youtube-nocookie.com', 'youtube.com'];

  function ikiHane(n) { return (n < 10 ? '0' : '') + n; }

  
  function sureMetni(sn) {
    if (!isFinite(sn) || sn < 0) return '0:00';
    sn = Math.floor(sn);
    var s = sn % 60, d = Math.floor(sn / 60) % 60, sa = Math.floor(sn / 3600);
    return sa > 0 ? sa + ':' + ikiHane(d) + ':' + ikiHane(s) : d + ':' + ikiHane(s);
  }

  function youtubeKaynagi(e) {
    try {
      var o = new URL(e.origin);
      for (var i = 0; i < YT_KOKLER.length; i++) {
        if (o.hostname === YT_KOKLER[i] || o.hostname.endsWith('.' + YT_KOKLER[i])) return true;
      }
    } catch (err) {  }
    return false;
  }

  function Oynatici(kok) {
    this.kok = kok;
    this.videoId = kok.getAttribute('data-video') || '';
    this.canli = kok.getAttribute('data-canli') === '1';
    this.kanal = kok.getAttribute('data-kanal') || '';
    this.baslik = kok.getAttribute('data-baslik') || 'Video';
    this.sonraki = kok.getAttribute('data-sonraki') || '';
    this.sonrakiBaslik = kok.getAttribute('data-sonraki-baslik') || '';

    this.sahne = kok.querySelector('.cd-vplayer__sahne');
    this.katman = kok.querySelector('[data-cd-katman]');
    this.cerceve = kok.querySelector('[data-cd-cerceve]');
    this.kapak = kok.querySelector('[data-cd-kapak]');
    this.bos = kok.querySelector('[data-cd-bos]');
    this.buyuk = kok.querySelector('[data-cd-buyuk]');
    this.bekle = kok.querySelector('[data-cd-bekle]');
    this.rozet = kok.querySelector('[data-cd-rozet]');
    this.canliRozet = kok.querySelector('[data-cd-canli-rozet]');
    this.kontrol = kok.querySelector('[data-cd-kontrol]');
    this.cizgi = kok.querySelector('[data-cd-cizgi]');
    this.tamponEl = kok.querySelector('[data-cd-tampon]');
    this.doluEl = kok.querySelector('[data-cd-dolu]');
    this.oynatBtn = kok.querySelector('[data-cd-oynat]');
    this.sesBtn = kok.querySelector('[data-cd-ses]');
    this.sesAralik = kok.querySelector('[data-cd-ses-aralik]');
    this.gecenEl = kok.querySelector('[data-cd-gecen]');
    this.toplamEl = kok.querySelector('[data-cd-toplam]');
    this.hizBtn = kok.querySelector('[data-cd-hiz]');
    this.menu = kok.querySelector('[data-cd-menu]');
    this.tamBtn = kok.querySelector('[data-cd-tam]');
    this.duyuru = kok.querySelector('[data-cd-duyuru]');

    this.cerceveHazir = false;
    this.hazir = false;
    this.oynuyor = false;
    this.basladi = false;
    this.sure = 0;
    this.an = 0;
    this.ses = 100;
    this.sessiz = false;
    this.hiz = 1;
    this._zamanlayici = null;

    
    if (this.canli) this.kok.classList.add('cd-vplayer--canli');

    this.bagla();
  }

  Oynatici.prototype.bagla = function () {
    var self = this;

    this.buyuk.addEventListener('click', function () { self.baslat(); });
    this.oynatBtn.addEventListener('click', function () { self.oynatDegistir(); });
    this.sesBtn.addEventListener('click', function () { self.sessizDegistir(); });
    this.sesAralik.addEventListener('input', function () {
      self.sesAyarla(Number.parseInt(self.sesAralik.value, 10), true);
    });
    this.tamBtn.addEventListener('click', function () { self.tamEkranDegistir(); });
    this.hizBtn.addEventListener('click', function (e) {
      e.stopPropagation();
      self.menuDegistir();
    });
    this.menu.addEventListener('click', function (e) {
      var b = e.target.closest ? e.target.closest('[data-cd-hiz-sec]') : null;
      if (!b) return;
      self.hizAyarla(parseFloat(b.getAttribute('data-cd-hiz-sec')));
      self.menuDegistir(false);
    });

    
    this.cizgi.addEventListener('pointerdown', function (e) {
      if (self.canli || !self.hazir) return;
      self.cizgi.setPointerCapture(e.pointerId);
      self.sarmaBasladi = true;
      self.cizgiKonumu(e);
    });
    this.cizgi.addEventListener('pointermove', function (e) {
      if (self.sarmaBasladi) self.cizgiKonumu(e);
    });
    this.cizgi.addEventListener('pointerup', function (e) {
      if (!self.sarmaBasladi) return;
      self.sarmaBasladi = false;
      try { self.cizgi.releasePointerCapture(e.pointerId); } catch (err) {  }
      self.sarmaBitir(e);
    });
    this.cizgi.addEventListener('keydown', function (e) {
      if (self.canli || !self.hazir) return;
      var adim = e.shiftKey ? 30 : 5;
      if (e.key === 'ArrowRight') { self.sarma(self.an + adim); e.preventDefault(); }
      else if (e.key === 'ArrowLeft') { self.sarma(self.an - adim); e.preventDefault(); }
      else if (e.key === 'Home') { self.sarma(0); e.preventDefault(); }
      else if (e.key === 'End') { self.sarma(self.sure - 0.5); e.preventDefault(); }
    });

    
    this.kok.tabIndex = -1;
    this.kok.addEventListener('keydown', function (e) {
      if (e.target.matches('input,select,textarea')) return;
      var k = e.key;
      if (k === ' ' || k === 'k' || k === 'K') { self.oynatDegistir(); e.preventDefault(); }
      else if (k === 'ArrowRight') { self.sarma(self.an + 5); e.preventDefault(); }
      else if (k === 'ArrowLeft') { self.sarma(self.an - 5); e.preventDefault(); }
      else if (k === 'ArrowUp') { self.sesAyarla(self.ses + 5, true); e.preventDefault(); }
      else if (k === 'ArrowDown') { self.sesAyarla(self.ses - 5, true); e.preventDefault(); }
      else if (k === 'm' || k === 'M') { self.sessizDegistir(); e.preventDefault(); }
      else if (k === 'f' || k === 'F') { self.tamEkranDegistir(); e.preventDefault(); }
    });

    
    this.kok.addEventListener('pointermove', function () { self.kontrolGoster(); });
    this.kok.addEventListener('pointerdown', function () {
      
      if (self.kok.classList.contains('cd-vplayer--sakin')) {
        self._sadeceGoster = true;
        setTimeout(function () { self._sadeceGoster = false; }, 700);
      }
      self.kontrolGoster();
    });
    this.kok.addEventListener('pointerenter', function () {
      self._fareUzerinde = true;
      self.kontrolGoster();
    });
    
    this.kok.addEventListener('focusin', function () { self.kontrolGoster(); });
    this.kok.addEventListener('pointerleave', function () {
      self._fareUzerinde = false;
      if (self.oynuyor) self.kok.classList.add('cd-vplayer--sakin');
    });

    
    this.sahne.addEventListener('click', function (e) {
      if (e.target.closest('[data-cd-kontrol]')) return;
      if (e.target.closest('[data-cd-buyuk]')) return;
      if (e.target.closest('[data-cd-ses-cip]')) return;
      if (e.target.closest('.cd-vplayer__kunye')) return;
      if (e.target.closest('.cd-vplayer__son')) return;
      if (!self._oynadi) {
        
        self.baslat();
        return;
      }
      
      if (self._sadeceGoster) return;
      if (Date.now() - (self._baslamaZaman || 0) < 900) return;
      self.oynatDegistir();
    });

    document.addEventListener('fullscreenchange', function () {
      var tam = document.fullscreenElement === self.kok;
      self.kok.classList.toggle('cd-vplayer--tam', tam);
      self.simgeDegistir(self.tamBtn, '--tam', !tam);
      self.tamBtn.setAttribute('aria-label', tam ? 'Tam ekrandan çık' : 'Tam ekran');
    });

    window.addEventListener('message', function (e) { self.mesaj(e); });

    
    document.addEventListener('visibilitychange', function () {
      if (document.hidden || !self.hazir) return;
      self.komut('getPlayerState');
      if (self.oynuyor) self.kontrolGoster();
    });
    document.addEventListener('click', function (e) {
      if (!self.kok.contains(e.target)) self.menuDegistir(false);
    });
  };

  
  Oynatici.prototype.kaynak = function (sessiz) {
    var temel = 'https://www.youtube-nocookie.com/embed/';
    var ortak = 'autoplay=1&controls=0&enablejsapi=1&rel=0&modestbranding=1' +
      '&playsinline=1&iv_load_policy=3&fs=0&cc_load_policy=0' +
      '&origin=' + encodeURIComponent(location.origin);
    if (sessiz) ortak = 'mute=1&' + ortak;
    if (this.canli) {
      return temel + 'live_stream?channel=' + encodeURIComponent(this.kanal) + '&' + ortak;
    }
    return temel + encodeURIComponent(this.videoId) + '?' + ortak;
  };

  
  Oynatici.prototype.sesiGeriAc = function () {
    if (this._sesAcildi) return;
    
    this.komut('setVolume', [this.sessiz ? 0 : this.ses]);
    this._sesAcildi = true;
  };

  
  Oynatici.prototype.cerceveYenile = function () {
    if (this._zamanlayici) { clearInterval(this._zamanlayici); this._zamanlayici = null; }
    this.cerceve.innerHTML = '';
    this.iframe = null;
    this.cerceveHazir = false;
    this.hazir = false;
    this._sonDurum = undefined;
    this._sesAcildi = false;
  };

  Oynatici.prototype.cerceveKur = function (sessiz) {
    if (this.cerceveHazir) return;
    this.cerceveHazir = true;
    var f = document.createElement('iframe');
    f.className = 'cd-vplayer__iframe';
    this.sessizBasladi = !!sessiz;
    f.src = this.kaynak(!!sessiz);
    f.title = this.baslik;
    f.setAttribute('frameborder', '0');
    f.setAttribute('allow',
      'accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share');
    f.setAttribute('allowfullscreen', '');
    f.setAttribute('referrerpolicy', 'strict-origin-when-cross-origin');
    this.cerceve.appendChild(f);
    this.iframe = f;

    var self = this;
    f.addEventListener('load', function () { self.elSikisi(); });
    
    this._zamanlayici = setInterval(function () { self.elSikisi(); }, 800);
    setTimeout(function () {
      if (!self.hazir && self.bekle) self.bekle.hidden = true;
    }, 12000);
  };

  
  Oynatici.prototype.elSikisi = function () {
    if (!this.iframe || !this.iframe.contentWindow) return;
    try {
      this.iframe.contentWindow.postMessage(JSON.stringify({
        event: 'listening', id: 'cd-player', channel: 'widget'
      }), '*');
    } catch (err) {  }
  };

  Oynatici.prototype.komut = function (fn, args) {
    if (!this.iframe || !this.iframe.contentWindow) return;
    try {
      this.iframe.contentWindow.postMessage(JSON.stringify({
        event: 'command', func: fn, args: args || []
      }), '*');
    } catch (err) {  }
  };

  
  Oynatici.prototype.mesaj = function (e) {
    if (!youtubeKaynagi(e)) return;
    var v;
    try { v = JSON.parse(e.data); } catch (err) { return; }
    if (!v || !v.event) return;

    if (v.event === 'onReady' || v.event === 'initialDelivery') {
      this.hazir = true;
      if (this._zamanlayici) { clearInterval(this._zamanlayici); this._zamanlayici = null; }
      
      this.komut('setVolume', [this.sessiz ? 0 : this.ses]);
      this.komut('setPlaybackRate', [this.hiz]);
      if (v.info && typeof v.info.duration === 'number') this.bilgi(v.info);
      return;
    }
    if (v.event === 'onStateChange') {
      this.durum(v.info);
      return;
    }
    if (v.event === 'infoDelivery' && v.info) {
      this.bilgi(v.info);
    }
  };

  Oynatici.prototype.durum = function (s) {
    if (typeof s !== 'number') return;
    if (s === 1) {
      this.oynuyor = true;
      this.basladi = true;
      
      this.oynamaBasladi();
      this.sesiGeriAc();
    }
    else if (s === 2 || s === 5 || s === -1) {
      this.oynuyor = false;
      
      this.kok.classList.remove('cd-vplayer--bekliyor');
      this.kok.classList.remove('cd-vplayer--oynuyor');
      if (this.bekle) this.bekle.hidden = true;
      
      if (this._oynadi && this.buyuk && !this.canli) {
        
        this.buyuk.hidden = false;
        this.buyuk.setAttribute('aria-label', 'Oynat');
      }
      
      if (this.sessizBasladi && !this._kullaniciSesiKapatti) this.sesCipiGoster();
    }
    else if (s === 0) { this.oynuyor = false; this.bitis(); }
    else if (s === 3) { if (this.bekle) this.bekle.hidden = false; }
    this.dugmeDurumlari();
  };

  
  Oynatici.prototype.bilgi = function (i) {
    if (typeof i.duration === 'number' && i.duration > 0) this.sure = i.duration;
    if (typeof i.currentTime === 'number') this.an = i.currentTime;
    if (typeof i.playerState === 'number' && i.playerState !== this._sonDurum) {
      this._sonDurum = i.playerState;
      this.durum(i.playerState);
    }
    if (this.bekle) this.bekle.hidden = true;
    this.ciz(i.videoLoadedFraction);
  };

  

  
  Oynatici.prototype.baslat = function () {
    var self = this;

    
    if (this.basladi && !this._oynadi) {
      this.cerceveYenile();
      this.sessizDenendi = false;
    }

    
    var sessiz = this.sessizDenendi === true || this.sessiz === true || this.ses === 0;
    this.cerceveKur(sessiz);

    if (this.bos) this.bos.hidden = true;
    
    this.kok.classList.add('cd-vplayer--bekliyor', 'cd-vplayer--basladi');
    this._baslamaZaman = Date.now();
    if (this.bekle) this.bekle.hidden = false;
    if (this.buyuk) this.buyuk.hidden = true;
    this.kok.classList.remove('cd-vplayer--gecikti');
    this.basladi = true;
    this.oynuyor = true;
    this.dugmeDurumlari();

    this.komut('playVideo');
    
    [150, 400, 800, 1400, 2200, 3200].forEach(function (ms) {
      setTimeout(function () {
        if (!self._oynadi) self.komut('playVideo');
      }, ms);
    });

    

    
    setTimeout(function () {
      if (self._oynadi || self.sessizBasladi) return;
      self.sessizDenendi = true;
      self.cerceveYenile();
      self.cerceveKur(true);
      self.komut('playVideo');
      [200, 600, 1200, 2200].forEach(function (ms) {
        setTimeout(function () { if (!self._oynadi) self.komut('playVideo'); }, ms);
      });
      self.sesCipiGoster();
    }, 1200);

    
    setTimeout(function () {
      if (self._oynadi) return;
      self.kapakGeriGetir();
    }, 5200);

    this.kontrolGoster();
  };

  
  Oynatici.prototype.oynamaBasladi = function () {
    this._oynadi = true;
    this.kok.classList.remove('cd-vplayer--bekliyor', 'cd-vplayer--gecikti');
    this.kok.classList.add('cd-vplayer--oynuyor');
    if (this.kapak) this.kapak.hidden = true;
    if (this.buyuk) this.buyuk.hidden = true;
    if (this.bekle) this.bekle.hidden = true;
    if (this.bos) this.bos.hidden = true;
    if (this.buyuk) this.buyuk.setAttribute('aria-label', 'Duraklat');
    
    if (this.sessizBasladi && !this._kullaniciSesiKapatti && !this.sessiz && this.ses > 0) {
      this.sesCipiGoster();
    } else {
      this.sesCipiGizle();
    }
  };

  
  Oynatici.prototype.kapakGeriGetir = function () {
    this.kok.classList.remove('cd-vplayer--bekliyor', 'cd-vplayer--oynuyor');
    this.kok.classList.add('cd-vplayer--gecikti');
    if (this.kapak) this.kapak.hidden = false;
    if (this.buyuk) this.buyuk.hidden = false;
    if (this.bekle) this.bekle.hidden = true;
    this.cizgiMesaj('Oynatmak için tekrar dokunun');
  };

  Oynatici.prototype.cizgiMesaj = function (metin) {
    if (this.duyuru) this.duyuru.textContent = metin;
    if (this.kontrol) this.kontrol.setAttribute('data-mesaj', metin);
  };

  
  Oynatici.prototype.sesCipiGoster = function () {
    if (this.sessiz || this.ses === 0) return;
    var cip = this.kok.querySelector('[data-cd-ses-cip]');
    if (!cip) {
      cip = document.createElement('button');
      cip.type = 'button';
      cip.className = 'cd-vplayer__ses-cip';
      cip.setAttribute('data-cd-ses-cip', '');
      cip.innerHTML = '<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" aria-hidden="true">' +
        '<path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3a4.5 4.5 0 0 0-2.5-4v8a4.5 4.5 0 0 0 2.5-4zM14 3.2v2.1a6.8 6.8 0 0 1 0 13.4v2.1a8.9 8.9 0 0 0 0-17.6z"/></svg> Sesi aç';
      var self = this;
      cip.addEventListener('click', function (e) {
        e.stopPropagation();
        self.sesiAc();
      });
      this.sahne.appendChild(cip);
    }
    cip.hidden = false;
  };

  Oynatici.prototype.sesCipiGizle = function () {
    var cip = this.kok.querySelector('[data-cd-ses-cip]');
    if (cip) cip.hidden = true;
  };

  
  Oynatici.prototype.sesiAc = function () {
    this._kullaniciSesiKapatti = false;
    this.sessiz = false;
    if (this.ses === 0) this.ses = 100;
    this.komut('unMute');
    this.komut('setVolume', [this.ses]);
    this.sesCipiGizle();
    this.dugmeDurumlari();
    this.duyur('Ses açıldı');
  };

  Oynatici.prototype.oynatDegistir = function () {
    if (!this.basladi || !this._oynadi) { this.baslat(); return; }
    if (this.oynuyor) {
      this.komut('pauseVideo');
      this.oynuyor = false;
    } else {
      this.komut('playVideo');
      this.oynuyor = true;
      
      this.kok.classList.add('cd-vplayer--oynuyor');
      if (this.kapak) this.kapak.hidden = true;
      if (this.buyuk) this.buyuk.hidden = true;
    }
    this.dugmeDurumlari();
  };

  Oynatici.prototype.sarma = function (sn) {
    if (!this.hazir) return;
    if (this.sure > 0) sn = Math.max(0, Math.min(this.sure - 0.2, sn));
    else sn = Math.max(0, sn);
    this.komut('seekTo', [sn, true]);
    this.an = sn;
    this.ciz();
  };

  Oynatici.prototype.cizgiKonumu = function (e) {
    var r = this.cizgi.getBoundingClientRect();
    var oran = Math.max(0, Math.min(1, (e.clientX - r.left) / r.width));
    this._sarmaOran = oran;
    this.ciz(undefined, oran);
  };

  Oynatici.prototype.sarmaBitir = function (e) {
    var oran = this._sarmaOran;
    if (oran === undefined) {
      var r = this.cizgi.getBoundingClientRect();
      oran = Math.max(0, Math.min(1, (e.clientX - r.left) / r.width));
    }
    this._sarmaOran = undefined;
    this.sarma(oran * (this.sure || 0));
  };

  Oynatici.prototype.sesAyarla = function (deger, uygula) {
    deger = Math.max(0, Math.min(100, Math.round(deger)));
    this.ses = deger;
    if (this.sesAralik) this.sesAralik.value = String(deger);
    this.sessiz = deger === 0;
    if (uygula) {
      this.komut('setVolume', [deger]);
      this.komut(deger === 0 ? 'mute' : 'unMute');
    }
    this.dugmeDurumlari();
  };

  Oynatici.prototype.sessizDegistir = function () {
    this._kullaniciSesiKapatti = !(this.sessiz || this.ses === 0) ? true : false;
    if (this.sessiz || this.ses === 0) {
      this.sessiz = false;
      if (this.ses === 0) this.ses = 100;
      this.komut('unMute');
      this.komut('setVolume', [this.ses]);
      if (this.sesAralik) this.sesAralik.value = String(this.ses);
    } else {
      this.sessiz = true;
      this.komut('mute');
    }
    this.dugmeDurumlari();
  };

  Oynatici.prototype.hizAyarla = function (h) {
    if (!isFinite(h) || h <= 0) return;
    this.hiz = h;
    this.komut('setPlaybackRate', [h]);
    if (this.hizBtn) this.hizBtn.textContent = (h % 1 === 0 ? h : h) + 'x';
    this.duyur('Oynatma hızı ' + h + 'x');
  };

  Oynatici.prototype.menuDegistir = function (acik) {
    if (!this.menu) return;
    var suan = acik === undefined ? this.menu.hidden : acik;
    this.menu.hidden = !suan;
    this.hizBtn.setAttribute('aria-expanded', suan ? 'true' : 'false');
  };

  Oynatici.prototype.tamEkranDegistir = function () {
    var self = this;
    if (document.fullscreenElement === this.kok) {
      if (document.exitFullscreen) document.exitFullscreen();
      return;
    }
    if (this.kok.requestFullscreen) {
      
      var sonuc = this.kok.requestFullscreen();
      if (sonuc && sonuc.catch) {
        sonuc.catch(function () {
          self.kok.classList.add('cd-vplayer--sahte-tam');
          var cik = function (ev) {
            if (ev.key === 'Escape') {
              self.kok.classList.remove('cd-vplayer--sahte-tam');
              document.removeEventListener('keydown', cik);
            }
          };
          document.addEventListener('keydown', cik);
          self.duyur('Tam ekran');
        });
      }
      return;
    }
    
    this.kok.classList.toggle('cd-vplayer--sahte-tam');
    self.duyur('Tam ekran');
  };

  Oynatici.prototype.kontrolGoster = function () {
    var self = this;
    this.kok.classList.remove('cd-vplayer--sakin');
    clearTimeout(this._sakinZaman);
    if (!this.oynuyor) return;
    this._sakinZaman = setTimeout(function () {
      if (!self.oynuyor) return;
      
      if (self.menu && !self.menu.hidden) { self.kontrolGoster(); return; }
      
      if (self._fareUzerinde) { self.kontrolGoster(); return; }
      self.kok.classList.add('cd-vplayer--sakin');
    }, 2500);
  };

  Oynatici.prototype.bitis = function () {
    if (this.canli) return;
    this.dugmeDurumlari();
    
    var sonaYakin = this.sure > 0 && this.an >= this.sure - 2;
    if (this._oynadi && sonaYakin && this.kok.classList.contains('cd-vplayer--basladi')) {
      this.sonEkrani();
    }
  };

  
  Oynatici.prototype.sonEkrani = function () {
    var self = this;
    var eski = this.kok.querySelector('.cd-vplayer__son');
    if (eski) eski.remove();

    var kap = document.createElement('div');
    kap.className = 'cd-vplayer__son';
    var ic = document.createElement('div');
    ic.className = 'cd-vplayer__son-ic';

    var tekrar = document.createElement('button');
    tekrar.type = 'button';
    tekrar.className = 'cd-vplayer__son-tekrar';
    tekrar.innerHTML = '<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" aria-hidden="true"><path d="M12 5V1L7 6l5 5V7a6 6 0 1 1-6 6H4a8 8 0 1 0 8-8z"/></svg> Tekrar oynat';
    tekrar.addEventListener('click', function (e) {
      e.stopPropagation();
      kap.remove();
      self.sarma(0);
      self.komut('playVideo');
      self.oynuyor = true;
      self.dugmeDurumlari();
    });
    ic.appendChild(tekrar);

    if (this.sonraki) {
      var a = document.createElement('a');
      a.className = 'cd-vplayer__son-sonraki';
      a.href = '/video/' + this.sonraki;
      var etiket = document.createElement('span');
      etiket.className = 'cd-vplayer__son-etiket';
      etiket.textContent = 'SIRADAKİ';
      var ad = document.createElement('span');
      ad.className = 'cd-vplayer__son-baslik';
      ad.textContent = this.sonrakiBaslik || 'Sıradaki video';
      a.appendChild(etiket);
      a.appendChild(ad);
      ic.appendChild(a);

      
      var sayac = document.createElement('span');
      sayac.className = 'cd-vplayer__son-sayac';
      var kalan = 5;
      sayac.textContent = kalan + ' sn içinde geçilecek';
      ic.appendChild(sayac);
      var z = setInterval(function () {
        kalan--;
        if (kalan <= 0) {
          clearInterval(z);
          location.href = '/video/' + self.sonraki;
          return;
        }
        sayac.textContent = kalan + ' sn içinde geçilecek';
      }, 1000);
      a.addEventListener('click', function () { clearInterval(z); });
      kap.addEventListener('click', function () { clearInterval(z); sayac.textContent = 'Otomatik geçiş durduruldu'; });
    }

    kap.appendChild(ic);
    this.sahne.appendChild(kap);
  };

  
  Oynatici.prototype.ciz = function (tampon, sarmaOran) {
    if (!this.cizgi) return;
    var oran = sarmaOran;
    if (oran === undefined) oran = this.sure > 0 ? this.an / this.sure : 0;
    oran = Math.max(0, Math.min(1, oran));
    if (this.doluEl) this.doluEl.style.width = (oran * 100) + '%';
    if (tampon !== undefined && this.tamponEl) {
      this.tamponEl.style.width = (Math.max(0, Math.min(1, tampon)) * 100) + '%';
    }
    if (this.gecenEl) this.gecenEl.textContent = sureMetni(oran * (this.sure || 0));
    if (this.toplamEl && !this.canli && this.sure > 0) {
      this.toplamEl.textContent = sureMetni(this.sure);
    }
    if (this.rozet && !this.canli && this.sure > 0 && this.basladi) {
      
      this.rozet.textContent = '-' + sureMetni(Math.max(0, this.sure - oran * this.sure));
    }
    this.cizgi.setAttribute('aria-valuenow', String(Math.round(oran * 100)));
    this.cizgi.setAttribute('aria-valuetext',
      sureMetni(oran * (this.sure || 0)) + ' / ' + sureMetni(this.sure || 0));
  };

  Oynatici.prototype.dugmeDurumlari = function () {
    if (!this.oynatBtn) return;
    this.simgeDegistir(this.oynatBtn, '--duraklat', this.oynuyor);
    this.oynatBtn.setAttribute('aria-label', this.oynuyor ? 'Duraklat' : 'Oynat');
    if (this.sesBtn) {
      this.simgeDegistir(this.sesBtn, '--sessiz', this.sessiz || this.ses === 0);
      this.sesBtn.setAttribute('aria-label',
        (this.sessiz || this.ses === 0) ? 'Sesi aç' : 'Sesi kapat');
    }
    if (this.canli) this.kok.classList.add('cd-vplayer--canli');
  };

  Oynatici.prototype.simgeDegistir = function (btn, ad, goster) {
    if (!btn) return;
    var a = btn.querySelector('.cd-vplayer__simge--oynat');
    var b = btn.querySelector('.cd-vplayer__simge--duraklat');
    var ses = btn.querySelector('.cd-vplayer__simge--ses');
    var sessiz = btn.querySelector('.cd-vplayer__simge--sessiz');
    var tam = btn.querySelector('.cd-vplayer__simge--tam');
    var tamdan = btn.querySelector('.cd-vplayer__simge--tamdan');
    if (ad === '--duraklat' && a && b) { a.hidden = goster; b.hidden = !goster; }
    else if (ad === '--sessiz' && ses && sessiz) { ses.hidden = goster; sessiz.hidden = !goster; }
    else if (ad === '--tam' && tam && tamdan) { tam.hidden = !goster; tamdan.hidden = goster; }
  };

  Oynatici.prototype.duyur = function (metin) {
    if (this.duyuru) this.duyuru.textContent = metin;
  };

  
  function kur() {
    var kokler = document.querySelectorAll('[data-cd-player]');
    for (var i = 0; i < kokler.length; i++) {
      if (kokler[i].__cdPlayer) continue;
      kokler[i].__cdPlayer = new Oynatici(kokler[i]);
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', kur);
  } else {
    kur();
  }
})();
