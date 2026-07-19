import { describe, it, expect } from 'vitest'
import { errorMessage } from '../errors.js'

describe('errorMessage', () => {
  it('returns the message from an Error object', () => {
    expect(errorMessage(new Error('boom'))).toBe('boom')
  })

  it('returns the string value for a string error', () => {
    expect(errorMessage('fail')).toBe('fail')
  })

  it('appends prefix when provided', () => {
    expect(errorMessage(new Error('error'), 'Prefix: ')).toBe('Prefix: error')
  })

  it('returns empty string for null', () => {
    expect(errorMessage(null)).toBe('null')
  })

  it('returns message for object with message field', () => {
    expect(errorMessage({ message: 'custom' })).toBe('custom')
  })
})
