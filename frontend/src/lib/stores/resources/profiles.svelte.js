import { createResource } from '../lib/createResource.svelte.js'
import { listProfiles } from '../../api/server.js'

export const profiles = createResource({
  name: 'profiles',
  fetch: () => listProfiles(),
})
