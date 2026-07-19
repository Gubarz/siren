import { createResource } from '../lib/createResource.svelte.js'
import { listHTTPC2Profiles } from '../../api/httpc2.js'

export const httpC2Profiles = createResource({
  name: 'httpC2Profiles',
  fetch: () => listHTTPC2Profiles(),
})
