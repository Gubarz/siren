import { describe, expect, it } from 'vitest'
import { validateHTTPC2ProfileText } from '../httpC2ProfileValidation.js'

function validProfile(overrides = {}) {
  return {
    Name: 'review-profile',
    ServerConfig: {
      Cookies: [{ Name: 'JSESSIONID' }],
      Headers: [{ Method: 'GET', Name: 'Cache-Control', Value: 'no-store', Probability: 100 }],
    },
    ImplantConfig: {
      UserAgent: 'Mozilla/5.0',
      NonceQueryArgChars: 'abcdef',
      NonceQueryLength: 3,
      NonceMode: 'UrlParam',
      ExtraURLParameters: [{ Method: 'GET', Name: 'v', Value: '1', Probability: 100 }],
      Headers: [{ Method: 'GET', Name: 'Accept', Value: '*/*', Probability: 100 }],
      MinFileGen: 1,
      MaxFileGen: 2,
      MinPathGen: 1,
      MaxPathGen: 2,
      MinPathLength: 1,
      MaxPathLength: 4,
      Extensions: ['js'],
      PathSegments: [
        { IsFile: true, Value: 'jquery.js' },
        { IsFile: false, Value: 'assets' },
      ],
    },
    ...overrides,
  }
}

describe('validateHTTPC2ProfileText', () => {
  it('rejects invalid JSON', () => {
    const result = validateHTTPC2ProfileText('{')

    expect(result.validJson).toBe(false)
    expect(result.canSave).toBe(false)
    expect(result.errors[0]).toContain('Invalid JSON')
  })

  it('accepts a valid protobuf-shaped HTTP C2 profile and renders a sample request', () => {
    const result = validateHTTPC2ProfileText(JSON.stringify(validProfile()))

    expect(result.canSave).toBe(true)
    expect(result.errors).toEqual([])
    expect(result.summary).toMatchObject({
      cookies: 1,
      extensions: 1,
      pathSegments: 2,
      headers: 2,
      urlParameters: 1,
    })
    expect(result.sampleRequest).toContain('GET /assets/jquery.js?abc=abc&v=1 HTTP/1.1')
    expect(result.sampleRequest).toContain('Cookie: JSESSIONID=...')
  })

  it('reports server-blocking validation errors before save', () => {
    const profile = validProfile({
      Name: '',
      ServerConfig: { Cookies: [] },
      ImplantConfig: {
        ...validProfile().ImplantConfig,
        UserAgent: 'bad`agent',
        Extensions: [],
      },
    })
    const result = validateHTTPC2ProfileText(JSON.stringify(profile))

    expect(result.canSave).toBe(false)
    expect(result.errors).toContain('Profile Name is required.')
    expect(result.errors).toContain('ServerConfig.Cookies must include at least one cookie.')
    expect(result.errors).toContain('ImplantConfig.Extensions must include at least one extension.')
    expect(result.errors).toContain('ImplantConfig.UserAgent cannot contain a backtick because it breaks implant compilation.')
  })

  it('surfaces warnings for duplicate and coerced values', () => {
    const profile = validProfile({
      ServerConfig: {
        Cookies: [{ Name: 'auth' }, { Name: 'auth' }],
        Headers: [
          { Method: 'GET', Name: 'X-Test', Value: 'a', Probability: 101 },
          { Method: 'GET', Name: 'X-Test', Value: 'b', Probability: 100 },
        ],
      },
      ImplantConfig: {
        ...validProfile().ImplantConfig,
        MinFileGen: 0,
        MaxFileGen: -1,
        Extensions: ['.php', '.php'],
      },
    })
    const result = validateHTTPC2ProfileText(JSON.stringify(profile))

    expect(result.canSave).toBe(true)
    expect(result.warnings).toEqual(expect.arrayContaining([
      'Duplicate cookie "auth" appears 2 times.',
      'Duplicate GET header "x-test" appears 2 times.',
      'Minimum file count is below 1; Sliver will coerce it upward.',
      'Maximum file count is lower than the minimum; Sliver will coerce it upward.',
    ]))
    expect(result.warnings.some((warning) => warning.includes('starts with a dot'))).toBe(true)
  })
})
