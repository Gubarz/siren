import { describe, expect, it } from 'vitest'
import { addBloodhoundOverlay, BH_W, BH_H } from '../bhOverlay.js'

const enrichment = {
  a1: {
    entity: { objectId: 'S-1-5-21-77', name: 'JANE@CORP.LOCAL', kind: 'User', owned: true },
    owned: true,
    tierZero: false,
    distanceToTierZero: 2,
    paths: [
      { id: 'S-1-5-21-77', label: 'JANE@CORP.LOCAL', kind: 'User', owned: true, tierZero: false },
      { id: 'S-1-5-21-999', label: 'DOMAIN ADMINS@CORP.LOCAL', kind: 'Group', owned: false, tierZero: false },
      { id: 'S-1-5-21-1', label: 'DC1.CORP.LOCAL', kind: 'Computer', owned: false, tierZero: true },
    ],
  },
}

const agents = [{ ID: 'a1' }]

describe('addBloodhoundOverlay', () => {
  it('adds an entity node and correlation edge per enriched agent', () => {
    const rawNodes = [{ id: 'a1', w: 256, h: 144, data: { variant: 'agent' } }]
    const rawEdges = []
    addBloodhoundOverlay(rawNodes, rawEdges, { agents, enrichment, showEdges: false, direction: 'TB' })

    expect(rawNodes).toHaveLength(2)
    const node = rawNodes.find((n) => n.id === 'bh_S-1-5-21-77')
    expect(node).toMatchObject({
      w: BH_W, h: BH_H,
      data: { variant: 'bloodhound', kind: 'User', owned: true, distance: 2 },
    })
    expect(rawEdges).toHaveLength(1)
    expect(rawEdges[0]).toMatchObject({ source: 'a1', target: 'bh_S-1-5-21-77' })
  })

  it('skips agents without enrichment', () => {
    const rawNodes = [{ id: 'a2', w: 256, h: 144, data: { variant: 'agent' } }]
    const rawEdges = []
    addBloodhoundOverlay(rawNodes, rawEdges, { agents: [{ ID: 'a2' }], enrichment, showEdges: false, direction: 'TB' })
    expect(rawNodes).toHaveLength(1)
    expect(rawEdges).toHaveLength(0)
  })

  it('renders the path chain when edges are shown', () => {
    const rawNodes = [{ id: 'a1', w: 256, h: 144, data: { variant: 'agent' } }]
    const rawEdges = []
    addBloodhoundOverlay(rawNodes, rawEdges, { agents, enrichment, showEdges: true, direction: 'TB' })

    const ids = rawNodes.map((n) => n.id)
    expect(ids).toContain('bh_S-1-5-21-77')
    expect(ids).toContain('bh_S-1-5-21-999')
    expect(ids).toContain('bh_S-1-5-21-1')

    // correlation edge + 3 chain edges
    expect(rawEdges).toHaveLength(4)
    const chainSources = rawEdges.map((e) => e.source)
    expect(chainSources).toContain('bh_S-1-5-21-77')
    expect(chainSources).toContain('bh_S-1-5-21-999')
  })

  it('dedupes entities shared by multiple agents', () => {
    const rawNodes = [{ id: 'a1', w: 256, h: 144, data: {} }, { id: 'a2', w: 256, h: 144, data: {} }]
    const rawEdges = []
    const both = [{ ID: 'a1' }, { ID: 'a2' }]
    const shared = { ...enrichment, a2: enrichment.a1 }
    addBloodhoundOverlay(rawNodes, rawEdges, { agents: both, enrichment: shared, showEdges: false, direction: 'TB' })

    expect(rawNodes.filter((n) => n.id === 'bh_S-1-5-21-77')).toHaveLength(1)
    expect(rawEdges.filter((e) => e.target === 'bh_S-1-5-21-77')).toHaveLength(2)
  })
})
