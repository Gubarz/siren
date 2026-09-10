// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen, fireEvent } from '@testing-library/svelte'

const runtime = vi.hoisted(() => ({
  onSliverEvent: vi.fn(() => vi.fn()),
  onWailsEvent: vi.fn(() => vi.fn()),
  onBloodhoundEvent: vi.fn(() => vi.fn()),
}))
vi.mock('$api/runtime.js', () => runtime)

vi.mock('$api/bloodhound.js', () => ({
  correlateAgents: vi.fn(async () => ({})),
  getBloodHoundIngestJobs: vi.fn(async () => []),
  getBloodHoundCollections: vi.fn(async () => []),
  getBloodHoundStatus: vi.fn(async () => null),
}))

vi.mock('$api/agents.js', () => ({
  GetAgentNotes: vi.fn(async () => ({})),
  SaveAgentNote: vi.fn(async () => undefined),
}))

vi.mock('$api/tags.js', () => ({
  GetAllAgentTags: vi.fn(async () => ({})),
  GetAllEntityTags: vi.fn(async () => ({})),
  GetAllAgentColors: vi.fn(async () => ({})),
  GetAllEntityColors: vi.fn(async () => ({})),
}))

import SessionsTable from '../SessionsTable.svelte'
import { STATUS_DOT_VARIANTS } from '$components/ui/StatusDot.svelte'

afterEach(cleanup)

const lostAgent = {
  ID: 'lost-1',
  Name: 'ghost',
  Hostname: 'GONE-PC',
  Username: 'CORP\\jane',
  OS: 'windows',
  _kind: 'session',
  _lost: true,
  LastCheckin: 0,
}

const liveAgent = {
  ID: 'live-1',
  Name: 'alive',
  Hostname: 'LIVE-PC',
  Username: 'CORP\\jane',
  OS: 'windows',
  _kind: 'session',
  LastCheckin: Math.floor(Date.now() / 1000),
}

describe('SessionsTable lost rows', () => {
  it('greys the lost row, hides last checkin, and blocks interact', async () => {
    const oninteract = vi.fn()
    const { container } = render(SessionsTable, { props: { data: [lostAgent], oninteract } })

    const dot = container.querySelector('[role="status"]')
    expect(dot.getAttribute('aria-label')).toBe('lost')
    expect(dot.className).toContain(STATUS_DOT_VARIANTS.lost)

    expect(screen.getByText('—')).toBeTruthy()

    const row = container.querySelector('tbody tr')
    expect(row.className).toContain('text-fg-muted')

    await fireEvent.dblClick(row)
    expect(oninteract).not.toHaveBeenCalled()
  })

  it('keeps live rows interactive', async () => {
    const oninteract = vi.fn()
    const { container } = render(SessionsTable, { props: { data: [liveAgent], oninteract } })

    const row = container.querySelector('tbody tr')
    await fireEvent.dblClick(row)
    expect(oninteract).toHaveBeenCalledWith('live-1')
  })
})
