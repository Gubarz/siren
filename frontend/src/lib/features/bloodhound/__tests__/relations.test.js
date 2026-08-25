import { describe, expect, it } from 'vitest'
import { uniqueEntities, isComputerKind, sessionHeading, adminHeading } from '../relations.js'

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

  it('tolerates null graphs and malformed nodes', () => {
    expect(uniqueEntities(null)).toEqual([])
    expect(uniqueEntities({})).toEqual([])
    expect(uniqueEntities({ nodes: [null, {}, { id: 'x' }] })).toEqual([{ id: 'x' }])
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
