#!/usr/bin/env node
// Enforces the frontend layer contract documented in CONTRIBUTING.md.
//
// eslint-plugin-boundaries is configured for this but reports nothing with the
// eslint version in use, so the contract had no enforcement behind it. This is
// the same shape as scripts/check-import-boundaries.sh on the Go side: walk the
// real import graph and fail on anything the allowlist does not cover.

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const FRONTEND = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const SRC = join(FRONTEND, 'src/lib')

// Elements, longest path first so components/patterns is not read as ui.
const LAYERS = [
  ['patterns', 'src/lib/components/patterns'],
  ['system', 'src/lib/components/system'],
  ['ui', 'src/lib/components/ui'],
  ['features', 'src/lib/features'],
  ['stores', 'src/lib/stores'],
  ['api', 'src/lib/api'],
  ['utils', 'src/lib/utils'],
]

// What each layer may import, per CONTRIBUTING.md's stack diagram. A layer
// importing itself is always fine and is short-circuited below.
const ALLOWED = {
  features: ['api', 'stores', 'patterns', 'ui', 'utils'],
  patterns: ['api', 'stores', 'ui', 'utils'],
  system: ['api', 'stores', 'patterns', 'ui'],
  ui: ['utils'],
  stores: ['api', 'utils'],
  api: ['utils', 'bindings'],
  utils: [],
}

const ALIASES = {
  '$components/': 'src/lib/components/',
  '$features/': 'src/lib/features/',
  '$stores/': 'src/lib/stores/',
  '$api/': 'src/lib/api/',
  '$utils/': 'src/lib/utils/',
}

// Violations that predate this check. Frozen so new ones fail, and reported as
// stale when one is fixed so the entry gets removed.
//
// These are not all the same kind of problem. The five system/*Root components
// and system -> utils look like the documented policy being narrower than the
// app: those components compose features into the app shell, and every other
// layer may import utils. The two ui -> patterns/stores and utils -> api entries
// look like real layering inversions that want code moved. Deciding which is
// which is a design call, so this freezes all ten rather than guessing.
const KNOWN = new Set([
  'src/lib/components/system/AddToCaseRoot.svelte -> features',
  'src/lib/components/system/EntityCommentsRoot.svelte -> features',
  'src/lib/components/system/EntityTagsRoot.svelte -> features',
  'src/lib/components/system/KeyboardShortcutsRoot.svelte -> features',
  'src/lib/components/system/PanelRouter.svelte -> features',
  'src/lib/components/system/Toasts.svelte -> utils',
  'src/lib/components/ui/EntityTagBadges.svelte -> stores',
  'src/lib/components/ui/PrivilegeTable.svelte -> patterns',
  'src/lib/utils/__tests__/clientLog.test.js -> api',
  'src/lib/utils/clientLog.js -> api',
])

const FROM_RE = /\bfrom\s+['"]([^'"]+)['"]/g
const BARE_RE = /\bimport\s*\(\s*['"]([^'"]+)['"]\s*\)|\bimport\s+['"]([^'"]+)['"]/g

function layerOf(relPath) {
  for (const [name, dir] of LAYERS) {
    if (relPath === dir || relPath.startsWith(`${dir}/`)) return name
  }
  return null
}

function targetOf(spec, fromAbs) {
  let path = spec
  for (const [alias, replacement] of Object.entries(ALIASES)) {
    if (path.startsWith(alias)) {
      path = replacement + path.slice(alias.length)
      break
    }
  }
  if (path.startsWith('.')) path = relative(FRONTEND, resolve(dirname(fromAbs), path))
  if (path.startsWith('bindings/') || path.includes('/bindings/')) return 'bindings'
  return layerOf(path)
}

function* files(dir) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    if (statSync(full).isDirectory()) yield* files(full)
    else if (/\.(js|svelte)$/.test(entry)) yield full
  }
}

const violations = new Set()
for (const file of files(SRC)) {
  const rel = relative(FRONTEND, file)
  const from = layerOf(rel)
  if (!from) continue

  const text = readFileSync(file, 'utf8')
  const specs = []
  for (const re of [FROM_RE, BARE_RE]) {
    re.lastIndex = 0
    for (const match of text.matchAll(re)) specs.push(match[1] ?? match[2])
  }

  for (const spec of specs) {
    if (!spec) continue
    const to = targetOf(spec, file)
    if (!to || to === from) continue
    if ((ALLOWED[from] ?? []).includes(to)) continue
    violations.add(`${rel} -> ${to}`)
  }
}

const sorted = [...violations].sort()
const unexpected = sorted.filter((v) => !KNOWN.has(v))
const stale = [...KNOWN].filter((k) => !violations.has(k)).sort()

if (sorted.length > 0) {
  console.log(`frontend boundaries: ${sorted.length} violation(s)`)
  for (const v of sorted) console.log(`  ${KNOWN.has(v) ? 'known' : 'NEW  '} ${v}`)
}
if (stale.length > 0) {
  console.log('frontend boundaries: allowlist entries no longer violated, remove them')
  for (const s of stale) console.log(`  ${s}`)
}
if (unexpected.length > 0 || stale.length > 0) {
  console.error('frontend boundaries: FAILED')
  process.exit(1)
}
console.log('frontend boundaries: all layer rules passed')
