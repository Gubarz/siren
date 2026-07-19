import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('config store', () => {
  let config

  beforeEach(async () => {
    vi.stubGlobal('localStorage', {
      _data: {},
      getItem(key) { return this._data[key] ?? null },
      setItem(key, val) { this._data[key] = String(val) },
      removeItem(key) { delete this._data[key] },
    })
    config = (await import('../config.svelte.js')).config
    config.reset()
  })

  it('starts with defaults', () => {
    expect(config.theme).toBe('dark')
    expect(config.agentViewMode).toBe('table')
  })

  it('sets a value', () => {
    config.set('agentViewMode', 'graph')
    expect(config.agentViewMode).toBe('graph')
  })

  it('persists to localStorage', () => {
    config.set('theme', 'hacker')
    const saved = JSON.parse(localStorage.getItem('sliver-config'))
    expect(saved.theme).toBe('hacker')
  })

  it('resets to defaults', () => {
    config.set('topPaneHeight', 80)
    config.reset()
    expect(config.topPaneHeight).toBe(50)
  })

  it('does not mutate other keys when setting one', () => {
    config.set('theme', 'light')
    config.set('agentViewMode', 'graph')
    expect(config.theme).toBe('light')
    expect(config.agentViewMode).toBe('graph')
    expect(config.topPaneHeight).toBe(50)
  })
})
