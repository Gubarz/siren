import { beforeEach, describe, expect, it, vi } from 'vitest'

const stores = vi.hoisted(() => ({
  sessions: { data: [], acquire: vi.fn(), release: vi.fn() },
  beacons: { data: [], acquire: vi.fn(), release: vi.fn() },
  selection: { agents: new Set() },
  agentTabs: { panes: {} },
}))

vi.mock('$stores/resources/sessions.svelte.js', () => ({ sessions: stores.sessions }))
vi.mock('$stores/resources/beacons.svelte.js', () => ({ beacons: stores.beacons }))
vi.mock('$stores/ui/selection.svelte.js', () => ({ selection: stores.selection }))
vi.mock('$stores/agentTabs.svelte.js', () => ({ agentTabs: stores.agentTabs }))

import { getActions } from '../actions.js'
import { registerCommandActions } from '../registry.js'

describe('palette live-agent gating', () => {
  beforeEach(() => {
    stores.sessions.data = []
    stores.beacons.data = []
    stores.selection.agents = new Set()
    registerCommandActions([
      { id: 'test-cmd-live', label: 'Run: whoami', requiresLiveAgents: true, on: vi.fn() },
      { id: 'test-cmd-server', label: 'Run: jobs', on: vi.fn() },
    ])
  })

  function ids() {
    return getActions().map((action) => action.id)
  }

  it('hides live-agent command actions when nothing is selected', () => {
    expect(ids()).not.toContain('test-cmd-live')
  })

  it('keeps server command actions available when nothing is selected', () => {
    expect(ids()).toContain('test-cmd-server')
  })

  it('hides live-agent command actions when only a lost id is selected', () => {
    stores.sessions.data = [{ ID: 'live-1' }]
    stores.selection.agents = new Set(['lost-1'])
    expect(ids()).not.toContain('test-cmd-live')
  })

  it('keeps server command actions available when only a lost id is selected', () => {
    stores.sessions.data = [{ ID: 'live-1' }]
    stores.selection.agents = new Set(['lost-1'])
    expect(ids()).toContain('test-cmd-server')
  })

  it('shows live-agent command actions when a live session is selected', () => {
    stores.sessions.data = [{ ID: 'live-1' }]
    stores.selection.agents = new Set(['live-1'])
    expect(ids()).toContain('test-cmd-live')
  })

  it('shows live-agent command actions when a live beacon is selected', () => {
    stores.beacons.data = [{ ID: 'beacon-1' }]
    stores.selection.agents = new Set(['beacon-1'])
    expect(ids()).toContain('test-cmd-live')
  })
})
