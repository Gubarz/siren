// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import ExecuteAssemblyModal from '../ExecuteAssemblyModal.svelte'
import MakeTokenModal from '../MakeTokenModal.svelte'

afterEach(() => {
  cleanup()
})

describe('CommandModalFrame-backed modals', () => {
  it('applies saved initial values and emits the built command', async () => {
    const onexecute = vi.fn()
    render(ExecuteAssemblyModal, {
      props: {
        open: true,
        firstSessionID: '',
        initialValues: { process: 'notepad.exe', ppid: 4321 },
        onexecute,
      },
    })

    expect(screen.getByLabelText('Process').value).toBe('notepad.exe')

    await fireEvent.click(screen.getByRole('button', { name: 'Execute' }))
    expect(onexecute).toHaveBeenCalledWith({
      cmd: expect.stringContaining('--process notepad.exe'),
    })
  })

  it('resets credential fields from saved initial values', async () => {
    render(MakeTokenModal, {
      props: {
        open: true,
        initialValues: { username: 'alice', password: 'secret', domain: 'CORP' },
        onexecute: vi.fn(),
      },
    })

    await fireEvent.click(screen.getByRole('button', { name: /Credentials/ }))
    expect(screen.getByLabelText('Username').value).toBe('alice')
    expect(screen.getByLabelText('Domain').value).toBe('CORP')
  })
})
