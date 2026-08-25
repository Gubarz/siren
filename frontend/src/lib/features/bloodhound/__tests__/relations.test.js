import { describe, expect, it } from 'vitest'
import { uniqueEntities, isComputerKind, sessionHeading, adminHeading, toActionEntity } from '../relations.js'

describe('uniqueEntities', () => {
  it('dedupes nodes by id preserving first-seen order', () => {
    const graph = {
      nodes: [
        { id: 'u1', label: 'jane' },
        { id: 'g1', label: 'IT ADMINS' },
        { id: 'u1', label: 'jane-dupe' },
      ],
      edges: [],
    }
    const out = uniqueEntities(graph)
    expect(out.map((n) => n.id)).toEqual(['u1', 'g1'])
    expect(out[0].label).toBe('jane')
  })

  it('drops the excluded entity from its own relation lists', () => {
    const graph = { nodes: [{ id: 'u1', label: 'jane' }, { id: 'g1', label: 'IT ADMINS' }] }
    expect(uniqueEntities(graph, 'u1').map((n) => n.id)).toEqual(['g1'])
    expect(uniqueEntities(graph, 'missing-id').map((n) => n.id)).toEqual(['u1', 'g1'])
  })

  it('ignores blank excludeId values', () => {
    const graph = { nodes: [{ id: 'u1', label: 'jane' }] }
    expect(uniqueEntities(graph, '').map((n) => n.id)).toEqual(['u1'])
  })

  it('tolerates null graphs and malformed nodes', () => {
    expect(uniqueEntities(null)).toEqual([])
    expect(uniqueEntities({})).toEqual([])
    expect(uniqueEntities({ nodes: 'nope' })).toEqual([])
    expect(uniqueEntities({ nodes: [null, {}, { id: 'x' }] })).toEqual([{ id: 'x' }])
  })
})

describe('toActionEntity', () => {
  it('prefers the explicit objectId over the node id', () => {
    const out = toActionEntity({ id: 'n1', objectId: 'S-1-5-21-9', label: 'jane@corp.local', kind: 'User' })
    expect(out.objectId).toBe('S-1-5-21-9')
    expect(out.id).toBe('n1')
  })

  it('falls back to the node id when objectId is missing', () => {
    expect(toActionEntity({ id: 'n2', label: 'IT ADMINS' }).objectId).toBe('n2')
    expect(toActionEntity({ id: 'n2', objectId: '', label: 'IT ADMINS' }).objectId).toBe('n2')
  })

  it('derives name from label and preserves the rest of the node', () => {
    const out = toActionEntity({ id: 'n3', label: 'DC01', kind: 'Computer', tierZero: true, owned: false })
    expect(out.name).toBe('DC01')
    expect(out.kind).toBe('Computer')
    expect(out.tierZero).toBe(true)
    expect(out.owned).toBe(false)
  })

  it('tolerates null nodes', () => {
    expect(toActionEntity(null)).toEqual({ objectId: '', name: '' })
  })
})

describe('kind helpers', () => {
  it('treats only exact Computer kind as host-centric', () => {
    expect(isComputerKind('Computer')).toBe(true)
    expect(isComputerKind('computer')).toBe(false)
    expect(isComputerKind('User')).toBe(false)
    expect(isComputerKind(undefined)).toBe(false)
  })

  it('selects host-centric copy for computers', () => {
    expect(sessionHeading('Computer')).toBe('Users with sessions on this host')
    expect(adminHeading('Computer')).toBe('Local admins of this host')
  })

  it('selects principal-centric copy otherwise', () => {
    expect(sessionHeading('User')).toBe('Sessions')
    expect(adminHeading('Group')).toBe('Local admin on')
  })
})
