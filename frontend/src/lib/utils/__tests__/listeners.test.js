import { describe, expect, it } from 'vitest'
import { formatListenerC2, listenerHost, listenerProtocol } from '../listeners.js'

describe('listener utilities', () => {
  describe('listenerProtocol', () => {
    it('detects mtls', () => {
      expect(listenerProtocol({ Protocol: 'mtls' })).toBe('mtls')
      expect(listenerProtocol({ name: 'mtls-listener' })).toBe('mtls')
    })

    it('detects https and http', () => {
      expect(listenerProtocol({ protocol: 'https' })).toBe('https')
      expect(listenerProtocol({ description: 'HTTP listener' })).toBe('http')
    })

    it('detects dns and wireguard', () => {
      expect(listenerProtocol({ Name: 'dns' })).toBe('dns')
      expect(listenerProtocol({ Name: 'wg' })).toBe('wg')
      expect(listenerProtocol({ Description: 'wireguard tunnel' })).toBe('wg')
    })

    it('returns empty string on unknown/null', () => {
      expect(listenerProtocol(null)).toBe('')
      expect(listenerProtocol({ name: 'custom' })).toBe('')
    })
  })

  describe('listenerHost', () => {
    it('prefers explicit valid domains', () => {
      expect(listenerHost({ Domains: ['c2.example.com', '0.0.0.0'] })).toBe('c2.example.com')
    })

    it('skips wildcard / bind-all domains and checks host / bindHost', () => {
      expect(listenerHost({ Domains: ['0.0.0.0'], Host: '192.168.1.50' })).toBe('192.168.1.50')
      expect(listenerHost({ BindHost: '10.0.0.1' })).toBe('10.0.0.1')
    })

    it('uses fallback when host is 0.0.0.0 or empty', () => {
      expect(listenerHost({ Host: '0.0.0.0' }, 'server.local')).toBe('server.local')
      expect(listenerHost(null, 'default.host')).toBe('default.host')
    })
  })

  describe('formatListenerC2', () => {
    it('formats tcp/ip based listeners (mtls, https, http, wg)', () => {
      expect(formatListenerC2({ protocol: 'mtls', host: '10.0.0.5', port: 8888 })).toBe('mtls://10.0.0.5:8888')
      expect(formatListenerC2({ protocol: 'https', host: 'example.com', port: 443 })).toBe('https://example.com:443')
    })

    it('formats dns listeners with domains', () => {
      expect(formatListenerC2({ protocol: 'dns', domains: ['ns1.c2.io'] })).toBe('dns://ns1.c2.io')
    })

    it('handles fallbacks on missing values', () => {
      expect(formatListenerC2(null, 'teamserver.lan')).toBe('mtls://teamserver.lan:443')
    })
  })
})
