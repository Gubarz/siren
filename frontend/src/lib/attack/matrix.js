import verbsMap from './verbs.json'

export function joinCountsWithMap(verbCounts) {
  const techniques = new Map()
  const unmapped = []
  for (const [verb, count] of Object.entries(verbCounts ?? {})) {
    const ids = verbsMap[verb]
    if (!ids) {
      if (count > 0) unmapped.push({ verb, count })
      continue
    }
    for (const id of ids) {
      const cell = techniques.get(id) ?? { techniqueID: id, score: 0, verbs: [] }
      cell.score += count
      cell.verbs.push({ verb, count })
      techniques.set(id, cell)
    }
  }
  return {
    techniques: [...techniques.values()].sort((a, b) => b.score - a.score),
    unmapped: unmapped.sort((a, b) => b.count - a.count),
  }
}

export function navigatorLayer(techniques, name = 'siren operation') {
  return {
    name,
    versions: { layer: '4.5', attack: '16', navigator: '5.1' },
    domain: 'enterprise-attack',
    description: 'Generated from siren activity journal verb counts',
    techniques: techniques.map((t) => ({
      techniqueID: t.techniqueID,
      score: t.score,
      comment: t.verbs.map((v) => `${v.verb}×${v.count}`).join(', '),
    })),
  }
}

export function downloadNavigatorJSON(techniques, name) {
  const blob = new Blob([JSON.stringify(navigatorLayer(techniques, name), null, 2)], {
    type: 'application/json',
  })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `${(name ?? 'attack-layer').replaceAll(' ', '-')}.json`
  anchor.click()
  URL.revokeObjectURL(url)
}
