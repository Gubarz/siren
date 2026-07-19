import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../../api/runtime.js', () => ({
  onSliverEvent: vi.fn(() => vi.fn()),
  onWailsEvent: vi.fn(() => vi.fn()),
}))

describe('createResource', () => {
  let createResource

  async function flushPromises() {
    await Promise.resolve()
    await Promise.resolve()
  }

  beforeEach(async () => {
    vi.clearAllMocks()
    createResource = (await import('../createResource.svelte.js')).createResource
  })

  it('starts with empty data', () => {
    const resource = createResource({ fetch: () => [] })
    expect(resource.data).toEqual([])
    expect(resource.loading).toBe(false)
    expect(resource.error).toBeNull()
  })

  it('sets data on successful fetch', async () => {
    const fetch = vi.fn().mockResolvedValue(['item1', 'item2'])
    const resource = createResource({ fetch })
    await resource.refresh()
    expect(resource.data).toEqual(['item1', 'item2'])
    expect(resource.loading).toBe(false)
    expect(resource.error).toBeNull()
  })

  it('sets error on failed fetch', async () => {
    const fetch = vi.fn().mockRejectedValue(new Error('network error'))
    const resource = createResource({ fetch })
    await resource.refresh()
    expect(resource.data).toEqual([])
    expect(resource.loading).toBe(false)
    expect(resource.error).toBe('Error: network error')
  })

  it('deduplicates concurrent fetches', async () => {
    let callCount = 0
    const fetch = vi.fn().mockImplementation(async () => {
      callCount++
      return ['data']
    })
    const resource = createResource({ fetch })
    await Promise.all([resource.refresh(), resource.refresh(), resource.refresh()])
    expect(callCount).toBe(1)
  })

  it('retries initial load for later acquirers when no fetch has succeeded', async () => {
    const fetch = vi.fn()
      .mockRejectedValueOnce(new Error('not connected'))
      .mockResolvedValueOnce(['agent-1'])
    const resource = createResource({ fetch })

    resource.acquire()
    await flushPromises()
    expect(fetch).toHaveBeenCalledTimes(1)

    resource.acquire()
    await flushPromises()

    expect(fetch).toHaveBeenCalledTimes(2)
    expect(resource.data).toEqual(['agent-1'])
    expect(resource.error).toBeNull()

    resource.release()
    resource.release()
  })
})
