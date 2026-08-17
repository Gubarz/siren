import { describe, expect, it } from 'vitest'
import { implantFormat } from '../formats.js'

describe('implantFormat', () => {
  it('maps numeric formats to labels', () => {
    expect(implantFormat(0)).toBe('shared lib')
    expect(implantFormat(1)).toBe('shellcode')
    expect(implantFormat(2)).toBe('executable')
    expect(implantFormat(3)).toBe('service')
    expect(implantFormat(4)).toBe('third-party')
  })

  it('passes through unknown values', () => {
    expect(implantFormat(9)).toBe(9)
    expect(implantFormat(undefined)).toBe(undefined)
  })
})
