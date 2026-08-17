import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocked = vi.hoisted(() => ({
  SetText: vi.fn(() => Promise.resolve()),
  Text: vi.fn(() => Promise.resolve('')),
}))

vi.mock('@wailsio/runtime', () => ({
  Clipboard: { SetText: mocked.SetText, Text: mocked.Text },
}))

import { createClipboardKeyHandler } from '../xterm.js'

function keyEvent(overrides = {}) {
  return {
    type: 'keydown',
    ctrlKey: true,
    shiftKey: true,
    altKey: false,
    metaKey: false,
    key: 'C',
    ...overrides,
  }
}

describe('createClipboardKeyHandler', () => {
  let term
  let handler

  beforeEach(() => {
    vi.clearAllMocks()
    term = { getSelection: vi.fn(() => ''), paste: vi.fn() }
    handler = createClipboardKeyHandler(term)
  })

  it('copies the selection on Ctrl+Shift+C and swallows the key', () => {
    term.getSelection.mockReturnValue('payload')
    expect(handler(keyEvent({ key: 'C' }))).toBe(false)
    expect(mocked.SetText).toHaveBeenCalledWith('payload')
  })

  it('swallows Ctrl+Shift+C with no selection without touching the clipboard', () => {
    // Must still swallow: xterm's default keymap would translate this to \x03
    // and SIGINT the remote process.
    term.getSelection.mockReturnValue('')
    expect(handler(keyEvent({ key: 'C' }))).toBe(false)
    expect(mocked.SetText).not.toHaveBeenCalled()
  })

  it('pastes the clipboard on Ctrl+Shift+V and swallows the key', async () => {
    mocked.Text.mockResolvedValue('pasted text')
    expect(handler(keyEvent({ key: 'V' }))).toBe(false)
    await vi.waitFor(() => expect(term.paste).toHaveBeenCalledWith('pasted text'))
  })

  it('does not paste when the clipboard is empty', async () => {
    mocked.Text.mockResolvedValue('')
    expect(handler(keyEvent({ key: 'V' }))).toBe(false)
    await Promise.resolve()
    await Promise.resolve()
    expect(term.paste).not.toHaveBeenCalled()
  })

  it('passes plain Ctrl+C through to xterm (SIGINT stays available)', () => {
    expect(handler(keyEvent({ shiftKey: false, key: 'c' }))).toBe(true)
  })

  it('passes unrelated keys through to xterm', () => {
    expect(handler(keyEvent({ key: 'X' }))).toBe(true)
    expect(handler({ type: 'keydown', key: 'c' })).toBe(true)
  })

  it('ignores non-keydown events', () => {
    expect(handler(keyEvent({ type: 'keyup', key: 'C' }))).toBe(true)
    expect(mocked.SetText).not.toHaveBeenCalled()
  })
})
