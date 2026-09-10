import { beforeEach, describe, expect, it, vi } from 'vitest'

// api/agents.js re-exports the full binding surface, so the generated
// module mock must provide every name it touches.
const app = vi.hoisted(() => {
  const names = [
    'CancelBeaconTask', 'Chmod', 'CopyPath', 'DownloadFile', 'DownloadDirectory',
    'DownloadMultipleTar', 'GetDownloadHistory', 'GetAllDownloadHistory', 'ClearDownloadHistory',
    'GetAgentNotes', 'SaveAgentNote', 'GetBeaconTaskOutput', 'KillAgent', 'KillProcess',
    'MakeDir', 'Pwd', 'RemovePath', 'RemoveBeacon', 'RenameAgent', 'RenamePath',
    'TakeScreenshot', 'UploadFile', 'GrepFiles', 'GetServices', 'ListProxies', 'ListRportfwds',
    'StartPortfwd', 'StartRportfwd', 'StartSocks', 'StopPortfwd', 'StopRportfwd', 'StopSocks',
    'StartService', 'StopService', 'RemoveService', 'ViewRemoteFile', 'UploadFiles',
    'GetBeaconTasks', 'GetBeacons', 'GetFileList', 'GetProcessList', 'GetSessions',
    'ListKnownAgents', 'RemoveKnownAgent',
    'CreateRegistryKey', 'DeleteRegistryEntry', 'ListRegistrySubKeys', 'ListRegistryValues',
    'ReadRegistryValue', 'WriteRegistryValue',
  ]
  return Object.fromEntries(names.map((name) => [name, vi.fn()]))
})

vi.mock('../../../../bindings/siren/cmd/gui/app.js', () => app)

describe('known agents api', () => {
  let listKnownAgents

  beforeEach(async () => {
    vi.resetModules()
    vi.clearAllMocks()
    ;({ listKnownAgents } = await import('../agents.js'))
  })

  it('returns records from the binding', async () => {
    app.ListKnownAgents.mockResolvedValue([{ id: 'a', status: 'lost' }])
    await expect(listKnownAgents()).resolves.toEqual([{ id: 'a', status: 'lost' }])
  })

  it('normalizes a null binding result to an empty list', async () => {
    app.ListKnownAgents.mockResolvedValue(null)
    await expect(listKnownAgents()).resolves.toEqual([])
  })

  it('exposes RemoveKnownAgent through the wrapper module', async () => {
    const api = await import('../agents.js')
    expect(api.RemoveKnownAgent).toBe(app.RemoveKnownAgent)
  })
})
