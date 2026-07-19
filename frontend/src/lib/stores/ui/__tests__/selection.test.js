import { describe, it, expect, beforeEach } from 'vitest'
import { selection } from '../selection.svelte.js'

describe('selection store', () => {
  beforeEach(() => {
    selection.clear()
  })

  it('starts empty', () => {
    expect([...selection.agents]).toEqual([])
    expect([...selection.devices]).toEqual([])
  })

  it('replaces agents and devices', () => {
    selection.replace({ agents: ['a1', 'a2'], devices: ['d1'] })
    expect([...selection.agents]).toEqual(['a1', 'a2'])
    expect([...selection.devices]).toEqual(['d1'])
  })

  it('toggles agent selection', () => {
    selection.select('agent', 'a1')
    expect([...selection.agents]).toEqual(['a1'])

    selection.toggle('agent', 'a1')
    expect([...selection.agents]).toEqual([])

    selection.toggle('agent', 'a2')
    expect([...selection.agents]).toEqual(['a2'])
  })

  it('toggles device selection', () => {
    selection.select('device', 'd1')
    expect([...selection.devices]).toEqual(['d1'])
  })

  it('select replaces existing selection when not additive', () => {
    selection.select('agent', 'a1')
    selection.select('agent', 'a2')
    expect([...selection.agents]).toEqual(['a2'])
  })

  it('select adds when additive is true', () => {
    selection.select('agent', 'a1')
    selection.select('agent', 'a2', true)
    expect([...selection.agents]).toEqual(['a1', 'a2'])
  })

  it('clear removes everything', () => {
    selection.replace({ agents: ['a1', 'a2'], devices: ['d1', 'd2'] })
    selection.clear()
    expect([...selection.agents]).toEqual([])
    expect([...selection.devices]).toEqual([])
  })
})
