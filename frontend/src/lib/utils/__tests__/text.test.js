import { describe, it, expect } from 'vitest'
import { stripTerminalFormatting, stripTerminalMetadata, commandTooltip, cleanUsername } from '../text.js'

describe('stripTerminalFormatting', () => {
  it('removes ANSI escape sequences', () => {
    const input = '\u001b[31mred\u001b[0m'
    expect(stripTerminalFormatting(input)).toBe('red')
  })

  it('removes carriage returns', () => {
    expect(stripTerminalFormatting('line1\r\nline2')).toBe('line1\nline2')
  })

  it('removes control characters', () => {
    const input = 'hello\u0000world'
    expect(stripTerminalFormatting(input)).toBe('helloworld')
  })

  it('handles null or undefined', () => {
    expect(stripTerminalFormatting(null)).toBe('')
    expect(stripTerminalFormatting(undefined)).toBe('')
  })
})

describe('stripTerminalMetadata', () => {
  it('removes OSC sequences', () => {
    const input = 'text\x1b]0;title\x07more'
    expect(stripTerminalMetadata(input)).toBe('textmore')
  })
})

describe('commandTooltip', () => {
  it('returns unavailable text when set', () => {
    const cmd = { unavailable: '\u001b[31mnot available' }
    expect(commandTooltip(cmd)).toBe('not available')
  })

  it('extracts summary from description', () => {
    const cmd = { description: 'About: this is the tooltip\nMore info' }
    expect(commandTooltip(cmd)).toBe('this is the tooltip')
  })

  it('truncates long summaries', () => {
    const long = 'a'.repeat(300)
    const cmd = { description: `About: ${long}` }
    const result = commandTooltip(cmd)
    expect(result.length).toBeLessThanOrEqual(243)
  })
})

describe('cleanUsername', () => {
  it('extracts username from user@ip (id) format', () => {
    expect(cleanUsername('goober@192.168.1.10 (123)')).toBe('goober')
    expect(cleanUsername('alice@10.0.0.1:31337')).toBe('alice')
  })

  it('handles clean usernames', () => {
    expect(cleanUsername('root')).toBe('root')
  })

  it('fallbacks to Operator if empty or null', () => {
    expect(cleanUsername('')).toBe('Operator')
    expect(cleanUsername(null)).toBe('Operator')
  })
})
