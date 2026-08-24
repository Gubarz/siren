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
