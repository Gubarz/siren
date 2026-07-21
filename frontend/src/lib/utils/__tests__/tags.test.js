import { describe, it, expect } from 'vitest'
import { parseTag, getTagCategoryStyle } from '../tags.js'

describe('tags utilities', () => {
  it('parses plain tags correctly', () => {
    const res = parseTag('prod')
    expect(res.isTyped).toBe(false)
    expect(res.key).toBe('')
    expect(res.value).toBe('prod')
  })

  it('parses typed tags correctly', () => {
    const res = parseTag('env:production')
    expect(res.isTyped).toBe(true)
    expect(res.key).toBe('env')
    expect(res.value).toBe('production')
  })

  it('handles spaces in typed tags', () => {
    const res = parseTag('  role : domain controller ')
    expect(res.isTyped).toBe(true)
    expect(res.key).toBe('role')
    expect(res.value).toBe('domain controller')
  })

  it('provides category styling for known keys', () => {
    expect(getTagCategoryStyle('env').container).toContain('blue')
    expect(getTagCategoryStyle('role').container).toContain('teal')
    expect(getTagCategoryStyle('prio').container).toContain('red')
    expect(getTagCategoryStyle('group').container).toContain('purple')
    expect(getTagCategoryStyle('status').container).toContain('emerald')
    expect(getTagCategoryStyle('owner').container).toContain('amber')
    expect(getTagCategoryStyle('ip').container).toContain('indigo')
    expect(getTagCategoryStyle('custom').container).toContain('slate')
  })
})
