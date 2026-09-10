// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest'
import { cleanup, render } from '@testing-library/svelte'
import StatusDot, { STATUS_DOT_VARIANTS } from '../StatusDot.svelte'

afterEach(cleanup)

function dotFor(variant) {
  const { container } = render(StatusDot, { props: { variant } })
  return container.querySelector('[role="status"]')
}

describe('StatusDot variants', () => {
  it('pins the lost variant to grey in the exported map', () => {
    expect(STATUS_DOT_VARIANTS.lost).toBe('bg-gray-500')
  })

  it('renders the lost variant from the exported map', () => {
    expect(dotFor('lost').className).toContain(STATUS_DOT_VARIANTS.lost)
  })

  it('maps online and offline to their status colors', () => {
    expect(dotFor('online').className).toContain(STATUS_DOT_VARIANTS.online)
    expect(dotFor('offline').className).toContain(STATUS_DOT_VARIANTS.offline)
    expect(STATUS_DOT_VARIANTS.online).toBe('bg-success-500')
    expect(STATUS_DOT_VARIANTS.offline).toBe('bg-danger-500')
  })
})
