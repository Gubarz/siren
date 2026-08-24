import { describe, expect, it } from 'vitest'
import { buildCollectionOptions, COLLECTION_METHODS } from '../collectionOptions.js'

describe('buildCollectionOptions', () => {
  it('sanitizes a full form into wire options', () => {
    const opts = buildCollectionOptions({
      collector: 'SharpHound',
      methods: ['Default', 'Session'],
      flags: '--Stealth --Loop',
      domain: 'corp.local',
      timeoutMinutes: 15,
      ingest: true,
      loot: true,
    })
    expect(opts).toEqual({
      collector: 'sharphound',
      methods: ['Default', 'Session'],
      flags: ['--Stealth'],
      domain: 'corp.local',
      timeoutSeconds: 900,
      ingest: true,
      loot: true,
    })
  })

  it('defaults empty methods to Default', () => {
    const opts = buildCollectionOptions({ methods: [], flags: '' })
    expect(opts.methods).toEqual(['Default'])
    expect(opts.flags).toEqual([])
  })

  it('clamps timeout to 1 minute..1 hour in seconds', () => {
    expect(buildCollectionOptions({ timeoutMinutes: 0 }).timeoutSeconds).toBe(60)
    expect(buildCollectionOptions({ timeoutMinutes: 9999 }).timeoutSeconds).toBe(216000)
    expect(buildCollectionOptions({ timeoutMinutes: 'garbage' }).timeoutSeconds).toBe(900)
  })

  it('defaults ingest and loot to true', () => {
    const opts = buildCollectionOptions({})
    expect(opts.ingest).toBe(true)
    expect(opts.loot).toBe(true)
    expect(buildCollectionOptions({ ingest: false }).ingest).toBe(false)
  })

  it('exposes the method catalog', () => {
    expect(COLLECTION_METHODS).toContain('Default')
    expect(COLLECTION_METHODS).toContain('CertServices')
  })
})
