import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../api/console.js', () => ({
  CloseShell: vi.fn(() => Promise.resolve()),
  StartShell: vi.fn(() => Promise.resolve({ id: 'shell-1' })),
}))

vi.mock('../ui/dialog.svelte.js', () => ({
  dialog: {
    alert: vi.fn(() => Promise.resolve()),
    confirm: vi.fn(() => Promise.resolve(true)),
  },
}))

describe('agentTabs store', () => {
  let agentTabs
  let CloseShell
  let StartShell
  let dialog
  let config

  beforeEach(async () => {
    vi.resetModules()
    ;({ CloseShell, StartShell } = await import('../../api/console.js'))
    ;({ dialog } = await import('../ui/dialog.svelte.js'))
    ;({ config } = await import('../config.svelte.js'))
    ;({ agentTabs } = await import('../agentTabs.svelte.js'))
    vi.clearAllMocks()
  })

  it('does not create duplicate IDs when moving a duplicated tab back into a pane', () => {
    const sessionID = 'abcdef123456'
    const tabID = `${sessionID}-console`

    agentTabs.openTab(sessionID, 'console', 'left')
    agentTabs.openTab(sessionID, 'console', 'right')
    agentTabs.moveTab(tabID, 'right', 'left')

    const state = getState()
    expect(state.panes.left.tabs.map((tab) => tab.id)).toEqual([tabID])
    expect(state.panes.left.activeTabId).toBe(tabID)
    expect(state.panes.right).toBeUndefined()
  })

  it('keeps a shell alive while another pane still has that shell tab', () => {
    const sessionID = 'abcdef123456'
    const tabID = `${sessionID}-shell-1`

    agentTabs.registerShell({ id: 'shell-1' })
    agentTabs.openTab(sessionID, 'shell-1', 'left')
    agentTabs.openTab(sessionID, 'shell-1', 'right')

    agentTabs.closeTab('right', tabID)
    expect(CloseShell).not.toHaveBeenCalled()

    agentTabs.closeTab('left', tabID)
    expect(CloseShell).toHaveBeenCalledWith('shell-1')
  })

  it('keeps the active tab pointed at an existing tab after close', () => {
    const sessionID = 'abcdef123456'

    agentTabs.openTab(sessionID, 'console', 'left')
    agentTabs.openTab(sessionID, 'tasks', 'left')
    agentTabs.closeTab('left', `${sessionID}-tasks`)

    const state = getState()
    expect(state.panes.left.activeTabId).toBe(`${sessionID}-console`)
  })

  it('collapses back to the left pane when the only tab moves right', () => {
    const sessionID = 'abcdef123456'

    agentTabs.openTab(sessionID, 'console', 'left')
    agentTabs.moveTab(`${sessionID}-console`, 'left', 'right')

    const state = getState()
    expect(state.panes.left.tabs.map((tab) => tab.type)).toEqual(['console'])
    expect(state.panes.left.activeTabId).toBe(`${sessionID}-console`)
    expect(state.panes.right).toBeUndefined()
    expect(state.focusPane).toBe('left')
  })

  it('reorders a tab to the end of the same pane', () => {
    const sessionID = 'abcdef123456'

    agentTabs.openTab(sessionID, 'console', 'left')
    agentTabs.openTab(sessionID, 'tasks', 'left')
    agentTabs.openTab(sessionID, 'fileBrowser', 'left')
    agentTabs.reorderTab('left', `${sessionID}-console`, 3)

    expect(getState().panes.left.tabs.map((tab) => tab.type)).toEqual([
      'tasks',
      'fileBrowser',
      'console',
    ])
  })

  it('moves a tab across panes at the requested drop index', () => {
    const sessionID = 'abcdef123456'

    agentTabs.openTab(sessionID, 'console', 'left')
    agentTabs.openTab(sessionID, 'tasks', 'left')
    agentTabs.openTab(sessionID, 'fileBrowser', 'right')
    agentTabs.moveTab(`${sessionID}-tasks`, 'left', 'right', 0)

    expect(getState().panes.right.tabs.map((tab) => tab.type)).toEqual([
      'tasks',
      'fileBrowser',
    ])
  })

  it('collapses right-pane tabs back left when the last left tab moves right', () => {
    const sessionID = 'abcdef123456'

    agentTabs.openTab(sessionID, 'console', 'left')
    agentTabs.openTab(sessionID, 'fileBrowser', 'right')
    agentTabs.moveTab(`${sessionID}-console`, 'left', 'right')

    const state = getState()
    expect(state.panes.left.tabs.map((tab) => tab.type)).toEqual(['fileBrowser', 'console'])
    expect(state.panes.left.activeTabId).toBe(`${sessionID}-console`)
    expect(state.panes.right).toBeUndefined()
    expect(state.focusPane).toBe('left')
  })

  it('selects a lower tab by index in the focused pane', () => {
    const sessionID = 'abcdef123456'

    agentTabs.openTab(sessionID, 'console', 'left')
    agentTabs.openTab(sessionID, 'tasks', 'left')
    agentTabs.openTab(sessionID, 'fileBrowser', 'right')
    agentTabs.openTab(sessionID, 'processExplorer', 'right')

    agentTabs.selectFocusedTabByIndex(0)

    let state = getState()
    expect(state.focusPane).toBe('right')
    expect(state.panes.right.activeTabId).toBe(`${sessionID}-fileBrowser`)

    agentTabs.selectTab('left', `${sessionID}-console`)
    agentTabs.selectFocusedTabByIndex(1)

    state = getState()
    expect(state.focusPane).toBe('left')
    expect(state.panes.left.activeTabId).toBe(`${sessionID}-tasks`)
  })

  it('can focus a pane without selecting a specific tab first', () => {
    const sessionID = 'abcdef123456'

    agentTabs.openTab(sessionID, 'console', 'left')
    agentTabs.openTab(sessionID, 'tasks', 'left')
    agentTabs.openTab(sessionID, 'fileBrowser', 'right')

    agentTabs.setFocusPane('left')
    agentTabs.selectFocusedTabByIndex(1)

    let state = getState()
    expect(state.focusPane).toBe('left')
    expect(state.panes.left.activeTabId).toBe(`${sessionID}-tasks`)

    agentTabs.setFocusPane('right')
    agentTabs.selectFocusedTabByIndex(0)

    state = getState()
    expect(state.focusPane).toBe('right')
    expect(state.panes.right.activeTabId).toBe(`${sessionID}-fileBrowser`)
  })

  it('moves the focused lower tab between panes', () => {
    const sessionID = 'abcdef123456'

    agentTabs.openTab(sessionID, 'console', 'left')
    agentTabs.openTab(sessionID, 'tasks', 'left')
    agentTabs.openTab(sessionID, 'fileBrowser', 'right')
    agentTabs.setFocusPane('left')
    agentTabs.selectFocusedTabByIndex(1)

    agentTabs.moveFocusedTabToPane('right')

    let state = getState()
    expect(state.focusPane).toBe('right')
    expect(state.panes.left.tabs.map((tab) => tab.type)).toEqual(['console'])
    expect(state.panes.right.tabs.map((tab) => tab.type)).toEqual(['fileBrowser', 'tasks'])
    expect(state.panes.right.activeTabId).toBe(`${sessionID}-tasks`)

    agentTabs.moveFocusedTabToPane('left')

    state = getState()
    expect(state.focusPane).toBe('left')
    expect(state.panes.left.tabs.map((tab) => tab.type)).toEqual(['console', 'tasks'])
    expect(state.panes.right.tabs.map((tab) => tab.type)).toEqual(['fileBrowser'])
  })

  it('selects the tenth lower tab by zero-based index', () => {
    for (let i = 1; i <= 10; i++) {
      agentTabs.openTab(`session-${i}`, 'console', 'left')
    }

    agentTabs.selectFocusedTabByIndex(9)

    expect(getState().panes.left.activeTabId).toBe('session-10-console')
  })

  it('coalesces duplicate shell launches while the first launch is pending', async () => {
    const sessionID = 'abcdef123456'
    let resolveShell
    StartShell.mockImplementationOnce(() => new Promise((resolve) => {
      resolveShell = resolve
    }))

    const first = agentTabs.launchShell(sessionID, '')
    const second = agentTabs.launchShell(sessionID, '')
    await waitForShellStart()

    expect(StartShell).toHaveBeenCalledTimes(1)

    resolveShell({ id: 'shell-1', sessionID })
    const [firstShell, secondShell] = await Promise.all([first, second])

    expect(firstShell).toEqual(expect.objectContaining({ id: 'shell-1', sessionID }))
    expect(secondShell).toEqual(expect.objectContaining({ id: 'shell-1', sessionID }))
    expect(getState().panes.left.tabs.map((tab) => tab.type)).toEqual(['shell-1'])
  })

  it('normalizes shell info returned with Go-style field names', async () => {
    const sessionID = 'abcdef123456'
    StartShell.mockResolvedValueOnce({ ID: 'shell-99', SessionID: sessionID, PTY: true })

    const shell = await agentTabs.launchShell(sessionID, '')

    expect(shell.id).toBe('shell-99')
    expect(shell.sessionID).toBe(sessionID)
    expect(shell.pty).toBe(true)
    expect(getState().shellsByID['shell-99']).toEqual(expect.objectContaining({ id: 'shell-99' }))
    expect(getState().panes.left.tabs.map((tab) => tab.type)).toEqual(['shell-99'])
  })

  it('does not start a shell when the adult check is declined', async () => {
    dialog.confirm.mockResolvedValueOnce(false)

    const shell = await agentTabs.launchShell('abcdef123456', '')

    expect(shell).toBeNull()
    expect(dialog.confirm).toHaveBeenCalledWith(
      'This is not opsec safe. Are you an adult?',
      'Start Shell',
    )
    expect(StartShell).not.toHaveBeenCalled()
  })

  it('skips the adult check when disabled in settings', async () => {
    config.set('confirmShellAdult', false)

    await agentTabs.launchShell('abcdef123456', '')

    expect(dialog.confirm).not.toHaveBeenCalled()
    expect(StartShell).toHaveBeenCalledTimes(1)
  })

  it('reuses a shell for duplicate launch requests that arrive just after creation', async () => {
    vi.useFakeTimers()
    try {
      const sessionID = 'abcdef123456'

      const firstShell = await agentTabs.launchShell(sessionID, '')
      const secondShell = await agentTabs.launchShell(sessionID, '')

      expect(StartShell).toHaveBeenCalledTimes(1)
      expect(secondShell).toEqual(firstShell)
      expect(getState().panes.left.tabs.map((tab) => tab.type)).toEqual(['shell-1'])

      await vi.advanceTimersByTimeAsync(2001)
      await agentTabs.launchShell(sessionID, '')

      expect(StartShell).toHaveBeenCalledTimes(2)
    } finally {
      vi.useRealTimers()
    }
  })

  function getState() {
    return agentTabs.state
  }

  async function waitForShellStart() {
    for (let i = 0; i < 10 && StartShell.mock.calls.length === 0; i++) {
      await Promise.resolve()
    }
  }
})
