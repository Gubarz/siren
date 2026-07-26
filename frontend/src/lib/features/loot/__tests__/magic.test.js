import { describe, it, expect } from 'vitest'
import { detectMime, fileTypeLabel } from '../magic.js'

function bytes(arr) { return new Uint8Array(arr) }

describe('detectMime', () => {
  it('detects PNG', () => {
    const r = detectMime(bytes([0x89, 0x50, 0x4E, 0x47]))
    expect(r?.mime).toBe('image/png')
  })
  it('detects JPEG', () => {
    const r = detectMime(bytes([0xFF, 0xD8, 0xFF, 0xE0]))
    expect(r?.mime).toBe('image/jpeg')
  })
  it('detects PDF', () => {
    const r = detectMime(bytes([0x25, 0x50, 0x44, 0x46]))
    expect(r?.mime).toBe('application/pdf')
  })
  it('detects ELF', () => {
    const r = detectMime(bytes([0x7F, 0x45, 0x4C, 0x46]))
    expect(r?.mime).toBe('application/x-elf')
  })
  it('detects PEM', () => {
    const r = detectMime(new TextEncoder().encode('-----BEGIN CERTIFICATE-----'))
    expect(r?.mime).toBe('application/x-pem-file')
  })
  it('handles empty buffer', () => {
    expect(detectMime(bytes([]))).toBeNull()
  })
  it('handles buffer shorter than signature', () => {
    expect(detectMime(bytes([0x89, 0x50]))).toBeNull()
  })
  it('returns null for unknown bytes', () => {
    expect(detectMime(bytes([0x00, 0x00, 0x00, 0x00]))).toBeNull()
  })
})

describe('fileTypeLabel', () => {
  it('returns TEXT for 0', () => expect(fileTypeLabel(0)).toBe('TEXT'))
  it('returns BINARY for 1', () => expect(fileTypeLabel(1)).toBe('BINARY'))
  it('returns UNKNOWN for others', () => expect(fileTypeLabel(99)).toBe('UNKNOWN'))
})
