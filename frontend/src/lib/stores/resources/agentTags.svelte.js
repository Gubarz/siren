import { createResource } from '../lib/createResource.svelte.js'
import { GetAllAgentTags } from '../../api/tags.js'

// One store powers both the tag chip display on rows and the filter chip
// list up top — the backing map is small (agent-id -> string[]) so keeping
// it flat and refetching on any tag change is cheaper than reactive slicing.
export const agentTags = createResource({
  name: 'agent-tags',
  fetch: async () => GetAllAgentTags(),
  events: ['agent-tags-updated', 'entity-tags-updated'],
})
