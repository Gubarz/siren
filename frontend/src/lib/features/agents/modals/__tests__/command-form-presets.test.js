// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup, waitFor } from '@testing-library/svelte'
import CommandFormV2 from '../CommandFormV2.svelte'

// Mirrors the catalog entry the backend produces for the nanodump armory
// extension: arguments are parsed from bare UPPERCASE words in the cobra
// command's Use string, plus the standard save/timeout extension flags.
const nanodump = {
  name: 'nanodump',
  path: 'nanodump',
  usage: 'nanodump PID DUMP-NAME WRITE-FILE SIGNATURE',
  description: 'A Beacon Object File that creates a minidump of the LSASS process.',
  arguments: [
    { name: 'PID', required: true, variadic: false },
    { name: 'DUMP-NAME', required: true, variadic: false },
    { name: 'WRITE-FILE', required: true, variadic: false },
    { name: 'SIGNATURE', required: true, variadic: false },
  ],
  flags: [
    { name: 'save', shorthand: 's', usage: 'Save output to disk', type: 'bool', default: 'false', required: false, boolean: true },
    { name: 'timeout', shorthand: 't', usage: 'command timeout in seconds', type: 'int', default: '30', required: false, boolean: false },
  ],
  needsInput: true,
  supported: true,
  unavailable: '',
}

async function openPresetsMenu() {
  const btn = screen.getByRole('button', { name: /presets/i })
  // flowbite-svelte's Popper opens click-triggered popovers on mousedown.
  await fireEvent.mouseDown(btn)
}

describe('CommandFormV2 presets (third-class modal)', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    cleanup()
  })

  it('saves a preset and restores values when applied', async () => {
    render(CommandFormV2, {
      props: {
        command: nanodump,
        open: true,
        sessionID: '',
        firstSessionID: '',
      },
    })

    const pidInput = screen.getByLabelText('PID')
    const dumpNameInput = screen.getByLabelText('DUMP NAME')

    await fireEvent.input(pidInput, { target: { value: '4444' } })
    await fireEvent.input(dumpNameInput, { target: { value: 'lsass.dmp' } })
    expect(pidInput.value).toBe('4444')

    // Fill the advanced command-line field too.
    await fireEvent.click(screen.getByRole('button', { name: /advanced command-line arguments/i }))
    const advancedInput = screen.getByLabelText('Additional arguments')
    await fireEvent.input(advancedInput, { target: { value: '--extra-flag foo' } })

    // Save the preset
    await openPresetsMenu()
    const saveItem = await screen.findByText('Save current preset…', undefined, { timeout: 3000 })
    await fireEvent.click(saveItem)

    const nameInput = await screen.findByPlaceholderText('Preset name...', undefined, { timeout: 3000 })
    await fireEvent.input(nameInput, { target: { value: 'my preset' } })
    await fireEvent.keyDown(nameInput, { key: 'Enter' })

    // Verify it landed in localStorage
    const stored = JSON.parse(localStorage.getItem('gui-command-presets'))
    expect(stored.presets['nanodump']).toHaveLength(1)
    expect(stored.presets['nanodump'][0].values.args['PID']).toBe('4444')
    expect(stored.presets['nanodump'][0].values.args['DUMP-NAME']).toBe('lsass.dmp')
    expect(stored.presets['nanodump'][0].values.advanced).toBe('--extra-flag foo')

    // Simulate fields being cleared (e.g. modal reopened)
    await fireEvent.input(pidInput, { target: { value: '' } })
    await fireEvent.input(dumpNameInput, { target: { value: '' } })
    await fireEvent.input(advancedInput, { target: { value: '' } })
    expect(pidInput.value).toBe('')

    // Apply the preset
    await openPresetsMenu()
    const presetItem = await screen.findByText('my preset', undefined, { timeout: 3000 })
    await fireEvent.click(presetItem)

    await waitFor(() => {
      expect(pidInput.value).toBe('4444')
    }, { timeout: 3000 })
    expect(dumpNameInput.value).toBe('lsass.dmp')
    expect(screen.getByLabelText('Additional arguments').value).toBe('--extra-flag foo')
  })

  it('restores a saved preset after the modal is closed and reopened', async () => {
    // Seed a preset as if saved in a previous session of the modal.
    localStorage.setItem(
      'gui-command-presets',
      JSON.stringify({
        version: 1,
        presets: {
          nanodump: [
            {
              name: 'my preset',
              values: {
                args: { 'PID': '4444', 'DUMP-NAME': 'lsass.dmp', 'WRITE-FILE': '1', 'SIGNATURE': 'PMDM' },
                flags: { save: false, timeout: '30' },
                advanced: '--extra-flag foo',
              },
            },
          ],
        },
      }),
    )

    render(CommandFormV2, {
      props: {
        command: nanodump,
        open: true,
        sessionID: '',
        firstSessionID: '',
      },
    })

    const pidInput = screen.getByLabelText('PID')
    const dumpNameInput = screen.getByLabelText('DUMP NAME')
    expect(pidInput.value).toBe('')

    await openPresetsMenu()
    const presetItem = await screen.findByText('my preset', undefined, { timeout: 3000 })
    await fireEvent.click(presetItem)

    await waitFor(() => {
      expect(pidInput.value).toBe('4444')
    }, { timeout: 3000 })
    expect(dumpNameInput.value).toBe('lsass.dmp')
    expect(screen.getByLabelText('Additional arguments').value).toBe('--extra-flag foo')
  })
})
