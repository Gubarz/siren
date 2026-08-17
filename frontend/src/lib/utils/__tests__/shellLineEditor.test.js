import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { createShellLineEditor } from '../shellLineEditor.js'

const CR = '\r'
const DEL = '\x7f'
const BS = '\x08'

describe('createShellLineEditor', () => {
  let editor

  beforeEach(() => {
    vi.useFakeTimers()
    editor = createShellLineEditor()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('echoes printable characters instantly and sends nothing until Enter', () => {
    expect(editor.input('w')).toEqual({ render: 'w', send: '' })
    expect(editor.input('h')).toEqual({ render: 'h', send: '' })
    expect(editor.line()).toBe('wh')
  })

  it('submits the completed line with CR on Enter and echoes the newline', () => {
    editor.input('whoami')
    expect(editor.input(CR)).toEqual({ render: '\r\n', send: 'whoami\r' })
    expect(editor.line()).toBe('')
  })

  it('treats LF as Enter', () => {
    editor.input('dir')
    expect(editor.input('\n')).toEqual({ render: '\r\n', send: 'dir\r' })
  })

  it('erases the last character on backspace (DEL and BS)', () => {
    editor.input('abc')
    const del = editor.input(DEL)
    expect(del.render).toBe('\x08\x1b[K')
    expect(del.send).toBe('')
    expect(editor.line()).toBe('ab')
    editor.input(BS)
    expect(editor.line()).toBe('a')
  })

  it('backspace pops a whole surrogate pair', () => {
    editor.input('a😀')
    editor.input(DEL)
    expect(editor.line()).toBe('a')
    expect(editor.input('b\r').send).toBe('ab\r')
  })

  it('ignores backspace on an empty line', () => {
    expect(editor.input(DEL)).toEqual({ render: '', send: '' })
  })

  it('clears the line and shows ^C on Ctrl+C without sending it', () => {
    editor.input('abc')
    expect(editor.input('\x03')).toEqual({ render: '^C\r\n', send: '' })
    expect(editor.line()).toBe('')
  })

  it('ignores cursor movement escape sequences', () => {
    expect(editor.input('\x1b[A')).toEqual({ render: '', send: '' })
    expect(editor.input('\x1b[D')).toEqual({ render: '', send: '' })
    expect(editor.line()).toBe('')
  })

  it('passes bracketed paste content through as typed input', () => {
    editor.input('\x1b[200~whoami\r\x1b[201~')
    expect(editor.line()).toBe('')
  })

  it('suppresses the remote echo of a submitted line', () => {
    editor.input('whoami\r')
    expect(editor.output('whoami\r\n')).toBe('')
    expect(editor.output('user\r\nPS C:\\> ')).toBe('user\r\nPS C:\\> ')
  })

  it('suppresses an echo split across output events', () => {
    editor.input('whoami\r')
    expect(editor.output('who')).toBe('')
    expect(editor.output('ami\r\n')).toBe('')
  })

  it('suppresses an echo preceded by a newline', () => {
    editor.input('whoami\r')
    expect(editor.output('\r\nwhoami\r\n')).toBe('')
  })

  it('passes output through untouched when the remote never echoes', () => {
    editor.input('whoami\r')
    expect(editor.output('\r\nuser\r\nPS C:\\> ')).toBe('\r\nuser\r\nPS C:\\> ')
  })

  it('disarms suppression on mismatch', () => {
    editor.input('whoami\r')
    expect(editor.output('banner\r\n')).toBe('banner\r\n')
    expect(editor.output('whoami\r\n')).toBe('whoami\r\n')
  })

  it('consumes echoes of multiple pasted lines in order', () => {
    editor.input('echo a\r')
    editor.input('echo b\r')
    expect(editor.output('echo a')).toBe('')
    expect(editor.output('\r\n')).toBe('')
    expect(editor.output('echo b')).toBe('')
    expect(editor.output('\r\na\r\nb\r\n')).toBe('a\r\nb\r\n')
  })

  it('expires suppression after the timeout', () => {
    editor.input('whoami\r')
    vi.advanceTimersByTime(3000)
    expect(editor.output('whoami\r\n')).toBe('whoami\r\n')
  })

  it('does not suppress when the line was empty', () => {
    editor.input('\r')
    expect(editor.output('PS C:\\> ')).toBe('PS C:\\> ')
  })
})
