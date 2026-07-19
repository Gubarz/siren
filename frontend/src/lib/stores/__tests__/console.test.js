import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../api/console.js', () => ({
  ListCommands: vi.fn(() => Promise.resolve([])),
  RunSessionCommand: vi.fn(() => Promise.resolve('ok')),
  SendToSessionConsole: vi.fn(() => Promise.resolve()),
}))

describe('console store', () => {
  let dispatchCommand
  let emptySession
  let ensureSession
  let peekSession
  let SendToSessionConsole

  beforeEach(async () => {
    vi.resetModules()
    ;({ SendToSessionConsole } = await import('../../api/console.js'))
    ;({ dispatchCommand, emptySession, ensureSession, peekSession } = await import('../console.svelte.js'))
  })

  it('does not create a session when peeking', () => {
    expect(peekSession('agent-1')).toBeUndefined()
    expect(emptySession().lines).toEqual([])
  })

  it('creates a session explicitly', () => {
    const session = ensureSession('agent-1')
    expect(peekSession('agent-1')).toBe(session)
  })

  it('dispatch routes session commands to the live console subprocess', async () => {
    await dispatchCommand('agent-1', 'whoami')
    expect(SendToSessionConsole).toHaveBeenCalledWith('agent-1', 'whoami')
    expect(peekSession('agent-1').history).toEqual(['whoami'])
    expect(peekSession('agent-1').lines).toEqual([])
  })
})
