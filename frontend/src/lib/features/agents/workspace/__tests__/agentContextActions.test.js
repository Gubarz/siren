import { describe, expect, it, vi } from 'vitest'
import { buildAgentContextSections, buildAgentActionsSections, buildCommandCategories } from '../agentContextActions.js'

function sectionItems(sections, label) {
  for (const section of sections) {
    for (const item of section.items ?? []) {
      if (item.label === label) return item
      for (const child of item.children ?? []) {
        if (child.label === label) return child
      }
    }
  }
  return null
}

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

function baseCtx(overrides = {}) {
  const agentTabs = { openTab: vi.fn() }
  return {
    agent: { ID: 'a1', OS: 'windows', Name: 'impl', _kind: 'session', Hostname: 'PC1', Username: 'CORP\\jane' },
    isBeacon: false,
    isWindows: true,
    catalog: [],
    targetIDs: ['a1'],
    targetAgents: [],
    agentTabs,
    automationRules: [],
    contextMenuHandlers: {
      findAttackPaths: (agents) => agents.forEach((a) => agentTabs.openTab(a.ID, 'bloodhound')),
    },
    ...overrides,
  }
}

describe('agent context actions — find attack paths', () => {
  it('opens one bloodhound tab per selected agent', () => {
    const ctx = baseCtx({
      targetAgents: [
        { ID: 'a1', OS: 'windows', Name: 'impl1', _kind: 'session', Hostname: 'PC1', Username: 'CORP\\jane' },
        { ID: 'a2', OS: 'windows', Name: 'impl2', _kind: 'session', Hostname: 'PC2', Username: 'CORP\\bob' },
      ],
    })
    const sections = buildAgentContextSections(ctx)
    const item = sectionItems(sections, 'Find attack paths (2)')
    expect(item).not.toBeNull()
    expect(item.disabled).toBeFalsy()
    item.on()
    expect(ctx.agentTabs.openTab).toHaveBeenCalledTimes(2)
    expect(ctx.agentTabs.openTab).toHaveBeenCalledWith('a1', 'bloodhound')
    expect(ctx.agentTabs.openTab).toHaveBeenCalledWith('a2', 'bloodhound')
  })

  it('shows the single-agent label without a count', () => {
    const ctx = baseCtx()
    const sections = buildAgentContextSections(ctx)
    const item = sectionItems(sections, 'Find attack paths')
    expect(item).not.toBeNull()
    item.on()
    expect(ctx.agentTabs.openTab).toHaveBeenCalledWith('a1', 'bloodhound')
  })

  it('disables the action when no handler is provided', () => {
    const ctx = baseCtx()
    delete ctx.contextMenuHandlers.findAttackPaths
    const sections = buildAgentContextSections(ctx)
    const item = sectionItems(sections, 'Find attack paths')
    expect(item.disabled).toBeTruthy()
  })
})

describe('buildCommandCategories', () => {
  const catalog = [
    { category: 'Sliver', commands: [{ name: 'whoami', description: 'who am i' }] },
    { category: 'Empty', commands: [] },
  ]

  it('builds nested items and drops empty categories', () => {
    const execute = vi.fn()
    const items = buildCommandCategories({ catalog, targetIDs: ['a1'], executeAgentCommand: execute })
    expect(items).toHaveLength(1)
    expect(items[0]).toMatchObject({ icon: 'command', label: 'Sliver' })
    const child = items[0].children[0]
    expect(child.label).toBe('whoami')
    child.on()
    expect(execute).toHaveBeenCalledWith({ name: 'whoami', description: 'who am i' }, ['a1'])
  })

  it('applies disabled from the isDisabled callback', () => {
    const items = buildCommandCategories({
      catalog,
      targetIDs: [],
      executeAgentCommand: vi.fn(),
      isDisabled: (cmd) => cmd.name === 'whoami',
    })
    expect(items[0].children[0].disabled).toBe(true)
  })

  it('leaves items enabled without isDisabled', () => {
    const items = buildCommandCategories({ catalog, targetIDs: [], executeAgentCommand: vi.fn() })
    expect(items[0].children[0].disabled).toBe(false)
  })
})

describe('buildAgentActionsSections', () => {
  function actionsCtx(overrides = {}) {
    return {
      agent: { ID: 'a1', OS: 'windows', Name: 'impl', _kind: 'session', Hostname: 'PC1' },
      isBeacon: false,
      isWindows: true,
      hasInteractiveSession: false,
      targetAgents: [],
      agentTabs: { openTab: vi.fn(), launchShell: vi.fn() },
      openBeaconDetail: vi.fn(),
      promoteBeacon: vi.fn(),
      demoteSession: vi.fn(),
      newShell: vi.fn(),
      findAttackPaths: vi.fn(),
      ...overrides,
    }
  }

  it('hides windows-only tabs for a linux session', () => {
    const linux = { ID: 'a1', OS: 'linux', Name: 'lin', _kind: 'session', Hostname: 'PC1' }
    const sections = buildAgentActionsSections(actionsCtx({
      agent: linux,
      isWindows: false,
      targetAgents: [linux],
    }))
    const labels = flattenItems(sections).map((i) => i.label)
    expect(labels).toContain('Console')
    expect(labels).toContain('New Shell')
    expect(labels).not.toContain('Services')
    expect(labels).not.toContain('Registry')
    expect(labels).not.toContain('Privileges')
  })

  it('shows windows-only tabs for a windows session', () => {
    const sections = buildAgentActionsSections(actionsCtx({
      targetAgents: [{ ID: 'a1', OS: 'windows', Name: 'impl', _kind: 'session', Hostname: 'PC1' }],
    }))
    const labels = flattenItems(sections).map((i) => i.label)
    expect(labels).toContain('Services')
    expect(labels).toContain('Registry')
    expect(labels).toContain('Privileges')
  })

  it('uses terminal-square for New Shell', () => {
    const sections = buildAgentActionsSections(actionsCtx({
      targetAgents: [{ ID: 'a1', OS: 'windows', Name: 'impl', _kind: 'session', Hostname: 'PC1' }],
    }))
    expect(sectionItems(sections, 'New Shell').icon).toBe('terminal-square')
  })

  it('keeps Tunnels in the More submenu, not the top level', () => {
    const sections = buildAgentActionsSections(actionsCtx({
      targetAgents: [{ ID: 'a1', OS: 'windows', Name: 'impl', _kind: 'session', Hostname: 'PC1' }],
    }))
    const topLabels = (sections[0]?.items ?? []).map((i) => i.label)
    const moreChildren = sections[1]?.items?.[0]?.children ?? []
    expect(topLabels).not.toContain('Tunnels')
    expect(moreChildren.map((i) => i.label)).toContain('Tunnels')
  })

  it('disables every leaf item when there are no targets', () => {
    const sections = buildAgentActionsSections(actionsCtx({ agent: null, isWindows: false }))
    const leaves = flattenItems(sections).filter((i) => !i.children)
    expect(leaves.length).toBeGreaterThan(0)
    for (const item of leaves) {
      expect(item.disabled).toBeTruthy()
    }
  })

  it('builds beacon actions for a beacon context', () => {
    const beacon = { ID: 'b1', OS: 'windows', Name: 'b', _kind: 'beacon', Hostname: 'PC1' }
    const sections = buildAgentActionsSections(actionsCtx({
      agent: beacon,
      isBeacon: true,
      targetAgents: [beacon],
    }))
    const labels = flattenItems(sections).map((i) => i.label)
    expect(labels).toContain('Tasks')
    expect(labels).toContain('Beacon Detail…')
    expect(labels).toContain('Open Interactive Session')
    expect(labels).toContain('Close Interactive Session')
    expect(labels).not.toContain('New Shell')
  })
})

describe('danger actions — selection aware', () => {
  function dangerCtx(overrides = {}) {
    const killAgent = vi.fn()
    const killAgents = vi.fn()
    const removeBeaconRecord = vi.fn()
    const removeBeaconRecords = vi.fn()
    return {
      agent: { ID: 'b1', OS: 'windows', Name: 'beacon1', _kind: 'beacon', Hostname: 'PC1' },
      isBeacon: true,
      isWindows: true,
      catalog: [],
      targetIDs: ['b1'],
      targetAgents: [],
      agentTabs: { openTab: vi.fn() },
      automationRules: [],
      contextMenuHandlers: { killAgent, killAgents, removeBeaconRecord, removeBeaconRecords },
      ...overrides,
    }
  }

  it('keeps single-agent labels and handlers for one beacon', () => {
    const ctx = dangerCtx()
    const sections = buildAgentContextSections(ctx)
    const kill = sectionItems(sections, 'Kill Agent')
    const remove = sectionItems(sections, 'Remove Beacon Record')
    kill.on()
    remove.on()
    expect(ctx.contextMenuHandlers.killAgent).toHaveBeenCalledTimes(1)
    expect(ctx.contextMenuHandlers.killAgents).not.toHaveBeenCalled()
    expect(ctx.contextMenuHandlers.removeBeaconRecord).toHaveBeenCalledTimes(1)
    expect(ctx.contextMenuHandlers.removeBeaconRecords).not.toHaveBeenCalled()
  })

  it('labels and routes to bulk handlers when multiple beacons are selected', () => {
    const beacons = [
      { ID: 'b1', OS: 'windows', Name: 'beacon1', _kind: 'beacon', Hostname: 'PC1' },
      { ID: 'b2', OS: 'windows', Name: 'beacon2', _kind: 'beacon', Hostname: 'PC2' },
    ]
    const ctx = dangerCtx({
      targetIDs: ['b1', 'b2'],
      targetAgents: beacons,
    })
    const sections = buildAgentContextSections(ctx)
    expect(sectionItems(sections, 'Kill Agent (2)')).not.toBeNull()
    const remove = sectionItems(sections, 'Remove Beacon Record (2)')
    expect(remove).not.toBeNull()
    remove.on()
    expect(ctx.contextMenuHandlers.removeBeaconRecords).toHaveBeenCalledWith(beacons)
    expect(ctx.contextMenuHandlers.removeBeaconRecord).not.toHaveBeenCalled()
  })

  it('hides Remove Beacon Record when no beacon is in the selection', () => {
    const session = { ID: 's1', OS: 'linux', Name: 'sess', _kind: 'session', Hostname: 'PC1' }
    const ctx = dangerCtx({
      agent: session,
      isBeacon: false,
      isWindows: false,
      targetAgents: [session],
    })
    const labels = flattenItems(buildAgentContextSections(ctx)).map((i) => i.label)
    expect(labels).not.toContain('Remove Beacon Record')
    expect(labels).toContain('Kill Agent')
  })
})
