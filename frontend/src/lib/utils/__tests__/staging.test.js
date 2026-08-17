import { describe, expect, it } from 'vitest'
import {
  STAGER_JOB_DESCRIPTION,
  isStagerJob,
  stagedBuildRows,
  stagerListenerRows,
  stagerListeners,
} from '../staging.js'

describe('isStagerJob', () => {
  it('identifies stager jobs by description in PascalCase or camelCase', () => {
    expect(isStagerJob({ Description: STAGER_JOB_DESCRIPTION })).toBe(true)
    expect(isStagerJob({ description: STAGER_JOB_DESCRIPTION })).toBe(true)
  })

  it('rejects non-stager jobs and missing descriptions', () => {
    expect(isStagerJob({ Description: 'mTLS listener' })).toBe(false)
    expect(isStagerJob({})).toBe(false)
    expect(isStagerJob(null)).toBe(false)
  })
})

describe('stagerListeners', () => {
  it('filters stager jobs out of a jobs list', () => {
    const jobs = [
      { ID: 1, Description: 'mTLS listener' },
      { ID: 2, Description: STAGER_JOB_DESCRIPTION, ProfileName: 'implant-profile' },
    ]
    expect(stagerListeners(jobs)).toEqual([jobs[1]])
  })

  it('defaults to an empty list', () => {
    expect(stagerListeners()).toEqual([])
    expect(stagerListeners(undefined)).toEqual([])
  })
})

describe('stagedBuildRows', () => {
  it('shapes only staged builds with derived columns', () => {
    const rows = stagedBuildRows([
      { name: 'a', staged: true, nonce: 4242, GOOS: 'linux', GOARCH: 'amd64', Format: 2, IsBeacon: true },
      { name: 'b', staged: false },
    ])
    expect(rows).toEqual([
      {
        _rowKey: 'a',
        _name: 'a',
        _nonce: 4242,
        _osArch: 'linux/amd64',
        _format: 'executable',
        _type: 'beacon',
      },
    ])
  })

  it('defaults nonce to null when absent', () => {
    const rows = stagedBuildRows([{ name: 'a', staged: true }])
    expect(rows[0]._nonce).toBeNull()
  })
})

describe('stagerListenerRows', () => {
  it('shapes stager jobs with id, port and profile', () => {
    const rows = stagerListenerRows([
      { ID: 7, Description: STAGER_JOB_DESCRIPTION, Port: 8080, ProfileName: 'win64' },
    ])
    expect(rows).toEqual([
      { _rowKey: 7, _id: 7, _port: 8080, _profile: 'win64' },
    ])
  })
})

describe('row shapers camelCase fallback', () => {
  it('stagedBuildRows falls back to camelCase build fields', () => {
    const rows = stagedBuildRows([
      { name: 'x', staged: true, nonce: 555, goos: 'windows', goarch: 'amd64', format: 1, isBeacon: false },
    ])
    expect(rows[0]._osArch).toBe('windows/amd64')
    expect(rows[0]._format).toBe('shellcode')
    expect(rows[0]._type).toBe('session')
    expect(rows[0]._nonce).toBe(555)
  })

  it('stagerListenerRows falls back to camelCase job fields', () => {
    const rows = stagerListenerRows([
      { id: 3, Description: STAGER_JOB_DESCRIPTION, port: 8080, profileName: 'win64' },
    ])
    expect(rows).toEqual([{ _rowKey: 3, _id: 3, _port: 8080, _profile: 'win64' }])
  })
})
