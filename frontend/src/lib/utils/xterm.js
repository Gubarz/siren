import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

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
  return { term, fit }
}
