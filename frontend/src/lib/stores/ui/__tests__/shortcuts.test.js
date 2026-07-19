import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('shortcuts store', () => {
  let handleKeydown
  let register

  beforeEach(async () => {
    vi.resetModules()
    ;({ handleKeydown, register } = await import('../shortcuts.svelte.js'))
  })

  it('normalizes shifted digit keys to the requested digit shortcut', () => {
    const first = vi.fn()
    const tenth = vi.fn()

    register('Ctrl+Shift+1', first)
    register('Ctrl+Shift+0', tenth)

    const firstEvent = keyEvent({ code: 'Digit1', key: '!' })
    const tenthEvent = keyEvent({ code: 'Digit0', key: ')' })

    handleKeydown(firstEvent)
    handleKeydown(tenthEvent)

    expect(first).toHaveBeenCalledOnce()
    expect(tenth).toHaveBeenCalledOnce()
    expect(firstEvent.defaultPrevented).toBe(true)
    expect(tenthEvent.defaultPrevented).toBe(true)
  })

  it('normalizes shifted bracket keys to bracket shortcuts', () => {
    const focusLeft = vi.fn()
    const moveRight = vi.fn()

    register('Ctrl+Alt+[', focusLeft)
    register('Ctrl+Alt+Shift+]', moveRight)

    const focusEvent = keyEvent({ code: 'BracketLeft', key: '[', altKey: true, shiftKey: false })
    const moveEvent = keyEvent({ code: 'BracketRight', key: '}', altKey: true })

    handleKeydown(focusEvent)
    handleKeydown(moveEvent)

    expect(focusLeft).toHaveBeenCalledOnce()
    expect(moveRight).toHaveBeenCalledOnce()
  })

  function keyEvent({ code, key, altKey = false, shiftKey = true }) {
    return {
      code,
      key,
      ctrlKey: true,
      metaKey: false,
      altKey,
      shiftKey,
      target: { tagName: 'BODY' },
      defaultPrevented: false,
      preventDefault() {
        this.defaultPrevented = true
      },
    }
  }
})
