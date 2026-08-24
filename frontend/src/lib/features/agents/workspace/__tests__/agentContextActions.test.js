import { describe, expect, it, vi } from 'vitest'
import { buildAgentContextSections } from '../agentContextActions.js'

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
