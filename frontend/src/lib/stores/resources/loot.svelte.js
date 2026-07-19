import { createResource } from '../lib/createResource.svelte.js'
import { listLoot } from '../../api/server.js'

export const loot = createResource({
  name: 'loot',
  fetch: () => listLoot(),
  events: ['loot-added', 'loot-removed'],
})
