import { describe, expect, it } from 'vitest'
import {
  credentialLoginFields,
  credentialPickerOptions,
  parseCredentialUsername,
  plaintextCredentials,
} from '../credentials.js'

describe('credential utilities', () => {
  it('filters to credentials usable by password-backed dialogs', () => {
    const rows = [
      { ID: '1', Username: 'alice', Plaintext: 'Password1!' },
      { ID: '2', Username: 'bob', Hash: 'deadbeef' },
      { ID: '3', Plaintext: 'missing-user' },
    ]

    expect(plaintextCredentials(rows)).toEqual([rows[0]])
  })

  it('builds picker labels with collection context', () => {
    expect(credentialPickerOptions([
      { ID: '1', Username: 'CORP\\alice', Plaintext: 'Password1!', Collection: 'domain-admins' },
    ])).toEqual([
      { value: '1', label: 'CORP\\alice (domain-admins)' },
    ])
  })

  it('parses domain-qualified usernames', () => {
    expect(parseCredentialUsername('CORP\\alice')).toEqual({ username: 'alice', domain: 'CORP' })
    expect(parseCredentialUsername('alice@corp.local')).toEqual({ username: 'alice', domain: 'corp.local' })
    expect(parseCredentialUsername('alice')).toEqual({ username: 'alice', domain: '' })
  })

  it('maps credentials into login fields', () => {
    expect(credentialLoginFields({
      Username: 'CORP\\alice',
      Plaintext: 'Password1!',
    })).toEqual({
      username: 'alice',
      domain: 'CORP',
      password: 'Password1!',
    })
  })
})
