import { SvelteMap } from 'svelte/reactivity'

const registry = new SvelteMap()

function normalizeKeyName(key) {
  if (key === ' ') return 'Space'
  if (key === '?') return '/'
  if (key.length === 1 && /[a-z]/i.test(key)) return key.toUpperCase()
  return key
}

function normalizeShortcut(shortcut) {
  const parts = shortcut.split('+')
  const key = normalizeKeyName(parts.pop())
  const modifiers = new Set(parts)
  const normalized = []
  if (modifiers.has('Ctrl') || modifiers.has('Cmd') || modifiers.has('Meta')) normalized.push('Ctrl')
  if (modifiers.has('Alt')) normalized.push('Alt')
  if (modifiers.has('Shift')) normalized.push('Shift')
  normalized.push(key)
  return normalized.join('+')
}

function eventKey(e) {
  const digit = /^Digit([0-9])$/.exec(e.code || '')
  if (digit) return digit[1]
  if (e.code === 'Slash') return '/'
  if (e.code === 'BracketLeft') return '['
  if (e.code === 'BracketRight') return ']'
  if (e.code === 'Comma') return ','
  if (e.code === 'Equal') return '='
  if (e.code === 'Minus') return '-'
  return normalizeKeyName(e.key)
}

export function register(shortcut, fn, label = '', category = 'global') {
  registry.set(normalizeShortcut(shortcut), { fn, label, category, key: shortcut })
}

export function unregister(shortcut) {
  registry.delete(normalizeShortcut(shortcut))
}

// Returns a reactive array of all registered shortcuts
export function getActiveShortcuts() {
  return Array.from(registry.values())
}

// True if the keydown target is inside an editable surface — an input,
// textarea, contenteditable, or select. Bare-key shortcuts (j, /, Enter)
// must not fire while the operator is typing.
function isTypingContext(e) {
  const el = e.target
  if (!el) return false
  const tag = el.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  return !!(el.isContentEditable)
}

export function handleKeydown(e) {
  const key = []
  if (e.ctrlKey || e.metaKey) key.push('Ctrl')
  if (e.altKey) key.push('Alt')
  if (e.shiftKey) key.push('Shift')
  key.push(eventKey(e))
  const combo = key.join('+')
  const item = registry.get(combo)
  if (!item) return
  const bare = !e.ctrlKey && !e.metaKey && !e.altKey
  if (bare && isTypingContext(e)) return
  e.preventDefault()
  item.fn()
}
