import { describe, expect, it, vi } from 'vitest'

import { buildProcessContextSections, normalizeProcess } from '../processExplorerHelpers.js'

function openSpy() {
  return vi.fn()
}

function findItem(sections, label) {
  for (const section of sections) {
    const item = (section.items || []).find((i) => i.label === label)
    if (item) return item
  }
  return null
}

describe('normalizeProcess', () => {
  it('maps PascalCase live process fields onto column keys', () => {
    expect(normalizeProcess({
      Pid: 42,
      Ppid: 1,
      Executable: 'explorer.exe',
      Owner: 'DOMAIN\\user',
      Architecture: 'x64',
      SessionID: 3,
    })).toMatchObject({
      PidStr: '42',
      PpidStr: '1',
      ExecutableStr: 'explorer.exe',
      OwnerStr: 'DOMAIN\\user',
      ArchStr: 'x64',
      SessionStr: '3',
    })
  })

  it('falls back to lowercase snapshot fields and defaults', () => {
    expect(normalizeProcess({ pid: 7, executable: 'beacon.exe' })).toMatchObject({
      PidStr: '7',
      PpidStr: '0',
      ExecutableStr: 'beacon.exe',
      OwnerStr: '',
      ArchStr: '',
      SessionStr: '',
    })
  })
})

describe('buildProcessContextSections', () => {
  it('binds command modals to the agent session', () => {
    const commandModal = { open: openSpy() }
    const sections = buildProcessContextSections({
      pid: 564,
      procName: 'explorer.exe',
      commandModal,
      killProcess: () => {},
      sessionID: 'session-1',
    })

    findItem(sections, 'Migrate Into…').on()
    expect(commandModal.open).toHaveBeenCalledWith(
      expect.objectContaining({
        command: expect.objectContaining({ name: 'migrate' }),
        useSession: true,
        targetIDs: ['session-1'],
        initialValues: { pid: 564, arch: '' },
      }),
    )

    commandModal.open.mockClear()
    findItem(sections, 'Execute Assembly…').on()
    expect(commandModal.open).toHaveBeenCalledWith(
      expect.objectContaining({
        targetIDs: ['session-1'],
        initialValues: { ppid: 564, process: 'explorer.exe' },
      }),
    )
  })

  it('falls back to the server scope when no session is available', () => {
    const commandModal = { open: openSpy() }
    const sections = buildProcessContextSections({
      pid: 100,
      procName: '',
      commandModal,
      killProcess: () => {},
    })

    findItem(sections, 'Migrate Into…').on()
    expect(commandModal.open).toHaveBeenCalledWith(
      expect.objectContaining({
        useSession: false,
        targetIDs: [],
      }),
    )
  })
})
