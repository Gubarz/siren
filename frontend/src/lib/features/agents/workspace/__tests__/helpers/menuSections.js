// Shared lookup helpers for context-menu section assertions.

export function flattenItems(sections) {
  const items = []
  for (const section of sections) {
    for (const item of section.items ?? []) {
      items.push(item)
      for (const child of item.children ?? []) items.push(child)
    }
  }
  return items
}

export function findItem(sections, label) {
  return flattenItems(sections).find((item) => item.label === label)
}

export function sectionItems(sections, label) {
  return findItem(sections, label) ?? null
}
