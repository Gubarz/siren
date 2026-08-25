// @vitest-environment jsdom
import { describe, expect, it, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte'
import RelationSection from '../RelationSection.svelte'
import { toActionEntity } from '../relations.js'
import { actionsForEntity } from '../actions.js'

const rows = [
  { id: 'n1', objectId: 'S-1-5-21-777', label: 'JANE@CORP.LOCAL', kind: 'User', tierZero: true, owned: true },
  { id: 'n2', label: 'SRV01.CORP.LOCAL', kind: 'Computer', tierZero: false, owned: false },
]

function actionsFor(entity) {
  return actionsForEntity({
    entity: toActionEntity(entity),
    enrichment: { owned: true, tierZero: entity.tierZero },
    addToCase: { open: vi.fn() },
    openTags: vi.fn(),
    openComments: vi.fn(),
  })
}

describe('RelationSection rows', () => {
  afterEach(() => cleanup())

  it('reveals entity actions when a row is expanded', async () => {
    render(RelationSection, { props: { title: 'Sessions', entities: rows, actionsFor } })

    await fireEvent.click(screen.getByRole('button', { name: /jane@corp\.local/i }))
    // The row's own objectId drives the bridge even though the graph node
    // also carries an opaque key (n1) — "Add to case" proves the tier-zero
    // seed reached the action list.
    expect(screen.getByRole('button', { name: 'Add to case' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Tag' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Comment' })).toBeTruthy()

    cleanup()
    render(RelationSection, { props: { title: 'Sessions', entities: rows, actionsFor } })
    await fireEvent.click(screen.getByRole('button', { name: /srv01\.corp\.local/i }))
    // Un-owned computers surface the lateral-movement action via the node-id
    // fallback for objectId.
    expect(screen.getByRole('button', { name: /move to srv01/i })).toBeTruthy()
  })

  it('keeps collapsed rows free of action buttons', () => {
    render(RelationSection, { props: { title: 'Sessions', entities: rows, actionsFor } })
    expect(screen.queryByRole('button', { name: 'Tag' })).toBeNull()
  })

  it('marks expanded rows with aria-expanded', async () => {
    render(RelationSection, { props: { title: 'Sessions', entities: [rows[0]], actionsFor } })
    const row = screen.getByRole('button', { name: /jane@corp\.local/i })
    expect(row.getAttribute('aria-expanded')).toBe('false')
    await fireEvent.click(row)
    expect(row.getAttribute('aria-expanded')).toBe('true')
  })
})
