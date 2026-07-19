import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { toast } from '../toast.svelte.js'

describe('toast store', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    toast.clear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('starts empty', () => {
    expect(toast.items).toEqual([])
  })

  it('adds a toast with defaults', () => {
    const id = toast.push({ message: 'hello' })
    expect(toast.items).toHaveLength(1)
    expect(toast.items[0].id).toBe(id)
    expect(toast.items[0].message).toBe('hello')
    expect(toast.items[0].variant).toBe('info')
  })

  it('adds a toast with custom variant', () => {
    toast.push({ message: 'error!', variant: 'danger' })
    expect(toast.items[0].variant).toBe('danger')
  })

  it('dismisses a toast by id', () => {
    const id = toast.push({ message: 'test' })
    expect(toast.items).toHaveLength(1)
    toast.dismiss(id)
    expect(toast.items).toHaveLength(0)
  })

  it('dismisses only the specified toast', () => {
    const id1 = toast.push({ message: 'one' })
    toast.push({ message: 'two' })
    toast.dismiss(id1)
    expect(toast.items).toHaveLength(1)
    expect(toast.items[0].message).toBe('two')
  })

  it('auto-dismisses after duration', () => {
    toast.push({ message: 'auto', duration: 500 })
    expect(toast.items).toHaveLength(1)
    vi.advanceTimersByTime(500)
    expect(toast.items).toHaveLength(0)
  })

  it('does not auto-dismiss when duration is 0', () => {
    toast.push({ message: 'sticky', duration: 0 })
    vi.advanceTimersByTime(10000)
    expect(toast.items).toHaveLength(1)
  })

  it('clears all toasts', () => {
    toast.push({ message: 'a' })
    toast.push({ message: 'b' })
    toast.clear()
    expect(toast.items).toEqual([])
  })
})
