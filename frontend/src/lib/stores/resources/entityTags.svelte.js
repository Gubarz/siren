import { createResource } from '../lib/createResource.svelte.js'
import { GetAllEntityTags } from '../../api/tags.js'

export const entityTags = createResource({
  name: 'entity-tags',
  fetch: async () => GetAllEntityTags(),
  events: ['entity-tags-updated'],
})
