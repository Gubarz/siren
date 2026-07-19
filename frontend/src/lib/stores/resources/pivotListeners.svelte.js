import { createResource } from '../lib/createResource.svelte.js'
import { GetPivotListeners } from '../../api/server.js'

export const pivotListeners = createResource({
  name: 'pivotListeners',
  fetch: () => GetPivotListeners(),
  pollInterval: 5000,
})
