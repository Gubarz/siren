import { createResource } from '../lib/createResource.svelte.js'
import { listOperators } from '../../api/server.js'

export const operators = createResource({
  name: 'operators',
  fetch: () => listOperators(),
  events: ['client-joined', 'client-left'],
})
