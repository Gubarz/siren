import { describe, expect, it } from 'vitest'
import { catalogToCategories } from '../catalog.js'

describe('catalogToCategories', () => {
  it('maps groups to { category, commands } and drops empty groups', () => {
    const categories = catalogToCategories({
      groups: [
        { title: 'Sliver', commands: [{ name: 'sessions' }] },
        { title: 'Empty', commands: [] },
        { title: 'Nameless', commands: [{ description: 'no name' }] },
      ],
    })
    expect(categories).toEqual([
      { category: 'Sliver', commands: [{ name: 'sessions' }] },
    ])
  })

  it('falls back to the group id when title is missing', () => {
    const categories = catalogToCategories({
      groups: [{ id: 'other', commands: [{ name: 'whoami' }] }],
    })
    expect(categories[0].category).toBe('other')
  })

  it('handles a missing catalog', () => {
    expect(catalogToCategories(null)).toEqual([])
    expect(catalogToCategories(undefined)).toEqual([])
  })
})
