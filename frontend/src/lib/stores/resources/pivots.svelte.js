import { createResource } from '../lib/createResource.svelte.js'
import { GetPivots } from '../../api/server.js'

export const pivots = createResource({
  name: 'pivots',
  fetch: () => GetPivots(),
  pollInterval: 5000,
})
