import { createResource } from '../lib/createResource.svelte.js'
import { GetAllAgentColors } from '../../api/tags.js'

// Per-agent row colors, persisted per teamserver on the Go side alongside
// the tags store (internal/tags). Flat map (agent-id -> color name) — small
// enough that a full refetch on any change beats reactive slicing.
export const agentColors = createResource({
  name: 'agent-colors',
  fetch: async () => GetAllAgentColors(),
  events: ['agent-colors-updated'],
})
