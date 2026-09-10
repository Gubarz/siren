// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'

const mocks = vi.hoisted(() => ({
  sessions: { data: [], acquire: vi.fn(), release: vi.fn(), refresh: vi.fn() },
  beacons: { data: [], acquire: vi.fn(), release: vi.fn(), refresh: vi.fn() },
  agentColors: { data: {}, acquire: vi.fn(), release: vi.fn(), refresh: vi.fn() },
  knownAgents: { data: [], load: vi.fn(), lost: [] },
}))

vi.mock('$stores/resources/sessions.svelte.js', () => ({ sessions: mocks.sessions }))
vi.mock('$stores/resources/beacons.svelte.js', () => ({ beacons: mocks.beacons }))
vi.mock('$stores/resources/agentColors.svelte.js', () => ({ agentColors: mocks.agentColors }))
vi.mock('$stores/knownAgents.svelte.js', () => ({ knownAgents: mocks.knownAgents }))

import AgentGlobalMenu from '../AgentGlobalMenu.svelte'
import { selection } from '$stores/ui/selection.svelte.js'
import { contextMenu } from '$stores/ui/contextMenu.svelte.js'
import { commandModal } from '$stores/ui/commandModal.svelte.js'

const liveSession = {
  ID: 'live-1',
  Name: 'alive',
  Hostname: 'LIVE-PC',
  Username: 'CORP\\live',
  OS: 'windows',
  RemoteAddress: '10.0.0.1:4444',
  Transport: 'mtls',
}

const lostRecord = {
  id: 'lost-1',
  kind: 'session',
  name: 'ghost',
  hostname: 'GONE-PC',
  username: 'CORP\\ghost',
  os: 'windows',
  status: 'lost',
  lastSeen: 2000,
}

const categories = [
  { category: 'Sliver', commands: [{ name: 'whoami', description: 'who am i' }] },
]

function flattenItems(sections) {
  const items = []
  for (const section of sections) {
    for (const item of section.items ?? []) {
      items.push(item)
      for (const child of item.children ?? []) items.push(child)
    }
  }
  return items
}

function findItem(sections, label) {
  return flattenItems(sections).find((item) => item.label === label)
}

async function clickMenu(name) {
  await fireEvent.click(screen.getByRole('button', { name }))
}

describe('AgentGlobalMenu lost gating', () => {
  beforeEach(() => {
    mocks.sessions.data = []
    mocks.beacons.data = []
    mocks.knownAgents.data = []
    selection.clear()
    contextMenu.close()
    contextMenu.sections = []
    commandModal.close()
  })

  afterEach(cleanup)

  it('gives a lost-only selection the reduced history menu with no live actions', async () => {
    mocks.knownAgents.data = [lostRecord]
    selection.replace({ agents: ['lost-1'] })
    render(AgentGlobalMenu, { props: { categories } })

    await clickMenu(/actions/i)

    const labels = flattenItems(contextMenu.sections).map((item) => item.label)
    expect(labels).toContain('Tasks')
    expect(labels).toContain('Remove')
    expect(labels).not.toContain('Console')
    expect(labels).not.toContain('New Shell')
    expect(labels).not.toContain('Kill Agent')
    expect(labels).not.toContain('Discovery')

    for (const item of flattenItems(contextMenu.sections)) {
      if (item.children || item.disabled || !item.on) continue
      expect(() => item.on()).not.toThrow()
    }
  })

  it('hides command targets for a lost-only selection', async () => {
    mocks.knownAgents.data = [lostRecord]
    selection.replace({ agents: ['lost-1'] })
    render(AgentGlobalMenu, { props: { categories } })

    await clickMenu(/commands/i)

    const items = contextMenu.sections[0]?.items ?? []
    expect(items).toEqual([{ label: 'No live agents selected', disabled: true }])
    expect(findItem(contextMenu.sections, 'whoami')).toBeUndefined()
  })

  it('routes only live ids to command targets for a mixed selection', async () => {
    mocks.sessions.data = [liveSession]
    mocks.knownAgents.data = [lostRecord]
    selection.replace({ agents: ['live-1', 'lost-1'] })
    render(AgentGlobalMenu, { props: { categories } })

    await clickMenu(/commands/i)
    const command = findItem(contextMenu.sections, 'whoami')
    expect(command).toBeTruthy()
    command.on()
    expect(commandModal.targetIDs).toEqual(['live-1'])
  })

  it('keeps live actions and excludes lost ids in the mixed Actions menu', async () => {
    mocks.sessions.data = [liveSession]
    mocks.knownAgents.data = [lostRecord]
    selection.replace({ agents: ['live-1', 'lost-1'] })
    render(AgentGlobalMenu, { props: { categories } })

    await clickMenu(/actions/i)

    const labels = flattenItems(contextMenu.sections).map((item) => item.label)
    expect(labels).toContain('Kill Agent')
    expect(labels).not.toContain('Remove')

    const command = findItem(contextMenu.sections, 'whoami')
    expect(command).toBeTruthy()
    command.on()
    expect(commandModal.targetIDs).toEqual(['live-1'])
  })
})
