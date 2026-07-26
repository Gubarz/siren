import { describe, it, expect } from 'vitest'

function exportCSV(rows) {
  const header = 'ID,Name,FileType,Size\n'
  const body = rows.map((r) =>
    [r._id, `"${(r._name ?? '').replaceAll('"', '""')}"`, r._fileType, r._size].join(',')
  ).join('\n')
  return header + body
}

describe('Loot CSV export', () => {
  it('produces correct CSV', () => {
    const rows = [
      { _id: 'abc', _name: 'test.txt', _fileType: 'TEXT', _size: 1024 },
      { _id: 'def', _name: 'data,bin', _fileType: 'BINARY', _size: 0 },
    ]
    const csv = exportCSV(rows)
    expect(csv).toContain('abc,"test.txt",TEXT,1024')
    expect(csv).toContain('def,"data,bin",BINARY,0')
  })
})
