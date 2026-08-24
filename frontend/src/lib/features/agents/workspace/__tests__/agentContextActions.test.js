import { describe, expect, it, vi } from 'vitest'
import { buildAgentContextSections, buildCommandCategories } from '../agentContextActions.js'

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
