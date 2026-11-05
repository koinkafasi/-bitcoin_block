import { Node, Mark, mergeAttributes } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Underline from '@tiptap/extension-underline'
import { TableKit } from '@tiptap/extension-table'
import CodeBlock from '@tiptap/extension-code-block'

const CodeBlockLang = CodeBlock.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      language: { default: null },
    }
  },
})

const MediaImage = Image.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      mediaId: { default: null },
      caption: { default: '' },
      layout: { default: 'normal' },
      width: { default: 100 },
      crop: { default: '' },
      credit: { default: '' },
      creditUrl: { default: '' },
      overlay: { default: '' },
      focalX: { default: 50 },
      focalY: { default: 50 },
    }
  },
})

const Callout = Node.create({
  name: 'callout',
  group: 'block',
  content: 'block+',
  defining: true,
  addAttributes() { return { kind: { default: 'info' } } },
  parseHTML() { return [{ tag: 'aside[data-callout]' }] },
  renderHTML({ HTMLAttributes }) {
    return ['aside', mergeAttributes(HTMLAttributes, { 'data-callout': HTMLAttributes.kind || 'info' }), 0]
  },
})

const SuggestionInsert = Mark.create({
  name: 'suggestionInsert',
  inclusive: false,
  excludes: 'suggestionDelete',
  addAttributes() {
    return { id: { default: '' }, user: { default: '' }, createdAt: { default: 0 } }
  },
  parseHTML() { return [{ tag: 'span[data-suggestion-insert]' }] },
  renderHTML({ HTMLAttributes }) {
    return ['span', { 'data-suggestion-insert': '', 'data-suggestion-id': HTMLAttributes.id || '' }, 0]
  },
})

const SuggestionDelete = Mark.create({
  name: 'suggestionDelete',
  inclusive: false,
  excludes: 'suggestionInsert',
  addAttributes() {
    return { id: { default: '' }, user: { default: '' }, createdAt: { default: 0 } }
  },
  parseHTML() { return [{ tag: 'span[data-suggestion-delete]' }] },
  renderHTML({ HTMLAttributes }) {
    return ['span', { 'data-suggestion-delete': '', 'data-suggestion-id': HTMLAttributes.id || '' }, 0]
  },
})

const Toggle = Node.create({
  name: 'toggle',
  group: 'block',
  content: 'block+',
  defining: true,
  addAttributes() { return { title: { default: 'Başlık' }, open: { default: true } } },
  parseHTML() { return [{ tag: 'details[data-toggle]' }] },
  renderHTML({ HTMLAttributes }) {
    return ['details', mergeAttributes(HTMLAttributes, { 'data-toggle': '' }), ['summary', {}, HTMLAttributes.title || 'Başlık'], ['div', {}, 0]]
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
  parseHTML() { return [{ tag: 'a[data-bookmark]' }] },
  renderHTML({ HTMLAttributes }) {
    return ['a', mergeAttributes(HTMLAttributes, { 'data-bookmark': '', href: HTMLAttributes.url || '#' })]
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
  parseHTML() { return [{ tag: 'div[data-media-card]' }] },
  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-media-card': '' })]
  },
})

const Embed = Node.create({
  name: 'embed',
  group: 'block',
  atom: true,
  draggable: true,
  addAttributes() { return { url: { default: '' }, provider: { default: '' } } },
  parseHTML() { return [{ tag: 'div[data-embed]' }] },
  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-embed': HTMLAttributes.provider || 'embed' })]
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
  parseHTML() { return [{ tag: 'div[data-upload-placeholder]' }] },
  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-upload-placeholder': '' })]
  },
})

const Gallery = Node.create({
  name: 'gallery',
  group: 'block',
  atom: true,
  draggable: true,
  addAttributes() { return { items: { default: [] } } },
  parseHTML() { return [{ tag: 'div[data-gallery]' }] },
  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-gallery': '' })]
  },
})

export const schemaExtensions = [
  StarterKit.configure({ heading: { levels: [2, 3, 4] }, codeBlock: false, link: false, underline: false, undoRedo: false }),
  CodeBlockLang,
  Link,
  Underline,
  SuggestionInsert,
  SuggestionDelete,
  MediaImage,
  TableKit,
  Callout,
  Toggle,
  Bookmark,
  MediaCard,
  UploadPlaceholder,
  Embed,
  Gallery,
]
