import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  getBloodHoundStatus,
  getBloodHoundConfig,
  correlateAgents,
  markBloodHoundOwned,
  unmarkBloodHoundOwned,
} from '../bloodhound.js'

const app = vi.hoisted(() => ({
  BloodHoundGetConfig: vi.fn(),
  BloodHoundSaveConfig: vi.fn(),
  BloodHoundTestConnection: vi.fn(),
  BloodHoundConnect: vi.fn(),
  BloodHoundDisconnect: vi.fn(),
  BloodHoundStatus: vi.fn(),
  BloodHoundCorrelate: vi.fn(),
  BloodHoundMarkOwned: vi.fn(),
  BloodHoundUnmarkOwned: vi.fn(),
}))
vi.mock('../../../../bindings/siren/cmd/gui/app.js', () => app)

beforeEach(() => vi.clearAllMocks())

describe('bloodhound api wrapper', () => {
  it('normalizes status payloads to camelCase', async () => {
    app.BloodHoundStatus.mockResolvedValue({
      Configured: true, Connected: false, ServerUrl: 'https://bh.example.com',
    })
    await expect(getBloodHoundStatus()).resolves.toEqual({
      configured: true, connected: false, serverUrl: 'https://bh.example.com', error: '',
    })
  })

  it('normalizes camelCase status payloads identically', async () => {
    app.BloodHoundStatus.mockResolvedValue({
      configured: false, connected: true, serverUrl: 'https://bh.internal', error: 'stale token',
    })
    await expect(getBloodHoundStatus()).resolves.toEqual({
      configured: false, connected: true, serverUrl: 'https://bh.internal', error: 'stale token',
    })
  })

  it('normalizes config views to camelCase across casing', async () => {
    app.BloodHoundGetConfig.mockResolvedValue({
      ServerUrl: 'https://bh.example.com', TokenId: 'tok-1', HasTokenKey: true, InsecureTls: false,
    })
    await expect(getBloodHoundConfig()).resolves.toEqual({
      serverUrl: 'https://bh.example.com', tokenId: 'tok-1', hasTokenKey: true, insecureTls: false,
    })
    app.BloodHoundGetConfig.mockResolvedValue({
      serverUrl: '', tokenId: '', hasTokenKey: false, insecureTls: true,
    })
    await expect(getBloodHoundConfig()).resolves.toEqual({
      serverUrl: '', tokenId: '', hasTokenKey: false, insecureTls: true,
    })
  })

  it('surfaces binding errors as rejected promises', async () => {
    app.BloodHoundStatus.mockRejectedValue(new Error('not connected'))
    await expect(getBloodHoundStatus()).rejects.toThrow('not connected')
  })

  it('maps agents to correlation refs across casing', async () => {
    app.BloodHoundCorrelate.mockResolvedValue({ a1: { owned: true } })
    await expect(correlateAgents([
      { ID: 'a1', Hostname: 'PC1', Username: 'CORP\\jane', RemoteAddress: '10.0.0.1' },
    ])).resolves.toEqual({ a1: { owned: true } })
    expect(app.BloodHoundCorrelate).toHaveBeenCalledWith([
      { id: 'a1', hostname: 'PC1', username: 'CORP\\jane', remoteAddress: '10.0.0.1' },
    ])
  })

  it('correlateAgents tolerates null bindings and empty lists', async () => {
    app.BloodHoundCorrelate.mockResolvedValue(null)
    await expect(correlateAgents([])).resolves.toEqual({})
  })
})

describe('owned marking', () => {
  it('passes the object ID through to the mark binding', async () => {
    app.BloodHoundMarkOwned.mockResolvedValue(null)
    await markBloodHoundOwned('S-1-5-21-999')
    expect(app.BloodHoundMarkOwned).toHaveBeenCalledWith('S-1-5-21-999')
  })

  it('passes the object ID through to the unmark binding', async () => {
    app.BloodHoundUnmarkOwned.mockResolvedValue(null)
    await unmarkBloodHoundOwned('S-1-5-21-999')
    expect(app.BloodHoundUnmarkOwned).toHaveBeenCalledWith('S-1-5-21-999')
  })
})
