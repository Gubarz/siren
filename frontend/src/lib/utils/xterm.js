import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { Clipboard } from '@wailsio/runtime'
import '@xterm/xterm/css/xterm.css'

// xterm.js ships no clipboard keybindings (that's the embedder's job) and a
// Wails webview has no browser chrome to provide them, so Ctrl+Shift+C would
// otherwise fall through to xterm's default keymap and send \x03 — SIGINT to
// the remote process. Copy the selection / paste via the native clipboard,
// swallowing the key either way. Paste goes through term.paste() so it flows
// out the normal onData path with bracketed-paste handling.
export function createClipboardKeyHandler(term) {
  return (event) => {
    if (event.type !== 'keydown') return true
    if (!event.ctrlKey || !event.shiftKey || event.altKey || event.metaKey) return true
    const key = event.key?.toLowerCase()
    if (key === 'c') {
      const selection = term.getSelection()
      if (selection) Clipboard.SetText(selection).catch(() => {})
      return false
    }
    if (key === 'v') {
      Clipboard.Text()
        .then((text) => {
          if (text) term.paste(text)
        })
        .catch(() => {})
      return false
    }
    return true
  }
}

// Shared xterm.js factory. Returns { term, fit } already opened onto hostEl
// and configured with the console's font/theme/scrollback plus the OSC 10/11
// swallow that keeps color-query replies from leaking onto the remote shell.
// See ConsolePTY.svelte for the full incident notes on OSC 10/11.
export function createXterm(hostEl, overrides = {}) {
  const term = new Terminal({
    cursorBlink: true,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
    fontSize: 13,
    theme: { background: '#0b0f14' },
    scrollback: 10000,
    ...overrides,
  })
  const fit = new FitAddon()
  term.loadAddon(fit)
  term.open(hostEl)
  fit.fit()
  term.parser.registerOscHandler(10, () => true)
  term.parser.registerOscHandler(11, () => true)
  term.attachCustomKeyEventHandler(createClipboardKeyHandler(term))
  return { term, fit }
}
