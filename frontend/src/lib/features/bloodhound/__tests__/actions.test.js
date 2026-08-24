import { describe, expect, it, vi } from 'vitest'
import { actionsForEntity, actionsForEdge } from '../actions.js'

function baseCtx(overrides = {}) {
  return {
    agent: { ID: 'a1', OS: 'windows', Hostname: 'PC1' },
    enrichment: { owned: true, tierZero: false },
    entity: { objectId: 'S-1-5-21-77', name: 'JANE@CORP.LOCAL', kind: 'User', owned: false },
    kerberoastableIDs: new Set(['S-1-5-21-77']),
    runCommand: vi.fn(),
    addToCase: { open: vi.fn() },
    openTags: vi.fn(),
    openComments: vi.fn(),
    ...overrides,
  }
}

describe('actionsForEntity', () => {
  it('enables kerberoast from an owned windows session', () => {
    const ctx = baseCtx()
    const actions = actionsForEntity(ctx)
    const roast = actions.find((a) => a.label.startsWith('Kerberoast'))
    expect(roast).toBeDefined()
    expect(roast.disabled).toBeFalsy()
    roast.on()
    expect(ctx.runCommand).toHaveBeenCalledWith('a1', 'kerberoast')
  })

  it('disables kerberoast without an owned session, with a reason', () => {
    const ctx = baseCtx({ enrichment: { owned: false } })
    const roast = actionsForEntity(ctx).find((a) => a.label.startsWith('Kerberoast'))
    expect(roast.disabled).toBeTruthy()
    expect(roast.reason).toBeTruthy()
    roast.on()
    expect(ctx.runCommand).not.toHaveBeenCalled()
  })

  it('omits kerberoast for non-kerberoastable users', () => {
    const ctx = baseCtx({ kerberoastableIDs: new Set() })
    expect(actionsForEntity(ctx).some((a) => a.label.startsWith('Kerberoast'))).toBe(false)
  })

  it('offers lateral movement for un-owned computers', () => {
    const ctx = baseCtx({
      entity: { objectId: 'S-1-5-21-9', name: 'SRV01.CORP.LOCAL', kind: 'Computer', owned: false },
      kerberoastableIDs: new Set(),
    })
    const move = actionsForEntity(ctx).find((a) => a.label.startsWith('Move to'))
    expect(move).toBeDefined()
    expect(move.disabled).toBeFalsy()
    move.on()
    expect(ctx.runCommand).toHaveBeenCalledWith('a1', 'psexec')
  })

  it('omits movement for already-owned computers', () => {
    const ctx = baseCtx({
      entity: { objectId: 'S-1-5-21-9', name: 'SRV01.CORP.LOCAL', kind: 'Computer', owned: true },
      kerberoastableIDs: new Set(),
    })
    expect(actionsForEntity(ctx).some((a) => a.label.startsWith('Move to'))).toBe(false)
  })

  it('adds tier-zero entities to cases', () => {
    const ctx = baseCtx({ enrichment: { owned: true, tierZero: true } })
    const toCase = actionsForEntity(ctx).find((a) => a.label === 'Add to case')
    expect(toCase).toBeDefined()
    toCase.on()
    expect(ctx.addToCase.open).toHaveBeenCalledWith({
      collection: 'bloodhound', itemID: 'S-1-5-21-77', label: 'JANE@CORP.LOCAL',
    })
  })

  it('always offers tag and comment', () => {
    const ctx = baseCtx({ kerberoastableIDs: new Set() })
    const actions = actionsForEntity(ctx)
    const tag = actions.find((a) => a.label === 'Tag')
    const comment = actions.find((a) => a.label === 'Comment')
    expect(tag).toBeDefined()
    expect(comment).toBeDefined()
    tag.on()
    expect(ctx.openTags).toHaveBeenCalledWith('bloodhound', 'S-1-5-21-77', 'JANE@CORP.LOCAL')
    comment.on()
    expect(ctx.openComments).toHaveBeenCalledWith('bloodhound', 'S-1-5-21-77', 'JANE@CORP.LOCAL')
  })

  it('returns no actions without an entity', () => {
    expect(actionsForEntity(baseCtx({ entity: null }))).toEqual([])
  })
})

describe('actionsForEdge', () => {
  it('offers movement on AdminTo / HasSession edges from an owned agent', () => {
    const ctx = baseCtx({
      entity: { objectId: 'S-1-5-21-9', name: 'SRV01.CORP.LOCAL', kind: 'Computer', owned: false },
      edge: { label: 'AdminTo', source: 'S-1-5-21-77', target: 'S-1-5-21-9' },
    })
    const move = actionsForEdge(ctx).find((a) => a.label.startsWith('Move to'))
    expect(move).toBeDefined()
    expect(move.disabled).toBeFalsy()
  })

  it('ignores non-movement edges', () => {
    const ctx = baseCtx({ edge: { label: 'MemberOf' } })
    expect(actionsForEdge(ctx)).toEqual([])
  })
})
