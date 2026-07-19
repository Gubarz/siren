import { createResource } from '../lib/createResource.svelte.js'
import { listHosts } from '../../api/hosts.js'

export const hosts = createResource({
  name: 'hosts',
  fetch: () => listHosts(),
})
