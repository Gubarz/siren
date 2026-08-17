import { describe, expect, it } from 'vitest'
import {
  isPointerAtWindowBoundary,
  isScreenPointOutsideWindow,
  pointerScreenPoint,
} from '../tabTearOut.js'

const hostWindow = {
  innerWidth: 1000,
  innerHeight: 700,
  outerWidth: 1024,
  outerHeight: 768,
  screenX: 200,
  screenY: 100,
}

describe('agent tab tear-out', () => {
  it('keeps the last usable screen point when a WebView reports zeroes', () => {
    expect(pointerScreenPoint({ screenX: 0, screenY: 0 }, { x: 50, y: 80 })).toEqual({ x: 50, y: 80 })
  })

  it('detects pointer capture reaching the viewport edge', () => {
    expect(isPointerAtWindowBoundary({ clientX: 0, clientY: 300 }, null, hostWindow)).toBe(true)
    expect(isPointerAtWindowBoundary({ clientX: 400, clientY: 300 }, null, hostWindow)).toBe(false)
  })

  it('uses native screen coordinates when client coordinates are unavailable', () => {
    const event = {}
    expect(isPointerAtWindowBoundary(event, { x: 1223, y: 400 }, hostWindow)).toBe(true)
    expect(isPointerAtWindowBoundary(event, { x: 600, y: 400 }, hostWindow)).toBe(false)
  })

  it('distinguishes points inside and outside the native window', () => {
    expect(isScreenPointOutsideWindow({ x: 1300, y: 400 }, hostWindow)).toBe(true)
    expect(isScreenPointOutsideWindow({ x: 600, y: 400 }, hostWindow)).toBe(false)
  })
})
