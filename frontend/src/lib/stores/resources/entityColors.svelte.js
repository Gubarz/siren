import { createResource } from '../lib/createResource.svelte.js'
import { GetAllEntityColors } from '../../api/tags.js'

export const entityColors = createResource({
  name: 'entity-colors',
  fetch: async () => GetAllEntityColors(),
  events: ['entity-colors-updated'],
})
