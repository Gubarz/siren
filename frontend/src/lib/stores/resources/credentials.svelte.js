import { createResource } from '../lib/createResource.svelte.js'
import { listCredentials } from '../../api/server.js'

export const credentials = createResource({
  name: 'credentials',
  fetch: () => listCredentials(),
  events: ['credential-added', 'credential-removed'],
})
