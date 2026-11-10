import { Editor, Node, Mark, Extension, mergeAttributes } from '@tiptap/core'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Underline from '@tiptap/extension-underline'
import CodeBlock from '@tiptap/extension-code-block'
import Placeholder from '@tiptap/extension-placeholder'
import CharacterCount from '@tiptap/extension-character-count'
import { TableKit } from '@tiptap/extension-table'
import { Suggestion } from '@tiptap/suggestion'
import { BubbleMenu } from '@tiptap/extension-bubble-menu'
import { DragHandle } from '@tiptap/extension-drag-handle'
import { NodeRange, NodeRangeSelection } from '@tiptap/extension-node-range'
import Collaboration from '@tiptap/extension-collaboration'
import { HocuspocusProvider } from '@hocuspocus/provider'
import * as Y from 'yjs'
import { Awareness } from 'y-protocols/awareness'
import { yCursorPlugin, ySyncPluginKey } from '@tiptap/y-tiptap'
import { marked } from 'marked'
import { toggleDictation, stopDictation, dictationSupported } from './dictation.js'

let suggestionMode = false
let suggestionUserName = 'Editör'
let lastInsertSuggestion = null

// Öneri kimliği: yalnız DOM içinde benzersiz olması yeter, gizlilik değeri
// taşımaz. Yine de Math.random KULLANILMAZ (Sonar S2245): zayıf PRNG'yi
// "kimlik" gerekçesiyle kod tabanında bırakmak, bir gün yetki jetonu için
// kopyalanmasının önünü açar. Sayaç + zaman damgası hem benzersiz hem
// tahmin edilebilir biçimde sıralı.
let suggestionSayac = 0

function suggestionID() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  suggestionSayac += 1
  return 'oneri-' + Date.now().toString(36) + '-' + suggestionSayac.toString(36)
}

function newSuggestionAttrs() {
  return { id: suggestionID(), user: suggestionUserName || 'Editör', createdAt: Date.now() }
}

const SuggestionInsert = Mark.create({
  name: 'suggestionInsert',
  inclusive: false,
  excludes: 'suggestionDelete',
  addAttributes() {
    return {
      id: { default: '' },
      user: { default: '' },
      createdAt: { default: 0 },
    }
  },
  parseHTML() {
    return [{ tag: 'span[data-suggestion-insert]' }]
  },
  renderHTML({ HTMLAttributes }) {
    return ['span', {
      'data-suggestion-insert': '',
      'data-suggestion-id': HTMLAttributes.id || '',
      'data-user': HTMLAttributes.user || '',
      class: 'suggestion-insert',
    }, 0]
  },
})

const SuggestionDelete = Mark.create({
  name: 'suggestionDelete',
  inclusive: false,
  excludes: 'suggestionInsert',
  addAttributes() {
    return {
      id: { default: '' },
      user: { default: '' },
      createdAt: { default: 0 },
    }
  },
  parseHTML() {
    return [{ tag: 'span[data-suggestion-delete]' }]
  },
  renderHTML({ HTMLAttributes }) {
    return ['span', {
      'data-suggestion-delete': '',
      'data-suggestion-id': HTMLAttributes.id || '',
      'data-user': HTMLAttributes.user || '',
      class: 'suggestion-delete',
    }, 0]
  },
})

const SuggestionTracker = Extension.create({
  name: 'suggestionTracker',
  addProseMirrorPlugins() {
    return [new Plugin({
      key: new PluginKey('suggestionTracker'),
      appendTransaction: (transactions, oldState, newState) => {
        if (!suggestionMode) return null
        if (transactions.some((tr) => tr.getMeta('suggestion'))) return null
        if (transactions.some((tr) => {
          const meta = tr.getMeta(ySyncPluginKey)
          return !!(meta && meta.isChangeOrigin)
        })) return null
        if (!transactions.some((tr) => tr.docChanged)) return null
        const type = newState.schema.marks.suggestionInsert
        if (!type) return null
        const tr = newState.tr
        let changed = false
        transactions.forEach((tx) => {
          if (!tx.docChanged) return
          tx.steps.forEach((step, index) => {
            if (!step.slice || step.slice.content.size === 0) return
            if (!step.slice.content.textBetween(0, step.slice.content.size, '\n')) return
            let alreadyMarked = false
            step.slice.content.descendants((node) => {
              if (node.marks && node.marks.some((mark) => mark.type.name === 'suggestionInsert')) alreadyMarked = true
            })
            if (alreadyMarked) return
            const from = tx.mapping.slice(0, index).map(step.from)
            const to = from + step.slice.content.size
            if (from < 0 || to <= from || to > tr.doc.content.size) return
            const now = Date.now()
            let attrs
            if (lastInsertSuggestion && now - lastInsertSuggestion.at < 2000 && from === lastInsertSuggestion.to) {
              attrs = {
                id: lastInsertSuggestion.id,
                user: lastInsertSuggestion.user,
                createdAt: lastInsertSuggestion.createdAt,
              }
            } else {
              attrs = newSuggestionAttrs()
            }
            tr.addMark(from, to, type.create(attrs))
            lastInsertSuggestion = { ...attrs, at: now, to }
            changed = true
          })
        })
        if (!changed) return null
        tr.setMeta('suggestion', true)
        return tr
      },
    })]
  },
})

const BlockRange = NodeRange.extend({
  addKeyboardShortcuts() {
    return {}
  },
})

const CursorPresence = Extension.create({
  name: 'cursorPresence',
  addOptions() { return { awareness: null } },
  addProseMirrorPlugins() {
    if (!this.options.awareness) return []
    return [yCursorPlugin(this.options.awareness)]
  },
})

const CodeBlockLang = CodeBlock.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      language: {
        default: null,
        parseHTML: (el) => el.getAttribute('data-language') || '',
        renderHTML: (attrs) => (attrs.language ? { 'data-language': attrs.language } : {}),
      },
    }
  },
})

const MediaImage = Image.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      mediaId: {
        default: null,
        parseHTML: (el) => el.getAttribute('data-media-id'),
        renderHTML: (attrs) => (attrs.mediaId ? { 'data-media-id': attrs.mediaId } : {}),
      },
      caption: {
        default: '',
        parseHTML: (el) => el.getAttribute('data-caption') || '',
        renderHTML: (attrs) => (attrs.caption ? { 'data-caption': attrs.caption } : {}),
      },
      layout: {
        default: 'normal',
        parseHTML: (el) => el.getAttribute('data-layout') || 'normal',
        renderHTML: (attrs) => (attrs.layout && attrs.layout !== 'normal' ? { 'data-layout': attrs.layout } : {}),
      },
      width: {
        default: 100,
        parseHTML: (el) => Number(el.getAttribute('data-width')) || 100,
        renderHTML: (attrs) => (Number(attrs.width) && Number(attrs.width) !== 100 ? { 'data-width': Number(attrs.width) } : {}),
      },
      crop: {
        default: '',
        parseHTML: (el) => el.getAttribute('data-crop') || '',
        renderHTML: (attrs) => (attrs.crop ? { 'data-crop': attrs.crop } : {}),
      },
      credit: {
        default: '',
        parseHTML: (el) => el.getAttribute('data-credit') || '',
        renderHTML: (attrs) => (attrs.credit ? { 'data-credit': attrs.credit } : {}),
      },
      creditUrl: {
        default: '',
        parseHTML: (el) => el.getAttribute('data-credit-url') || '',
        renderHTML: (attrs) => (attrs.creditUrl ? { 'data-credit-url': attrs.creditUrl } : {}),
      },
      overlay: {
        default: '',
        parseHTML: (el) => el.getAttribute('data-overlay') || '',
        renderHTML: (attrs) => (attrs.overlay ? { 'data-overlay': attrs.overlay } : {}),
      },
      focalX: { default: 50 },
      focalY: { default: 50 },
    }
  },
  addNodeView() {
    return ({ node, editor, getPos }) => {
      const figure = document.createElement('figure')
      figure.className = 'be-image'
      const img = document.createElement('img')
      const cap = document.createElement('figcaption')
      const frame = document.createElement('div')
      frame.className = 'be-image-frame'
      const overlay = document.createElement('div')
      overlay.className = 'be-image-overlay'
      const handle = document.createElement('span')
      handle.className = 'be-image-resize'
      handle.title = 'Yeniden boyutlandır'
      frame.appendChild(img)
      frame.appendChild(overlay)
      figure.appendChild(frame)
      figure.appendChild(cap)
      figure.appendChild(handle)
      const sync = () => {
        img.src = node.attrs.src || ''
        img.alt = node.attrs.alt || ''
        img.title = node.attrs.title || ''
        const credit = node.attrs.credit ? ' — ' + node.attrs.credit : ''
        cap.textContent = (node.attrs.caption || '') + credit
        cap.hidden = !node.attrs.caption && !node.attrs.credit
        overlay.textContent = node.attrs.overlay || ''
        overlay.hidden = !node.attrs.overlay
        figure.dataset.layout = node.attrs.layout || 'normal'
        const layout = node.attrs.layout || 'normal'
        const width = Number(node.attrs.width)
        figure.style.width = layout === 'normal' && width && width !== 100 ? width + '%' : ''
        handle.hidden = layout !== 'normal'
        const crop = node.attrs.crop || ''
        if (crop) {
          img.style.aspectRatio = crop
          img.style.objectFit = 'cover'
          img.style.objectPosition = (Number(node.attrs.focalX) || 50) + '% ' + (Number(node.attrs.focalY) || 50) + '%'
        } else {
          img.style.aspectRatio = ''
          img.style.objectFit = ''
          img.style.objectPosition = ''
        }
      }
      sync()
      handle.addEventListener('pointerdown', (e) => {
        e.preventDefault()
        e.stopPropagation()
        const parent = figure.parentElement
        if (!parent || typeof getPos !== 'function') return
        const parentWidth = parent.getBoundingClientRect().width
        const startX = e.clientX
        const startWidth = figure.getBoundingClientRect().width
        let nextWidth = Number(node.attrs.width) || 100
        const move = (ev) => {
          nextWidth = Math.round(Math.min(100, Math.max(25, ((startWidth + ev.clientX - startX) / parentWidth) * 100)))
          figure.style.width = nextWidth >= 100 ? '' : nextWidth + '%'
        }
        const up = () => {
          document.removeEventListener('pointermove', move)
          document.removeEventListener('pointerup', up)
          const pos = getPos()
          if (pos != null) {
            editor.view.dispatch(editor.view.state.tr.setNodeMarkup(pos, undefined, {
              ...node.attrs, width: nextWidth,
            }))
          }
        }
        document.addEventListener('pointermove', move)
        document.addEventListener('pointerup', up)
      })
      return {
        dom: figure,
        update(updated) {
          if (updated.type.name !== 'image') return false
          node = updated
          sync()
          return true
        },
      }
    }
  },
})

const CALLOUT_LABELS = { info: 'Bilgi', warning: 'Uyarı', success: 'Başarı', danger: 'Dikkat' }

const Callout = Node.create({
  name: 'callout',
  group: 'block',
  content: 'block+',
  defining: true,
  addAttributes() {
    return { kind: { default: 'info' } }
  },
  parseHTML() {
    return [{ tag: 'aside[data-callout]' }]
  },
  renderHTML({ HTMLAttributes }) {
    return ['aside', mergeAttributes(HTMLAttributes, { 'data-callout': HTMLAttributes.kind || 'info' }), 0]
  },
  addNodeView() {
    return ({ node }) => {
      const dom = document.createElement('aside')
      const title = document.createElement('div')
      title.className = 'be-callout-title'
      const content = document.createElement('div')
      content.className = 'be-callout-body'
      dom.appendChild(title)
      dom.appendChild(content)
      const sync = () => {
        const kind = node.attrs.kind || 'info'
        dom.className = 'be-callout be-callout-' + kind
        title.textContent = CALLOUT_LABELS[kind] || 'Bilgi'
      }
      sync()
      return {
        dom,
        contentDOM: content,
        update(updated) {
          if (updated.type.name !== 'callout') return false
          node = updated
          sync()
          return true
        },
      }
    }
  },
})

function youtubeID(raw) {
  const m = String(raw || '').match(/(?:youtube\.com\/(?:watch\?v=|embed\/|shorts\/)|youtu\.be\/)([A-Za-z0-9_-]{6,})/)
  return m ? m[1] : ''
}

function providerForUrl(raw) {
  const url = String(raw || '').trim()
  if (youtubeID(url)) return 'youtube'
  if (/(?:twitter|x)\.com\/[^/]+\/status\/\d+/.test(url)) return 'x'
  return ''
}

const Embed = Node.create({
  name: 'embed',
  group: 'block',
  atom: true,
  draggable: true,
  addAttributes() {
    return {
      url: {
        default: '',
        parseHTML: (el) => el.getAttribute('data-url') || '',
        renderHTML: (attrs) => (attrs.url ? { 'data-url': attrs.url } : {}),
      },
      provider: {
        default: '',
        parseHTML: (el) => el.getAttribute('data-provider') || '',
        renderHTML: (attrs) => (attrs.provider ? { 'data-provider': attrs.provider } : {}),
      },
    }
  },
  parseHTML() {
    return [{ tag: 'div[data-embed]' }]
  },
  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-embed': HTMLAttributes.provider || 'embed' })]
  },
  addNodeView() {
    return ({ node }) => {
      const dom = document.createElement('div')
      dom.className = 'be-embed'
      const sync = () => {
        dom.innerHTML = ''
        const url = node.attrs.url || ''
        const provider = node.attrs.provider || providerForUrl(url)
        if (provider === 'youtube') {
          const id = youtubeID(url)
          if (!id) {
            dom.textContent = 'Geçersiz YouTube bağlantısı'
            return
          }
          const frame = document.createElement('iframe')
          frame.src = 'https://www.youtube-nocookie.com/embed/' + id
          frame.title = 'YouTube video'
          frame.allow = 'accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture'
          frame.allowFullscreen = true
          frame.setAttribute('loading', 'lazy')
          dom.appendChild(frame)
          return
        }
        const p = document.createElement('p')
        p.className = 'be-embed-link'
        p.textContent = 'Gömülü içerik: '
        const a = document.createElement('a')
        a.href = url
        a.target = '_blank'
        a.rel = 'noopener noreferrer'
        a.textContent = provider === 'x' ? 'X gönderisini aç' : url
        p.appendChild(a)
        dom.appendChild(p)
      }
      sync()
      return {
        dom,
        update(updated) {
          if (updated.type.name !== 'embed') return false
          node = updated
          sync()
          return true
        },
      }
    }
  },
})

const Toggle = Node.create({
  name: 'toggle',
  group: 'block',
  content: 'block+',
  defining: true,
  addAttributes() {
    return {
      title: { default: 'Başlık' },
      open: { default: true },
    }
  },
  parseHTML() {
    return [{ tag: 'details[data-toggle]' }]
  },
  renderHTML({ HTMLAttributes }) {
    return ['details', mergeAttributes(HTMLAttributes, { 'data-toggle': '' }), ['summary', {}, HTMLAttributes.title || 'Başlık'], ['div', {}, 0]]
  },
  addNodeView() {
    return ({ node, editor, getPos }) => {
      const dom = document.createElement('details')
      dom.className = 'be-toggle'
      const summary = document.createElement('summary')
      const input = document.createElement('input')
      input.className = 'be-toggle-title'
      input.type = 'text'
      input.value = node.attrs.title || 'Başlık'
      input.addEventListener('input', () => {
        const pos = typeof getPos === 'function' ? getPos() : null
        if (pos == null) return
        editor.view.dispatch(editor.view.state.tr.setNodeMarkup(pos, undefined, {
          ...node.attrs, title: input.value,
        }))
      })
      input.addEventListener('mousedown', (e) => e.stopPropagation())
      input.addEventListener('click', (e) => e.stopPropagation())
      summary.appendChild(input)
      const body = document.createElement('div')
      body.className = 'be-toggle-body'
      dom.appendChild(summary)
      dom.appendChild(body)
      dom.open = node.attrs.open !== false
      dom.addEventListener('toggle', () => {
        const pos = typeof getPos === 'function' ? getPos() : null
        if (pos == null) return
        editor.view.dispatch(editor.view.state.tr.setNodeMarkup(pos, undefined, {
          ...node.attrs, open: dom.open,
        }))
      })
      const sync = () => {
        input.value = node.attrs.title || 'Başlık'
        dom.open = node.attrs.open !== false
      }
      sync()
      return {
        dom,
        contentDOM: body,
        update(updated) {
          if (updated.type.name !== 'toggle') return false
          node = updated
          sync()
          return true
        },
      }
    }
  },
})

const Bookmark = Node.create({
  name: 'bookmark',
  group: 'block',
  atom: true,
  draggable: true,
  addAttributes() {
    return {
      url: { default: '' },
      title: { default: '' },
      description: { default: '' },
      site: { default: '' },
      image: { default: '' },
    }
  },
  parseHTML() {
    return [{
      tag: 'a[data-bookmark]',
      getAttrs: (el) => ({
        url: el.getAttribute('href') || '',
        title: el.getAttribute('data-title') || '',
        description: el.getAttribute('data-description') || '',
        site: el.getAttribute('data-site') || '',
        image: el.getAttribute('data-image') || '',
      }),
    }]
  },
  renderHTML({ node }) {
    const a = node.attrs
    const children = [
      ['div', { class: 'editor-bookmark-body' },
        ['strong', {}, a.title || a.url],
        ['p', {}, a.description || ''],
        ['span', {}, a.site || ''],
      ],
    ]
    if (a.image) children.unshift(['img', { src: a.image, alt: '', loading: 'lazy' }])
    return ['a', {
      href: a.url, 'data-bookmark': '', 'data-title': a.title, 'data-description': a.description,
      'data-site': a.site, 'data-image': a.image, class: 'editor-bookmark',
      target: '_blank', rel: 'noopener noreferrer',
    }, ...children]
  },
  addNodeView() {
    return ({ node }) => {
      const dom = document.createElement('a')
      dom.className = 'editor-bookmark'
      dom.target = '_blank'
      dom.rel = 'noopener noreferrer'
      const img = document.createElement('img')
      const body = document.createElement('div')
      body.className = 'editor-bookmark-body'
      const title = document.createElement('strong')
      const desc = document.createElement('p')
      const site = document.createElement('span')
      body.append(title, desc, site)
      dom.append(img, body)
      const sync = () => {
        const a = node.attrs
        dom.href = a.url || '#'
        dom.dataset.bookmark = ''
        img.src = a.image || ''
        img.hidden = !a.image
        title.textContent = a.title || a.url || ''
        desc.textContent = a.description || ''
        desc.hidden = !a.description
        site.textContent = a.site || ''
      }
      sync()
      dom.addEventListener('click', (e) => {
        e.preventDefault()
        if (node.attrs.url) window.open(node.attrs.url, '_blank', 'noopener')
      })
      return {
        dom,
        update(updated) {
          if (updated.type.name !== 'bookmark') return false
          node = updated
          sync()
          return true
        },
      }
    }
  },
})

const MediaCard = Node.create({
  name: 'mediaCard',
  group: 'block',
  atom: true,
  draggable: true,
  addAttributes() {
    return {
      kind: { default: 'file' },
      url: { default: '' },
      title: { default: '' },
      filename: { default: '' },
    }
  },
  parseHTML() {
    return [{
      tag: 'div[data-media-card]',
      getAttrs: (el) => ({
        kind: el.getAttribute('data-kind') || 'file',
        url: el.getAttribute('data-url') || '',
        title: el.getAttribute('data-title') || '',
        filename: el.getAttribute('data-filename') || '',
      }),
    }]
  },
  renderHTML({ node }) {
    return ['div', {
      'data-media-card': '',
      'data-kind': node.attrs.kind || 'file',
      'data-url': node.attrs.url || '',
      'data-title': node.attrs.title || '',
      'data-filename': node.attrs.filename || '',
    }, node.attrs.title || node.attrs.filename || node.attrs.url || 'Medya']
  },
  addNodeView() {
    return ({ node }) => {
      const dom = document.createElement('div')
      const sync = () => {
        const a = node.attrs
        const kind = ['audio', 'video'].includes(a.kind) ? a.kind : 'file'
        dom.className = 'be-media-card be-media-' + kind
        dom.innerHTML = ''
        const title = document.createElement('strong')
        title.textContent = a.title || a.filename || a.url || 'Medya'
        if (kind === 'audio') {
          const audio = document.createElement('audio')
          audio.controls = true
          audio.preload = 'metadata'
          audio.src = a.url || ''
          dom.append(title, audio)
          return
        }
        if (kind === 'video') {
          const video = document.createElement('video')
          video.controls = true
          video.preload = 'metadata'
          video.playsInline = true
          video.src = a.url || ''
          dom.append(title, video)
          return
        }
        const link = document.createElement('a')
        link.className = 'be-media-file-link'
        link.href = a.url || '#'
        link.target = '_blank'
        link.rel = 'noopener noreferrer'
        if (a.filename) link.download = a.filename
        const icon = document.createElement('span')
        icon.className = 'be-media-file-icon'
        icon.textContent = '📄'
        const body = document.createElement('span')
        body.className = 'be-media-file-body'
        const name = document.createElement('strong')
        name.textContent = a.title || a.filename || a.url || 'Dosya'
        const meta = document.createElement('small')
        meta.textContent = a.filename || a.url || ''
        body.append(name, meta)
        link.append(icon, body)
        dom.append(link)
      }
      sync()
      return {
        dom,
        update(updated) {
          if (updated.type.name !== 'mediaCard') return false
          node = updated
          sync()
          return true
        },
        stopEvent: (event) => !!event.target.closest('audio,video,a'),
        ignoreMutation: () => true,
      }
    }
  },
})

const UploadPlaceholder = Node.create({
  name: 'uploadPlaceholder',
  group: 'block',
  atom: true,
  selectable: true,
  addAttributes() {
    return {
      id: { default: '' },
      filename: { default: '' },
      status: { default: 'loading' },
      message: { default: 'Yükleniyor…' },
    }
  },
  parseHTML() {
    return [{ tag: 'div[data-upload-placeholder]' }]
  },
  renderHTML({ node }) {
    return ['div', {
      'data-upload-placeholder': '',
      'data-status': node.attrs.status || 'loading',
      'data-filename': node.attrs.filename || '',
    }, node.attrs.message || 'Yükleniyor…']
  },
  addNodeView() {
    return ({ node, editor, getPos }) => {
      const dom = document.createElement('div')
      const sync = () => {
        const error = node.attrs.status === 'error'
        dom.className = 'be-upload-card' + (error ? ' is-error' : '')
        dom.innerHTML = ''
        const icon = document.createElement('span')
        icon.className = 'be-upload-icon'
        icon.textContent = error ? '⚠' : '⏳'
        const body = document.createElement('span')
        body.className = 'be-upload-body'
        const title = document.createElement('strong')
        title.textContent = node.attrs.filename || 'Medya'
        const message = document.createElement('small')
        message.textContent = node.attrs.message || (error ? 'Yükleme başarısız.' : 'Yükleniyor…')
        body.append(title, message)
        dom.append(icon, body)
        if (error) {
          const remove = document.createElement('button')
          remove.type = 'button'
          remove.className = 'btn ghost small'
          remove.textContent = 'Kaldır'
          remove.onclick = () => {
            const pos = typeof getPos === 'function' ? getPos() : null
            if (pos != null) editor.chain().focus().deleteRange({ from: pos, to: pos + node.nodeSize }).run()
          }
          dom.appendChild(remove)
        }
      }
      sync()
      return {
        dom,
        update(updated) {
          if (updated.type.name !== 'uploadPlaceholder') return false
          node = updated
          sync()
          return true
        },
        stopEvent: (event) => !!event.target.closest('button'),
        ignoreMutation: () => true,
      }
    }
  },
})

const Gallery = Node.create({
  name: 'gallery',
  group: 'block',
  atom: true,
  draggable: true,
  addAttributes() {
    return {
      items: {
        default: [],
        parseHTML: (el) => {
          try { return JSON.parse(el.getAttribute('data-items') || '[]') } catch (_) { return [] }
        },
        renderHTML: (attrs) => (attrs.items && attrs.items.length ? { 'data-items': JSON.stringify(attrs.items) } : {}),
      },
    }
  },
  parseHTML() {
    return [{ tag: 'div[data-gallery]' }]
  },
  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-gallery': '' })]
  },
  addNodeView() {
    return ({ node }) => {
      const dom = document.createElement('div')
      dom.className = 'be-gallery'
      const sync = () => {
        dom.innerHTML = ''
        const items = node.attrs.items || []
        if (!items.length) {
          dom.textContent = 'Galeri boş'
          return
        }
        items.forEach((item) => {
          const figure = document.createElement('figure')
          const img = document.createElement('img')
          img.src = item.src || ''
          img.alt = item.alt || ''
          img.loading = 'lazy'
          figure.appendChild(img)
          if (item.caption) {
            const cap = document.createElement('figcaption')
            cap.textContent = item.caption
            figure.appendChild(cap)
          }
          dom.appendChild(figure)
        })
      }
      sync()
      return {
        dom,
        update(updated) {
          if (updated.type.name !== 'gallery') return false
          node = updated
          sync()
          return true
        },
      }
    }
  },
})

let instance = null
let bubbleElement = null
let editorES = null
let editorLiveID = null
let collabProvider = null

const SLASH_ITEMS = [
  { title: 'Başlık 2', desc: 'Büyük bölüm başlığı', keywords: 'h2 başlık heading', action: (ed) => ed.chain().focus().toggleHeading({ level: 2 }).run() },
  { title: 'Başlık 3', desc: 'Alt başlık', keywords: 'h3 alt başlık heading', action: (ed) => ed.chain().focus().toggleHeading({ level: 3 }).run() },
  { title: 'Madde listesi', desc: 'İşaretli liste', keywords: 'liste bullet ul', action: (ed) => ed.chain().focus().toggleBulletList().run() },
  { title: 'Numaralı liste', desc: 'Sıralı liste', keywords: 'liste ordered ol', action: (ed) => ed.chain().focus().toggleOrderedList().run() },
  { title: 'Alıntı', desc: 'Alıntı bloğu', keywords: 'quote blockquote', action: (ed) => ed.chain().focus().toggleBlockquote().run() },
  { title: 'Kod bloğu', desc: 'Kod parçası', keywords: 'code pre', action: insertCodeBlock },
  { title: 'Tablo', desc: '3x3 tablo ekle', keywords: 'table tablo', action: (ed) => ed.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run() },
  { title: 'Callout', desc: 'Bilgi/uyarı kutusu', keywords: 'callout uyarı bilgi', action: insertCallout },
  { title: 'Toggle', desc: 'Açılır/kapanır bölüm', keywords: 'toggle açılır kapanır', action: insertToggle },
  { title: 'İç link', desc: 'Yayındaki bir habere bağlantı', keywords: 'link iç bağlantı haber', action: setInternalLinkFromDialog },
  { title: 'Editör notu', desc: 'Seçili metne not ekle', keywords: 'yorum not comment', action: openComments },
  { title: 'Görsel', desc: 'Medya kütüphanesinden görsel', keywords: 'image resim görsel', action: setImageFromDialog },
  { title: 'Galeri', desc: 'Çoklu görsel', keywords: 'gallery galeri', action: setGalleryFromDialog },
  { title: 'Embed', desc: 'YouTube / X bağlantısı', keywords: 'video youtube embed x twitter', action: promptEmbed },
  { title: 'Yer imi', desc: 'Bağlantı kartı', keywords: 'bookmark yer imi link kart', action: openBookmarkDialog },
  { title: 'Uzak görsel', desc: 'URL veya Unsplash görseli', keywords: 'görsel resim image unsplash uzak url', action: openRemoteImageDialog },
  { title: 'Dosya', desc: 'Dosya indirme kartı', keywords: 'dosya file pdf zip', action: (ed) => openMediaCardDialog(ed, 'file') },
  { title: 'Ses', desc: 'Ses oynatıcı kartı', keywords: 'ses audio müzik mp3', action: (ed) => openMediaCardDialog(ed, 'audio') },
  { title: 'Video', desc: 'Video oynatıcı kartı', keywords: 'video mp4 webm', action: (ed) => openMediaCardDialog(ed, 'video') },
  { title: 'Yatay çizgi', desc: 'Ayırıcı', keywords: 'hr çizgi', action: (ed) => ed.chain().focus().setHorizontalRule().run() },
]

let slashEl = null
let slashProps = null
let slashIndex = 0
let slashUnmount = null

function slashChoose(i) {
  if (!slashProps || !slashProps.items || !slashProps.items[i]) return
  slashProps.command(slashProps.items[i])
}

function renderSlash() {
  if (!slashEl || !slashProps) return
  slashEl.innerHTML = ''
  const items = slashProps.items || []
  if (!items.length) {
    slashEl.hidden = true
    return
  }
  slashEl.hidden = false
  if (slashIndex >= items.length) slashIndex = 0
  items.forEach((item, i) => {
    const b = document.createElement('button')
    b.type = 'button'
    b.className = 'be-slash-item' + (i === slashIndex ? ' is-active' : '')
    b.innerHTML = '<strong></strong><span></span>'
    b.querySelector('strong').textContent = item.title
    b.querySelector('span').textContent = item.desc || ''
    b.addEventListener('mousedown', (e) => {
      e.preventDefault()
      slashChoose(i)
    })
    slashEl.appendChild(b)
  })
}

const slashRenderer = {
  onStart(props) {
    slashProps = props
    slashIndex = 0
    slashEl = document.createElement('div')
    slashEl.className = 'be-slash'
    slashUnmount = props.mount ? props.mount(slashEl) : null
    if (!slashUnmount) document.body.appendChild(slashEl)
    renderSlash()
  },
  onUpdate(props) {
    slashProps = props
    slashIndex = 0
    renderSlash()
  },
  onKeyDown(props) {
    const items = (props.items || [])
    if (props.event.key === 'Escape') {
      slashRenderer.onExit()
      return true
    }
    if (props.event.key === 'ArrowDown') {
      slashIndex = (slashIndex + 1) % Math.max(items.length, 1)
      renderSlash()
      return true
    }
    if (props.event.key === 'ArrowUp') {
      slashIndex = (slashIndex - 1 + items.length) % Math.max(items.length, 1)
      renderSlash()
      return true
    }
    if (props.event.key === 'Enter') {
      slashChoose(slashIndex)
      return true
    }
    return false
  },
  onExit() {
    if (slashUnmount) slashUnmount()
    if (slashEl) slashEl.remove()
    slashEl = null
    slashProps = null
    slashUnmount = null
  },
}

const SlashCommand = Extension.create({
  name: 'slashCommand',
  addProseMirrorPlugins() {
    return [Suggestion({
      editor: this.editor,
      char: '/',
      allowSpaces: false,
      startOfLine: false,
      items: ({ query }) => {
        const q = (query || '').toLowerCase()
        return SLASH_ITEMS.filter((i) => (i.title + ' ' + i.keywords).toLowerCase().includes(q)).slice(0, 10)
      },
      command: ({ editor, range, props }) => {
        editor.chain().focus().deleteRange(range).run()
        props.action(editor)
      },
      render: () => slashRenderer,
    })]
  },
})

function mediaOptions() {
  const source = document.getElementById('editorMediaSelect')
  return source ? [...source.options].filter((o) => o.value).map((o) => ({
    id: o.value,
    url: o.getAttribute('data-url') || '',
    thumb: o.getAttribute('data-thumb') || '',
    alt: o.getAttribute('data-alt') || '',
    caption: o.getAttribute('data-caption') || '',
    title: o.getAttribute('data-title') || '',
    description: o.getAttribute('data-description') || '',
    seoTitle: o.getAttribute('data-seo-title') || '',
    seoDescription: o.getAttribute('data-seo-description') || '',
    category: o.getAttribute('data-category') || '',
    label: o.textContent || o.value,
  })) : []
}

function closeDialog(id) {
  const d = document.getElementById(id)
  if (d) d.hidden = true
}

function suggestionRanges(editor, name, id) {
  const type = editor.schema.marks[name]
  if (!type) return []
  const ranges = []
  editor.state.doc.descendants((node, pos) => {
    if (!node.isText) return
    const match = node.marks.some((mark) => mark.type === type && (!id || mark.attrs.id === id))
    if (!match) return
    const from = pos
    const to = pos + node.nodeSize
    const last = ranges[ranges.length - 1]
    if (last && last.to === from) last.to = to
    else ranges.push({ from, to })
  })
  return ranges
}

function trackDeletion(editor, from, to) {
  if (from >= to) return false
  const { doc, tr } = editor.state
  const insertType = editor.schema.marks.suggestionInsert
  const deleteType = editor.schema.marks.suggestionDelete
  if (!deleteType) return false
  const markRanges = []
  const deleteRanges = []
  doc.nodesBetween(from, to, (node, pos) => {
    if (!node.isText) return
    const start = Math.max(from, pos)
    const end = Math.min(to, pos + node.nodeSize)
    if (start >= end) return
    if (insertType && node.marks.some((mark) => mark.type === insertType)) {
      deleteRanges.push({ from: start, to: end })
    } else {
      markRanges.push({ from: start, to: end })
    }
  })
  if (!markRanges.length && !deleteRanges.length) return false
  const attrs = newSuggestionAttrs()
  for (const range of markRanges) tr.addMark(range.from, range.to, deleteType.create(attrs))
  deleteRanges.sort((a, b) => b.from - a.from).forEach((range) => tr.delete(range.from, range.to))
  if (!tr.steps.length) return false
  tr.setMeta('suggestion', true)
  editor.view.dispatch(tr)
  editor.commands.focus()
  return true
}

function getSuggestionAtCursor(editor) {
  const { selection, doc } = editor.state
  const marks = []
  if (selection.empty) {
    const $from = selection.$from
    if ($from.nodeBefore) marks.push(...$from.nodeBefore.marks)
    if ($from.nodeAfter) marks.push(...$from.nodeAfter.marks)
    marks.push(...$from.marks())
  } else {
    doc.nodesBetween(selection.from, selection.to, (node) => {
      if (node.isText) marks.push(...node.marks)
    })
  }
  for (const name of ['suggestionInsert', 'suggestionDelete']) {
    const mark = marks.find((m) => m.type.name === name)
    if (mark) return { kind: name, id: mark.attrs.id || '' }
  }
  return null
}

function applySuggestion(editor, accept, target) {
  const actions = target ? [target] : [
    { kind: 'suggestionInsert', id: '' },
    { kind: 'suggestionDelete', id: '' },
  ]
  const tr = editor.state.tr
  const removeActions = []
  const deleteRanges = []
  for (const action of actions) {
    const ranges = suggestionRanges(editor, action.kind, action.id)
    if (!ranges.length) continue
    const shouldRemoveMark = action.kind === 'suggestionInsert' ? accept : !accept
    if (shouldRemoveMark) {
      removeActions.push({ ranges, type: editor.schema.marks[action.kind] })
    } else {
      deleteRanges.push(...ranges)
    }
  }
  for (const action of removeActions) {
    if (!action.type) continue
    for (const range of action.ranges) tr.removeMark(range.from, range.to, action.type)
  }
  deleteRanges.sort((a, b) => b.from - a.from).forEach((range) => tr.delete(range.from, range.to))
  if (tr.steps.length) {
    tr.setMeta('suggestion', true)
    editor.view.dispatch(tr)
    editor.commands.focus()
    lastInsertSuggestion = null
  }
  updateSuggestionUI(editor)
}

function suggestionGroups(editor) {
  const groups = new Map()
  editor.state.doc.descendants((node, pos) => {
    if (!node.isText) return
    for (const mark of node.marks) {
      const kind = mark.type.name
      if (kind !== 'suggestionInsert' && kind !== 'suggestionDelete') continue
      const id = mark.attrs.id || ''
      const key = kind + ':' + id
      let group = groups.get(key)
      if (!group) {
        group = {
          kind,
          id,
          user: mark.attrs.user || '',
          createdAt: Number(mark.attrs.createdAt) || 0,
          text: '',
          from: pos,
          to: pos,
        }
        groups.set(key, group)
      }
      group.text += node.text
      group.to = pos + node.nodeSize
    }
  })
  return [...groups.values()].sort((a, b) => (a.createdAt || 0) - (b.createdAt || 0))
}

function openSuggestionList(editor) {
  const dialog = document.getElementById('editorSuggestionDialog')
  const list = document.getElementById('editorSuggestionList')
  if (!dialog || !list) return
  const close = () => { dialog.hidden = true }
  document.getElementById('editorSuggestionClose').onclick = close
  const render = () => {
    const groups = suggestionGroups(editor)
    list.innerHTML = ''
    if (!groups.length) {
      const empty = document.createElement('p')
      empty.className = 'be-suggestion-empty'
      empty.textContent = 'Bekleyen öneri yok.'
      list.appendChild(empty)
      return
    }
    groups.forEach((group) => {
      const row = document.createElement('div')
      row.className = 'be-suggestion-row'
      const head = document.createElement('div')
      head.className = 'be-suggestion-head'
      const kind = document.createElement('span')
      kind.className = 'be-suggestion-kind'
      kind.textContent = group.kind === 'suggestionInsert' ? 'Ekleme' : 'Silme'
      const meta = document.createElement('span')
      const when = group.createdAt ? new Date(group.createdAt).toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit' }) : ''
      meta.textContent = [group.user || 'Editör', when].filter(Boolean).join(' · ')
      head.append(kind, meta)
      const excerpt = document.createElement('p')
      excerpt.className = 'be-suggestion-excerpt'
      const text = group.text.replace(/\s+/g, ' ').trim()
      excerpt.textContent = text.length > 160 ? text.slice(0, 157) + '…' : text
      const actions = document.createElement('div')
      actions.className = 'be-suggestion-row-actions'
      const go = document.createElement('button')
      go.type = 'button'
      go.className = 'btn ghost small'
      go.textContent = 'Git'
      go.onclick = () => {
        close()
        editor.chain().focus().setTextSelection({ from: group.from, to: group.to }).scrollIntoView().run()
      }
      const accept = document.createElement('button')
      accept.type = 'button'
      accept.className = 'btn ghost small'
      accept.textContent = 'Kabul'
      accept.onclick = () => {
        applySuggestion(editor, true, { kind: group.kind, id: group.id })
        render()
      }
      const reject = document.createElement('button')
      reject.type = 'button'
      reject.className = 'btn ghost small'
      reject.textContent = 'Reddet'
      reject.onclick = () => {
        applySuggestion(editor, false, { kind: group.kind, id: group.id })
        render()
      }
      actions.append(go, accept, reject)
      row.append(head, excerpt, actions)
      list.appendChild(row)
    })
  }
  render()
  dialog.hidden = false
}

function updateSuggestionUI(editor) {
  const bar = document.getElementById('suggestionBar')
  const count = document.getElementById('suggestionCount')
  const toggle = document.querySelectorAll('.be-toolbar [data-cmd="suggestion"]')
  toggle.forEach((btn) => btn.classList.toggle('is-active', suggestionMode))
  document.body.classList.toggle('be-suggestion-mode', suggestionMode)
  if (bar) bar.hidden = !suggestionMode
  if (!bar || !editor) return
  const insertCount = suggestionRanges(editor, 'suggestionInsert').reduce((n, r) => n + r.to - r.from, 0)
  const deleteCount = suggestionRanges(editor, 'suggestionDelete').reduce((n, r) => n + r.to - r.from, 0)
  if (count) count.textContent = insertCount + ' ekleme · ' + deleteCount + ' silme'
  const current = getSuggestionAtCursor(editor)
  for (const id of ['suggestionAccept', 'suggestionReject']) {
    const btn = document.getElementById(id)
    if (btn) btn.disabled = !current
  }
}

function toggleSuggestionMode(editor) {
  suggestionMode = !suggestionMode
  try { localStorage.setItem('be-suggestion-mode', suggestionMode ? '1' : '0') } catch (_) { /* yok say */ }
  updateSuggestionUI(editor)
  editor.commands.focus()
}

function selectBlocks(editor) {
  const { state, view } = editor
  const { from, to } = state.selection
  try {
    const selection = NodeRangeSelection.create(state.doc, from, to, 1)
    view.dispatch(state.tr.setSelection(selection))
    editor.commands.focus()
  } catch (_) {
    editor.commands.selectAll()
  }
}

function bindSuggestionBar(editor) {
  const accept = document.getElementById('suggestionAccept')
  const reject = document.getElementById('suggestionReject')
  const acceptAll = document.getElementById('suggestionAcceptAll')
  const rejectAll = document.getElementById('suggestionRejectAll')
  const list = document.getElementById('suggestionList')
  if (list) list.onclick = () => openSuggestionList(editor)
  if (accept) accept.onclick = () => {
    const current = getSuggestionAtCursor(editor)
    if (current) applySuggestion(editor, true, current)
  }
  if (reject) reject.onclick = () => {
    const current = getSuggestionAtCursor(editor)
    if (current) applySuggestion(editor, false, current)
  }
  if (acceptAll) acceptAll.onclick = () => applySuggestion(editor, true, null)
  if (rejectAll) rejectAll.onclick = () => applySuggestion(editor, false, null)
}

function openRemoteImageDialog(editor) {
  const dialog = document.getElementById('editorRemoteImageDialog')
  if (!dialog) return
  const url = document.getElementById('editorRemoteImageUrl')
  const alt = document.getElementById('editorRemoteImageAlt')
  const caption = document.getElementById('editorRemoteImageCaption')
  const credit = document.getElementById('editorRemoteImageCredit')
  const creditUrl = document.getElementById('editorRemoteImageCreditUrl')
  const overlay = document.getElementById('editorRemoteImageOverlay')
  const query = document.getElementById('editorUnsplashQuery')
  const status = document.getElementById('editorUnsplashStatus')
  const results = document.getElementById('editorUnsplashResults')
  url.value = ''
  alt.value = ''
  caption.value = ''
  credit.value = ''
  creditUrl.value = ''
  overlay.value = ''
  query.value = ''
  status.textContent = ''
  results.innerHTML = ''
  dialog.hidden = false
  url.focus()
  document.getElementById('editorRemoteImageCancel').onclick = () => closeDialog('editorRemoteImageDialog')
  document.getElementById('editorRemoteImageConfirm').onclick = () => {
    const safeUrl = safeMediaCardUrl(url.value)
    if (!safeUrl) {
      status.textContent = 'Geçerli bir http(s) adresi veya /media/… yolu girin.'
      return
    }
    editor.chain().focus().insertContent({
      type: 'image',
      attrs: {
        mediaId: 0,
        src: safeUrl,
        alt: alt.value.trim(),
        title: alt.value.trim(),
        caption: caption.value.trim(),
        credit: credit.value.trim(),
        creditUrl: safeMediaCardUrl(creditUrl.value),
        overlay: overlay.value.trim(),
        layout: 'normal',
        width: 100,
        crop: '',
        focalX: 50,
        focalY: 50,
      },
    }).run()
    closeDialog('editorRemoteImageDialog')
  }
  const runSearch = async () => {
    const q = query.value.trim()
    if (!q) {
      status.textContent = 'Arama için bir kelime yazın.'
      return
    }
    const button = document.getElementById('editorUnsplashSearch')
    button.disabled = true
    status.textContent = 'Unsplash’ta aranıyor…'
    results.innerHTML = ''
    try {
      const res = await fetch('/admin/medya/unsplash?q=' + encodeURIComponent(q), {
        credentials: 'same-origin',
        headers: { 'X-Requested-With': 'fetch' },
      })
      const data = await res.json().catch(() => null)
      if (!res.ok || !data || !data.ok) throw new Error((data && data.error) || 'Unsplash araması başarısız.')
      const items = Array.isArray(data.results) ? data.results : []
      if (!items.length) {
        status.textContent = 'Sonuç bulunamadı.'
        return
      }
      status.textContent = items.length + ' sonuç'
      items.forEach((item) => {
        const btn = document.createElement('button')
        btn.type = 'button'
        btn.className = 'be-unsplash-item'
        btn.title = item.alt || ''
        const img = document.createElement('img')
        img.src = item.thumb || item.url
        img.alt = item.alt || ''
        img.loading = 'lazy'
        const label = document.createElement('small')
        label.textContent = item.credit || 'Unsplash'
        btn.append(img, label)
        btn.onclick = () => {
          url.value = item.url || ''
          alt.value = item.alt || ''
          credit.value = item.credit ? 'Fotoğraf: ' + item.credit + ' / Unsplash' : 'Fotoğraf: Unsplash'
          creditUrl.value = item.creditUrl || ''
          status.textContent = 'Görsel seçildi.'
        }
        results.appendChild(btn)
      })
    } catch (err) {
      status.textContent = err.message || 'Unsplash araması başarısız.'
    } finally {
      button.disabled = false
    }
  }
  // `runSearch` async: olay dinleyicisine DOĞRUDAN bağlanınca dönen söz
  // hiçbir yerde yakalanmıyor (Sonar S9383). try/catch gövdenin içinde olsa
  // da blok DIŞINDA atılan bir hata sessizce kaybolurdu; sarmalayıcı bunu
  // kullanıcıya bildirir.
  const aramaBaslat = () => {
    runSearch().catch((err) => {
      status.textContent = err.message || 'Unsplash araması başarısız.'
    })
  }
  document.getElementById('editorUnsplashSearch').onclick = aramaBaslat
  query.onkeydown = (event) => {
    if (event.key === 'Enter') {
      event.preventDefault()
      aramaBaslat()
    }
  }
}

function setImageFromDialog(editor) {
  const dialog = document.getElementById('editorImageDialog')
  const select = document.getElementById('editorImageSelect')
  if (!dialog || !select) return
  const opts = mediaOptions()
  if (!opts.length) {
    alert('Medya kütüphanesinde görsel yok. Önce Medya Yöneticisi’nden görsel yükleyin.')
    return
  }
  select.innerHTML = ''
  opts.forEach((o) => {
    const opt = document.createElement('option')
    opt.value = o.id
    opt.textContent = o.label
    opt.dataset.url = o.url
    opt.dataset.alt = o.alt
    opt.dataset.caption = o.caption
    select.appendChild(opt)
  })
  document.getElementById('editorImageAlt').value = ''
  document.getElementById('editorImageCaption').value = ''
  document.getElementById('editorImageWidth').value = '100'
  document.getElementById('editorImageCrop').value = ''
  document.getElementById('editorImageCropW').value = '3'
  document.getElementById('editorImageCropH').value = '2'
  document.getElementById('editorImageOverlay').value = ''
  document.getElementById('editorImageCredit').value = ''
  document.getElementById('editorImageFocalX').value = '50'
  document.getElementById('editorImageFocalY').value = '50'
  const preview = document.getElementById('editorImagePreview')
  const grid = document.getElementById('editorImageGrid')
  const choose = (index) => {
    const opt = select.options[index]
    if (!opt) return
    select.selectedIndex = index
    grid.querySelectorAll('.be-media-choice').forEach((el) => el.classList.toggle('is-selected', Number(el.dataset.index) === index))
    preview.src = opt.dataset.url || opt.dataset.thumb || ''
    preview.alt = opt.dataset.alt || opt.dataset.title || opt.textContent || ''
    preview.hidden = !preview.src
    document.getElementById('editorImageAlt').value = opt.dataset.alt || ''
    document.getElementById('editorImageCaption').value = opt.dataset.caption || ''
  }
  grid.innerHTML = ''
  opts.forEach((o, index) => {
    const choice = document.createElement('button')
    choice.type = 'button'
    choice.className = 'be-media-choice'
    choice.dataset.index = String(index)
    choice.title = o.title || o.label
    const img = document.createElement('img')
    img.src = o.thumb || o.url
    img.alt = o.alt || o.title || o.label
    img.loading = 'lazy'
    const label = document.createElement('span')
    label.textContent = o.title || o.label
    choice.append(img, label)
    choice.onclick = () => choose(index)
    grid.appendChild(choice)
  })
  const cropSelect = document.getElementById('editorImageCrop')
  const cropCustom = document.getElementById('editorImageCropCustom')
  const syncCrop = () => { cropCustom.hidden = cropSelect.value !== 'custom' }
  cropSelect.onchange = syncCrop
  syncCrop()
  dialog.hidden = false
  choose(0)
  const selectedCrop = () => {
    if (cropSelect.value !== 'custom') return cropSelect.value
    const w = Math.max(1, Math.min(100, Number(document.getElementById('editorImageCropW').value) || 3))
    const h = Math.max(1, Math.min(100, Number(document.getElementById('editorImageCropH').value) || 2))
    return w + '/' + h
  }
  document.getElementById('editorImageConfirm').onclick = () => {
    const opt = select.options[select.selectedIndex]
    if (!opt) return closeDialog('editorImageDialog')
    editor.chain().focus().insertContent({
      type: 'image',
      attrs: {
        src: opt.dataset.url || '', mediaId: Number(opt.value),
        alt: document.getElementById('editorImageAlt').value.trim() || opt.dataset.alt || '',
        title: opt.dataset.title || '',
        caption: document.getElementById('editorImageCaption').value.trim() || opt.dataset.caption || '',
        layout: document.getElementById('editorImageLayout').value || 'normal',
        width: Number(document.getElementById('editorImageWidth').value) || 100,
        crop: selectedCrop(),
        credit: document.getElementById('editorImageCredit').value.trim(),
        overlay: document.getElementById('editorImageOverlay').value.trim(),
        focalX: Number(document.getElementById('editorImageFocalX').value) || 50,
        focalY: Number(document.getElementById('editorImageFocalY').value) || 50,
      },
    }).run()
    closeDialog('editorImageDialog')
  }
  document.getElementById('editorImageCancel').onclick = () => closeDialog('editorImageDialog')
}

function setGalleryFromDialog(editor) {
  const dialog = document.getElementById('editorGalleryDialog')
  const select = document.getElementById('editorGallerySelect')
  if (!dialog || !select) return
  const opts = mediaOptions()
  if (!opts.length) {
    alert('Medya kütüphanesinde görsel yok.')
    return
  }
  select.innerHTML = ''
  opts.forEach((o) => {
    const opt = document.createElement('option')
    opt.value = o.id
    opt.textContent = o.label
    opt.dataset.url = o.url
    opt.dataset.alt = o.alt
    opt.dataset.caption = o.caption
    select.appendChild(opt)
  })
  const grid = document.getElementById('editorGalleryGrid')
  grid.innerHTML = ''
  opts.forEach((o, index) => {
    const choice = document.createElement('button')
    choice.type = 'button'
    choice.className = 'be-media-choice'
    choice.title = o.title || o.label
    const img = document.createElement('img')
    img.src = o.thumb || o.url
    img.alt = o.alt || o.title || o.label
    img.loading = 'lazy'
    const label = document.createElement('span')
    label.textContent = o.title || o.label
    choice.append(img, label)
    choice.onclick = () => {
      const opt = select.options[index]
      opt.selected = !opt.selected
      choice.classList.toggle('is-selected', opt.selected)
    }
    grid.appendChild(choice)
  })
  dialog.hidden = false
  document.getElementById('editorGalleryConfirm').onclick = () => {
    const items = [...select.selectedOptions].map((opt) => ({
      mediaId: Number(opt.value),
      src: opt.dataset.url || '',
      alt: opt.dataset.alt || '',
      caption: opt.dataset.caption || '',
    }))
    if (!items.length) return closeDialog('editorGalleryDialog')
    editor.chain().focus().insertContent({ type: 'gallery', attrs: { items } }).run()
    closeDialog('editorGalleryDialog')
  }
  document.getElementById('editorGalleryCancel').onclick = () => closeDialog('editorGalleryDialog')
}

function safeMediaCardUrl(raw) {
  const value = String(raw || '').trim()
  if (!value || value.startsWith('//')) return ''
  if (value.startsWith('/')) return value
  try {
    const parsed = new URL(value)
    if (parsed.protocol === 'http:' || parsed.protocol === 'https:') return parsed.href
  } catch (_) { /* geçersiz URL */ }
  return ''
}

function openMediaCardDialog(editor, kind = 'file') {
  const dialog = document.getElementById('editorMediaCardDialog')
  if (!dialog) return
  const kindSelect = document.getElementById('editorMediaCardKind')
  const url = document.getElementById('editorMediaCardUrl')
  const title = document.getElementById('editorMediaCardTitle')
  const filename = document.getElementById('editorMediaCardFilename')
  const status = document.getElementById('editorMediaCardStatus')
  kindSelect.value = ['file', 'audio', 'video'].includes(kind) ? kind : 'file'
  url.value = ''
  title.value = ''
  filename.value = ''
  status.textContent = ''
  dialog.hidden = false
  url.focus()
  document.getElementById('editorMediaCardCancel').onclick = () => closeDialog('editorMediaCardDialog')
  document.getElementById('editorMediaCardConfirm').onclick = () => {
    const safeUrl = safeMediaCardUrl(url.value)
    if (!safeUrl) {
      status.textContent = 'Geçerli bir http(s) adresi veya /media/… yolu girin.'
      return
    }
    let fallbackName = ''
    try { fallbackName = new URL(safeUrl, location.href).pathname.split('/').pop() || '' } catch (_) { /* yok */ }
    const selectedKind = ['audio', 'video'].includes(kindSelect.value) ? kindSelect.value : 'file'
    const name = filename.value.trim() || fallbackName
    editor.chain().focus().insertContent({
      type: 'mediaCard',
      attrs: {
        kind: selectedKind,
        url: safeUrl,
        title: title.value.trim() || name || safeUrl,
        filename: name,
      },
    }).run()
    closeDialog('editorMediaCardDialog')
  }
}

function openBookmarkDialog(editor) {
  const dialog = document.getElementById('editorBookmarkDialog')
  if (!dialog) return
  const urlInput = document.getElementById('editorBookmarkUrl')
  const titleInput = document.getElementById('editorBookmarkTitle')
  const descInput = document.getElementById('editorBookmarkDescription')
  const siteInput = document.getElementById('editorBookmarkSite')
  const imageInput = document.getElementById('editorBookmarkImage')
  const status = document.getElementById('editorBookmarkStatus')
  const fetchBtn = document.getElementById('editorBookmarkFetch')
  const confirmBtn = document.getElementById('editorBookmarkConfirm')
  const cancelBtn = document.getElementById('editorBookmarkCancel')
  const sel = editor.state.selection
  const selected = editor.state.doc.textBetween(sel.from, sel.to, ' ').trim()
  urlInput.value = /^https?:\/\//i.test(selected) ? selected : ''
  titleInput.value = ''
  descInput.value = ''
  siteInput.value = ''
  imageInput.value = ''
  status.textContent = ''
  dialog.hidden = false
  urlInput.focus()
  const fill = (b) => {
    titleInput.value = b.title || ''
    descInput.value = b.description || ''
    siteInput.value = b.site || ''
    imageInput.value = b.image || ''
  }
  const fetchMeta = async () => {
    let url = urlInput.value.trim()
    if (!url) return
    if (!/^https?:\/\//i.test(url)) {
      url = 'https://' + url
      urlInput.value = url
    }
    status.textContent = 'Sayfa bilgileri getiriliyor…'
    try {
      const res = await fetch('/admin/icerikler/bookmark?url=' + encodeURIComponent(url), { credentials: 'same-origin' })
      const data = await res.json()
      if (!res.ok || !data.ok || !data.bookmark) throw new Error(data.error || 'Sayfa getirilemedi.')
      fill(data.bookmark)
      status.textContent = 'Bilgiler getirildi. Gerekirse düzenleyin.'
    } catch (err) {
      status.textContent = 'Otomatik bilgi alınamadı: ' + (err.message || 'hata') + ' — alanları elle doldurabilirsiniz.'
    }
  }
  // `fetchMeta` async: olay dinleyicisine doğrudan bağlanınca dönen söz
  // yakalanmıyordu (Sonar S9383). Sarmalayıcı, gövde dışında atılan bir
  // hatayı da kullanıcıya bildirir.
  const metaGetir = () => {
    fetchMeta().catch((err) => {
      status.textContent = err.message || 'Adres bilgisi alınamadı.'
    })
  }
  fetchBtn.onclick = metaGetir
  urlInput.onkeydown = (e) => {
    if (e.key === 'Enter') { e.preventDefault(); metaGetir() }
  }
  confirmBtn.onclick = () => {
    const url = urlInput.value.trim()
    if (!url) { status.textContent = 'Adres gerekli.'; return }
    let site = siteInput.value.trim()
    if (!site) {
      try { site = new URL(url).hostname } catch (_) { site = '' }
    }
    editor.chain().focus().insertContent({
      type: 'bookmark',
      attrs: {
        url,
        title: titleInput.value.trim() || url,
        description: descInput.value.trim(),
        site,
        image: imageInput.value.trim(),
      },
    }).run()
    closeDialog('editorBookmarkDialog')
  }
  cancelBtn.onclick = () => closeDialog('editorBookmarkDialog')
}

function insertCodeBlock(editor) {
  if (editor.isActive('codeBlock')) {
    editor.chain().focus().toggleCodeBlock().run()
    return
  }
  const lang = (window.prompt('Kod dili (js, python, go, html… — boş bırakılabilir)', '') || '').trim()
  const chain = editor.chain().focus().toggleCodeBlock()
  if (lang) chain.updateAttributes('codeBlock', { language: lang }).run()
  else chain.run()
}

function insertToggle(editor) {
  editor.chain().focus().insertContent({
    type: 'toggle',
    attrs: { title: 'Başlık', open: true },
    content: [{ type: 'paragraph' }],
  }).run()
}

function insertCallout(editor) {
  let kind = (window.prompt('Callout türü: info, warning, success, danger', 'info') || 'info').toLowerCase().trim()
  if (!CALLOUT_LABELS[kind]) kind = 'info'
  editor.chain().focus().insertContent({
    type: 'callout',
    attrs: { kind },
    content: [{ type: 'paragraph' }],
  }).run()
}

function openLinkDialog(editor) {
  const dialog = document.getElementById('editorLinkEditDialog')
  const hrefInput = document.getElementById('editorLinkHref')
  const textInput = document.getElementById('editorLinkText')
  const textWrap = document.getElementById('editorLinkTextWrap')
  if (!dialog || !hrefInput || !textInput) return
  const sel = editor.state.selection
  const current = editor.getAttributes('link').href || ''
  hrefInput.value = current || 'https://'
  if (sel.empty) {
    textWrap.hidden = false
    textInput.value = ''
    textInput.focus()
  } else {
    textWrap.hidden = true
    textInput.value = editor.state.doc.textBetween(sel.from, sel.to, ' ')
  }
  dialog.hidden = false
  if (!sel.empty) hrefInput.focus()
  const close = () => { dialog.hidden = true }
  const href = () => hrefInput.value.trim()
  document.getElementById('editorLinkEditCancel').onclick = close
  document.getElementById('editorLinkRemove').onclick = () => {
    editor.chain().focus().extendMarkRange('link').unsetLink().run()
    close()
  }
  document.getElementById('editorLinkOpen').onclick = () => {
    if (href()) window.open(href(), '_blank', 'noopener')
  }
  document.getElementById('editorLinkSave').onclick = () => {
    const url = href()
    if (!url) {
      editor.chain().focus().extendMarkRange('link').unsetLink().run()
      return close()
    }
    if (editor.state.selection.empty) {
      const text = textInput.value.trim() || url
      editor.chain().focus().insertContent({
        type: 'text', text, marks: [{ type: 'link', attrs: { href: url } }],
      }).run()
    } else {
      editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
    }
    close()
  }
}

function openShortcuts() {
  const d = document.getElementById('editorShortcuts')
  if (!d) return
  d.hidden = false
  const close = () => { d.hidden = true }
  document.getElementById('editorShortcutsClose').onclick = close
}

function blockAt(editor, x, y) {
  const found = editor.view.posAtCoords({ left: x, top: y })
  if (!found) return null
  const $pos = editor.state.doc.resolve(found.pos)
  for (let d = 1; d <= $pos.depth; d++) {
    const node = $pos.node(d)
    if (node.isBlock && node.type.name !== 'doc') {
      return { pos: $pos.before(d), node }
    }
  }
  return null
}

let blockMenuTarget = null

function showBlockMenu(editor, x, y, target) {
  const menu = document.getElementById('editorBlockMenu')
  if (!menu || !target) return
  blockMenuTarget = { editor, ...target }
  menu.hidden = false
  menu.style.left = Math.min(x, window.innerWidth - 170) + 'px'
  menu.style.top = Math.min(y, window.innerHeight - 90) + 'px'
}

function bindBlockMenu(editor) {
  const menu = document.getElementById('editorBlockMenu')
  if (menu) {
    menu.onclick = (e) => {
      const b = e.target.closest('[data-block-action]')
      if (!b || !blockMenuTarget) return
      const { pos, node } = blockMenuTarget
      if (b.dataset.blockAction === 'delete') {
        editor.chain().focus().deleteRange({ from: pos, to: pos + node.nodeSize }).run()
      } else if (b.dataset.blockAction === 'duplicate') {
        editor.chain().focus().insertContentAt(pos + node.nodeSize, node.toJSON()).run()
      }
      menu.hidden = true
    }
  }
  editor.view.dom.addEventListener('contextmenu', (e) => {
    const target = blockAt(editor, e.clientX, e.clientY)
    if (!target) return
    e.preventDefault()
    showBlockMenu(editor, e.clientX, e.clientY, target)
  })
}

function setInternalLinkFromDialog(editor) {
  if (editor.state.selection.empty) {
    alert('İç link için önce metin seçin.')
    return
  }
  const dialog = document.getElementById('editorLinkDialog')
  const input = document.getElementById('editorLinkSearch')
  const results = document.getElementById('editorLinkResults')
  if (!dialog || !input || !results) return
  dialog.hidden = false
  input.value = ''
  results.innerHTML = '<p class="small muted">Haber başlığı yazın…</p>'
  input.focus()
  let timer = null
  let seq = 0
  const search = () => {
    const q = input.value.trim()
    if (q.length < 2) {
      results.innerHTML = '<p class="small muted">En az 2 karakter yazın.</p>'
      return
    }
    const my = ++seq
    results.innerHTML = '<p class="small muted">Aranıyor…</p>'
    fetch('/admin/icerikler/ic-link-ara?q=' + encodeURIComponent(q), { credentials: 'same-origin' })
      .then((r) => r.json())
      .then((data) => {
        if (my !== seq) return
        results.innerHTML = ''
        const items = (data && data.items) || []
        if (!items.length) {
          results.innerHTML = '<p class="small muted">Sonuç bulunamadı.</p>'
          return
        }
        items.forEach((item) => {
          const b = document.createElement('button')
          b.type = 'button'
          b.className = 'be-link-result'
          b.textContent = item.title
          b.addEventListener('click', () => {
            editor.chain().focus().extendMarkRange('link').setLink({ href: item.url }).run()
            closeDialog('editorLinkDialog')
          })
          results.appendChild(b)
        })
      })
      .catch(() => {
        if (my === seq) results.innerHTML = '<p class="small muted">Arama başarısız.</p>'
      })
  }
  input.oninput = () => {
    clearTimeout(timer)
    timer = setTimeout(search, 220)
  }
  document.getElementById('editorLinkCancel').onclick = () => closeDialog('editorLinkDialog')
}

function editorContentID() {
  const host = document.getElementById('blockEditor')
  return host ? (host.getAttribute('data-content-id') || '0') : '0'
}

function openComments(editor) {
  const panel = document.getElementById('editorComments')
  if (!panel) return
  panel.hidden = false
  const quote = document.getElementById('editorNoteQuote')
  if (quote) {
    const sel = editor.state.selection
    const text = sel && !sel.empty ? editor.state.doc.textBetween(sel.from, sel.to, '\n') : ''
    quote.textContent = text ? 'Seçili: ' + text.slice(0, 160) : 'Genel not'
  }
  loadEditorNotes()
}

function loadEditorNotes() {
  const list = document.getElementById('editorCommentsList')
  if (!list) return
  const id = editorContentID()
  list.innerHTML = '<p class="small muted">Yükleniyor…</p>'
  fetch(`/admin/icerikler/${id}/yorumlar`, { credentials: 'same-origin' })
    .then((r) => r.json())
    .then((data) => {
      list.innerHTML = ''
      const items = (data && data.items) || []
      if (!items.length) {
        list.innerHTML = '<p class="small muted">Henüz editör notu yok.</p>'
        return
      }
      items.forEach((n) => {
        const card = document.createElement('div')
        card.className = 'be-note' + (n.Resolved ? ' resolved' : '')
        if (n.Quote) {
          const q = document.createElement('div')
          q.className = 'be-note-quote'
          q.textContent = n.Quote.slice(0, 200)
          card.appendChild(q)
        }
        const body = document.createElement('div')
        body.className = 'be-note-body'
        body.textContent = n.Body
        card.appendChild(body)
        const meta = document.createElement('div')
        meta.className = 'be-note-meta'
        meta.textContent = (n.UserName || 'Editör') + ' · ' + new Date(n.CreatedAt).toLocaleString('tr-TR')
        card.appendChild(meta)
        const actions = document.createElement('div')
        actions.className = 'be-note-actions'
        const resolve = document.createElement('button')
        resolve.type = 'button'
        resolve.className = 'btn ghost small'
        resolve.textContent = n.Resolved ? 'Geri al' : 'Çöz'
        resolve.onclick = () => {
          const fd = new URLSearchParams()
          fd.set('resolved', n.Resolved ? '0' : '1')
          fetch(`/admin/icerikler/${editorContentID()}/yorumlar/${n.ID}/cozum`, {
            method: 'POST', credentials: 'same-origin',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
            body: fd.toString(),
          }).then(loadEditorNotes)
            .catch(() => alert('Not durumu güncellenemedi.'))
        }
        actions.appendChild(resolve)
        const del = document.createElement('button')
        del.type = 'button'
        del.className = 'btn ghost small'
        del.textContent = 'Sil'
        del.onclick = () => {
          if (!confirm('Not silinsin mi?')) return
          fetch(`/admin/icerikler/${editorContentID()}/yorumlar/${n.ID}/sil`, {
            method: 'POST', credentials: 'same-origin',
          }).then(loadEditorNotes)
            .catch(() => alert('Not silinemedi.'))
        }
        actions.appendChild(del)
        card.appendChild(actions)
        list.appendChild(card)
      })
    })
    .catch(() => { list.innerHTML = '<p class="small muted">Notlar yüklenemedi.</p>' })
}

function addEditorNote() {
  const body = document.getElementById('editorNoteBody')
  const quote = document.getElementById('editorNoteQuote')
  if (!body || !body.value.trim()) return
  const fd = new URLSearchParams()
  fd.set('body', body.value.trim())
  fd.set('quote', (quote && quote.textContent.startsWith('Seçili: ')) ? quote.textContent.slice(8) : '')
  fetch(`/admin/icerikler/${editorContentID()}/yorumlar`, {
    method: 'POST', credentials: 'same-origin',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
    body: fd.toString(),
  }).then(() => {
    body.value = ''
    loadEditorNotes()
  }).catch(() => alert('Not eklenemedi.'))
}

function bindComments() {
  const close = document.getElementById('editorCommentsClose')
  if (close) close.onclick = () => { const p = document.getElementById('editorComments'); if (p) p.hidden = true }
  const add = document.getElementById('editorNoteAdd')
  if (add) add.onclick = addEditorNote
}

function showRemoteSave(ev) {
  if (ev.user_id && Number(ev.user_id) === Number((document.getElementById('blockEditor') || {}).dataset?.userId || 0)) return
  const current = Number((document.getElementById('f_version') || {}).value || 0)
  if (ev.version && Number(ev.version) <= current) return
  let box = document.getElementById('editorRemoteSave')
  if (!box) {
    box = document.createElement('div')
    box.id = 'editorRemoteSave'
    box.className = 'be-remote-save'
    document.body.appendChild(box)
  }
  box.innerHTML = ''
  const text = document.createElement('span')
  text.textContent = (ev.user_name || 'Başka bir editör') + ' bu içeriği kaydetti (v' + (ev.version || '?') + ').'
  const btn = document.createElement('button')
  btn.type = 'button'
  btn.className = 'btn small'
  btn.textContent = 'Yenile'
  btn.onclick = () => window.location.reload()
  box.appendChild(text)
  box.appendChild(btn)
  box.hidden = false
}

function renderPresence(users) {
  let box = document.getElementById('editorPresence')
  if (!box) {
    box = document.createElement('div')
    box.id = 'editorPresence'
    box.className = 'be-presence'
    document.body.appendChild(box)
  }
  const names = (users || []).map((u) => u.name).filter(Boolean)
  if (names.length <= 1) {
    box.hidden = true
    return
  }
  box.hidden = false
  box.textContent = 'Düzenleyenler: ' + names.join(', ')
}

function connectEditorLive() {
  const host = document.getElementById('blockEditor')
  if (!host) return
  const id = Number(host.getAttribute('data-content-id') || 0)
  if (!id) return
  if (editorES && editorLiveID === id) return
  if (editorES) editorES.close()
  editorLiveID = id
  editorES = new EventSource('/admin/icerikler/' + id + '/duzenleme')
  editorES.addEventListener('saved', (e) => {
    try { showRemoteSave(JSON.parse(e.data)) } catch (_) { /* yok say */ }
  })
  editorES.addEventListener('presence', (e) => {
    try { renderPresence(JSON.parse(e.data).users) } catch (_) { /* yok say */ }
  })
}

function toggleFocusMode() {
  document.body.classList.toggle('be-focus')
  const on = document.body.classList.contains('be-focus')
  if (on && window.__contentEditor) window.__contentEditor.commands.focus()
}

function promptEmbed(editor) {
  const url = (window.prompt('YouTube veya X gönderi bağlantısı', 'https://') || '').trim()
  if (!url) return
  const provider = providerForUrl(url)
  if (!provider) {
    alert('Yalnızca YouTube video ve X/Twitter gönderi bağlantıları destekleniyor.')
    return
  }
  editor.chain().focus().insertContent({ type: 'embed', attrs: { url, provider } }).run()
}

function findEditorNode(editor, typeName, id) {
  let found = null
  editor.state.doc.descendants((node, pos) => {
    if (found) return false
    if (node.type.name === typeName && node.attrs.id === id) {
      found = { node, pos }
      return false
    }
    return true
  })
  return found
}

function insertUploadPlaceholder(editor, file) {
  const id = suggestionID()
  editor.chain().focus().insertContent({
    type: 'uploadPlaceholder',
    attrs: {
      id,
      filename: file.name || 'Görsel',
      status: 'loading',
      message: 'Yükleniyor…',
    },
  }).run()
  return id
}

function replaceUploadPlaceholder(editor, id, content) {
  const found = findEditorNode(editor, 'uploadPlaceholder', id)
  if (!found) return false
  const replacement = editor.schema.nodeFromJSON(content)
  const tr = editor.state.tr.replaceWith(found.pos, found.pos + found.node.nodeSize, replacement)
  tr.setMeta('suggestion', true)
  editor.view.dispatch(tr)
  return true
}

function failUploadPlaceholder(editor, id, message) {
  const found = findEditorNode(editor, 'uploadPlaceholder', id)
  if (!found) return false
  const tr = editor.state.tr.setNodeMarkup(found.pos, undefined, {
    ...found.node.attrs,
    status: 'error',
    message: message || 'Yükleme başarısız.',
  })
  tr.setMeta('suggestion', true)
  editor.view.dispatch(tr)
  return true
}

async function uploadEditorFiles(editor, files) {
  const images = Array.from(files || []).filter((f) => f && f.type && f.type.startsWith('image/'))
  if (!images.length) return false
  for (const file of images) {
    const placeholderID = insertUploadPlaceholder(editor, file)
    const fd = new FormData()
    fd.append('dosya', file, file.name || 'gorsel.jpg')
    try {
      const res = await fetch('/admin/dosyalar?json=1', {
        method: 'POST',
        body: fd,
        credentials: 'same-origin',
        headers: { 'X-Requested-With': 'fetch' },
      })
      const data = await res.json()
      if (!res.ok || !data.ok || !data.url) throw new Error(data.error || 'Yükleme başarısız.')
      if (editor.isDestroyed) return true
      const image = {
        type: 'image',
        attrs: { src: data.url, mediaId: data.id, alt: file.name.replace(/\.[^.]+$/, '') },
      }
      if (!replaceUploadPlaceholder(editor, placeholderID, image)) {
        editor.chain().focus().insertContent(image).run()
      }
    } catch (err) {
      if (!editor.isDestroyed) failUploadPlaceholder(editor, placeholderID, 'Görsel yüklenemedi: ' + (err.message || 'bilinmeyen hata'))
    }
  }
  return true
}

function cleanPastedHTML(html) {
  const doc = new DOMParser().parseFromString(html || '', 'text/html')
  doc.querySelectorAll('script,style,meta,link').forEach((el) => el.remove())
  doc.querySelectorAll('*').forEach((el) => {
    el.removeAttribute('style')
    el.removeAttribute('class')
    el.removeAttribute('id')
  })
  return doc.body.innerHTML
}

function aiBlocks(value) {
  return String(value || '').split(/\n{2,}/).map((s) => s.trim()).filter(Boolean).map((text) => ({
    type: 'paragraph', content: [{ type: 'text', text }],
  }))
}

// Simgeler yerel Tabler sprite'ından gelir (Faz 161).
// NEDEN: düğme etiketleri emoji ve geometrik glifti (🎤 ▦ ☰ ▸ ▧ …). Emoji
// yazı tipi olmayan sistemlerde kutu çiziliyordu; geometrik glifler ise her
// yazı tipinde farklı kalınlıkta görünüp araç çubuğunu dengesiz bırakıyordu.
// "@ad" ile başlayan etiket SVG olarak basılır, diğerleri düz metin kalır
// (H2, B, I, U, </>, 1., • gibi tipografik kısayol etiketleri).
function ikonSVG(ad) {
  const NS = 'http://www.w3.org/2000/svg'
  const svg = document.createElementNS(NS, 'svg')
  svg.setAttribute('class', 'ti')
  svg.setAttribute('aria-hidden', 'true')
  const use = document.createElementNS(NS, 'use')
  use.setAttribute('href', '/static/vendor/tabler-icons.svg#ti-' + ad)
  svg.appendChild(use)
  return svg
}

function iconButton(label, title, command, extra) {
  const btn = document.createElement('button')
  btn.type = 'button'
  btn.className = 'be-btn' + (extra ? ' ' + extra : '')
  if (label.charAt(0) === '@') {
    btn.appendChild(ikonSVG(label.slice(1)))
    btn.classList.add('be-btn-ikon')
  } else {
    btn.textContent = label
  }
  btn.title = title
  btn.setAttribute('aria-label', title)
  btn.dataset.cmd = command
  return btn
}

function buildToolbar() {
  const bar = document.createElement('div')
  bar.className = 'be-toolbar'
  const groups = [
    [['@arrow-back-up', 'Geri al', 'undo'], ['@arrow-forward-up', 'İleri al', 'redo']],
    [['H2', 'Başlık 2', 'h2'], ['H3', 'Başlık 3', 'h3']],
    [['B', 'Kalın', 'bold', 'be-bold'], ['I', 'İtalik', 'italic', 'be-italic'], ['U', 'Altı çizili', 'underline', 'be-underline']],
    [['•', 'Madde listesi', 'bullet'], ['1.', 'Numaralı liste', 'ordered'], ['“', 'Alıntı', 'blockquote'], ['@microphone', 'Sesle yaz (dikte)', 'dictate']],
    [['</>', 'Kod bloğu', 'codeblock'], ['—', 'Yatay çizgi', 'hr'], ['@table', 'Tablo', 'table']],
    [['@message-2', 'Callout', 'callout'], ['@chevron-right', 'Toggle', 'toggle'], ['@photo', 'Görsel', 'image'], ['@world', 'Uzak görsel', 'remoteimage'], ['@layout-grid', 'Galeri', 'gallery'], ['@bookmark', 'Yer imi', 'bookmark'], ['@file-text', 'Dosya', 'filecard'], ['@music', 'Ses', 'audiocard'], ['@video', 'Video', 'videocard'], ['@player-play', 'Embed', 'embed'], ['@box-multiple', 'Blok seçimi', 'blockselect']],
    [['@link', 'Bağlantı', 'link'], ['@arrow-up-right', 'İç link', 'internallink'], ['@message-circle', 'Editör notları', 'comments'], ['@pencil', 'Öneri modu', 'suggestion'], ['@focus-2', 'Odak modu', 'focus'], ['@history', 'Sürüm geçmişi', 'history'], ['@eraser', 'Biçimi temizle', 'clear'], ['@keyboard', 'Klavye kısayolları', 'shortcuts']],
  ]
  groups.forEach((group, gi) => {
    if (gi > 0) {
      const sep = document.createElement('span')
      sep.className = 'be-sep'
      bar.appendChild(sep)
    }
    const wrap = document.createElement('span')
    wrap.className = 'be-toolbar-group' + (gi >= 4 ? ' be-secondary-group' : '')
    group.forEach(([label, title, cmd, extra]) => wrap.appendChild(iconButton(label, title, cmd, extra)))
    bar.appendChild(wrap)
  })
  const more = iconButton('@dots', 'Daha fazla araç', 'more')
  more.classList.add('be-more-toggle')
  bar.appendChild(more)
  const menu = document.createElement('div')
  menu.className = 'be-more-menu'
  menu.id = 'editorMoreMenu'
  menu.hidden = true
  bar.querySelectorAll('.be-secondary-group [data-cmd]').forEach((btn) => {
    const item = document.createElement('button')
    item.type = 'button'
    item.className = 'be-more-item'
    item.dataset.cmd = btn.dataset.cmd
    const ik = document.createElement('span')
    ik.setAttribute('aria-hidden', 'true')
    ik.appendChild(btn.firstChild.cloneNode(true))
    const et = document.createElement('span')
    et.textContent = btn.title
    item.appendChild(ik)
    item.appendChild(et)
    menu.appendChild(item)
  })
  bar.appendChild(menu)
  let outsideHandler = null
  const closeMenu = () => {
    menu.hidden = true
    if (outsideHandler) {
      document.removeEventListener('click', outsideHandler)
      outsideHandler = null
    }
  }
  more.addEventListener('click', (event) => {
    event.stopPropagation()
    if (!menu.hidden) {
      closeMenu()
      return
    }
    menu.hidden = false
    const rect = more.getBoundingClientRect()
    menu.style.top = rect.bottom + 6 + 'px'
    menu.style.right = Math.max(8, window.innerWidth - rect.right) + 'px'
    setTimeout(() => {
      outsideHandler = (outsideEvent) => {
        if (!menu.contains(outsideEvent.target) && !more.contains(outsideEvent.target)) closeMenu()
      }
      document.addEventListener('click', outsideHandler)
    }, 0)
  })
  window.addEventListener('resize', closeMenu)
  return bar
}

function handleCommand(editor, cmd) {
  if (cmd === 'undo') editor.chain().focus().undo().run()
  else if (cmd === 'redo') editor.chain().focus().redo().run()
  else if (cmd === 'h2') editor.chain().focus().toggleHeading({ level: 2 }).run()
  else if (cmd === 'h3') editor.chain().focus().toggleHeading({ level: 3 }).run()
  else if (cmd === 'bold') editor.chain().focus().toggleBold().run()
  else if (cmd === 'italic') editor.chain().focus().toggleItalic().run()
  else if (cmd === 'underline') editor.chain().focus().toggleUnderline().run()
  else if (cmd === 'bullet') editor.chain().focus().toggleBulletList().run()
  else if (cmd === 'ordered') editor.chain().focus().toggleOrderedList().run()
  else if (cmd === 'blockquote') editor.chain().focus().toggleBlockquote().run()
  else if (cmd === 'codeblock') insertCodeBlock(editor)
  else if (cmd === 'hr') editor.chain().focus().setHorizontalRule().run()
  else if (cmd === 'table') editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
  else if (cmd === 'callout') insertCallout(editor)
  else if (cmd === 'toggle') insertToggle(editor)
  else if (cmd === 'image') setImageFromDialog(editor)
  else if (cmd === 'remoteimage') openRemoteImageDialog(editor)
  else if (cmd === 'gallery') setGalleryFromDialog(editor)
  else if (cmd === 'bookmark') openBookmarkDialog(editor)
  else if (cmd === 'filecard') openMediaCardDialog(editor, 'file')
  else if (cmd === 'audiocard') openMediaCardDialog(editor, 'audio')
  else if (cmd === 'videocard') openMediaCardDialog(editor, 'video')
  else if (cmd === 'blockselect') selectBlocks(editor)
  else if (cmd === 'embed') promptEmbed(editor)
  else if (cmd === 'internallink') setInternalLinkFromDialog(editor)
  else if (cmd === 'comments') openComments(editor)
  else if (cmd === 'suggestion') toggleSuggestionMode(editor)
  else if (cmd === 'focus') toggleFocusMode()
  else if (cmd === 'shortcuts') openShortcuts()
  else if (cmd === 'clear') editor.chain().focus().unsetAllMarks().clearNodes().run()
  else if (cmd === 'history') {
    const b = document.querySelector('#aiEditor [data-ai-history]')
    if (b) b.click()
  } else if (cmd === 'link') {
    openLinkDialog(editor)
  } else if (cmd === 'dictate') {
    toggleDictation(editor)
  }
}

function updateStatus(editor) {
  const el = document.getElementById('editorStatus')
  if (!el) return
  el.textContent = `${editor.storage.characterCount.words()} kelime · ${editor.storage.characterCount.characters()} karakter`
}

function updateToolbar(editor) {
  const host = document.getElementById('blockEditor')
  if (!host) return
  const bar = host.previousElementSibling
  if (bar && bar.classList.contains('be-toolbar')) {
    const states = {
      bold: editor.isActive('bold'), italic: editor.isActive('italic'), underline: editor.isActive('underline'),
      h2: editor.isActive('heading', { level: 2 }), h3: editor.isActive('heading', { level: 3 }),
      bullet: editor.isActive('bulletList'), ordered: editor.isActive('orderedList'),
      blockquote: editor.isActive('blockquote'), codeblock: editor.isActive('codeBlock'), link: editor.isActive('link'),
    }
    bar.querySelectorAll('[data-cmd]').forEach((b) => {
      // dictate dikte durumunu kendi yönetir; burada "kapalı"ya çekilmemeli.
      if (b.dataset.cmd === 'dictate') return
      b.classList.toggle('is-active', !!states[b.dataset.cmd])
    })
  }
  if (bubbleElement) {
    bubbleElement.querySelectorAll('[data-cmd]').forEach((b) => {
      const on = { bold: 'bold', italic: 'italic', underline: 'underline', h2: 'heading', link: 'link' }[b.dataset.cmd]
      b.classList.toggle('is-active', on ? editor.isActive(on, b.dataset.cmd === 'h2' ? { level: 2 } : undefined) : false)
    })
  }
}

function collabColor(name) {
  const palette = ['#2563eb', '#7c3aed', '#db2777', '#ea580c', '#16a34a', '#0891b2', '#4f46e5', '#b45309']
  let sum = 0
  for (const ch of String(name || 'editör')) sum += ch.codePointAt(0)
  return palette[sum % palette.length]
}

async function init(host) {
  if (!host || host.dataset.ready) return
  const hidden = document.querySelector('[name="detay"]')
  if (!hidden) return
  host.dataset.ready = 'loading'
  suggestionUserName = host.getAttribute('data-user-name') || 'Editör'
  try {
    suggestionMode = localStorage.getItem('be-suggestion-mode') === '1'
  } catch (_) {
    suggestionMode = false
  }

  let content = { type: 'doc', content: [{ type: 'paragraph' }] }
  try {
    const parsed = JSON.parse(hidden.value || '{}')
    if (parsed && parsed.type === 'doc') content = parsed
  } catch (_) { /* boş/eski içerik */ }

  let collab = null
  const contentID = Number(host.getAttribute('data-content-id') || 0)
  if (contentID > 0) {
    try {
      const res = await fetch(`/admin/icerikler/${contentID}/collab-token`, { credentials: 'same-origin' })
      if (res.ok) {
        const data = await res.json()
        if (data.url && data.token) collab = data
      }
    } catch (_) { /* yerel düzenleme */ }
  }

  bubbleElement = document.createElement('div')
  bubbleElement.className = 'be-bubble'
  ;[['B', 'Kalın', 'bold'], ['I', 'İtalik', 'italic'], ['U', 'Altı çizili', 'underline'], ['H2', 'Başlık', 'h2'], ['🔗', 'Bağlantı', 'link']]
    .forEach(([label, title, cmd]) => bubbleElement.appendChild(iconButton(label, title, cmd)))
  document.body.appendChild(bubbleElement)

  const extensions = [
    StarterKit.configure({
      heading: { levels: [2, 3, 4] },
      codeBlock: false,
      link: false,
      underline: false,
      undoRedo: false,
    }),
    Link.configure({
      openOnClick: false,
      autolink: true,
      linkOnPaste: true,
      HTMLAttributes: { rel: 'noopener noreferrer', target: '_blank' },
    }),
    Underline,
    SuggestionInsert,
    SuggestionDelete,
    SuggestionTracker,
    CodeBlockLang.configure({ HTMLAttributes: { class: 'be-code' } }),
    MediaImage.configure({ inline: false, allowBase64: false }),
    Placeholder.configure({ placeholder: 'Yazmaya başlayın veya / ile blok ekleyin…' }),
    CharacterCount,
    TableKit,
    BlockRange.configure({ key: 'Mod' }),
    Callout,
    Toggle,
    Bookmark,
    MediaCard,
    UploadPlaceholder,
    Embed,
    Gallery,
    SlashCommand,
    DragHandle.configure({
      render: () => {
        const el = document.createElement('div')
        el.className = 'be-drag'
        el.textContent = '⠿'
        el.title = 'Bloğu taşı'
        return el
      },
    }),
    BubbleMenu.configure({
      element: bubbleElement,
      shouldShow: ({ editor }) => !editor.state.selection.empty && !editor.isActive('codeBlock'),
    }),
  ]

  if (collab) {
    const ydoc = new Y.Doc()
    const awareness = new Awareness(ydoc)
    collabProvider = new HocuspocusProvider({
      url: collab.url,
      name: collab.name || ('content-' + contentID),
      token: collab.token,
      document: ydoc,
      awareness,
    })
    window.__collabSynced = false
    collabProvider.on('synced', () => { window.__collabSynced = true })
    extensions.push(Collaboration.configure({ document: ydoc, field: 'default' }))
    awareness.setLocalStateField('user', {
      name: host.getAttribute('data-user-name') || 'Editör',
      color: collabColor(host.getAttribute('data-user-name') || 'Editör'),
    })
    extensions.push(CursorPresence.configure({ awareness }))
    window.__collabActive = true
    const ver = document.getElementById('f_version')
    if (ver) ver.value = '0'
  } else {
    window.__collabActive = false
  }

  let editor = null
  editor = new Editor({
    element: host,
    ...(collab ? {} : { content }),
    extensions,
    editorProps: {
      attributes: { class: 'be-content', spellcheck: 'true' },
      transformPastedHTML: cleanPastedHTML,
      handleKeyDown: (view, event) => {
        if (!suggestionMode || event.isComposing) return false
        const sel = editor.state.selection
        if (event.key === 'Backspace' || event.key === 'Delete') {
          let from = sel.from
          let to = sel.to
          if (sel.empty) {
            if (event.key === 'Backspace') from = Math.max(0, from - 1)
            else to = Math.min(editor.state.doc.content.size, to + 1)
          }
          if (from >= to) return false
          event.preventDefault()
          trackDeletion(editor, from, to)
          updateSuggestionUI(editor)
          return true
        }
        if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'x' && !sel.empty) {
          event.preventDefault()
          trackDeletion(editor, sel.from, sel.to)
          updateSuggestionUI(editor)
          return true
        }
        return false
      },
      handleTextInput: (view, from, to, text) => {
        if (!suggestionMode || from === to || view.composing || !text) return false
        if (!trackDeletion(editor, from, to)) return false
        editor.chain().focus().setTextSelection(editor.state.selection.to).insertContent({
          type: 'text',
          text,
          marks: [{ type: 'suggestionInsert', attrs: newSuggestionAttrs() }],
        }).run()
        return true
      },
      handlePaste: (view, event) => {
        const files = event.clipboardData && event.clipboardData.files
        if (files && files.length && Array.from(files).some((f) => f.type.startsWith('image/'))) {
          event.preventDefault()
          uploadEditorFiles(editor, files)
            .catch(() => alert('Görsel yüklenemedi.'))
          return true
        }
        if (suggestionMode && !editor.state.selection.empty) {
          const sel = editor.state.selection
          const text = event.clipboardData && event.clipboardData.getData('text/plain')
          const html = event.clipboardData && event.clipboardData.getData('text/html')
          event.preventDefault()
          trackDeletion(editor, sel.from, sel.to)
          const content = html ? cleanPastedHTML(html) : text
          if (content) {
            editor.chain().focus().setTextSelection(editor.state.selection.to).insertContent(content).run()
            updateSuggestionUI(editor)
          }
          return true
        }
        const text = event.clipboardData && event.clipboardData.getData('text/plain')
        const html = event.clipboardData && event.clipboardData.getData('text/html')
        if (text && !html && /(^|\n)(#{1,6} |[-*+] |\d+\. |```|> )/.test(text)) {
          event.preventDefault()
          editor.commands.insertContent(marked.parse(text))
          return true
        }
        return false
      },
      handleDrop: (view, event) => {
        const files = event.dataTransfer && event.dataTransfer.files
        if (files && files.length && Array.from(files).some((f) => f.type.startsWith('image/'))) {
          event.preventDefault()
          uploadEditorFiles(editor, files)
            .catch(() => alert('Görsel yüklenemedi.'))
          return true
        }
        return false
      },
    },
    onUpdate: ({ editor }) => {
      hidden.value = JSON.stringify(editor.getJSON())
      hidden.dispatchEvent(new Event('input', { bubbles: true }))
      updateStatus(editor)
      updateToolbar(editor)
      updateSuggestionUI(editor)
    },
    onSelectionUpdate: ({ editor }) => {
      updateToolbar(editor)
      updateSuggestionUI(editor)
    },
    onCreate: ({ editor }) => {
      hidden.value = JSON.stringify(editor.getJSON())
      updateStatus(editor)
      updateToolbar(editor)
      updateSuggestionUI(editor)
    },
  })

  const bar = buildToolbar()
  host.parentNode.insertBefore(bar, host)
  bar.addEventListener('click', (e) => {
    const btn = e.target.closest('[data-cmd]')
    if (!btn) return
    handleCommand(editor, btn.dataset.cmd)
    updateToolbar(editor)
    const menu = document.getElementById('editorMoreMenu')
    if (menu && btn.closest('.be-more-menu')) menu.hidden = true
  })
  bubbleElement.addEventListener('click', (e) => {
    const btn = e.target.closest('[data-cmd]')
    if (!btn) return
    handleCommand(editor, btn.dataset.cmd)
    updateToolbar(editor)
  })

  host.dataset.ready = '1'
  instance = editor
  window.__contentEditor = editor
  window.__contentEditorSync = () => {
    hidden.value = JSON.stringify(editor.getJSON())
    hidden.dispatchEvent(new Event('input', { bubbles: true }))
  }
  window.__contentEditorAIBlocks = aiBlocks
  window.__editorUploadFiles = (files) =>
    uploadEditorFiles(editor, files).catch(() => alert('Görsel yüklenemedi.'))
  bindComments()
  bindBlockMenu(editor)
  bindSuggestionBar(editor)
  updateSuggestionUI(editor)
  connectEditorLive()
  document.addEventListener('mousedown', (e) => {
    const menu = document.getElementById('editorBlockMenu')
    if (menu && !e.target.closest('#editorBlockMenu')) menu.hidden = true
  })
}

function boot() {
  const host = document.getElementById('blockEditor')
  if (!host) {
    // Editör kaldırıldıysa dikte de durmalı; aksi hâlde tanıyıcı eski
    // örneğe yazmaya devam eder.
    stopDictation()
    return
  }
  if (instance) {
    stopDictation()
    instance.destroy()
    instance = null
    const old = document.querySelector('.be-toolbar')
    if (old) old.remove()
    if (bubbleElement) {
      bubbleElement.remove()
      bubbleElement = null
    }
    if (collabProvider) {
      collabProvider.destroy()
      collabProvider = null
    }
    window.__collabActive = false
    host.dataset.ready = '0'
  }
  // `init` async: dönen söz yakalanmazsa editör kurulumundaki bir hata
  // (collab bağlantısı, eklenti yüklemesi) sessizce kaybolur ve sayfa
  // "hazır" görünürken editör çalışmaz (Sonar S9383).
  init(host).catch((err) => {
    console.error('Editör başlatılamadı:', err)
    const h = document.getElementById('blockEditor')
    if (h) {
      h.dataset.ready = '0'
      h.innerHTML = '<p class="flash error" style="margin:12px">Editör yüklenemedi. Sayfayı yenileyin.</p>'
    }
  })
}

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && document.body.classList.contains('be-focus')) {
    document.body.classList.remove('be-focus')
  }
  // Escape dikteyi de durdurur (mikrofonun açık kalmasını önler).
  if (e.key === 'Escape') {
    const bar = document.getElementById('dictationBar')
    if (bar && !bar.hidden) stopDictation()
  }
})
document.addEventListener('DOMContentLoaded', boot)
document.addEventListener('htmx:afterSwap', boot)
document.addEventListener('htmx:afterSettle', boot)
