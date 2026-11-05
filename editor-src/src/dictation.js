/**
 * Dikte — mikrofondan canlı yazma (Faz 117).
 *
 * Tarayıcının SpeechRecognition API'si kullanılır (Chrome/Edge). Ses
 * tarayıcı üreticisinin konuşma servisine gider; bu yüzden gizli röportaj
 * kayıtları için DEĞİL, editörün kendi diktesi için tasarlanmıştır. Sunucu
 * tarafı canlı transkripsiyon (Deepgram) ileride eklenebilir.
 *
 * Davranış:
 *  - Metin imlecin olduğu yere eklenir; hiçbir şey otomatik kaydedilmez.
 *  - Ara sonuç (interim) durum çubuğunda gösterilir, editöre yazılmaz.
 *  - Konuşulan noktalama komutları ayrı söylendiğinde işaret üretir
 *    ("nokta", "virgül", "yeni paragraf"...). Yanlış pozitif olmaması için
 *    yalnızca komut tek başına söylendiğinde uygulanır.
 */

const LANGS = [
  { code: 'tr-TR', label: 'Türkçe' },
  { code: 'en-US', label: 'İngilizce' },
]

const LANG_KEY = 'tb_dictation_lang'

// Tek başına söylendiğinde noktalamaya dönüşen komutlar.
const PUNCTUATION = {
  'nokta': '.',
  'virgül': ',',
  'soru işareti': '?',
  'ünlem': '!',
  'ünlem işareti': '!',
  'iki nokta': ':',
  'iki nokta üst üste': ':',
  'noktalı virgül': ';',
  'tire': '—',
  'uzun tire': '—',
  'tırnak aç': '“',
  'tırnak kapa': '”',
  'parantez aç': '(',
  'parantez kapa': ')',
  'yüzde': '%',
}

let state = null

function speechRecognition() {
  return window.SpeechRecognition || window.webkitSpeechRecognition || null
}

export function dictationSupported() {
  return !!speechRecognition()
}

function currentLang() {
  try {
    const saved = localStorage.getItem(LANG_KEY)
    if (saved && LANGS.some((l) => l.code === saved)) return saved
  } catch (_) { /* localStorage kapalı olabilir */ }
  return 'tr-TR'
}

function saveLang(code) {
  try { localStorage.setItem(LANG_KEY, code) } catch (_) { /* yoksay */ }
}

/** Komut sözcüğünü normalize eder: küçük harf, noktalama ve boşluk sadeleştirme. */
function normalizeCommand(text) {
  let s = String(text || '').toLocaleLowerCase('tr')
  s = s.replace(/[.,!?;:…"'“”()]/g, ' ').replace(/\s+/g, ' ').trim()
  return s
}

/**
 * Her zaman CANLI editör örneğini döndürür.
 *
 * Neden: `boot()` (DOMContentLoaded + htmx swap olayları) editörü yeniden
 * kurar; init sırasında yakalanan `editor` referansı eski (DOM'dan kopmuş)
 * örneğe işaret edebilir. O örneğe yazmak sessizce kaybolur. Bu yüzden
 * yazma anında `window.__contentEditor` kullanılır.
 */
function liveEditor(fallback) {
  return window.__contentEditor || fallback || null
}

/** Metin ekle. Boş/boşluk metin eklenmez. */
function insertText(editor, text) {
  const value = String(text || '').replace(/\s+/g, ' ').trim()
  if (!value) return
  try {
    editor.chain().focus().insertContent(value + ' ').run()
  } catch (err) {
    console.error('[dictation] insert failed', err)
  }
}

/**
 * Noktalama ekler: imleçten önceki boşlukları siler, sonra işaret + boşluk
 * yazar. Böylece "duyurdu ." gibi boşluk-before-punctuation oluşmaz.
 *
 * Tek transaction kullanılır: `chain().focus()` imleci değiştirebildiği için
 * silme ve ekleme farklı konumlara düşebiliyordu.
 */
function insertPunctuation(editor, mark) {
  const { state, view } = editor
  const tr = state.tr
  let pos = state.selection.from
  const from = Math.max(0, pos - 4)
  const tail = state.doc.textBetween(from, pos, '\n', '\n')
  const extra = tail.length - tail.replace(/[ \t]+$/, '').length
  if (extra > 0) {
    tr.delete(pos - extra, pos)
    pos -= extra
  }
  tr.insertText(mark + ' ', pos)
  view.dispatch(tr.scrollIntoView())
  view.focus()
}

/** Paragraf/satır kırılımı ekle. */
function insertBreak(editor, kind) {
  if (kind === 'paragraph') {
    editor.chain().focus().splitBlock().run()
    return
  }
  editor.chain().focus().setHardBreak().run()
}

/** Tek başına söylenen komutu işler; işlendiyse true döner. */
function applyCommand(editor, finalText) {
  const cmd = normalizeCommand(finalText)
  if (!cmd) return false
  if (cmd === 'yeni paragraf' || cmd === 'paragraf') {
    insertBreak(editor, 'paragraph')
    return true
  }
  if (cmd === 'yeni satır' || cmd === 'satır') {
    insertBreak(editor, 'line')
    return true
  }
  if (Object.prototype.hasOwnProperty.call(PUNCTUATION, cmd)) {
    insertPunctuation(editor, PUNCTUATION[cmd])
    return true
  }
  return false
}

function ensureBar(editor) {
  let bar = document.getElementById('dictationBar')
  if (bar) return bar
  const host = document.getElementById('blockEditor')
  if (!host) return null
  bar = document.createElement('div')
  bar.id = 'dictationBar'
  bar.className = 'be-dictation-bar'
  bar.hidden = true
  bar.setAttribute('role', 'status')
  bar.setAttribute('aria-live', 'polite')
  bar.innerHTML = [
    '<span class="be-dict-dot" aria-hidden="true"></span>',
    '<span class="be-dict-label">Dinliyorum…</span>',
    '<span class="be-dict-interim" id="dictationInterim"></span>',
    '<span class="be-dict-hint small muted">Noktalama için tek başına “nokta”, “virgül”, “yeni paragraf” deyin.</span>',
    '<label class="be-dict-lang"><span class="sr-only">Dikte dili</span>',
    '<select id="dictationLang"></select></label>',
    '<button type="button" class="btn ghost small" id="dictationStop">Durdur</button>',
  ].join('')
  const sel = bar.querySelector('#dictationLang')
  LANGS.forEach((l) => {
    const opt = document.createElement('option')
    opt.value = l.code
    opt.textContent = l.label
    sel.appendChild(opt)
  })
  sel.value = currentLang()
  sel.addEventListener('change', () => {
    saveLang(sel.value)
    if (state && state.listening) {
      // Dil değişince tanıyıcıyı yeniden başlat.
      state.restartNeeded = true
      state.recognition.stop()
    }
  })
  bar.querySelector('#dictationStop').addEventListener('click', () => stopDictation())
  host.parentNode.insertBefore(bar, host.nextSibling)
  return bar
}

function setActiveButton(active) {
  document.querySelectorAll('[data-cmd="dictate"]').forEach((b) => {
    b.classList.toggle('is-active', active)
    b.setAttribute('aria-pressed', active ? 'true' : 'false')
  })
}

function showMessage(editor, text, kind) {
  const bar = ensureBar(editor)
  if (!bar) return
  bar.hidden = false
  bar.classList.remove('be-dict-error')
  if (kind === 'error') bar.classList.add('be-dict-error')
  bar.querySelector('.be-dict-label').textContent = text
  const interim = bar.querySelector('#dictationInterim')
  if (interim) interim.textContent = ''
}

export function stopDictation() {
  if (!state) return
  state.listening = false
  state.restartNeeded = false
  try { state.recognition.stop() } catch (_) { /* zaten durmuş olabilir */ }
  const bar = document.getElementById('dictationBar')
  if (bar) {
    document.getElementById('dictationInterim').textContent = ''
    bar.classList.remove('be-dict-error')
    bar.hidden = true
  }
  setActiveButton(false)
}

/** Dikteyi başlat/durdur. */
export function toggleDictation(editor) {
  if (!editor) return
  if (state && state.listening) {
    stopDictation()
    return
  }
  const SR = speechRecognition()
  if (!SR) {
    showMessage(editor, 'Bu tarayıcı sesle yazmayı desteklemiyor. Chrome veya Edge kullanın.', 'error')
    return
  }
  if (!window.isSecureContext) {
    showMessage(editor, 'Sesle yazma yalnızca HTTPS (veya localhost) üzerinde çalışır.', 'error')
    return
  }

  const bar = ensureBar(editor)
  if (bar) {
    bar.hidden = false
    bar.classList.remove('be-dict-error')
    bar.querySelector('.be-dict-label').textContent = 'Dinliyorum…'
    bar.querySelector('#dictationInterim').textContent = ''
  }

  const recognition = new SR()
  recognition.lang = currentLang()
  recognition.continuous = true
  recognition.interimResults = true
  recognition.maxAlternatives = 1

  state = { recognition, listening: true, restartNeeded: false, editor }

  recognition.onresult = (event) => {
    let interim = ''
    // Yazma anında canlı örneği çöz: editör yeniden kurulmuş olabilir.
    const ed = liveEditor(state && state.editor)
    if (!ed) return
    for (let i = event.resultIndex; i < event.results.length; i++) {
      const result = event.results[i]
      const text = result[0] ? result[0].transcript : ''
      if (!result.isFinal) {
        interim += text
        continue
      }
      if (applyCommand(ed, text)) continue
      insertText(ed, text)
    }
    const interimEl = document.getElementById('dictationInterim')
    if (interimEl) interimEl.textContent = interim
  }

  recognition.onerror = (event) => {
    const code = event && event.error
    if (code === 'no-speech') return // sessizlik: yeniden başlatma döngüsü hallediyor
    if (code === 'aborted') return
    let msg = 'Ses tanınamadı. Mikrofonu ve izni kontrol edin.'
    if (code === 'not-allowed' || code === 'service-not-allowed') {
      msg = 'Mikrofon izni verilmedi. Tarayıcı adres çubuğundan izin verin.'
      state.listening = false
    }
    if (code === 'audio-capture') msg = 'Mikrofon bulunamadı.'
    if (code === 'network') msg = 'Konuşma servisine ulaşılamadı (ağ hatası).'
    showMessage(editor, msg, 'error')
    if (!state.listening) setActiveButton(false)
  }

  recognition.onend = () => {
    if (!state) return
    if (state.listening) {
      // Chrome sessizlikte kendiliğinden durur; kullanıcı durdurmadıysa
      // yeniden başlat (dil değiştiyse yeni dil ile).
      state.restartNeeded = false
      try {
        state.recognition.lang = currentLang()
        state.recognition.start()
      } catch (_) {
        stopDictation()
      }
      return
    }
    stopDictation()
  }

  try {
    recognition.start()
    setActiveButton(true)
  } catch (_) {
    showMessage(editor, 'Dikte başlatılamadı. Sayfayı yenileyip tekrar deneyin.', 'error')
    state = null
  }
}

export { LANGS }
